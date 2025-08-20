package controllers

import (
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
	if err := c.ShouldBindJSON(&testReq); err != nil {
		// 如果没有提供测试配置，使用当前配置
		config, err := lc.ldapService.GetConfig()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误且无法获取当前配置"})
			return
		}
		
		result := lc.ldapService.TestConnection(config)
		c.JSON(http.StatusOK, result)
		return
	}
	
	// 使用提供的测试配置
	config := &models.LDAPConfig{
		Server:       testReq.Server,
		Port:         testReq.Port,
		UseSSL:       testReq.UseSSL,
		BaseDN:       testReq.BaseDN,
		BindDN:       testReq.BindDN,
		BindPassword: testReq.BindPassword,
	}
	
	result := lc.ldapService.TestConnection(config)
	c.JSON(http.StatusOK, result)
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
