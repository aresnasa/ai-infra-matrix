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
