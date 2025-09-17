package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/go-ldap/ldap/v3"
	"gorm.io/gorm"
)

type LDAPService struct {
	db             *gorm.DB
	connectionHelper *LDAPConnectionHelper
}

func NewLDAPService(db *gorm.DB) *LDAPService {
	return &LDAPService{
		db:             db,
		connectionHelper: NewLDAPConnectionHelper(),
	}
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
	// 验证配置
	if err := ls.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}
	
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

// validateConfig 验证LDAP配置
func (ls *LDAPService) validateConfig(config *models.LDAPConfig) error {
	if config.Server == "" {
		return fmt.Errorf("LDAP服务器地址不能为空")
	}
	
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}
	
	if config.BaseDN == "" {
		return fmt.Errorf("BaseDN不能为空")
	}
	
	// Windows AD 特殊验证
	if ls.isWindowsAD(config) {
		return ls.validateWindowsAD(config)
	}
	
	return nil
}

// isWindowsAD 检测是否为Windows Active Directory
func (ls *LDAPService) isWindowsAD(config *models.LDAPConfig) bool {
	// 简单检测：Windows AD 通常使用389/636端口，BaseDN包含dc=
	return (config.Port == 389 || config.Port == 636) && 
		   (len(config.BaseDN) > 3 && config.BaseDN[:3] == "dc=")
}

// validateWindowsAD 验证Windows AD配置
func (ls *LDAPService) validateWindowsAD(config *models.LDAPConfig) error {
	// Windows AD 特殊要求
	if config.BindDN == "" {
		return fmt.Errorf("Windows AD 需要提供绑定用户DN")
	}
	
	if config.BindPassword == "" {
		return fmt.Errorf("Windows AD 需要提供绑定用户密码")
	}
	
	// 建议使用SSL连接
	if !config.UseSSL && config.Port != 389 {
		// 这是警告，不阻止配置
	}
	
	return nil
}

// TestConnection 测试LDAP连接
func (ls *LDAPService) TestConnection(config *models.LDAPConfig) *models.LDAPTestResponse {
	// 验证配置
	if err := ls.validateConfig(config); err != nil {
		return &models.LDAPTestResponse{
			Success: false,
			Message: "配置验证失败",
			Details: err.Error(),
		}
	}
	
	// 使用助手类进行连接测试，包含重试机制
	return ls.connectionHelper.TestConnectionWithRetry(config, 3)
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

// GetLDAPConfig 获取LDAP配置 (兼容性方法)
func (ls *LDAPService) GetLDAPConfig() (*models.LDAPConfig, error) {
	return ls.GetConfig()
}

// UpdateLDAPConfig 更新LDAP配置 (兼容性方法)
func (ls *LDAPService) UpdateLDAPConfig(req *models.LDAPConfigRequest) (*models.LDAPConfig, error) {
	config := &models.LDAPConfig{
		Server:       req.Server,
		Port:         req.Port,
		BindDN:       req.BindDN,
		BindPassword: req.BindPassword,
		BaseDN:       req.BaseDN,
		UserFilter:   req.UserFilter,
		UsernameAttr: req.UsernameAttr,
		EmailAttr:    req.EmailAttr,
		NameAttr:     req.NameAttr,
		UseSSL:       req.UseSSL,
		SkipVerify:   req.SkipVerify,
		IsEnabled:    req.IsEnabled,
		UsersOU:      req.UsersOU,
		GroupsOU:     req.GroupsOU,
		AdminGroupDN: req.AdminGroupDN,
		GroupMemberAttr: req.GroupMemberAttr,
	}
	
	err := ls.UpdateConfig(config)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// TestLDAPConnection 测试LDAP连接 - 管理员接口使用
func (ls *LDAPService) TestLDAPConnection(req *models.LDAPTestRequest) error {
	// 验证请求参数
	if req.Server == "" {
		return fmt.Errorf("LDAP服务器地址不能为空")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}
	if req.BaseDN == "" {
		return fmt.Errorf("BaseDN不能为空")
	}
	
	// 构建配置
	config := &models.LDAPConfig{
		Server:       req.Server,
		Port:         req.Port,
		BindDN:       req.BindDN,
		BindPassword: req.BindPassword,
		BaseDN:       req.BaseDN,
		UserFilter:   req.UserFilter,
		UseSSL:       req.UseSSL,
		SkipVerify:   req.SkipVerify,
	}
	
	// 使用新的测试连接方法
	response := ls.TestConnection(config)
	if !response.Success {
		return fmt.Errorf("%s: %s", response.Message, response.Details)
	}
	
	return nil
}

// AuthenticateUser 认证用户
func (ls *LDAPService) AuthenticateUser(username, password string) (*models.LDAPUser, error) {
	config, err := ls.GetConfig()
	if err != nil {
		return nil, err
	}
	
	if !config.IsEnabled {
		return nil, fmt.Errorf("LDAP认证未启用")
	}
	
	// 创建LDAP连接
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.Server, config.Port))
	if err != nil {
		return nil, fmt.Errorf("连接LDAP服务器失败: %v", err)
	}
	defer conn.Close()
	
	// 搜索用户
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&%s(%s=%s))", config.UserFilter, config.UsernameAttr, username),
		[]string{config.UsernameAttr, config.EmailAttr, config.NameAttr, "dn"},
		nil,
	)
	
	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %v", err)
	}
	
	if len(searchResult.Entries) == 0 {
		return nil, fmt.Errorf("用户不存在")
	}
	
	userEntry := searchResult.Entries[0]
	userDN := userEntry.DN
	
	// 尝试绑定用户
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("用户认证失败: %v", err)
	}
	
	// 构建用户对象
	ldapUser := &models.LDAPUser{
		DN:          userDN,
		Username:    userEntry.GetAttributeValue(config.UsernameAttr),
		Email:       userEntry.GetAttributeValue(config.EmailAttr),
		Name:        userEntry.GetAttributeValue(config.NameAttr),
		DisplayName: userEntry.GetAttributeValue(config.DisplayNameAttr),
	}
	
	return ldapUser, nil
}

// IsUserAdmin 检查用户是否为管理员
func (ls *LDAPService) IsUserAdmin(ldapUser *models.LDAPUser) bool {
	config, err := ls.GetConfig()
	if err != nil {
		return false
	}
	
	if config.AdminGroupDN == "" {
		return false
	}
	
	// 创建LDAP连接
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.Server, config.Port))
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// 绑定管理员账户
	err = conn.Bind(config.BindDN, config.BindPassword)
	if err != nil {
		return false
	}
	
	// 搜索管理员组
	searchRequest := ldap.NewSearchRequest(
		config.AdminGroupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(%s=%s)", config.GroupMemberAttr, ldapUser.DN),
		[]string{config.GroupMemberAttr},
		nil,
	)
	
	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return false
	}
	
	return len(searchResult.Entries) > 0
}

// CreateUser 创建LDAP用户 (暂不实现，返回错误)
func (ls *LDAPService) CreateUser(username, password, email, displayName, department string) error {
	return fmt.Errorf("LDAP用户创建功能暂未实现")
}
