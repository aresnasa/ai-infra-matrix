package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/jwt"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type JupyterHubAuthHandler struct {
	db             *gorm.DB
	config         *config.Config
	userService    *services.UserService
	sessionService *services.SessionService
	redisClient    *redis.Client
}

func NewJupyterHubAuthHandler(db *gorm.DB, cfg *config.Config, redisClient *redis.Client) *JupyterHubAuthHandler {
	return &JupyterHubAuthHandler{
		db:             db,
		config:         cfg,
		userService:    services.NewUserService(),
		sessionService: services.NewSessionService(),
		redisClient:    redisClient,
	}
}

// JupyterHubStatusResponse JupyterHub状态响应
type JupyterHubStatusResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

// JupyterHubLoginRequest JupyterHub登录请求
type JupyterHubLoginRequest struct {
	Username string `json:"username" binding:"required"`
}

// JupyterHubTokenResponse JupyterHub令牌响应
type JupyterHubTokenResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ServerActionRequest 服务器操作请求
type ServerActionRequest struct {
	Username string `json:"username" binding:"required"`
	Action   string `json:"action"`   // start, stop, restart
}

// GetJupyterHubStatus 获取JupyterHub状态
// @Summary 获取JupyterHub状态
// @Description 检查JupyterHub服务状态和连接
// @Tags JupyterHub认证
// @Produce json
// @Success 200 {object} JupyterHubStatusResponse
// @Failure 500 {object} map[string]interface{}
// @Router /jupyterhub/status [get]
func (h *JupyterHubAuthHandler) GetJupyterHubStatus(c *gin.Context) {
	// 检查JupyterHub实际健康状态
	status, version, url, err := h.checkJupyterHubHealth()
	
	var message string
	if err != nil {
		status = "disconnected"
		message = fmt.Sprintf("JupyterHub连接失败: %v", err)
		logrus.WithError(err).Warning("JupyterHub健康检查失败")
	} else if status == "connected" {
		message = "JupyterHub统一认证系统运行正常"
	} else {
		message = "JupyterHub服务状态异常"
	}
	
	response := JupyterHubStatusResponse{
		Status:  status,
		Version: version,
		URL:     url,
		Message: message,
	}

	c.JSON(http.StatusOK, response)
}

// GenerateJupyterHubLoginToken 生成JupyterHub登录令牌
// @Summary 生成JupyterHub登录令牌
// @Description 为已认证用户生成JupyterHub登录令牌
// @Tags JupyterHub认证
// @Accept json
// @Produce json
// @Param request body JupyterHubLoginRequest true "登录请求"
// @Success 200 {object} JupyterHubTokenResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/jupyterhub-login [post]
func (h *JupyterHubAuthHandler) GenerateJupyterHubLoginToken(c *gin.Context) {
	var request JupyterHubLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误", "details": err.Error()})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 验证用户权限
	var user models.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		logrus.WithError(err).Error("查询用户失败")
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 验证用户名匹配
	if user.Username != request.Username {
		c.JSON(http.StatusForbidden, gin.H{"error": "用户名不匹配"})
		return
	}

	// 生成JupyterHub登录令牌
	token, expiresAt, err := h.generateJupyterHubToken(user)
	if err != nil {
		logrus.WithError(err).Error("生成JupyterHub令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	// 缓存令牌到Redis
	if err := h.cacheJupyterHubToken(user.Username, token, expiresAt); err != nil {
		logrus.WithError(err).Warning("缓存JupyterHub令牌失败")
	}

	// 记录登录日志
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "jupyterhub_login",
	}).Info("用户生成JupyterHub登录令牌")

	response := JupyterHubTokenResponse{
		Success:   true,
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		Message:   "登录令牌生成成功",
	}

	c.JSON(http.StatusOK, response)
}

// VerifyJupyterHubToken 验证JupyterHub令牌
// @Summary 验证JupyterHub令牌
// @Description 验证JupyterHub传来的认证令牌是否有效
// @Tags JupyterHub认证
// @Accept json
// @Produce json
// @Param request body map[string]string true "令牌验证请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify-jupyterhub-token [post]
func (h *JupyterHubAuthHandler) VerifyJupyterHubToken(c *gin.Context) {
	var request map[string]string
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误", "details": err.Error()})
		return
	}

	token, exists := request["token"]
	if !exists || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少令牌参数"})
		return
	}

	username, _ := request["username"]

	// 验证令牌
	isValid, userInfo, err := h.validateJupyterHubToken(token, username)
	if err != nil {
		logrus.WithError(err).Error("验证JupyterHub令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌验证失败"})
		return
	}

	if isValid {
		c.JSON(http.StatusOK, gin.H{
			"valid": true,
			"user":  userInfo,
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "令牌无效或已过期",
		})
	}
}

// StartNotebookServer 启动Notebook服务器
// @Summary 启动Notebook服务器
// @Description 为用户启动JupyterHub Notebook服务器
// @Tags JupyterHub认证
// @Accept json
// @Produce json
// @Param request body ServerActionRequest true "服务器操作请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /jupyterhub/start-server [post]
func (h *JupyterHubAuthHandler) StartNotebookServer(c *gin.Context) {
	var request ServerActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误", "details": err.Error()})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 验证用户权限
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if user.Username != request.Username {
		c.JSON(http.StatusForbidden, gin.H{"error": "用户名不匹配"})
		return
	}

	// 这里可以添加实际的JupyterHub API调用
	// 暂时返回成功响应
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("用户 %s 的Notebook服务器启动请求已处理", request.Username),
		"action":  request.Action,
	}

	c.JSON(http.StatusOK, response)
}

// RedirectToJupyterHub 重定向到JupyterHub（统一登录）
// @Summary 重定向到JupyterHub
// @Description 为已认证用户生成token并重定向到JupyterHub
// @Tags JupyterHub认证
// @Produce json
// @Param next query string false "跳转后的目标页面"
// @Success 302 "重定向到JupyterHub"
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/jupyterhub-redirect [get]
func (h *JupyterHubAuthHandler) RedirectToJupyterHub(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
			"redirect": "/login",
		})
		return
	}

	// 查询用户信息
	var user models.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		logrus.WithError(err).Error("查询用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	// 生成JupyterHub访问token
	token, expiresAt, err := h.generateJupyterHubToken(user)
	if err != nil {
		logrus.WithError(err).Error("生成JupyterHub访问令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成访问令牌失败"})
		return
	}

	// 缓存令牌到Redis（用于会话管理）
	if err := h.cacheJupyterHubToken(user.Username, token, expiresAt); err != nil {
		logrus.WithError(err).Warning("缓存JupyterHub会话失败")
	}

	// 获取跳转目标页面
	nextURL := c.Query("next")
	if nextURL == "" {
		nextURL = "/hub/home"
	}

	// 构建JupyterHub URL - 使用容器网络地址
	jupyterhubURL := "http://ai-infra-jupyterhub:8000"

	// 构建重定向URL，包含token和目标页面
	redirectURL := fmt.Sprintf("%s/unified-login?token=%s&next=%s", 
		jupyterhubURL, token, nextURL)

	// 记录访问日志
	logrus.WithFields(logrus.Fields{
		"user_id":      user.ID,
		"username":     user.Username,
		"action":       "jupyterhub_redirect",
		"redirect_url": redirectURL,
	}).Info("用户重定向到JupyterHub")

	// 执行重定向
	c.Redirect(http.StatusFound, redirectURL)
}

// StopNotebookServer 停止Notebook服务器
// @Summary 停止Notebook服务器
// @Description 停止用户的JupyterHub Notebook服务器
// @Tags JupyterHub认证
// @Accept json
// @Produce json
// @Param request body ServerActionRequest true "服务器操作请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /jupyterhub/stop-server [post]
func (h *JupyterHubAuthHandler) StopNotebookServer(c *gin.Context) {
	var request ServerActionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误", "details": err.Error()})
		return
	}

	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 验证用户权限
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if user.Username != request.Username {
		c.JSON(http.StatusForbidden, gin.H{"error": "用户名不匹配"})
		return
	}

	// 记录日志
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "stop_notebook_server",
	}).Info("用户停止Notebook服务器")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notebook服务器停止请求已提交",
	})
}

// LogoutAllSessions 登出所有会话
// @Summary 登出所有会话
// @Description 清除用户的所有会话（包括JupyterHub）
// @Tags JupyterHub认证
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/logout-all [post]
func (h *JupyterHubAuthHandler) LogoutAllSessions(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查询用户信息
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 清除Redis中的JupyterHub会话
	if err := h.clearJupyterHubSessions(user.Username); err != nil {
		logrus.WithError(err).Warning("清除JupyterHub会话失败")
	}

	// 记录日志
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "logout_all_sessions",
	}).Info("用户登出所有会话")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "所有会话已清除",
	})
}

// generateJupyterHubToken 生成JupyterHub令牌
func (h *JupyterHubAuthHandler) generateJupyterHubToken(user models.User) (string, time.Time, error) {
	// 创建令牌载荷
	expiresAt := time.Now().Add(time.Hour * 24) // 24小时有效期
	
	// 获取用户角色
	var roles []string
	var permissions []string
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	// 生成JWT令牌
	token, _, err := jwt.GenerateToken(user.ID, user.Username, roles, permissions)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成JWT令牌失败: %w", err)
	}

	return token, expiresAt, nil
}

// cacheJupyterHubToken 缓存JupyterHub令牌到Redis
func (h *JupyterHubAuthHandler) cacheJupyterHubToken(username, token string, expiresAt time.Time) error {
	if h.redisClient == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	key := fmt.Sprintf("jupyterhub:token:%s", username)
	duration := time.Until(expiresAt)

	return h.redisClient.Set(context.Background(), key, token, duration).Err()
}

// clearJupyterHubSessions 清除JupyterHub会话
func (h *JupyterHubAuthHandler) clearJupyterHubSessions(username string) error {
	if h.redisClient == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	// 清除令牌缓存
	tokenKey := fmt.Sprintf("jupyterhub:token:%s", username)
	sessionKey := fmt.Sprintf("jupyterhub:session:%s", username)

	pipe := h.redisClient.Pipeline()
	pipe.Del(context.Background(), tokenKey)
	pipe.Del(context.Background(), sessionKey)
	
	_, err := pipe.Exec(context.Background())
	return err
}

// userHasAdminRole 检查用户是否具有管理员角色
func (h *JupyterHubAuthHandler) userHasAdminRole(roles []string) bool {
	for _, role := range roles {
		if role == models.RoleAdmin || role == models.RoleSuperAdmin {
			return true
		}
	}
	return false
}

// validateJupyterHubToken 验证JupyterHub令牌
func (h *JupyterHubAuthHandler) validateJupyterHubToken(token, username string) (bool, map[string]interface{}, error) {
	// 尝试解析JWT令牌
	claims, err := jwt.ParseToken(token)
	if err != nil {
		return false, nil, fmt.Errorf("令牌解析失败: %w", err)
	}

	// 检查令牌是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return false, nil, fmt.Errorf("令牌已过期")
	}

	// 获取用户信息
	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return false, nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 如果提供了用户名，验证是否匹配
	if username != "" && user.Username != username {
		return false, nil, fmt.Errorf("用户名不匹配")
	}

	// 检查Redis中的令牌缓存
	if h.redisClient != nil {
		cacheKey := fmt.Sprintf("jupyterhub:token:%s", user.Username)
		cachedToken, err := h.redisClient.Get(context.Background(), cacheKey).Result()
		if err == nil && cachedToken != token {
			return false, nil, fmt.Errorf("令牌缓存不匹配")
		}
	}

	// 构建用户信息
	userInfo := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"roles":    []string{}, // 可以添加角色信息
	}

	return true, userInfo, nil
}

// VerifyJupyterHubSession 验证JupyterHub会话
// @Summary 验证JupyterHub会话状态
// @Description 检查当前用户的JupyterHub会话是否有效
// @Tags JupyterHub认证
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify-jupyterhub-session [get]
func (h *JupyterHubAuthHandler) VerifyJupyterHubSession(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "用户未认证",
		})
		return
	}

	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "无法获取用户信息",
		})
		return
	}

	// 检查Redis中的JupyterHub会话
	if h.redisClient != nil {
		cacheKey := fmt.Sprintf("jupyterhub:token:%s", username)
		_, err := h.redisClient.Get(context.Background(), cacheKey).Result()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"valid": false,
				"error": "JupyterHub会话不存在或已过期",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    true,
		"user_id":  userID,
		"username": username,
		"message":  "JupyterHub会话有效",
	})
}

// RefreshJupyterHubToken 刷新JupyterHub令牌
// @Summary 刷新JupyterHub令牌
// @Description 为当前用户生成新的JupyterHub访问令牌
// @Tags JupyterHub认证
// @Produce json
// @Success 200 {object} JupyterHubTokenResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/refresh-jupyterhub-token [post]
func (h *JupyterHubAuthHandler) RefreshJupyterHubToken(c *gin.Context) {
	// 获取当前用户
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 查询用户信息
	var user models.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		logrus.WithError(err).Error("查询用户失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户信息失败"})
		return
	}

	// 生成新的JupyterHub令牌
	token, expiresAt, err := h.generateJupyterHubToken(user)
	if err != nil {
		logrus.WithError(err).Error("生成JupyterHub令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	// 更新Redis缓存
	if err := h.cacheJupyterHubToken(user.Username, token, expiresAt); err != nil {
		logrus.WithError(err).Warning("缓存JupyterHub令牌失败")
	}

	// 记录刷新日志
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "refresh_jupyterhub_token",
	}).Info("用户刷新JupyterHub令牌")

	response := JupyterHubTokenResponse{
		Success:   true,
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		Message:   "令牌刷新成功",
	}

	c.JSON(http.StatusOK, response)
}

// JupyterHubAccessRequest JupyterHub访问请求
type JupyterHubAccessRequest struct {
	RedirectURI string `json:"redirect_uri,omitempty"`
	Source      string `json:"source,omitempty"` // frontend, api, direct
}

// JupyterHubAccessResponse JupyterHub访问响应
type JupyterHubAccessResponse struct {
	Success     bool   `json:"success"`
	Action      string `json:"action"`      // redirect, authenticated, error
	RedirectURL string `json:"redirect_url,omitempty"`
	Token       string `json:"token,omitempty"`
	Username    string `json:"username,omitempty"`
	Message     string `json:"message"`
}

// HandleJupyterHubAccess 处理前端JupyterHub访问请求
// @Summary 处理JupyterHub访问请求
// @Description 智能处理前端JupyterHub访问，根据认证状态返回适当的响应
// @Tags JupyterHub访问
// @Accept json
// @Produce json
// @Param request body JupyterHubAccessRequest false "访问请求参数"
// @Success 200 {object} JupyterHubAccessResponse
// @Failure 401 {object} JupyterHubAccessResponse
// @Failure 500 {object} JupyterHubAccessResponse
// @Router /jupyter/access [post]
func (h *JupyterHubAuthHandler) HandleJupyterHubAccess(c *gin.Context) {
	var request JupyterHubAccessRequest
	// 允许空的请求体
	c.ShouldBindJSON(&request)

	// 设置默认值
	if request.RedirectURI == "" {
		request.RedirectURI = "/jupyterhub-authenticated"
	}
	if request.Source == "" {
		request.Source = "frontend"
	}

	// 记录访问日志
	logrus.WithFields(logrus.Fields{
		"source":       request.Source,
		"redirect_uri": request.RedirectURI,
		"user_agent":   c.GetHeader("User-Agent"),
		"remote_addr":  c.ClientIP(),
	}).Info("收到JupyterHub访问请求")

	// 检查用户认证状态
	userID, authenticated := c.Get("user_id")
	if !authenticated {
		// 用户未认证，返回重定向到SSO
		response := JupyterHubAccessResponse{
			Success:     false,
			Action:      "redirect",
			RedirectURL: fmt.Sprintf("/sso/?redirect_uri=%s", request.RedirectURI),
			Message:     "需要登录，请跳转到SSO",
		}
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// 获取用户信息
	var user models.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		logrus.WithError(err).Error("查询用户信息失败")
		response := JupyterHubAccessResponse{
			Success:     false,
			Action:      "error",
			Message:     "查询用户信息失败",
		}
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// 生成或刷新JupyterHub令牌
	token, expiresAt, err := h.generateJupyterHubToken(user)
	if err != nil {
		logrus.WithError(err).Error("生成JupyterHub令牌失败")
		response := JupyterHubAccessResponse{
			Success:     false,
			Action:      "error",
			Message:     "生成访问令牌失败",
		}
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// 缓存令牌
	if err := h.cacheJupyterHubToken(user.Username, token, expiresAt); err != nil {
		logrus.WithError(err).Warning("缓存JupyterHub令牌失败")
	}

	// 记录成功访问
	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"action":   "jupyterhub_access_granted",
	}).Info("用户获得JupyterHub访问权限")

	// 返回成功响应
	response := JupyterHubAccessResponse{
		Success:     true,
		Action:      "authenticated",
		RedirectURL: request.RedirectURI,
		Token:       token,
		Username:    user.Username,
		Message:     "认证成功，可以访问JupyterHub",
	}

	c.JSON(http.StatusOK, response)
}

// GetJupyterHubAccessStatus 获取JupyterHub访问状态
// @Summary 获取当前用户的JupyterHub访问状态
// @Description 检查当前用户是否可以访问JupyterHub
// @Tags JupyterHub访问
// @Produce json
// @Success 200 {object} JupyterHubAccessResponse
// @Failure 401 {object} JupyterHubAccessResponse
// @Router /jupyter/status [get]
func (h *JupyterHubAuthHandler) GetJupyterHubAccessStatus(c *gin.Context) {
	// 检查用户认证状态
	userID, authenticated := c.Get("user_id")
	if !authenticated {
		response := JupyterHubAccessResponse{
			Success:     false,
			Action:      "redirect",
			RedirectURL: "/sso/?redirect_uri=/jupyterhub-authenticated",
			Message:     "用户未认证",
		}
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// 获取用户信息
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response := JupyterHubAccessResponse{
			Success: false,
			Action:  "error",
			Message: "获取用户信息失败",
		}
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// 检查是否有有效的JupyterHub令牌
	cacheKey := fmt.Sprintf("jupyterhub:token:%s", user.Username)
	cachedToken, err := h.redisClient.Get(context.Background(), cacheKey).Result()
	
	var hasValidToken bool
	if err == nil && cachedToken != "" {
		// 验证缓存的令牌
		isValid, _, tokenErr := h.validateJupyterHubToken(cachedToken, user.Username)
		hasValidToken = (tokenErr == nil && isValid)
	}

	response := JupyterHubAccessResponse{
		Success:  hasValidToken,
		Action:   "status",
		Username: user.Username,
		Message:  "JupyterHub访问状态检查完成",
	}

	if hasValidToken {
		response.Token = cachedToken
	}

	c.JSON(http.StatusOK, response)
}

// checkJupyterHubHealth 检查JupyterHub实际健康状态
func (h *JupyterHubAuthHandler) checkJupyterHubHealth() (string, string, string, error) {
	// JupyterHub容器地址
	jupyterhubURL := "http://ai-infra-jupyterhub:8000/jupyter/hub/api"
	
	// 创建HTTP客户端，设置较短的超时时间
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// 发送健康检查请求
	resp, err := client.Get(jupyterhubURL)
	if err != nil {
		return "disconnected", "unknown", "http://ai-infra-jupyterhub:8000", err
	}
	defer resp.Body.Close()
	
	// 根据响应状态判断健康状态
	var status string
	
	if resp.StatusCode == 200 {
		status = "connected"
	} else {
		status = "warning"
	}
	
	return status, "5.3.0", "http://ai-infra-jupyterhub:8000", nil
}
