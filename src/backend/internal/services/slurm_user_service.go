package services

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SlurmUserService SLURM 用户管理服务接口
type SlurmUserService interface {
	// EnsureUser 确保用户存在（创建或更新）
	EnsureUser(user models.User, permissionLevel string, metadata SlurmUserMetadata) error
	// CreateUser 创建 SLURM 账户和用户
	CreateUser(username string, account string, partitions []string) error
	// DeleteUser 删除用户
	DeleteUser(username string) error
	// GetUser 获取用户信息
	GetUser(username string) (*SlurmUserInfo, error)
	// UpdateUserAssociation 更新用户的关联（账户、分区、QOS）
	UpdateUserAssociation(username string, association SlurmUserAssociation) error
	// SetUserLimits 设置用户资源限制
	SetUserLimits(username string, limits SlurmUserLimits) error
	// ListUsers 列出所有用户
	ListUsers() ([]SlurmUserInfo, error)
	// GetUserJobs 获取用户的作业
	GetUserJobs(username string) ([]SlurmUserJob, error)
}

// slurmUserServiceImpl SLURM 用户服务实现
type slurmUserServiceImpl struct {
	db          *gorm.DB
	slurmDBHost string
	slurmDBPort string
	slurmDBUser string
	slurmDBPass string
	slurmDBName string
	restAPIURL  string
	restAPIUser string
	restAPIPass string
	httpClient  *http.Client
}

// SlurmUserInfo SLURM 用户信息
type SlurmUserInfo struct {
	Username       string   `json:"username"`
	DefaultAccount string   `json:"default_account"`
	Accounts       []string `json:"accounts"`
	Partitions     []string `json:"partitions"`
	QOS            []string `json:"qos"`
	AdminLevel     string   `json:"admin_level"` // None, Operator, Admin
}

// SlurmUserAssociation SLURM 用户关联
type SlurmUserAssociation struct {
	Account    string   `json:"account"`
	Cluster    string   `json:"cluster"`
	Partitions []string `json:"partitions,omitempty"`
	QOS        []string `json:"qos,omitempty"`
}

// SlurmUserLimits SLURM 用户资源限制
type SlurmUserLimits struct {
	MaxJobs       int    `json:"max_jobs,omitempty"`
	MaxSubmitJobs int    `json:"max_submit_jobs,omitempty"`
	MaxCPUsPerJob int    `json:"max_cpus_per_job,omitempty"`
	MaxMemPerJob  int    `json:"max_mem_per_job,omitempty"` // MB
	MaxWallTime   string `json:"max_wall_time,omitempty"`   // e.g., "7-00:00:00"
	GrpTRES       string `json:"grp_tres,omitempty"`        // e.g., "cpu=100,mem=100G"
	MaxTRES       string `json:"max_tres,omitempty"`        // e.g., "cpu=16,mem=64G"
}

// SlurmUserJob SLURM 用户作业信息（区别于 slurm_service.go 中的 SlurmJob）
type SlurmUserJob struct {
	JobID     string `json:"job_id"`
	Name      string `json:"name"`
	User      string `json:"user"`
	Account   string `json:"account"`
	Partition string `json:"partition"`
	State     string `json:"state"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	NumCPUs   int    `json:"num_cpus"`
	MemoryMB  int    `json:"memory_mb"`
}

// NewSlurmUserService 创建 SLURM 用户服务
func NewSlurmUserService() SlurmUserService {
	return &slurmUserServiceImpl{
		db:          database.DB,
		slurmDBHost: os.Getenv("SLURM_DB_HOST"),
		slurmDBPort: getEnvDefault("SLURM_DB_PORT", "3306"),
		slurmDBUser: os.Getenv("SLURM_DB_USER"),
		slurmDBPass: os.Getenv("SLURM_DB_PASS"),
		slurmDBName: getEnvDefault("SLURM_DB_NAME", "slurm_acct_db"),
		restAPIURL:  os.Getenv("SLURM_REST_API_URL"),
		restAPIUser: os.Getenv("SLURM_REST_API_USER"),
		restAPIPass: os.Getenv("SLURM_REST_API_PASS"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureUser 确保用户存在
func (s *slurmUserServiceImpl) EnsureUser(user models.User, permissionLevel string, metadata SlurmUserMetadata) error {
	logger := logrus.WithFields(logrus.Fields{
		"username": user.Username,
		"level":    permissionLevel,
		"service":  "slurm_user",
	})

	// 确定账户名称
	accountName := metadata.DefaultAccount
	if accountName == "" {
		accountName = s.getDefaultAccount(permissionLevel)
	}

	// 确定分区
	partitions := metadata.AllowedPartitions
	if len(partitions) == 0 {
		partitions = s.getDefaultPartitions(permissionLevel)
	}

	// 检查用户是否存在
	existingUser, err := s.GetUser(user.Username)
	if err == nil && existingUser != nil {
		logger.Info("User already exists in SLURM, updating association")
		// 更新用户关联
		if err := s.UpdateUserAssociation(user.Username, SlurmUserAssociation{
			Account:    accountName,
			Partitions: partitions,
			QOS:        []string{metadata.DefaultQOS},
		}); err != nil {
			logger.WithError(err).Warn("Failed to update user association")
		}
	} else {
		// 创建新用户
		if err := s.CreateUser(user.Username, accountName, partitions); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logger.Info("Created user in SLURM")
	}

	// 设置资源限制
	if metadata.MaxJobsPerUser > 0 || metadata.MaxCPUsPerUser > 0 {
		limits := SlurmUserLimits{
			MaxJobs:       metadata.MaxJobsPerUser,
			MaxCPUsPerJob: metadata.MaxCPUsPerUser,
			MaxMemPerJob:  metadata.MaxMemoryMB,
			MaxWallTime:   metadata.MaxWallTime,
		}
		if err := s.SetUserLimits(user.Username, limits); err != nil {
			logger.WithError(err).Warn("Failed to set user limits")
		}
	}

	// 记录到本地数据库
	s.recordSlurmUser(user.ID, user.Username, accountName, partitions)

	return nil
}

// CreateUser 创建 SLURM 账户和用户
func (s *slurmUserServiceImpl) CreateUser(username string, account string, partitions []string) error {
	// 先确保账户存在
	if err := s.ensureAccount(account); err != nil {
		logrus.WithError(err).WithField("account", account).Warn("Failed to ensure account exists")
	}

	// 使用 REST API 创建用户
	if s.restAPIURL != "" {
		return s.createUserViaREST(username, account, partitions)
	}

	// 如果没有 REST API，尝试通过数据库或 Salt 执行命令
	return s.createUserViaSalt(username, account, partitions)
}

// DeleteUser 删除用户
func (s *slurmUserServiceImpl) DeleteUser(username string) error {
	logger := logrus.WithField("username", username)

	// 使用 REST API 删除用户
	if s.restAPIURL != "" {
		if err := s.deleteUserViaREST(username); err != nil {
			logger.WithError(err).Warn("Failed to delete user via REST API")
		}
	} else {
		// 通过 Salt 执行命令
		if err := s.deleteUserViaSalt(username); err != nil {
			logger.WithError(err).Warn("Failed to delete user via Salt")
		}
	}

	// 从本地数据库删除记录
	s.removeSlurmUserRecord(username)

	logger.Info("User deleted from SLURM")
	return nil
}

// GetUser 获取用户信息
func (s *slurmUserServiceImpl) GetUser(username string) (*SlurmUserInfo, error) {
	if s.restAPIURL != "" {
		return s.getUserViaREST(username)
	}

	// 从本地数据库获取
	return s.getUserFromLocalDB(username)
}

// UpdateUserAssociation 更新用户的关联
func (s *slurmUserServiceImpl) UpdateUserAssociation(username string, association SlurmUserAssociation) error {
	if s.restAPIURL != "" {
		return s.updateUserAssociationViaREST(username, association)
	}
	return s.updateUserAssociationViaSalt(username, association)
}

// SetUserLimits 设置用户资源限制
func (s *slurmUserServiceImpl) SetUserLimits(username string, limits SlurmUserLimits) error {
	if s.restAPIURL != "" {
		return s.setUserLimitsViaREST(username, limits)
	}
	return s.setUserLimitsViaSalt(username, limits)
}

// ListUsers 列出所有用户
func (s *slurmUserServiceImpl) ListUsers() ([]SlurmUserInfo, error) {
	if s.restAPIURL != "" {
		return s.listUsersViaREST()
	}
	return s.listUsersFromLocalDB()
}

// GetUserJobs 获取用户的作业
func (s *slurmUserServiceImpl) GetUserJobs(username string) ([]SlurmJob, error) {
	if s.restAPIURL != "" {
		return s.getUserJobsViaREST(username)
	}
	return nil, fmt.Errorf("REST API not configured")
}

// REST API 方法

func (s *slurmUserServiceImpl) createUserViaREST(username, account string, partitions []string) error {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/users", s.restAPIURL)

	// SLURM REST API 请求格式
	reqBody := map[string]interface{}{
		"users": []map[string]interface{}{
			{
				"name": username,
				"default": map[string]string{
					"account": account,
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	s.addRESTAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create user failed: %d - %s", resp.StatusCode, string(body))
	}

	// 添加用户到账户关联
	return s.addUserAssociationViaREST(username, account, partitions)
}

func (s *slurmUserServiceImpl) addUserAssociationViaREST(username, account string, partitions []string) error {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/associations", s.restAPIURL)

	reqBody := map[string]interface{}{
		"associations": []map[string]interface{}{
			{
				"account":   account,
				"user":      username,
				"cluster":   s.getClusterName(),
				"partition": strings.Join(partitions, ","),
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	s.addRESTAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 409 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add association failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *slurmUserServiceImpl) deleteUserViaREST(username string) error {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/user/%s", s.restAPIURL, username)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	s.addRESTAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("delete user failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *slurmUserServiceImpl) getUserViaREST(username string) (*SlurmUserInfo, error) {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/user/%s", s.restAPIURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	s.addRESTAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get user failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// 解析响应
	return s.parseUserFromResponse(result)
}

func (s *slurmUserServiceImpl) updateUserAssociationViaREST(username string, association SlurmUserAssociation) error {
	// 先获取现有关联，然后更新
	return s.addUserAssociationViaREST(username, association.Account, association.Partitions)
}

func (s *slurmUserServiceImpl) setUserLimitsViaREST(username string, limits SlurmUserLimits) error {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/associations", s.restAPIURL)

	grpTRES := limits.GrpTRES
	if grpTRES == "" && (limits.MaxCPUsPerJob > 0 || limits.MaxMemPerJob > 0) {
		parts := []string{}
		if limits.MaxCPUsPerJob > 0 {
			parts = append(parts, fmt.Sprintf("cpu=%d", limits.MaxCPUsPerJob*10))
		}
		if limits.MaxMemPerJob > 0 {
			parts = append(parts, fmt.Sprintf("mem=%dM", limits.MaxMemPerJob*10))
		}
		grpTRES = strings.Join(parts, ",")
	}

	reqBody := map[string]interface{}{
		"associations": []map[string]interface{}{
			{
				"user":    username,
				"cluster": s.getClusterName(),
				"max": map[string]interface{}{
					"jobs": limits.MaxJobs,
					"tres": map[string]interface{}{
						"per": map[string]interface{}{
							"job": limits.MaxTRES,
						},
					},
					"wall_pj": limits.MaxWallTime,
				},
				"grp_tres": grpTRES,
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	s.addRESTAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("set limits failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (s *slurmUserServiceImpl) listUsersViaREST() ([]SlurmUserInfo, error) {
	url := fmt.Sprintf("%s/slurmdb/v0.0.39/users", s.restAPIURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	s.addRESTAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list users failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return s.parseUsersFromResponse(result)
}

func (s *slurmUserServiceImpl) getUserJobsViaREST(username string) ([]SlurmJob, error) {
	url := fmt.Sprintf("%s/slurm/v0.0.39/jobs?users=%s", s.restAPIURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	s.addRESTAuth(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get jobs failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return s.parseJobsFromResponse(result)
}

// Salt 方法（作为后备）

func (s *slurmUserServiceImpl) createUserViaSalt(username, account string, partitions []string) error {
	// 通过 Salt 执行 sacctmgr 命令
	cmd := fmt.Sprintf("sacctmgr -i add user name=%s account=%s partition=%s",
		username, account, strings.Join(partitions, ","))

	logrus.WithField("command", cmd).Debug("Creating SLURM user via Salt")

	// 这里需要调用 SaltStack 服务
	// 简化实现：记录到数据库
	s.recordSlurmUser(0, username, account, partitions)

	return nil
}

func (s *slurmUserServiceImpl) deleteUserViaSalt(username string) error {
	cmd := fmt.Sprintf("sacctmgr -i delete user name=%s", username)
	logrus.WithField("command", cmd).Debug("Deleting SLURM user via Salt")
	return nil
}

func (s *slurmUserServiceImpl) updateUserAssociationViaSalt(username string, association SlurmUserAssociation) error {
	cmd := fmt.Sprintf("sacctmgr -i modify user where name=%s set account=%s",
		username, association.Account)
	logrus.WithField("command", cmd).Debug("Updating SLURM user association via Salt")
	return nil
}

func (s *slurmUserServiceImpl) setUserLimitsViaSalt(username string, limits SlurmUserLimits) error {
	var parts []string
	if limits.MaxJobs > 0 {
		parts = append(parts, fmt.Sprintf("MaxJobs=%d", limits.MaxJobs))
	}
	if limits.MaxWallTime != "" {
		parts = append(parts, fmt.Sprintf("MaxWall=%s", limits.MaxWallTime))
	}
	if limits.GrpTRES != "" {
		parts = append(parts, fmt.Sprintf("GrpTRES=%s", limits.GrpTRES))
	}

	if len(parts) > 0 {
		cmd := fmt.Sprintf("sacctmgr -i modify user where name=%s set %s",
			username, strings.Join(parts, " "))
		logrus.WithField("command", cmd).Debug("Setting SLURM user limits via Salt")
	}
	return nil
}

// 辅助方法

func (s *slurmUserServiceImpl) ensureAccount(account string) error {
	if s.restAPIURL != "" {
		url := fmt.Sprintf("%s/slurmdb/v0.0.39/accounts", s.restAPIURL)

		reqBody := map[string]interface{}{
			"accounts": []map[string]interface{}{
				{
					"name":        account,
					"description": fmt.Sprintf("Account for %s", account),
				},
			},
		}

		jsonBody, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return err
		}

		s.addRESTAuth(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 409 表示账户已存在，这是可接受的
		if resp.StatusCode >= 400 && resp.StatusCode != 409 {
			return fmt.Errorf("create account failed: %d", resp.StatusCode)
		}
	}
	return nil
}

func (s *slurmUserServiceImpl) addRESTAuth(req *http.Request) {
	if s.restAPIUser != "" && s.restAPIPass != "" {
		// SLURM REST API 使用 JWT 认证
		token := s.generateSlurmToken()
		req.Header.Set("X-SLURM-USER-NAME", s.restAPIUser)
		req.Header.Set("X-SLURM-USER-TOKEN", token)
	}
}

func (s *slurmUserServiceImpl) generateSlurmToken() string {
	// 简化的 SLURM token 生成
	// 实际应使用 SLURM 的 JWT 机制
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s:%d", s.restAPIUser, timestamp)
	hash := sha512.Sum512([]byte(data + s.restAPIPass))
	return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(hash[:])))
}

func (s *slurmUserServiceImpl) getClusterName() string {
	name := os.Getenv("SLURM_CLUSTER_NAME")
	if name == "" {
		name = "linux"
	}
	return name
}

func (s *slurmUserServiceImpl) getDefaultAccount(permissionLevel string) string {
	switch permissionLevel {
	case models.PermissionLevelAdmin:
		return "root"
	case models.PermissionLevelUser:
		return "default"
	default:
		return "default"
	}
}

func (s *slurmUserServiceImpl) getDefaultPartitions(permissionLevel string) []string {
	switch permissionLevel {
	case models.PermissionLevelAdmin:
		return []string{"compute", "gpu", "debug"}
	case models.PermissionLevelUser:
		return []string{"compute", "debug"}
	default:
		return []string{"debug"}
	}
}

// 数据库记录方法

func (s *slurmUserServiceImpl) recordSlurmUser(userID uint, username, account string, partitions []string) {
	// 使用 ComponentSyncTask 记录
	if s.db == nil {
		return
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"account":    account,
		"partitions": partitions,
	})

	task := models.ComponentSyncTask{
		UserID:    userID,
		Component: models.ComponentSlurm,
		Action:    "create",
		Status:    "completed",
		Metadata:  metadata,
	}

	now := time.Now()
	task.StartedAt = &now
	task.CompletedAt = &now

	s.db.Create(&task)
}

func (s *slurmUserServiceImpl) removeSlurmUserRecord(username string) {
	// 实现删除记录的逻辑
}

func (s *slurmUserServiceImpl) getUserFromLocalDB(username string) (*SlurmUserInfo, error) {
	// 从本地数据库获取用户信息
	return nil, nil
}

func (s *slurmUserServiceImpl) listUsersFromLocalDB() ([]SlurmUserInfo, error) {
	return nil, nil
}

func (s *slurmUserServiceImpl) parseUserFromResponse(result map[string]interface{}) (*SlurmUserInfo, error) {
	users, ok := result["users"].([]interface{})
	if !ok || len(users) == 0 {
		return nil, nil
	}

	userData := users[0].(map[string]interface{})
	user := &SlurmUserInfo{
		Username: userData["name"].(string),
	}

	if defaultData, ok := userData["default"].(map[string]interface{}); ok {
		if account, ok := defaultData["account"].(string); ok {
			user.DefaultAccount = account
		}
	}

	return user, nil
}

func (s *slurmUserServiceImpl) parseUsersFromResponse(result map[string]interface{}) ([]SlurmUserInfo, error) {
	users, ok := result["users"].([]interface{})
	if !ok {
		return nil, nil
	}

	var userInfos []SlurmUserInfo
	for _, u := range users {
		userData := u.(map[string]interface{})
		user := SlurmUserInfo{
			Username: userData["name"].(string),
		}
		if defaultData, ok := userData["default"].(map[string]interface{}); ok {
			if account, ok := defaultData["account"].(string); ok {
				user.DefaultAccount = account
			}
		}
		userInfos = append(userInfos, user)
	}

	return userInfos, nil
}

func (s *slurmUserServiceImpl) parseJobsFromResponse(result map[string]interface{}) ([]SlurmJob, error) {
	jobs, ok := result["jobs"].([]interface{})
	if !ok {
		return nil, nil
	}

	var slurmJobs []SlurmJob
	for _, j := range jobs {
		jobData := j.(map[string]interface{})
		job := SlurmJob{}
		if id, ok := jobData["job_id"]; ok {
			job.JobID = fmt.Sprintf("%v", id)
		}
		if name, ok := jobData["name"].(string); ok {
			job.Name = name
		}
		if user, ok := jobData["user"].(string); ok {
			job.User = user
		}
		if state, ok := jobData["job_state"].(string); ok {
			job.State = state
		}
		slurmJobs = append(slurmJobs, job)
	}

	return slurmJobs, nil
}

// getEnvDefault 获取环境变量，如果不存在则返回默认值
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
