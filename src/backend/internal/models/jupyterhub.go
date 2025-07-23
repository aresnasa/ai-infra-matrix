package models

import (
	"time"
)

// JupyterHubConfig JupyterHub配置
type JupyterHubConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"unique;not null"`
	URL          string    `json:"url" gorm:"not null"`
	Token        string    `json:"token" gorm:"not null"`
	GPUNodes     string    `json:"gpu_nodes" gorm:"type:text"` // JSON格式存储GPU节点信息
	IsEnabled    bool      `json:"is_enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JupyterTask JupyterHub任务
type JupyterTask struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	TaskName      string     `json:"task_name" gorm:"not null"`
	UserID        uint       `json:"user_id" gorm:"not null"`
	HubConfigID   uint       `json:"hub_config_id" gorm:"not null"`
	NotebookPath  string     `json:"notebook_path"`
	PythonCode    string     `json:"python_code" gorm:"type:text"`
	GPURequested  int        `json:"gpu_requested" gorm:"default:0"`
	MemoryGB      int        `json:"memory_gb" gorm:"default:4"`
	CPUCores      int        `json:"cpu_cores" gorm:"default:2"`
	Status        string     `json:"status" gorm:"default:'pending'"` // pending, running, completed, failed
	JobID         string     `json:"job_id"`
	RemoteJobID   string     `json:"remote_job_id"`
	AnsibleTaskID string     `json:"ansible_task_id"`
	ResultPath    string     `json:"result_path"`
	LogPath       string     `json:"log_path"`
	ErrorMessage  string     `json:"error_message" gorm:"type:text"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	
	// 关联
	User      User             `json:"user" gorm:"foreignKey:UserID"`
	HubConfig JupyterHubConfig `json:"hub_config" gorm:"foreignKey:HubConfigID"`
}

// GPUNode GPU节点信息
type GPUNode struct {
	NodeName     string `json:"node_name"`
	IPAddress    string `json:"ip_address"`
	GPUCount     int    `json:"gpu_count"`
	GPUModel     string `json:"gpu_model"`
	TotalMemory  int    `json:"total_memory_gb"`
	AvailableGPU int    `json:"available_gpu"`
	IsOnline     bool   `json:"is_online"`
}

// JupyterHubUser JupyterHub API响应中的用户信息
type JupyterHubUser struct {
	Name         string                 `json:"name"`
	Admin        bool                   `json:"admin"`
	Groups       []string               `json:"groups"`
	ServerName   string                 `json:"server"`
	Pending      *string                `json:"pending"`
	Created      string                 `json:"created"`
	LastActivity string                 `json:"last_activity"`
	Servers      map[string]interface{} `json:"servers"`
}
