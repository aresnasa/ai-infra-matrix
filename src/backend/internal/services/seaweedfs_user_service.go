package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// SeaweedFSUserService SeaweedFS 用户管理服务接口
type SeaweedFSUserService interface {
	// EnsureUser 确保用户存在（创建或更新）
	EnsureUser(user models.User, permissionLevel string, metadata SeaweedFSUserMetadata) error
	// CreateUserBucket 为用户创建存储桶
	CreateUserBucket(username string, bucketName string, quota int64) error
	// DeleteUser 删除用户（可选删除存储桶）
	DeleteUser(username string) error
	// SetUserQuota 设置用户配额
	SetUserQuota(username string, quotaBytes int64) error
	// GetUserBuckets 获取用户的存储桶列表
	GetUserBuckets(username string) ([]string, error)
	// CreateIAMPolicy 创建 IAM 策略
	CreateIAMPolicy(policyName string, policy SeaweedFSIAMPolicy) error
	// AssignPolicyToUser 为用户分配策略
	AssignPolicyToUser(username string, policyName string) error
}

// seaweedFSUserServiceImpl SeaweedFS 用户服务实现
type seaweedFSUserServiceImpl struct {
	config     *models.ObjectStorageConfig
	httpClient *http.Client
	s3Client   *minio.Client
}

// SeaweedFSIAMPolicy IAM 策略结构
type SeaweedFSIAMPolicy struct {
	Version   string                  `json:"Version"`
	Statement []SeaweedFSIAMStatement `json:"Statement"`
}

// SeaweedFSIAMStatement IAM 策略声明
type SeaweedFSIAMStatement struct {
	Effect    string                 `json:"Effect"`
	Action    []string               `json:"Action"`
	Resource  []string               `json:"Resource"`
	Condition map[string]interface{} `json:"Condition,omitempty"`
}

// SeaweedFSIAMUser IAM 用户
type SeaweedFSIAMUser struct {
	Name      string    `json:"name"`
	AccessKey string    `json:"accessKey,omitempty"`
	SecretKey string    `json:"secretKey,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// NewSeaweedFSUserService 创建 SeaweedFS 用户服务
func NewSeaweedFSUserService(config *models.ObjectStorageConfig) (SeaweedFSUserService, error) {
	if config == nil {
		return nil, fmt.Errorf("SeaweedFS config is nil")
	}

	service := &seaweedFSUserServiceImpl{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 初始化 S3 客户端
	if config.Endpoint != "" && config.AccessKey != "" && config.SecretKey != "" {
		client, err := minio.New(config.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
			Secure: config.SSLEnabled,
			Region: config.Region,
		})
		if err != nil {
			logrus.WithError(err).Warn("[SeaweedFS] Failed to initialize S3 client")
		} else {
			service.s3Client = client
		}
	}

	return service, nil
}

// EnsureUser 确保用户存在
func (s *seaweedFSUserServiceImpl) EnsureUser(user models.User, permissionLevel string, metadata SeaweedFSUserMetadata) error {
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"level":    permissionLevel,
		"service":  "seaweedfs_user",
	})

	// 1. 创建 IAM 用户
	if err := s.createIAMUser(user.Username); err != nil {
		logger.WithError(err).Warn("Failed to create IAM user (may already exist)")
	}

	// 2. 确定存储桶名称
	bucketName := metadata.BucketName
	if bucketName == "" {
		bucketName = fmt.Sprintf("user-%s", user.Username)
	}

	// 3. 创建用户专属存储桶
	quota := metadata.QuotaBytes
	if quota == 0 {
		quota = s.getDefaultQuota(permissionLevel)
	}
	if err := s.CreateUserBucket(user.Username, bucketName, quota); err != nil {
		logger.WithError(err).Warn("Failed to create user bucket")
		// 继续执行，存储桶可能已存在
	}

	// 4. 创建并分配 IAM 策略
	policyName := fmt.Sprintf("policy-%s", user.Username)
	policy := s.generateUserPolicy(user.Username, bucketName, permissionLevel)
	if err := s.CreateIAMPolicy(policyName, policy); err != nil {
		logger.WithError(err).Warn("Failed to create IAM policy")
	}

	if err := s.AssignPolicyToUser(user.Username, policyName); err != nil {
		logger.WithError(err).Warn("Failed to assign policy to user")
	}

	logger.Info("User ensured in SeaweedFS")
	return nil
}

// CreateUserBucket 为用户创建存储桶
func (s *seaweedFSUserServiceImpl) CreateUserBucket(username string, bucketName string, quota int64) error {
	if s.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	ctx := context.Background()

	// 检查存储桶是否存在
	exists, err := s.s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}

	if !exists {
		// 创建存储桶
		err = s.s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: s.config.Region,
		})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		logrus.WithField("bucket", bucketName).Info("Created bucket for user")
	}

	// 设置存储桶配额（如果 SeaweedFS 支持）
	if quota > 0 {
		if err := s.setBucketQuota(bucketName, quota); err != nil {
			logrus.WithError(err).Warn("Failed to set bucket quota")
		}
	}

	return nil
}

// DeleteUser 删除用户
func (s *seaweedFSUserServiceImpl) DeleteUser(username string) error {
	logger := logrus.WithField("username", username)

	// 删除 IAM 用户
	if err := s.deleteIAMUser(username); err != nil {
		logger.WithError(err).Warn("Failed to delete IAM user")
	}

	// 注意：默认不删除用户的存储桶，以保留数据
	// 如需删除，可以通过管理接口手动处理

	logger.Info("User deleted from SeaweedFS")
	return nil
}

// SetUserQuota 设置用户配额
func (s *seaweedFSUserServiceImpl) SetUserQuota(username string, quotaBytes int64) error {
	bucketName := fmt.Sprintf("user-%s", username)
	return s.setBucketQuota(bucketName, quotaBytes)
}

// GetUserBuckets 获取用户的存储桶列表
func (s *seaweedFSUserServiceImpl) GetUserBuckets(username string) ([]string, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	ctx := context.Background()
	buckets, err := s.s3Client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var userBuckets []string
	prefix := fmt.Sprintf("user-%s", username)
	for _, bucket := range buckets {
		if bucket.Name == prefix || len(bucket.Name) >= len(prefix) && bucket.Name[:len(prefix)] == prefix {
			userBuckets = append(userBuckets, bucket.Name)
		}
	}

	return userBuckets, nil
}

// CreateIAMPolicy 创建 IAM 策略
func (s *seaweedFSUserServiceImpl) CreateIAMPolicy(policyName string, policy SeaweedFSIAMPolicy) error {
	filerURL := s.config.FilerURL
	if filerURL == "" {
		return fmt.Errorf("filer URL not configured")
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/iam/policies/%s", filerURL, policyName)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(policyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// 添加认证
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("create policy failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AssignPolicyToUser 为用户分配策略
func (s *seaweedFSUserServiceImpl) AssignPolicyToUser(username string, policyName string) error {
	filerURL := s.config.FilerURL
	if filerURL == "" {
		return fmt.Errorf("filer URL not configured")
	}

	url := fmt.Sprintf("%s/admin/iam/users/%s/policies/%s", filerURL, username, policyName)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("assign policy failed with status: %d", resp.StatusCode)
	}

	return nil
}

// createIAMUser 创建 IAM 用户
func (s *seaweedFSUserServiceImpl) createIAMUser(username string) error {
	filerURL := s.config.FilerURL
	if filerURL == "" {
		return fmt.Errorf("filer URL not configured")
	}

	url := fmt.Sprintf("%s/admin/iam/users/%s", filerURL, username)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 409 {
		return fmt.Errorf("create user failed with status: %d", resp.StatusCode)
	}

	return nil
}

// deleteIAMUser 删除 IAM 用户
func (s *seaweedFSUserServiceImpl) deleteIAMUser(username string) error {
	filerURL := s.config.FilerURL
	if filerURL == "" {
		return fmt.Errorf("filer URL not configured")
	}

	url := fmt.Sprintf("%s/admin/iam/users/%s", filerURL, username)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("delete user failed with status: %d", resp.StatusCode)
	}

	return nil
}

// setBucketQuota 设置存储桶配额
func (s *seaweedFSUserServiceImpl) setBucketQuota(bucketName string, quotaBytes int64) error {
	filerURL := s.config.FilerURL
	if filerURL == "" {
		return fmt.Errorf("filer URL not configured")
	}

	// SeaweedFS 的配额设置 API
	url := fmt.Sprintf("%s/admin/buckets/%s/quota", filerURL, bucketName)
	body := map[string]int64{"quota": quotaBytes}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("set quota failed with status: %d", resp.StatusCode)
	}

	return nil
}

// generateUserPolicy 生成用户策略
func (s *seaweedFSUserServiceImpl) generateUserPolicy(username, bucketName, permissionLevel string) SeaweedFSIAMPolicy {
	var actions []string

	switch permissionLevel {
	case models.PermissionLevelAdmin:
		actions = []string{
			"s3:*",
		}
	case models.PermissionLevelUser:
		actions = []string{
			"s3:GetObject",
			"s3:PutObject",
			"s3:DeleteObject",
			"s3:ListBucket",
			"s3:GetBucketLocation",
		}
	case models.PermissionLevelReadonly:
		actions = []string{
			"s3:GetObject",
			"s3:ListBucket",
			"s3:GetBucketLocation",
		}
	default:
		actions = []string{
			"s3:ListBucket",
		}
	}

	return SeaweedFSIAMPolicy{
		Version: "2012-10-17",
		Statement: []SeaweedFSIAMStatement{
			{
				Effect: "Allow",
				Action: actions,
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s", bucketName),
					fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
			},
		},
	}
}

// getDefaultQuota 获取默认配额
func (s *seaweedFSUserServiceImpl) getDefaultQuota(permissionLevel string) int64 {
	switch permissionLevel {
	case models.PermissionLevelAdmin:
		return 100 * 1024 * 1024 * 1024 // 100GB
	case models.PermissionLevelUser:
		return 10 * 1024 * 1024 * 1024 // 10GB
	case models.PermissionLevelReadonly:
		return 1 * 1024 * 1024 * 1024 // 1GB
	default:
		return 1 * 1024 * 1024 * 1024 // 1GB
	}
}

// addAuth 添加认证头
func (s *seaweedFSUserServiceImpl) addAuth(req *http.Request) {
	if s.config.AccessKey != "" && s.config.SecretKey != "" {
		req.SetBasicAuth(s.config.AccessKey, s.config.SecretKey)
	}
	// 如果配置了 JWT，也可以添加
	if s.config.JWTSecret != "" {
		// 使用 SeaweedFSService 生成 JWT
		// 这里简化处理
	}
}
