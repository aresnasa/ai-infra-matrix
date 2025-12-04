package models

import (
	"time"

	"gorm.io/gorm"
)

// 对象存储类型常量
const (
	StorageTypeSeaweedFS  = "seaweedfs"
	StorageTypeMinIO      = "minio"
	StorageTypeAWSS3      = "aws_s3"
	StorageTypeAliyunOSS  = "aliyun_oss"
	StorageTypeTencentCOS = "tencent_cos"
)

// ObjectStorageConfig 对象存储配置模型
type ObjectStorageConfig struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:100"`        // 配置名称
	Type        string         `json:"type" gorm:"not null;size:50"`         // 存储类型：seaweedfs, minio, aws_s3, aliyun_oss, tencent_cos
	Endpoint    string         `json:"endpoint" gorm:"not null;size:255"`    // 服务端点 (S3 API 端点)
	AccessKey   string         `json:"access_key" gorm:"not null;size:255"`  // 访问密钥ID
	SecretKey   string         `json:"secret_key" gorm:"not null;size:500"`  // 访问密钥Secret
	Region      string         `json:"region" gorm:"size:100"`               // 区域
	WebURL      string         `json:"web_url" gorm:"size:255"`              // Web控制台地址（SeaweedFS Filer UI / MinIO Console）
	FilerURL    string         `json:"filer_url" gorm:"size:255"`            // SeaweedFS Filer 地址
	MasterURL   string         `json:"master_url" gorm:"size:255"`           // SeaweedFS Master 地址
	JWTSecret   string         `json:"jwt_secret" gorm:"size:255"`           // SeaweedFS JWT Secret (用于认证)
	SSLEnabled  bool           `json:"ssl_enabled" gorm:"default:false"`     // 是否启用SSL
	Timeout     int            `json:"timeout" gorm:"default:30"`            // 超时时间（秒）
	IsActive    bool           `json:"is_active" gorm:"default:false"`       // 是否为激活配置
	Status      string         `json:"status" gorm:"default:'disconnected'"` // 连接状态：connected, disconnected, error
	Description string         `json:"description" gorm:"type:text"`         // 描述信息
	LastTested  *time.Time     `json:"last_tested"`                          // 上次测试时间
	CreatedBy   uint           `json:"created_by"`                           // 创建用户ID
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// 关联
	Creator *User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// TableName 设置表名
func (ObjectStorageConfig) TableName() string {
	return "object_storage_configs"
}

// BeforeSave GORM钩子 - 保存前处理
func (o *ObjectStorageConfig) BeforeSave(tx *gorm.DB) error {
	// 如果设置为激活配置，需要将其他配置设为非激活
	if o.IsActive {
		// 将其他配置设为非激活
		tx.Model(&ObjectStorageConfig{}).Where("id != ? AND is_active = ?", o.ID, true).Update("is_active", false)
	}
	return nil
}

// ObjectStorageStatistics 对象存储统计信息
type ObjectStorageStatistics struct {
	ConfigID     uint   `json:"config_id"`
	BucketCount  int64  `json:"bucket_count"`  // 存储桶数量
	ObjectCount  int64  `json:"object_count"`  // 对象数量
	UsedSpace    string `json:"used_space"`    // 已用空间
	TotalSpace   string `json:"total_space"`   // 总空间
	UsagePercent int    `json:"usage_percent"` // 使用百分比
}

// BucketInfo 存储桶信息
type BucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	ObjectCount  int64     `json:"object_count"`
	Size         int64     `json:"size"`
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
	ContentType  string    `json:"content_type"`
}

// 对象存储操作日志
type ObjectStorageLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ConfigID  uint      `json:"config_id" gorm:"not null"`                // 配置ID
	Operation string    `json:"operation" gorm:"not null;size:100"`       // 操作类型
	Bucket    string    `json:"bucket" gorm:"size:100"`                   // 存储桶名称
	ObjectKey string    `json:"object_key" gorm:"size:500"`               // 对象键
	Status    string    `json:"status" gorm:"not null;default:'pending'"` // 状态：success, failed, pending
	Message   string    `json:"message" gorm:"type:text"`                 // 消息
	UserID    uint      `json:"user_id"`                                  // 操作用户ID
	Duration  int       `json:"duration"`                                 // 操作耗时（毫秒）
	CreatedAt time.Time `json:"created_at"`

	// 关联
	Config *ObjectStorageConfig `json:"config,omitempty" gorm:"foreignKey:ConfigID"`
	User   *User                `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 设置表名
func (ObjectStorageLog) TableName() string {
	return "object_storage_logs"
}
