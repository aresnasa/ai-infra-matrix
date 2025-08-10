package services

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
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

// GetLDAPConfig 获取LDAP配置
func (s *LDAPService) GetLDAPConfig() (*models.LDAPConfig, error) {
	var config models.LDAPConfig
	err := s.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			return &models.LDAPConfig{
				IsEnabled:    false,
				Port:         389,
				UseSSL:       false,
				SkipVerify:   false,
				UserFilter:   "(objectClass=person)",
				UsernameAttr: "uid",
				NameAttr:     "cn",
				EmailAttr:    "mail",
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// UpdateLDAPConfig 更新LDAP配置
func (s *LDAPService) UpdateLDAPConfig(req *models.LDAPConfigRequest) (*models.LDAPConfig, error) {
	var config models.LDAPConfig
	err := s.db.First(&config).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.LDAPConfig{
			IsEnabled:    req.IsEnabled,
			Server:       req.Server,
			Port:         req.Port,
			UseSSL:       req.UseSSL,
			SkipVerify:   req.SkipVerify,
			BindDN:       req.BindDN,
			BindPassword: req.BindPassword,
			BaseDN:       req.BaseDN,
			UserFilter:   req.UserFilter,
			UsernameAttr: req.UsernameAttr,
			NameAttr:     req.NameAttr,
			EmailAttr:    req.EmailAttr,
		}
		
		// 设置默认值
		if config.UserFilter == "" {
			config.UserFilter = "(objectClass=person)"
		}
		if config.UsernameAttr == "" {
			config.UsernameAttr = "uid"
		}
		if config.NameAttr == "" {
			config.NameAttr = "cn"
		}
		if config.EmailAttr == "" {
			config.EmailAttr = "mail"
	}
		
		err = s.db.Create(&config).Error
	} else if err == nil {
		// 更新现有配置
		config.IsEnabled = req.IsEnabled
		config.Server = req.Server
		config.Port = req.Port
		config.UseSSL = req.UseSSL
		config.SkipVerify = req.SkipVerify
		config.BindDN = req.BindDN
		config.BindPassword = req.BindPassword
		config.BaseDN = req.BaseDN
		config.UserFilter = req.UserFilter
		config.UsernameAttr = req.UsernameAttr
		config.NameAttr = req.NameAttr
		config.EmailAttr = req.EmailAttr
	config.UsersOU = req.UsersOU
	config.GroupsOU = req.GroupsOU
	config.AdminGroupDN = req.AdminGroupDN
	config.GroupMemberAttr = req.GroupMemberAttr
		
		err = s.db.Save(&config).Error
	}
	
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

// TestLDAPConnection 测试LDAP连接
func (s *LDAPService) TestLDAPConnection(req *models.LDAPTestRequest) error {
	conn, err := s.createLDAPConnection(req.Server, req.Port, req.UseSSL, req.SkipVerify)
	if err != nil {
		return fmt.Errorf("连接LDAP服务器失败: %v", err)
	}
	defer conn.Close()

	// 尝试绑定
	err = conn.Bind(req.BindDN, req.BindPassword)
	if err != nil {
		return fmt.Errorf("LDAP认证失败: %v", err)
	}

	// 测试基本搜索功能，使用限制性查询避免大小限制问题
	searchRequest := ldap.NewSearchRequest(
		req.BaseDN,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 1, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	_, err = conn.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("LDAP搜索失败: %v", err)
	}

	return nil
}

// AuthenticateUser 通过LDAP认证用户
func (s *LDAPService) AuthenticateUser(username, password string) (*models.LDAPUser, error) {
	config, err := s.GetLDAPConfig()
	if err != nil {
		log.Printf("LDAP配置获取失败: %v", err)
		return nil, err
	}

	if !config.IsEnabled {
		log.Printf("LDAP认证未启用")
		return nil, fmt.Errorf("LDAP认证未启用")
	}

	log.Printf("LDAP认证开始: username=%s, server=%s:%d", username, config.Server, config.Port)

	conn, err := s.createLDAPConnection(config.Server, config.Port, config.UseSSL, config.SkipVerify)
	if err != nil {
		log.Printf("连接LDAP服务器失败: %v", err)
		return nil, fmt.Errorf("连接LDAP服务器失败: %v", err)
	}
	defer conn.Close()

	// 使用管理员账号绑定
	log.Printf("使用管理员账号绑定: %s", config.BindDN)
	err = conn.Bind(config.BindDN, config.BindPassword)
	if err != nil {
		log.Printf("LDAP管理员绑定失败: %v", err)
		return nil, fmt.Errorf("LDAP管理员绑定失败: %v", err)
	}
	log.Printf("管理员绑定成功")

	// 搜索用户
	var userFilter string
	if strings.Contains(config.UserFilter, "{username}") {
		// 如果用户过滤器包含 {username} 占位符，直接替换
		userFilter = strings.ReplaceAll(config.UserFilter, "{username}", username)
	} else if config.UsernameAttr != "" {
		// 如果配置了用户名属性，构造过滤器
		userFilter = fmt.Sprintf("(&%s(%s=%s))", config.UserFilter, config.UsernameAttr, username)
	} else {
		// 默认使用uid属性
		userFilter = fmt.Sprintf("(uid=%s)", username)
	}
	
	log.Printf("搜索用户: baseDN=%s, filter=%s", config.BaseDN, userFilter)
	
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		userFilter,
		[]string{"dn", config.NameAttr, config.EmailAttr},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Printf("搜索用户失败: %v", err)
		return nil, fmt.Errorf("搜索用户失败: %v", err)
	}

	log.Printf("搜索结果: 找到 %d 个用户", len(sr.Entries))
	if len(sr.Entries) == 0 {
		log.Printf("用户不存在: %s", username)
		return nil, fmt.Errorf("用户不存在")
	}

	userEntry := sr.Entries[0]
	userDN := userEntry.DN
	log.Printf("找到用户: DN=%s", userDN)

	// 使用用户凭据进行认证
	log.Printf("尝试用户认证: %s", userDN)
	err = conn.Bind(userDN, password)
	if err != nil {
		log.Printf("用户认证失败: %v", err)
		return nil, fmt.Errorf("用户认证失败: %v", err)
	}
	log.Printf("用户认证成功: %s", userDN)

	// 获取用户信息
	ldapUser := &models.LDAPUser{
		DN:       userDN,
		Username: username, // 始终使用登录时的用户名
	}

	log.Printf("创建LDAPUser对象: Username=%s, DN=%s", username, userDN)

	// 获取显示名称（但不覆盖用户名）
	if config.NameAttr != "" {
		displayName := userEntry.GetAttributeValue(config.NameAttr)
		if displayName != "" {
			ldapUser.DisplayName = displayName
			log.Printf("设置DisplayName: %s", displayName)
		}
	}

	// 获取邮箱
	if config.EmailAttr != "" {
		ldapUser.Email = userEntry.GetAttributeValue(config.EmailAttr)
	}

	log.Printf("返回LDAPUser对象: Username=%s, DisplayName=%s, Email=%s", 
		ldapUser.Username, ldapUser.DisplayName, ldapUser.Email)

	return ldapUser, nil
}

// IsUserAdmin 检查用户是否为管理员（简化版本）
func (s *LDAPService) IsUserAdmin(ldapUser *models.LDAPUser) bool {
	// 由于模型中没有AdminGroups字段，这里返回false
	// 可以根据需要扩展其他管理员判断逻辑
	return false
}

// createLDAPConnection 创建LDAP连接
func (s *LDAPService) createLDAPConnection(server string, port int, useSSL, skipVerify bool) (*ldap.Conn, error) {
	address := fmt.Sprintf("%s:%d", server, port)

	var conn *ldap.Conn
	var err error

	if useSSL {
		// 使用SSL连接
		conn, err = ldap.DialTLS("tcp", address, &tls.Config{InsecureSkipVerify: skipVerify})
	} else {
		// 使用普通连接
		conn, err = ldap.Dial("tcp", address)
	}

	return conn, err
}

// CreateUser 在LDAP中创建用户并可选加入组
func (s *LDAPService) CreateUser(username, password, email, displayName, department string) error {
	config, err := s.GetLDAPConfig()
	if err != nil {
		return fmt.Errorf("获取LDAP配置失败: %w", err)
	}
	if !config.IsEnabled {
		return fmt.Errorf("LDAP未启用")
	}

	conn, err := s.createLDAPConnection(config.Server, config.Port, config.UseSSL, config.SkipVerify)
	if err != nil {
		return fmt.Errorf("连接LDAP失败: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(config.BindDN, config.BindPassword); err != nil {
		return fmt.Errorf("LDAP管理员绑定失败: %w", err)
	}

	// 构造用户DN: uid={username},<UsersOU or BaseDN>
	usersBase := strings.TrimSpace(config.UsersOU)
	if usersBase == "" {
		usersBase = config.BaseDN
	} else if !strings.Contains(usersBase, config.BaseDN) {
		usersBase = usersBase + "," + config.BaseDN
	}
	userDN := fmt.Sprintf("uid=%s,%s", username, usersBase)

	// inetOrgPerson + posixAccount（大多数OpenLDAP常用）
	req := ldap.NewAddRequest(userDN, nil)
	req.Attribute("objectClass", []string{"inetOrgPerson", "posixAccount"})
	req.Attribute("uid", []string{username})
	// cn/sn/givenName 尽量填充
	cn := displayName
	if cn == "" {
		cn = username
	}
	req.Attribute("cn", []string{cn})
	req.Attribute("sn", []string{cn})
	req.Attribute("givenName", []string{cn})
	if email != "" {
		req.Attribute("mail", []string{email})
	}
	// 简化：使用随机的 uidNumber/gidNumber 起步，生产应有分配器
	// 这里用时间戳低位生成一个相对唯一的整数
	uidNumber := fmt.Sprintf("%d", (time.Now().Unix()%90000)+10000)
	gidNumber := uidNumber
	req.Attribute("uidNumber", []string{uidNumber})
	req.Attribute("gidNumber", []string{gidNumber})
	req.Attribute("homeDirectory", []string{fmt.Sprintf("/home/%s", username)})
	// 明文密码属性（OpenLDAP会根据配置加密），也可使用 userPassword: {SSHA}... 预加密
	req.Attribute("userPassword", []string{password})

	if err := conn.Add(req); err != nil {
		// 若已存在则忽略
		if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return fmt.Errorf("创建LDAP用户失败: %w", err)
		}
	}

	// 如果提供了部门，则尝试把用户加入对应组: cn={department},<GroupsOU or BaseDN>
	if department != "" {
		groupsBase := strings.TrimSpace(config.GroupsOU)
		if groupsBase == "" {
			groupsBase = config.BaseDN
		} else if !strings.Contains(groupsBase, config.BaseDN) {
			groupsBase = groupsBase + "," + config.BaseDN
		}
		groupDN := fmt.Sprintf("cn=%s,%s", department, groupsBase)
		mod := ldap.NewModifyRequest(groupDN, nil)
		memberAttr := config.GroupMemberAttr
		if memberAttr == "" {
			// 默认使用 groupOfNames 的 member 属性
			memberAttr = "member"
		}
		// 常见两种：member(需要DN) 或 memberUid(只要uid)
		if strings.ToLower(memberAttr) == "memberuid" {
			mod.Add(memberAttr, []string{username})
		} else {
			mod.Add(memberAttr, []string{userDN})
		}
		if err := conn.Modify(mod); err != nil {
			// 组不存在或已存在成员，忽略警告
			if !(strings.Contains(strings.ToLower(err.Error()), "no such object") ||
				 strings.Contains(strings.ToLower(err.Error()), "already exists")) {
				return fmt.Errorf("添加用户到组失败: %w", err)
			}
		}
	}

	return nil
}
