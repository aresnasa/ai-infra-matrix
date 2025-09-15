package services

import (
	"errors"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	ldapService *LDAPService
	rbacService *RBACService
}

func NewUserService() *UserService {
	return &UserService{
		ldapService: NewLDAPService(database.DB),
		rbacService: NewRBACService(database.DB),
	}
}

// Register 用户注册
func (s *UserService) Register(req *models.RegisterRequest) (*models.User, error) {
	db := database.DB
	
	// 检查用户名是否已存在
	var existingUser models.User
	if err := db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("username or email already exists")
	}

	// LDAP验证：检查用户是否在LDAP中存在
	_, err := s.ldapService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		return nil, errors.New("LDAP验证失败: 用户不存在或密码错误")
	}

	// 如果需要审批，创建审批记录
	if req.RequiresApproval {
		approval := &models.RegistrationApproval{
			Username:     req.Username,
			Email:        req.Email,
			Department:   req.Department,
			RoleTemplate: req.RoleTemplate,
			Status:       "pending",
		}
		
		if err := db.Create(approval).Error; err != nil {
			return nil, errors.New("创建注册审批记录失败")
		}
		
		// 返回用户对象但标记为未激活
		user := &models.User{
			Username:     req.Username,
			Email:        req.Email,
			Password:     "", // 密码暂时不设置
			IsActive:     false,
			AuthSource:   "ldap",
			DashboardRole: req.Role,
		}
		return user, nil
	}

	// 直接注册流程
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:      req.Username,
		Email:         req.Email,
		Password:      string(hashedPassword),
		IsActive:      true,
		AuthSource:    "ldap", // 设置认证源为LDAP
		DashboardRole: req.Role,
		RoleTemplate:  req.RoleTemplate, // 设置角色模板
	}

	// 开始事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 如果指定了角色模板，为用户分配角色
	if req.RoleTemplate != "" {
		if err := s.rbacService.AssignRoleTemplateToUser(user.ID, req.RoleTemplate); err != nil {
			tx.Rollback()
			return nil, errors.New("分配角色模板失败: " + err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("提交事务失败")
	}

	return user, nil
}

// Login 用户登录
func (s *UserService) Login(req *models.LoginRequest) (*models.User, error) {
	db := database.DB
	
	var user models.User
	if err := db.Where("username = ? AND is_active = ?", req.Username, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found or inactive")
		}
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid password")
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLogin = &now
	db.Save(&user)

	return &user, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	db := database.DB
	
	var user models.User
	if err := db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUsers 获取用户列表（管理员功能）
func (s *UserService) GetUsers(page, pageSize int) ([]models.User, int64, error) {
	db := database.DB
	
	var users []models.User
	var total int64

	offset := (page - 1) * pageSize

	if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(userID uint, updates map[string]interface{}) error {
	db := database.DB
	
	// 如果要更新密码，需要加密
	if password, ok := updates["password"].(string); ok {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updates["password"] = string(hashedPassword)
	}

	return db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(userID uint) error {
	db := database.DB
	return db.Delete(&models.User{}, userID).Error
}

// ChangePassword 修改用户密码
func (s *UserService) ChangePassword(userID uint, req *models.ChangePasswordRequest) error {
	db := database.DB
	
	// 获取用户当前信息
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}
	
	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码不正确")
	}
	
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	// 更新密码
	return db.Model(&user).Update("password", string(hashedPassword)).Error
}

// UpdateUserProfile 更新用户个人信息
func (s *UserService) UpdateUserProfile(userID uint, req *models.UpdateUserProfileRequest) (*models.User, error) {
	db := database.DB
	
	// 获取用户
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	
	// 检查用户名和邮箱是否被其他用户使用
	if req.Username != "" && req.Username != user.Username {
		var existingUser models.User
		if err := db.Where("username = ? AND id != ?", req.Username, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("用户名已被使用")
		}
		user.Username = req.Username
	}
	
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := db.Where("email = ? AND id != ?", req.Email, userID).First(&existingUser).Error; err == nil {
			return nil, errors.New("邮箱已被使用")
		}
		user.Email = req.Email
	}
	
	// 保存更新
	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}
	
	return &user, nil
}

// AdminResetPassword 管理员重置用户密码
func (s *UserService) AdminResetPassword(userID uint, req *models.AdminResetPasswordRequest) error {
	db := database.DB
	
	// 获取用户
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}
	
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	// 更新密码
	return db.Model(&user).Update("password", string(hashedPassword)).Error
}

// AdminUpdateUserGroups 管理员更新用户的用户组
func (s *UserService) AdminUpdateUserGroups(userID uint, req *models.UpdateUserGroupsRequest) error {
	db := database.DB
	
	// 开始事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 删除用户的所有用户组关联
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserGroupMembership{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// 添加新的用户组关联
	for _, groupID := range req.UserGroupIDs {
		membership := models.UserGroupMembership{
			UserID:      userID,
			UserGroupID: groupID,
		}
		if err := tx.Create(&membership).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	
	return tx.Commit().Error
}

// GetUserWithDetails 获取用户详细信息（包含角色和用户组）
func (s *UserService) GetUserWithDetails(userID uint) (*models.User, error) {
	db := database.DB
	
	var user models.User
	if err := db.Preload("Roles").Preload("UserGroups").Preload("Projects").First(&user, userID).Error; err != nil {
		return nil, err
	}
	
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	db := database.DB
	
	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	
	return &user, nil
}

// CreateUserDirectly 直接创建用户（用于LDAP用户）
func (s *UserService) CreateUserDirectly(user *models.User) error {
	db := database.DB
	return db.Create(user).Error
}

// GetPendingApprovals 获取待审批的注册申请
func (s *UserService) GetPendingApprovals() ([]models.RegistrationApproval, error) {
	db := database.DB
	var approvals []models.RegistrationApproval
	err := db.Where("status = ?", "pending").Preload("User").Find(&approvals).Error
	return approvals, err
}

// ApproveRegistration 审批注册申请
func (s *UserService) ApproveRegistration(approvalID uint, adminID uint) error {
	db := database.DB
	
	var approval models.RegistrationApproval
	if err := db.First(&approval, approvalID).Error; err != nil {
		return errors.New("审批记录不存在")
	}
	
	if approval.Status != "pending" {
		return errors.New("该申请已被处理")
	}
	
	// 开始事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	now := time.Now()
	approval.Status = "approved"
	approval.ApprovedBy = &adminID
	approval.ApprovedAt = &now
	
	if err := tx.Save(&approval).Error; err != nil {
		tx.Rollback()
		return errors.New("更新审批状态失败")
	}
	
	// 创建用户
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("temp_password"), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	user := &models.User{
		Username:      approval.Username,
		Email:         approval.Email,
		Password:      string(hashedPassword),
		IsActive:      true,
		AuthSource:    "ldap",
		DashboardRole: "user", // 默认角色
	}
	
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return errors.New("创建用户失败")
	}
	
	// 如果指定了角色模板，为用户分配角色
	if approval.RoleTemplate != "" {
		if err := s.rbacService.AssignRoleTemplateToUser(user.ID, approval.RoleTemplate); err != nil {
			tx.Rollback()
			return errors.New("分配角色模板失败: " + err.Error())
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return errors.New("提交事务失败")
	}
	
	return nil
}

// RejectRegistration 拒绝注册申请
func (s *UserService) RejectRegistration(approvalID uint, adminID uint, reason string) error {
	db := database.DB
	
	var approval models.RegistrationApproval
	if err := db.First(&approval, approvalID).Error; err != nil {
		return errors.New("审批记录不存在")
	}
	
	if approval.Status != "pending" {
		return errors.New("该申请已被处理")
	}
	
	now := time.Now()
	approval.Status = "rejected"
	approval.RejectedBy = &adminID
	approval.RejectedAt = &now
	approval.RejectReason = reason
	
	return db.Save(&approval).Error
}
