package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// JobService 作业管理服务
type JobService struct {
	db         *gorm.DB
	slurmSvc   *SlurmService
	sshSvc     *SSHService
	cacheSvc   CacheService
}

// NewJobService 创建作业服务
func NewJobService(db *gorm.DB, slurmSvc *SlurmService, sshSvc *SSHService, cacheSvc CacheService) *JobService {
	return &JobService{
		db:       db,
		slurmSvc: slurmSvc,
		sshSvc:   sshSvc,
		cacheSvc: cacheSvc,
	}
}

// ListJobs 获取作业列表
func (js *JobService) ListJobs(ctx context.Context, userID, clusterID, status string, page, pageSize int) ([]models.Job, int64, error) {
	var jobs []models.Job
	var total int64

	query := js.db.Model(&models.Job{}).Where("user_id = ?", userID)

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count jobs failed: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Cluster").
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&jobs).Error; err != nil {
		return nil, 0, fmt.Errorf("query jobs failed: %w", err)
	}

	return jobs, total, nil
}

// SubmitJob 提交作业
func (js *JobService) SubmitJob(ctx context.Context, req *models.SubmitJobRequest, userID uint) (*models.Job, error) {
	// 创建作业记录
	job := &models.Job{
		UserID:    userID,
		ClusterID: req.ClusterID,
		Name:      req.Name,
		Command:   req.Command,
		Partition: req.Partition,
		Nodes:     req.Nodes,
		CPUs:      req.CPUs,
		Memory:    req.Memory,
		TimeLimit: req.TimeLimit,
		StdOut:    fmt.Sprintf("/tmp/%s-%%j.out", req.Name),
		StdErr:    fmt.Sprintf("/tmp/%s-%%j.err", req.Name),
		Status:    "PENDING",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存到数据库
	if err := js.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("create job failed: %w", err)
	}

	// 异步提交到SLURM
	go func() {
		if err := js.submitToSlurm(job); err != nil {
			fmt.Printf("Submit job to SLURM failed: %v\n", err)
		}
	}()

	return job, nil
}

// submitToSlurm 提交作业到SLURM
func (js *JobService) submitToSlurm(job *models.Job) error {
	// 构建SLURM作业脚本
	script := js.buildSlurmScript(job)

	// 通过SSH提交到集群
	cluster := &models.Cluster{ID: job.ClusterID}
	if err := js.db.First(cluster).Error; err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to get cluster info: %v", err))
		return fmt.Errorf("get cluster info failed: %w", err)
	}

	// 获取集群认证信息
	username, password := js.getClusterAuth(cluster)

	// 上传脚本到集群
	scriptPath := fmt.Sprintf("/tmp/job_%d.sh", job.ID)
	if err := js.sshSvc.UploadFile(cluster.Host, cluster.Port, username, password, []byte(script), scriptPath); err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to upload script: %v", err))
		return fmt.Errorf("upload script failed: %w", err)
	}

	// 设置脚本可执行权限
	chmodCmd := fmt.Sprintf("chmod +x %s", scriptPath)
	if _, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, chmodCmd); err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to set script permissions: %v", err))
		return fmt.Errorf("set script permissions failed: %w", err)
	}

	// 提交作业
	cmd := fmt.Sprintf("sbatch %s", scriptPath)
	output, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, cmd)
	if err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to submit job: %v", err))
		return fmt.Errorf("submit job failed: %w", err)
	}

	// 解析作业ID
	jobIDStr := strings.TrimSpace(output)
	if strings.HasPrefix(jobIDStr, "Submitted batch job ") {
		jobIDStr = strings.TrimPrefix(jobIDStr, "Submitted batch job ")
	}
	
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to parse job ID from output: %s", output))
		return fmt.Errorf("parse job ID failed: %w", err)
	}

	// 更新作业记录
	job.JobID = uint32(jobID)
	job.Status = "SUBMITTED"
	job.UpdatedAt = time.Now()

	if err := js.db.Save(job).Error; err != nil {
		fmt.Printf("Warning: failed to update job record: %v\n", err)
	}

	// 清理临时脚本文件
	cleanupCmd := fmt.Sprintf("rm -f %s", scriptPath)
	if _, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, cleanupCmd); err != nil {
		fmt.Printf("Warning: failed to cleanup script file: %v\n", err)
	}

	return nil
}

// getClusterAuth 获取集群认证信息
func (js *JobService) getClusterAuth(cluster *models.Cluster) (string, string) {
	if cluster.Username != "" {
		return cluster.Username, cluster.Password
	}
	
	var node models.SlurmNode
	err := js.db.Where("cluster_id = ? AND status = 'active'", cluster.ID).First(&node).Error
	if err == nil && node.Username != "" {
		return node.Username, node.Password
	}
	
	return "root", ""
}

// updateJobStatus 更新作业状态的辅助方法
func (js *JobService) updateJobStatus(job *models.Job, status, message string) {
	job.Status = status
	job.UpdatedAt = time.Now()
	
	if status == "FAILED" && message != "" {
		fmt.Printf("Job %d failed: %s\n", job.ID, message)
	}
	
	js.db.Save(job)
}

// buildSlurmScript 构建SLURM作业脚本
func (js *JobService) buildSlurmScript(job *models.Job) string {
	script := fmt.Sprintf(`#!/bin/bash
#SBATCH --job-name=%s`, job.Name)

	if job.StdOut != "" {
		script += fmt.Sprintf("\n#SBATCH --output=%s", job.StdOut)
	}
	if job.StdErr != "" {
		script += fmt.Sprintf("\n#SBATCH --error=%s", job.StdErr)
	}
	if job.Partition != "" {
		script += fmt.Sprintf("\n#SBATCH --partition=%s", job.Partition)
	}
	if job.Nodes > 0 {
		script += fmt.Sprintf("\n#SBATCH --nodes=%d", job.Nodes)
	}
	if job.CPUs > 0 {
		script += fmt.Sprintf("\n#SBATCH --cpus-per-task=%d", job.CPUs)
	}
	if job.Memory != "" {
		script += fmt.Sprintf("\n#SBATCH --mem=%s", job.Memory)
	}
	if job.TimeLimit != "" {
		script += fmt.Sprintf("\n#SBATCH --time=%s", job.TimeLimit)
	}

	script += fmt.Sprintf("\n\n# 作业执行内容\n%s", job.Command)

	return script
}

// GetJobStatus 获取作业状态
func (js *JobService) GetJobStatus(ctx context.Context, jobID uint) (*models.JobStatus, error) {
	var job models.Job
	if err := js.db.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("query job failed: %w", err)
	}

	var cluster models.Cluster
	if err := js.db.Where("id = ?", job.ClusterID).First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("get cluster info failed: %w", err)
	}

	username, password := js.getClusterAuth(&cluster)

	cmd := fmt.Sprintf("squeue -h -j %d -o '%%T'", job.JobID)
	output, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, cmd)
	if err != nil {
		cmd = fmt.Sprintf("sacct -j %d --format=State -n", job.JobID)
		output, err = js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, cmd)
		if err != nil {
			return nil, fmt.Errorf("query job status failed: %w", err)
		}
	}

	state := strings.TrimSpace(output)
	if state == "" {
		state = "UNKNOWN"
	}

	job.Status = state
	job.UpdatedAt = time.Now()
	js.db.Save(&job)

	return &models.JobStatus{
		JobID: job.JobID,
		State: state,
	}, nil
}

// CancelJob 取消作业
func (js *JobService) CancelJob(ctx context.Context, userID, clusterID string, jobID uint) error {
	var job models.Job
	userIDUint, _ := strconv.ParseUint(userID, 10, 32)

	if err := js.db.Where("id = ? AND user_id = ? AND cluster_id = ?", jobID, userIDUint, clusterID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("job not found or access denied")
		}
		return fmt.Errorf("query job failed: %w", err)
	}

	// 获取集群信息
	var cluster models.Cluster
	if err := js.db.Where("id = ?", clusterID).First(&cluster).Error; err != nil {
		return fmt.Errorf("get cluster info failed: %w", err)
	}

	// 获取认证信息
	username, password := js.getClusterAuth(&cluster)

	// 发送取消命令到SLURM
	cmd := fmt.Sprintf("scancel %d", job.JobID)
	_, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, username, password, cmd)
	if err != nil {
		return fmt.Errorf("cancel job failed: %w", err)
	}

	// 更新作业状态
	job.Status = "CANCELLED"
	job.EndTime = &time.Time{}
	*job.EndTime = time.Now()
	job.UpdatedAt = time.Now()

	if err := js.db.Save(&job).Error; err != nil {
		return fmt.Errorf("update job status failed: %w", err)
	}

	return nil
}

// GetJobDetail 获取作业详情
func (js *JobService) GetJobDetail(ctx context.Context, userID, clusterID string, jobID uint) (*models.Job, error) {
	var job models.Job
	userIDUint, _ := strconv.ParseUint(userID, 10, 32)

	if err := js.db.Where("id = ? AND user_id = ? AND cluster_id = ?", jobID, userIDUint, clusterID).
		Preload("User").Preload("Cluster").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found or access denied")
		}
		return nil, fmt.Errorf("query job failed: %w", err)
	}

	return &job, nil
}

// GetJobOutput 获取作业输出
func (js *JobService) GetJobOutput(ctx context.Context, userID, clusterID string, jobID uint) (*models.JobOutput, error) {
	job, err := js.GetJobDetail(ctx, userID, clusterID, jobID)
	if err != nil {
		return nil, err
	}

	// 获取集群信息
	var cluster models.Cluster
	if err := js.db.Where("id = ?", clusterID).First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("get cluster info failed: %w", err)
	}

	// 获取认证信息
	username, password := js.getClusterAuth(&cluster)

	// 读取输出文件
	var stdout, stderr string
	if job.StdOut != "" {
		stdoutPath := strings.ReplaceAll(job.StdOut, "%j", fmt.Sprintf("%d", job.JobID))
		if content, err := js.sshSvc.ReadFile(cluster.Host, cluster.Port, username, password, stdoutPath); err == nil {
			stdout = content
		}
	}

	if job.StdErr != "" {
		stderrPath := strings.ReplaceAll(job.StdErr, "%j", fmt.Sprintf("%d", job.JobID))
		if content, err := js.sshSvc.ReadFile(cluster.Host, cluster.Port, username, password, stderrPath); err == nil {
			stderr = content
		}
	}

	return &models.JobOutput{
		JobID:  uint32(job.ID),
		StdOut: stdout,
		StdErr: stderr,
	}, nil
}

// GetDashboardStats 获取仪表板统计信息
func (js *JobService) GetDashboardStats(ctx context.Context, userID string) (*models.JobDashboardStats, error) {
	userIDUint, _ := strconv.ParseUint(userID, 10, 32)

	var stats models.JobDashboardStats

	// 统计不同状态的作业数量
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = ?", userIDUint, "RUNNING").Count(&stats.RunningJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = ?", userIDUint, "PENDING").Count(&stats.PendingJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = ?", userIDUint, "COMPLETED").Count(&stats.CompletedJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = ?", userIDUint, "FAILED").Count(&stats.FailedJobs)

	// 计算总作业数
	stats.TotalJobs = stats.RunningJobs + stats.PendingJobs + stats.CompletedJobs + stats.FailedJobs

	return &stats, nil
}

// ListClusters 列出可用集群
func (js *JobService) ListClusters(ctx context.Context) ([]models.Cluster, error) {
	var clusters []models.Cluster
	if err := js.db.Where("status = ?", "active").Find(&clusters).Error; err != nil {
		return nil, fmt.Errorf("query clusters failed: %w", err)
	}
	return clusters, nil
}