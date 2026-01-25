package services

import (
	"fmt"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RBACService struct {
	db *gorm.DB
}

func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{db: db}
}

// CheckPermission 检查用户是否有权限执行特定操作
func (s *RBACService) CheckPermission(userID uint, resource, verb, scope, namespace string) bool {
	// 如果没有指定命名空间，使用默认命名空间
	if namespace == "" {
		namespace = "default"
	}

	// 如果没有指定scope，使用通配符
	if scope == "" {
		scope = "*"
	}

	logrus.WithFields(logrus.Fields{
		"user_id":   userID,
		"resource":  resource,
		"verb":      verb,
		"scope":     scope,
		"namespace": namespace,
	}).Info("Checking permission")

	// 1. 通过用户直接分配的角色检查权限
	var userRoles []models.Role
	err := s.db.Preload("Permissions").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&userRoles).Error
	if err != nil {
		logrus.WithError(err).Error("Failed to get user roles")
		return false
	}

	logrus.WithFields(logrus.Fields{
		"user_id":    userID,
		"role_count": len(userRoles),
	}).Info("Found user roles")

	// 检查用户直接角色权限
	if s.checkRolePermissions(userRoles, resource, verb, scope) {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"source":  "direct_role",
		}).Info("Permission granted via direct role")
		return true
	}

	// 2. 通过用户组的角色检查权限
	var userGroupRoles []models.Role
	err = s.db.Preload("Permissions").
		Joins("JOIN user_group_roles ON user_group_roles.role_id = roles.id").
		Joins("JOIN user_group_memberships ON user_group_memberships.user_group_id = user_group_roles.user_group_id").
		Where("user_group_memberships.user_id = ?", userID).
		Find(&userGroupRoles).Error
	if err != nil {
		logrus.WithError(err).Error("Failed to get user group roles")
		return false
	}

	// 检查用户组角色权限
	if s.checkRolePermissions(userGroupRoles, resource, verb, scope) {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"source":  "group_role",
		}).Info("Permission granted via group role")
		return true
	}

	// 3. 通过RoleBinding检查权限
	if s.checkRoleBindingPermissions(userID, resource, verb, scope, namespace) {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"source":  "role_binding",
		}).Info("Permission granted via role binding")
		return true
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  userID,
		"resource": resource,
		"verb":     verb,
		"scope":    scope,
	}).Info("Permission denied")

	return false
}

// checkRolePermissions 检查角色列表中是否包含所需权限
func (s *RBACService) checkRolePermissions(roles []models.Role, resource, verb, scope string) bool {
	for _, role := range roles {
		for _, permission := range role.Permissions {
			if s.matchPermission(permission, resource, verb, scope) {
				return true
			}
		}
	}
	return false
}

// checkRoleBindingPermissions 通过RoleBinding检查权限
func (s *RBACService) checkRoleBindingPermissions(userID uint, resource, verb, scope, namespace string) bool {
	// 检查用户的RoleBinding
	var userRoleBindings []models.RoleBinding
	err := s.db.Preload("Role.Permissions").
		Where("subject_type = ? AND subject_id = ? AND namespace = ?", "user", userID, namespace).
		Find(&userRoleBindings).Error
	if err != nil {
		return false
	}

	for _, binding := range userRoleBindings {
		for _, permission := range binding.Role.Permissions {
			if s.matchPermission(permission, resource, verb, scope) {
				return true
			}
		}
	}

	// 检查用户组的RoleBinding
	var userGroupIDs []uint
	err = s.db.Model(&models.UserGroupMembership{}).
		Where("user_id = ?", userID).
		Pluck("user_group_id", &userGroupIDs).Error
	if err != nil {
		return false
	}

	if len(userGroupIDs) > 0 {
		var groupRoleBindings []models.RoleBinding
		err = s.db.Preload("Role.Permissions").
			Where("subject_type = ? AND subject_id IN ? AND namespace = ?", "group", userGroupIDs, namespace).
			Find(&groupRoleBindings).Error
		if err != nil {
			return false
		}

		for _, binding := range groupRoleBindings {
			for _, permission := range binding.Role.Permissions {
				if s.matchPermission(permission, resource, verb, scope) {
					return true
				}
			}
		}
	}

	return false
}

// matchPermission 检查权限是否匹配
func (s *RBACService) matchPermission(permission models.Permission, resource, verb, scope string) bool {
	// 检查资源匹配
	if permission.Resource != "*" && permission.Resource != resource {
		return false
	}

	// 检查动词匹配
	if permission.Verb != "*" && permission.Verb != verb {
		return false
	}

	// 检查作用域匹配
	if permission.Scope != "*" && permission.Scope != scope {
		return false
	}

	return true
}

// IsAdmin 检查用户是否是管理员
func (s *RBACService) IsAdmin(userID uint) bool {
	return s.CheckPermission(userID, "*", "*", "*", "*")
}

// HasRoleInProject 检查用户在特定项目中是否有特定角色
func (s *RBACService) HasRoleInProject(userID uint, projectID uint, roleName string) bool {
	projectScope := fmt.Sprintf("project:%d", projectID)
	return s.CheckPermission(userID, "projects", "access", projectScope, "default")
}

// GetUserPermissions 获取用户的所有权限
func (s *RBACService) GetUserPermissions(userID uint) ([]models.Permission, error) {
	var permissions []models.Permission
	permissionMap := make(map[uint]models.Permission)

	// 获取用户直接分配的角色权限
	var userRoles []models.Role
	err := s.db.Preload("Permissions").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&userRoles).Error
	if err != nil {
		return nil, err
	}

	for _, role := range userRoles {
		for _, permission := range role.Permissions {
			permissionMap[permission.ID] = permission
		}
	}

	// 获取用户组角色权限
	var userGroupRoles []models.Role
	err = s.db.Preload("Permissions").
		Joins("JOIN user_group_roles ON user_group_roles.role_id = roles.id").
		Joins("JOIN user_group_memberships ON user_group_memberships.user_group_id = user_group_roles.user_group_id").
		Where("user_group_memberships.user_id = ?", userID).
		Find(&userGroupRoles).Error
	if err != nil {
		return nil, err
	}

	for _, role := range userGroupRoles {
		for _, permission := range role.Permissions {
			permissionMap[permission.ID] = permission
		}
	}

	// 转换为切片
	for _, permission := range permissionMap {
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// GetUserRoles 获取用户的所有角色
func (s *RBACService) GetUserRoles(userID uint) ([]models.Role, error) {
	var roles []models.Role
	roleMap := make(map[uint]models.Role)

	// 获取用户直接分配的角色
	var userRoles []models.Role
	err := s.db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&userRoles).Error
	if err != nil {
		return nil, err
	}

	for _, role := range userRoles {
		roleMap[role.ID] = role
	}

	// 获取用户组角色
	var userGroupRoles []models.Role
	err = s.db.Joins("JOIN user_group_roles ON user_group_roles.role_id = roles.id").
		Joins("JOIN user_group_memberships ON user_group_memberships.user_group_id = user_group_roles.user_group_id").
		Where("user_group_memberships.user_id = ?", userID).
		Find(&userGroupRoles).Error
	if err != nil {
		return nil, err
	}

	for _, role := range userGroupRoles {
		roleMap[role.ID] = role
	}

	// 转换为切片
	for _, role := range roleMap {
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRoleToUser 为用户分配角色
func (s *RBACService) AssignRoleToUser(userID, roleID uint) error {
	userRole := models.UserRole{
		UserID: userID,
		RoleID: roleID,
	}

	// 检查是否已经存在
	var existing models.UserRole
	err := s.db.Where("user_id = ? AND role_id = ?", userID, roleID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("用户已经拥有该角色")
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	return s.db.Create(&userRole).Error
}

// RemoveRoleFromUser 移除用户角色
func (s *RBACService) RemoveRoleFromUser(userID, roleID uint) error {
	return s.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error
}

// AssignRoleToUserGroup 为用户组分配角色
func (s *RBACService) AssignRoleToUserGroup(userGroupID, roleID uint) error {
	userGroupRole := models.UserGroupRole{
		UserGroupID: userGroupID,
		RoleID:      roleID,
	}

	// 检查是否已经存在
	var existing models.UserGroupRole
	err := s.db.Where("user_group_id = ? AND role_id = ?", userGroupID, roleID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("用户组已经拥有该角色")
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	return s.db.Create(&userGroupRole).Error
}

// AddUserToGroup 将用户添加到用户组
func (s *RBACService) AddUserToGroup(userID, userGroupID uint) error {
	membership := models.UserGroupMembership{
		UserID:      userID,
		UserGroupID: userGroupID,
	}

	// 检查是否已经存在
	var existing models.UserGroupMembership
	err := s.db.Where("user_id = ? AND user_group_id = ?", userID, userGroupID).First(&existing).Error
	if err == nil {
		return fmt.Errorf("用户已经是该用户组的成员")
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	return s.db.Create(&membership).Error
}

// RemoveUserFromGroup 从用户组中移除用户
func (s *RBACService) RemoveUserFromGroup(userID, userGroupID uint) error {
	return s.db.Where("user_id = ? AND user_group_id = ?", userID, userGroupID).Delete(&models.UserGroupMembership{}).Error
}

// CreateRole 创建角色
func (s *RBACService) CreateRole(name, description string, permissionIDs []uint, isSystem bool) (*models.Role, error) {
	role := models.Role{
		Name:        name,
		Description: description,
		IsSystem:    isSystem,
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建角色
	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 分配权限
	if len(permissionIDs) > 0 {
		for _, permissionID := range permissionIDs {
			rolePermission := models.RolePermission{
				RoleID:       role.ID,
				PermissionID: permissionID,
			}
			if err := tx.Create(&rolePermission).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &role, nil
}

// CreateUserGroup 创建用户组
func (s *RBACService) CreateUserGroup(name, description string) (*models.UserGroup, error) {
	userGroup := models.UserGroup{
		Name:        name,
		Description: description,
	}

	if err := s.db.Create(&userGroup).Error; err != nil {
		return nil, err
	}

	return &userGroup, nil
}

// CreatePermission 创建权限
func (s *RBACService) CreatePermission(resource, verb, scope, description string) (*models.Permission, error) {
	permission := models.Permission{
		Resource:    resource,
		Verb:        verb,
		Scope:       scope,
		Description: description,
	}

	if err := s.db.Create(&permission).Error; err != nil {
		return nil, err
	}

	return &permission, nil
}

// 默认角色模板定义（用于初始化）
type DefaultRoleTemplate struct {
	Name          string
	DisplayName   string // 中文显示名称
	DisplayNameEN string // 英文显示名称
	Description   string // 中文描述
	DescriptionEN string // 英文描述
	Color         string
	Icon          string
	Priority      int
	Permissions   []struct {
		Resource string
		Verb     string
		Scope    string
	}
	IsSystem bool
}

// 预定义角色模板（仅用于初始化时创建默认模板）
var defaultRoleTemplates = []DefaultRoleTemplate{
	{
		Name:          "admin",
		DisplayName:   "系统管理员",
		DisplayNameEN: "System Administrator",
		Description:   "系统管理员 - 拥有所有权限",
		DescriptionEN: "System Administrator - Has all permissions",
		Color:         "red",
		Icon:          "crown",
		Priority:      100,
		Permissions: []struct {
			Resource string
			Verb     string
			Scope    string
		}{
			{Resource: "*", Verb: "*", Scope: "*"},
		},
		IsSystem: true,
	},
	{
		Name:          "data-developer",
		DisplayName:   "数据开发",
		DisplayNameEN: "Data Developer",
		Description:   "数据开发人员 - 主要关注数据处理和分析",
		DescriptionEN: "Data Developer - Focuses on data processing and analysis",
		Color:         "blue",
		Icon:          "database",
		Priority:      50,
		Permissions: []struct {
			Resource string
			Verb     string
			Scope    string
		}{
			{Resource: "projects", Verb: "create", Scope: "*"},
			{Resource: "projects", Verb: "read", Scope: "*"},
			{Resource: "projects", Verb: "update", Scope: "*"},
			{Resource: "projects", Verb: "delete", Scope: "own"},
			{Resource: "hosts", Verb: "read", Scope: "*"},
			{Resource: "variables", Verb: "create", Scope: "*"},
			{Resource: "variables", Verb: "read", Scope: "*"},
			{Resource: "variables", Verb: "update", Scope: "*"},
			{Resource: "variables", Verb: "delete", Scope: "own"},
		},
		IsSystem: true,
	},
	{
		Name:          "model-developer",
		DisplayName:   "模型开发",
		DisplayNameEN: "Model Developer",
		Description:   "模型开发人员 - 主要关注Jupyter环境",
		DescriptionEN: "Model Developer - Focuses on Jupyter environment",
		Color:         "purple",
		Icon:          "experiment",
		Priority:      50,
		Permissions: []struct {
			Resource string
			Verb     string
			Scope    string
		}{
			{Resource: "jupyterhub", Verb: "create", Scope: "*"},
			{Resource: "jupyterhub", Verb: "read", Scope: "*"},
			{Resource: "jupyterhub", Verb: "update", Scope: "own"},
			{Resource: "jupyterhub", Verb: "delete", Scope: "own"},
			{Resource: "projects", Verb: "read", Scope: "*"},
			{Resource: "projects", Verb: "create", Scope: "own"},
		},
		IsSystem: true,
	},
	{
		Name:          "sre",
		DisplayName:   "SRE工程师",
		DisplayNameEN: "SRE Engineer",
		Description:   "SRE工程师 - 关注SaltStack、Ansible、K8s、监控和日志",
		DescriptionEN: "SRE Engineer - Focuses on SaltStack, Ansible, K8s, Monitoring and Logs",
		Color:         "orange",
		Icon:          "tool",
		Priority:      60,
		Permissions: []struct {
			Resource string
			Verb     string
			Scope    string
		}{
			// SaltStack管理
			{Resource: "saltstack", Verb: "create", Scope: "*"},
			{Resource: "saltstack", Verb: "read", Scope: "*"},
			{Resource: "saltstack", Verb: "update", Scope: "*"},
			{Resource: "saltstack", Verb: "delete", Scope: "*"},
			{Resource: "saltstack", Verb: "list", Scope: "*"},
			// Ansible自动化
			{Resource: "ansible", Verb: "create", Scope: "*"},
			{Resource: "ansible", Verb: "read", Scope: "*"},
			{Resource: "ansible", Verb: "update", Scope: "*"},
			{Resource: "ansible", Verb: "delete", Scope: "*"},
			{Resource: "ansible", Verb: "list", Scope: "*"},
			// Kubernetes集群管理
			{Resource: "kubernetes", Verb: "create", Scope: "*"},
			{Resource: "kubernetes", Verb: "read", Scope: "*"},
			{Resource: "kubernetes", Verb: "update", Scope: "*"},
			{Resource: "kubernetes", Verb: "delete", Scope: "*"},
			{Resource: "kubernetes", Verb: "list", Scope: "*"},
			// 主机管理
			{Resource: "hosts", Verb: "read", Scope: "*"},
			{Resource: "hosts", Verb: "create", Scope: "*"},
			{Resource: "hosts", Verb: "update", Scope: "*"},
			{Resource: "hosts", Verb: "delete", Scope: "*"},
			{Resource: "hosts", Verb: "list", Scope: "*"},
			// 系统监控 (Nightingale)
			{Resource: "nightingale", Verb: "create", Scope: "*"},
			{Resource: "nightingale", Verb: "read", Scope: "*"},
			{Resource: "nightingale", Verb: "update", Scope: "*"},
			{Resource: "nightingale", Verb: "delete", Scope: "*"},
			{Resource: "nightingale", Verb: "list", Scope: "*"},
			// 日志管理
			{Resource: "audit-logs", Verb: "read", Scope: "*"},
			{Resource: "audit-logs", Verb: "list", Scope: "*"},
			// 项目管理 (SRE需要访问项目页面进行Ansible项目管理)
			{Resource: "projects", Verb: "create", Scope: "*"},
			{Resource: "projects", Verb: "read", Scope: "*"},
			{Resource: "projects", Verb: "update", Scope: "*"},
			{Resource: "projects", Verb: "delete", Scope: "own"},
			{Resource: "projects", Verb: "list", Scope: "*"},
			// 变量管理 (Ansible变量)
			{Resource: "variables", Verb: "create", Scope: "*"},
			{Resource: "variables", Verb: "read", Scope: "*"},
			{Resource: "variables", Verb: "update", Scope: "*"},
			{Resource: "variables", Verb: "delete", Scope: "own"},
			{Resource: "variables", Verb: "list", Scope: "*"},
			// 任务管理 (Ansible任务)
			{Resource: "tasks", Verb: "create", Scope: "*"},
			{Resource: "tasks", Verb: "read", Scope: "*"},
			{Resource: "tasks", Verb: "update", Scope: "*"},
			{Resource: "tasks", Verb: "delete", Scope: "own"},
			{Resource: "tasks", Verb: "list", Scope: "*"},
			// Playbook管理
			{Resource: "playbooks", Verb: "create", Scope: "*"},
			{Resource: "playbooks", Verb: "read", Scope: "*"},
			{Resource: "playbooks", Verb: "update", Scope: "*"},
			{Resource: "playbooks", Verb: "delete", Scope: "own"},
			{Resource: "playbooks", Verb: "list", Scope: "*"},
		},
		IsSystem: true,
	},
	{
		Name:          "engineer",
		DisplayName:   "工程研发",
		DisplayNameEN: "Software Engineer",
		Description:   "工程研发人员 - 主要关注K8s环境",
		DescriptionEN: "Software Engineer - Focuses on K8s environment",
		Color:         "green",
		Icon:          "code",
		Priority:      40,
		Permissions: []struct {
			Resource string
			Verb     string
			Scope    string
		}{
			{Resource: "kubernetes", Verb: "create", Scope: "*"},
			{Resource: "kubernetes", Verb: "read", Scope: "*"},
			{Resource: "kubernetes", Verb: "update", Scope: "*"},
			{Resource: "kubernetes", Verb: "delete", Scope: "*"},
			{Resource: "projects", Verb: "create", Scope: "*"},
			{Resource: "projects", Verb: "read", Scope: "*"},
			{Resource: "projects", Verb: "update", Scope: "own"},
			{Resource: "projects", Verb: "delete", Scope: "own"},
			{Resource: "hosts", Verb: "read", Scope: "*"},
		},
		IsSystem: true,
	},
}

// InitializeDefaultRBAC 初始化默认的RBAC权限和角色
func (s *RBACService) InitializeDefaultRBAC() error {
	// 创建基础权限
	defaultPermissions := []models.Permission{
		{Resource: "*", Verb: "*", Scope: "*", Description: "超级管理员权限"},
		// 项目权限
		{Resource: "projects", Verb: "create", Scope: "*", Description: "创建项目权限"},
		{Resource: "projects", Verb: "read", Scope: "*", Description: "查看项目权限"},
		{Resource: "projects", Verb: "update", Scope: "*", Description: "更新项目权限"},
		{Resource: "projects", Verb: "delete", Scope: "*", Description: "删除项目权限"},
		{Resource: "projects", Verb: "list", Scope: "*", Description: "列出项目权限"},
		// 主机权限
		{Resource: "hosts", Verb: "create", Scope: "*", Description: "创建主机权限"},
		{Resource: "hosts", Verb: "read", Scope: "*", Description: "查看主机权限"},
		{Resource: "hosts", Verb: "update", Scope: "*", Description: "更新主机权限"},
		{Resource: "hosts", Verb: "delete", Scope: "*", Description: "删除主机权限"},
		{Resource: "hosts", Verb: "list", Scope: "*", Description: "列出主机权限"},
		// 变量权限
		{Resource: "variables", Verb: "create", Scope: "*", Description: "创建变量权限"},
		{Resource: "variables", Verb: "read", Scope: "*", Description: "查看变量权限"},
		{Resource: "variables", Verb: "update", Scope: "*", Description: "更新变量权限"},
		{Resource: "variables", Verb: "delete", Scope: "*", Description: "删除变量权限"},
		{Resource: "variables", Verb: "list", Scope: "*", Description: "列出变量权限"},
		// 任务权限
		{Resource: "tasks", Verb: "create", Scope: "*", Description: "创建任务权限"},
		{Resource: "tasks", Verb: "read", Scope: "*", Description: "查看任务权限"},
		{Resource: "tasks", Verb: "update", Scope: "*", Description: "更新任务权限"},
		{Resource: "tasks", Verb: "delete", Scope: "*", Description: "删除任务权限"},
		{Resource: "tasks", Verb: "list", Scope: "*", Description: "列出任务权限"},
		// Playbook权限
		{Resource: "playbooks", Verb: "create", Scope: "*", Description: "创建Playbook权限"},
		{Resource: "playbooks", Verb: "read", Scope: "*", Description: "查看Playbook权限"},
		{Resource: "playbooks", Verb: "update", Scope: "*", Description: "更新Playbook权限"},
		{Resource: "playbooks", Verb: "delete", Scope: "*", Description: "删除Playbook权限"},
		{Resource: "playbooks", Verb: "list", Scope: "*", Description: "列出Playbook权限"},
		// 用户权限
		{Resource: "users", Verb: "create", Scope: "*", Description: "创建用户权限"},
		{Resource: "users", Verb: "read", Scope: "*", Description: "查看用户权限"},
		{Resource: "users", Verb: "update", Scope: "*", Description: "更新用户权限"},
		{Resource: "users", Verb: "delete", Scope: "*", Description: "删除用户权限"},
		{Resource: "users", Verb: "list", Scope: "*", Description: "列出用户权限"},
		// 角色和组权限
		{Resource: "roles", Verb: "*", Scope: "*", Description: "角色管理权限"},
		{Resource: "groups", Verb: "*", Scope: "*", Description: "用户组管理权限"},
	}

	for _, permission := range defaultPermissions {
		var existing models.Permission
		err := s.db.Where("resource = ? AND verb = ? AND scope = ?",
			permission.Resource, permission.Verb, permission.Scope).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(&permission).Error; err != nil {
				return fmt.Errorf("创建权限失败: %v", err)
			}
		}
	}

	// 创建默认角色
	var superAdminRole models.Role
	err := s.db.Where("name = ?", "super-admin").First(&superAdminRole).Error
	if err == gorm.ErrRecordNotFound {
		// 获取超级管理员权限ID
		var superAdminPermission models.Permission
		if err := s.db.Where("resource = ? AND verb = ? AND scope = ?", "*", "*", "*").First(&superAdminPermission).Error; err != nil {
			return fmt.Errorf("找不到超级管理员权限: %v", err)
		}

		_, err := s.CreateRole("super-admin", "超级管理员角色", []uint{superAdminPermission.ID}, true)
		if err != nil {
			return fmt.Errorf("创建超级管理员角色失败: %v", err)
		}
	}

	// 创建admin角色（如果不存在），与super-admin拥有相同权限
	var adminRole models.Role
	err = s.db.Where("name = ?", "admin").First(&adminRole).Error
	if err == gorm.ErrRecordNotFound {
		// 获取超级管理员权限ID
		var superAdminPermission models.Permission
		if err := s.db.Where("resource = ? AND verb = ? AND scope = ?", "*", "*", "*").First(&superAdminPermission).Error; err != nil {
			return fmt.Errorf("找不到超级管理员权限: %v", err)
		}

		_, err := s.CreateRole("admin", "管理员角色", []uint{superAdminPermission.ID}, true)
		if err != nil {
			return fmt.Errorf("创建管理员角色失败: %v", err)
		}
	}

	var userRole models.Role
	err = s.db.Where("name = ?", "user").First(&userRole).Error
	if err == gorm.ErrRecordNotFound {
		// 获取普通用户权限ID (包括所有必要的资源权限)
		var userPermissions []models.Permission
		resources := []string{"projects", "hosts", "variables", "tasks", "playbooks"}
		verbs := []string{"create", "read", "update", "delete", "list"}

		if err := s.db.Where("resource IN ? AND verb IN ?", resources, verbs).Find(&userPermissions).Error; err != nil {
			return fmt.Errorf("找不到用户权限: %v", err)
		}

		var permissionIDs []uint
		for _, p := range userPermissions {
			permissionIDs = append(permissionIDs, p.ID)
		}

		_, err := s.CreateRole("user", "普通用户角色", permissionIDs, true)
		if err != nil {
			return fmt.Errorf("创建普通用户角色失败: %v", err)
		}
	} else if err == nil {
		// 如果用户角色已存在，检查并更新权限
		var currentPermissions []models.Permission
		if err := s.db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
			Where("role_permissions.role_id = ?", userRole.ID).Find(&currentPermissions).Error; err != nil {
			return fmt.Errorf("获取当前用户角色权限失败: %v", err)
		}

		// 检查是否缺少hosts, variables, tasks, playbooks权限
		resources := []string{"hosts", "variables", "tasks", "playbooks"}
		verbs := []string{"create", "read", "update", "delete", "list"}

		existingPermissionMap := make(map[string]bool)
		for _, p := range currentPermissions {
			key := fmt.Sprintf("%s:%s", p.Resource, p.Verb)
			existingPermissionMap[key] = true
		}

		var missingPermissions []models.Permission
		for _, resource := range resources {
			for _, verb := range verbs {
				key := fmt.Sprintf("%s:%s", resource, verb)
				if !existingPermissionMap[key] {
					var permission models.Permission
					if err := s.db.Where("resource = ? AND verb = ?", resource, verb).First(&permission).Error; err == nil {
						missingPermissions = append(missingPermissions, permission)
					}
				}
			}
		}

		// 添加缺失的权限
		for _, permission := range missingPermissions {
			rolePermission := models.RolePermission{
				RoleID:       userRole.ID,
				PermissionID: permission.ID,
			}
			if err := s.db.Create(&rolePermission).Error; err != nil {
				return fmt.Errorf("添加用户角色权限失败: %v", err)
			}
		}
	}

	// 初始化默认角色模板到数据库
	if err := s.initializeDefaultRoleTemplates(); err != nil {
		return fmt.Errorf("初始化默认角色模板失败: %v", err)
	}

	// 创建预定义角色（从角色模板同步）
	if err := s.syncRoleTemplatesAsRoles(); err != nil {
		return fmt.Errorf("同步角色模板为角色失败: %v", err)
	}

	return nil
}

// initializeDefaultRoleTemplates 初始化默认角色模板到数据库
func (s *RBACService) initializeDefaultRoleTemplates() error {
	for _, template := range defaultRoleTemplates {
		var existing models.RoleTemplate
		err := s.db.Where("name = ?", template.Name).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			// 创建角色模板
			roleTemplate := models.RoleTemplate{
				Name:          template.Name,
				DisplayName:   template.DisplayName,
				DisplayNameEN: template.DisplayNameEN,
				Description:   template.Description,
				DescriptionEN: template.DescriptionEN,
				IsSystem:      template.IsSystem,
				IsActive:      true,
				Priority:      template.Priority,
				Color:         template.Color,
				Icon:          template.Icon,
			}
			if err := s.db.Create(&roleTemplate).Error; err != nil {
				return fmt.Errorf("创建角色模板失败: %v", err)
			}

			// 创建模板权限
			for _, perm := range template.Permissions {
				templatePerm := models.RoleTemplatePermission{
					RoleTemplateID: roleTemplate.ID,
					Resource:       perm.Resource,
					Verb:           perm.Verb,
					Scope:          perm.Scope,
				}
				if err := s.db.Create(&templatePerm).Error; err != nil {
					return fmt.Errorf("创建角色模板权限失败: %v", err)
				}
			}
		} else if err == nil {
			// 更新现有角色模板的英文字段（如果为空）
			if existing.DisplayNameEN == "" || existing.DescriptionEN == "" {
				existing.DisplayNameEN = template.DisplayNameEN
				existing.DescriptionEN = template.DescriptionEN
				if err := s.db.Save(&existing).Error; err != nil {
					return fmt.Errorf("更新角色模板英文字段失败: %v", err)
				}
			}
		}
	}
	return nil
}

// syncRoleTemplatesAsRoles 将角色模板同步为实际角色
func (s *RBACService) syncRoleTemplatesAsRoles() error {
	var templates []models.RoleTemplate
	if err := s.db.Preload("Permissions").Where("is_active = ?", true).Find(&templates).Error; err != nil {
		return err
	}

	for _, template := range templates {
		var role models.Role
		err := s.db.Where("name = ?", template.Name).First(&role).Error
		if err == gorm.ErrRecordNotFound {
			// 创建角色
			_, err := s.CreateRole(template.Name, template.Description, nil, template.IsSystem)
			if err != nil {
				return fmt.Errorf("创建角色失败: %v", err)
			}
			// 重新获取角色
			s.db.Where("name = ?", template.Name).First(&role)
		} else if err == nil {
			// 更新角色描述
			role.Description = template.Description
			s.db.Save(&role)
		} else {
			return err
		}

		// 分配权限
		for _, perm := range template.Permissions {
			var permission models.Permission
			scope := perm.Scope
			if scope == "" {
				scope = "*"
			}
			err := s.db.Where("resource = ? AND verb = ? AND scope = ?", perm.Resource, perm.Verb, scope).First(&permission).Error
			if err == gorm.ErrRecordNotFound {
				// 权限不存在，创建权限
				permission = models.Permission{
					Resource:    perm.Resource,
					Verb:        perm.Verb,
					Scope:       scope,
					Description: fmt.Sprintf("%s %s %s", perm.Verb, perm.Resource, scope),
				}
				if err := s.db.Create(&permission).Error; err != nil {
					continue
				}
			} else if err != nil {
				continue
			}

			var rolePermission models.RolePermission
			err = s.db.Where("role_id = ? AND permission_id = ?", role.ID, permission.ID).First(&rolePermission).Error
			if err == gorm.ErrRecordNotFound {
				// 添加权限到角色
				rolePermission = models.RolePermission{
					RoleID:       role.ID,
					PermissionID: permission.ID,
				}
				s.db.Create(&rolePermission)
			}
		}
	}
	return nil
}

// RevokeRoleFromUser 撤销用户角色
func (s *RBACService) RevokeRoleFromUser(userID, roleID uint) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %v", err)
	}

	// 检查角色是否存在
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("角色不存在")
		}
		return fmt.Errorf("查询角色失败: %v", err)
	}

	// 删除用户角色关联
	result := s.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{})
	if result.Error != nil {
		return fmt.Errorf("撤销用户角色失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("用户未分配该角色")
	}

	return nil
}

// RevokeRoleFromUserGroup 撤销用户组角色
func (s *RBACService) RevokeRoleFromUserGroup(groupID, roleID uint) error {
	// 检查用户组是否存在
	var group models.UserGroup
	if err := s.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户组不存在")
		}
		return fmt.Errorf("查询用户组失败: %v", err)
	}

	// 检查角色是否存在
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("角色不存在")
		}
		return fmt.Errorf("查询角色失败: %v", err)
	}

	// 删除用户组角色关联
	result := s.db.Where("user_group_id = ? AND role_id = ?", groupID, roleID).Delete(&models.UserGroupRole{})
	if result.Error != nil {
		return fmt.Errorf("撤销用户组角色失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("用户组未分配该角色")
	}

	return nil
}

// CreateRoleTemplateFromRequest 从请求创建新的角色模板（管理员使用）
func (s *RBACService) CreateRoleTemplateFromRequest(req models.CreateRoleTemplateRequest) (*models.RoleTemplate, error) {
	// 检查模板名称是否已存在
	var existing models.RoleTemplate
	if err := s.db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("角色模板名称已存在: %s", req.Name)
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建角色模板
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	color := "blue"
	if req.Color != "" {
		color = req.Color
	}

	roleTemplate := models.RoleTemplate{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false, // 用户创建的模板不是系统模板
		IsActive:    isActive,
		Priority:    req.Priority,
		Color:       color,
		Icon:        req.Icon,
	}

	if err := tx.Create(&roleTemplate).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建角色模板失败: %v", err)
	}

	// 创建模板权限
	for _, perm := range req.Permissions {
		scope := perm.Scope
		if scope == "" {
			scope = "*"
		}
		templatePerm := models.RoleTemplatePermission{
			RoleTemplateID: roleTemplate.ID,
			Resource:       perm.Resource,
			Verb:           perm.Verb,
			Scope:          scope,
		}
		if err := tx.Create(&templatePerm).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建角色模板权限失败: %v", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	// 重新加载完整模板
	if err := s.db.Preload("Permissions").First(&roleTemplate, roleTemplate.ID).Error; err != nil {
		return nil, err
	}

	return &roleTemplate, nil
}

// UpdateRoleTemplate 更新角色模板
func (s *RBACService) UpdateRoleTemplate(id uint, req models.UpdateRoleTemplateRequest) (*models.RoleTemplate, error) {
	var roleTemplate models.RoleTemplate
	if err := s.db.First(&roleTemplate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色模板不存在")
		}
		return nil, err
	}

	// 系统模板名称不能修改
	if roleTemplate.IsSystem && req.Name != "" && req.Name != roleTemplate.Name {
		return nil, fmt.Errorf("系统角色模板名称不能修改")
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新基本信息
	if req.Name != "" {
		roleTemplate.Name = req.Name
	}
	if req.DisplayName != "" {
		roleTemplate.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		roleTemplate.Description = req.Description
	}
	if req.IsActive != nil {
		roleTemplate.IsActive = *req.IsActive
	}
	if req.Priority != 0 {
		roleTemplate.Priority = req.Priority
	}
	if req.Color != "" {
		roleTemplate.Color = req.Color
	}
	if req.Icon != "" {
		roleTemplate.Icon = req.Icon
	}

	if err := tx.Save(&roleTemplate).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新角色模板失败: %v", err)
	}

	// 如果提供了权限，更新权限
	if req.Permissions != nil {
		// 删除旧权限
		if err := tx.Where("role_template_id = ?", id).Delete(&models.RoleTemplatePermission{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("删除旧权限失败: %v", err)
		}

		// 添加新权限
		for _, perm := range req.Permissions {
			scope := perm.Scope
			if scope == "" {
				scope = "*"
			}
			templatePerm := models.RoleTemplatePermission{
				RoleTemplateID: id,
				Resource:       perm.Resource,
				Verb:           perm.Verb,
				Scope:          scope,
			}
			if err := tx.Create(&templatePerm).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("创建角色模板权限失败: %v", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	// 重新加载完整模板
	if err := s.db.Preload("Permissions").First(&roleTemplate, id).Error; err != nil {
		return nil, err
	}

	return &roleTemplate, nil
}

// DeleteRoleTemplate 删除角色模板
func (s *RBACService) DeleteRoleTemplate(id uint) error {
	var roleTemplate models.RoleTemplate
	if err := s.db.First(&roleTemplate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("角色模板不存在")
		}
		return err
	}

	// 系统模板不能删除
	if roleTemplate.IsSystem {
		return fmt.Errorf("系统角色模板不能删除")
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除模板权限
	if err := tx.Where("role_template_id = ?", id).Delete(&models.RoleTemplatePermission{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除模板权限失败: %v", err)
	}

	// 删除模板
	if err := tx.Delete(&roleTemplate).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除角色模板失败: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// GetRoleTemplateByID 根据ID获取角色模板
func (s *RBACService) GetRoleTemplateByID(id uint) (*models.RoleTemplate, error) {
	var roleTemplate models.RoleTemplate
	if err := s.db.Preload("Permissions").First(&roleTemplate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色模板不存在")
		}
		return nil, err
	}
	return &roleTemplate, nil
}

// GetRoleTemplateByName 根据名称获取角色模板
func (s *RBACService) GetRoleTemplateByName(name string) (*models.RoleTemplate, error) {
	var roleTemplate models.RoleTemplate
	if err := s.db.Preload("Permissions").Where("name = ?", name).First(&roleTemplate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色模板不存在: %s", name)
		}
		return nil, err
	}
	return &roleTemplate, nil
}

// ListRoleTemplates 列出所有角色模板
func (s *RBACService) ListRoleTemplates(activeOnly bool) ([]models.RoleTemplate, error) {
	var templates []models.RoleTemplate
	query := s.db.Preload("Permissions").Order("priority DESC, name ASC")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetAvailableResources 获取可配置的资源列表
func (s *RBACService) GetAvailableResources() []string {
	return []string{
		"*",
		"projects",
		"hosts",
		"variables",
		"tasks",
		"playbooks",
		"users",
		"roles",
		"groups",
		"saltstack",
		"ansible",
		"kubernetes",
		"jupyterhub",
		"gitea",
		"object_storage",
		"monitoring",
		"role_templates",
	}
}

// GetAvailableVerbs 获取可配置的操作列表
func (s *RBACService) GetAvailableVerbs() []string {
	return []string{
		"*",
		"create",
		"read",
		"update",
		"delete",
		"list",
		"execute",
		"admin",
	}
}

// CreateRoleFromTemplate 根据模板名称创建角色（保留向后兼容）
func (s *RBACService) CreateRoleFromTemplate(templateName string) (*models.Role, error) {
	// 从数据库获取模板
	template, err := s.GetRoleTemplateByName(templateName)
	if err != nil {
		return nil, err
	}

	// 检查角色是否已存在
	var existingRole models.Role
	if err := s.db.Where("name = ?", template.Name).First(&existingRole).Error; err == nil {
		return &existingRole, nil // 角色已存在，直接返回
	}

	// 创建角色
	role := &models.Role{
		Name:        template.Name,
		Description: template.Description,
		IsSystem:    template.IsSystem,
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(role).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建角色失败: %v", err)
	}

	// 为角色添加权限
	for _, perm := range template.Permissions {
		scope := perm.Scope
		if scope == "" {
			scope = "*"
		}
		var permission models.Permission
		err := tx.Where("resource = ? AND verb = ? AND scope = ?", perm.Resource, perm.Verb, scope).First(&permission).Error
		if err == gorm.ErrRecordNotFound {
			// 权限不存在，创建权限
			permission = models.Permission{
				Resource:    perm.Resource,
				Verb:        perm.Verb,
				Scope:       scope,
				Description: fmt.Sprintf("%s %s %s", perm.Verb, perm.Resource, scope),
			}
			if err := tx.Create(&permission).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("创建权限失败: %v", err)
			}
		} else if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("查询权限失败: %v", err)
		}

		// 创建角色权限关联
		rolePermission := models.RolePermission{
			RoleID:       role.ID,
			PermissionID: permission.ID,
		}
		if err := tx.Create(&rolePermission).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建角色权限关联失败: %v", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	return role, nil
}

// AssignRoleTemplateToUser 为用户分配角色模板
func (s *RBACService) AssignRoleTemplateToUser(userID uint, templateName string) error {
	// 从数据库获取模板并创建角色
	role, err := s.CreateRoleFromTemplate(templateName)
	if err != nil {
		return fmt.Errorf("创建角色模板失败: %v", err)
	}

	// 检查用户角色关联是否已存在
	var userRole models.UserRole
	err = s.db.Where("user_id = ? AND role_id = ?", userID, role.ID).First(&userRole).Error
	if err == nil {
		return nil // 已存在关联
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询用户角色关联失败: %v", err)
	}

	// 创建用户角色关联
	userRole = models.UserRole{
		UserID: userID,
		RoleID: role.ID,
	}
	if err := s.db.Create(&userRole).Error; err != nil {
		return fmt.Errorf("创建用户角色关联失败: %v", err)
	}

	return nil
}

// GrantUserModulePermissions 为用户授予模块权限
func (s *RBACService) GrantUserModulePermissions(userID uint, modules []string, verbs []string) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 为每个模块和操作创建权限
	for _, module := range modules {
		for _, verb := range verbs {
			// 检查权限是否存在，不存在则创建
			var permission models.Permission
			err := tx.Where("resource = ? AND verb = ? AND scope = ?", module, verb, "*").First(&permission).Error
			if err == gorm.ErrRecordNotFound {
				permission = models.Permission{
					Resource:    module,
					Verb:        verb,
					Scope:       "*",
					Description: fmt.Sprintf("%s %s 权限", module, verb),
				}
				if err := tx.Create(&permission).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("创建权限失败: %v", err)
				}
			} else if err != nil {
				tx.Rollback()
				return fmt.Errorf("查询权限失败: %v", err)
			}

			// 检查用户是否已有此权限授予
			var existingGrant models.PermissionGrant
			err = tx.Where("user_id = ? AND module = ? AND verb = ? AND is_active = ?", userID, module, verb, true).First(&existingGrant).Error
			if err == gorm.ErrRecordNotFound {
				// 为用户创建权限授予
				grant := models.PermissionGrant{
					UserID:    userID,
					Module:    module,
					Resource:  module,
					Verb:      verb,
					Scope:     "*",
					GrantType: "manual",
					GrantedBy: userID, // 可以改为管理员ID
					Reason:    "管理员审批授予",
					IsActive:  true,
				}
				if err := tx.Create(&grant).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("创建权限授予失败: %v", err)
				}
			} else if err != nil {
				tx.Rollback()
				return fmt.Errorf("查询权限授予失败: %v", err)
			}
			// 如果已存在则跳过
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}
