package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NightingaleUserService 定义 Nightingale 用户同步服务接口
type NightingaleUserService interface {
	// EnsureUser 创建或更新 Nightingale 用户
	EnsureUser(u models.User, roles []string) error
	// SyncAllUsers 同步所有用户到 Nightingale
	SyncAllUsers() (created, updated, skipped int, err error)
	// DeleteUser 删除 Nightingale 用户
	DeleteUser(username string) error
	// UpdateUserRoles 更新用户角色
	UpdateUserRoles(username string, roles []string) error
}

// nightingaleUserServiceImpl Nightingale 用户同步服务实现
type nightingaleUserServiceImpl struct {
	cfg        *config.Config
	db         *gorm.DB
	httpClient *http.Client
}

// N9eUserCreateReq Nightingale 用户创建请求
type N9eUserCreateReq struct {
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone,omitempty"`
	Roles    []string `json:"roles"`
}

// N9eUserUpdateReq Nightingale 用户更新请求
type N9eUserUpdateReq struct {
	Nickname string   `json:"nickname,omitempty"`
	Email    string   `json:"email,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

// N9eUserResp Nightingale 用户响应
type N9eUserResp struct {
	ID       int64    `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Roles    []string `json:"roles"`
}

// N9eAPIResponse Nightingale API 通用响应
type N9eAPIResponse struct {
	Err string      `json:"err"`
	Dat interface{} `json:"dat"`
}

// NewNightingaleUserService 创建 Nightingale 用户同步服务
func NewNightingaleUserService(cfg *config.Config, db *gorm.DB) NightingaleUserService {
	return &nightingaleUserServiceImpl{
		cfg:        cfg,
		db:         db,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// api 构建 Nightingale API 路径
func (s *nightingaleUserServiceImpl) api(path string) string {
	baseURL := s.cfg.Nightingale.BaseURL
	if baseURL == "" {
		baseURL = "http://nightingale:17000"
	}
	return fmt.Sprintf("%s/api/n9e%s", baseURL, path)
}

// auth 设置认证头
func (s *nightingaleUserServiceImpl) auth(req *http.Request) {
	// Nightingale 使用 Basic Auth 进行 API 认证
	username := s.cfg.Nightingale.APIUsername
	password := s.cfg.Nightingale.APIPassword
	if username == "" {
		username = "n9e-api"
	}
	if password == "" {
		// 默认使用 MD5 哈希
		password = "e10adc3949ba59abbe56e057f20f883e" // MD5 of "123456"
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")
}

// md5Hash 计算 MD5 哈希
func md5Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// mapRolesToN9e 将系统角色映射到 Nightingale 角色
func mapRolesToN9e(roles []string) []string {
	n9eRoles := []string{}
	for _, role := range roles {
		switch strings.ToLower(role) {
		case "admin", "administrator", "system_admin":
			n9eRoles = append(n9eRoles, "Admin")
		case "sre", "ops", "devops":
			n9eRoles = append(n9eRoles, "Admin")
		case "engineer", "developer":
			n9eRoles = append(n9eRoles, "Standard")
		case "viewer", "readonly", "guest":
			n9eRoles = append(n9eRoles, "Guest")
		default:
			// 默认为 Standard 角色
			if len(n9eRoles) == 0 {
				n9eRoles = append(n9eRoles, "Standard")
			}
		}
	}
	if len(n9eRoles) == 0 {
		n9eRoles = []string{"Standard"}
	}
	// 去重
	seen := make(map[string]bool)
	unique := []string{}
	for _, r := range n9eRoles {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// EnsureUser 创建或更新 Nightingale 用户
func (s *nightingaleUserServiceImpl) EnsureUser(u models.User, roles []string) error {
	if !s.cfg.Nightingale.Enabled {
		return nil
	}

	logger := logrus.WithFields(logrus.Fields{
		"username": u.Username,
		"service":  "nightingale_user_sync",
	})

	// 检查用户是否存在
	exists, existingUser, err := s.getUserByUsername(u.Username)
	if err != nil {
		logger.WithError(err).Warn("Failed to check user existence in Nightingale")
	}

	n9eRoles := mapRolesToN9e(roles)

	if exists && existingUser != nil {
		// 更新用户
		updateReq := N9eUserUpdateReq{
			Nickname: u.Name,
			Email:    u.Email,
			Roles:    n9eRoles,
		}
		if err := s.updateUser(existingUser.ID, updateReq); err != nil {
			logger.WithError(err).Warn("Failed to update user in Nightingale")
			return err
		}
		logger.Info("Updated user in Nightingale")
	} else {
		// 创建用户
		// 使用默认密码 (MD5 哈希)
		defaultPassword := md5Hash("ai-infra-n9e-default")
		createReq := N9eUserCreateReq{
			Username: u.Username,
			Password: defaultPassword,
			Nickname: u.Name,
			Email:    u.Email,
			Roles:    n9eRoles,
		}
		if err := s.createUser(createReq); err != nil {
			logger.WithError(err).Warn("Failed to create user in Nightingale")
			return err
		}
		logger.Info("Created user in Nightingale")
	}

	// 同步到本地数据库
	s.syncToLocalDB(u, n9eRoles)

	return nil
}

// getUserByUsername 通过用户名获取用户
func (s *nightingaleUserServiceImpl) getUserByUsername(username string) (bool, *N9eUserResp, error) {
	req, err := http.NewRequest("GET", s.api("/user?limit=1&query="+username), nil)
	if err != nil {
		return false, nil, err
	}
	s.auth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil, nil
	}

	var apiResp N9eAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return false, nil, err
	}

	if apiResp.Err != "" {
		return false, nil, fmt.Errorf("N9e API error: %s", apiResp.Err)
	}

	// 解析用户列表
	if dat, ok := apiResp.Dat.(map[string]interface{}); ok {
		if list, ok := dat["list"].([]interface{}); ok && len(list) > 0 {
			// 找到匹配的用户
			for _, item := range list {
				if userData, ok := item.(map[string]interface{}); ok {
					if userData["username"] == username {
						user := &N9eUserResp{
							ID:       int64(userData["id"].(float64)),
							Username: userData["username"].(string),
							Nickname: userData["nickname"].(string),
							Email:    userData["email"].(string),
						}
						return true, user, nil
					}
				}
			}
		}
	}

	return false, nil, nil
}

// createUser 创建用户
func (s *nightingaleUserServiceImpl) createUser(req N9eUserCreateReq) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", s.api("/user"), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	s.auth(httpReq)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiResp N9eAPIResponse
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return fmt.Errorf("failed to create user: %s", apiResp.Err)
	}

	return nil
}

// updateUser 更新用户
func (s *nightingaleUserServiceImpl) updateUser(id int64, req N9eUserUpdateReq) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("PUT", s.api(fmt.Sprintf("/user/%d", id)), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	s.auth(httpReq)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiResp N9eAPIResponse
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return fmt.Errorf("failed to update user: %s", apiResp.Err)
	}

	return nil
}

// DeleteUser 删除用户
func (s *nightingaleUserServiceImpl) DeleteUser(username string) error {
	exists, user, err := s.getUserByUsername(username)
	if err != nil {
		return err
	}
	if !exists || user == nil {
		return nil // 用户不存在，无需删除
	}

	httpReq, err := http.NewRequest("DELETE", s.api(fmt.Sprintf("/user/%d", user.ID)), nil)
	if err != nil {
		return err
	}
	s.auth(httpReq)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiResp N9eAPIResponse
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return fmt.Errorf("failed to delete user: %s", apiResp.Err)
	}

	// 从本地数据库删除
	s.db.Where("username = ?", username).Delete(&models.NightingaleUser{})

	return nil
}

// UpdateUserRoles 更新用户角色
func (s *nightingaleUserServiceImpl) UpdateUserRoles(username string, roles []string) error {
	exists, user, err := s.getUserByUsername(username)
	if err != nil {
		return err
	}
	if !exists || user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	n9eRoles := mapRolesToN9e(roles)
	return s.updateUser(user.ID, N9eUserUpdateReq{Roles: n9eRoles})
}

// SyncAllUsers 同步所有用户到 Nightingale
func (s *nightingaleUserServiceImpl) SyncAllUsers() (created, updated, skipped int, err error) {
	if !s.cfg.Nightingale.Enabled {
		return 0, 0, 0, nil
	}

	logger := logrus.WithField("service", "nightingale_user_sync")
	logger.Info("Starting Nightingale user sync")

	// 获取所有激活的用户
	var users []models.User
	if err := s.db.Where("is_active = ?", true).Find(&users).Error; err != nil {
		logger.WithError(err).Error("Failed to fetch users from database")
		return 0, 0, 0, err
	}

	for _, user := range users {
		// 获取用户角色
		var roles []string
		if user.RoleTemplate != "" {
			roles = []string{user.RoleTemplate}
		} else {
			roles = []string{"viewer"}
		}

		err := s.EnsureUser(user, roles)
		if err != nil {
			logger.WithError(err).WithField("username", user.Username).Warn("Failed to sync user")
			skipped++
			continue
		}

		// 简化计数逻辑
		created++ // 这里简化处理，实际应该区分创建和更新
	}

	logger.WithFields(logrus.Fields{
		"created": created,
		"updated": updated,
		"skipped": skipped,
	}).Info("Nightingale user sync completed")

	return created, updated, skipped, nil
}

// syncToLocalDB 同步到本地数据库
func (s *nightingaleUserServiceImpl) syncToLocalDB(u models.User, roles []string) {
	n9eUser := models.NightingaleUser{
		Username: u.Username,
		Nickname: u.Name,
		Email:    u.Email,
		Roles:    strings.Join(roles, ","),
	}

	// Upsert
	var existing models.NightingaleUser
	if err := s.db.Where("username = ?", u.Username).First(&existing).Error; err == nil {
		n9eUser.ID = existing.ID
		s.db.Save(&n9eUser)
	} else {
		s.db.Create(&n9eUser)
	}
}
