package services

import (
	"errors"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// Register 用户注册
func (s *UserService) Register(req *models.RegisterRequest) (*models.User, error) {
	db := database.DB
	
	// 检查用户名是否已存在
	var existingUser models.User
	if err := db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("username or email already exists")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   string(hashedPassword),
		IsActive:   true,
		AuthSource: "local", // 设置认证源为本地
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
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
