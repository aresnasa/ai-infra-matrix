package services

import (
	"fmt"
	"log"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/go-ldap/ldap/v3"
	"gorm.io/gorm"
)

type LDAPService struct {
	db *gorm.DB
}

func NewLDAPService(db *gorm.DB) *LDAPService {
	return &LDAPService{db: db}
}

// GetConfig 获取LDAP配置
func (ls *LDAPService) GetConfig() (*models.LDAPConfig, error) {
	var config models.LDAPConfig
	err := ls.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			return &models.LDAPConfig{
				Port:             389,
				UseSSL:           false,
				UsernameAttr:     "uid",
				EmailAttr:        "mail",
				DisplayNameAttr:  "cn",
				GroupNameAttr:    "cn",
				MemberAttr:       "member",
				SyncEnabled:      false,
				AutoCreateUser:   true,
				AutoCreateGroup:  true,
				DefaultRole:      "user",
				SyncStatus:       "never",
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// UpdateConfig 更新LDAP配置
func (ls *LDAPService) UpdateConfig(config *models.LDAPConfig) error {
	var existingConfig models.LDAPConfig
	err := ls.db.First(&existingConfig).Error
	if err == gorm.ErrRecordNotFound {
		return ls.db.Create(config).Error
	} else if err != nil {
		return err
	}
	
	config.ID = existingConfig.ID
	return ls.db.Save(config).Error
}

// TestConnection 测试LDAP连接
func (ls *LDAPService) TestConnection(config *models.LDAPConfig) *models.LDAPTestResponse {
	// 构建连接地址
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)
	
	var conn *ldap.Conn
	var err error
	
	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, nil)
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	
	if err != nil {
		return &models.LDAPTestResponse{
			Success: false,
			Message: "连接LDAP服务器失败",
			Details: err.Error(),
		}
	}
	defer conn.Close()
	
	// 测试绑定
	if config.BindDN != "" {
		err = conn.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			return &models.LDAPTestResponse{
				Success: false,
				Message: "LDAP绑定失败",
				Details: err.Error(),
			}
		}
	}
	
	// 测试搜索
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)
	
	_, err = conn.Search(searchRequest)
	if err != nil {
		return &models.LDAPTestResponse{
			Success: false,
			Message: "LDAP搜索测试失败",
			Details: err.Error(),
		}
	}
	
	return &models.LDAPTestResponse{
		Success: true,
		Message: "LDAP连接测试成功",
	}
}

// SyncUsers 同步LDAP用户
func (ls *LDAPService) SyncUsers(options *models.LDAPSyncOptions) (*models.LDAPSyncResult, error) {
	config, err := ls.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("获取LDAP配置失败: %v", err)
	}
	
	if !config.SyncEnabled {
		return nil, fmt.Errorf("LDAP同步未启用")
	}
	
	startTime := time.Now()
	result := &models.LDAPSyncResult{
		StartTime: startTime,
		Details:   make([]models.LDAPSyncDetail, 0),
	}
	
	// 连接LDAP
	ldapUsers, err := ls.searchLDAPUsers(config, options)
	if err != nil {
		return nil, fmt.Errorf("搜索LDAP用户失败: %v", err)
	}
	
	// 同步用户
	for _, ldapUser := range ldapUsers {
		detail := ls.syncSingleUser(config, ldapUser, options)
		result.Details = append(result.Details, detail)
		
		switch detail.Action {
		case "created":
			result.Created++
		case "updated":
			result.Updated++
		case "skipped":
			result.Skipped++
		case "error":
			result.Errors++
		}
	}
	
	// 更新同步状态
	config.LastSync = &startTime
	config.SyncStatus = "success"
	ls.db.Save(config)
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).String()
	
	return result, nil
}

// searchLDAPUsers 搜索LDAP用户
func (ls *LDAPService) searchLDAPUsers(config *models.LDAPConfig, options *models.LDAPSyncOptions) ([]models.LDAPUser, error) {
	// 连接LDAP
	addr := fmt.Sprintf("%s:%d", config.Server, config.Port)
	var conn *ldap.Conn
	var err error
	
	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, nil)
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	
	// 绑定
	if config.BindDN != "" {
		err = conn.Bind(config.BindDN, config.BindPassword)
		if err != nil {
			return nil, err
		}
	}
	
	// 构建用户搜索过滤器
	filter := config.UserFilter
	if filter == "" {
		filter = "(objectClass=person)"
	}
	
	if options != nil && options.UserFilter != "" {
		filter = options.UserFilter
	}
	
	// 搜索用户
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{config.UsernameAttr, config.EmailAttr, config.DisplayNameAttr, "dn"},
		nil,
	)
	
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	
	var users []models.LDAPUser
	for _, entry := range sr.Entries {
		user := models.LDAPUser{
			DN:          entry.DN,
			Username:    entry.GetAttributeValue(config.UsernameAttr),
			Email:       entry.GetAttributeValue(config.EmailAttr),
			DisplayName: entry.GetAttributeValue(config.DisplayNameAttr),
		}
		
		// 如果指定了特定用户，跳过不在列表中的用户
		if options != nil && len(options.SelectedUsers) > 0 {
			found := false
			for _, selectedUser := range options.SelectedUsers {
				if user.Username == selectedUser {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		users = append(users, user)
	}
	
	return users, nil
}

// syncSingleUser 同步单个用户
func (ls *LDAPService) syncSingleUser(config *models.LDAPConfig, ldapUser models.LDAPUser, options *models.LDAPSyncOptions) models.LDAPSyncDetail {
	detail := models.LDAPSyncDetail{
		Username: ldapUser.Username,
		Email:    ldapUser.Email,
	}
	
	// 检查用户是否已存在
	var existingUser models.User
	err := ls.db.Where("username = ? OR email = ?", ldapUser.Username, ldapUser.Email).First(&existingUser).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新用户
		if options != nil && options.DryRun {
			detail.Action = "created"
			detail.Message = "将创建新用户（仅模拟）"
			return detail
		}
		
		if !config.AutoCreateUser {
			detail.Action = "skipped"
			detail.Message = "跳过创建新用户（自动创建用户已禁用）"
			return detail
		}
		
		newUser := models.User{
			Username:   ldapUser.Username,
			Email:      ldapUser.Email,
			AuthSource: "ldap",
			LDAPDn:     ldapUser.DN,
			IsActive:   true,
		}
		
		if err := ls.db.Create(&newUser).Error; err != nil {
			detail.Action = "error"
			detail.Message = "创建用户失败: " + err.Error()
			return detail
		}
		
		// 分配默认角色
		if config.DefaultRole != "" {
			var role models.Role
			if err := ls.db.Where("name = ?", config.DefaultRole).First(&role).Error; err == nil {
				ls.db.Model(&newUser).Association("Roles").Append(&role)
			}
		}
		
		detail.Action = "created"
		detail.Message = "成功创建新用户"
		
	} else if err != nil {
		detail.Action = "error"
		detail.Message = "查询用户失败: " + err.Error()
		
	} else {
		// 更新已存在的用户
		if existingUser.AuthSource != "ldap" {
			detail.Action = "skipped"
			detail.Message = "跳过非LDAP用户"
			return detail
		}
		
		if options != nil && !options.ForceUpdate {
			detail.Action = "skipped"
			detail.Message = "用户已存在，跳过更新"
			return detail
		}
		
		if options != nil && options.DryRun {
			detail.Action = "updated"
			detail.Message = "将更新用户信息（仅模拟）"
			return detail
		}
		
		// 更新用户信息
		existingUser.Email = ldapUser.Email
		existingUser.LDAPDn = ldapUser.DN
		
		if err := ls.db.Save(&existingUser).Error; err != nil {
			detail.Action = "error"
			detail.Message = "更新用户失败: " + err.Error()
			return detail
		}
		
		detail.Action = "updated"
		detail.Message = "成功更新用户信息"
	}
	
	return detail
}

// SearchUsers 搜索LDAP用户
func (ls *LDAPService) SearchUsers(query string) ([]models.LDAPUser, error) {
	config, err := ls.GetConfig()
	if err != nil {
		return nil, err
	}
	
	// 构建搜索过滤器
	filter := fmt.Sprintf("(&(objectClass=person)(|(uid=*%s*)(cn=*%s*)(mail=*%s*)))", query, query, query)
	
	options := &models.LDAPSyncOptions{
		UserFilter: filter,
	}
	
	return ls.searchLDAPUsers(config, options)
}
