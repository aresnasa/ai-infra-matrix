package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ClusterPermissionMiddleware 集群权限检查中间件
type ClusterPermissionMiddleware struct {
	service *services.ClusterPermissionService
	log     *logrus.Logger
}

// NewClusterPermissionMiddleware 创建集群权限中间件
func NewClusterPermissionMiddleware() *ClusterPermissionMiddleware {
	return &ClusterPermissionMiddleware{
		service: services.NewClusterPermissionService(database.DB),
		log:     logrus.StandardLogger(),
	}
}

// RequireSlurmAccess 检查SLURM集群访问权限
// clusterIDParam: 从URL参数或查询参数中获取集群ID的参数名
// verb: 需要的权限动作
// partitionParam: 可选的分区参数名
func (m *ClusterPermissionMiddleware) RequireSlurmAccess(clusterIDParam string, verb models.ClusterPermissionVerb, partitionParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// 获取用户角色，管理员跳过检查
		if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
			c.Next()
			return
		}

		// 获取集群ID
		var clusterID uint64
		var err error

		// 尝试从路径参数获取
		clusterIDStr := c.Param(clusterIDParam)
		if clusterIDStr == "" {
			// 尝试从查询参数获取
			clusterIDStr = c.Query(clusterIDParam)
		}
		if clusterIDStr == "" {
			// 尝试从请求体获取（需要绑定）
			var body struct {
				ClusterID uint `json:"cluster_id"`
			}
			if err := c.ShouldBindJSON(&body); err == nil && body.ClusterID > 0 {
				clusterID = uint64(body.ClusterID)
			}
		} else {
			clusterID, err = strconv.ParseUint(clusterIDStr, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cluster ID"})
				c.Abort()
				return
			}
		}

		if clusterID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cluster ID is required"})
			c.Abort()
			return
		}

		// 获取分区（可选）
		partition := ""
		if partitionParam != "" {
			partition = c.Param(partitionParam)
			if partition == "" {
				partition = c.Query(partitionParam)
			}
		}

		// 检查权限
		result, err := m.service.CheckSlurmAccess(c.Request.Context(), userID.(uint), uint(clusterID), verb, partition)
		if err != nil {
			m.log.WithError(err).Error("Failed to check SLURM access")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
			c.Abort()
			return
		}

		if !result.Allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Access denied",
				"reason":        result.Reason,
				"missing_verbs": result.MissingVerbs,
				"required_verb": verb,
				"cluster_id":    clusterID,
				"partition":     partition,
			})
			c.Abort()
			return
		}

		// 将资源限制传递给后续处理器
		if result.ResourceLimits != nil {
			c.Set("resource_limits", result.ResourceLimits)
		}

		c.Next()
	}
}

// RequireSaltstackAccess 检查SaltStack集群访问权限
// masterIDParam: 从URL参数或查询参数中获取Master ID的参数名
// verb: 需要的权限动作
// minionIDParam: 可选的Minion ID参数名
// functionParam: 可选的Salt函数参数名
func (m *ClusterPermissionMiddleware) RequireSaltstackAccess(masterIDParam string, verb models.ClusterPermissionVerb, minionIDParam, functionParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// 获取用户角色，管理员跳过检查
		if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
			c.Next()
			return
		}

		// 获取Master ID
		masterID := c.Param(masterIDParam)
		if masterID == "" {
			masterID = c.Query(masterIDParam)
		}
		if masterID == "" {
			// 使用默认的master ID
			masterID = "default"
		}

		// 获取Minion ID（可选）
		minionID := ""
		if minionIDParam != "" {
			minionID = c.Param(minionIDParam)
			if minionID == "" {
				minionID = c.Query(minionIDParam)
			}
		}

		// 获取函数名（可选）
		function := ""
		if functionParam != "" {
			function = c.Param(functionParam)
			if function == "" {
				function = c.Query(functionParam)
			}
		}

		// 检查权限
		result, err := m.service.CheckSaltstackAccess(c.Request.Context(), userID.(uint), masterID, verb, minionID, function)
		if err != nil {
			m.log.WithError(err).Error("Failed to check SaltStack access")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
			c.Abort()
			return
		}

		if !result.Allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Access denied",
				"reason":        result.Reason,
				"missing_verbs": result.MissingVerbs,
				"required_verb": verb,
				"master_id":     masterID,
				"minion_id":     minionID,
				"function":      function,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckSlurmPartitions 动态检查SLURM分区权限
// 当需要检查多个分区时使用
func (m *ClusterPermissionMiddleware) CheckSlurmPartitions(userID uint, clusterID uint, partitions []string) ([]string, []string) {
	allowedPartitions := []string{}
	deniedPartitions := []string{}

	for _, partition := range partitions {
		result, err := m.service.CheckSlurmAccess(nil, userID, clusterID, models.VerbSubmit, partition)
		if err != nil {
			m.log.WithError(err).WithField("partition", partition).Error("Failed to check partition access")
			deniedPartitions = append(deniedPartitions, partition)
			continue
		}

		if result.Allowed {
			allowedPartitions = append(allowedPartitions, partition)
		} else {
			deniedPartitions = append(deniedPartitions, partition)
		}
	}

	return allowedPartitions, deniedPartitions
}

// CheckSaltstackMinions 动态检查SaltStack Minion权限
// 当需要检查多个Minion时使用
func (m *ClusterPermissionMiddleware) CheckSaltstackMinions(userID uint, masterID string, minions []string) ([]string, []string) {
	allowedMinions := []string{}
	deniedMinions := []string{}

	for _, minion := range minions {
		result, err := m.service.CheckSaltstackAccess(nil, userID, masterID, models.VerbExecute, minion, "")
		if err != nil {
			m.log.WithError(err).WithField("minion", minion).Error("Failed to check minion access")
			deniedMinions = append(deniedMinions, minion)
			continue
		}

		if result.Allowed {
			allowedMinions = append(allowedMinions, minion)
		} else {
			deniedMinions = append(deniedMinions, minion)
		}
	}

	return allowedMinions, deniedMinions
}

// FilterSlurmClusters 过滤用户可访问的SLURM集群列表
func (m *ClusterPermissionMiddleware) FilterSlurmClusters(userID uint, clusterIDs []uint) []uint {
	accessibleClusters := []uint{}

	for _, clusterID := range clusterIDs {
		result, err := m.service.CheckSlurmAccess(nil, userID, clusterID, models.VerbView, "")
		if err != nil {
			m.log.WithError(err).WithField("cluster_id", clusterID).Error("Failed to check cluster access")
			continue
		}

		if result.Allowed {
			accessibleClusters = append(accessibleClusters, clusterID)
		}
	}

	return accessibleClusters
}

// FilterSaltstackMasters 过滤用户可访问的SaltStack Master列表
func (m *ClusterPermissionMiddleware) FilterSaltstackMasters(userID uint, masterIDs []string) []string {
	accessibleMasters := []string{}

	for _, masterID := range masterIDs {
		result, err := m.service.CheckSaltstackAccess(nil, userID, masterID, models.VerbView, "", "")
		if err != nil {
			m.log.WithError(err).WithField("master_id", masterID).Error("Failed to check master access")
			continue
		}

		if result.Allowed {
			accessibleMasters = append(accessibleMasters, masterID)
		}
	}

	return accessibleMasters
}

// RequireClusterPermissionWithConfig 配置化的权限检查中间件
// 支持从请求中动态获取各种参数
type ClusterPermissionConfig struct {
	ClusterType     string // "slurm" 或 "saltstack"
	RequiredVerb    models.ClusterPermissionVerb
	ClusterIDSource string // 参数来源: "path", "query", "body"
	ClusterIDParam  string // 参数名
	PartitionParam  string // 分区参数名（SLURM）
	MinionParam     string // Minion参数名（SaltStack）
	FunctionParam   string // 函数参数名（SaltStack）
	AllowAdmin      bool   // 是否允许管理员跳过检查
}

// RequireClusterPermission 通用的集群权限检查
func (m *ClusterPermissionMiddleware) RequireClusterPermission(config ClusterPermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// 检查管理员权限
		if config.AllowAdmin {
			if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
				c.Next()
				return
			}
		}

		// 根据配置的集群类型进行权限检查
		switch strings.ToLower(config.ClusterType) {
		case "slurm":
			m.checkSlurmPermission(c, userID.(uint), config)
		case "saltstack", "salt":
			m.checkSaltstackPermission(c, userID.(uint), config)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cluster type"})
			c.Abort()
		}
	}
}

func (m *ClusterPermissionMiddleware) checkSlurmPermission(c *gin.Context, userID uint, config ClusterPermissionConfig) {
	// 获取集群ID
	var clusterIDStr string
	switch config.ClusterIDSource {
	case "path":
		clusterIDStr = c.Param(config.ClusterIDParam)
	case "query":
		clusterIDStr = c.Query(config.ClusterIDParam)
	case "body":
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err == nil {
			if id, ok := body[config.ClusterIDParam]; ok {
				clusterIDStr = toString(id)
			}
		}
	default:
		clusterIDStr = c.Param(config.ClusterIDParam)
		if clusterIDStr == "" {
			clusterIDStr = c.Query(config.ClusterIDParam)
		}
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil || clusterID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing cluster ID"})
		c.Abort()
		return
	}

	// 获取分区（可选）
	partition := ""
	if config.PartitionParam != "" {
		partition = c.Param(config.PartitionParam)
		if partition == "" {
			partition = c.Query(config.PartitionParam)
		}
	}

	// 检查权限
	result, err := m.service.CheckSlurmAccess(c.Request.Context(), userID, uint(clusterID), config.RequiredVerb, partition)
	if err != nil {
		m.log.WithError(err).Error("Failed to check SLURM access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		c.Abort()
		return
	}

	if !result.Allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "clusterPermission.slurmAccessDenied",
			"reason":        result.Reason,
			"cluster_id":    clusterID,
			"partition":     partition,
			"required_verb": string(config.RequiredVerb),
		})
		c.Abort()
		return
	}

	// 保存资源限制到上下文
	if result.ResourceLimits != nil {
		c.Set("slurm_resource_limits", result.ResourceLimits)
	}

	c.Next()
}

func (m *ClusterPermissionMiddleware) checkSaltstackPermission(c *gin.Context, userID uint, config ClusterPermissionConfig) {
	// 获取Master ID
	masterID := "default"
	switch config.ClusterIDSource {
	case "path":
		if id := c.Param(config.ClusterIDParam); id != "" {
			masterID = id
		}
	case "query":
		if id := c.Query(config.ClusterIDParam); id != "" {
			masterID = id
		}
	default:
		if id := c.Param(config.ClusterIDParam); id != "" {
			masterID = id
		} else if id := c.Query(config.ClusterIDParam); id != "" {
			masterID = id
		}
	}

	// 获取Minion ID（可选）
	minionID := ""
	if config.MinionParam != "" {
		minionID = c.Param(config.MinionParam)
		if minionID == "" {
			minionID = c.Query(config.MinionParam)
		}
	}

	// 获取函数名（可选）
	function := ""
	if config.FunctionParam != "" {
		function = c.Param(config.FunctionParam)
		if function == "" {
			function = c.Query(config.FunctionParam)
		}
	}

	// 检查权限
	result, err := m.service.CheckSaltstackAccess(c.Request.Context(), userID, masterID, config.RequiredVerb, minionID, function)
	if err != nil {
		m.log.WithError(err).Error("Failed to check SaltStack access")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		c.Abort()
		return
	}

	if !result.Allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"error":         "clusterPermission.saltstackAccessDenied",
			"reason":        result.Reason,
			"master_id":     masterID,
			"minion_id":     minionID,
			"function":      function,
			"required_verb": string(config.RequiredVerb),
		})
		c.Abort()
		return
	}

	c.Next()
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', 0, 64)
	case int:
		return strconv.Itoa(val)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	default:
		return ""
	}
}
