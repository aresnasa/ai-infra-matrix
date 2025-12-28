package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuditConfig 审计中间件配置
type AuditConfig struct {
	// 需要审计的路径前缀
	AuditPaths []AuditPathConfig
	// 排除的路径（优先级高于 AuditPaths）
	ExcludePaths []string
	// 是否记录请求体
	LogRequestBody bool
	// 是否记录响应体
	LogResponseBody bool
	// 请求体最大记录大小（字节）
	MaxRequestBodySize int
	// 响应体最大记录大小（字节）
	MaxResponseBodySize int
}

// AuditPathConfig 审计路径配置
type AuditPathConfig struct {
	PathPrefix   string               // 路径前缀
	Category     models.AuditCategory // 审计类别
	ResourceType string               // 资源类型
	Methods      []string             // 需要审计的 HTTP 方法（空表示全部）
}

// responseWriter 包装的响应写入器，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// GetDefaultAuditConfig 获取默认审计配置
func GetDefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		AuditPaths: []AuditPathConfig{
			// Ansible 相关
			{PathPrefix: "/api/v1/ansible", Category: models.AuditCategoryAnsible, ResourceType: "ansible"},
			{PathPrefix: "/api/v1/playbooks", Category: models.AuditCategoryAnsible, ResourceType: "playbook"},
			{PathPrefix: "/api/v1/inventory", Category: models.AuditCategoryAnsible, ResourceType: "inventory"},

			// SLURM 相关
			{PathPrefix: "/api/v1/slurm", Category: models.AuditCategorySlurm, ResourceType: "slurm"},
			{PathPrefix: "/api/v1/slurm-clusters", Category: models.AuditCategorySlurm, ResourceType: "slurm_cluster"},
			{PathPrefix: "/api/v1/slurm-nodes", Category: models.AuditCategorySlurm, ResourceType: "slurm_node"},
			{PathPrefix: "/api/v1/slurm-jobs", Category: models.AuditCategorySlurm, ResourceType: "slurm_job"},
			{PathPrefix: "/api/v1/slurm-partitions", Category: models.AuditCategorySlurm, ResourceType: "slurm_partition"},

			// SaltStack 相关
			{PathPrefix: "/api/v1/salt", Category: models.AuditCategorySaltstack, ResourceType: "salt"},
			{PathPrefix: "/api/v1/saltstack", Category: models.AuditCategorySaltstack, ResourceType: "saltstack"},
			{PathPrefix: "/api/v1/minions", Category: models.AuditCategorySaltstack, ResourceType: "minion"},
			{PathPrefix: "/api/v1/salt-keys", Category: models.AuditCategorySaltstack, ResourceType: "salt_key"},
			{PathPrefix: "/api/v1/salt-jobs", Category: models.AuditCategorySaltstack, ResourceType: "salt_job"},

			// 角色模板相关
			{PathPrefix: "/api/v1/roles", Category: models.AuditCategoryRoleTemplate, ResourceType: "role"},
			{PathPrefix: "/api/v1/role-templates", Category: models.AuditCategoryRoleTemplate, ResourceType: "role_template"},
			{PathPrefix: "/api/v1/permissions", Category: models.AuditCategoryRoleTemplate, ResourceType: "permission"},
			{PathPrefix: "/api/v1/rbac", Category: models.AuditCategoryRoleTemplate, ResourceType: "rbac"},

			// Kubernetes 相关
			{PathPrefix: "/api/v1/kubernetes", Category: models.AuditCategoryKubernetes, ResourceType: "kubernetes"},
			{PathPrefix: "/api/v1/k8s", Category: models.AuditCategoryKubernetes, ResourceType: "kubernetes"},
			{PathPrefix: "/api/v1/clusters", Category: models.AuditCategoryKubernetes, ResourceType: "cluster"},
			{PathPrefix: "/api/v1/namespaces", Category: models.AuditCategoryKubernetes, ResourceType: "namespace"},
			{PathPrefix: "/api/v1/deployments", Category: models.AuditCategoryKubernetes, ResourceType: "deployment"},
			{PathPrefix: "/api/v1/services", Category: models.AuditCategoryKubernetes, ResourceType: "service"},
			{PathPrefix: "/api/v1/helm", Category: models.AuditCategoryKubernetes, ResourceType: "helm"},

			// 监控相关
			{PathPrefix: "/api/v1/monitor", Category: models.AuditCategoryMonitor, ResourceType: "monitor"},
			{PathPrefix: "/api/v1/alerts", Category: models.AuditCategoryMonitor, ResourceType: "alert"},
			{PathPrefix: "/api/v1/dashboards", Category: models.AuditCategoryMonitor, ResourceType: "dashboard"},
			{PathPrefix: "/api/v1/metrics", Category: models.AuditCategoryMonitor, ResourceType: "metrics", Methods: []string{"POST", "PUT", "DELETE"}},
			{PathPrefix: "/api/v1/nightingale", Category: models.AuditCategoryMonitor, ResourceType: "nightingale"},

			// Admin 相关
			{PathPrefix: "/api/v1/admin", Category: models.AuditCategoryAdmin, ResourceType: "admin"},
			{PathPrefix: "/api/v1/users", Category: models.AuditCategoryAdmin, ResourceType: "user"},
			{PathPrefix: "/api/v1/user-groups", Category: models.AuditCategoryAdmin, ResourceType: "user_group"},
			{PathPrefix: "/api/v1/settings", Category: models.AuditCategoryAdmin, ResourceType: "settings"},
			{PathPrefix: "/api/v1/config", Category: models.AuditCategoryAdmin, ResourceType: "config"},
			{PathPrefix: "/api/v1/ldap", Category: models.AuditCategoryAdmin, ResourceType: "ldap"},
			{PathPrefix: "/api/v1/security", Category: models.AuditCategorySecurity, ResourceType: "security"},
		},
		ExcludePaths: []string{
			"/api/v1/health",
			"/api/v1/metrics",
			"/api/v1/ws",
			"/api/v1/auth/refresh",
			"/api/v1/audit", // 审计接口本身不审计
		},
		LogRequestBody:      true,
		LogResponseBody:     false, // 响应体通常较大，默认不记录
		MaxRequestBodySize:  10240, // 10KB
		MaxResponseBodySize: 1024,  // 1KB
	}
}

// AuditMiddleware 审计中间件
func AuditMiddleware(config *AuditConfig) gin.HandlerFunc {
	if config == nil {
		config = GetDefaultAuditConfig()
	}

	auditService := services.GetAuditService()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		// 检查是否需要排除
		for _, excludePath := range config.ExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				c.Next()
				return
			}
		}

		// 只审计写操作（POST, PUT, PATCH, DELETE）
		// GET 请求只在特定路径下审计（如执行命令等）
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			// 检查是否是特殊的 GET 请求需要审计
			if !shouldAuditGetRequest(path) {
				c.Next()
				return
			}
		}

		// 查找匹配的审计配置
		var auditPath *AuditPathConfig
		for i := range config.AuditPaths {
			if strings.HasPrefix(path, config.AuditPaths[i].PathPrefix) {
				// 检查方法是否需要审计
				if len(config.AuditPaths[i].Methods) > 0 {
					methodMatch := false
					for _, m := range config.AuditPaths[i].Methods {
						if m == method {
							methodMatch = true
							break
						}
					}
					if !methodMatch {
						continue
					}
				}
				auditPath = &config.AuditPaths[i]
				break
			}
		}

		if auditPath == nil {
			c.Next()
			return
		}

		// 记录开始时间
		startTime := time.Now()

		// 读取请求体
		var requestBody string
		if config.LogRequestBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) > config.MaxRequestBodySize {
				requestBody = string(bodyBytes[:config.MaxRequestBodySize]) + "...(truncated)"
			} else {
				requestBody = string(bodyBytes)
			}
		}

		// 包装响应写入器
		var responseBody string
		if config.LogResponseBody {
			rw := &responseWriter{ResponseWriter: c.Writer, body: bytes.NewBufferString("")}
			c.Writer = rw
			defer func() {
				if rw.body.Len() > config.MaxResponseBodySize {
					responseBody = rw.body.String()[:config.MaxResponseBodySize] + "...(truncated)"
				} else {
					responseBody = rw.body.String()
				}
			}()
		}

		// 处理请求
		c.Next()

		// 计算执行时间
		executionTime := time.Since(startTime)

		// 确定操作动作
		action := determineAction(method, path)

		// 确定状态
		status := models.AuditStatusSuccess
		var errorMessage string
		if c.Writer.Status() >= 400 {
			status = models.AuditStatusFailed
			// 尝试从响应中获取错误信息
			if responseBody != "" {
				var resp map[string]interface{}
				if json.Unmarshal([]byte(responseBody), &resp) == nil {
					if msg, ok := resp["message"].(string); ok {
						errorMessage = msg
					}
				}
			}
		}

		// 确定严重程度
		severity := determineSeverity(auditPath.Category, action, status)

		// 提取资源ID
		resourceID := extractResourceID(path)

		// 创建审计条目
		entry := auditService.NewAuditEntry(auditPath.Category, action).
			WithUserFromContext(c).
			WithResource(auditPath.ResourceType, resourceID, "").
			WithRequest(method, path, nil).
			WithClient(c.ClientIP(), c.Request.UserAgent(), "").
			WithStatus(status).
			WithSeverity(severity).
			WithExecutionTime(executionTime)

		// 记录请求参数
		if requestBody != "" {
			entry.WithRequestParams(requestBody)
		}

		// 记录错误信息
		if errorMessage != "" {
			entry.WithErrorMessage(errorMessage)
		}

		// 异步保存
		entry.SaveAsync()
	}
}

// shouldAuditGetRequest 判断 GET 请求是否需要审计
func shouldAuditGetRequest(path string) bool {
	// 一些执行类的 GET 请求需要审计
	auditableGetPaths := []string{
		"/execute",
		"/run",
		"/trigger",
		"/export",
		"/download",
	}

	for _, p := range auditableGetPaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// determineAction 根据 HTTP 方法和路径确定操作类型
func determineAction(method, path string) models.AuditAction {
	// 特殊路径处理
	pathLower := strings.ToLower(path)

	// Salt 相关
	if strings.Contains(pathLower, "/salt") {
		if strings.Contains(pathLower, "/execute") || strings.Contains(pathLower, "/run") {
			return models.AuditActionSaltExecute
		}
		if strings.Contains(pathLower, "/state") || strings.Contains(pathLower, "/apply") {
			return models.AuditActionSaltStateApply
		}
		if strings.Contains(pathLower, "/key") {
			if strings.Contains(pathLower, "/accept") {
				return models.AuditActionSaltKeyAccept
			}
			if strings.Contains(pathLower, "/reject") {
				return models.AuditActionSaltKeyReject
			}
			if method == "DELETE" {
				return models.AuditActionSaltKeyDelete
			}
		}
	}

	// Ansible 相关
	if strings.Contains(pathLower, "/playbook") {
		if strings.Contains(pathLower, "/execute") || strings.Contains(pathLower, "/run") {
			if strings.Contains(pathLower, "/dry") {
				return models.AuditActionPlaybookDryRun
			}
			return models.AuditActionPlaybookRun
		}
	}

	// SLURM 相关
	if strings.Contains(pathLower, "/slurm") || strings.Contains(pathLower, "/job") {
		if strings.Contains(pathLower, "/submit") {
			return models.AuditActionJobSubmit
		}
		if strings.Contains(pathLower, "/cancel") {
			return models.AuditActionJobCancel
		}
		if strings.Contains(pathLower, "/node") {
			if strings.Contains(pathLower, "/drain") {
				return models.AuditActionNodeDrain
			}
			if strings.Contains(pathLower, "/resume") {
				return models.AuditActionNodeResume
			}
		}
		if strings.Contains(pathLower, "/deploy") {
			return models.AuditActionClusterDeploy
		}
	}

	// Kubernetes 相关
	if strings.Contains(pathLower, "/k8s") || strings.Contains(pathLower, "/kubernetes") {
		if strings.Contains(pathLower, "/helm") {
			if strings.Contains(pathLower, "/install") {
				return models.AuditActionK8sHelmInstall
			}
			if strings.Contains(pathLower, "/upgrade") {
				return models.AuditActionK8sHelmUpgrade
			}
			if strings.Contains(pathLower, "/uninstall") || method == "DELETE" {
				return models.AuditActionK8sHelmUninstall
			}
		}
		if strings.Contains(pathLower, "/scale") {
			return models.AuditActionScale
		}
		if strings.Contains(pathLower, "/restart") {
			return models.AuditActionRestart
		}
	}

	// 角色相关
	if strings.Contains(pathLower, "/role") {
		if strings.Contains(pathLower, "/assign") {
			return models.AuditActionRoleAssign
		}
		if strings.Contains(pathLower, "/revoke") {
			return models.AuditActionRoleRevoke
		}
	}

	// 权限相关
	if strings.Contains(pathLower, "/permission") {
		if strings.Contains(pathLower, "/grant") {
			return models.AuditActionPermissionGrant
		}
		if strings.Contains(pathLower, "/revoke") {
			return models.AuditActionPermissionRevoke
		}
	}

	// 用户相关
	if strings.Contains(pathLower, "/user") {
		if strings.Contains(pathLower, "/password") || strings.Contains(pathLower, "/reset") {
			return models.AuditActionPasswordReset
		}
		if strings.Contains(pathLower, "/lock") {
			return models.AuditActionUserLock
		}
		if strings.Contains(pathLower, "/unlock") {
			return models.AuditActionUserUnlock
		}
	}

	// 配置相关
	if strings.Contains(pathLower, "/config") || strings.Contains(pathLower, "/setting") {
		if method == "PUT" || method == "PATCH" || method == "POST" {
			return models.AuditActionConfigUpdate
		}
	}

	// 导入导出
	if strings.Contains(pathLower, "/import") {
		return models.AuditActionImport
	}
	if strings.Contains(pathLower, "/export") {
		return models.AuditActionExport
	}

	// 同步
	if strings.Contains(pathLower, "/sync") {
		return models.AuditActionSync
	}

	// 部署
	if strings.Contains(pathLower, "/deploy") {
		return models.AuditActionDeploy
	}

	// 启用/禁用
	if strings.Contains(pathLower, "/enable") {
		return models.AuditActionEnable
	}
	if strings.Contains(pathLower, "/disable") {
		return models.AuditActionDisable
	}

	// 默认根据 HTTP 方法判断
	switch method {
	case "POST":
		return models.AuditActionCreate
	case "PUT", "PATCH":
		return models.AuditActionUpdate
	case "DELETE":
		return models.AuditActionDelete
	case "GET":
		return models.AuditActionRead
	default:
		return models.AuditActionExecute
	}
}

// determineSeverity 确定操作的严重程度
func determineSeverity(category models.AuditCategory, action models.AuditAction, status models.AuditStatus) models.AuditSeverity {
	// 失败的操作提升严重程度
	if status == models.AuditStatusFailed {
		return models.AuditSeverityWarning
	}

	// 管理员和安全相关操作默认较高
	if category == models.AuditCategoryAdmin || category == models.AuditCategorySecurity {
		return models.AuditSeverityCritical
	}

	// 角色模板变更
	if category == models.AuditCategoryRoleTemplate {
		switch action {
		case models.AuditActionRoleAssign, models.AuditActionRoleRevoke,
			models.AuditActionPermissionGrant, models.AuditActionPermissionRevoke:
			return models.AuditSeverityCritical
		case models.AuditActionDelete:
			return models.AuditSeverityCritical
		}
	}

	// 危险操作
	dangerousActions := map[models.AuditAction]bool{
		models.AuditActionDelete:           true,
		models.AuditActionSaltStateApply:   true,
		models.AuditActionSaltKeyDelete:    true,
		models.AuditActionClusterDeploy:    true,
		models.AuditActionK8sHelmUninstall: true,
		models.AuditActionPasswordReset:    true,
		models.AuditActionUserDelete:       true,
	}

	if dangerousActions[action] {
		return models.AuditSeverityWarning
	}

	return models.AuditSeverityInfo
}

// extractResourceID 从路径中提取资源ID
func extractResourceID(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// 跳过空字符串和常见的动作词
		if part == "" {
			continue
		}
		// 检查是否是数字ID
		if isNumeric(part) {
			return part
		}
		// 检查是否是 UUID
		if isUUID(part) {
			return part
		}
	}
	return ""
}

// isNumeric 检查字符串是否是数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isUUID 检查字符串是否是 UUID
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// AuditableHandler 可审计的处理器包装器
// 用于在需要更精细控制的地方手动触发审计
type AuditableHandler struct {
	Category     models.AuditCategory
	ResourceType string
}

// NewAuditableHandler 创建可审计处理器
func NewAuditableHandler(category models.AuditCategory, resourceType string) *AuditableHandler {
	return &AuditableHandler{
		Category:     category,
		ResourceType: resourceType,
	}
}

// Audit 手动创建审计条目
func (h *AuditableHandler) Audit(c *gin.Context, action models.AuditAction) *services.AuditEntry {
	auditService := services.GetAuditService()
	return auditService.NewAuditEntry(h.Category, action).
		WithUserFromContext(c).
		WithRequestFromContext(c)
}

// LogOperation 记录操作（简化接口）
func (h *AuditableHandler) LogOperation(c *gin.Context, action models.AuditAction, resourceID, resourceName string, oldValue, newValue interface{}, err error) {
	entry := h.Audit(c, action).
		WithResource(h.ResourceType, resourceID, resourceName).
		WithChange(oldValue, newValue, "")

	if err != nil {
		entry.WithError(err)
	}

	entry.SaveAsync()
}

// LogCriticalOperation 记录关键操作（同步保存）
func (h *AuditableHandler) LogCriticalOperation(c *gin.Context, action models.AuditAction, resourceID, resourceName string, oldValue, newValue interface{}, err error) {
	entry := h.Audit(c, action).
		WithResource(h.ResourceType, resourceID, resourceName).
		WithChange(oldValue, newValue, "").
		WithSeverity(models.AuditSeverityCritical)

	if err != nil {
		entry.WithError(err)
	}

	if saveErr := entry.Save(); saveErr != nil {
		logrus.WithError(saveErr).Error("Failed to save critical audit log")
	}
}
