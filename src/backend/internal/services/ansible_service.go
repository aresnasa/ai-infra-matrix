package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// AnsibleService 提供Ansible执行服务
type AnsibleService struct {
	logger *logrus.Logger
}

// NewAnsibleService 创建新的Ansible服务
func NewAnsibleService() *AnsibleService {
	return &AnsibleService{
		logger: logrus.New(),
	}
}

// ExecutePlaybook 执行Ansible Playbook
func (s *AnsibleService) ExecutePlaybook(ctx context.Context, execution *models.AnsibleExecution, playbookContent, inventoryContent string) error {
	// 创建临时目录存放playbook和inventory文件
	tempDir, err := os.MkdirTemp("", "ansible-execution-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 创建playbook文件
	playbookPath := filepath.Join(tempDir, "playbook.yml")
	if err := os.WriteFile(playbookPath, []byte(playbookContent), 0644); err != nil {
		return fmt.Errorf("创建playbook文件失败: %v", err)
	}

	// 创建inventory文件
	inventoryPath := filepath.Join(tempDir, "inventory.ini")
	if inventoryContent != "" {
		if err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644); err != nil {
			return fmt.Errorf("创建inventory文件失败: %v", err)
		}
	}

	// 更新execution记录中的路径
	execution.PlaybookPath = playbookPath
	execution.InventoryPath = inventoryPath

	// 构建ansible-playbook命令
	args := []string{"ansible-playbook"}
	
	// 添加inventory参数
	if inventoryContent != "" {
		args = append(args, "-i", inventoryPath)
	}
	
	// 处理dry-run模式
	if execution.ExecutionType == "dry-run" {
		args = append(args, "--check", "--diff")
	}
	
	// 添加额外变量
	if execution.ExtraVars != "" {
		// 验证JSON格式
		var extraVarsMap map[string]interface{}
		if err := json.Unmarshal([]byte(execution.ExtraVars), &extraVarsMap); err != nil {
			return fmt.Errorf("额外变量格式错误，必须是有效的JSON: %v", err)
		}
		args = append(args, "--extra-vars", execution.ExtraVars)
	}
	
	// 添加环境变量
	if execution.Environment != "" {
		args = append(args, "--extra-vars", fmt.Sprintf(`{"environment":"%s"}`, execution.Environment))
	}
	
	// 添加其他有用的参数
	args = append(args, 
		"-v", // verbose输出
		"--force-color", // 强制彩色输出
		playbookPath,
	)

	s.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"command":      strings.Join(args, " "),
		"tempDir":      tempDir,
	}).Info("开始执行Ansible playbook")

	// 执行命令
	return s.runAnsibleCommand(ctx, execution, args)
}

// runAnsibleCommand 运行ansible命令
func (s *AnsibleService) runAnsibleCommand(ctx context.Context, execution *models.AnsibleExecution, args []string) error {
	// 创建命令
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	
	// 设置环境变量
	cmd.Env = append(os.Environ(),
		"ANSIBLE_HOST_KEY_CHECKING=False", // 禁用主机密钥检查
		"ANSIBLE_STDOUT_CALLBACK=json",    // 使用JSON输出格式
		"ANSIBLE_LOAD_CALLBACK_PLUGINS=True",
		"PYTHONUNBUFFERED=1", // 禁用Python输出缓冲
	)
	
	// 创建输出缓冲区
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// 记录开始时间和PID
	execution.StartTime = &time.Time{}
	*execution.StartTime = time.Now()
	execution.Status = "running"
	
	// 启动命令
	if err := cmd.Start(); err != nil {
		execution.Status = "failed"
		execution.ErrorOutput = fmt.Sprintf("启动命令失败: %v", err)
		return err
	}
	
	// 记录进程ID
	execution.PID = cmd.Process.Pid
	
	s.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"pid":          execution.PID,
	}).Info("Ansible进程已启动")
	
	// 等待命令完成
	err := cmd.Wait()
	
	// 记录结束时间和输出
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int(endTime.Sub(*execution.StartTime).Seconds())
	execution.Output = stdout.String()
	execution.ErrorOutput = stderr.String()
	
	// 处理退出状态
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				execution.ExitCode = status.ExitStatus()
			}
		}
		execution.Status = "failed"
		s.logger.WithFields(logrus.Fields{
			"execution_id": execution.ID,
			"exit_code":    execution.ExitCode,
			"error":        err,
		}).Error("Ansible执行失败")
	} else {
		execution.ExitCode = 0
		execution.Status = "success"
		s.logger.WithFields(logrus.Fields{
			"execution_id": execution.ID,
			"duration":     execution.Duration,
		}).Info("Ansible执行成功")
	}
	
	return nil
}

// CancelExecution 取消正在执行的任务
func (s *AnsibleService) CancelExecution(execution *models.AnsibleExecution) error {
	if execution.PID <= 0 {
		return fmt.Errorf("无效的进程ID")
	}
	
	// 查找进程
	process, err := os.FindProcess(execution.PID)
	if err != nil {
		return fmt.Errorf("找不到进程 %d: %v", execution.PID, err)
	}
	
	// 尝试优雅地终止进程
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// 如果优雅终止失败，强制杀死进程
		if killErr := process.Kill(); killErr != nil {
			return fmt.Errorf("终止进程失败: %v", killErr)
		}
	}
	
	execution.Status = "cancelled"
	endTime := time.Now()
	execution.EndTime = &endTime
	if execution.StartTime != nil {
		execution.Duration = int(endTime.Sub(*execution.StartTime).Seconds())
	}
	
	s.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"pid":          execution.PID,
	}).Info("Ansible执行已取消")
	
	return nil
}

// ValidatePlaybook 验证playbook语法
func (s *AnsibleService) ValidatePlaybook(playbookContent string) error {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "playbook-validate-*.yml")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	// 写入playbook内容
	if _, err := tempFile.WriteString(playbookContent); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	tempFile.Close()
	
	// 使用ansible-playbook --syntax-check验证语法
	cmd := exec.Command("ansible-playbook", "--syntax-check", tempFile.Name())
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return fmt.Errorf("playbook语法错误: %s", string(output))
	}
	
	return nil
}

// GenerateInventoryFromProject 从项目生成inventory文件内容
func (s *AnsibleService) GenerateInventoryFromProject(project models.Project) string {
	var inventory strings.Builder
	
	// 添加默认组
	inventory.WriteString("[" + project.Name + "]\n")
	
	// 添加主机
	for _, host := range project.Hosts {
		line := fmt.Sprintf("%s ansible_host=%s", host.Name, host.IP)
		
		if host.Port != 22 {
			line += fmt.Sprintf(" ansible_port=%d", host.Port)
		}
		
		if host.User != "" {
			line += fmt.Sprintf(" ansible_user=%s", host.User)
		}
		
		inventory.WriteString(line + "\n")
	}
	
	// 添加变量组
	if len(project.Variables) > 0 {
		inventory.WriteString(fmt.Sprintf("\n[%s:vars]\n", project.Name))
		for _, variable := range project.Variables {
			inventory.WriteString(fmt.Sprintf("%s=%s\n", variable.Name, variable.Value))
		}
	}
	
	return inventory.String()
}

// GetExecutionStatus 获取执行状态
func (s *AnsibleService) GetExecutionStatus(execution *models.AnsibleExecution) string {
	// 如果是运行状态，检查进程是否还在运行
	if execution.Status == "running" && execution.PID > 0 {
		// 检查进程是否存在
		if process, err := os.FindProcess(execution.PID); err == nil {
			// 发送信号0来检查进程是否还活着
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// 进程不存在，更新状态
				return "failed"
			}
		}
	}
	
	return execution.Status
}

// FormatExecutionLogs 格式化执行日志用于显示
func (s *AnsibleService) FormatExecutionLogs(execution *models.AnsibleExecution) map[string]interface{} {
	logs := map[string]interface{}{
		"execution_id": execution.ID,
		"status":       execution.Status,
		"start_time":   execution.StartTime,
		"end_time":     execution.EndTime,
		"duration":     execution.Duration,
		"exit_code":    execution.ExitCode,
	}
	
	// 解析JSON输出（如果可能）
	if execution.Output != "" {
		var jsonOutput interface{}
		if err := json.Unmarshal([]byte(execution.Output), &jsonOutput); err == nil {
			logs["structured_output"] = jsonOutput
		} else {
			logs["raw_output"] = execution.Output
		}
	}
	
	if execution.ErrorOutput != "" {
		logs["error_output"] = execution.ErrorOutput
	}
	
	return logs
}
