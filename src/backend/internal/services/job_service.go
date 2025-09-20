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

	// 保存到数据库
	if err := js.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("create job failed: %w", err)
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
		return fmt.Errorf("get cluster info failed: %w", err)
	}

	// 上传脚本到集群
	scriptPath := fmt.Sprintf("/tmp/job_%d.sh", job.ID)
	if err := js.sshSvc.UploadFile(cluster.Host, cluster.Port, "root", "", []byte(script), scriptPath); err != nil {
		return fmt.Errorf("upload script failed: %w", err)
	}

	// 提交作业
	cmd := fmt.Sprintf("sbatch %s", scriptPath)
	output, err := js.sshSvc.ExecuteCommand(cluster.Host, cluster.Port, "root", "", cmd)
	if err != nil {
		return fmt.Errorf("submit job failed: %w", err)
	}

	// 解析作业ID
	jobIDStr := strings.TrimSpace(strings.TrimPrefix(output, "Submitted batch job "))
	jobID, err := strconv.ParseUint(jobIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("parse job ID failed: %w", err)
	}

	// 更新作业记录
	job.JobID = uint32(jobID)
	job.Status = "PENDING"
	job.UpdatedAt = time.Now()

	if err := js.db.Save(job).Error; err != nil {
		return fmt.Errorf("update job failed: %w", err)
	}

	return nil
}

// buildSlurmScript 构建SLURM作业脚本
func (js *JobService) buildSlurmScript(job *models.Job) string {
	script := fmt.Sprintf(`#!/bin/bash
#SBATCH --job-name=%s
#SBATCH --output=%s
#SBATCH --error=%s
`, job.Name, job.StdOut, job.StdErr)

	if job.Partition != "" {
		script += fmt.Sprintf("#SBATCH --partition=%s\n", job.Partition)
	}

	if job.Nodes > 0 {
		script += fmt.Sprintf("#SBATCH --nodes=%d\n", job.Nodes)
	}

	if job.CPUs > 0 {
		script += fmt.Sprintf("#SBATCH --ntasks=%d\n", job.CPUs)
	}

	if job.Memory != "" {
		script += fmt.Sprintf("#SBATCH --mem=%s\n", job.Memory)
	}

	if job.TimeLimit != "" {
		script += fmt.Sprintf("#SBATCH --time=%s\n", job.TimeLimit)
	}

	if job.WorkingDir != "" {
		script += fmt.Sprintf("#SBATCH --chdir=%s\n", job.WorkingDir)
	}

	script += "\n" + job.Command + "\n"

	return script
}

// CancelJob 取消作业
func (js *JobService) CancelJob(ctx context.Context, userID, clusterID string, jobID uint32) error {
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
func (js *JobService) GetJobDetail(ctx context.Context, userID, clusterID string, jobID uint32) (*models.Job, error) {
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
func (js *JobService) GetJobOutput(ctx context.Context, userID, clusterID string, jobID uint32) (*models.JobOutput, error) {
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

// ListClusters 获取集群列表
func (js *JobService) ListClusters(ctx context.Context) ([]models.Cluster, error) {
	var clusters []models.Cluster
	if err := js.db.Where("status = 'active'").Find(&clusters).Error; err != nil {
		return nil, fmt.Errorf("query clusters failed: %w", err)
	}

	return clusters, nil
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