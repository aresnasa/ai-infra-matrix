package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// ArgoCDService ArgoCD 集成服务
// 用于将 KubernetesCluster 同步到 ArgoCD
type ArgoCDService struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
	enabled    bool
	mu         sync.RWMutex
}

// ArgoCDCluster ArgoCD 集群配置
type ArgoCDCluster struct {
	Server string            `json:"server"`
	Name   string            `json:"name"`
	Config ArgoCDClusterConfig `json:"config"`
}

// ArgoCDClusterConfig ArgoCD 集群认证配置
type ArgoCDClusterConfig struct {
	// Bearer Token 认证
	BearerToken string `json:"bearerToken,omitempty"`
	// TLS 配置
	TLSClientConfig *ArgoCDTLSClientConfig `json:"tlsClientConfig,omitempty"`
}

// ArgoCDTLSClientConfig TLS 客户端配置
type ArgoCDTLSClientConfig struct {
	Insecure bool   `json:"insecure,omitempty"`
	CAData   string `json:"caData,omitempty"`
	CertData string `json:"certData,omitempty"`
	KeyData  string `json:"keyData,omitempty"`
}

// ArgoCDClusterResponse ArgoCD 集群响应
type ArgoCDClusterResponse struct {
	Items []ArgoCDClusterItem `json:"items"`
}

// ArgoCDClusterItem ArgoCD 集群项
type ArgoCDClusterItem struct {
	Server     string            `json:"server"`
	Name       string            `json:"name"`
	Config     ArgoCDClusterConfig `json:"config"`
	ServerVersion string          `json:"serverVersion,omitempty"`
	ConnectionState *ArgoCDConnectionState `json:"connectionState,omitempty"`
}

// ArgoCDConnectionState 连接状态
type ArgoCDConnectionState struct {
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	ModifiedAt string `json:"attemptedAt,omitempty"`
}

var (
	argoCDServiceInstance *ArgoCDService
	argoCDServiceOnce     sync.Once
)

// GetArgoCDService 获取 ArgoCD 服务单例
func GetArgoCDService() *ArgoCDService {
	argoCDServiceOnce.Do(func() {
		argoCDServiceInstance = newArgoCDService()
	})
	return argoCDServiceInstance
}

// newArgoCDService 创建 ArgoCD 服务
func newArgoCDService() *ArgoCDService {
	baseURL := os.Getenv("ARGOCD_SERVER_URL")
	if baseURL == "" {
		baseURL = "http://argocd-server:8080"
	}

	authToken := os.Getenv("ARGOCD_AUTH_TOKEN")

	// 创建 HTTP 客户端
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	svc := &ArgoCDService{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		authToken:  authToken,
		httpClient: client,
		enabled:    false,
	}

	// 检查 ArgoCD 是否可用
	go svc.checkAvailability()

	return svc
}

// checkAvailability 检查 ArgoCD 服务是否可用
func (s *ArgoCDService) checkAvailability() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 尝试访问 ArgoCD 版本接口
	url := fmt.Sprintf("%s/api/v1/version", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.WithError(err).Debug("ArgoCD service check: failed to create request")
		s.enabled = false
		return
	}

	if s.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Debug("ArgoCD service check: connection failed (service may not be running)")
		s.enabled = false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		s.enabled = true
		logrus.Info("ArgoCD service is available and enabled")
	} else {
		s.enabled = false
		logrus.WithField("status", resp.StatusCode).Debug("ArgoCD service check: unexpected status code")
	}
}

// IsEnabled 检查 ArgoCD 是否启用
func (s *ArgoCDService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// RefreshAvailability 刷新可用性状态
func (s *ArgoCDService) RefreshAvailability() {
	s.checkAvailability()
}

// doRequest 执行 ArgoCD API 请求
func (s *ArgoCDService) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	url := fmt.Sprintf("%s/api/v1%s", s.baseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}

	return respBody, resp.StatusCode, nil
}

// SyncClusterToArgoCD 将 KubernetesCluster 同步到 ArgoCD
func (s *ArgoCDService) SyncClusterToArgoCD(cluster *models.KubernetesCluster) error {
	if !s.IsEnabled() {
		logrus.Debug("ArgoCD is not enabled, skipping cluster sync")
		return nil
	}

	if cluster.KubeConfig == "" {
		return fmt.Errorf("kubeconfig is empty")
	}

	// 解析 kubeconfig 获取集群信息
	kubeConfig := cluster.KubeConfig
	if database.CryptoService != nil && database.CryptoService.IsEncrypted(kubeConfig) {
		kubeConfig = database.CryptoService.DecryptSafely(kubeConfig)
	}

	// 从 kubeconfig 提取必要信息
	clusterConfig, err := s.parseKubeConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %v", err)
	}

	// 构建 ArgoCD 集群配置
	argoCDCluster := map[string]interface{}{
		"server": cluster.APIServer,
		"name":   fmt.Sprintf("ai-infra-%d-%s", cluster.ID, cluster.Name),
		"config": clusterConfig,
	}

	// 检查集群是否已存在
	existingCluster, err := s.getClusterByServer(cluster.APIServer)
	if err == nil && existingCluster != nil {
		// 更新现有集群
		logrus.WithField("server", cluster.APIServer).Info("Updating existing cluster in ArgoCD")
		body, statusCode, err := s.doRequest("PUT", fmt.Sprintf("/clusters/%s", cluster.APIServer), argoCDCluster)
		if err != nil {
			return fmt.Errorf("failed to update cluster: %v", err)
		}
		if statusCode != http.StatusOK {
			return fmt.Errorf("failed to update cluster, status: %d, body: %s", statusCode, string(body))
		}
	} else {
		// 创建新集群
		logrus.WithField("server", cluster.APIServer).Info("Adding new cluster to ArgoCD")
		body, statusCode, err := s.doRequest("POST", "/clusters", argoCDCluster)
		if err != nil {
			return fmt.Errorf("failed to create cluster: %v", err)
		}
		if statusCode != http.StatusOK && statusCode != http.StatusCreated {
			return fmt.Errorf("failed to create cluster, status: %d, body: %s", statusCode, string(body))
		}
	}

	logrus.WithFields(logrus.Fields{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"api_server":   cluster.APIServer,
	}).Info("Successfully synced cluster to ArgoCD")

	return nil
}

// RemoveClusterFromArgoCD 从 ArgoCD 移除集群
func (s *ArgoCDService) RemoveClusterFromArgoCD(cluster *models.KubernetesCluster) error {
	if !s.IsEnabled() {
		logrus.Debug("ArgoCD is not enabled, skipping cluster removal")
		return nil
	}

	// 使用 API Server URL 作为集群标识
	serverURL := cluster.APIServer
	if serverURL == "" {
		return fmt.Errorf("API server URL is empty")
	}

	body, statusCode, err := s.doRequest("DELETE", fmt.Sprintf("/clusters/%s", serverURL), nil)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %v", err)
	}

	// 404 表示集群不存在，可以忽略
	if statusCode == http.StatusNotFound {
		logrus.WithField("server", serverURL).Debug("Cluster not found in ArgoCD, already removed")
		return nil
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete cluster, status: %d, body: %s", statusCode, string(body))
	}

	logrus.WithFields(logrus.Fields{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"api_server":   serverURL,
	}).Info("Successfully removed cluster from ArgoCD")

	return nil
}

// getClusterByServer 通过 server URL 获取 ArgoCD 集群
func (s *ArgoCDService) getClusterByServer(server string) (*ArgoCDClusterItem, error) {
	body, statusCode, err := s.doRequest("GET", fmt.Sprintf("/clusters/%s", server), nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	var cluster ArgoCDClusterItem
	if err := json.Unmarshal(body, &cluster); err != nil {
		return nil, err
	}
	return &cluster, nil
}

// ListArgoCDClusters 列出 ArgoCD 中的所有集群
func (s *ArgoCDService) ListArgoCDClusters() ([]ArgoCDClusterItem, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("ArgoCD is not enabled")
	}

	body, statusCode, err := s.doRequest("GET", "/clusters", nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	var response ArgoCDClusterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	return response.Items, nil
}

// parseKubeConfig 解析 kubeconfig 提取认证信息
func (s *ArgoCDService) parseKubeConfig(kubeConfig string) (map[string]interface{}, error) {
	// 简单的 YAML 解析，提取必要的认证信息
	// 这里使用简化的方式，实际生产环境可能需要更复杂的解析
	
	config := make(map[string]interface{})
	
	// 检查是否包含 token
	if strings.Contains(kubeConfig, "token:") {
		// 提取 token (简化处理)
		lines := strings.Split(kubeConfig, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "token:") {
				token := strings.TrimPrefix(trimmed, "token:")
				token = strings.TrimSpace(token)
				config["bearerToken"] = token
				break
			}
		}
	}
	
	// 检查是否包含客户端证书
	if strings.Contains(kubeConfig, "client-certificate-data:") {
		tlsConfig := make(map[string]interface{})
		lines := strings.Split(kubeConfig, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "client-certificate-data:") {
				certData := strings.TrimPrefix(trimmed, "client-certificate-data:")
				tlsConfig["certData"] = strings.TrimSpace(certData)
			}
			if strings.HasPrefix(trimmed, "client-key-data:") {
				keyData := strings.TrimPrefix(trimmed, "client-key-data:")
				tlsConfig["keyData"] = strings.TrimSpace(keyData)
			}
			if strings.HasPrefix(trimmed, "certificate-authority-data:") {
				caData := strings.TrimPrefix(trimmed, "certificate-authority-data:")
				tlsConfig["caData"] = strings.TrimSpace(caData)
			}
		}
		if len(tlsConfig) > 0 {
			config["tlsClientConfig"] = tlsConfig
		}
	}
	
	// 如果没有找到认证信息，使用 insecure 模式
	if len(config) == 0 {
		config["tlsClientConfig"] = map[string]interface{}{
			"insecure": true,
		}
	}
	
	return config, nil
}

// SyncAllClustersToArgoCD 同步所有 K8s 集群到 ArgoCD
func (s *ArgoCDService) SyncAllClustersToArgoCD() error {
	if !s.IsEnabled() {
		return fmt.Errorf("ArgoCD is not enabled")
	}

	var clusters []models.KubernetesCluster
	if err := database.DB.Where("status = ? AND is_active = ?", "connected", true).Find(&clusters).Error; err != nil {
		return fmt.Errorf("failed to fetch clusters: %v", err)
	}

	var errs []string
	for _, cluster := range clusters {
		if err := s.SyncClusterToArgoCD(&cluster); err != nil {
			errs = append(errs, fmt.Sprintf("cluster %s: %v", cluster.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some clusters failed to sync: %s", strings.Join(errs, "; "))
	}

	logrus.WithField("count", len(clusters)).Info("Successfully synced all clusters to ArgoCD")
	return nil
}

// GetArgoCDStatus 获取 ArgoCD 服务状态
func (s *ArgoCDService) GetArgoCDStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled":  s.IsEnabled(),
		"base_url": s.baseURL,
	}

	if s.IsEnabled() {
		// 尝试获取版本信息
		body, statusCode, err := s.doRequest("GET", "/version", nil)
		if err == nil && statusCode == http.StatusOK {
			var version map[string]interface{}
			if json.Unmarshal(body, &version) == nil {
				status["version"] = version
			}
		}

		// 获取集群数量
		clusters, err := s.ListArgoCDClusters()
		if err == nil {
			status["cluster_count"] = len(clusters)
		}
	}

	return status
}
