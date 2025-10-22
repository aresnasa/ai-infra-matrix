package controllers

import (
	"fmt"
	"net/http"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LDAPController struct {
	ldapService *services.LDAPService
}

func NewLDAPController(db *gorm.DB) *LDAPController {
	return &LDAPController{
		ldapService: services.NewLDAPService(db),
	}
}

// GetConfig 获取LDAP配置
func (lc *LDAPController) GetConfig(c *gin.Context) {
	config, err := lc.ldapService.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取LDAP配置失败"})
		return
	}

	// 隐藏敏感信息
	config.BindPassword = ""

	c.JSON(http.StatusOK, gin.H{"data": config})
}

// UpdateConfig 更新LDAP配置
func (lc *LDAPController) UpdateConfig(c *gin.Context) {
	var config models.LDAPConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	if err := lc.ldapService.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新LDAP配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "LDAP配置更新成功"})
}

// TestConnection 测试LDAP连接
func (lc *LDAPController) TestConnection(c *gin.Context) {
	var testReq models.LDAPTestRequest

	// 尝试解析请求体
	if err := c.ShouldBindJSON(&testReq); err != nil {
		// 记录详细的解析错误
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求格式错误",
			"details": err.Error(),
			"message": "请检查JSON格式是否正确，必填字段: server, port, bind_dn, bind_password, base_dn",
		})
		return
	}

	// 验证必填字段
	if testReq.Server == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "LDAP服务器地址不能为空"})
		return
	}
	if testReq.Port <= 0 || testReq.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "端口号必须在1-65535之间"})
		return
	}
	if testReq.BaseDN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BaseDN不能为空"})
		return
	}

	// 设置Windows兼容的默认值
	if testReq.Timeout <= 0 {
		testReq.Timeout = 10 // 默认10秒超时
	}
	if testReq.MaxConnections <= 0 {
		testReq.MaxConnections = 5 // 默认最大5个连接
	}

	// 构建LDAP配置
	config := &models.LDAPConfig{
		Server:       testReq.Server,
		Port:         testReq.Port,
		UseSSL:       testReq.UseSSL,
		BaseDN:       testReq.BaseDN,
		BindDN:       testReq.BindDN,
		BindPassword: testReq.BindPassword,
	}

	// 测试连接
	result := lc.ldapService.TestConnection(config)

	// 确保返回正确的HTTP状态码
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, result)
	}
}

// SyncUsers 同步LDAP用户
func (lc *LDAPController) SyncUsers(c *gin.Context) {
	var options models.LDAPSyncOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		// 使用默认选项
		options = models.LDAPSyncOptions{
			ForceUpdate:   false,
			SyncGroups:    false,
			DeleteMissing: false,
			DryRun:        false,
		}
	}

	result, err := lc.ldapService.SyncUsers(&options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LDAP同步失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// SearchUsers 搜索LDAP用户
func (lc *LDAPController) SearchUsers(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索查询不能为空"})
		return
	}

	users, err := lc.ldapService.SearchUsers(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索LDAP用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// GetUserGroups 获取LDAP用户组（占位符，可后续实现）
func (lc *LDAPController) GetUserGroups(c *gin.Context) {
	// 这里可以实现LDAP组的搜索
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
}

// SyncGroups 同步LDAP用户组（占位符，可后续实现）
func (lc *LDAPController) SyncGroups(c *gin.Context) {
	// 这里可以实现LDAP组的同步
	c.JSON(http.StatusOK, gin.H{"message": "LDAP组同步功能待实现"})
}

// GetRecommendedSettings 获取推荐的LDAP设置
func (lc *LDAPController) GetRecommendedSettings(c *gin.Context) {
	serverType := c.Query("type")
	if serverType == "" {
		serverType = "windows" // 默认为Windows AD
	}

	helper := services.NewLDAPConnectionHelper()
	settings := helper.GetRecommendedSettings(serverType)

	c.JSON(http.StatusOK, gin.H{
		"server_type": serverType,
		"settings":    settings,
		"message":     fmt.Sprintf("推荐的%s LDAP设置", serverType),
	})
}

// ValidateConfig 验证LDAP配置
func (lc *LDAPController) ValidateConfig(c *gin.Context) {
	var config models.LDAPConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "配置格式错误",
			"details": err.Error(),
		})
		return
	}

	// 进行配置验证，但不保存
	testConfig := &models.LDAPConfig{
		Server:       config.Server,
		Port:         config.Port,
		UseSSL:       config.UseSSL,
		BaseDN:       config.BaseDN,
		BindDN:       config.BindDN,
		BindPassword: config.BindPassword,
	}

	result := lc.ldapService.TestConnection(testConfig)

	if result.Success {
		c.JSON(http.StatusOK, gin.H{
			"valid":   true,
			"message": "配置验证通过",
			"details": result.Details,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"message": result.Message,
			"details": result.Details,
		})
	}
}
