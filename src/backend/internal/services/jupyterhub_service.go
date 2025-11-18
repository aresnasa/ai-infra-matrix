package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// JupyterHubService JupyterHub管理服务
type JupyterHubService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewJupyterHubService 创建JupyterHub服务
func NewJupyterHubService() *JupyterHubService {
	return &JupyterHubService{
		db:     database.DB,
		logger: logrus.New(),
	}
}

// CreateHubConfig 创建JupyterHub配置
func (s *JupyterHubService) CreateHubConfig(config *models.JupyterHubConfig) error {
	if err := s.db.Create(config).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create JupyterHub config")
		return err
	}
	return nil
}

// GetHubConfigs 获取所有JupyterHub配置
func (s *JupyterHubService) GetHubConfigs() ([]models.JupyterHubConfig, error) {
	var configs []models.JupyterHubConfig
	if err := s.db.Find(&configs).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get JupyterHub configs")
		return nil, err
	}
	return configs, nil
}

// GetHubConfig 获取指定的JupyterHub配置
func (s *JupyterHubService) GetHubConfig(id uint) (*models.JupyterHubConfig, error) {
	var config models.JupyterHubConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateHubConfig 更新JupyterHub配置
func (s *JupyterHubService) UpdateHubConfig(id uint, updates map[string]interface{}) error {
	if err := s.db.Model(&models.JupyterHubConfig{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		s.logger.WithError(err).Error("Failed to update JupyterHub config")
		return err
	}
	return nil
}

// DeleteHubConfig 删除JupyterHub配置
func (s *JupyterHubService) DeleteHubConfig(id uint) error {
	if err := s.db.Delete(&models.JupyterHubConfig{}, id).Error; err != nil {
		s.logger.WithError(err).Error("Failed to delete JupyterHub config")
		return err
	}
	return nil
}

// CreateTask 创建JupyterHub任务
func (s *JupyterHubService) CreateTask(task *models.JupyterTask) error {
	// 生成唯一的任务ID
	task.JobID = fmt.Sprintf("jupyter-task-%d-%d", task.UserID, time.Now().Unix())

	if err := s.db.Create(task).Error; err != nil {
		s.logger.WithError(err).Error("Failed to create JupyterHub task")
		return err
	}

	// 异步执行任务
	go s.executeTask(task)

	return nil
}

// GetTasks 获取用户的任务列表
func (s *JupyterHubService) GetTasks(userID uint, limit, offset int) ([]models.JupyterTask, int64, error) {
	var tasks []models.JupyterTask
	var total int64

	query := s.db.Where("user_id = ?", userID)

	if err := query.Model(&models.JupyterTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Preload("User").Preload("HubConfig").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetTask 获取指定任务
func (s *JupyterHubService) GetTask(taskID uint, userID uint) (*models.JupyterTask, error) {
	var task models.JupyterTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).
		Preload("User").Preload("HubConfig").
		First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTaskStatus 更新任务状态
func (s *JupyterHubService) UpdateTaskStatus(taskID uint, status string, errorMsg ...string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "running" {
		now := time.Now()
		updates["started_at"] = &now
	} else if status == "completed" || status == "failed" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if len(errorMsg) > 0 && errorMsg[0] != "" {
		updates["error_message"] = errorMsg[0]
	}

	return s.db.Model(&models.JupyterTask{}).Where("id = ?", taskID).Updates(updates).Error
}

// executeTask 执行JupyterHub任务
func (s *JupyterHubService) executeTask(task *models.JupyterTask) {
	s.logger.WithField("task_id", task.ID).Info("Starting to execute JupyterHub task")

	// 更新状态为运行中
	if err := s.UpdateTaskStatus(task.ID, "running"); err != nil {
		s.logger.WithError(err).Error("Failed to update task status to running")
		return
	}

	// 获取JupyterHub配置
	hubConfig, err := s.GetHubConfig(task.HubConfigID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get hub config")
		s.UpdateTaskStatus(task.ID, "failed", err.Error())
		return
	}

	// 生成Ansible playbook来执行远程任务
	if err := s.generateAndExecuteAnsiblePlaybook(task, hubConfig); err != nil {
		s.logger.WithError(err).Error("Failed to execute ansible playbook")
		s.UpdateTaskStatus(task.ID, "failed", err.Error())
		return
	}

	s.logger.WithField("task_id", task.ID).Info("JupyterHub task completed successfully")
	s.UpdateTaskStatus(task.ID, "completed")
}

// generateAndExecuteAnsiblePlaybook 生成并执行Ansible playbook
func (s *JupyterHubService) generateAndExecuteAnsiblePlaybook(task *models.JupyterTask, hubConfig *models.JupyterHubConfig) error {
	// 解析GPU节点信息
	var gpuNodes []models.GPUNode
	if hubConfig.GPUNodes != "" {
		if err := json.Unmarshal([]byte(hubConfig.GPUNodes), &gpuNodes); err != nil {
			return fmt.Errorf("failed to parse GPU nodes: %w", err)
		}
	}

	// 选择合适的GPU节点
	selectedNode, err := s.selectGPUNode(gpuNodes, task.GPURequested)
	if err != nil {
		return fmt.Errorf("failed to select GPU node: %w", err)
	}

	// 生成任务脚本
	scriptContent := s.generateTaskScript(task, hubConfig)

	// 生成Ansible playbook
	playbookContent := s.generateJupyterPlaybook(task, selectedNode, scriptContent)

	// 保存playbook文件
	playbookPath := filepath.Join("outputs", fmt.Sprintf("jupyter-task-%d.yml", task.ID))
	if err := os.WriteFile(playbookPath, []byte(playbookContent), 0644); err != nil {
		return fmt.Errorf("failed to write playbook: %w", err)
	}

	// 执行Ansible playbook
	return s.executeAnsiblePlaybook(playbookPath, selectedNode.IPAddress)
}

// selectGPUNode 选择合适的GPU节点
func (s *JupyterHubService) selectGPUNode(nodes []models.GPUNode, requiredGPUs int) (*models.GPUNode, error) {
	for _, node := range nodes {
		if node.IsOnline && node.AvailableGPU >= requiredGPUs {
			return &node, nil
		}
	}

	// 如果没有找到合适的GPU节点，选择第一个在线节点
	for _, node := range nodes {
		if node.IsOnline {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("no available GPU nodes found")
}

// generateTaskScript 生成任务执行脚本
func (s *JupyterHubService) generateTaskScript(task *models.JupyterTask, hubConfig *models.JupyterHubConfig) string {
	script := fmt.Sprintf(`#!/bin/bash
# JupyterHub Task Execution Script
# Task ID: %d
# Task Name: %s

set -e

echo "Starting JupyterHub task execution..."
echo "Task ID: %d"
echo "Task Name: %s"
echo "GPU Requested: %d"
echo "Memory: %dGB"
echo "CPU Cores: %d"

# 设置环境变量
export CUDA_VISIBLE_DEVICES=0
export JUPYTER_TOKEN="%s"

# 创建工作目录
WORK_DIR="/tmp/jupyter-task-%d"
mkdir -p $WORK_DIR
cd $WORK_DIR

# 创建Python脚本
cat > task_script.py << 'EOF'
%s
EOF

# 执行Python脚本
echo "Executing Python script..."
python3 task_script.py

echo "Task execution completed successfully"
`, task.ID, task.TaskName, task.ID, task.TaskName, task.GPURequested, task.MemoryGB, task.CPUCores, hubConfig.Token, task.ID, task.PythonCode)

	return script
}

// generateJupyterPlaybook 生成JupyterHub任务的Ansible playbook
func (s *JupyterHubService) generateJupyterPlaybook(task *models.JupyterTask, node *models.GPUNode, scriptContent string) string {
	playbook := fmt.Sprintf(`---
- name: Execute JupyterHub Task
  hosts: %s
  become: yes
  vars:
    task_id: %d
    task_name: "%s"
    work_dir: "/tmp/jupyter-task-%d"
    
  tasks:
    - name: Create work directory
      file:
        path: "{{ work_dir }}"
        state: directory
        mode: '0755'
        
    - name: Copy task script
      copy:
        content: |
%s
        dest: "{{ work_dir }}/run_task.sh"
        mode: '0755'
        
    - name: Execute task script
      shell: |
        cd {{ work_dir }}
        ./run_task.sh > task_output.log 2>&1
      register: task_result
      async: 3600  # 1小时超时
      poll: 10
      
    - name: Fetch task output
      fetch:
        src: "{{ work_dir }}/task_output.log"
        dest: "./outputs/jupyter-task-%d-output.log"
        flat: yes
        
    - name: Clean up work directory
      file:
        path: "{{ work_dir }}"
        state: absent
      when: task_result is succeeded
`, node.IPAddress, task.ID, task.TaskName, task.ID,
		strings.ReplaceAll(scriptContent, "\n", "\n          "), task.ID)

	return playbook
}

// executeAnsiblePlaybook 执行Ansible playbook
func (s *JupyterHubService) executeAnsiblePlaybook(playbookPath, targetHost string) error {
	// 这里应该调用Ansible服务来执行playbook
	// 为了简化，这里只是模拟执行过程
	s.logger.WithFields(logrus.Fields{
		"playbook": playbookPath,
		"host":     targetHost,
	}).Info("Executing Ansible playbook for JupyterHub task")

	// 实际实现中，这里应该调用ansible-playbook命令或使用Ansible Go SDK
	// 暂时模拟成功执行
	time.Sleep(5 * time.Second)

	return nil
}

// GetJupyterHubUsers 获取JupyterHub用户列表
func (s *JupyterHubService) GetJupyterHubUsers(hubConfigID uint) ([]models.JupyterHubUser, error) {
	hubConfig, err := s.GetHubConfig(hubConfigID)
	if err != nil {
		return nil, err
	}

	// 调用JupyterHub API
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", hubConfig.URL+"/hub/api/users", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+hubConfig.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JupyterHub API error: %s", string(body))
	}

	var users []models.JupyterHubUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	return users, nil
}

// TestJupyterHubConnection 测试JupyterHub连接
func (s *JupyterHubService) TestJupyterHubConnection(hubConfig *models.JupyterHubConfig) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", hubConfig.URL+"/hub/api/info", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+hubConfig.Token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JupyterHub connection test failed: %s", string(body))
	}

	return nil
}

// CancelTask 取消任务
func (s *JupyterHubService) CancelTask(taskID uint, userID uint) error {
	task, err := s.GetTask(taskID, userID)
	if err != nil {
		return err
	}

	if task.Status == "running" {
		// 这里应该实现实际的任务取消逻辑
		// 比如停止Ansible执行或者杀死远程进程
		s.logger.WithField("task_id", taskID).Info("Cancelling running task")

		return s.UpdateTaskStatus(taskID, "cancelled", "Task cancelled by user")
	}

	return fmt.Errorf("task is not in running state")
}

// GetTaskOutput 获取任务输出
func (s *JupyterHubService) GetTaskOutput(taskID uint, userID uint) (string, error) {
	task, err := s.GetTask(taskID, userID)
	if err != nil {
		return "", err
	}

	outputPath := fmt.Sprintf("outputs/jupyter-task-%d-output.log", task.ID)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("task output not found")
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// GetHubStatus 获取JupyterHub状态
func (s *JupyterHubService) GetHubStatus() (map[string]interface{}, error) {
	// 这里应该实现与JupyterHub API的实际通信
	// 目前返回模拟数据
	status := map[string]interface{}{
		"running":         true,
		"users_online":    5,
		"servers_running": 3,
		"total_memory_gb": 32,
		"used_memory_gb":  12,
		"total_cpu_cores": 16,
		"used_cpu_cores":  6,
		"version":         "5.3.0",
		"url":             "/jupyter/",
		"last_updated":    time.Now().Format(time.RFC3339),
	}
	return status, nil
}

// GetUserTasks 获取用户任务列表（用于前端页面显示）
func (s *JupyterHubService) GetUserTasks(userID uint) ([]models.JupyterTask, error) {
	var tasks []models.JupyterTask

	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(10).
		Find(&tasks).Error; err != nil {
		s.logger.WithError(err).Error("Failed to get user tasks")
		return nil, err
	}

	return tasks, nil
}
