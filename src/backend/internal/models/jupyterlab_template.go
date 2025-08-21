package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// JupyterLabTemplate JupyterLab模板
type JupyterLabTemplate struct {
	ID               uint                   `json:"id" gorm:"primaryKey"`
	Name             string                 `json:"name" gorm:"not null"`
	Description      string                 `json:"description"`
	PythonVersion    string                 `json:"python_version" gorm:"default:'3.11'"`
	CondaVersion     string                 `json:"conda_version" gorm:"default:'23.7.0'"`
	BaseImage        string                 `json:"base_image" gorm:"default:'jupyter/scipy-notebook:latest'"`
	Requirements     string                 `json:"requirements" gorm:"type:text"` // pip requirements
	CondaPackages    string                 `json:"conda_packages" gorm:"type:text"` // conda packages
	SystemPackages   string                 `json:"system_packages" gorm:"type:text"` // apt packages
	EnvironmentVars  string                 `json:"environment_vars" gorm:"type:text"` // JSON格式的环境变量
	StartupScript    string                 `json:"startup_script" gorm:"type:text"` // 启动脚本
	IsActive         bool                   `json:"is_active" gorm:"default:true"`
	IsDefault        bool                   `json:"is_default" gorm:"default:false"`
	CreatedBy        uint                   `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`
	
	// 关联资源配额
	ResourceQuota    *JupyterLabResourceQuota `json:"resource_quota,omitempty" gorm:"foreignKey:TemplateID"`
}

// JupyterLabResourceQuota JupyterLab资源配额
type JupyterLabResourceQuota struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	TemplateID   uint    `json:"template_id" gorm:"not null"`
	CPULimit     string  `json:"cpu_limit" gorm:"default:'2'"` // CPU核心数限制
	CPURequest   string  `json:"cpu_request" gorm:"default:'1'"` // CPU请求
	MemoryLimit  string  `json:"memory_limit" gorm:"default:'4Gi'"` // 内存限制
	MemoryRequest string `json:"memory_request" gorm:"default:'2Gi'"` // 内存请求
	DiskLimit    string  `json:"disk_limit" gorm:"default:'10Gi'"` // 磁盘限制
	GPULimit     int     `json:"gpu_limit" gorm:"default:0"` // GPU数量限制
	GPUType      string  `json:"gpu_type"` // GPU类型限制 (nvidia.com/gpu, amd.com/gpu)
	MaxReplicas  int     `json:"max_replicas" gorm:"default:1"` // 最大副本数
	MaxLifetime  int     `json:"max_lifetime" gorm:"default:86400"` // 最大生存时间(秒)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JupyterLabInstance JupyterLab实例
type JupyterLabInstance struct {
	ID            uint                    `json:"id" gorm:"primaryKey"`
	UserID        uint                    `json:"user_id" gorm:"not null"`
	TemplateID    uint                    `json:"template_id" gorm:"not null"`
	Name          string                  `json:"name" gorm:"not null"`
	Status        string                  `json:"status" gorm:"default:'pending'"` // pending, running, stopped, failed
	URL           string                  `json:"url"`
	PodName       string                  `json:"pod_name"`
	Namespace     string                  `json:"namespace" gorm:"default:'jupyterhub'"`
	NodeName      string                  `json:"node_name"`
	StartTime     *time.Time              `json:"start_time"`
	StopTime      *time.Time              `json:"stop_time"`
	LastAccess    *time.Time              `json:"last_access"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	DeletedAt     gorm.DeletedAt          `json:"-" gorm:"index"`
	
	// 关联
	Template      *JupyterLabTemplate     `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	User          *User                   `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// EnvironmentVariable 环境变量结构
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetEnvironmentVars 获取环境变量
func (t *JupyterLabTemplate) GetEnvironmentVars() ([]EnvironmentVariable, error) {
	if t.EnvironmentVars == "" {
		return []EnvironmentVariable{}, nil
	}
	
	var envVars []EnvironmentVariable
	err := json.Unmarshal([]byte(t.EnvironmentVars), &envVars)
	return envVars, err
}

// SetEnvironmentVars 设置环境变量
func (t *JupyterLabTemplate) SetEnvironmentVars(envVars []EnvironmentVariable) error {
	data, err := json.Marshal(envVars)
	if err != nil {
		return err
	}
	t.EnvironmentVars = string(data)
	return nil
}

// GetRequirementsList 获取pip依赖列表
func (t *JupyterLabTemplate) GetRequirementsList() []string {
	if t.Requirements == "" {
		return []string{}
	}
	
	var requirements []string
	json.Unmarshal([]byte(t.Requirements), &requirements)
	return requirements
}

// GetCondaPackagesList 获取conda包列表
func (t *JupyterLabTemplate) GetCondaPackagesList() []string {
	if t.CondaPackages == "" {
		return []string{}
	}
	
	var packages []string
	json.Unmarshal([]byte(t.CondaPackages), &packages)
	return packages
}

// GetSystemPackagesList 获取系统包列表
func (t *JupyterLabTemplate) GetSystemPackagesList() []string {
	if t.SystemPackages == "" {
		return []string{}
	}
	
	var packages []string
	json.Unmarshal([]byte(t.SystemPackages), &packages)
	return packages
}

// TableName 指定表名
func (JupyterLabTemplate) TableName() string {
	return "jupyterlab_templates"
}

func (JupyterLabResourceQuota) TableName() string {
	return "jupyterlab_resource_quotas"
}

func (JupyterLabInstance) TableName() string {
	return "jupyterlab_instances"
}
