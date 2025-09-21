package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// OSInfo 操作系统信息
type OSInfo struct {
	OS      string `json:"os"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

// SlurmClusterService SLURM集群管理服务，整合SaltStack和SSH功能
type SlurmClusterService struct {
	db              *gorm.DB
	sshService      *SSHService
	saltService     *SaltStackService
	deploymentTasks map[string]*DeploymentTask
	taskMutex       sync.RWMutex
}

// DeploymentTask 部署任务
type DeploymentTask struct {
	ID           string                   `json:"id"`
	ClusterID    uint                     `json:"cluster_id"`
	DeploymentID uint                     `json:"deployment_id"`
	Status       string                   `json:"status"`
	Progress     int                      `json:"progress"`
	CurrentStep  string                   `json:"current_step"`
	StartedAt    time.Time                `json:"started_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
	Context      context.Context          `json:"-"`
	Cancel       context.CancelFunc       `json:"-"`
	NodeTasks    map[uint]*NodeDeployTask `json:"node_tasks"`
}

// NodeDeployTask 节点部署任务
type NodeDeployTask struct {
	NodeID      uint   `json:"node_id"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CurrentStep string `json:"current_step"`
	Error       string `json:"error,omitempty"`
}

func NewSlurmClusterService(db *gorm.DB) *SlurmClusterService {
	return &SlurmClusterService{
		db:              db,
		sshService:      NewSSHService(),
		saltService:     NewSaltStackService(),
		deploymentTasks: make(map[string]*DeploymentTask),
	}
}

// CreateCluster 创建SLURM集群
func (s *SlurmClusterService) CreateCluster(ctx context.Context, req models.CreateClusterRequest, userID uint) (*models.SlurmCluster, error) {
	// 创建集群记录
	cluster := &models.SlurmCluster{
		Name:        req.Name,
		Description: req.Description,
		Status:      "pending",
		MasterHost:  req.MasterHost,
		MasterPort:  req.MasterPort,
		SaltMaster:  req.SaltMaster,
		Config:      req.Config,
		CreatedBy:   userID,
	}

	if err := s.db.Create(cluster).Error; err != nil {
		return nil, fmt.Errorf("failed to create cluster: %v", err)
	}

	// 创建节点记录
	for _, nodeReq := range req.Nodes {
		node := &models.SlurmNode{
			ClusterID:    cluster.ID,
			NodeName:     nodeReq.NodeName,
			NodeType:     nodeReq.NodeType,
			Host:         nodeReq.Host,
			Port:         nodeReq.Port,
			Username:     nodeReq.Username,
			AuthType:     nodeReq.AuthType,
			Password:     nodeReq.Password,
			KeyPath:      nodeReq.KeyPath,
			Status:       "pending",
			CPUs:         nodeReq.CPUs,
			Memory:       nodeReq.Memory,
			Storage:      nodeReq.Storage,
			GPUs:         nodeReq.GPUs,
			NodeConfig:   nodeReq.NodeConfig,
			SaltMinionID: fmt.Sprintf("%s-%s", cluster.Name, nodeReq.NodeName),
		}

		if err := s.db.Create(node).Error; err != nil {
			return nil, fmt.Errorf("failed to create node %s: %v", nodeReq.NodeName, err)
		}
	}

	// 重新加载集群和节点信息
	if err := s.db.Preload("Nodes").First(cluster, cluster.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload cluster: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"node_count":   len(cluster.Nodes),
	}).Info("SLURM cluster created successfully")

	return cluster, nil
}

// DeployClusterAsync 异步部署SLURM集群
func (s *SlurmClusterService) DeployClusterAsync(ctx context.Context, req models.DeployClusterRequest, userID uint) (string, error) {
	// 获取集群信息
	var cluster models.SlurmCluster
	if err := s.db.Preload("Nodes").First(&cluster, req.ClusterID).Error; err != nil {
		return "", fmt.Errorf("cluster not found: %v", err)
	}

	// 生成部署ID
	deploymentID := s.generateDeploymentID()

	// 创建部署记录
	deployment := &models.ClusterDeployment{
		ClusterID:    cluster.ID,
		DeploymentID: deploymentID,
		Action:       req.Action,
		Status:       "pending",
		Progress:     0,
		Config:       req.Config,
		CreatedBy:    userID,
	}

	if err := s.db.Create(deployment).Error; err != nil {
		return "", fmt.Errorf("failed to create deployment record: %v", err)
	}

	// 创建部署任务
	taskCtx, cancel := context.WithCancel(ctx)
	task := &DeploymentTask{
		ID:           deploymentID,
		ClusterID:    cluster.ID,
		DeploymentID: deployment.ID,
		Status:       "pending",
		Progress:     0,
		CurrentStep:  "Initializing deployment",
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Context:      taskCtx,
		Cancel:       cancel,
		NodeTasks:    make(map[uint]*NodeDeployTask),
	}

	// 初始化节点任务
	for _, node := range cluster.Nodes {
		task.NodeTasks[node.ID] = &NodeDeployTask{
			NodeID:      node.ID,
			Status:      "pending",
			Progress:    0,
			CurrentStep: "Waiting for deployment",
		}
	}

	s.taskMutex.Lock()
	s.deploymentTasks[deploymentID] = task
	s.taskMutex.Unlock()

	// 启动异步部署
	go s.runDeploymentTask(task, &cluster, deployment)

	logrus.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"cluster_id":    cluster.ID,
		"action":        req.Action,
	}).Info("SLURM cluster deployment task created")

	return deploymentID, nil
}

// runDeploymentTask 执行部署任务
func (s *SlurmClusterService) runDeploymentTask(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) {
	defer func() {
		if r := recover(); r != nil {
			s.updateDeploymentStatus(deployment, "failed", fmt.Sprintf("Deployment panicked: %v", r))
			logrus.WithFields(logrus.Fields{
				"deployment_id": task.ID,
				"cluster_id":    task.ClusterID,
				"panic":         r,
			}).Error("Deployment task panicked")
		}
	}()

	logrus.WithFields(logrus.Fields{
		"deployment_id": task.ID,
		"cluster_id":    task.ClusterID,
	}).Info("Starting SLURM cluster deployment")

	// 更新任务状态
	task.Status = "running"
	task.CurrentStep = "Starting deployment"
	s.updateDeploymentStatus(deployment, "running", "Starting deployment")
	s.updateTaskProgress(task, 5)

	// 步骤1: 验证SSH连接
	if err := s.validateSSHConnections(task, cluster); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("SSH validation failed: %v", err))
		return
	}
	s.updateTaskProgress(task, 15)

	// 步骤2: 安装SaltStack Minion
	if err := s.installSaltMinions(task, cluster, deployment); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("SaltStack installation failed: %v", err))
		return
	}
	s.updateTaskProgress(task, 40)

	// 步骤3: 配置SaltStack
	if err := s.configureSalt(task, cluster, deployment); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("SaltStack configuration failed: %v", err))
		return
	}
	s.updateTaskProgress(task, 55)

	// 步骤4: 安装SLURM
	if err := s.installSlurm(task, cluster, deployment); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("SLURM installation failed: %v", err))
		return
	}
	s.updateTaskProgress(task, 80)

	// 步骤5: 启动和验证SLURM服务
	if err := s.startAndValidateSlurm(task, cluster, deployment); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("SLURM startup failed: %v", err))
		return
	}
	s.updateTaskProgress(task, 95)

	// 步骤6: 最终验证
	if err := s.finalValidation(task, cluster, deployment); err != nil {
		s.failDeployment(task, deployment, fmt.Sprintf("Final validation failed: %v", err))
		return
	}

	// 部署成功
	s.completeDeployment(task, deployment, cluster)
	s.updateTaskProgress(task, 100)

	logrus.WithFields(logrus.Fields{
		"deployment_id": task.ID,
		"cluster_id":    task.ClusterID,
	}).Info("SLURM cluster deployment completed successfully")
}

// validateSSHConnections 验证SSH连接
func (s *SlurmClusterService) validateSSHConnections(task *DeploymentTask, cluster *models.SlurmCluster) error {
	task.CurrentStep = "Validating SSH connections"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "ssh-validation", "ssh", "Validating SSH connections to all nodes")

	var errors []string
	for _, node := range cluster.Nodes {
		nodeTask := task.NodeTasks[node.ID]
		nodeTask.Status = "running"
		nodeTask.CurrentStep = "Testing SSH connection"

		// 记录SSH连接测试
		sessionID := s.generateSessionID()
		sshLog := &models.SSHExecutionLog{
			SessionID: sessionID,
			NodeID:    node.ID,
			StepID:    stepRecord.ID,
			Host:      node.Host,
			Port:      node.Port,
			Username:  node.Username,
			Command:   "echo 'SSH connection test'",
			StartedAt: time.Now(),
		}

		// 测试SSH连接
		if err := s.testNodeSSHConnection(&node); err != nil {
			errors = append(errors, fmt.Sprintf("Node %s (%s): %v", node.NodeName, node.Host, err))
			nodeTask.Status = "failed"
			nodeTask.Error = err.Error()

			sshLog.Success = false
			sshLog.ErrorOutput = err.Error()
			sshLog.ExitCode = 1
		} else {
			nodeTask.Status = "completed"
			nodeTask.Progress = 100

			sshLog.Success = true
			sshLog.Output = "SSH connection successful"
			sshLog.ExitCode = 0
		}

		// 完成SSH日志记录
		completedAt := time.Now()
		sshLog.CompletedAt = &completedAt
		sshLog.Duration = int(completedAt.Sub(sshLog.StartedAt).Milliseconds())
		s.db.Create(sshLog)
	}

	if len(errors) > 0 {
		s.completeDeploymentStep(stepRecord, "failed", strings.Join(errors, "; "))
		return fmt.Errorf("SSH validation failed: %s", strings.Join(errors, "; "))
	}

	s.completeDeploymentStep(stepRecord, "completed", "All SSH connections validated successfully")
	return nil
}

// installSaltMinions 安装SaltStack Minion
func (s *SlurmClusterService) installSaltMinions(task *DeploymentTask, cluster *models.SlurmCluster, deployment *models.ClusterDeployment) error {
	task.CurrentStep = "Installing SaltStack Minions"

	stepRecord := s.createDeploymentStep(task.DeploymentID, "salt-installation", "salt", "Installing SaltStack Minion on all nodes")

	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.Nodes))

	// 并发安装SaltStack Minion
	for _, node := range cluster.Nodes {
		wg.Add(1)
		go func(n models.SlurmNode) {
			defer wg.Done()

			nodeTask := task.NodeTasks[n.ID]
			nodeTask.Status = "running"
			nodeTask.CurrentStep = "Installing SaltStack Minion"

			// 创建节点安装任务记录
			installTask := &models.NodeInstallTask{
				TaskID:       s.generateTaskID(),
				NodeID:       n.ID,
				DeploymentID: deployment.ID,
				TaskType:     "salt-minion",
				Status:       "running",
				Progress:     0,
				InstallConfig: models.InstallTaskConfig{
					PackageSource: "repository",
					Version:       cluster.Config.SaltVersion,
					InstallType:   "package",
				},
			}
			startedAt := time.Now()
			installTask.StartedAt = &startedAt
			s.db.Create(installTask)

			if err := s.installSaltMinionOnNode(&n, cluster, installTask); err != nil {
				nodeTask.Status = "failed"
				nodeTask.Error = err.Error()
				installTask.Status = "failed"
				installTask.ErrorMessage = err.Error()
				errChan <- fmt.Errorf("node %s: %v", n.NodeName, err)
			} else {
				nodeTask.Status = "completed"
				nodeTask.Progress = 100
				installTask.Status = "completed"
				installTask.Progress = 100
			}

			// 完成安装任务记录
			completedAt := time.Now()
			installTask.CompletedAt = &completedAt
			s.db.Save(installTask)

		}(node)
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		s.completeDeploymentStep(stepRecord, "failed", strings.Join(errors, "; "))
		return fmt.Errorf("SaltStack installation failed: %s", strings.Join(errors, "; "))
	}

	s.completeDeploymentStep(stepRecord, "completed", "SaltStack Minion installed on all nodes")
	return nil
}

// installSaltMinionOnNode 在单个节点上安装SaltStack Minion
func (s *SlurmClusterService) installSaltMinionOnNode(node *models.SlurmNode, cluster *models.SlurmCluster, installTask *models.NodeInstallTask) error {
	sessionID := s.generateSessionID()

	// 建立SSH连接
	sshConfig := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(node.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", node.Host, node.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	defer client.Close()

	// 步骤1: 检测操作系统
	step1 := s.createInstallStep(installTask.ID, "detect-os", "ssh-connect", "Detecting operating system")
	osInfo, err := s.detectOSInfo(client, sessionID, node, installTask, step1)
	if err != nil {
		s.completeInstallStep(step1, "failed", err.Error())
		return fmt.Errorf("OS detection failed: %v", err)
	}
	s.completeInstallStep(step1, "completed", fmt.Sprintf("Detected OS: %s %s", osInfo.OS, osInfo.Version))

	// 步骤2: 安装SaltStack仓库
	step2 := s.createInstallStep(installTask.ID, "install-repo", "install", "Installing SaltStack repository")
	if err := s.installSaltRepository(client, osInfo, sessionID, node, installTask, step2); err != nil {
		s.completeInstallStep(step2, "failed", err.Error())
		return fmt.Errorf("repository installation failed: %v", err)
	}
	s.completeInstallStep(step2, "completed", "SaltStack repository installed")

	// 步骤3: 安装Salt Minion
	step3 := s.createInstallStep(installTask.ID, "install-minion", "install", "Installing Salt Minion")
	if err := s.installSaltMinionPackage(client, osInfo, sessionID, node, installTask, step3); err != nil {
		s.completeInstallStep(step3, "failed", err.Error())
		return fmt.Errorf("Salt Minion installation failed: %v", err)
	}
	s.completeInstallStep(step3, "completed", "Salt Minion installed")

	// 步骤4: 配置Salt Minion
	step4 := s.createInstallStep(installTask.ID, "configure-minion", "configure", "Configuring Salt Minion")
	if err := s.configureSaltMinion(client, cluster, node, sessionID, installTask, step4); err != nil {
		s.completeInstallStep(step4, "failed", err.Error())
		return fmt.Errorf("Salt Minion configuration failed: %v", err)
	}
	s.completeInstallStep(step4, "completed", "Salt Minion configured")

	// 步骤5: 启动Salt Minion服务
	step5 := s.createInstallStep(installTask.ID, "start-service", "start", "Starting Salt Minion service")
	if err := s.startSaltMinionService(client, sessionID, node, installTask, step5); err != nil {
		s.completeInstallStep(step5, "failed", err.Error())
		return fmt.Errorf("Salt Minion service start failed: %v", err)
	}
	s.completeInstallStep(step5, "completed", "Salt Minion service started")

	return nil
}

// 辅助方法
func (s *SlurmClusterService) generateDeploymentID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "deploy-" + hex.EncodeToString(bytes)
}

func (s *SlurmClusterService) generateTaskID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "task-" + hex.EncodeToString(bytes)
}

func (s *SlurmClusterService) generateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "ssh-" + hex.EncodeToString(bytes)
}

func (s *SlurmClusterService) updateTaskProgress(task *DeploymentTask, progress int) {
	task.Progress = progress
	task.UpdatedAt = time.Now()

	// 更新数据库中的部署记录
	s.db.Model(&models.ClusterDeployment{}).
		Where("id = ?", task.DeploymentID).
		Updates(map[string]interface{}{
			"progress":   progress,
			"updated_at": task.UpdatedAt,
		})
}

func (s *SlurmClusterService) updateDeploymentStatus(deployment *models.ClusterDeployment, status, message string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}

	if status == "running" && deployment.StartedAt == nil {
		updates["started_at"] = now
	} else if status == "completed" || status == "failed" {
		updates["completed_at"] = now
	}

	if message != "" {
		if status == "failed" {
			updates["error_message"] = message
		}
	}

	s.db.Model(deployment).Updates(updates)
}

func (s *SlurmClusterService) failDeployment(task *DeploymentTask, deployment *models.ClusterDeployment, message string) {
	task.Status = "failed"
	task.CurrentStep = "Deployment failed"
	s.updateDeploymentStatus(deployment, "failed", message)

	// 更新集群状态
	s.db.Model(&models.SlurmCluster{}).
		Where("id = ?", task.ClusterID).
		Update("status", "failed")
}

func (s *SlurmClusterService) completeDeployment(task *DeploymentTask, deployment *models.ClusterDeployment, cluster *models.SlurmCluster) {
	task.Status = "completed"
	task.CurrentStep = "Deployment completed"
	s.updateDeploymentStatus(deployment, "completed", "")

	// 更新集群状态
	s.db.Model(cluster).Update("status", "running")

	// 更新节点状态
	for _, node := range cluster.Nodes {
		s.db.Model(&models.SlurmNode{}).
			Where("id = ?", node.ID).
			Update("status", "active")
	}
}

// GetDeploymentStatus 获取部署状态
func (s *SlurmClusterService) GetDeploymentStatus(deploymentID string) (*DeploymentTask, error) {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	task, exists := s.deploymentTasks[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment task not found: %s", deploymentID)
	}

	return task, nil
}

// ListClusters 列出集群
func (s *SlurmClusterService) ListClusters(userID uint) ([]models.SlurmCluster, error) {
	var clusters []models.SlurmCluster
	if err := s.db.Preload("Nodes").Preload("User").Where("created_by = ?", userID).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

// GetCluster 获取集群详情
func (s *SlurmClusterService) GetCluster(clusterID, userID uint) (*models.SlurmCluster, error) {
	var cluster models.SlurmCluster
	if err := s.db.Preload("Nodes").Preload("Deployments").
		Where("id = ? AND created_by = ?", clusterID, userID).
		First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}
