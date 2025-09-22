package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// JobTemplateService 作业模板管理服务
type JobTemplateService struct {
	db *gorm.DB
}

// NewJobTemplateService 创建作业模板服务
func NewJobTemplateService(db *gorm.DB) *JobTemplateService {
	return &JobTemplateService{
		db: db,
	}
}

// CreateTemplate 创建作业模板
func (jts *JobTemplateService) CreateTemplate(ctx context.Context, userID uint, req *models.CreateJobTemplateRequest) (*models.JobTemplate, error) {
	// 处理标签
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	template := &models.JobTemplate{
		Name:        req.Name,
		Description: req.Description,
		Script:      req.Script,
		Command:     req.Command,
		Partition:   req.Partition,
		Nodes:       req.Nodes,
		CPUs:        req.CPUs,
		Memory:      req.Memory,
		TimeLimit:   req.TimeLimit,
		WorkingDir:  req.WorkingDir,
		IsPublic:    req.IsPublic,
		Category:    req.Category,
		Tags:        string(tagsJSON),
		UserID:      userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := jts.db.Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// GetTemplate 获取单个作业模板
func (jts *JobTemplateService) GetTemplate(ctx context.Context, id uint, userID uint) (*models.JobTemplate, error) {
	var template models.JobTemplate
	query := jts.db.Preload("User")

	// 用户只能看到自己的私有模板或所有公开模板
	query = query.Where("id = ? AND (user_id = ? OR is_public = ?)", id, userID, true)

	if err := query.First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found or access denied")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &template, nil
}

// ListTemplates 获取作业模板列表
func (jts *JobTemplateService) ListTemplates(ctx context.Context, userID uint, category string, isPublic *bool, page, pageSize int) ([]models.JobTemplate, int64, error) {
	var templates []models.JobTemplate
	var total int64

	query := jts.db.Model(&models.JobTemplate{}).Preload("User")

	// 用户只能看到自己的模板或公开模板
	query = query.Where("user_id = ? OR is_public = ?", userID, true)

	// 分类筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 公开/私有筛选
	if isPublic != nil {
		if *isPublic {
			query = query.Where("is_public = ?", true)
		} else {
			query = query.Where("user_id = ? AND is_public = ?", userID, false)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count templates: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, total, nil
}

// UpdateTemplate 更新作业模板
func (jts *JobTemplateService) UpdateTemplate(ctx context.Context, id uint, userID uint, req *models.UpdateJobTemplateRequest) (*models.JobTemplate, error) {
	var template models.JobTemplate
	
	// 检查模板是否存在且用户有权限修改
	if err := jts.db.Where("id = ? AND user_id = ?", id, userID).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found or access denied")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// 处理标签
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	// 更新字段
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"script":      req.Script,
		"command":     req.Command,
		"partition":   req.Partition,
		"nodes":       req.Nodes,
		"cpus":        req.CPUs,
		"memory":      req.Memory,
		"time_limit":  req.TimeLimit,
		"working_dir": req.WorkingDir,
		"is_public":   req.IsPublic,
		"category":    req.Category,
		"tags":        string(tagsJSON),
		"updated_at":  time.Now(),
	}

	if err := jts.db.Model(&template).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// 重新加载模板
	if err := jts.db.Preload("User").First(&template, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload template: %w", err)
	}

	return &template, nil
}

// DeleteTemplate 删除作业模板
func (jts *JobTemplateService) DeleteTemplate(ctx context.Context, id uint, userID uint) error {
	result := jts.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.JobTemplate{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete template: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found or access denied")
	}

	return nil
}

// GetTemplateCategories 获取模板分类列表
func (jts *JobTemplateService) GetTemplateCategories(ctx context.Context, userID uint) ([]string, error) {
	var categories []string
	
	if err := jts.db.Model(&models.JobTemplate{}).
		Select("DISTINCT category").
		Where("(user_id = ? OR is_public = ?) AND category != ''", userID, true).
		Pluck("category", &categories).Error; err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return categories, nil
}

// CreateJobFromTemplate 从模板创建作业
func (jts *JobTemplateService) CreateJobFromTemplate(ctx context.Context, templateID uint, userID uint, jobRequest *models.SubmitJobRequest) (*models.JobTemplate, error) {
	template, err := jts.GetTemplate(ctx, templateID, userID)
	if err != nil {
		return nil, err
	}

	// 将模板值应用到作业请求中（如果请求中没有设置）
	if jobRequest.Command == "" {
		jobRequest.Command = template.Command
	}
	if jobRequest.Partition == "" {
		jobRequest.Partition = template.Partition
	}
	if jobRequest.Nodes == 0 {
		jobRequest.Nodes = template.Nodes
	}
	if jobRequest.CPUs == 0 {
		jobRequest.CPUs = template.CPUs
	}
	if jobRequest.Memory == "" {
		jobRequest.Memory = template.Memory
	}
	if jobRequest.TimeLimit == "" {
		jobRequest.TimeLimit = template.TimeLimit
	}
	if jobRequest.WorkingDir == "" {
		jobRequest.WorkingDir = template.WorkingDir
	}

	return template, nil
}

// ParseTemplateTags 解析模板标签
func (jts *JobTemplateService) ParseTemplateTags(template *models.JobTemplate) ([]string, error) {
	if template.Tags == "" {
		return []string{}, nil
	}

	var tags []string
	if err := json.Unmarshal([]byte(template.Tags), &tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}

	return tags, nil
}

// GenerateScriptFromTemplate 根据模板和参数生成脚本
func (jts *JobTemplateService) GenerateScriptFromTemplate(template *models.JobTemplate, params map[string]string) string {
	script := template.Script

	// 替换脚本中的占位符
	for key, value := range params {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		script = strings.ReplaceAll(script, placeholder, value)
	}

	// 如果模板中没有完整脚本，则生成基本的 sbatch 脚本
	if script == "" {
		script = jts.generateBasicScript(template, params)
	}

	return script
}

// generateBasicScript 生成基本的 sbatch 脚本
func (jts *JobTemplateService) generateBasicScript(template *models.JobTemplate, params map[string]string) string {
	script := "#!/bin/bash\n"
	
	if template.Name != "" {
		script += fmt.Sprintf("#SBATCH --job-name=%s\n", template.Name)
	}
	
	if template.Partition != "" {
		script += fmt.Sprintf("#SBATCH --partition=%s\n", template.Partition)
	}
	
	if template.Nodes > 0 {
		script += fmt.Sprintf("#SBATCH --nodes=%d\n", template.Nodes)
	}
	
	if template.CPUs > 0 {
		script += fmt.Sprintf("#SBATCH --ntasks=%d\n", template.CPUs)
	}
	
	if template.Memory != "" {
		script += fmt.Sprintf("#SBATCH --mem=%s\n", template.Memory)
	}
	
	if template.TimeLimit != "" {
		script += fmt.Sprintf("#SBATCH --time=%s\n", template.TimeLimit)
	}
	
	if template.WorkingDir != "" {
		script += fmt.Sprintf("#SBATCH --chdir=%s\n", template.WorkingDir)
	}
	
	// 添加输出文件（通过参数传入）
	if jobID, ok := params["JobID"]; ok {
		script += fmt.Sprintf("#SBATCH --output=/tmp/slurm_job_%s.out\n", jobID)
		script += fmt.Sprintf("#SBATCH --error=/tmp/slurm_job_%s.err\n", jobID)
	}
	
	script += "\n# User command\n"
	if template.Command != "" {
		script += template.Command + "\n"
	}
	
	return script
}