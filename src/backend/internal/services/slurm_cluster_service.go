package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

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

// GetClusterByID 根据ID获取集群信息（不检查用户权限）
func (s *SlurmClusterService) GetClusterByID(ctx context.Context, clusterID uint) (*models.SlurmCluster, error) {
	var cluster models.SlurmCluster
	if err := s.db.Preload("Nodes").Preload("Deployments").
		First(&cluster, clusterID).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

// ConnectExternalCluster 连接已有的SLURM集群
func (s *SlurmClusterService) ConnectExternalCluster(ctx context.Context, req models.ConnectExternalClusterRequest, userID uint) (*models.SlurmCluster, error) {
	logrus.WithFields(logrus.Fields{
		"name":        req.Name,
		"master_host": req.MasterHost,
		"user_id":     userID,
	}).Info("Connecting to external SLURM cluster")

	// 创建SSH客户端配置
	sshConfig := &ssh.ClientConfig{
		User:            req.MasterSSH.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// 配置认证方式
	if req.MasterSSH.AuthType == "password" && req.MasterSSH.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(req.MasterSSH.Password),
		}
	} else if req.MasterSSH.AuthType == "key" && req.MasterSSH.KeyPath != "" {
		// 这里简化处理，实际应该读取密钥文件
		logrus.Warn("SSH key authentication not fully implemented yet")
		return nil, fmt.Errorf("SSH key authentication not supported in this version")
	} else {
		return nil, fmt.Errorf("invalid SSH authentication configuration")
	}

	// 测试SSH连接
	address := fmt.Sprintf("%s:%d", req.MasterSSH.Host, req.MasterSSH.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to SLURM master via SSH")
		return nil, fmt.Errorf("failed to connect to SLURM master: %v", err)
	}
	defer client.Close()

	// 验证SLURM是否已安装
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("scontrol --version")
	if err != nil {
		return nil, fmt.Errorf("SLURM is not installed or not accessible: %v", err)
	}

	slurmVersion := strings.TrimSpace(string(output))
	logrus.Infof("Detected SLURM version: %s", slurmVersion)

	// 获取集群信息
	session2, _ := client.NewSession()
	clusterInfo, _ := session2.CombinedOutput("scontrol show config | grep ClusterName")
	session2.Close()

	_ = clusterInfo // 暂时不使用，保留用于日志
	if len(clusterInfo) > 0 {
		parts := strings.Split(string(clusterInfo), "=")
		if len(parts) > 1 {
			detectedName := strings.TrimSpace(parts[1])
			if detectedName != "" {
				logrus.Infof("Detected cluster name: %s", detectedName)
			}
		}
	}

	// 创建集群记录
	cluster := &models.SlurmCluster{
		Name:        req.Name,
		Description: req.Description,
		Status:      "running", // external集群默认为running状态
		ClusterType: "external",
		MasterHost:  req.MasterHost,
		MasterPort:  req.MasterPort,
		MasterSSH:   &req.MasterSSH,
		Config:      req.Config,
		CreatedBy:   userID,
	}

	if cluster.Config.SlurmVersion == "" {
		cluster.Config.SlurmVersion = slurmVersion
	}

	if err := s.db.Create(cluster).Error; err != nil {
		return nil, fmt.Errorf("failed to create cluster record: %v", err)
	}

	// 异步获取节点信息
	go s.discoverClusterNodes(cluster.ID, req.MasterSSH)

	logrus.WithFields(logrus.Fields{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
	}).Info("External cluster connected successfully")

	return cluster, nil
}

// discoverClusterNodes 发现集群节点
func (s *SlurmClusterService) discoverClusterNodes(clusterID uint, sshConfig models.SSHConfig) {
	logrus.Infof("Discovering nodes for cluster %d", clusterID)

	// 创建SSH客户端
	clientConfig := &ssh.ClientConfig{
		User:            sshConfig.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	if sshConfig.AuthType == "password" && sshConfig.Password != "" {
		clientConfig.Auth = []ssh.AuthMethod{
			ssh.Password(sshConfig.Password),
		}
	}

	address := fmt.Sprintf("%s:%d", sshConfig.Host, sshConfig.Port)
	client, err := ssh.Dial("tcp", address, clientConfig)
	if err != nil {
		logrus.WithError(err).Error("Failed to connect for node discovery")
		return
	}
	defer client.Close()

	// 获取节点信息
	session, err := client.NewSession()
	if err != nil {
		logrus.WithError(err).Error("Failed to create session for node discovery")
		return
	}
	defer session.Close()

	// 执行 sinfo 获取节点列表
	output, err := session.CombinedOutput("sinfo -N -h -o '%N %t %c %m %f'")
	if err != nil {
		logrus.WithError(err).Error("Failed to get node info")
		return
	}

	// 解析节点信息
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		nodeName := fields[0]
		nodeState := fields[1]
		cpus := 1
		memory := 1024

		// 尝试解析CPU和内存
		if len(fields) > 2 {
			fmt.Sscanf(fields[2], "%d", &cpus)
		}
		if len(fields) > 3 {
			fmt.Sscanf(fields[3], "%d", &memory)
		}

		// 创建节点记录
		node := &models.SlurmNode{
			ClusterID:      clusterID,
			NodeName:       nodeName,
			NodeType:       "compute", // 默认为compute节点
			Host:           nodeName,  // 使用节点名作为host
			Port:           22,
			Username:       sshConfig.Username,
			AuthType:       sshConfig.AuthType,
			Password:       sshConfig.Password,
			KeyPath:        sshConfig.KeyPath,
			Status:         s.mapSlurmStateToStatus(nodeState),
			CPUs:           cpus,
			Memory:         memory,
			SaltMinionID:   nodeName,
			CoresPerSocket: 1,
			ThreadsPerCore: 1,
		}

		if err := s.db.Create(node).Error; err != nil {
			logrus.WithError(err).Errorf("Failed to create node record for %s", nodeName)
			continue
		}

		logrus.Infof("Discovered node: %s (CPUs: %d, Memory: %dMB, State: %s)",
			nodeName, cpus, memory, nodeState)
	}
}

// mapSlurmStateToStatus 将SLURM节点状态映射到我们的状态
func (s *SlurmClusterService) mapSlurmStateToStatus(slurmState string) string {
	switch strings.ToLower(slurmState) {
	case "idle", "alloc", "mixed":
		return "active"
	case "down", "drain", "drained":
		return "failed"
	case "unknown":
		return "pending"
	default:
		return "active"
	}
}

// GetClusterInfo 获取集群详细信息
func (s *SlurmClusterService) GetClusterInfo(ctx context.Context, clusterID uint) (map[string]interface{}, error) {
	var cluster models.SlurmCluster
	if err := s.db.Preload("Nodes").First(&cluster, clusterID).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %v", err)
	}

	// 如果是external类型，通过SSH获取实时信息
	if cluster.ClusterType == "external" && cluster.MasterSSH != nil {
		return s.getExternalClusterInfo(cluster)
	}

	// 对于managed类型，返回数据库中的信息
	info := map[string]interface{}{
		"cluster":      cluster,
		"node_count":   len(cluster.Nodes),
		"cluster_type": cluster.ClusterType,
		"status":       cluster.Status,
	}

	// 统计节点状态
	nodeStatus := make(map[string]int)
	for _, node := range cluster.Nodes {
		nodeStatus[node.Status]++
	}
	info["node_status"] = nodeStatus

	return info, nil
}

// getExternalClusterInfo 获取外部集群的实时信息
func (s *SlurmClusterService) getExternalClusterInfo(cluster models.SlurmCluster) (map[string]interface{}, error) {
	// 创建SSH客户端
	clientConfig := &ssh.ClientConfig{
		User:            cluster.MasterSSH.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	if cluster.MasterSSH.AuthType == "password" && cluster.MasterSSH.Password != "" {
		clientConfig.Auth = []ssh.AuthMethod{
			ssh.Password(cluster.MasterSSH.Password),
		}
	}

	address := fmt.Sprintf("%s:%d", cluster.MasterSSH.Host, cluster.MasterSSH.Port)
	client, err := ssh.Dial("tcp", address, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %v", err)
	}
	defer client.Close()

	info := map[string]interface{}{
		"cluster":      cluster,
		"node_count":   len(cluster.Nodes),
		"cluster_type": "external",
		"status":       cluster.Status,
	}

	// 获取sinfo信息
	session1, _ := client.NewSession()
	sinfoOutput, err := session1.CombinedOutput("sinfo -h")
	session1.Close()
	if err == nil {
		info["sinfo"] = string(sinfoOutput)
	}

	// 获取squeue信息
	session2, _ := client.NewSession()
	squeueOutput, err := session2.CombinedOutput("squeue -h")
	session2.Close()
	if err == nil {
		jobLines := strings.Split(strings.TrimSpace(string(squeueOutput)), "\n")
		if len(jobLines) > 0 && jobLines[0] != "" {
			info["running_jobs"] = len(jobLines)
		} else {
			info["running_jobs"] = 0
		}
	}

	// 获取节点统计
	session3, _ := client.NewSession()
	nodeStatsOutput, err := session3.CombinedOutput("sinfo -N -h -o '%t' | sort | uniq -c")
	session3.Close()
	if err == nil {
		nodeStats := make(map[string]int)
		lines := strings.Split(strings.TrimSpace(string(nodeStatsOutput)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				count := 0
				fmt.Sscanf(fields[0], "%d", &count)
				state := fields[1]
				nodeStats[state] = count
			}
		}
		info["node_stats"] = nodeStats
	}

	return info, nil
}

// CreateNode 创建节点记录
func (s *SlurmClusterService) CreateNode(ctx context.Context, node *models.SlurmNode) error {
	return s.db.WithContext(ctx).Create(node).Error
}

// DeleteNode 删除节点
// 该方法会停止节点上的服务，从集群配置中移除节点，并删除数据库记录
func (s *SlurmClusterService) DeleteNode(ctx context.Context, nodeID uint, force bool) error {
	// 查询节点信息
	var node models.SlurmNode
	if err := s.db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("节点不存在 (ID: %d)", nodeID)
		}
		return fmt.Errorf("查询节点失败: %v", err)
	}

	logrus.Infof("开始删除节点: %s (ID: %d, Host: %s)", node.NodeName, nodeID, node.Host)

	// 如果节点有SSH配置，尝试远程停止服务
	if node.Host != "" && !force {
		config := RemoteNodeInitConfig{
			NodeID:   node.ID,
			NodeName: node.NodeName,
			Host:     node.Host,
			Port:     node.Port,
			Username: node.Username,
			AuthType: node.AuthType,
			Password: node.Password,
			KeyPath:  node.KeyPath,
		}

		// 尝试停止服务（忽略错误）
		logrus.Infof("尝试停止节点 %s (%s) 的服务...", node.NodeName, node.Host)
		if err := s.stopNodeServices(config); err != nil {
			logrus.Warnf("停止节点服务失败（继续删除）: %v", err)
		} else {
			logrus.Infof("节点 %s 服务已停止", node.NodeName)
		}
	} else if node.Host == "" {
		logrus.Infof("节点 %s 没有配置SSH信息，跳过停止服务步骤", node.NodeName)
	} else {
		logrus.Infof("强制删除模式，跳过停止服务步骤")
	}

	// 从数据库彻底删除节点记录（使用Unscoped进行硬删除）
	if err := s.db.WithContext(ctx).Unscoped().Delete(&node).Error; err != nil {
		logrus.Errorf("删除节点记录失败: %v", err)
		return fmt.Errorf("删除节点记录失败: %v", err)
	}

	logrus.Infof("节点删除成功: %s (ID: %d)", node.NodeName, nodeID)
	return nil
}

// DeleteNodeByName 通过节点名称删除节点
func (s *SlurmClusterService) DeleteNodeByName(ctx context.Context, nodeName string, force bool) error {
	// 先尝试从数据库中查找节点
	var node models.SlurmNode
	err := s.db.WithContext(ctx).Where("node_name = ?", nodeName).First(&node).Error

	if err == nil {
		// 节点在数据库中，使用标准删除流程
		logrus.Infof("找到数据库中的节点 %s (ID: %d)，执行标准删除流程", nodeName, node.ID)
		return s.DeleteNode(ctx, node.ID, force)
	}

	if err != gorm.ErrRecordNotFound {
		// 查询出错
		return fmt.Errorf("查询节点失败: %v", err)
	}

	// 节点不在数据库中，从 SLURM 配置中移除
	logrus.Infof("节点 %s 不在数据库中，尝试从 SLURM 配置中移除", nodeName)

	// 从 SLURM 中移除节点
	if err := s.removeNodeFromSlurmConfig(nodeName); err != nil {
		logrus.Errorf("从 SLURM 配置中移除节点 %s 失败: %v", nodeName, err)
		return fmt.Errorf("从 SLURM 配置中移除节点失败: %v", err)
	}

	logrus.Infof("成功从 SLURM 配置中移除节点 %s", nodeName)
	return nil
}

// removeNodeFromSlurmConfig 从 SLURM 配置中移除节点并重新加载配置
func (s *SlurmClusterService) removeNodeFromSlurmConfig(nodeName string) error {
	// 获取 SLURM Master SSH 配置
	slurmMasterHost := os.Getenv("SLURM_MASTER_HOST")
	if slurmMasterHost == "" {
		slurmMasterHost = "slurm-master"
	}

	slurmMasterPort := 22
	if portStr := os.Getenv("SLURM_MASTER_PORT"); portStr != "" {
		if parsed, err := strconv.Atoi(portStr); err == nil {
			slurmMasterPort = parsed
		} else {
			logrus.Warnf("SLURM_MASTER_PORT 无法解析 (%s): %v", portStr, err)
		}
	}

	slurmMasterUser := os.Getenv("SLURM_MASTER_USER")
	if slurmMasterUser == "" {
		slurmMasterUser = "root"
	}

	var authMethods []ssh.AuthMethod
	keyPathEnv := os.Getenv("SLURM_MASTER_PRIVATE_KEY")
	keyPath := keyPathEnv
	if keyPath == "" {
		keyPath = "/root/.ssh/id_rsa"
	}

	if keyPath != "" {
		if keyData, err := os.ReadFile(keyPath); err == nil {
			if passphrase := os.Getenv("SLURM_MASTER_PRIVATE_KEY_PASSPHRASE"); passphrase != "" {
				signer, parseErr := ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
				if parseErr != nil {
					logrus.Warnf("无法解析带口令的SLURM主节点私钥 %s: %v", keyPath, parseErr)
				} else {
					authMethods = append(authMethods, ssh.PublicKeys(signer))
				}
			} else if signer, parseErr := ssh.ParsePrivateKey(keyData); parseErr == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else {
				logrus.Warnf("无法解析SLURM主节点私钥 %s: %v", keyPath, parseErr)
			}
		} else if keyPathEnv != "" || !os.IsNotExist(err) {
			logrus.Warnf("读取SLURM主节点私钥失败 (%s): %v", keyPath, err)
		}
	}

	if password := os.Getenv("SLURM_MASTER_PASSWORD"); password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("未配置SLURM Master认证方式，请设置 SLURM_MASTER_PASSWORD 或 SLURM_MASTER_PRIVATE_KEY")
	}

	sshConfig := &ssh.ClientConfig{
		User:            slurmMasterUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", slurmMasterHost, slurmMasterPort)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("创建SSH连接到SLURM Master失败: %v", err)
	}
	defer client.Close()

	logrus.Infof("已连接到 SLURM Master (%s)，开始移除节点 %s", address, nodeName)

	// 1. 设置节点为 DOWN 状态
	logrus.Infof("设置节点 %s 为 DOWN 状态", nodeName)
	session1, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason='Removed via Web UI'", nodeName)
	output1, _ := session1.CombinedOutput(downCmd)
	session1.Close()
	logrus.Infof("设置节点 DOWN 状态输出: %s", string(output1))

	// 2. 从 slurm.conf 中移除节点定义
	logrus.Infof("从 slurm.conf 中移除节点 %s 的定义", nodeName)

	// 尝试多个可能的 slurm.conf 路径
	possiblePaths := []string{
		"/etc/slurm/slurm.conf",
		"/usr/local/etc/slurm.conf",
		"/etc/slurm-llnl/slurm.conf",
	}

	var slurmConfPath string
	var confContent string

	// 查找存在的配置文件
	for _, path := range possiblePaths {
		session2, err := client.NewSession()
		if err != nil {
			continue
		}
		checkCmd := fmt.Sprintf("test -f %s && echo 'EXISTS'", path)
		output2, _ := session2.CombinedOutput(checkCmd)
		session2.Close()

		if strings.Contains(string(output2), "EXISTS") {
			slurmConfPath = path
			logrus.Infof("找到 SLURM 配置文件: %s", path)
			break
		}
	}

	if slurmConfPath == "" {
		return fmt.Errorf("未找到 SLURM 配置文件，尝试的路径: %v", possiblePaths)
	}

	// 读取配置文件
	session3, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	readCmd := fmt.Sprintf("cat %s", slurmConfPath)
	output3, err := session3.CombinedOutput(readCmd)
	session3.Close()
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}
	confContent = string(output3)

	// 处理配置内容，移除包含该节点的行
	lines := strings.Split(confContent, "\n")
	var newLines []string
	removed := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过包含该节点名称的 NodeName 定义行
		if strings.HasPrefix(trimmed, "NodeName=") && strings.Contains(trimmed, nodeName) {
			logrus.Infof("移除配置行: %s", line)
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		logrus.Warnf("在 slurm.conf 中未找到节点 %s 的定义", nodeName)
	}

	// 3. 写回配置文件
	newContent := strings.Join(newLines, "\n")

	session4, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	writeCmd := fmt.Sprintf("cat > %s << 'SLURM_CONF_EOF'\n%s\nSLURM_CONF_EOF", slurmConfPath, newContent)
	output4, err := session4.CombinedOutput(writeCmd)
	session4.Close()
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v, 输出: %s", err, string(output4))
	}

	logrus.Infof("已更新 slurm.conf 文件")

	// 4. 重新加载 SLURM 配置
	logrus.Infof("重新加载 SLURM 配置")
	session5, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	reconfigCmd := "scontrol reconfigure"
	output5, err := session5.CombinedOutput(reconfigCmd)
	session5.Close()
	if err != nil {
		logrus.Errorf("重新加载 SLURM 配置失败: %v, 输出: %s", err, string(output5))
		return fmt.Errorf("重新加载 SLURM 配置失败: %v", err)
	}

	logrus.Infof("SLURM 配置已重新加载，节点 %s 已移除", nodeName)
	return nil
}

// stopNodeServices 停止节点上的SLURM和Munge服务
func (s *SlurmClusterService) stopNodeServices(config RemoteNodeInitConfig) error {
	client, err := s.createSSHClient(config)
	if err != nil {
		return fmt.Errorf("创建SSH连接失败: %v", err)
	}
	defer client.Close()

	logrus.Infof("停止节点 %s 的服务", config.NodeName)

	// 停止SLURMD
	session1, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session1.Close()

	stopSlurmdCmd := `
		if command -v systemctl >/dev/null 2>&1; then
			systemctl stop slurmd 2>/dev/null || true
		elif command -v rc-service >/dev/null 2>&1; then
			rc-service slurmd stop 2>/dev/null || true
		else
			pkill -9 slurmd 2>/dev/null || true
		fi
	`
	session1.CombinedOutput(stopSlurmdCmd)

	// 停止Munge
	session2, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session2.Close()

	stopMungeCmd := `
		if command -v systemctl >/dev/null 2>&1; then
			systemctl stop munge 2>/dev/null || true
		elif command -v rc-service >/dev/null 2>&1; then
			rc-service munge stop 2>/dev/null || true
		else
			pkill -9 munged 2>/dev/null || true
		fi
	`
	session2.CombinedOutput(stopMungeCmd)

	logrus.Infof("节点 %s 服务已停止", config.NodeName)
	return nil
}

// GetNode 获取节点信息
func (s *SlurmClusterService) GetNode(ctx context.Context, nodeID uint) (*models.SlurmNode, error) {
	var node models.SlurmNode
	if err := s.db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// ListNodes 列出集群的所有节点
func (s *SlurmClusterService) ListNodes(ctx context.Context, clusterID uint) ([]models.SlurmNode, error) {
	var nodes []models.SlurmNode
	if err := s.db.WithContext(ctx).Where("cluster_id = ?", clusterID).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// RemoteNodeInitConfig 远程节点初始化配置
type RemoteNodeInitConfig struct {
	NodeID          uint   // 节点ID
	NodeName        string // 节点名称
	Host            string // 节点主机地址
	Port            int    // SSH端口
	Username        string // SSH用户名
	AuthType        string // 认证类型: password, key
	Password        string // SSH密码
	KeyPath         string // SSH私钥路径
	MungeKeyContent string // Munge密钥内容(base64)
	SlurmConfPath   string // SLURM配置文件路径(在master上)
	InstallPackages bool   // 是否安装软件包
}

// InitializeRemoteNode 初始化远程SLURM节点
// 该方法会通过SSH连接到远程节点，安装配置SLURM和Munge
func (s *SlurmClusterService) InitializeRemoteNode(ctx context.Context, config RemoteNodeInitConfig) error {
	logrus.Infof("开始初始化远程节点: %s (%s:%d)", config.NodeName, config.Host, config.Port)

	// 创建SSH客户端
	client, err := s.createSSHClient(config)
	if err != nil {
		return fmt.Errorf("创建SSH连接失败: %v", err)
	}
	defer client.Close()

	// 步骤1: 上传初始化脚本
	if err := s.uploadInitScript(client); err != nil {
		return fmt.Errorf("上传初始化脚本失败: %v", err)
	}

	// 步骤2: 安装软件包（可选）
	if config.InstallPackages {
		if err := s.installNodePackages(client); err != nil {
			return fmt.Errorf("安装软件包失败: %v", err)
		}
	}

	// 步骤3: 设置目录和权限
	if err := s.setupNodeDirectories(client); err != nil {
		return fmt.Errorf("设置目录失败: %v", err)
	}

	// 步骤4: 同步Munge密钥
	if config.MungeKeyContent != "" {
		if err := s.syncMungeKey(client, config.MungeKeyContent); err != nil {
			return fmt.Errorf("同步Munge密钥失败: %v", err)
		}
	}

	// 步骤5: 同步SLURM配置文件
	if config.SlurmConfPath != "" {
		if err := s.syncSlurmConfig(client, config.SlurmConfPath); err != nil {
			return fmt.Errorf("同步SLURM配置失败: %v", err)
		}
	}

	// 步骤6: 启动Munge服务
	if err := s.startMungeService(client); err != nil {
		return fmt.Errorf("启动Munge服务失败: %v", err)
	}

	// 步骤7: 启动SLURMD服务
	if err := s.startSlurmdService(client); err != nil {
		return fmt.Errorf("启动SLURMD服务失败: %v", err)
	}

	// 步骤8: 验证节点状态
	if err := s.verifyNodeStatus(client, config.NodeName); err != nil {
		return fmt.Errorf("节点验证失败: %v", err)
	}

	logrus.Infof("远程节点初始化完成: %s", config.NodeName)
	return nil
}

// createSSHClient 创建SSH客户端连接
func (s *SlurmClusterService) createSSHClient(config RemoteNodeInitConfig) (*ssh.Client, error) {
	clientConfig := &ssh.ClientConfig{
		User:            config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// 根据认证类型配置
	if config.AuthType == "password" {
		clientConfig.Auth = []ssh.AuthMethod{
			ssh.Password(config.Password),
		}
	} else if config.AuthType == "key" && config.KeyPath != "" {
		// 读取私钥
		key, err := os.ReadFile(config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取私钥文件失败: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}

		clientConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else {
		return nil, fmt.Errorf("无效的认证配置")
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败 (%s): %v", address, err)
	}

	return client, nil
}

// uploadInitScript 上传初始化脚本到远程节点
func (s *SlurmClusterService) uploadInitScript(client *ssh.Client) error {
	logrus.Info("上传初始化脚本...")

	// 读取脚本内容
	scriptPath := "/app/scripts/init-slurm-node.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取脚本文件失败: %v", err)
	}

	// 在远程创建脚本文件
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("cat > /tmp/init-slurm-node.sh && chmod +x /tmp/init-slurm-node.sh")
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	if _, err := stdin.Write(scriptContent); err != nil {
		return err
	}
	stdin.Close()

	if err := session.Wait(); err != nil {
		return err
	}

	logrus.Info("初始化脚本上传成功")
	return nil
}

// installNodePackages 在远程节点安装SLURM和Munge包
func (s *SlurmClusterService) installNodePackages(client *ssh.Client) error {
	logrus.Info("安装SLURM和Munge软件包...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := "/tmp/init-slurm-node.sh --install-packages"
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("软件包安装完成: %s", string(output))
	return nil
}

// setupNodeDirectories 设置远程节点的目录和权限
func (s *SlurmClusterService) setupNodeDirectories(client *ssh.Client) error {
	logrus.Info("设置目录和权限...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := "/tmp/init-slurm-node.sh --setup-dirs"
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("目录设置完成: %s", string(output))
	return nil
}

// syncMungeKey 同步Munge密钥到远程节点
func (s *SlurmClusterService) syncMungeKey(client *ssh.Client, keyContent string) error {
	logrus.Info("同步Munge密钥...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// keyContent已经是base64编码的
	cmd := fmt.Sprintf("/tmp/init-slurm-node.sh --munge-key '%s'", keyContent)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("Munge密钥同步完成: %s", string(output))
	return nil
}

// syncSlurmConfig 同步SLURM配置文件到远程节点
func (s *SlurmClusterService) syncSlurmConfig(client *ssh.Client, configPath string) error {
	logrus.Info("同步SLURM配置文件...")

	// 从master节点读取配置文件
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 将配置内容传输到远程节点
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("/tmp/init-slurm-node.sh --slurm-conf '%s'", string(configContent))
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("SLURM配置同步完成: %s", string(output))
	return nil
}

// startMungeService 启动远程节点的Munge服务
func (s *SlurmClusterService) startMungeService(client *ssh.Client) error {
	logrus.Info("启动Munge服务...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := "/tmp/init-slurm-node.sh --start-munge"
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("Munge服务启动成功: %s", string(output))
	return nil
}

// startSlurmdService 启动远程节点的SLURMD服务
func (s *SlurmClusterService) startSlurmdService(client *ssh.Client) error {
	logrus.Info("启动SLURMD服务...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := "/tmp/init-slurm-node.sh --start-slurmd"
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("SLURMD服务启动成功: %s", string(output))
	return nil
}

// verifyNodeStatus 验证远程节点状态
func (s *SlurmClusterService) verifyNodeStatus(client *ssh.Client, nodeName string) error {
	logrus.Info("验证节点状态...")

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("/tmp/init-slurm-node.sh --verify %s", nodeName)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("验证失败: %v, 输出: %s", err, string(output))
	}

	logrus.Infof("节点验证成功: %s", string(output))
	return nil
}

// SyncSSHKeysToNodes 同步SSH密钥到所有节点
// 从slurm-master的/root/.ssh/目录同步公钥到各个节点的authorized_keys
func (s *SlurmClusterService) SyncSSHKeysToNodes(ctx context.Context, clusterID uint, publicKeyPath string) error {
	// 查询集群及其节点
	var cluster models.SlurmCluster
	if err := s.db.Preload("Nodes").First(&cluster, clusterID).Error; err != nil {
		return fmt.Errorf("查询集群失败: %v", err)
	}

	// 读取公钥内容
	publicKeyContent, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("读取公钥文件失败: %v", err)
	}

	logrus.Infof("开始同步SSH公钥到 %d 个节点", len(cluster.Nodes))

	// 并发同步到所有节点
	var wg sync.WaitGroup
	errChan := make(chan error, len(cluster.Nodes))

	for _, node := range cluster.Nodes {
		wg.Add(1)
		go func(n models.SlurmNode) {
			defer wg.Done()

			config := RemoteNodeInitConfig{
				NodeID:   n.ID,
				NodeName: n.NodeName,
				Host:     n.Host,
				Port:     n.Port,
				Username: n.Username,
				AuthType: n.AuthType,
				Password: n.Password,
				KeyPath:  n.KeyPath,
			}

			if err := s.syncPublicKeyToNode(config, string(publicKeyContent)); err != nil {
				errChan <- fmt.Errorf("同步公钥到节点 %s 失败: %v", n.NodeName, err)
			} else {
				logrus.Infof("同步公钥到节点 %s 成功", n.NodeName)
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分节点同步失败: %s", strings.Join(errors, "; "))
	}

	logrus.Info("所有节点SSH公钥同步完成")
	return nil
}

// syncPublicKeyToNode 同步公钥到单个节点
func (s *SlurmClusterService) syncPublicKeyToNode(config RemoteNodeInitConfig, publicKey string) error {
	client, err := s.createSSHClient(config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 添加公钥到authorized_keys
	cmd := fmt.Sprintf(`
		mkdir -p ~/.ssh
		chmod 700 ~/.ssh
		echo '%s' >> ~/.ssh/authorized_keys
		chmod 600 ~/.ssh/authorized_keys
		# 去重
		sort -u ~/.ssh/authorized_keys > ~/.ssh/authorized_keys.tmp
		mv ~/.ssh/authorized_keys.tmp ~/.ssh/authorized_keys
	`, publicKey)

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("执行失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// GetMungeKeyFromMaster 从SLURM Master获取Munge密钥
func (s *SlurmClusterService) GetMungeKeyFromMaster(masterHost string) (string, error) {
	// 这里假设从本地容器的/etc/munge/munge.key读取
	// 实际部署时可能需要根据实际情况调整
	mungeKeyPath := "/etc/munge/munge.key"

	keyBytes, err := os.ReadFile(mungeKeyPath)
	if err != nil {
		return "", fmt.Errorf("读取Munge密钥失败: %v", err)
	}

	// 返回base64编码的密钥
	return base64.StdEncoding.EncodeToString(keyBytes), nil
}
