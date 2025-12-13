package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// JupyterLabTemplateService JupyterLab模板服务接口
type JupyterLabTemplateService interface {
	// 模板管理
	CreateTemplate(template *models.JupyterLabTemplate) error
	GetTemplate(id uint) (*models.JupyterLabTemplate, error)
	GetTemplateByName(name string) (*models.JupyterLabTemplate, error)
	ListTemplates(userID uint, includeInactive bool) ([]models.JupyterLabTemplate, error)
	UpdateTemplate(template *models.JupyterLabTemplate) error
	DeleteTemplate(id uint) error
	SetDefaultTemplate(id uint) error

	// 资源配额管理
	CreateResourceQuota(quota *models.JupyterLabResourceQuota) error
	GetResourceQuota(templateID uint) (*models.JupyterLabResourceQuota, error)
	UpdateResourceQuota(quota *models.JupyterLabResourceQuota) error
	DeleteResourceQuota(templateID uint) error

	// 实例管理
	CreateInstance(instance *models.JupyterLabInstance) error
	GetInstance(id uint) (*models.JupyterLabInstance, error)
	ListUserInstances(userID uint) ([]models.JupyterLabInstance, error)
	UpdateInstance(instance *models.JupyterLabInstance) error
	DeleteInstance(id uint) error

	// 模板克隆和导入导出
	CloneTemplate(templateID uint, newName string, userID uint) (*models.JupyterLabTemplate, error)
	ExportTemplate(id uint) (map[string]interface{}, error)
	ImportTemplate(data map[string]interface{}, userID uint) (*models.JupyterLabTemplate, error)

	// 预定义模板
	CreatePredefinedTemplates() error
}

// jupyterLabTemplateServiceImpl 实现
type jupyterLabTemplateServiceImpl struct {
	db *gorm.DB
}

// NewJupyterLabTemplateService 创建服务实例
func NewJupyterLabTemplateService() JupyterLabTemplateService {
	return &jupyterLabTemplateServiceImpl{
		db: database.GetDB(),
	}
}

// CreateTemplate 创建模板
func (s *jupyterLabTemplateServiceImpl) CreateTemplate(template *models.JupyterLabTemplate) error {
	// 如果设置为默认模板，先取消其他默认模板
	if template.IsDefault {
		s.db.Model(&models.JupyterLabTemplate{}).Where("is_default = ?", true).Update("is_default", false)
	}

	return s.db.Create(template).Error
}

// GetTemplate 获取模板
func (s *jupyterLabTemplateServiceImpl) GetTemplate(id uint) (*models.JupyterLabTemplate, error) {
	var template models.JupyterLabTemplate
	err := s.db.Preload("ResourceQuota").First(&template, id).Error
	return &template, err
}

// GetTemplateByName 根据名称获取模板
func (s *jupyterLabTemplateServiceImpl) GetTemplateByName(name string) (*models.JupyterLabTemplate, error) {
	var template models.JupyterLabTemplate
	err := s.db.Preload("ResourceQuota").Where("name = ?", name).First(&template).Error
	return &template, err
}

// ListTemplates 列出模板
func (s *jupyterLabTemplateServiceImpl) ListTemplates(userID uint, includeInactive bool) ([]models.JupyterLabTemplate, error) {
	var templates []models.JupyterLabTemplate
	query := s.db.Preload("ResourceQuota")

	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	// 用户只能看到自己创建的和公共的模板
	if userID > 0 {
		query = query.Where("created_by = ? OR created_by = 0", userID)
	}

	err := query.Order("is_default DESC, created_at DESC").Find(&templates).Error
	return templates, err
}

// UpdateTemplate 更新模板
func (s *jupyterLabTemplateServiceImpl) UpdateTemplate(template *models.JupyterLabTemplate) error {
	// 如果设置为默认模板，先取消其他默认模板
	if template.IsDefault {
		s.db.Model(&models.JupyterLabTemplate{}).Where("id != ? AND is_default = ?", template.ID, true).Update("is_default", false)
	}

	return s.db.Save(template).Error
}

// DeleteTemplate 删除模板
func (s *jupyterLabTemplateServiceImpl) DeleteTemplate(id uint) error {
	// 软删除模板和相关资源配额
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.JupyterLabResourceQuota{}, "template_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&models.JupyterLabTemplate{}, id).Error
	})
}

// SetDefaultTemplate 设置默认模板
func (s *jupyterLabTemplateServiceImpl) SetDefaultTemplate(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 取消其他默认模板
		if err := tx.Model(&models.JupyterLabTemplate{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		// 设置新的默认模板
		return tx.Model(&models.JupyterLabTemplate{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

// CreateResourceQuota 创建资源配额
func (s *jupyterLabTemplateServiceImpl) CreateResourceQuota(quota *models.JupyterLabResourceQuota) error {
	return s.db.Create(quota).Error
}

// GetResourceQuota 获取资源配额
func (s *jupyterLabTemplateServiceImpl) GetResourceQuota(templateID uint) (*models.JupyterLabResourceQuota, error) {
	var quota models.JupyterLabResourceQuota
	err := s.db.Where("template_id = ?", templateID).First(&quota).Error
	return &quota, err
}

// UpdateResourceQuota 更新资源配额
func (s *jupyterLabTemplateServiceImpl) UpdateResourceQuota(quota *models.JupyterLabResourceQuota) error {
	return s.db.Save(quota).Error
}

// DeleteResourceQuota 删除资源配额
func (s *jupyterLabTemplateServiceImpl) DeleteResourceQuota(templateID uint) error {
	return s.db.Where("template_id = ?", templateID).Delete(&models.JupyterLabResourceQuota{}).Error
}

// CreateInstance 创建实例
func (s *jupyterLabTemplateServiceImpl) CreateInstance(instance *models.JupyterLabInstance) error {
	return s.db.Create(instance).Error
}

// GetInstance 获取实例
func (s *jupyterLabTemplateServiceImpl) GetInstance(id uint) (*models.JupyterLabInstance, error) {
	var instance models.JupyterLabInstance
	err := s.db.Preload("Template").Preload("Template.ResourceQuota").Preload("User").First(&instance, id).Error
	return &instance, err
}

// ListUserInstances 列出用户实例
func (s *jupyterLabTemplateServiceImpl) ListUserInstances(userID uint) ([]models.JupyterLabInstance, error) {
	var instances []models.JupyterLabInstance
	err := s.db.Preload("Template").Where("user_id = ?", userID).Order("created_at DESC").Find(&instances).Error
	return instances, err
}

// UpdateInstance 更新实例
func (s *jupyterLabTemplateServiceImpl) UpdateInstance(instance *models.JupyterLabInstance) error {
	return s.db.Save(instance).Error
}

// DeleteInstance 删除实例
func (s *jupyterLabTemplateServiceImpl) DeleteInstance(id uint) error {
	return s.db.Delete(&models.JupyterLabInstance{}, id).Error
}

// CloneTemplate 克隆模板
func (s *jupyterLabTemplateServiceImpl) CloneTemplate(templateID uint, newName string, userID uint) (*models.JupyterLabTemplate, error) {
	// 获取原模板
	original, err := s.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// 创建克隆模板
	cloned := &models.JupyterLabTemplate{
		Name:            newName,
		Description:     fmt.Sprintf("克隆自: %s", original.Name),
		PythonVersion:   original.PythonVersion,
		CondaVersion:    original.CondaVersion,
		BaseImage:       original.BaseImage,
		Requirements:    original.Requirements,
		CondaPackages:   original.CondaPackages,
		SystemPackages:  original.SystemPackages,
		EnvironmentVars: original.EnvironmentVars,
		StartupScript:   original.StartupScript,
		IsActive:        true,
		IsDefault:       false,
		CreatedBy:       userID,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 创建模板
		if err := tx.Create(cloned).Error; err != nil {
			return err
		}

		// 克隆资源配额
		if original.ResourceQuota != nil {
			clonedQuota := &models.JupyterLabResourceQuota{
				TemplateID:    cloned.ID,
				CPULimit:      original.ResourceQuota.CPULimit,
				CPURequest:    original.ResourceQuota.CPURequest,
				MemoryLimit:   original.ResourceQuota.MemoryLimit,
				MemoryRequest: original.ResourceQuota.MemoryRequest,
				DiskLimit:     original.ResourceQuota.DiskLimit,
				GPULimit:      original.ResourceQuota.GPULimit,
				GPUType:       original.ResourceQuota.GPUType,
				MaxReplicas:   original.ResourceQuota.MaxReplicas,
				MaxLifetime:   original.ResourceQuota.MaxLifetime,
			}
			if err := tx.Create(clonedQuota).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return cloned, err
}

// ExportTemplate 导出模板
func (s *jupyterLabTemplateServiceImpl) ExportTemplate(id uint) (map[string]interface{}, error) {
	template, err := s.GetTemplate(id)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"template":    template,
		"version":     "1.0",
		"exported_at": time.Now(),
	}

	return export, nil
}

// ImportTemplate 导入模板
func (s *jupyterLabTemplateServiceImpl) ImportTemplate(data map[string]interface{}, userID uint) (*models.JupyterLabTemplate, error) {
	templateData, ok := data["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid template data")
	}

	// 解析模板数据
	templateJSON, err := json.Marshal(templateData)
	if err != nil {
		return nil, err
	}

	var template models.JupyterLabTemplate
	if err := json.Unmarshal(templateJSON, &template); err != nil {
		return nil, err
	}

	// 重置ID和创建信息
	template.ID = 0
	template.CreatedBy = userID
	template.IsDefault = false
	template.CreatedAt = time.Time{}
	template.UpdatedAt = time.Time{}

	// 创建导入的模板
	err = s.CreateTemplate(&template)
	return &template, err
}

// CreatePredefinedTemplates 创建预定义模板
func (s *jupyterLabTemplateServiceImpl) CreatePredefinedTemplates() error {
	predefinedTemplates := []models.JupyterLabTemplate{
		{
			Name:            "Python 机器学习模板",
			Description:     "包含常用机器学习库的Python环境 / Python ML environment with common libraries",
			PythonVersion:   "3.11",
			CondaVersion:    "23.7.0",
			BaseImage:       "jupyter/datascience-notebook:latest",
			Requirements:    `["numpy", "pandas", "scikit-learn", "matplotlib", "seaborn", "plotly", "jupyterlab"]`,
			CondaPackages:   `["pytorch", "tensorflow", "xgboost"]`,
			SystemPackages:  `["git", "vim", "curl"]`,
			EnvironmentVars: `[{"name":"PYTHONPATH","value":"/home/jovyan/work"}]`,
			StartupScript: `#!/bin/bash
# 设置工作目录
mkdir -p /home/jovyan/work/projects
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: true,
			CreatedBy: 0, // 系统创建
		},
		{
			Name:            "深度学习GPU模板",
			Description:     "支持GPU的深度学习环境 / Deep learning environment with GPU support",
			PythonVersion:   "3.11",
			CondaVersion:    "23.7.0",
			BaseImage:       "tensorflow/tensorflow:latest-gpu-jupyter",
			Requirements:    `["numpy", "pandas", "torch", "torchvision", "tensorflow", "keras", "transformers"]`,
			CondaPackages:   `["cudatoolkit", "cudnn"]`,
			SystemPackages:  `["git", "vim", "htop", "nvidia-smi"]`,
			EnvironmentVars: `[{"name":"CUDA_VISIBLE_DEVICES","value":"0"},{"name":"PYTHONPATH","value":"/home/jovyan/work"}]`,
			StartupScript: `#!/bin/bash
# 验证GPU可用性
nvidia-smi
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		{
			Name:            "数据科学模板",
			Description:     "数据分析和可视化环境 / Data analysis and visualization environment",
			PythonVersion:   "3.11",
			CondaVersion:    "23.7.0",
			BaseImage:       "jupyter/datascience-notebook:latest",
			Requirements:    `["pandas", "numpy", "scipy", "matplotlib", "seaborn", "plotly", "dash", "streamlit", "jupyter-dash"]`,
			CondaPackages:   `["r-base", "r-ggplot2", "r-dplyr"]`,
			SystemPackages:  `["git", "vim"]`,
			EnvironmentVars: `[{"name":"PYTHONPATH","value":"/home/jovyan/work"}]`,
			StartupScript: `#!/bin/bash
# 设置R环境
R --version
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// VERL 分布式强化学习模板
		{
			Name:            "VERL 强化学习模板",
			Description:     "Volcano Engine强化学习框架环境，支持PPO/GRPO算法 / VERL RL framework for PPO/GRPO algorithms",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "nvcr.io/nvidia/pytorch:24.01-py3",
			Requirements:    `["verl", "ray[default]", "torch", "transformers", "accelerate", "datasets", "tensorboard", "wandb"]`,
			CondaPackages:   `["cudatoolkit=12.1", "nccl"]`,
			SystemPackages:  `["git", "vim", "htop", "openssh-client"]`,
			EnvironmentVars: `[{"name":"VERL_BACKEND","value":"ray"},{"name":"CUDA_VISIBLE_DEVICES","value":"all"},{"name":"NCCL_DEBUG","value":"INFO"}]`,
			StartupScript: `#!/bin/bash
# 检查GPU
nvidia-smi
# 检查Ray状态
ray status || echo "Ray not started yet"
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// Ray 分布式计算模板
		{
			Name:            "Ray 分布式计算模板",
			Description:     "Ray分布式框架环境，支持分布式训练和推理 / Ray distributed framework for training and inference",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "rayproject/ray-ml:latest-gpu",
			Requirements:    `["ray[default,tune,serve]", "torch", "tensorflow", "numpy", "pandas", "scikit-learn"]`,
			CondaPackages:   `["cudatoolkit=12.1"]`,
			SystemPackages:  `["git", "vim", "htop"]`,
			EnvironmentVars: `[{"name":"RAY_ADDRESS","value":"auto"},{"name":"CUDA_VISIBLE_DEVICES","value":"all"}]`,
			StartupScript: `#!/bin/bash
# 检查Ray集群
ray status || echo "Ray cluster not connected"
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// LLaMA 大语言模型模板
		{
			Name:            "LLaMA 大模型训练模板",
			Description:     "Meta LLaMA大语言模型训练环境 / Meta LLaMA LLM training environment",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "nvcr.io/nvidia/pytorch:24.01-py3",
			Requirements:    `["torch", "transformers>=4.36.0", "accelerate", "bitsandbytes", "peft", "trl", "datasets", "sentencepiece", "protobuf", "flash-attn"]`,
			CondaPackages:   `["cudatoolkit=12.1", "nccl"]`,
			SystemPackages:  `["git", "git-lfs", "vim", "htop"]`,
			EnvironmentVars: `[{"name":"HF_HOME","value":"/home/jovyan/.cache/huggingface"},{"name":"CUDA_VISIBLE_DEVICES","value":"all"},{"name":"PYTORCH_CUDA_ALLOC_CONF","value":"max_split_size_mb:512"}]`,
			StartupScript: `#!/bin/bash
# 检查GPU
nvidia-smi
# 配置git-lfs
git lfs install
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// Qwen 大语言模型模板
		{
			Name:            "Qwen 大模型训练模板",
			Description:     "阿里Qwen系列大语言模型训练环境 / Alibaba Qwen LLM training environment",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "nvcr.io/nvidia/pytorch:24.01-py3",
			Requirements:    `["torch", "transformers>=4.37.0", "accelerate", "peft", "trl", "datasets", "tiktoken", "einops", "transformers_stream_generator", "flash-attn"]`,
			CondaPackages:   `["cudatoolkit=12.1", "nccl"]`,
			SystemPackages:  `["git", "git-lfs", "vim", "htop"]`,
			EnvironmentVars: `[{"name":"HF_HOME","value":"/home/jovyan/.cache/huggingface"},{"name":"CUDA_VISIBLE_DEVICES","value":"all"},{"name":"PYTORCH_CUDA_ALLOC_CONF","value":"max_split_size_mb:512"}]`,
			StartupScript: `#!/bin/bash
# 检查GPU
nvidia-smi
# 配置git-lfs
git lfs install
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// Qwen-VL 多模态模型模板
		{
			Name:            "Qwen-VL 多模态模型模板",
			Description:     "阿里Qwen-VL视觉语言模型训练环境 / Alibaba Qwen-VL Vision-Language model training",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "nvcr.io/nvidia/pytorch:24.01-py3",
			Requirements:    `["torch", "transformers>=4.37.0", "accelerate", "peft", "datasets", "tiktoken", "einops", "pillow", "torchvision", "flash-attn", "timm"]`,
			CondaPackages:   `["cudatoolkit=12.1", "nccl"]`,
			SystemPackages:  `["git", "git-lfs", "vim", "htop", "libgl1-mesa-glx"]`,
			EnvironmentVars: `[{"name":"HF_HOME","value":"/home/jovyan/.cache/huggingface"},{"name":"CUDA_VISIBLE_DEVICES","value":"all"},{"name":"PYTORCH_CUDA_ALLOC_CONF","value":"max_split_size_mb:512"}]`,
			StartupScript: `#!/bin/bash
# 检查GPU
nvidia-smi
# 配置git-lfs
git lfs install
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
		// DeepSpeed 分布式训练模板
		{
			Name:            "DeepSpeed 分布式训练模板",
			Description:     "微软DeepSpeed大模型分布式训练环境 / Microsoft DeepSpeed distributed training",
			PythonVersion:   "3.10",
			CondaVersion:    "23.7.0",
			BaseImage:       "nvcr.io/nvidia/pytorch:24.01-py3",
			Requirements:    `["torch", "deepspeed>=0.12.0", "transformers", "accelerate", "datasets", "mpi4py", "tensorboard"]`,
			CondaPackages:   `["cudatoolkit=12.1", "nccl", "openmpi"]`,
			SystemPackages:  `["git", "vim", "htop", "openssh-client", "pdsh"]`,
			EnvironmentVars: `[{"name":"CUDA_VISIBLE_DEVICES","value":"all"},{"name":"NCCL_DEBUG","value":"INFO"},{"name":"DS_ACCELERATOR","value":"cuda"}]`,
			StartupScript: `#!/bin/bash
# 检查GPU
nvidia-smi
# 检查DeepSpeed
ds_report || echo "DeepSpeed not configured"
# 启动JupyterLab
exec "$@"`,
			IsActive:  true,
			IsDefault: false,
			CreatedBy: 0,
		},
	}

	// 创建对应的资源配额
	resourceQuotas := []models.JupyterLabResourceQuota{
		{
			TemplateID:    1, // 机器学习模板
			CPULimit:      "4",
			CPURequest:    "2",
			MemoryLimit:   "8Gi",
			MemoryRequest: "4Gi",
			DiskLimit:     "20Gi",
			GPULimit:      0,
			MaxReplicas:   1,
			MaxLifetime:   7200, // 2小时
		},
		{
			TemplateID:    2, // 深度学习GPU模板
			CPULimit:      "8",
			CPURequest:    "4",
			MemoryLimit:   "16Gi",
			MemoryRequest: "8Gi",
			DiskLimit:     "50Gi",
			GPULimit:      1,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   14400, // 4小时
		},
		{
			TemplateID:    3, // 数据科学模板
			CPULimit:      "6",
			CPURequest:    "3",
			MemoryLimit:   "12Gi",
			MemoryRequest: "6Gi",
			DiskLimit:     "30Gi",
			GPULimit:      0,
			MaxReplicas:   1,
			MaxLifetime:   10800, // 3小时
		},
		{
			TemplateID:    4, // VERL 强化学习模板
			CPULimit:      "16",
			CPURequest:    "8",
			MemoryLimit:   "64Gi",
			MemoryRequest: "32Gi",
			DiskLimit:     "100Gi",
			GPULimit:      4,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   86400, // 24小时
		},
		{
			TemplateID:    5, // Ray 分布式计算模板
			CPULimit:      "16",
			CPURequest:    "8",
			MemoryLimit:   "64Gi",
			MemoryRequest: "32Gi",
			DiskLimit:     "100Gi",
			GPULimit:      4,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   8,
			MaxLifetime:   86400, // 24小时
		},
		{
			TemplateID:    6, // LLaMA 大模型训练模板
			CPULimit:      "32",
			CPURequest:    "16",
			MemoryLimit:   "128Gi",
			MemoryRequest: "64Gi",
			DiskLimit:     "500Gi",
			GPULimit:      8,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   172800, // 48小时
		},
		{
			TemplateID:    7, // Qwen 大模型训练模板
			CPULimit:      "32",
			CPURequest:    "16",
			MemoryLimit:   "128Gi",
			MemoryRequest: "64Gi",
			DiskLimit:     "500Gi",
			GPULimit:      8,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   172800, // 48小时
		},
		{
			TemplateID:    8, // Qwen-VL 多模态模型模板
			CPULimit:      "32",
			CPURequest:    "16",
			MemoryLimit:   "128Gi",
			MemoryRequest: "64Gi",
			DiskLimit:     "500Gi",
			GPULimit:      8,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   172800, // 48小时
		},
		{
			TemplateID:    9, // DeepSpeed 分布式训练模板
			CPULimit:      "32",
			CPURequest:    "16",
			MemoryLimit:   "128Gi",
			MemoryRequest: "64Gi",
			DiskLimit:     "500Gi",
			GPULimit:      8,
			GPUType:       "nvidia.com/gpu",
			MaxReplicas:   1,
			MaxLifetime:   172800, // 48小时
		},
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否已经创建过预定义模板
		var count int64
		tx.Model(&models.JupyterLabTemplate{}).Where("created_by = 0").Count(&count)
		if count > 0 {
			return nil // 已经创建过了
		}

		// 创建模板
		for _, template := range predefinedTemplates {
			if err := tx.Create(&template).Error; err != nil {
				return err
			}
		}

		// 创建资源配额
		for _, quota := range resourceQuotas {
			if err := tx.Create(&quota).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
