package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
)

// JupyterHubUserService JupyterHub 用户管理服务接口
type JupyterHubUserService interface {
	// EnsureUser 确保用户存在（创建或更新）
	EnsureUser(user models.User, password string, permissionLevel string, metadata JupyterHubUserMetadata) error
	// CreateUser 创建用户
	CreateUser(username string, admin bool) error
	// DeleteUser 删除用户
	DeleteUser(username string) error
	// GetUser 获取用户信息
	GetUser(username string) (*JupyterHubUser, error)
	// StartUserServer 启动用户的 Jupyter Server
	StartUserServer(username string) error
	// StopUserServer 停止用户的 Jupyter Server
	StopUserServer(username string) error
	// SetUserProperties 设置用户属性
	SetUserProperties(username string, properties JupyterHubUserProperties) error
	// AddUserToGroup 将用户添加到组
	AddUserToGroup(username string, groupName string) error
	// RemoveUserFromGroup 从组中移除用户
	RemoveUserFromGroup(username string, groupName string) error
	// ListUsers 列出所有用户
	ListUsers() ([]JupyterHubUser, error)
}

// jupyterHubUserServiceImpl JupyterHub 用户服务实现
type jupyterHubUserServiceImpl struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// JupyterHubUser JupyterHub 用户信息
type JupyterHubUser struct {
	Name         string                 `json:"name"`
	Admin        bool                   `json:"admin"`
	Groups       []string               `json:"groups"`
	Server       *JupyterHubServer      `json:"server"`
	Pending      string                 `json:"pending,omitempty"`
	Created      time.Time              `json:"created"`
	LastActivity time.Time              `json:"last_activity"`
	Servers      map[string]interface{} `json:"servers,omitempty"`
}

// JupyterHubServer 用户服务器信息
type JupyterHubServer struct {
	Name         string    `json:"name"`
	Ready        bool      `json:"ready"`
	Pending      string    `json:"pending,omitempty"`
	URL          string    `json:"url"`
	Started      time.Time `json:"started"`
	LastActivity time.Time `json:"last_activity"`
}

// JupyterHubUserProperties 用户属性
type JupyterHubUserProperties struct {
	Admin bool `json:"admin"`
}

// JupyterHubUserCreateRequest 创建用户请求
type JupyterHubUserCreateRequest struct {
	Usernames []string `json:"usernames"`
	Admin     bool     `json:"admin,omitempty"`
}

// NewJupyterHubUserService 创建 JupyterHub 用户服务
func NewJupyterHubUserService() JupyterHubUserService {
	baseURL := os.Getenv("JUPYTERHUB_API_URL")
	if baseURL == "" {
		baseURL = "http://jupyterhub:8081/hub/api"
	}

	apiToken := os.Getenv("JUPYTERHUB_API_TOKEN")

	return &jupyterHubUserServiceImpl{
		baseURL:  baseURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureUser 确保用户存在
func (s *jupyterHubUserServiceImpl) EnsureUser(user models.User, password string, permissionLevel string, metadata JupyterHubUserMetadata) error {
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"level":    permissionLevel,
		"service":  "jupyterhub_user",
	})

	// 检查用户是否存在
	existingUser, err := s.GetUser(user.Username)
	if err == nil && existingUser != nil {
		// 用户已存在，更新属性
		logger.Info("User already exists in JupyterHub, updating properties")
		isAdmin := permissionLevel == models.PermissionLevelAdmin
		if existingUser.Admin != isAdmin {
			if err := s.SetUserProperties(user.Username, JupyterHubUserProperties{Admin: isAdmin}); err != nil {
				logger.WithError(err).Warn("Failed to update user properties")
			}
		}
	} else {
		// 创建新用户
		isAdmin := permissionLevel == models.PermissionLevelAdmin
		if err := s.CreateUser(user.Username, isAdmin); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logger.Info("Created user in JupyterHub")
	}

	// 将用户添加到相应的组
	if len(metadata.AllowedGroups) > 0 {
		for _, group := range metadata.AllowedGroups {
			if err := s.AddUserToGroup(user.Username, group); err != nil {
				logger.WithError(err).WithField("group", group).Warn("Failed to add user to group")
			}
		}
	} else {
		// 根据权限级别添加到默认组
		defaultGroup := s.getDefaultGroup(permissionLevel)
		if defaultGroup != "" {
			if err := s.AddUserToGroup(user.Username, defaultGroup); err != nil {
				logger.WithError(err).Warn("Failed to add user to default group")
			}
		}
	}

	return nil
}

// CreateUser 创建用户
func (s *jupyterHubUserServiceImpl) CreateUser(username string, admin bool) error {
	url := fmt.Sprintf("%s/users/%s", s.baseURL, username)

	body := map[string]interface{}{
		"admin": admin,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	s.addAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 Created 或 409 Conflict（用户已存在）都是可接受的
	if resp.StatusCode >= 400 && resp.StatusCode != 409 {
		return fmt.Errorf("create user failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DeleteUser 删除用户
func (s *jupyterHubUserServiceImpl) DeleteUser(username string) error {
	// 先停止用户的服务器
	if err := s.StopUserServer(username); err != nil {
		logrus.WithError(err).Warn("Failed to stop user server before deletion")
	}

	url := fmt.Sprintf("%s/users/%s", s.baseURL, username)

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

// GetUser 获取用户信息
func (s *jupyterHubUserServiceImpl) GetUser(username string) (*JupyterHubUser, error) {
	url := fmt.Sprintf("%s/users/%s", s.baseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get user failed with status: %d", resp.StatusCode)
	}

	var user JupyterHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// StartUserServer 启动用户的 Jupyter Server
func (s *jupyterHubUserServiceImpl) StartUserServer(username string) error {
	url := fmt.Sprintf("%s/users/%s/server", s.baseURL, username)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 或 202 都表示成功
	if resp.StatusCode >= 400 {
		return fmt.Errorf("start server failed with status: %d", resp.StatusCode)
	}

	return nil
}

// StopUserServer 停止用户的 Jupyter Server
func (s *jupyterHubUserServiceImpl) StopUserServer(username string) error {
	url := fmt.Sprintf("%s/users/%s/server", s.baseURL, username)

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

	// 204 或 404 都是可接受的
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("stop server failed with status: %d", resp.StatusCode)
	}

	return nil
}

// SetUserProperties 设置用户属性
func (s *jupyterHubUserServiceImpl) SetUserProperties(username string, properties JupyterHubUserProperties) error {
	url := fmt.Sprintf("%s/users/%s", s.baseURL, username)

	jsonBody, _ := json.Marshal(properties)

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	s.addAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("update user properties failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AddUserToGroup 将用户添加到组
func (s *jupyterHubUserServiceImpl) AddUserToGroup(username string, groupName string) error {
	// 先确保组存在
	if err := s.ensureGroup(groupName); err != nil {
		logrus.WithError(err).WithField("group", groupName).Warn("Failed to ensure group exists")
	}

	url := fmt.Sprintf("%s/groups/%s/users", s.baseURL, groupName)

	body := map[string][]string{
		"users": {username},
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	s.addAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("add user to group failed with status: %d", resp.StatusCode)
	}

	return nil
}

// RemoveUserFromGroup 从组中移除用户
func (s *jupyterHubUserServiceImpl) RemoveUserFromGroup(username string, groupName string) error {
	url := fmt.Sprintf("%s/groups/%s/users", s.baseURL, groupName)

	body := map[string][]string{
		"users": {username},
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	s.addAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("remove user from group failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ListUsers 列出所有用户
func (s *jupyterHubUserServiceImpl) ListUsers() ([]JupyterHubUser, error) {
	url := fmt.Sprintf("%s/users", s.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list users failed with status: %d", resp.StatusCode)
	}

	var users []JupyterHubUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	return users, nil
}

// ensureGroup 确保组存在
func (s *jupyterHubUserServiceImpl) ensureGroup(groupName string) error {
	url := fmt.Sprintf("%s/groups/%s", s.baseURL, groupName)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	s.addAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 Created 或 409 Conflict 都是可接受的
	if resp.StatusCode >= 400 && resp.StatusCode != 409 {
		return fmt.Errorf("create group failed with status: %d", resp.StatusCode)
	}

	return nil
}

// getDefaultGroup 获取默认组
func (s *jupyterHubUserServiceImpl) getDefaultGroup(permissionLevel string) string {
	switch permissionLevel {
	case models.PermissionLevelAdmin:
		return "admins"
	case models.PermissionLevelUser:
		return "users"
	case models.PermissionLevelReadonly:
		return "viewers"
	default:
		return "users"
	}
}

// addAuth 添加认证头
func (s *jupyterHubUserServiceImpl) addAuth(req *http.Request) {
	if s.apiToken != "" {
		req.Header.Set("Authorization", "token "+s.apiToken)
	}
}
