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

	// 获取总数
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
func (js *JobService) SubmitJob(ctx context.Context, req *models.SubmitJobRequest) (*models.Job, error) {
	// 验证集群是否存在
	var cluster models.Cluster
	if err := js.db.Where("id = ? AND status = 'active'", req.ClusterID).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found or inactive: %s", req.ClusterID)
		}
		return nil, fmt.Errorf("query cluster failed: %w", err)
	}

	// 解析用户ID
	userID, err := strconv.ParseUint(req.UserID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 创建作业记录
	job := &models.Job{
		UserID:     uint(userID),
		ClusterID:  req.ClusterID,
		Name:       req.Name,
		Command:    req.Command,
		WorkingDir: req.WorkingDir,
		Status:     "PENDING",
		Partition:  req.Partition,
		Nodes:      req.Nodes,
		CPUs:       req.CPUs,
		Memory:     req.Memory,
		TimeLimit:  req.TimeLimit,
		SubmitTime: time.Now(),
	}

	// 保存到数据库以获得作业ID
	if err := js.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("create job failed: %w", err)
	}

	// 设置输出文件路径
	job.StdOut = fmt.Sprintf("/tmp/slurm_job_%d.out", job.ID)
	job.StdErr = fmt.Sprintf("/tmp/slurm_job_%d.err", job.ID)
	
	// 更新作业记录中的输出路径
	if err := js.db.Save(job).Error; err != nil {
		return nil, fmt.Errorf("update job output paths failed: %w", err)
	}

	// 异步提交到SLURM
	go func() {
		if err := js.submitToSlurm(job); err != nil {
			// 更新作业状态为失败
			js.db.Model(job).Updates(map[string]interface{}{
				"status":    "FAILED",
				"updated_at": time.Now(),
			})
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

	// 上传脚本到集群
	scriptPath := fmt.Sprintf("/tmp/job_%d.sh", job.ID)
	if err := js.sshSvc.UploadFile(cluster.Host, cluster.Port, "root", "", []byte(script), scriptPath); err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to upload script: %v", err))
		return fmt.Errorf("upload script failed: %w", err)
	}

	// 设置脚本可执行权限
	chmodCmd := fmt.Sprintf("chmod +x %s", scriptPath)
	if _, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", chmodCmd); err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to set script permissions: %v", err))
		return fmt.Errorf("set script permissions failed: %w", err)
	}

	// 提交作业
	cmd := fmt.Sprintf("sbatch %s", scriptPath)
	output, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
	if err != nil {
		js.updateJobStatus(job, "FAILED", fmt.Sprintf("Failed to submit job: %v", err))
		return fmt.Errorf("submit job failed: %w", err)
	}

	// 解析作业ID
	jobIDStr := strings.TrimSpace(output)
	
	// SLURM sbatch 通常返回 "Submitted batch job JOBID"
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
		// 即使数据库更新失败，作业已经提交了，记录警告但不返回错误
		fmt.Printf("Warning: failed to update job record: %v\n", err)
	}

	// 清理临时脚本文件
	cleanupCmd := fmt.Sprintf("rm -f %s", scriptPath)
	if _, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cleanupCmd); err != nil {
		// 清理失败不应该影响作业提交
		fmt.Printf("Warning: failed to cleanup script file: %v\n", err)
	}

	return nil
}

// updateJobStatus 更新作业状态的辅助方法
func (js *JobService) updateJobStatus(job *models.Job, status, message string) {
	job.Status = status
	job.UpdatedAt = time.Now()
	
	if status == "FAILED" && message != "" {
		// 可以添加一个错误消息字段到 Job 模型中
		fmt.Printf("Job %d failed: %s\n", job.ID, message)
	}
	
	js.db.Save(job)
}

// buildSlurmScript 构建SLURM作业脚本
func (js *JobService) buildSlurmScript(job *models.Job) string {
	script := fmt.Sprintf(`#!/bin/bash
#SBATCH --job-name=%s`, job.Name)

	// 设置输出和错误文件
	if job.StdOut != "" {
		script += fmt.Sprintf("\n#SBATCH --output=%s", job.StdOut)
	}
	if job.StdErr != "" {
		script += fmt.Sprintf("\n#SBATCH --error=%s", job.StdErr)
	}

	// 设置分区
	if job.Partition != "" {
		script += fmt.Sprintf("\n#SBATCH --partition=%s", job.Partition)
	}

	// 设置节点数
	if job.Nodes > 0 {
		script += fmt.Sprintf("\n#SBATCH --nodes=%d", job.Nodes)
	}

	// 设置CPU数量
	if job.CPUs > 0 {
		script += fmt.Sprintf("\n#SBATCH --ntasks=%d", job.CPUs)
	}

	// 设置内存
	if job.Memory != "" {
		script += fmt.Sprintf("\n#SBATCH --mem=%s", job.Memory)
	}

	// 设置时间限制
	if job.TimeLimit != "" {
		script += fmt.Sprintf("\n#SBATCH --time=%s", job.TimeLimit)
	}

	// 设置工作目录
	if job.WorkingDir != "" {
		script += fmt.Sprintf("\n#SBATCH --chdir=%s", job.WorkingDir)
	}

	// 添加空行和用户命令
	script += "\n\n# User command\n" + job.Command + "\n"

	return script
}

// CancelJob 取消作业
func (js *JobService) CancelJob(ctx context.Context, userID, clusterID string, jobID uint) error {
	// 验证作业所有权
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

	// 发送取消命令到SLURM
	cmd := fmt.Sprintf("scancel %d", job.JobID)
	_, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
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

	output := &models.JobOutput{
		JobID:    job.JobID,
		ExitCode: job.ExitCode,
	}

	// 获取标准输出
	if job.StdOut != "" {
		stdout, err := js.sshSvc.ReadFile(cluster.Host, cluster.Port, "root", "", job.StdOut)
		if err == nil {
			output.StdOut = stdout
		}
	}

	// 获取标准错误
	if job.StdErr != "" {
		stderr, err := js.sshSvc.ReadFile(cluster.Host, cluster.Port, "root", "", job.StdErr)
		if err == nil {
			output.StdErr = stderr
		}
	}

	return output, nil
}

// GetDashboardStats 获取仪表板统计信息
func (js *JobService) GetDashboardStats(ctx context.Context, userID string) (*models.JobDashboardStats, error) {
	userIDUint, _ := strconv.ParseUint(userID, 10, 32)

	var stats models.JobDashboardStats

	// 作业统计
	js.db.Model(&models.Job{}).Where("user_id = ?", userIDUint).Count(&stats.TotalJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = 'RUNNING'", userIDUint).Count(&stats.RunningJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = 'PENDING'", userIDUint).Count(&stats.PendingJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = 'COMPLETED'", userIDUint).Count(&stats.CompletedJobs)
	js.db.Model(&models.Job{}).Where("user_id = ? AND status = 'FAILED'", userIDUint).Count(&stats.FailedJobs)

	// 集群统计
	js.db.Model(&models.Cluster{}).Count(&stats.TotalClusters)
	js.db.Model(&models.Cluster{}).Where("status = 'active'").Count(&stats.ActiveClusters)

	return &stats, nil
}

// GetJobStatus 获取作业状态
func (js *JobService) GetJobStatus(ctx context.Context, jobID uint) (*models.JobStatus, error) {
	// 获取作业信息
	var job models.Job
	if err := js.db.Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("query job failed: %w", err)
	}

	// 获取集群信息
	var cluster models.Cluster
	if err := js.db.Where("id = ?", job.ClusterID).First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("get cluster info failed: %w", err)
	}

	// 查询SLURM作业状态
	cmd := fmt.Sprintf("squeue -h -j %d -o '%%T'", job.JobID)
	output, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
	if err != nil {
		// 如果squeue失败，可能是作业已完成，尝试sacct
		cmd = fmt.Sprintf("sacct -j %d --format=State -n", job.JobID)
		output, err = js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
		if err != nil {
			return nil, fmt.Errorf("query job status failed: %w", err)
		}
	}

	state := strings.TrimSpace(output)
	if state == "" {
		state = "UNKNOWN"
	}

	// 更新数据库状态
	job.Status = state
	job.UpdatedAt = time.Now()
	js.db.Save(&job)

	return &models.JobStatus{
		JobID: job.JobID,
		State: state,
	}, nil
}

// GetClusterInfo 获取集群详细信息
func (js *JobService) GetClusterInfo(ctx context.Context, clusterID string) (*models.ClusterInfo, error) {
	var cluster models.Cluster
	if err := js.db.Where("id = ? AND status = 'active'", clusterID).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found")
		}
		return nil, fmt.Errorf("query cluster failed: %w", err)
	}

	// 这里可以添加获取SLURM分区信息的逻辑
	// 暂时返回基本信息
	clusterInfo := &models.ClusterInfo{
		ID:          cluster.ID,
		Name:        cluster.Name,
		Description: cluster.Description,
		Status:      cluster.Status,
		Partitions:  []models.PartitionInfo{}, // TODO: 实现分区信息获取
	}

	return clusterInfo, nil
}

// ListClusters 获取集群列表
func (js *JobService) ListClusters(ctx context.Context) ([]models.Cluster, error) {
	var clusters []models.Cluster
	if err := js.db.Where("status = 'active'").Find(&clusters).Error; err != nil {
		return nil, fmt.Errorf("query clusters failed: %w", err)
	}

	return clusters, nil
}