package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// KeycloakService 定义 Keycloak 集成服务接口
type KeycloakService interface {
	// 用户管理
	CreateUser(user models.User, password string, roles []string) error
	UpdateUser(user models.User, roles []string) error
	DeleteUser(username string) error
	GetUser(username string) (*KeycloakUser, error)

	// 角色管理
	AssignRoles(username string, roles []string) error
	RemoveRoles(username string, roles []string) error
	GetUserRoles(username string) ([]string, error)

	// Token 验证
	ValidateToken(token string) (*KeycloakTokenInfo, error)
	IntrospectToken(token string) (*KeycloakIntrospectResponse, error)

	// OIDC 流程
	GetAuthorizationURL(state, redirectURI string) string
	ExchangeCodeForToken(code, redirectURI string) (*KeycloakTokenResponse, error)
	RefreshToken(refreshToken string) (*KeycloakTokenResponse, error)

	// 用户同步
	SyncAllUsers() (created, updated, skipped int, err error)
}

// keycloakServiceImpl Keycloak 服务实现
type keycloakServiceImpl struct {
	cfg           *config.Config
	db            *gorm.DB
	httpClient    *http.Client
	adminToken    string
	tokenExpireAt time.Time
}

// KeycloakUser Keycloak 用户结构
type KeycloakUser struct {
	ID            string               `json:"id,omitempty"`
	Username      string               `json:"username"`
	Email         string               `json:"email,omitempty"`
	FirstName     string               `json:"firstName,omitempty"`
	LastName      string               `json:"lastName,omitempty"`
	Enabled       bool                 `json:"enabled"`
	EmailVerified bool                 `json:"emailVerified"`
	Attributes    map[string][]string  `json:"attributes,omitempty"`
	Credentials   []KeycloakCredential `json:"credentials,omitempty"`
	RealmRoles    []string             `json:"realmRoles,omitempty"`
	Groups        []string             `json:"groups,omitempty"`
}

// KeycloakCredential Keycloak 凭证
type KeycloakCredential struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// KeycloakTokenResponse OIDC Token 响应
type KeycloakTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

// KeycloakIntrospectResponse Token 内省响应
type KeycloakIntrospectResponse struct {
	Active      bool         `json:"active"`
	Scope       string       `json:"scope,omitempty"`
	ClientID    string       `json:"client_id,omitempty"`
	Username    string       `json:"username,omitempty"`
	TokenType   string       `json:"token_type,omitempty"`
	Exp         int64        `json:"exp,omitempty"`
	Iat         int64        `json:"iat,omitempty"`
	Sub         string       `json:"sub,omitempty"`
	Aud         []string     `json:"aud,omitempty"`
	Iss         string       `json:"iss,omitempty"`
	RealmAccess *RealmAccess `json:"realm_access,omitempty"`
}

// RealmAccess Realm 访问权限
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// KeycloakTokenInfo Token 信息
type KeycloakTokenInfo struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
	Subject  string   `json:"sub"`
}

// KeycloakRole Keycloak 角色
type KeycloakRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Composite   bool   `json:"composite"`
}

// NewKeycloakService 创建 Keycloak 服务
func NewKeycloakService(cfg *config.Config, db *gorm.DB) KeycloakService {
	return &keycloakServiceImpl{
		cfg:        cfg,
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// 基础 URL 构建方法

func (s *keycloakServiceImpl) baseURL() string {
	return s.cfg.Keycloak.BaseURL
}

func (s *keycloakServiceImpl) realm() string {
	return s.cfg.Keycloak.Realm
}

func (s *keycloakServiceImpl) adminURL(path string) string {
	return fmt.Sprintf("%s/admin/realms/%s%s", s.baseURL(), s.realm(), path)
}

func (s *keycloakServiceImpl) oidcURL(path string) string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect%s", s.baseURL(), s.realm(), path)
}

// getAdminToken 获取管理员 Token
func (s *keycloakServiceImpl) getAdminToken(ctx context.Context) (string, error) {
	// 检查现有 Token 是否有效
	if s.adminToken != "" && time.Now().Before(s.tokenExpireAt.Add(-30*time.Second)) {
		return s.adminToken, nil
	}

	// 使用 client credentials 获取 Token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", s.cfg.Keycloak.ClientID)
	data.Set("client_secret", s.cfg.Keycloak.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", s.oidcURL("/token"), strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get admin token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get admin token: %s - %s", resp.Status, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	s.adminToken = tokenResp.AccessToken
	s.tokenExpireAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return s.adminToken, nil
}

// authRequest 添加认证头
func (s *keycloakServiceImpl) authRequest(ctx context.Context, req *http.Request) error {
	token, err := s.getAdminToken(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// CreateUser 创建 Keycloak 用户
func (s *keycloakServiceImpl) CreateUser(user models.User, password string, roles []string) error {
	if !s.cfg.Keycloak.Enabled {
		return nil
	}

	ctx := context.Background()
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"service":  "keycloak",
	})

	// 检查用户是否已存在
	existing, err := s.GetUser(user.Username)
	if err == nil && existing != nil {
		logger.Debug("User already exists in Keycloak, updating instead")
		return s.UpdateUser(user, roles)
	}

	// 构建 Keycloak 用户
	kcUser := KeycloakUser{
		Username:      user.Username,
		Email:         user.Email,
		FirstName:     user.Name,
		Enabled:       user.IsActive,
		EmailVerified: true,
		Attributes: map[string][]string{
			"ai_infra_user_id": {fmt.Sprintf("%d", user.ID)},
			"role_template":    {user.RoleTemplate},
		},
	}

	// 如果提供了密码，添加凭证
	if password != "" {
		kcUser.Credentials = []KeycloakCredential{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		}
	}

	// 创建用户
	body, err := json.Marshal(kcUser)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.adminURL("/users"), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create user: %s - %s", resp.Status, string(body))
	}

	logger.Info("Created user in Keycloak")

	// 分配角色
	if len(roles) > 0 {
		if err := s.AssignRoles(user.Username, roles); err != nil {
			logger.WithError(err).Warn("Failed to assign roles to user")
		}
	}

	return nil
}

// UpdateUser 更新 Keycloak 用户
func (s *keycloakServiceImpl) UpdateUser(user models.User, roles []string) error {
	if !s.cfg.Keycloak.Enabled {
		return nil
	}

	ctx := context.Background()
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"service":  "keycloak",
	})

	// 获取用户 ID
	kcUser, err := s.GetUser(user.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if kcUser == nil {
		return fmt.Errorf("user not found: %s", user.Username)
	}

	// 更新用户信息
	updateUser := KeycloakUser{
		Email:         user.Email,
		FirstName:     user.Name,
		Enabled:       user.IsActive,
		EmailVerified: true,
		Attributes: map[string][]string{
			"ai_infra_user_id": {fmt.Sprintf("%d", user.ID)},
			"role_template":    {user.RoleTemplate},
		},
	}

	body, err := json.Marshal(updateUser)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", s.adminURL(fmt.Sprintf("/users/%s", kcUser.ID)), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user: %s - %s", resp.Status, string(body))
	}

	logger.Info("Updated user in Keycloak")

	// 更新角色
	if len(roles) > 0 {
		if err := s.AssignRoles(user.Username, roles); err != nil {
			logger.WithError(err).Warn("Failed to update user roles")
		}
	}

	return nil
}

// DeleteUser 删除 Keycloak 用户
func (s *keycloakServiceImpl) DeleteUser(username string) error {
	if !s.cfg.Keycloak.Enabled {
		return nil
	}

	ctx := context.Background()

	// 获取用户 ID
	kcUser, err := s.GetUser(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if kcUser == nil {
		return nil // 用户不存在
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", s.adminURL(fmt.Sprintf("/users/%s", kcUser.ID)), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: %s - %s", resp.Status, string(body))
	}

	return nil
}

// GetUser 获取 Keycloak 用户
func (s *keycloakServiceImpl) GetUser(username string) (*KeycloakUser, error) {
	if !s.cfg.Keycloak.Enabled {
		return nil, nil
	}

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, "GET", s.adminURL(fmt.Sprintf("/users?username=%s&exact=true", url.QueryEscape(username))), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user: %s", resp.Status)
	}

	var users []KeycloakUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}

// AssignRoles 为用户分配角色
func (s *keycloakServiceImpl) AssignRoles(username string, roles []string) error {
	if !s.cfg.Keycloak.Enabled {
		return nil
	}

	ctx := context.Background()

	// 获取用户 ID
	kcUser, err := s.GetUser(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if kcUser == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	// 获取可用的 Realm 角色
	availableRoles, err := s.getRealmRoles(ctx)
	if err != nil {
		return fmt.Errorf("failed to get realm roles: %w", err)
	}

	// 筛选要分配的角色
	var rolesToAssign []KeycloakRole
	for _, roleName := range roles {
		for _, availableRole := range availableRoles {
			if strings.EqualFold(availableRole.Name, roleName) {
				rolesToAssign = append(rolesToAssign, availableRole)
				break
			}
		}
	}

	if len(rolesToAssign) == 0 {
		return nil
	}

	// 分配角色
	body, err := json.Marshal(rolesToAssign)
	if err != nil {
		return fmt.Errorf("failed to marshal roles: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		s.adminURL(fmt.Sprintf("/users/%s/role-mappings/realm", kcUser.ID)),
		bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to assign roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign roles: %s - %s", resp.Status, string(body))
	}

	return nil
}

// RemoveRoles 移除用户角色
func (s *keycloakServiceImpl) RemoveRoles(username string, roles []string) error {
	if !s.cfg.Keycloak.Enabled {
		return nil
	}

	ctx := context.Background()

	// 获取用户 ID
	kcUser, err := s.GetUser(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if kcUser == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	// 获取用户当前角色
	userRoles, err := s.getUserRealmRoles(ctx, kcUser.ID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	// 筛选要移除的角色
	var rolesToRemove []KeycloakRole
	for _, roleName := range roles {
		for _, userRole := range userRoles {
			if strings.EqualFold(userRole.Name, roleName) {
				rolesToRemove = append(rolesToRemove, userRole)
				break
			}
		}
	}

	if len(rolesToRemove) == 0 {
		return nil
	}

	// 移除角色
	body, err := json.Marshal(rolesToRemove)
	if err != nil {
		return fmt.Errorf("failed to marshal roles: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE",
		s.adminURL(fmt.Sprintf("/users/%s/role-mappings/realm", kcUser.ID)),
		bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove roles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove roles: %s - %s", resp.Status, string(body))
	}

	return nil
}

// GetUserRoles 获取用户角色
func (s *keycloakServiceImpl) GetUserRoles(username string) ([]string, error) {
	if !s.cfg.Keycloak.Enabled {
		return nil, nil
	}

	ctx := context.Background()

	// 获取用户 ID
	kcUser, err := s.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if kcUser == nil {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	roles, err := s.getUserRealmRoles(ctx, kcUser.ID)
	if err != nil {
		return nil, err
	}

	var roleNames []string
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}

	return roleNames, nil
}

// getRealmRoles 获取所有 Realm 角色
func (s *keycloakServiceImpl) getRealmRoles(ctx context.Context) ([]KeycloakRole, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.adminURL("/roles"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	defer resp.Body.Close()

	var roles []KeycloakRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return roles, nil
}

// getUserRealmRoles 获取用户的 Realm 角色
func (s *keycloakServiceImpl) getUserRealmRoles(ctx context.Context, userID string) ([]KeycloakRole, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		s.adminURL(fmt.Sprintf("/users/%s/role-mappings/realm", userID)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := s.authRequest(ctx, req); err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer resp.Body.Close()

	var roles []KeycloakRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return roles, nil
}

// ValidateToken 验证 Token
func (s *keycloakServiceImpl) ValidateToken(token string) (*KeycloakTokenInfo, error) {
	introspect, err := s.IntrospectToken(token)
	if err != nil {
		return nil, err
	}

	if !introspect.Active {
		return nil, fmt.Errorf("token is not active")
	}

	info := &KeycloakTokenInfo{
		Username: introspect.Username,
		Subject:  introspect.Sub,
	}

	if introspect.RealmAccess != nil {
		info.Roles = introspect.RealmAccess.Roles
	}

	return info, nil
}

// IntrospectToken Token 内省
func (s *keycloakServiceImpl) IntrospectToken(token string) (*KeycloakIntrospectResponse, error) {
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", s.cfg.Keycloak.ClientID)
	data.Set("client_secret", s.cfg.Keycloak.ClientSecret)

	req, err := http.NewRequest("POST", s.oidcURL("/token/introspect"), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}
	defer resp.Body.Close()

	var introspect KeycloakIntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&introspect); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &introspect, nil
}

// GetAuthorizationURL 获取授权 URL
func (s *keycloakServiceImpl) GetAuthorizationURL(state, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", s.cfg.Keycloak.ClientID)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", s.oidcURL("/auth"), params.Encode())
}

// ExchangeCodeForToken 用授权码换取 Token
func (s *keycloakServiceImpl) ExchangeCodeForToken(code, redirectURI string) (*KeycloakTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", s.cfg.Keycloak.ClientID)
	data.Set("client_secret", s.cfg.Keycloak.ClientSecret)

	req, err := http.NewRequest("POST", s.oidcURL("/token"), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to exchange code: %s - %s", resp.Status, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken 刷新 Token
func (s *keycloakServiceImpl) RefreshToken(refreshToken string) (*KeycloakTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", s.cfg.Keycloak.ClientID)
	data.Set("client_secret", s.cfg.Keycloak.ClientSecret)

	req, err := http.NewRequest("POST", s.oidcURL("/token"), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token: %s - %s", resp.Status, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// SyncAllUsers 同步所有用户到 Keycloak
func (s *keycloakServiceImpl) SyncAllUsers() (created, updated, skipped int, err error) {
	if !s.cfg.Keycloak.Enabled {
		return 0, 0, 0, nil
	}

	logger := logrus.WithField("service", "keycloak_user_sync")
	logger.Info("Starting Keycloak user sync")

	// 获取所有激活的用户
	var users []models.User
	if err := s.db.Where("is_active = ?", true).Find(&users).Error; err != nil {
		logger.WithError(err).Error("Failed to fetch users from database")
		return 0, 0, 0, err
	}

	for _, user := range users {
		// 获取用户角色
		roles := []string{}
		if user.RoleTemplate != "" {
			roles = append(roles, user.RoleTemplate)
		}

		// 检查用户是否存在
		kcUser, _ := s.GetUser(user.Username)

		if kcUser != nil {
			// 更新
			if err := s.UpdateUser(user, roles); err != nil {
				logger.WithError(err).WithField("username", user.Username).Warn("Failed to update user")
				skipped++
				continue
			}
			updated++
		} else {
			// 创建 (使用随机密码，用户需要通过找回密码设置)
			if err := s.CreateUser(user, "", roles); err != nil {
				logger.WithError(err).WithField("username", user.Username).Warn("Failed to create user")
				skipped++
				continue
			}
			created++
		}
	}

	logger.WithFields(logrus.Fields{
		"created": created,
		"updated": updated,
		"skipped": skipped,
	}).Info("Keycloak user sync completed")

	return created, updated, skipped, nil
}
