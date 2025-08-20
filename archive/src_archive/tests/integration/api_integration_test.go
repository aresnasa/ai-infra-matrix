// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ansible-playbook-generator-backend/internal/models"
	"ansible-playbook-generator-backend/internal/services"
)

// IntegrationTestSuite 集成测试套件
type IntegrationTestSuite struct {
	suite.Suite
	db       *gorm.DB
	baseURL  string
	adminToken string
	userToken  string
}

// SetupSuite 设置测试套件
func (suite *IntegrationTestSuite) SetupSuite() {
	// 连接测试数据库
	dsn := "host=localhost user=test_user password=test_password dbname=ansible_generator_test port=5433 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		suite.T().Fatalf("Failed to connect to test database: %v", err)
	}
	suite.db = db
	suite.baseURL = "http://localhost:8083/api"

	// 清理并准备测试数据
	suite.setupTestData()
	
	// 获取认证token
	suite.getAuthTokens()
}

// TearDownSuite 清理测试套件
func (suite *IntegrationTestSuite) TearDownSuite() {
	// 清理测试数据
	suite.cleanupTestData()
}

// setupTestData 设置测试数据
func (suite *IntegrationTestSuite) setupTestData() {
	// 清理现有数据
	suite.db.Exec("TRUNCATE TABLE users, projects, hosts, variables, tasks RESTART IDENTITY CASCADE")
	
	// 创建测试用户
	adminUser := models.User{
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password: secret
		Role:         "admin",
		IsActive:     true,
	}
	suite.db.Create(&adminUser)

	regularUser := models.User{
		Username:     "testuser",
		Email:        "test@test.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password: secret
		Role:         "user",
		IsActive:     true,
	}
	suite.db.Create(&regularUser)

	// 创建测试项目
	testProject := models.Project{
		Name:        "Integration Test Project",
		Description: "Test project for integration tests",
		UserID:      regularUser.ID,
		Status:      "active",
	}
	suite.db.Create(&testProject)
}

// cleanupTestData 清理测试数据
func (suite *IntegrationTestSuite) cleanupTestData() {
	suite.db.Exec("TRUNCATE TABLE users, projects, hosts, variables, tasks RESTART IDENTITY CASCADE")
}

// getAuthTokens 获取认证令牌
func (suite *IntegrationTestSuite) getAuthTokens() {
	// 获取管理员token
	adminLogin := map[string]string{
		"username": "admin",
		"password": "secret",
	}
	suite.adminToken = suite.login(adminLogin)

	// 获取普通用户token
	userLogin := map[string]string{
		"username": "testuser",
		"password": "secret",
	}
	suite.userToken = suite.login(userLogin)
}

// login 登录获取token
func (suite *IntegrationTestSuite) login(credentials map[string]string) string {
	jsonData, _ := json.Marshal(credentials)
	resp, err := http.Post(suite.baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)
	
	return response["token"].(string)
}

// makeAuthenticatedRequest 发起认证请求
func (suite *IntegrationTestSuite) makeAuthenticatedRequest(method, url, token string, body []byte) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	var req *http.Request
	var err error
	
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	return client.Do(req)
}

// TestAuthenticationFlow 测试认证流程
func (suite *IntegrationTestSuite) TestAuthenticationFlow() {
	// 测试登录
	loginData := map[string]string{
		"username": "testuser",
		"password": "secret",
	}
	jsonData, _ := json.Marshal(loginData)
	
	resp, err := http.Post(suite.baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var loginResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), loginResponse, "token")
	assert.Contains(suite.T(), loginResponse, "user")
	
	// 测试获取用户信息
	token := loginResponse["token"].(string)
	resp, err = suite.makeAuthenticatedRequest("GET", suite.baseURL+"/auth/profile", token, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var profileResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&profileResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", profileResponse["username"])
}

// TestAdminUserManagement 测试管理员用户管理功能
func (suite *IntegrationTestSuite) TestAdminUserManagement() {
	// 测试获取所有用户（管理员权限）
	resp, err := suite.makeAuthenticatedRequest("GET", suite.baseURL+"/admin/users", suite.adminToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var usersResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&usersResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), usersResponse, "users")
	assert.Contains(suite.T(), usersResponse, "total")
	
	users := usersResponse["users"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(users), 2) // 至少有admin和testuser
	
	// 测试普通用户无法访问管理员接口
	resp, err = suite.makeAuthenticatedRequest("GET", suite.baseURL+"/admin/users", suite.userToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

// TestProjectManagement 测试项目管理功能
func (suite *IntegrationTestSuite) TestProjectManagement() {
	// 创建新项目
	newProject := map[string]interface{}{
		"name":        "API Test Project",
		"description": "Project created via API test",
	}
	jsonData, _ := json.Marshal(newProject)
	
	resp, err := suite.makeAuthenticatedRequest("POST", suite.baseURL+"/projects", suite.userToken, jsonData)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	assert.NoError(suite.T(), err)
	projectID := int(createResponse["id"].(float64))
	
	// 获取项目列表
	resp, err = suite.makeAuthenticatedRequest("GET", suite.baseURL+"/projects", suite.userToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var projectsResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&projectsResponse)
	assert.NoError(suite.T(), err)
	
	projects := projectsResponse["projects"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(projects), 1)
	
	// 获取项目详情
	resp, err = suite.makeAuthenticatedRequest("GET", fmt.Sprintf("%s/projects/%d", suite.baseURL, projectID), suite.userToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var projectResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&projectResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "API Test Project", projectResponse["name"])
}

// TestAdminProjectManagement 测试管理员项目管理功能
func (suite *IntegrationTestSuite) TestAdminProjectManagement() {
	// 管理员获取所有项目
	resp, err := suite.makeAuthenticatedRequest("GET", suite.baseURL+"/admin/projects", suite.adminToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var projectsResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&projectsResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), projectsResponse, "projects")
	assert.Contains(suite.T(), projectsResponse, "total")
	
	// 管理员获取系统统计
	resp, err = suite.makeAuthenticatedRequest("GET", suite.baseURL+"/admin/stats", suite.adminToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var statsResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statsResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), statsResponse, "stats")
}

// TestPlaybookGeneration 测试Playbook生成功能
func (suite *IntegrationTestSuite) TestPlaybookGeneration() {
	// 首先创建一个项目
	newProject := map[string]interface{}{
		"name":        "Playbook Test Project",
		"description": "Project for testing playbook generation",
	}
	jsonData, _ := json.Marshal(newProject)
	
	resp, err := suite.makeAuthenticatedRequest("POST", suite.baseURL+"/projects", suite.userToken, jsonData)
	assert.NoError(suite.T(), err)
	projectID := int(resp.Header.Get("Location")[len("/projects/"):]) // 假设返回Location header
	
	// 添加主机
	newHost := map[string]interface{}{
		"name":       "test-server",
		"ip_address": "192.168.1.100",
		"ssh_user":   "ubuntu",
		"ssh_port":   22,
		"variables":  map[string]interface{}{"env": "test"},
	}
	jsonData, _ = json.Marshal(newHost)
	
	resp, err = suite.makeAuthenticatedRequest("POST", fmt.Sprintf("%s/projects/%d/hosts", suite.baseURL, projectID), suite.userToken, jsonData)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	// 添加任务
	newTask := map[string]interface{}{
		"name":             "Install Nginx",
		"playbook_content": "tasks:\n  - name: Install nginx\n    apt:\n      name: nginx\n      state: present",
		"variables":        map[string]interface{}{"port": 80},
		"tags":             "web,install",
	}
	jsonData, _ = json.Marshal(newTask)
	
	resp, err = suite.makeAuthenticatedRequest("POST", fmt.Sprintf("%s/projects/%d/tasks", suite.baseURL, projectID), suite.userToken, jsonData)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	// 生成playbook
	resp, err = suite.makeAuthenticatedRequest("POST", fmt.Sprintf("%s/projects/%d/generate", suite.baseURL, projectID), suite.userToken, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var playbookResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&playbookResponse)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), playbookResponse, "playbook")
	assert.Contains(suite.T(), playbookResponse["playbook"].(string), "nginx")
}

// TestIntegrationSuite 运行集成测试套件
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
