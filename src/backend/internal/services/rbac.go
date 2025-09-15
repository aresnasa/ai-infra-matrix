package services

import (
	"fmt"

	"gorm.io/gorm"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
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

// 角色模板定义
type RoleTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions []models.Permission `json:"permissions"`
	IsSystem    bool   `json:"is_system"`
}

// 预定义角色模板
var RoleTemplates = map[string]RoleTemplate{
	"model-developer": {
		Name:        "model-developer",
		Description: "模型开发人员 - 主要关注Jupyter环境",
		Permissions: []models.Permission{
			{Resource: "jupyterhub", Verb: "create", Scope: "*"},
			{Resource: "jupyterhub", Verb: "read", Scope: "*"},
			{Resource: "jupyterhub", Verb: "update", Scope: "own"},
			{Resource: "jupyterhub", Verb: "delete", Scope: "own"},
			{Resource: "projects", Verb: "read", Scope: "*"},
			{Resource: "projects", Verb: "create", Scope: "own"},
		},
		IsSystem: true,
	},
	"sre": {
		Name:        "sre",
		Description: "SRE工程师 - 关注SaltStack、Ansible和K8s",
		Permissions: []models.Permission{
			{Resource: "saltstack", Verb: "create", Scope: "*"},
			{Resource: "saltstack", Verb: "read", Scope: "*"},
			{Resource: "saltstack", Verb: "update", Scope: "*"},
			{Resource: "saltstack", Verb: "delete", Scope: "*"},
			{Resource: "ansible", Verb: "create", Scope: "*"},
			{Resource: "ansible", Verb: "read", Scope: "*"},
			{Resource: "ansible", Verb: "update", Scope: "*"},
			{Resource: "ansible", Verb: "delete", Scope: "*"},
			{Resource: "kubernetes", Verb: "create", Scope: "*"},
			{Resource: "kubernetes", Verb: "read", Scope: "*"},
			{Resource: "kubernetes", Verb: "update", Scope: "*"},
			{Resource: "kubernetes", Verb: "delete", Scope: "*"},
			{Resource: "hosts", Verb: "read", Scope: "*"},
		},
		IsSystem: true,
	},
	"engineer": {
		Name:        "engineer",
		Description: "工程研发人员 - 主要关注K8s环境",
		Permissions: []models.Permission{
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

	// 创建预定义角色模板
	for _, template := range RoleTemplates {
		var role models.Role
		err := s.db.Where("name = ?", template.Name).First(&role).Error
		if err == gorm.ErrRecordNotFound {
			// 创建角色
			_, err := s.CreateRole(template.Name, template.Description, nil, template.IsSystem)
			if err != nil {
				return fmt.Errorf("创建角色失败: %v", err)
			}
		} else if err == nil {
			// 更新角色描述
			role.Description = template.Description
			s.db.Save(&role)
		} else {
			return err
		}

		// 分配权限
		for _, permission := range template.Permissions {
			var perm models.Permission
			err := s.db.Where("resource = ? AND verb = ? AND scope = ?", permission.Resource, permission.Verb, permission.Scope).First(&perm).Error
			if err == nil {
				var rolePermission models.RolePermission
				err = s.db.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).First(&rolePermission).Error
				if err != nil && err == gorm.ErrRecordNotFound {
					// 添加权限到角色
					rolePermission = models.RolePermission{
						RoleID:       role.ID,
						PermissionID: perm.ID,
					}
					s.db.Create(&rolePermission)
				}
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

// CreateRoleTemplate 创建角色模板
func (s *RBACService) CreateRoleTemplate(templateName string) (*models.Role, error) {
	template, exists := RoleTemplates[templateName]
	if !exists {
		return nil, fmt.Errorf("角色模板不存在: %s", templateName)
	}

	// 检查角色是否已存在
	var existingRole models.Role
	err := s.db.Where("name = ?", template.Name).First(&existingRole).Error
	if err == nil {
		return &existingRole, nil // 角色已存在，直接返回
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询角色失败: %v", err)
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
		var permission models.Permission
		err := tx.Where("resource = ? AND verb = ? AND scope = ?", perm.Resource, perm.Verb, perm.Scope).First(&permission).Error
		if err == gorm.ErrRecordNotFound {
			// 权限不存在，创建权限
			permission = models.Permission{
				Resource:    perm.Resource,
				Verb:        perm.Verb,
				Scope:       perm.Scope,
				Description: fmt.Sprintf("%s %s %s", perm.Verb, perm.Resource, perm.Scope),
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

// GetRoleTemplates 获取所有角色模板
func (s *RBACService) GetRoleTemplates() map[string]RoleTemplate {
	return RoleTemplates
}

// AssignRoleTemplateToUser 为用户分配角色模板
func (s *RBACService) AssignRoleTemplateToUser(userID uint, templateName string) error {
	// 创建或获取角色模板
	role, err := s.CreateRoleTemplate(templateName)
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
