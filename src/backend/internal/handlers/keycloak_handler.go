package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// KeycloakHandler 处理 Keycloak 相关的 API 请求
type KeycloakHandler struct {
	baseURL     string
	adminUser   string
	adminPass   string
	accessToken string
	tokenExpiry time.Time
	httpClient  *http.Client
}

// NewKeycloakHandler 创建新的 Keycloak Handler
func NewKeycloakHandler() *KeycloakHandler {
	// 从环境变量获取 Keycloak 配置
	baseURL := os.Getenv("KEYCLOAK_URL")
	if baseURL == "" {
		baseURL = "http://keycloak:8080/auth"
	}

	adminUser := os.Getenv("KEYCLOAK_ADMIN")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("KEYCLOAK_ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
	}

	// 创建 HTTP 客户端，支持自签名证书
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	return &KeycloakHandler{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		adminUser:  adminUser,
		adminPass:  adminPass,
		httpClient: client,
	}
}

// getAccessToken 获取管理员访问令牌
func (h *KeycloakHandler) getAccessToken() (string, error) {
	// 检查是否有有效的缓存令牌
	if h.accessToken != "" && time.Now().Before(h.tokenExpiry) {
		return h.accessToken, nil
	}

	// 获取新的访问令牌
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", h.baseURL)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", "admin-cli")
	data.Set("username", h.adminUser)
	data.Set("password", h.adminPass)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %v", err)
	}

	// 缓存令牌，提前5分钟过期
	h.accessToken = tokenResp.AccessToken
	h.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)

	return h.accessToken, nil
}

// doRequest 执行 Keycloak Admin API 请求
func (h *KeycloakHandler) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	token, err := h.getAccessToken()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get access token: %v", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	url := fmt.Sprintf("%s/admin%s", h.baseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

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

// GetServerInfo 获取服务器信息
func (h *KeycloakHandler) GetServerInfo(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/serverinfo", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get server info", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListRealms 获取所有 Realm 列表
func (h *KeycloakHandler) ListRealms(c *gin.Context) {
	body, statusCode, err := h.doRequest("GET", "/realms", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list realms", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRealm 获取单个 Realm 详情
func (h *KeycloakHandler) GetRealm(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s", realm), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get realm", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListUsers 获取 Realm 中的用户列表
func (h *KeycloakHandler) ListUsers(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	// 构建查询参数
	query := ""
	if search := c.Query("search"); search != "" {
		query = fmt.Sprintf("?search=%s", url.QueryEscape(search))
	}
	if max := c.Query("max"); max != "" {
		if query == "" {
			query = "?"
		} else {
			query += "&"
		}
		query += "max=" + max
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/users%s", realm, query), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list users", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUser 获取单个用户详情
func (h *KeycloakHandler) GetUser(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/users/%s", realm, userId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get user", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateUser 创建新用户
func (h *KeycloakHandler) CreateUser(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	var userData map[string]interface{}
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	body, statusCode, err := h.doRequest("POST", fmt.Sprintf("/realms/%s/users", realm), userData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to create user", "details": string(body)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// UpdateUser 更新用户
func (h *KeycloakHandler) UpdateUser(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	var userData map[string]interface{}
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	body, statusCode, err := h.doRequest("PUT", fmt.Sprintf("/realms/%s/users/%s", realm, userId), userData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusNoContent && statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to update user", "details": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser 删除用户
func (h *KeycloakHandler) DeleteUser(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("DELETE", fmt.Sprintf("/realms/%s/users/%s", realm, userId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusNoContent && statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to delete user", "details": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ListClients 获取 Realm 中的客户端列表
func (h *KeycloakHandler) ListClients(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/clients", realm), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list clients", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetClient 获取单个客户端详情
func (h *KeycloakHandler) GetClient(c *gin.Context) {
	realm := c.Param("realm")
	clientId := c.Param("clientId")
	if realm == "" || clientId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and client ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/clients/%s", realm, clientId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get client", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListGroups 获取 Realm 中的用户组列表
func (h *KeycloakHandler) ListGroups(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/groups", realm), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list groups", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetGroup 获取单个用户组详情
func (h *KeycloakHandler) GetGroup(c *gin.Context) {
	realm := c.Param("realm")
	groupId := c.Param("groupId")
	if realm == "" || groupId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and group ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/groups/%s", realm, groupId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get group", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListRoles 获取 Realm 中的角色列表
func (h *KeycloakHandler) ListRoles(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/roles", realm), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list roles", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUserRoles 获取用户的角色映射
func (h *KeycloakHandler) GetUserRoles(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/users/%s/role-mappings", realm, userId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get user roles", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListSessions 获取 Realm 的活跃会话列表
func (h *KeycloakHandler) ListSessions(c *gin.Context) {
	realm := c.Param("realm")
	if realm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm name is required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/sessions-stats", realm), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to list sessions", "details": string(body)})
		return
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetUserSessions 获取用户的会话列表
func (h *KeycloakHandler) GetUserSessions(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	body, statusCode, err := h.doRequest("GET", fmt.Sprintf("/realms/%s/users/%s/sessions", realm, userId), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to get user sessions", "details": string(body)})
		return
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ResetPassword 重置用户密码
func (h *KeycloakHandler) ResetPassword(c *gin.Context) {
	realm := c.Param("realm")
	userId := c.Param("userId")
	if realm == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Realm and user ID are required"})
		return
	}

	var passwordData struct {
		Type      string `json:"type"`
		Value     string `json:"value"`
		Temporary bool   `json:"temporary"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	passwordData.Type = "password"

	body, statusCode, err := h.doRequest("PUT", fmt.Sprintf("/realms/%s/users/%s/reset-password", realm, userId), passwordData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusNoContent && statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": "Failed to reset password", "details": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
