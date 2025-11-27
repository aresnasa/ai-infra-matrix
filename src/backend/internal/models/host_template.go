package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gorm.io/gorm"
)

// HostTemplate 主机配置模板（用于存储用户上传的批量主机配置）
type HostTemplate struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"not null;index"`
	Description string `json:"description"`
	Format      string `json:"format" gorm:"not null"` // csv, json, yaml, ini
	HostCount   int    `json:"host_count"`

	// 加密存储的主机数据
	EncryptedData string `json:"-" gorm:"type:text"`

	// 元数据（不包含敏感信息）
	Groups    string `json:"groups"` // 逗号分隔的组列表
	CreatedBy uint   `json:"created_by" gorm:"index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// HostTemplateHost 主机配置（用于返回给前端，不包含加密数据）
type HostTemplateHost struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"` // 返回时可选择隐藏
	UseSudo  bool   `json:"use_sudo"`
	MinionID string `json:"minion_id,omitempty"`
	Group    string `json:"group,omitempty"`
}

// HostTemplateCreateRequest 创建主机模板请求
type HostTemplateCreateRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	Format      string             `json:"format"`
	Hosts       []HostTemplateHost `json:"hosts" binding:"required"`
}

// HostTemplateResponse 主机模板响应
type HostTemplateResponse struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Format      string             `json:"format"`
	HostCount   int                `json:"host_count"`
	Groups      []string           `json:"groups"`
	CreatedBy   uint               `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Hosts       []HostTemplateHost `json:"hosts,omitempty"` // 可选返回主机列表
}

// getEncryptionKey 获取加密密钥
func getEncryptionKey() []byte {
	key := os.Getenv("HOST_ENCRYPTION_KEY")
	if key == "" {
		key = os.Getenv("JWT_SECRET")
	}
	if key == "" {
		key = "ai-infra-matrix-default-key-32b" // 默认密钥（生产环境应该配置）
	}

	// 确保密钥长度为 32 字节（AES-256）
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		return padded
	}
	return keyBytes[:32]
}

// Encrypt 加密数据
func Encrypt(plaintext []byte) (string, error) {
	key := getEncryptionKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建加密器失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func Decrypt(ciphertext string) ([]byte, error) {
	key := getEncryptionKey()

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("base64 解码失败: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建解密器失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("密文太短")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	return plaintext, nil
}

// SetHosts 设置主机列表（加密存储）
func (t *HostTemplate) SetHosts(hosts []HostTemplateHost) error {
	data, err := json.Marshal(hosts)
	if err != nil {
		return fmt.Errorf("序列化主机数据失败: %v", err)
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		return fmt.Errorf("加密主机数据失败: %v", err)
	}

	t.EncryptedData = encrypted
	t.HostCount = len(hosts)

	// 提取组列表
	groupSet := make(map[string]bool)
	for _, h := range hosts {
		if h.Group != "" {
			groupSet[h.Group] = true
		}
	}
	groups := make([]string, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}
	groupsData, _ := json.Marshal(groups)
	t.Groups = string(groupsData)

	return nil
}

// GetHosts 获取主机列表（解密）
func (t *HostTemplate) GetHosts() ([]HostTemplateHost, error) {
	if t.EncryptedData == "" {
		return []HostTemplateHost{}, nil
	}

	decrypted, err := Decrypt(t.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("解密主机数据失败: %v", err)
	}

	var hosts []HostTemplateHost
	if err := json.Unmarshal(decrypted, &hosts); err != nil {
		return nil, fmt.Errorf("反序列化主机数据失败: %v", err)
	}

	return hosts, nil
}

// GetHostsMasked 获取主机列表（密码脱敏）
func (t *HostTemplate) GetHostsMasked() ([]HostTemplateHost, error) {
	hosts, err := t.GetHosts()
	if err != nil {
		return nil, err
	}

	for i := range hosts {
		if len(hosts[i].Password) > 0 {
			hosts[i].Password = "******"
		}
	}

	return hosts, nil
}

// ToResponse 转换为响应结构
func (t *HostTemplate) ToResponse(includeHosts bool, maskPasswords bool) (*HostTemplateResponse, error) {
	resp := &HostTemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Format:      t.Format,
		HostCount:   t.HostCount,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	// 解析组列表
	if t.Groups != "" {
		var groups []string
		json.Unmarshal([]byte(t.Groups), &groups)
		resp.Groups = groups
	}

	if includeHosts {
		var err error
		if maskPasswords {
			resp.Hosts, err = t.GetHostsMasked()
		} else {
			resp.Hosts, err = t.GetHosts()
		}
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// BeforeCreate GORM 钩子
func (t *HostTemplate) BeforeCreate(tx *gorm.DB) error {
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	return nil
}

// BeforeUpdate GORM 钩子
func (t *HostTemplate) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now()
	return nil
}
