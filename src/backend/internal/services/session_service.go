package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/sirupsen/logrus"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

// UserSession Redis中存储的用户会话信息
type UserSession struct {
	UserID       uint      `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	Permissions  []string  `json:"permissions"`
	LastActivity time.Time `json:"last_activity"`
	LoginTime    time.Time `json:"login_time"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
}

// CreateSession 创建用户会话
func (s *SessionService) CreateSession(user *models.User, token string, ipAddress, userAgent string) error {
	// 获取用户角色和权限
	var userWithRoles models.User
	if err := database.DB.Preload("Roles").Preload("Roles.Permissions").First(&userWithRoles, user.ID).Error; err != nil {
		return fmt.Errorf("failed to load user roles: %w", err)
	}

	// 构建角色和权限列表
	var roles []string
	var permissions []string
	permissionSet := make(map[string]bool) // 去重

	for _, role := range userWithRoles.Roles {
		roles = append(roles, role.Name)
		for _, permission := range role.Permissions {
			permKey := fmt.Sprintf("%s:%s:%s", permission.Resource, permission.Verb, permission.Scope)
			if !permissionSet[permKey] {
				permissions = append(permissions, permKey)
				permissionSet[permKey] = true
			}
		}
	}

	session := UserSession{
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Roles:        roles,
		Permissions:  permissions,
		LastActivity: time.Now(),
		LoginTime:    time.Now(),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	// 序列化为JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// 存储到Redis，24小时过期
	sessionKey := fmt.Sprintf("session:%s", token)
	ctx := context.Background()

	if err := cache.RDB.Set(ctx, sessionKey, sessionData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store session in Redis: %w", err)
	}

	// 更新用户最后登录时间
	now := time.Now()
	user.LastLogin = &now
	if err := database.DB.Save(user).Error; err != nil {
		logrus.WithError(err).Warn("Failed to update user last login time")
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"ip":       ipAddress,
	}).Info("User session created")

	return nil
}

// GetSession 获取用户会话
func (s *SessionService) GetSession(token string) (*UserSession, error) {
	sessionKey := fmt.Sprintf("session:%s", token)
	ctx := context.Background()

	sessionData, err := cache.RDB.Get(ctx, sessionKey).Result()
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	var session UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// UpdateSession 更新会话活动时间
func (s *SessionService) UpdateSession(token string) error {
	session, err := s.GetSession(token)
	if err != nil {
		return err
	}

	session.LastActivity = time.Now()

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionKey := fmt.Sprintf("session:%s", token)
	ctx := context.Background()

	// 重新设置过期时间为24小时
	if err := cache.RDB.Set(ctx, sessionKey, sessionData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// UpdateActivity 更新会话活动时间
func (s *SessionService) UpdateActivity(token string) error {
	sessionKey := fmt.Sprintf("session:%s", token)
	ctx := context.Background()

	// 获取现有会话数据
	sessionData, err := cache.RDB.Get(ctx, sessionKey).Result()
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	var session UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return fmt.Errorf("failed to parse session data: %w", err)
	}

	// 更新最后活动时间
	session.LastActivity = time.Now()

	// 保存更新后的会话数据
	updatedData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// 重新设置Redis中的数据，保持原有的TTL
	ttl, err := cache.RDB.TTL(ctx, sessionKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get session TTL: %w", err)
	}

	if err := cache.RDB.Set(ctx, sessionKey, updatedData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession 删除会话
func (s *SessionService) DeleteSession(token string) error {
	sessionKey := fmt.Sprintf("session:%s", token)
	ctx := context.Background()

	if err := cache.RDB.Del(ctx, sessionKey).Err(); err != nil {
		logrus.WithError(err).Warn("Failed to delete session from Redis")
	}

	return nil
}

// GetAllActiveSessions 获取所有活跃会话（管理员功能）
func (s *SessionService) GetAllActiveSessions() ([]UserSession, error) {
	ctx := context.Background()

	// 获取所有session键
	keys, err := cache.RDB.Keys(ctx, "session:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session keys: %w", err)
	}

	var sessions []UserSession
	for _, key := range keys {
		sessionData, err := cache.RDB.Get(ctx, key).Result()
		if err != nil {
			continue // 跳过已过期或无效的会话
		}

		var session UserSession
		if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
			continue // 跳过无效的会话数据
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SyncSessionsToDatabase 定期同步Redis会话数据到PostgreSQL
func (s *SessionService) SyncSessionsToDatabase() error {
	sessions, err := s.GetAllActiveSessions()
	if err != nil {
		return fmt.Errorf("failed to get active sessions: %w", err)
	}

	// 批量更新用户的最后活动时间
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, session := range sessions {
		if err := tx.Model(&models.User{}).
			Where("id = ?", session.UserID).
			Update("last_login", session.LastActivity).Error; err != nil {
			logrus.WithError(err).WithField("user_id", session.UserID).
				Warn("Failed to update user last activity")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit session sync: %w", err)
	}

	logrus.WithField("session_count", len(sessions)).Info("Sessions synced to database")
	return nil
}

// CleanupExpiredSessions 清理过期的会话
func (s *SessionService) CleanupExpiredSessions() error {
	ctx := context.Background()

	keys, err := cache.RDB.Keys(ctx, "session:*").Result()
	if err != nil {
		return fmt.Errorf("failed to get session keys: %w", err)
	}

	cleanedCount := 0
	for _, key := range keys {
		// 检查TTL
		ttl, err := cache.RDB.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		// 如果TTL为-1表示没有过期时间，或者TTL为-2表示key不存在
		if ttl == -2 {
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		logrus.WithField("cleaned_sessions", cleanedCount).Info("Expired sessions cleaned up")
	}

	return nil
}

// GetUserSessionsByUserID 获取指定用户的所有活跃会话
func (s *SessionService) GetUserSessionsByUserID(userID uint) ([]UserSession, error) {
	allSessions, err := s.GetAllActiveSessions()
	if err != nil {
		return nil, err
	}

	var userSessions []UserSession
	for _, session := range allSessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}

	return userSessions, nil
}

// RevokeUserSessions 撤销指定用户的所有会话
func (s *SessionService) RevokeUserSessions(userID uint) error {
	userSessions, err := s.GetUserSessionsByUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	ctx := context.Background()
	revokedCount := 0

	for range userSessions {
		// 这里我们需要从会话数据中推导出token，这在实际实现中可能需要调整
		// 一个更好的做法是在Redis中维护user_id -> tokens的映射
		keys, err := cache.RDB.Keys(ctx, "session:*").Result()
		if err != nil {
			continue
		}

		for _, key := range keys {
			sessionData, err := cache.RDB.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var sessionCheck UserSession
			if err := json.Unmarshal([]byte(sessionData), &sessionCheck); err != nil {
				continue
			}

			if sessionCheck.UserID == userID {
				if err := cache.RDB.Del(ctx, key).Err(); err == nil {
					revokedCount++
				}
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"user_id":       userID,
		"revoked_count": revokedCount,
	}).Info("User sessions revoked")

	return nil
}
