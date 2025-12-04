package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// SeaweedFSService SeaweedFS 对象存储服务
type SeaweedFSService struct {
	config     *models.ObjectStorageConfig
	httpClient *http.Client
	s3Client   *minio.Client
}

// SeaweedFSMasterStatus Master 状态响应
type SeaweedFSMasterStatus struct {
	IsLeader bool     `json:"IsLeader"`
	Leader   string   `json:"Leader"`
	Peers    []string `json:"Peers"`
}

// SeaweedFSClusterStatus 集群状态
type SeaweedFSClusterStatus struct {
	IsLeader  bool     `json:"isLeader"`
	Leader    string   `json:"leader"`
	Peers     []string `json:"peers"`
	Free      int64    `json:"free"`
	Max       int64    `json:"max"`
	VolumeMax int      `json:"volumeMax"`
}

// SeaweedFSVolumeStatus Volume 服务器状态
type SeaweedFSVolumeStatus struct {
	Version string `json:"Version"`
	Volumes []struct {
		Id               int    `json:"Id"`
		Size             int64  `json:"Size"`
		ReplicaPlacement string `json:"ReplicaPlacement"`
		Collection       string `json:"Collection"`
		FileCount        int    `json:"FileCount"`
		DeleteCount      int    `json:"DeleteCount"`
		DeletedByteCount int64  `json:"DeletedByteCount"`
		ReadOnly         bool   `json:"ReadOnly"`
	} `json:"Volumes"`
}

// SeaweedFSFilerStatus Filer 状态
type SeaweedFSFilerStatus struct {
	Version    string `json:"Version"`
	DiskUsages []struct {
		Dir       string `json:"dir"`
		Total     int64  `json:"total"`
		Used      int64  `json:"used"`
		Available int64  `json:"available"`
	} `json:"DiskUsages,omitempty"`
}

// SeaweedFSBucketInfo 存储桶信息
type SeaweedFSBucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
}

// NewSeaweedFSService 创建 SeaweedFS 服务实例
func NewSeaweedFSService(config *models.ObjectStorageConfig) (*SeaweedFSService, error) {
	service := &SeaweedFSService{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}

	// 如果配置了 S3 API 端点，初始化 S3 兼容客户端
	if config.Endpoint != "" && config.AccessKey != "" && config.SecretKey != "" {
		endpoint := strings.TrimPrefix(strings.TrimPrefix(config.Endpoint, "http://"), "https://")
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
			Secure: config.SSLEnabled,
			Region: config.Region,
		})
		if err != nil {
			logrus.Warnf("[SeaweedFS] 初始化 S3 客户端失败: %v", err)
		} else {
			service.s3Client = client
		}
	}

	return service, nil
}

// GenerateJWT 生成 JWT Token
func (s *SeaweedFSService) GenerateJWT(expiresIn time.Duration) (string, error) {
	if s.config.JWTSecret == "" {
		return "", fmt.Errorf("JWT Secret 未配置")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(expiresIn).Unix(),
		"sub": "ai-infra-matrix",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateJWT 验证 JWT Token
func (s *SeaweedFSService) ValidateJWT(tokenString string) (bool, error) {
	if s.config.JWTSecret == "" {
		return false, fmt.Errorf("JWT Secret 未配置")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return false, err
	}

	return token.Valid, nil
}

// doRequest 执行 HTTP 请求 (带 JWT 认证)
func (s *SeaweedFSService) doRequest(method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}

	// 如果配置了 JWT Secret，添加认证头
	if s.config.JWTSecret != "" {
		token, err := s.GenerateJWT(5 * time.Minute)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	req.Header.Set("Content-Type", "application/json")
	return s.httpClient.Do(req)
}

// TestConnection 测试连接
func (s *SeaweedFSService) TestConnection() error {
	// 优先测试 Master 连接
	if s.config.MasterURL != "" {
		if err := s.testMasterConnection(); err != nil {
			return fmt.Errorf("Master 连接失败: %v", err)
		}
	}

	// 测试 Filer 连接
	if s.config.FilerURL != "" {
		if err := s.testFilerConnection(); err != nil {
			return fmt.Errorf("Filer 连接失败: %v", err)
		}
	}

	// 测试 S3 API 连接
	if s.s3Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
		defer cancel()

		_, err := s.s3Client.ListBuckets(ctx)
		if err != nil {
			return fmt.Errorf("S3 API 连接失败: %v", err)
		}
	}

	return nil
}

// testMasterConnection 测试 Master 连接
func (s *SeaweedFSService) testMasterConnection() error {
	url := strings.TrimSuffix(s.config.MasterURL, "/") + "/cluster/status"
	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Master 返回状态码: %d", resp.StatusCode)
	}

	return nil
}

// testFilerConnection 测试 Filer 连接
func (s *SeaweedFSService) testFilerConnection() error {
	url := strings.TrimSuffix(s.config.FilerURL, "/") + "/?pretty=y"
	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Filer 返回状态码: %d", resp.StatusCode)
	}

	return nil
}

// GetClusterStatus 获取集群状态
func (s *SeaweedFSService) GetClusterStatus() (*SeaweedFSClusterStatus, error) {
	if s.config.MasterURL == "" {
		return nil, fmt.Errorf("Master URL 未配置")
	}

	url := strings.TrimSuffix(s.config.MasterURL, "/") + "/cluster/status"
	resp, err := s.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status SeaweedFSClusterStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("解析集群状态失败: %v", err)
	}

	return &status, nil
}

// GetStatistics 获取存储统计信息
func (s *SeaweedFSService) GetStatistics() (*models.ObjectStorageStatistics, error) {
	stats := &models.ObjectStorageStatistics{
		ConfigID: s.config.ID,
	}

	// 通过 S3 API 获取存储桶数量和对象数量
	if s.s3Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
		defer cancel()

		buckets, err := s.s3Client.ListBuckets(ctx)
		if err == nil {
			stats.BucketCount = int64(len(buckets))

			// 计算对象总数
			var totalObjects int64
			for _, bucket := range buckets {
				objectCh := s.s3Client.ListObjects(ctx, bucket.Name, minio.ListObjectsOptions{Recursive: true})
				for range objectCh {
					totalObjects++
				}
			}
			stats.ObjectCount = totalObjects
		}
	}

	// 通过 Master 获取集群容量信息
	if s.config.MasterURL != "" {
		clusterStatus, err := s.GetClusterStatus()
		if err == nil {
			stats.TotalSpace = formatBytes(clusterStatus.Max)
			stats.UsedSpace = formatBytes(clusterStatus.Max - clusterStatus.Free)
			if clusterStatus.Max > 0 {
				stats.UsagePercent = int(float64(clusterStatus.Max-clusterStatus.Free) / float64(clusterStatus.Max) * 100)
			}
		}
	}

	return stats, nil
}

// ListBuckets 列出所有存储桶
func (s *SeaweedFSService) ListBuckets() ([]models.BucketInfo, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	buckets, err := s.s3Client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("列出存储桶失败: %v", err)
	}

	var result []models.BucketInfo
	for _, bucket := range buckets {
		info := models.BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreationDate,
		}

		// 获取桶内对象统计
		objectCh := s.s3Client.ListObjects(ctx, bucket.Name, minio.ListObjectsOptions{Recursive: true})
		var objectCount int64
		var totalSize int64
		for obj := range objectCh {
			if obj.Err == nil {
				objectCount++
				totalSize += obj.Size
			}
		}
		info.ObjectCount = objectCount
		info.Size = totalSize

		result = append(result, info)
	}

	return result, nil
}

// CreateBucket 创建存储桶
func (s *SeaweedFSService) CreateBucket(bucketName string) error {
	if s.s3Client == nil {
		return fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	err := s.s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: s.config.Region,
	})
	if err != nil {
		// 检查桶是否已存在
		exists, errBucketExists := s.s3Client.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			return nil // 桶已存在，不是错误
		}
		return fmt.Errorf("创建存储桶失败: %v", err)
	}

	return nil
}

// DeleteBucket 删除存储桶
func (s *SeaweedFSService) DeleteBucket(bucketName string) error {
	if s.s3Client == nil {
		return fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	err := s.s3Client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("删除存储桶失败: %v", err)
	}

	return nil
}

// ListObjects 列出对象
func (s *SeaweedFSService) ListObjects(bucketName, prefix string, maxKeys int) ([]models.ObjectInfo, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	var result []models.ObjectInfo
	count := 0
	objectCh := s.s3Client.ListObjects(ctx, bucketName, opts)
	for obj := range objectCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("列出对象失败: %v", obj.Err)
		}

		result = append(result, models.ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
		})

		count++
		if maxKeys > 0 && count >= maxKeys {
			break
		}
	}

	return result, nil
}

// UploadObject 上传对象
func (s *SeaweedFSService) UploadObject(bucketName, objectKey string, reader io.Reader, size int64, contentType string) error {
	if s.s3Client == nil {
		return fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second*10) // 上传超时延长
	defer cancel()

	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}

	_, err := s.s3Client.PutObject(ctx, bucketName, objectKey, reader, size, opts)
	if err != nil {
		return fmt.Errorf("上传对象失败: %v", err)
	}

	return nil
}

// DownloadObject 下载对象
func (s *SeaweedFSService) DownloadObject(bucketName, objectKey string) (io.ReadCloser, error) {
	if s.s3Client == nil {
		return nil, fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second*10)
	defer cancel()

	obj, err := s.s3Client.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载对象失败: %v", err)
	}

	return obj, nil
}

// DeleteObject 删除对象
func (s *SeaweedFSService) DeleteObject(bucketName, objectKey string) error {
	if s.s3Client == nil {
		return fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	err := s.s3Client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("删除对象失败: %v", err)
	}

	return nil
}

// GetPresignedURL 获取预签名 URL
func (s *SeaweedFSService) GetPresignedURL(bucketName, objectKey string, expires time.Duration) (string, error) {
	if s.s3Client == nil {
		return "", fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	presignedURL, err := s.s3Client.PresignedGetObject(ctx, bucketName, objectKey, expires, nil)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %v", err)
	}

	return presignedURL.String(), nil
}

// GetPresignedPutURL 获取预签名上传 URL
func (s *SeaweedFSService) GetPresignedPutURL(bucketName, objectKey string, expires time.Duration) (string, error) {
	if s.s3Client == nil {
		return "", fmt.Errorf("S3 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	presignedURL, err := s.s3Client.PresignedPutObject(ctx, bucketName, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("生成预签名上传URL失败: %v", err)
	}

	return presignedURL.String(), nil
}

// UploadToFiler 直接通过 Filer 上传文件
func (s *SeaweedFSService) UploadToFiler(path string, reader io.Reader, contentType string) error {
	if s.config.FilerURL == "" {
		return fmt.Errorf("Filer URL 未配置")
	}

	filerURL := strings.TrimSuffix(s.config.FilerURL, "/") + "/" + strings.TrimPrefix(path, "/")

	req, err := http.NewRequest("POST", filerURL, reader)
	if err != nil {
		return err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// JWT 认证
	if s.config.JWTSecret != "" {
		token, err := s.GenerateJWT(5 * time.Minute)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Filer 上传失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Filer 上传失败 [%d]: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadFromFiler 通过 Filer 下载文件
func (s *SeaweedFSService) DownloadFromFiler(path string) (io.ReadCloser, error) {
	if s.config.FilerURL == "" {
		return nil, fmt.Errorf("Filer URL 未配置")
	}

	filerURL := strings.TrimSuffix(s.config.FilerURL, "/") + "/" + strings.TrimPrefix(path, "/")

	req, err := http.NewRequest("GET", filerURL, nil)
	if err != nil {
		return nil, err
	}

	// JWT 认证
	if s.config.JWTSecret != "" {
		token, err := s.GenerateJWT(5 * time.Minute)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Filer 下载失败: %v", err)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("Filer 下载失败 [%d]", resp.StatusCode)
	}

	return resp.Body, nil
}

// GenerateS3Signature 生成 S3 签名 (用于前端直传)
func (s *SeaweedFSService) GenerateS3Signature(stringToSign string) string {
	mac := hmac.New(sha256.New, []byte(s.config.SecretKey))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// GetWebConsoleURL 获取 Web 控制台 URL
func (s *SeaweedFSService) GetWebConsoleURL() string {
	if s.config.WebURL != "" {
		return s.config.WebURL
	}
	// 默认使用 Filer UI
	if s.config.FilerURL != "" {
		return s.config.FilerURL
	}
	return ""
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ProxyFilerUI 代理 Filer UI 请求
func (s *SeaweedFSService) ProxyFilerUI(targetPath string) (*http.Response, error) {
	if s.config.FilerURL == "" {
		return nil, fmt.Errorf("Filer URL 未配置")
	}

	filerURL := strings.TrimSuffix(s.config.FilerURL, "/") + "/" + strings.TrimPrefix(targetPath, "/")

	req, err := http.NewRequest("GET", filerURL, nil)
	if err != nil {
		return nil, err
	}

	// JWT 认证
	if s.config.JWTSecret != "" {
		token, err := s.GenerateJWT(5 * time.Minute)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	return s.httpClient.Do(req)
}

// GetVolumeServers 获取 Volume 服务器列表
func (s *SeaweedFSService) GetVolumeServers() ([]map[string]interface{}, error) {
	if s.config.MasterURL == "" {
		return nil, fmt.Errorf("Master URL 未配置")
	}

	apiURL := strings.TrimSuffix(s.config.MasterURL, "/") + "/dir/status"
	resp, err := s.doRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 Volume 服务器列表失败: %v", err)
	}

	if topology, ok := result["Topology"].(map[string]interface{}); ok {
		if dataCenters, ok := topology["DataCenters"].([]interface{}); ok {
			var servers []map[string]interface{}
			for _, dc := range dataCenters {
				if dcMap, ok := dc.(map[string]interface{}); ok {
					if racks, ok := dcMap["Racks"].([]interface{}); ok {
						for _, rack := range racks {
							if rackMap, ok := rack.(map[string]interface{}); ok {
								if dataNodes, ok := rackMap["DataNodes"].([]interface{}); ok {
									for _, node := range dataNodes {
										if nodeMap, ok := node.(map[string]interface{}); ok {
											servers = append(servers, nodeMap)
										}
									}
								}
							}
						}
					}
				}
			}
			return servers, nil
		}
	}

	return nil, nil
}

// CreateS3Credentials 创建 S3 访问凭证 (通过 Filer S3 配置)
func (s *SeaweedFSService) CreateS3Credentials(accessKey, secretKey string) error {
	if s.config.FilerURL == "" {
		return fmt.Errorf("Filer URL 未配置")
	}

	// SeaweedFS 通过配置文件管理 S3 凭证
	// 这里只是一个示例，实际需要通过配置文件或 API 管理
	logrus.Infof("[SeaweedFS] 创建 S3 凭证: accessKey=%s", accessKey)

	// TODO: 实现通过 API 或配置文件管理凭证
	return nil
}

// S3Config SeaweedFS S3 配置
type S3Config struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

// GetS3Config 获取 S3 配置信息
func (s *SeaweedFSService) GetS3Config(bucket string) *S3Config {
	return &S3Config{
		Endpoint:        s.config.Endpoint,
		AccessKeyID:     s.config.AccessKey,
		SecretAccessKey: s.config.SecretKey,
		Region:          s.config.Region,
		Bucket:          bucket,
	}
}

// HealthCheck 健康检查
func (s *SeaweedFSService) HealthCheck() error {
	// 检查 Master
	if s.config.MasterURL != "" {
		_, err := s.GetClusterStatus()
		if err != nil {
			return fmt.Errorf("Master 健康检查失败: %v", err)
		}
	}

	// 检查 Filer
	if s.config.FilerURL != "" {
		if err := s.testFilerConnection(); err != nil {
			return fmt.Errorf("Filer 健康检查失败: %v", err)
		}
	}

	// 检查 S3 API
	if s.s3Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := s.s3Client.ListBuckets(ctx)
		if err != nil {
			return fmt.Errorf("S3 API 健康检查失败: %v", err)
		}
	}

	return nil
}
