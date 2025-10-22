package services

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/go-ldap/ldap/v3"
	"gorm.io/gorm"
)

// SyncStatus 同步状态
type SyncStatus struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"` // running, completed, failed
	Progress  float64       `json:"progress"`
	Message   string        `json:"message"`
	Result    *SyncResult   `json:"result,omitempty"`
	Error     string        `json:"error,omitempty"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration"`
}

type LDAPSyncService struct {
	db           *gorm.DB
	ldapService  *LDAPService
	userService  *UserService
	rbacService  *RBACService
	syncStatuses map[string]*SyncStatus
	syncHistory  []*SyncStatus
	mutex        sync.RWMutex
}

func NewLDAPSyncService(db *gorm.DB, ldapService *LDAPService, userService *UserService, rbacService *RBACService) *LDAPSyncService {
	return &LDAPSyncService{
		db:           db,
		ldapService:  ldapService,
		userService:  userService,
		rbacService:  rbacService,
		syncStatuses: make(map[string]*SyncStatus),
		syncHistory:  make([]*SyncStatus, 0),
	}
}

// SyncResult 同步结果
type SyncResult struct {
	UsersCreated  int           `json:"users_created"`
	UsersUpdated  int           `json:"users_updated"`
	GroupsCreated int           `json:"groups_created"`
	GroupsUpdated int           `json:"groups_updated"`
	RolesAssigned int           `json:"roles_assigned"`
	Errors        []string      `json:"errors"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	TotalUsers    int           `json:"total_users"`
	TotalGroups   int           `json:"total_groups"`
}

// SyncLDAPUsersAndGroups 同步LDAP用户和组到PostgreSQL数据库
func (s *LDAPSyncService) SyncLDAPUsersAndGroups() (*SyncResult, error) {
	result := &SyncResult{
		StartTime: time.Now(),
		Errors:    []string{},
	}

	// 检查LDAP是否启用
	config, err := s.ldapService.GetConfig()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("获取LDAP配置失败: %v", err))
		return result, err
	}

	if !config.IsEnabled {
		result.Errors = append(result.Errors, "LDAP认证未启用")
		return result, fmt.Errorf("LDAP认证未启用")
	}

	log.Printf("开始LDAP同步，服务器: %s:%d", config.Server, config.Port)

	// 创建LDAP连接 - 使用本地函数而不是调用ldapService的私有方法
	conn, err := s.createLDAPConnection(config.Server, config.Port, config.UseSSL, config.SkipVerify)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("连接LDAP服务器失败: %v", err))
		return result, err
	}
	defer conn.Close()

	// 绑定管理员账户
	err = conn.Bind(config.BindDN, config.BindPassword)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("LDAP管理员绑定失败: %v", err))
		return result, err
	}

	// 同步用户组
	if err := s.syncUserGroups(conn, config, result); err != nil {
		log.Printf("同步用户组时出错: %v", err)
		result.Errors = append(result.Errors, fmt.Sprintf("同步用户组失败: %v", err))
	}

	// 同步用户
	if err := s.syncUsers(conn, config, result); err != nil {
		log.Printf("同步用户时出错: %v", err)
		result.Errors = append(result.Errors, fmt.Sprintf("同步用户失败: %v", err))
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	log.Printf("LDAP同步完成: 用户创建=%d, 用户更新=%d, 组创建=%d, 组更新=%d, 角色分配=%d, 错误数=%d, 耗时=%v",
		result.UsersCreated, result.UsersUpdated, result.GroupsCreated, result.GroupsUpdated,
		result.RolesAssigned, len(result.Errors), result.Duration)

	return result, nil
}

// syncUserGroups 同步LDAP用户组
func (s *LDAPSyncService) syncUserGroups(conn *ldap.Conn, config *models.LDAPConfig, result *SyncResult) error {
	// 由于模型中没有GroupFilter字段，跳过组同步
	log.Printf("组同步功能已禁用")
	return nil
}

// syncUsers 同步LDAP用户
func (s *LDAPSyncService) syncUsers(conn *ldap.Conn, config *models.LDAPConfig, result *SyncResult) error {
	log.Printf("开始同步用户...")

	// 构造用户搜索过滤器
	userFilter := config.UserFilter
	if userFilter == "" {
		userFilter = "(objectClass=person)"
	}

	// 搜索所有用户
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		userFilter,
		[]string{"dn", config.UsernameAttr, config.NameAttr, config.EmailAttr},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("搜索LDAP用户失败: %v", err)
	}

	result.TotalUsers = len(sr.Entries)
	log.Printf("找到 %d 个LDAP用户", result.TotalUsers)

	for _, entry := range sr.Entries {
		// 获取用户名
		username := entry.GetAttributeValue(config.UsernameAttr)
		if username == "" && config.UsernameAttr != "uid" {
			username = entry.GetAttributeValue("uid")
		}
		if username == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("用户 %s 缺少用户名属性", entry.DN))
			continue
		}

		// 获取显示名称
		displayName := entry.GetAttributeValue(config.NameAttr)
		if displayName == "" {
			displayName = username
		}

		// 获取邮箱
		email := entry.GetAttributeValue(config.EmailAttr)
		if email == "" {
			email = fmt.Sprintf("%s@ldap.local", username)
		}

		// 获取用户组
		userGroups, err := s.getUserGroupsFromLDAP(conn, config, entry.DN)
		if err != nil {
			log.Printf("获取用户 %s 的组信息失败: %v", username, err)
		}

		// 同步用户
		if err := s.syncSingleUser(username, displayName, email, entry.DN, userGroups, config, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("同步用户 %s 失败: %v", username, err))
		}
	}

	return nil
}

// syncSingleUser 同步单个用户
func (s *LDAPSyncService) syncSingleUser(username, displayName, email, userDN string, userGroups []string, config *models.LDAPConfig, result *SyncResult) error {
	// 检查用户是否已存在
	existingUser, err := s.userService.GetUserByUsername(username)

	if err == gorm.ErrRecordNotFound {
		// 创建新用户
		newUser := &models.User{
			Username:   username,
			Email:      email,
			Password:   "", // LDAP用户不设置本地密码
			IsActive:   true,
			AuthSource: "ldap", // 设置认证源为LDAP
			LDAPDn:     userDN, // 设置LDAP DN
		}

		if err := s.userService.CreateUserDirectly(newUser); err != nil {
			return fmt.Errorf("创建用户失败: %v", err)
		}

		result.UsersCreated++
		log.Printf("创建用户: %s (%s)", username, email)

		// 分配用户组
		if err := s.assignUserToGroups(newUser.ID, userGroups, result); err != nil {
			log.Printf("分配用户 %s 到组失败: %v", username, err)
		}

		// 检查并分配管理员角色（简化版本）
		if s.isUserAdmin(userGroups, "") {
			if err := s.assignAdminRole(newUser.ID, result); err != nil {
				log.Printf("分配管理员角色给用户 %s 失败: %v", username, err)
			}
		}

	} else if err == nil {
		// 更新现有用户
		updates := map[string]interface{}{
			"email":       email,
			"is_active":   true,
			"auth_source": "ldap", // 确保认证源设置为LDAP
			"ldap_dn":     userDN, // 更新LDAP DN
			"updated_at":  time.Now(),
		}

		if err := s.userService.UpdateUser(existingUser.ID, updates); err != nil {
			return fmt.Errorf("更新用户失败: %v", err)
		}

		result.UsersUpdated++
		log.Printf("更新用户: %s (%s)", username, email)

		// 重新分配用户组
		if err := s.assignUserToGroups(existingUser.ID, userGroups, result); err != nil {
			log.Printf("重新分配用户 %s 到组失败: %v", username, err)
		}

		// 检查并分配/移除管理员角色
		// 暂时跳过管理员角色分配，因为 LDAPConfig 中没有 AdminGroups 字段
		// 可以通过其他方式（如特定的 LDAP 组或用户属性）来判断管理员权限
		// if s.isUserAdmin(userGroups, "") {
		// 	if err := s.assignAdminRole(existingUser.ID, result); err != nil {
		// 		log.Printf("分配管理员角色给用户 %s 失败: %v", username, err)
		// 	}
		// }

	} else {
		return fmt.Errorf("查询用户失败: %v", err)
	}

	return nil
}

// getUserGroupsFromLDAP 从LDAP获取用户组（简化版本）
func (s *LDAPSyncService) getUserGroupsFromLDAP(conn *ldap.Conn, config *models.LDAPConfig, userDN string) ([]string, error) {
	// 由于模型中没有GroupFilter字段，返回空组列表
	return []string{}, nil
}

// assignUserToGroups 将用户分配到组
func (s *LDAPSyncService) assignUserToGroups(userID uint, groupNames []string, result *SyncResult) error {
	// 先清除用户现有的组关系
	if err := s.db.Where("user_id = ?", userID).Delete(&models.UserGroupMembership{}).Error; err != nil {
		return fmt.Errorf("清除用户组关系失败: %v", err)
	}

	// 为每个组分配用户
	for _, groupName := range groupNames {
		var userGroup models.UserGroup
		if err := s.db.Where("name = ?", groupName).First(&userGroup).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("用户组 %s 不存在，跳过分配", groupName)
				continue
			}
			return fmt.Errorf("查询用户组 %s 失败: %v", groupName, err)
		}

		// 创建用户组关系
		membership := models.UserGroupMembership{
			UserID:      userID,
			UserGroupID: userGroup.ID,
		}

		if err := s.db.Create(&membership).Error; err != nil {
			log.Printf("分配用户到组 %s 失败: %v", groupName, err)
			continue
		}

		log.Printf("用户 %d 已分配到组 %s", userID, groupName)
	}

	return nil
}

// isUserAdmin 检查用户是否为管理员
func (s *LDAPSyncService) isUserAdmin(userGroups []string, adminGroupsConfig string) bool {
	if adminGroupsConfig == "" {
		return false
	}

	adminGroups := strings.Split(adminGroupsConfig, ",")
	for _, adminGroup := range adminGroups {
		adminGroup = strings.TrimSpace(adminGroup)
		for _, userGroup := range userGroups {
			if strings.EqualFold(userGroup, adminGroup) {
				return true
			}
		}
	}

	return false
}

// assignAdminRole 分配管理员角色
func (s *LDAPSyncService) assignAdminRole(userID uint, result *SyncResult) error {
	// 首先尝试查找super-admin角色，如果找不到则使用admin角色
	var adminRole models.Role
	if err := s.db.Where("name = ?", models.RoleSuperAdmin).First(&adminRole).Error; err != nil {
		// 如果找不到super-admin，尝试查找admin角色
		if err := s.db.Where("name = ?", models.RoleAdmin).First(&adminRole).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("管理员角色不存在，跳过角色分配")
				return nil
			}
			return fmt.Errorf("查询管理员角色失败: %v", err)
		}
	}

	// 检查用户是否已有管理员角色
	var existingRole models.UserRole
	err := s.db.Where("user_id = ? AND role_id = ?", userID, adminRole.ID).First(&existingRole).Error
	if err == nil {
		// 角色已存在
		return nil
	}

	// 创建用户角色关系
	userRole := models.UserRole{
		UserID: userID,
		RoleID: adminRole.ID,
	}

	if err := s.db.Create(&userRole).Error; err != nil {
		return fmt.Errorf("分配管理员角色失败: %v", err)
	}

	result.RolesAssigned++
	log.Printf("已为用户 %d 分配管理员角色", userID)
	return nil
}

// TriggerSync 触发LDAP同步（异步）
func (s *LDAPSyncService) TriggerSync() string {
	syncID := s.generateSyncID()

	// 创建同步状态
	status := &SyncStatus{
		ID:        syncID,
		Status:    "running",
		Progress:  0.0,
		Message:   "正在启动LDAP同步...",
		StartTime: time.Now(),
	}

	s.mutex.Lock()
	s.syncStatuses[syncID] = status
	s.mutex.Unlock()

	go func() {
		s.performSync(syncID)
	}()

	return syncID
}

// generateSyncID 生成同步任务ID
func (s *LDAPSyncService) generateSyncID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// performSync 执行同步操作
func (s *LDAPSyncService) performSync(syncID string) {
	s.mutex.Lock()
	status := s.syncStatuses[syncID]
	s.mutex.Unlock()

	defer func() {
		endTime := time.Now()
		s.mutex.Lock()
		if status != nil {
			status.EndTime = &endTime
			status.Duration = endTime.Sub(status.StartTime)
			// 将完成的同步添加到历史记录
			s.syncHistory = append([]*SyncStatus{status}, s.syncHistory...)
			// 保持历史记录数量在合理范围内
			if len(s.syncHistory) > 50 {
				s.syncHistory = s.syncHistory[:50]
			}
		}
		s.mutex.Unlock()
	}()

	result, err := s.SyncLDAPUsersAndGroups()

	s.mutex.Lock()
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		status.Message = "同步失败: " + err.Error()
		log.Printf("LDAP同步失败 [%s]: %v", syncID, err)
	} else {
		status.Status = "completed"
		status.Progress = 100.0
		status.Message = "同步完成"
		status.Result = result
		log.Printf("LDAP同步完成 [%s]: %+v", syncID, result)
	}
	s.mutex.Unlock()
}

// GetSyncStatus 获取同步状态
func (s *LDAPSyncService) GetSyncStatus(syncID string) *SyncStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if status, exists := s.syncStatuses[syncID]; exists {
		return status
	}

	// 检查历史记录
	for _, status := range s.syncHistory {
		if status.ID == syncID {
			return status
		}
	}

	return nil
}

// GetSyncHistory 获取同步历史
func (s *LDAPSyncService) GetSyncHistory(limit int) []*SyncStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if limit <= 0 || limit > len(s.syncHistory) {
		limit = len(s.syncHistory)
	}

	history := make([]*SyncStatus, limit)
	copy(history, s.syncHistory[:limit])
	return history
}

// createLDAPConnection 创建LDAP连接
func (s *LDAPSyncService) createLDAPConnection(server string, port int, useSSL, skipVerify bool) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if useSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: skipVerify,
		}
		conn, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", server, port), tlsConfig)
	} else {
		conn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", server, port))
	}

	if err != nil {
		return nil, fmt.Errorf("连接LDAP服务器失败: %v", err)
	}

	return conn, nil
}
