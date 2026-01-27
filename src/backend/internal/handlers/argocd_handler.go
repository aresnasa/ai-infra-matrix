package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// ArgoCDHandler 处理 ArgoCD 相关的 API 请求
type ArgoCDHandler struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewArgoCDHandler 创建新的 ArgoCD Handler
func NewArgoCDHandler() *ArgoCDHandler {
	// 从环境变量获取 ArgoCD 配置
	baseURL := os.Getenv("ARGOCD_SERVER_URL")
	if baseURL == "" {
		baseURL = "http://argocd-server:8080"
	}

	authToken := os.Getenv("ARGOCD_AUTH_TOKEN")

	// 创建 HTTP 客户端，支持自签名证书
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	return &ArgoCDHandler{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		authToken:  authToken,
		httpClient: client,
	}
}

// doRequest 执行 ArgoCD API 请求
func (h *ArgoCDHandler) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	url := fmt.Sprintf("%s/api/v1%s", h.baseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if h.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.authToken))
	}

	resp, err := h.httpClient.Do(req)
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

// ListApplications 获取所有应用列表
func (h *ArgoCDHandler) ListApplications(c *gin.Context) {
	projects := c.Query("projects")
	path := "/applications"
	if projects != "" {
		path += "?projects=" + projects
	}

	body, statusCode, err := h.doRequest("GET", path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetApplication 获取单个应用详情
func (h *ArgoCDHandler) GetApplication(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/applications/%s", name), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateApplication 创建新应用
func (h *ArgoCDHandler) CreateApplication(c *gin.Context) {
	var app interface{}
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body, statusCode, err := h.doRequest("POST", "/applications", app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteApplication 删除应用
func (h *ArgoCDHandler) DeleteApplication(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application name is required"})
		return
	}

	cascade := c.DefaultQuery("cascade", "true")
	path := fmt.Sprintf("/applications/%s?cascade=%s", name, cascade)

	body, statusCode, err := h.doRequest("DELETE", path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application deleted"})
}

// SyncApplication 同步应用
func (h *ArgoCDHandler) SyncApplication(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application name is required"})
		return
	}

	var syncRequest interface{}
	c.ShouldBindJSON(&syncRequest)

	body, statusCode, err := h.doRequest("POST", fmt.Sprintf("/applications/%s/sync", name), syncRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RefreshApplication 刷新应用状态
func (h *ArgoCDHandler) RefreshApplication(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application name is required"})
		return
	}

	hard := c.DefaultQuery("hard", "false")
	path := fmt.Sprintf("/applications/%s?refresh=%s", name, hard)

	body, statusCode, err := h.doRequest("GET", path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetApplicationResourceTree 获取应用资源树
func (h *ArgoCDHandler) GetApplicationResourceTree(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/applications/%s/resource-tree", name), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListRepositories 获取所有仓库列表
func (h *ArgoCDHandler) ListRepositories(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/repositories", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateRepository 创建仓库
func (h *ArgoCDHandler) CreateRepository(c *gin.Context) {
	var repo interface{}
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body, statusCode, err := h.doRequest("POST", "/repositories", repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteRepository 删除仓库
func (h *ArgoCDHandler) DeleteRepository(c *gin.Context) {
	repo := c.Param("repo")
	if repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository URL is required"})
		return
	}

	body, statusCode, err := h.doRequest("DELETE", fmt.Sprintf("/repositories/%s", repo), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "repository deleted"})
}

// ListClusters 获取所有集群列表
func (h *ArgoCDHandler) ListClusters(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/clusters", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListProjects 获取所有项目列表
func (h *ArgoCDHandler) ListProjects(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/projects", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetProject 获取项目详情
func (h *ArgoCDHandler) GetProject(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/projects/%s", name), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateProject 创建项目
func (h *ArgoCDHandler) CreateProject(c *gin.Context) {
	var project interface{}
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body, statusCode, err := h.doRequest("POST", "/projects", project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetVersion 获取 ArgoCD 版本信息
func (h *ArgoCDHandler) GetVersion(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/version", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSettings 获取 ArgoCD 设置
func (h *ArgoCDHandler) GetSettings(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/settings", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetArgoCDStatus 获取 ArgoCD 服务状态
func (h *ArgoCDHandler) GetArgoCDStatus(c *gin.Context) {
	argoCDService := services.GetArgoCDService()
	status := argoCDService.GetArgoCDStatus()
	c.JSON(http.StatusOK, status)
}

// SyncAllClusters 同步所有 K8s 集群到 ArgoCD
func (h *ArgoCDHandler) SyncAllClusters(c *gin.Context) {
	argoCDService := services.GetArgoCDService()

	if !argoCDService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "ArgoCD is not enabled or not available",
			"message": "请先启动 ArgoCD 服务：docker-compose --profile argocd up -d",
		})
		return
	}

	if err := argoCDService.SyncAllClustersToArgoCD(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "所有已连接的 K8s 集群已同步到 ArgoCD",
	})
}

// RefreshArgoCDAvailability 刷新 ArgoCD 可用性状态
func (h *ArgoCDHandler) RefreshArgoCDAvailability(c *gin.Context) {
	argoCDService := services.GetArgoCDService()
	argoCDService.RefreshAvailability()
	status := argoCDService.GetArgoCDStatus()
	c.JSON(http.StatusOK, status)
}

// ListArgoCDManagedClusters 列出 ArgoCD 管理的所有集群
func (h *ArgoCDHandler) ListArgoCDManagedClusters(c *gin.Context) {
	argoCDService := services.GetArgoCDService()

	if !argoCDService.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "ArgoCD is not enabled",
			"message": "请先启动 ArgoCD 服务",
		})
		return
	}

	clusters, err := argoCDService.ListArgoCDClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": clusters,
		"count": len(clusters),
	})
}
