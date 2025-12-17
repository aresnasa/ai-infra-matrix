package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// JobState 作业状态枚举
type JobState string

const (
	JobStatePending   JobState = "pending"   // 待执行
	JobStateRunning   JobState = "running"   // 执行中
	JobStateCompleted JobState = "completed" // 已完成
	JobStateFailed    JobState = "failed"    // 失败
	JobStateTimeout   JobState = "timeout"   // 超时
	JobStatePartial   JobState = "partial"   // 部分成功
)

// JobStateTransition 状态转换规则
var validTransitions = map[JobState][]JobState{
	JobStatePending:   {JobStateRunning, JobStateFailed, JobStateTimeout},
	JobStateRunning:   {JobStateCompleted, JobStateFailed, JobStateTimeout, JobStatePartial},
	JobStateCompleted: {}, // 终态，不可转换
	JobStateFailed:    {}, // 终态，不可转换
	JobStateTimeout:   {}, // 终态，不可转换
	JobStatePartial:   {}, // 终态，不可转换
}

// IsValidTransition 检查状态转换是否合法
func IsValidTransition(from, to JobState) bool {
	validTargets, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, target := range validTargets {
		if target == to {
			return true
		}
	}
	return false
}

// IsFinalState 检查是否为终态
func IsFinalState(state JobState) bool {
	return state == JobStateCompleted || state == JobStateFailed || state == JobStateTimeout || state == JobStatePartial
}

// SaltJobWatcher 作业状态监控器
// 负责监控运行中的作业，通过 Salt API 查询实际状态并更新数据库
type SaltJobWatcher struct {
	db            *gorm.DB
	cache         *redis.Client
	httpClient    *http.Client
	saltMasterURL string
	saltUsername  string
	saltPassword  string
	saltEauth     string
	authToken     string
	tokenExpiry   time.Time
	tokenMu       sync.Mutex
	watchInterval time.Duration          // 监控间隔
	jobTimeout    time.Duration          // 作业超时时间
	stopChan      chan struct{}          // 停止信号
	jobCallbacks  map[string]JobCallback // JID -> 回调函数
	callbackMu    sync.RWMutex
	running       bool
	runningMu     sync.Mutex
}

// JobCallback 作业完成回调函数类型
type JobCallback func(jid string, status JobState, result map[string]interface{})

// JobUpdateEvent 作业状态更新事件
type JobUpdateEvent struct {
	JID          string                 `json:"jid"`
	TaskID       string                 `json:"task_id"`
	OldStatus    string                 `json:"old_status"`
	NewStatus    string                 `json:"new_status"`
	Result       map[string]interface{} `json:"result,omitempty"`
	SuccessCount int                    `json:"success_count"`
	FailedCount  int                    `json:"failed_count"`
	Duration     int64                  `json:"duration_ms"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// NewSaltJobWatcher 创建作业状态监控器
func NewSaltJobWatcher(db *gorm.DB, cache *redis.Client) *SaltJobWatcher {
	// 从环境变量读取 Salt API 配置
	masterURL := os.Getenv("SALTSTACK_MASTER_URL")
	if masterURL == "" {
		scheme := os.Getenv("SALT_API_SCHEME")
		if scheme == "" {
			scheme = "http"
		}
		host := os.Getenv("SALT_MASTER_HOST")
		if host == "" {
			host = "saltstack"
		}
		port := os.Getenv("SALT_API_PORT")
		if port == "" {
			port = "8002"
		}
		masterURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	return &SaltJobWatcher{
		db:            db,
		cache:         cache,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		saltMasterURL: masterURL,
		saltUsername:  username,
		saltPassword:  password,
		saltEauth:     eauth,
		watchInterval: 3 * time.Second,  // 每3秒检查一次
		jobTimeout:    10 * time.Minute, // 作业超时时间10分钟
		stopChan:      make(chan struct{}),
		jobCallbacks:  make(map[string]JobCallback),
	}
}

// Start 启动监控器
func (w *SaltJobWatcher) Start() {
	w.runningMu.Lock()
	if w.running {
		w.runningMu.Unlock()
		return
	}
	w.running = true
	w.runningMu.Unlock()

	log.Printf("[SaltJobWatcher] 作业状态监控器已启动，监控间隔: %v, 超时: %v", w.watchInterval, w.jobTimeout)
	go w.watchLoop()
}

// Stop 停止监控器
func (w *SaltJobWatcher) Stop() {
	w.runningMu.Lock()
	if !w.running {
		w.runningMu.Unlock()
		return
	}
	w.running = false
	w.runningMu.Unlock()

	close(w.stopChan)
	log.Printf("[SaltJobWatcher] 作业状态监控器已停止")
}

// RegisterCallback 注册作业完成回调
func (w *SaltJobWatcher) RegisterCallback(jid string, callback JobCallback) {
	w.callbackMu.Lock()
	defer w.callbackMu.Unlock()
	w.jobCallbacks[jid] = callback
	log.Printf("[SaltJobWatcher] 已注册作业 %s 的回调", jid)
}

// UnregisterCallback 取消注册回调
func (w *SaltJobWatcher) UnregisterCallback(jid string) {
	w.callbackMu.Lock()
	defer w.callbackMu.Unlock()
	delete(w.jobCallbacks, jid)
}

// watchLoop 监控循环
func (w *SaltJobWatcher) watchLoop() {
	ticker := time.NewTicker(w.watchInterval)
	defer ticker.Stop()

	// 首次立即检查
	w.checkRunningJobs()

	for {
		select {
		case <-ticker.C:
			w.checkRunningJobs()
		case <-w.stopChan:
			return
		}
	}
}

// checkRunningJobs 检查所有运行中的作业
func (w *SaltJobWatcher) checkRunningJobs() {
	ctx := context.Background()

	// 查询所有 running 状态的作业
	var runningJobs []models.SaltJob
	if err := w.db.WithContext(ctx).
		Where("status = ?", "running").
		Order("start_time ASC").
		Limit(100). // 每次最多处理100个
		Find(&runningJobs).Error; err != nil {
		log.Printf("[SaltJobWatcher] 查询运行中作业失败: %v", err)
		return
	}

	if len(runningJobs) == 0 {
		return
	}

	log.Printf("[SaltJobWatcher] 发现 %d 个运行中的作业，开始检查状态...", len(runningJobs))

	// 并发检查作业状态（限制并发数为10）
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for _, job := range runningJobs {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(j models.SaltJob) {
			defer wg.Done()
			defer func() { <-semaphore }()

			w.checkAndUpdateJob(ctx, &j)
		}(job)
	}

	wg.Wait()
}

// checkAndUpdateJob 检查并更新单个作业状态
func (w *SaltJobWatcher) checkAndUpdateJob(ctx context.Context, job *models.SaltJob) {
	// 检查是否超时
	if time.Since(job.StartTime) > w.jobTimeout {
		log.Printf("[SaltJobWatcher] 作业 %s 已超时（开始于 %v）", job.JID, job.StartTime)
		w.updateJobState(ctx, job, JobStateTimeout, nil, 0, 0)
		return
	}

	// 通过 Salt API 查询作业结果
	result, err := w.lookupJobResult(ctx, job.JID)
	if err != nil {
		log.Printf("[SaltJobWatcher] 查询作业 %s 失败: %v", job.JID, err)
		return
	}

	// 如果有结果，说明作业已完成
	if len(result) > 0 {
		successCount, failedCount := w.analyzeJobResult(result)
		var newState JobState

		if failedCount > 0 && successCount == 0 {
			newState = JobStateFailed
		} else if failedCount > 0 {
			newState = JobStatePartial
		} else {
			newState = JobStateCompleted
		}

		log.Printf("[SaltJobWatcher] 作业 %s 已完成: 状态=%s, 成功=%d, 失败=%d",
			job.JID, newState, successCount, failedCount)
		w.updateJobState(ctx, job, newState, result, successCount, failedCount)
	}
}

// updateJobState 更新作业状态（触发式状态机）
func (w *SaltJobWatcher) updateJobState(ctx context.Context, job *models.SaltJob, newState JobState, result map[string]interface{}, successCount, failedCount int) {
	oldState := JobState(job.Status)

	// 检查状态转换是否合法
	if !IsValidTransition(oldState, newState) {
		log.Printf("[SaltJobWatcher] 非法状态转换: %s -> %s (JID=%s)", oldState, newState, job.JID)
		return
	}

	// 计算执行时长
	duration := time.Since(job.StartTime).Milliseconds()
	endTime := time.Now()

	// 构建更新数据
	updates := map[string]interface{}{
		"status":        string(newState),
		"success_count": successCount,
		"failed_count":  failedCount,
		"duration":      duration,
		"end_time":      endTime,
		"updated_at":    time.Now(),
	}

	if result != nil {
		resultJSON, _ := json.Marshal(result)
		updates["result"] = resultJSON
	}

	// 更新数据库
	if err := w.db.WithContext(ctx).Model(&models.SaltJob{}).
		Where("jid = ? AND status = ?", job.JID, string(oldState)). // 乐观锁
		Updates(updates).Error; err != nil {
		log.Printf("[SaltJobWatcher] 更新作业 %s 状态失败: %v", job.JID, err)
		return
	}

	log.Printf("[SaltJobWatcher] 作业状态已更新: JID=%s, %s -> %s, 时长=%dms",
		job.JID, oldState, newState, duration)

	// 更新 Redis 缓存
	if w.cache != nil {
		jobDetailKey := fmt.Sprintf("saltstack:job_detail:%s", job.JID)
		if jobInfoJSON, err := w.cache.Get(ctx, jobDetailKey).Result(); err == nil {
			var jobInfo map[string]interface{}
			if json.Unmarshal([]byte(jobInfoJSON), &jobInfo) == nil {
				jobInfo["status"] = string(newState)
				jobInfo["success_count"] = successCount
				jobInfo["failed_count"] = failedCount
				jobInfo["duration_ms"] = duration
				jobInfo["end_time"] = endTime.Format(time.RFC3339)
				if newJSON, err := json.Marshal(jobInfo); err == nil {
					w.cache.Set(ctx, jobDetailKey, string(newJSON), 7*24*time.Hour)
				}
			}
		}
	}

	// 触发回调
	w.callbackMu.RLock()
	callback, exists := w.jobCallbacks[job.JID]
	w.callbackMu.RUnlock()

	if exists {
		go func() {
			callback(job.JID, newState, result)
			w.UnregisterCallback(job.JID)
		}()
	}

	// 发布状态更新事件（可用于 WebSocket 推送）
	w.publishStatusUpdate(ctx, &JobUpdateEvent{
		JID:          job.JID,
		TaskID:       job.TaskID,
		OldStatus:    string(oldState),
		NewStatus:    string(newState),
		Result:       result,
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Duration:     duration,
		UpdatedAt:    time.Now(),
	})
}

// publishStatusUpdate 发布状态更新事件到 Redis（供 WebSocket 订阅）
func (w *SaltJobWatcher) publishStatusUpdate(ctx context.Context, event *JobUpdateEvent) {
	if w.cache == nil {
		return
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	// 发布到 Redis pub/sub channel
	channelName := "saltstack:job_status_updates"
	w.cache.Publish(ctx, channelName, string(eventJSON))

	// 同时存储到作业特定的 key（供 HTTP 轮询查询）
	statusKey := fmt.Sprintf("saltstack:job_status:%s", event.JID)
	w.cache.Set(ctx, statusKey, string(eventJSON), 1*time.Hour)

	// 如果有 TaskID，也设置 TaskID 的状态 key
	if event.TaskID != "" {
		taskStatusKey := fmt.Sprintf("saltstack:task_status:%s", event.TaskID)
		w.cache.Set(ctx, taskStatusKey, string(eventJSON), 1*time.Hour)
	}
}

// analyzeJobResult 分析作业结果，统计成功/失败数量
func (w *SaltJobWatcher) analyzeJobResult(result map[string]interface{}) (successCount, failedCount int) {
	for minionID, minionResult := range result {
		_ = minionID // 避免编译警告

		// 检查返回码
		if resultMap, ok := minionResult.(map[string]interface{}); ok {
			if retcode, ok := resultMap["retcode"].(float64); ok {
				if retcode == 0 {
					successCount++
				} else {
					failedCount++
				}
				continue
			}
		}

		// 检查是否为错误字符串
		if resultStr, ok := minionResult.(string); ok {
			if len(resultStr) > 0 && (resultStr[0] == 'E' || resultStr[0] == 'e') {
				// 可能是错误信息
				failedCount++
				continue
			}
		}

		// 默认假设成功
		successCount++
	}
	return
}

// lookupJobResult 查询作业结果
func (w *SaltJobWatcher) lookupJobResult(ctx context.Context, jid string) (map[string]interface{}, error) {
	// 确保有有效的认证 token
	if err := w.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("认证失败: %v", err)
	}

	// 构建请求
	payload := map[string]interface{}{
		"client": "runner",
		"fun":    "jobs.lookup_jid",
		"kwarg": map[string]interface{}{
			"jid": jid,
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", w.saltMasterURL+"/", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", w.authToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 解析响应: {"return": [{"minion1": result1, "minion2": result2}]}
	if ret, ok := result["return"].([]interface{}); ok && len(ret) > 0 {
		if m, ok := ret[0].(map[string]interface{}); ok {
			return m, nil
		}
	}

	return nil, nil
}

// ensureAuthenticated 确保已认证
func (w *SaltJobWatcher) ensureAuthenticated(ctx context.Context) error {
	w.tokenMu.Lock()
	defer w.tokenMu.Unlock()

	// 检查 token 是否有效
	if w.authToken != "" && time.Now().Before(w.tokenExpiry) {
		return nil
	}

	// 登录获取新 token
	loginPayload := map[string]interface{}{
		"username": w.saltUsername,
		"password": w.saltPassword,
		"eauth":    w.saltEauth,
	}

	payloadBytes, _ := json.Marshal(loginPayload)
	req, err := http.NewRequestWithContext(ctx, "POST", w.saltMasterURL+"/login", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败: %s", string(body))
	}

	var loginResp map[string]interface{}
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return err
	}

	// 提取 token
	if returnData, ok := loginResp["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if token, ok := tokenData["token"].(string); ok {
				w.authToken = token
				// token 有效期设为11小时（Salt API 默认12小时过期）
				w.tokenExpiry = time.Now().Add(11 * time.Hour)
				return nil
			}
		}
	}

	return fmt.Errorf("无法从响应中提取 token")
}

// ForceCheckJob 强制检查指定作业状态（用于手动触发）
func (w *SaltJobWatcher) ForceCheckJob(ctx context.Context, jid string) error {
	var job models.SaltJob
	if err := w.db.WithContext(ctx).Where("jid = ?", jid).First(&job).Error; err != nil {
		return fmt.Errorf("作业不存在: %v", err)
	}

	if IsFinalState(JobState(job.Status)) {
		return fmt.Errorf("作业已处于终态: %s", job.Status)
	}

	w.checkAndUpdateJob(ctx, &job)
	return nil
}

// GetJobStatus 获取作业当前状态（优先从缓存）
func (w *SaltJobWatcher) GetJobStatus(ctx context.Context, jid string) (*JobUpdateEvent, error) {
	// 先尝试从 Redis 获取最新状态
	if w.cache != nil {
		statusKey := fmt.Sprintf("saltstack:job_status:%s", jid)
		if statusJSON, err := w.cache.Get(ctx, statusKey).Result(); err == nil {
			var event JobUpdateEvent
			if json.Unmarshal([]byte(statusJSON), &event) == nil {
				return &event, nil
			}
		}
	}

	// 从数据库获取
	var job models.SaltJob
	if err := w.db.WithContext(ctx).Where("jid = ?", jid).First(&job).Error; err != nil {
		return nil, err
	}

	return &JobUpdateEvent{
		JID:          job.JID,
		TaskID:       job.TaskID,
		NewStatus:    job.Status,
		SuccessCount: job.SuccessCount,
		FailedCount:  job.FailedCount,
		Duration:     job.Duration,
		UpdatedAt:    job.UpdatedAt,
	}, nil
}

// WaitForCompletion 等待作业完成（阻塞式，带超时）
func (w *SaltJobWatcher) WaitForCompletion(ctx context.Context, jid string, timeout time.Duration) (*JobUpdateEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			event, err := w.GetJobStatus(ctx, jid)
			if err != nil {
				continue
			}
			if IsFinalState(JobState(event.NewStatus)) {
				return event, nil
			}
		}
	}
}
