package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"ansible-playbook-generator-backend/internal/controllers"
	"ansible-playbook-generator-backend/internal/models"
	"ansible-playbook-generator-backend/internal/services"
)

// MockAdminService 模拟管理员服务
type MockAdminService struct {
	mock.Mock
}

func (m *MockAdminService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockAdminService) GetAllProjects(page, limit int) ([]models.Project, int64, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]models.Project), args.Get(1).(int64), args.Error(2)
}

func (m *MockAdminService) UpdateUserStatus(userID uint, isActive bool) error {
	args := m.Called(userID, isActive)
	return args.Error(0)
}

func (m *MockAdminService) DeleteUser(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

// MockRBACService 模拟RBAC服务
type MockRBACService struct {
	mock.Mock
}

func (m *MockRBACService) CheckPermission(userID uint, resource, verb, scope, namespace string) bool {
	args := m.Called(userID, resource, verb, scope, namespace)
	return args.Bool(0)
}

func (m *MockRBACService) InitializeDefaultRBAC() error {
	args := m.Called()
	return args.Error(0)
}

// TestAdminController 测试管理员控制器
func TestAdminController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("TestGetAllUsers_AsAdmin", func(t *testing.T) {
		// 准备测试数据
		testUsers := []models.User{
			{
				ID:        1,
				Username:  "admin",
				Email:     "admin@example.com",
				Role:      "admin",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        2,
				Username:  "user1",
				Email:     "user1@example.com",
				Role:      "user",
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		// 创建mock服务
		mockRBACService := new(MockRBACService)
		mockRBACService.On("CheckPermission", uint(1), "users", "list", "*", "").Return(true)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟管理员认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Set("user_role", "admin")
			c.Next()
		})

		adminController := controllers.NewAdminController(nil) // 在真实环境中传入数据库连接
		router.GET("/admin/users", adminController.GetAllUsers)

		// 创建请求
		req, _ := http.NewRequest("GET", "/admin/users?page=1&limit=10", nil)
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "users")
		assert.Contains(t, response, "total")
	})

	t.Run("TestGetAllUsers_AsNonAdmin", func(t *testing.T) {
		// 创建mock服务
		mockRBACService := new(MockRBACService)
		mockRBACService.On("CheckPermission", uint(2), "users", "list", "*", "").Return(false)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟普通用户认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(2))
			c.Set("user_role", "user")
			c.Next()
		})

		adminController := controllers.NewAdminController(nil)
		router.GET("/admin/users", adminController.GetAllUsers)

		// 创建请求
		req, _ := http.NewRequest("GET", "/admin/users", nil)
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果 - 应该返回403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Equal(t, "权限不足", response["error"])
	})

	t.Run("TestUpdateUserStatus", func(t *testing.T) {
		// 准备测试数据
		statusUpdate := map[string]interface{}{
			"is_active": false,
		}

		// 创建mock服务
		mockRBACService := new(MockRBACService)
		mockRBACService.On("CheckPermission", uint(1), "users", "update", "*", "").Return(true)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟管理员认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Set("user_role", "admin")
			c.Next()
		})

		adminController := controllers.NewAdminController(nil)
		router.PUT("/admin/users/:id/status", adminController.UpdateUserStatus)

		// 创建请求
		jsonData, _ := json.Marshal(statusUpdate)
		req, _ := http.NewRequest("PUT", "/admin/users/2/status", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("TestDeleteUser_CannotDeleteSelf", func(t *testing.T) {
		// 创建mock服务
		mockRBACService := new(MockRBACService)
		mockRBACService.On("CheckPermission", uint(1), "users", "delete", "*", "").Return(true)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟管理员认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Set("user_role", "admin")
			c.Next()
		})

		adminController := controllers.NewAdminController(nil)
		router.DELETE("/admin/users/:id", adminController.DeleteUser)

		// 创建请求 - 尝试删除自己
		req, _ := http.NewRequest("DELETE", "/admin/users/1", nil)
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果 - 应该返回400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Equal(t, "不能删除自己", response["error"])
	})
}
