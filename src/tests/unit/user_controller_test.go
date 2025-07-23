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
)

// MockUserService 模拟用户服务
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) GetUserByID(id uint) (*models.User, error) {
	args := m.Called(id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// TestUserController 测试用户控制器
func TestUserController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("TestGetUserProfile", func(t *testing.T) {
		// 准备测试数据
		testUser := &models.User{
			ID:        1,
			Username:  "testuser",
			Email:     "test@example.com",
			Role:      "user",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// 创建mock服务
		mockService := new(MockUserService)
		mockService.On("GetUserByID", uint(1)).Return(testUser, nil)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Next()
		})

		// 注册路由
		userController := controllers.NewUserController(nil) // 在真实环境中传入数据库连接
		router.GET("/profile", userController.GetProfile)

		// 创建请求
		req, _ := http.NewRequest("GET", "/profile", nil)
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", response["username"])
		assert.Equal(t, "test@example.com", response["email"])
	})

	t.Run("TestCreateUser", func(t *testing.T) {
		// 准备测试数据
		newUser := map[string]interface{}{
			"username": "newuser",
			"email":    "newuser@example.com",
			"password": "password123",
			"role":     "user",
		}

		// 创建mock服务
		mockService := new(MockUserService)
		mockService.On("CreateUser", mock.AnythingOfType("*models.User")).Return(nil)

		// 创建路由和控制器
		router := gin.New()
		
		// 模拟管理员认证中间件
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(1))
			c.Set("user_role", "admin")
			c.Next()
		})

		userController := controllers.NewUserController(nil)
		router.POST("/users", userController.CreateUser)

		// 创建请求
		jsonData, _ := json.Marshal(newUser)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证结果
		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})
}
