package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Slurm service provides lightweight access to Slurm cluster metrics by shelling out
// to common CLI tools (sinfo, squeue). This implementation is strict: it requires
// real tools/data and no longer falls back to demo data.

type SlurmService struct {
	db            *gorm.DB
	taskDB        *gorm.DB
	restAPIURL    string
	restAPIToken  string
	httpClient    *http.Client
	useSlurmrestd bool // 是否使用 slurmrestd API (true) 还是 SSH (false)
}

const (
	defaultSlurmClusterID          = 1
	autoRegisterMaxAttempts        = 3
	defaultAutoRegisterDelaySecond = 20
	autoRegisterRetrySecond        = 10
)

func NewSlurmService() *SlurmService {
	restAPIURL := os.Getenv("SLURM_REST_API_URL")
	if restAPIURL == "" {
		restAPIURL = "http://slurm-master:6820" // 默认URL
	}

	// 从环境变量读取是否使用 slurmrestd
	useSlurmrestd := os.Getenv("USE_SLURMRESTD") == "true"

	return &SlurmService{
		restAPIURL:    restAPIURL,
		useSlurmrestd: useSlurmrestd,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewSlurmServiceWithDB(db *gorm.DB) *SlurmService {
	return NewSlurmServiceWithStores(db, db)
}

func NewSlurmServiceWithStores(clusterDB, taskDB *gorm.DB) *SlurmService {
	restAPIURL := os.Getenv("SLURM_REST_API_URL")
	if restAPIURL == "" {
		restAPIURL = "http://slurm-master:6820" // 默认URL
	}

	// 从环境变量读取是否使用 slurmrestd
	useSlurmrestd := os.Getenv("USE_SLURMRESTD") == "true"

	return &SlurmService{
		db:            clusterDB,
		taskDB:        taskDB,
		restAPIURL:    restAPIURL,
		useSlurmrestd: useSlurmrestd,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SlurmService) taskStore() *gorm.DB {
	if s.taskDB != nil {
		return s.taskDB
	}
	return s.db
}

// GetUseSlurmRestd 返回是否启用 slurmrestd REST API
func (s *SlurmService) GetUseSlurmRestd() bool {
	return s.useSlurmrestd
}

func (s *SlurmService) GetSummary(ctx context.Context) (*SlurmSummary, error) {
	// Gather from sinfo/squeue; if tools are unavailable, return empty summary
	nodesTotal, nodesIdle, nodesAlloc, partitions, demo1 := s.getNodeStats(ctx)
	jobsRun, jobsPend, jobsOther, demo2 := s.getJobStats(ctx)

	log.Printf("[DEBUG] GetSummary: nodesTotal=%d, nodesIdle=%d, nodesAlloc=%d, demo=%v",
		nodesTotal, nodesIdle, nodesAlloc, demo1)

	// Return empty summary instead of error when tools are unavailable
	return &SlurmSummary{
		NodesTotal:  nodesTotal,
		NodesIdle:   nodesIdle,
		NodesAlloc:  nodesAlloc,
		Partitions:  partitions,
		JobsRunning: jobsRun,
		JobsPending: jobsPend,
		JobsOther:   jobsOther,
		Demo:        demo1 || demo2,
		GeneratedAt: time.Now(),
	}, nil
}

func (s *SlurmService) GetNodes(ctx context.Context) ([]SlurmNode, bool, error) {

	var dbNodes []models.SlurmNode
	if s.db != nil {
		if err := s.db.Find(&dbNodes).Error; err != nil {
			log.Printf("[WARN] 无法从数据库加载SLURM节点信息: %v", err)
		}
	}

	// 通过SSH执行sinfo命令获取节点信息
	// sinfo format: NodeName|State|CPUS(A/I/O/T)|Memory|Partition
	output, err := s.executeSlurmCommand(ctx, "sinfo -N -o '%N|%T|%C|%m|%P'")
	if err != nil {
		if len(dbNodes) > 0 {
			return mergeNodesWithDB(nil, dbNodes), false, nil
		}
		// No SSH connection and no DB fallback: return empty list with demo flag
		return []SlurmNode{}, true, nil
	}

	var cliNodes []SlurmNode
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// skip header if present
		if first && strings.Contains(line, "NODELIST") {
			first = false
			continue
		}
		first = false
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			// try to be lenient
			for len(parts) < 5 {
				parts = append(parts, "")
			}
		}
		cliNodes = append(cliNodes, SlurmNode{
			Name:      parts[0],
			State:     parts[1],
			CPUs:      parseCPUs(parts[2]),
			MemoryMB:  parts[3],
			Partition: parts[4],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	if len(dbNodes) == 0 {
		return cliNodes, false, nil
	}

	return mergeNodesWithDB(cliNodes, dbNodes), false, nil
}

func mergeNodesWithDB(cliNodes []SlurmNode, dbNodes []models.SlurmNode) []SlurmNode {
	if len(dbNodes) == 0 {
		return append([]SlurmNode{}, cliNodes...)
	}

	if len(cliNodes) == 0 {
		merged := make([]SlurmNode, 0, len(dbNodes))
		for _, dbNode := range dbNodes {
			merged = append(merged, buildNodeFromDB(dbNode))
		}
		return merged
	}

	merged := make([]SlurmNode, 0, len(cliNodes)+len(dbNodes))
	dbByID := make(map[uint]models.SlurmNode, len(dbNodes))
	nameIndex := make(map[string]uint, len(dbNodes)*2)
	for _, dbNode := range dbNodes {
		dbByID[dbNode.ID] = dbNode
		normalizedName := normalizeNodeNameValue(dbNode.NodeName, dbNode.Host, dbNode.ID)
		for _, key := range []string{dbNode.NodeName, dbNode.Host, dbNode.SaltMinionID, normalizedName} {
			if key == "" {
				continue
			}
			lower := strings.ToLower(key)
			if _, exists := nameIndex[lower]; !exists {
				nameIndex[lower] = dbNode.ID
			}
		}
	}

	used := make(map[uint]bool, len(dbNodes))
	for _, cliNode := range cliNodes {
		mergedNode := cliNode
		if id, ok := nameIndex[strings.ToLower(cliNode.Name)]; ok {
			used[id] = true
			dbNode := dbByID[id]
			if mergedNode.State == "" {
				mergedNode.State = dbNode.Status
			}
			if mergedNode.CPUs == "" {
				mergedNode.CPUs = safeIntString(dbNode.CPUs)
			}
			if mergedNode.MemoryMB == "" {
				mergedNode.MemoryMB = safeIntString(dbNode.Memory)
			}
			if mergedNode.Partition == "" {
				mergedNode.Partition = derivePartition(dbNode)
			}
		}
		merged = append(merged, mergedNode)
	}

	for _, dbNode := range dbNodes {
		if used[dbNode.ID] {
			continue
		}
		merged = append(merged, buildNodeFromDB(dbNode))
	}

	return merged
}

func buildNodeFromDB(dbNode models.SlurmNode) SlurmNode {
	name := normalizeNodeNameValue(dbNode.NodeName, dbNode.Host, dbNode.ID)
	return SlurmNode{
		Name:      name,
		State:     safeState(dbNode.Status),
		CPUs:      safeIntString(dbNode.CPUs),
		MemoryMB:  safeIntString(dbNode.Memory),
		Partition: derivePartition(dbNode),
	}
}

func derivePartition(dbNode models.SlurmNode) string {
	if len(dbNode.NodeConfig.Partitions) > 0 && dbNode.NodeConfig.Partitions[0] != "" {
		return dbNode.NodeConfig.Partitions[0]
	}
	if dbNode.NodeType != "" {
		return dbNode.NodeType
	}
	return "compute"
}

func safeIntString(v int) string {
	if v <= 0 {
		return "-"
	}
	return strconv.Itoa(v)
}

func safeState(state string) string {
	if strings.TrimSpace(state) == "" {
		return "unknown"
	}
	return state
}

func (s *SlurmService) ensureNodeHasValidName(node *models.SlurmNode) string {
	if node == nil {
		return ""
	}
	normalized := normalizeNodeNameValue(node.NodeName, node.Host, node.ID)
	if normalized == "" {
		return ""
	}
	if node.NodeName != normalized {
		if s.db != nil {
			if err := s.db.Model(node).Update("node_name", normalized).Error; err != nil {
				log.Printf("[WARN] 更新节点 %d 名称失败: %v", node.ID, err)
			} else {
				log.Printf("[DEBUG] 节点 %d 名称已规范化: %s -> %s", node.ID, node.NodeName, normalized)
			}
		}
		node.NodeName = normalized
	}
	return node.NodeName
}

func normalizeNodeNameValue(rawName, host string, fallbackID uint) string {
	name := strings.TrimSpace(rawName)
	if name == "" {
		name = strings.TrimSpace(host)
	}
	if name == "" && fallbackID > 0 {
		name = fmt.Sprintf("node-%d", fallbackID)
	}
	name = strings.ToLower(name)
	if name == "" {
		name = "node"
	}
	var builder strings.Builder
	prevHyphen := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			prevHyphen = false
			continue
		}
		if r == '-' || r == '_' {
			if !prevHyphen {
				builder.WriteRune('-')
				prevHyphen = true
			}
			continue
		}
		if !prevHyphen {
			builder.WriteRune('-')
			prevHyphen = true
		}
	}
	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		if host != "" {
			cleanedHost := strings.Map(func(r rune) rune {
				switch {
				case r >= 'a' && r <= 'z':
					return r
				case r >= '0' && r <= '9':
					return r
				case r == '-':
					return r
				case r == '.' || r == '_':
					return '-'
				case r >= 'A' && r <= 'Z':
					return r + 32
				default:
					return -1
				}
			}, host)
			normalized = strings.Trim(cleanedHost, "-")
		}
	}
	if normalized == "" {
		if fallbackID > 0 {
			normalized = fmt.Sprintf("node-%d", fallbackID)
		} else {
			normalized = fmt.Sprintf("node-%d", time.Now().Unix()%100000)
		}
	}
	if len(normalized) > 63 {
		normalized = normalized[:63]
	}
	return normalized
}

func (s *SlurmService) GetJobs(ctx context.Context) ([]SlurmJob, bool, error) {
	// squeue format: JobID|User|State|Elapsed|Nodes|Reason|Name|Partition
	// 通过SSH执行squeue命令
	output, err := s.executeSlurmCommand(ctx, "squeue -o '%i|%u|%T|%M|%D|%R|%j|%P'")
	if err != nil {
		// SSH命令失败: 返回空列表和demo标记
		return []SlurmJob{}, true, nil
	}

	var jobs []SlurmJob
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if first && (strings.Contains(line, "JOBID") || strings.Contains(line, "JOBID PARTITION")) {
			first = false
			continue
		}
		first = false
		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			for len(parts) < 8 {
				parts = append(parts, "")
			}
		}
		jobs = append(jobs, SlurmJob{
			ID:        parts[0],
			User:      parts[1],
			State:     parts[2],
			Elapsed:   parts[3],
			Nodes:     parts[4],
			Reason:    parts[5],
			Name:      parts[6],
			Partition: parts[7],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	return jobs, false, nil
}

// Helpers
func (s *SlurmService) getNodeStats(ctx context.Context) (total, idle, alloc, partitions int, demo bool) {
	// 通过SSH执行sinfo命令获取节点统计信息
	// sinfo summarized counts
	// partitions count via: sinfo -h -o %P | sort -u | wc -l (approx) – we'll parse simply
	output, err := s.executeSlurmCommand(ctx, "sinfo -h -o '%T|%P'")
	if err != nil {
		return s.getNodeStatsFromDB()
	}
	seenPartitions := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		state := parts[0]
		_ = state
		part := ""
		if len(parts) > 1 {
			part = parts[1]
		}
		if part != "" {
			seenPartitions[part] = struct{}{}
		}
		total++
		// Rough classification: lines correspond to partition-view; we'll refine with -N below
	}
	// Try better node-level stats
	out2, err2 := s.executeSlurmCommand(ctx, "sinfo -h -N -o '%T'")
	if err2 == nil {
		total = 0
		idle = 0
		alloc = 0
		scanner2 := bufio.NewScanner(strings.NewReader(out2))
		for scanner2.Scan() {
			total++
			st := strings.TrimSpace(scanner2.Text())
			up := strings.ToUpper(st)
			if strings.Contains(up, "IDLE") {
				idle++
			}
			if strings.Contains(up, "ALLOC") || up == "MIXED" {
				alloc++
			}
		}
	} else {
		return s.getNodeStatsFromDB()
	}
	partitions = len(seenPartitions)
	if partitions == 0 {
		// Try to derive partitions another way
		out3, err3 := s.executeSlurmCommand(ctx, "sinfo -h -o '%P'")
		if err3 == nil {
			set := map[string]struct{}{}
			scanner3 := bufio.NewScanner(strings.NewReader(out3))
			for scanner3.Scan() {
				set[strings.TrimSpace(scanner3.Text())] = struct{}{}
			}
			partitions = len(set)
		}
	}
	// 如果最终统计仍为 0，则使用数据库信息兜底，保证前端能够展示节点状态
	if total == 0 {
		return s.getNodeStatsFromDB()
	}

	return total, idle, alloc, partitions, false
}

// getNodeStatsFromDB 当直接执行 SLURM 命令失败时，从数据库兜底返回节点统计信息
func (s *SlurmService) getNodeStatsFromDB() (total, idle, alloc, partitions int, demo bool) {
	if s.db == nil {
		return 0, 0, 0, 0, true
	}

	var nodes []models.SlurmNode
	if err := s.db.Find(&nodes).Error; err != nil || len(nodes) == 0 {
		return 0, 0, 0, 0, true
	}

	for _, node := range nodes {
		total++
		status := strings.ToLower(node.Status)
		switch {
		case strings.Contains(status, "idle"), strings.Contains(status, "active"), strings.Contains(status, "ready"):
			idle++
		case strings.Contains(status, "alloc"), strings.Contains(status, "run"), strings.Contains(status, "busy"), strings.Contains(status, "mixed"):
			alloc++
		}
	}

	if partitions == 0 {
		partitions = 1
	}

	return total, idle, alloc, partitions, false
}

func (s *SlurmService) getJobStats(ctx context.Context) (running, pending, other int, demo bool) {
	// 通过SSH执行squeue命令获取作业统计信息
	output, err := s.executeSlurmCommand(ctx, "squeue -h -o '%T'")
	if err != nil {
		// Return zeros instead of mock data when SLURM is unavailable
		return 0, 0, 0, true
	}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		st := strings.ToUpper(strings.TrimSpace(scanner.Text()))
		switch st {
		case "RUNNING":
			running++
		case "PENDING":
			pending++
		default:
			other++
		}
	}
	return running, pending, other, false
}

func parseCPUs(c string) string {
	// CPUS format like A/I/O/T; return T if parsable
	if strings.Contains(c, "/") {
		parts := strings.Split(c, "/")
		return parts[len(parts)-1]
	}
	// Or a number
	if _, err := strconv.Atoi(c); err == nil {
		return c
	}
	return ""
}

// utility to detect missing binaries
func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

var ErrNotAvailable = errors.New("slurm tools not available")

// SlurmSummary SLURM集群摘要信息
type SlurmSummary struct {
	NodesTotal  int       `json:"nodes_total"`
	NodesIdle   int       `json:"nodes_idle"`
	NodesAlloc  int       `json:"nodes_alloc"`
	Partitions  int       `json:"partitions"`
	JobsRunning int       `json:"jobs_running"`
	JobsPending int       `json:"jobs_pending"`
	JobsOther   int       `json:"jobs_other"`
	Demo        bool      `json:"demo"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SlurmNode SLURM节点信息
type SlurmNode struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	CPUs      string `json:"cpus"`
	MemoryMB  string `json:"memory_mb"`
	Partition string `json:"partition"`
}

// SlurmJob SLURM作业信息
type SlurmJob struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	State     string `json:"state"`
	Elapsed   string `json:"elapsed"`
	Nodes     string `json:"nodes"`
	Reason    string `json:"reason"`
	Name      string `json:"name"`
	Partition string `json:"partition"`
}

// ScalingStatus 扩缩容状态
type ScalingStatus struct {
	ActiveOperations []ScalingOperation `json:"active_operations"`
	RecentOperations []ScalingOperation `json:"recent_operations"`
	NodeTemplates    []NodeTemplate     `json:"node_templates"`
	// 前端期望的字段
	Active       bool     `json:"active"`             // 是否有活跃任务
	ActiveTasks  int      `json:"active_tasks"`       // 活跃任务数
	SuccessNodes int      `json:"success_nodes"`      // 成功节点数
	FailedNodes  int      `json:"failed_nodes"`       // 失败节点数
	Progress     int      `json:"progress"`           // 进度百分比
	Warnings     []string `json:"warnings,omitempty"` // 数据源异常或校验提示
}

// ScalingOperation 扩缩容操作
type ScalingOperation struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"` // "scale-up" or "scale-down"
	Status       string     `json:"status"`
	Nodes        []string   `json:"nodes"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	Progress     int        `json:"progress"`
	SuccessNodes int        `json:"success_nodes"`
	FailedNodes  int        `json:"failed_nodes"`
	TotalNodes   int        `json:"total_nodes"`
	Source       string     `json:"source"`
}

// NodeConfig 节点配置
// NodeConfig 节点配置
type NodeConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	KeyPath    string `json:"key_path"`
	PrivateKey string `json:"private_key"` // 新增：内联私钥内容
	Password   string `json:"password"`
	MinionID   string `json:"minion_id"`

	// 硬件配置
	CPUs           int    `json:"cpus"`             // CPU 核心数
	Memory         int    `json:"memory"`           // 内存大小 (MB)
	Storage        int    `json:"storage"`          // 存储大小 (GB)
	GPUs           int    `json:"gpus"`             // GPU 数量
	XPUs           int    `json:"xpus"`             // XPU (昆仑芯) 数量
	Sockets        int    `json:"sockets"`          // CPU 插槽数
	CoresPerSocket int    `json:"cores_per_socket"` // 每个插槽的核心数
	ThreadsPerCore int    `json:"threads_per_core"` // 每个核心的线程数
	Features       string `json:"features"`         // 节点特性标签
}

// NodeTemplate 节点模板
type NodeTemplate struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      NodeConfig `json:"config"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ScalingResult 扩缩容结果
type ScalingResult struct {
	OperationID string              `json:"operation_id"`
	Success     bool                `json:"success"`
	Results     []NodeScalingResult `json:"results"`
}

// NodeScalingResult 节点扩缩容结果
type NodeScalingResult struct {
	NodeID  string `json:"node_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetScalingStatus 获取扩缩容状态
func (s *SlurmService) GetScalingStatus(ctx context.Context) (*ScalingStatus, error) {
	clusterDB := s.db
	taskDB := s.taskStore()
	warnings := make([]string, 0, 2)

	// 从数据库获取实际的任务状态
	var runningTasks int64
	var completedTasks int64
	var failedTasks int64

	// 查询 NodeInstallTask 表中的扩缩容相关任务数量
	// 查询最近24小时内的任务
	recentTime := time.Now().Add(-24 * time.Hour)

	if clusterDB == nil && taskDB == nil {
		log.Printf("[WARN] GetScalingStatus: 集群数据库与任务数据库均不可用，返回空状态")
		return &ScalingStatus{
			ActiveOperations: []ScalingOperation{},
			RecentOperations: []ScalingOperation{},
			NodeTemplates:    []NodeTemplate{},
			Active:           false,
			ActiveTasks:      0,
			SuccessNodes:     0,
			FailedNodes:      0,
			Progress:         0,
			Warnings:         []string{"数据库连接不可用，扩缩容进度无法统计"},
		}, nil
	}

	if s.taskDB == nil && clusterDB != nil {
		warnings = append(warnings, "任务存储未配置独立数据库，已回退到SLURM集群库，历史进度可能不完整")
	}

	// 查询 NodeInstallTask 表（InstallSlurmNode接口创建的任务）
	var nodeInstallRunning, nodeInstallCompleted, nodeInstallFailed int64
	if clusterDB != nil {
		clusterDB.Model(&models.NodeInstallTask{}).
			Where("created_at > ?", recentTime).
			Where("status IN ?", []string{"running", "in_progress", "processing", "pending"}).
			Count(&nodeInstallRunning)

		clusterDB.Model(&models.NodeInstallTask{}).
			Where("created_at > ?", recentTime).
			Where("status IN ?", []string{"completed", "success"}).
			Count(&nodeInstallCompleted)

		clusterDB.Model(&models.NodeInstallTask{}).
			Where("created_at > ?", recentTime).
			Where("status IN ?", []string{"failed", "error"}).
			Count(&nodeInstallFailed)
	} else {
		warnings = append(warnings, "无法访问SLURM节点数据库，节点安装任务统计为空")
	}

	// 查询 SlurmTask 表（ScaleUpAsync接口创建的任务）
	var slurmTaskRunning, slurmTaskCompleted, slurmTaskFailed int64
	if taskDB != nil {
		taskDB.Model(&models.SlurmTask{}).
			Where("created_at > ?", recentTime).
			Where("type IN ?", []string{"scale_up", "scale_down"}).
			Where("status IN ?", []string{"running", "in_progress", "processing", "pending"}).
			Count(&slurmTaskRunning)

		taskDB.Model(&models.SlurmTask{}).
			Where("created_at > ?", recentTime).
			Where("type IN ?", []string{"scale_up", "scale_down"}).
			Where("status IN ?", []string{"completed", "success"}).
			Count(&slurmTaskCompleted)

		taskDB.Model(&models.SlurmTask{}).
			Where("created_at > ?", recentTime).
			Where("type IN ?", []string{"scale_up", "scale_down"}).
			Where("status IN ?", []string{"failed", "error"}).
			Count(&slurmTaskFailed)
	} else {
		warnings = append(warnings, "未获取到任务数据库，扩缩容进度统计不可用")
	}

	// 合并两个表的统计结果
	runningTasks = nodeInstallRunning + slurmTaskRunning
	completedTasks = nodeInstallCompleted + slurmTaskCompleted
	failedTasks = nodeInstallFailed + slurmTaskFailed

	// 计算总体进度
	totalTasks := runningTasks + completedTasks + failedTasks
	progress := 0
	if totalTasks > 0 {
		progress = int((completedTasks * 100) / totalTasks)
	}

	// 获取最近的安装任务及其详细进度
	var recentTasks []models.NodeInstallTask
	if clusterDB != nil {
		clusterDB.Where("created_at > ?", recentTime).
			Order("created_at DESC").
			Limit(10).
			Find(&recentTasks)
	}

	// 如果有正在运行的任务，计算加权平均进度
	if runningTasks > 0 {
		var totalProgress int64
		var taskCount int64
		for _, task := range recentTasks {
			if task.Status == "running" || task.Status == "in_progress" || task.Status == "processing" {
				totalProgress += int64(task.Progress)
				taskCount++
			}
		}
		// 使用任务的实际进度来计算更准确的总体进度
		if taskCount > 0 {
			runningAvgProgress := totalProgress / taskCount
			// 综合考虑完成任务和正在运行任务的进度
			if totalTasks > 0 {
				progress = int((completedTasks*100 + runningTasks*runningAvgProgress) / totalTasks)
			}
		}
	}

	// 从 SlurmTask 表读取节点统计信息
	var slurmTasks []models.SlurmTask
	if taskDB != nil {
		taskDB.Where("created_at > ?", recentTime).
			Where("type IN ?", []string{"scale_up", "scale_down"}).
			Order("created_at DESC").
			Limit(10).
			Find(&slurmTasks)
	}

	// 计算总的成功/失败节点数
	var totalSuccessNodes, totalFailedNodes int
	log.Printf("[DEBUG] GetScalingStatus: 找到 %d 个 SlurmTask 记录", len(slurmTasks))
	for _, task := range slurmTasks {
		log.Printf("[DEBUG] SlurmTask: TaskID=%s, Status=%s, NodesSuccess=%d, NodesFailed=%d",
			task.TaskID, task.Status, task.NodesSuccess, task.NodesFailed)
		// 只统计已完成的任务
		if task.Status == "completed" || task.Status == "failed" {
			totalSuccessNodes += task.NodesSuccess
			totalFailedNodes += task.NodesFailed
		}
	}
	log.Printf("[DEBUG] GetScalingStatus: totalSuccessNodes=%d, totalFailedNodes=%d", totalSuccessNodes, totalFailedNodes)

	// 使用实际节点状态覆盖统计结果，避免历史任务影响当前视图
	var nodeStatusCounts []struct {
		Status string
		Count  int64
	}
	if clusterDB != nil {
		if err := clusterDB.Model(&models.SlurmNode{}).
			Select("status, COUNT(*) as count").
			Group("status").
			Find(&nodeStatusCounts).Error; err == nil {
			var activeCount, failedCount int64
			for _, item := range nodeStatusCounts {
				switch strings.ToLower(item.Status) {
				case "active":
					activeCount += item.Count
				case "failed", "error":
					failedCount += item.Count
				}
			}
			if activeCount > 0 || failedCount > 0 {
				log.Printf("[DEBUG] 覆盖节点统计: active=%d failed=%d", activeCount, failedCount)
				totalSuccessNodes = int(activeCount)
				totalFailedNodes = int(failedCount)
			}
		} else {
			log.Printf("[WARN] 获取节点状态统计失败: %v", err)
		}

		var trackedNodes []models.SlurmNode
		if err := clusterDB.Where("status IN ?", []string{"active", "running", "in_progress", "processing"}).Find(&trackedNodes).Error; err == nil {
			if missing, missErr := s.detectUnregisteredNodes(ctx, trackedNodes); missErr == nil {
				if len(missing) > 0 {
					sample := missing
					if len(sample) > 5 {
						sample = sample[:5]
					}
					warnings = append(warnings, fmt.Sprintf("检测到 %d 个节点未在Slurm注册 (示例: %s)", len(missing), strings.Join(sample, ", ")))
				}
			} else {
				log.Printf("[WARN] 对比sinfo失败，无法检测未注册节点: %v", missErr)
			}
		} else {
			log.Printf("[WARN] 查询节点明细失败，无法交叉校验注册情况: %v", err)
		}
	} else {
		warnings = append(warnings, "无法连接SLURM节点数据库，节点成功/失败统计不可用")
	}

	clampPercent := func(p int) int {
		switch {
		case p < 0:
			return 0
		case p > 100:
			return 100
		default:
			return p
		}
	}

	isActiveStatus := func(status string) bool {
		st := strings.ToLower(status)
		switch st {
		case "running", "in_progress", "processing", "pending":
			return true
		default:
			return false
		}
	}

	isTerminalStatus := func(status string) bool {
		st := strings.ToLower(status)
		switch st {
		case "completed", "success", "failed", "error", "cancelled":
			return true
		default:
			return false
		}
	}

	nodeNameCache := map[uint]string{}
	getNodeName := func(id uint) string {
		if id == 0 {
			return ""
		}
		if name, ok := nodeNameCache[id]; ok {
			return name
		}
		if clusterDB != nil {
			var node models.SlurmNode
			if err := clusterDB.First(&node, id).Error; err == nil {
				name := normalizeNodeNameValue(node.NodeName, node.Host, node.ID)
				nodeNameCache[id] = name
				return name
			}
		}
		fallback := fmt.Sprintf("node-%d", id)
		nodeNameCache[id] = fallback
		return fallback
	}

	buildNodeInstallOperation := func(task models.NodeInstallTask) ScalingOperation {
		successNodes := 0
		failedNodes := 0
		status := strings.ToLower(task.Status)
		if status == "completed" || status == "success" {
			successNodes = 1
		} else if status == "failed" || status == "error" {
			failedNodes = 1
		}
		op := ScalingOperation{
			ID:           task.TaskID,
			Type:         task.TaskType,
			Status:       task.Status,
			Nodes:        []string{getNodeName(task.NodeID)},
			StartedAt:    task.CreatedAt,
			Error:        task.ErrorMessage,
			Progress:     clampPercent(task.Progress),
			SuccessNodes: successNodes,
			FailedNodes:  failedNodes,
			TotalNodes:   1,
			Source:       "node_install",
		}
		if task.CompletedAt != nil {
			op.CompletedAt = task.CompletedAt
		}
		return op
	}

	normalizeTaskProgress := func(raw float64) int {
		switch {
		case raw <= 0:
			return 0
		case raw <= 1.0:
			return clampPercent(int(math.Round(raw * 100)))
		default:
			return clampPercent(int(math.Round(raw)))
		}
	}

	buildSlurmTaskOperation := func(task models.SlurmTask) ScalingOperation {
		totalNodes := task.NodesTotal
		targetNodes := make([]string, len(task.TargetNodes))
		copy(targetNodes, task.TargetNodes)
		if totalNodes == 0 && len(targetNodes) > 0 {
			totalNodes = len(targetNodes)
		}
		progressValue := normalizeTaskProgress(task.Progress)
		op := ScalingOperation{
			ID:           task.TaskID,
			Type:         task.Type,
			Status:       task.Status,
			Nodes:        targetNodes,
			StartedAt:    task.CreatedAt,
			Error:        task.ErrorMessage,
			Progress:     clampPercent(progressValue),
			SuccessNodes: task.NodesSuccess,
			FailedNodes:  task.NodesFailed,
			TotalNodes:   totalNodes,
			Source:       "slurm_task",
		}
		if task.CompletedAt != nil {
			op.CompletedAt = task.CompletedAt
		}
		return op
	}

	activeOperations := []ScalingOperation{}
	recentOperations := []ScalingOperation{}
	maxRecent := 5

	for _, task := range recentTasks {
		op := buildNodeInstallOperation(task)
		if isActiveStatus(task.Status) {
			activeOperations = append(activeOperations, op)
			continue
		}
		if isTerminalStatus(task.Status) && len(recentOperations) < maxRecent {
			recentOperations = append(recentOperations, op)
		}
	}

	for _, task := range slurmTasks {
		op := buildSlurmTaskOperation(task)
		if isActiveStatus(task.Status) {
			activeOperations = append(activeOperations, op)
			continue
		}
		if isTerminalStatus(task.Status) && len(recentOperations) < maxRecent {
			recentOperations = append(recentOperations, op)
		}
	}

	if len(activeOperations) > 0 {
		sumProgress := 0
		for _, op := range activeOperations {
			sumProgress += clampPercent(op.Progress)
		}
		progress = sumProgress / len(activeOperations)
	}

	// 如果没有活跃任务，根据节点结果进行兜底进度计算，避免长期停留在 0% 或 1%
	if len(activeOperations) == 0 && runningTasks == 0 {
		totalNodeOps := totalSuccessNodes + totalFailedNodes
		if totalNodeOps > 0 {
			if totalFailedNodes == 0 {
				progress = 100
			} else {
				progress = int((totalSuccessNodes * 100) / totalNodeOps)
			}
		}
	}

	activeTaskCount := len(activeOperations)
	activeFlag := activeTaskCount > 0
	if !activeFlag && runningTasks > 0 {
		activeFlag = true
		activeTaskCount = int(runningTasks)
	}

	return &ScalingStatus{
		ActiveOperations: activeOperations,
		RecentOperations: recentOperations,
		NodeTemplates:    []NodeTemplate{},
		Active:           activeFlag,
		ActiveTasks:      activeTaskCount,
		SuccessNodes:     totalSuccessNodes, // 从 slurm_tasks 表读取
		FailedNodes:      totalFailedNodes,  // 从 slurm_tasks 表读取
		Progress:         progress,
		Warnings:         warnings,
	}, nil
}

// ScaleUp 执行扩容操作
func (s *SlurmService) ScaleUp(ctx context.Context, nodes []NodeConfig) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleUp: 开始扩容操作，节点数量: %d", len(nodes))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	// 1. 更新数据库中的节点状态为active
	for _, node := range nodes {
		// 查找或创建节点记录
		var dbNode models.SlurmNode
		err := s.db.Where("host = ? OR node_name = ?", node.Host, node.MinionID).First(&dbNode).Error

		if err == gorm.ErrRecordNotFound {
			// 节点不存在，跳过（应该在之前的步骤中创建）
			log.Printf("[WARN] 节点 %s 在数据库中不存在，跳过", node.Host)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: "节点在数据库中不存在",
			})
			continue
		} else if err != nil {
			log.Printf("[ERROR] 查询节点 %s 失败: %v", node.Host, err)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: fmt.Sprintf("数据库错误: %v", err),
			})
			continue
		}

		// 更新节点状态和硬件配置为active
		updateData := map[string]interface{}{
			"status":         "active",
			"salt_minion_id": node.MinionID,
		}

		// 如果提供了硬件配置，更新到数据库
		if node.CPUs > 0 {
			updateData["cpus"] = node.CPUs
		}
		if node.Memory > 0 {
			updateData["memory"] = node.Memory
		}
		if node.Storage > 0 {
			updateData["storage"] = node.Storage
		}
		if node.GPUs > 0 {
			updateData["gpus"] = node.GPUs
		}
		if node.XPUs > 0 {
			updateData["xpus"] = node.XPUs
		}
		if node.Sockets > 0 {
			updateData["sockets"] = node.Sockets
		}
		if node.CoresPerSocket > 0 {
			updateData["cores_per_socket"] = node.CoresPerSocket
		}
		if node.ThreadsPerCore > 0 {
			updateData["threads_per_core"] = node.ThreadsPerCore
		}

		if err := s.db.Model(&dbNode).Updates(updateData).Error; err != nil {
			log.Printf("[ERROR] 更新节点 %s 状态失败: %v", node.Host, err)
			result.Results = append(result.Results, NodeScalingResult{
				NodeID:  node.Host,
				Success: false,
				Message: fmt.Sprintf("更新状态失败: %v", err),
			})
			continue
		}

		log.Printf("[DEBUG] 节点 %s 状态已更新为 active", node.Host)
		result.Results = append(result.Results, NodeScalingResult{
			NodeID:  node.Host,
			Success: true,
			Message: "节点已成功添加到SLURM集群",
		})
	}

	// 2. 在节点上并发安装slurmd服务
	log.Printf("[DEBUG] ScaleUp: 开始在 %d 个节点上并发安装slurmd服务", len(nodes))

	var installWg sync.WaitGroup
	var installMu sync.Mutex
	installResults := make(map[string]*InstallSlurmNodeResponse)

	// 限制并发数为 5，避免过多并发
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, node := range nodes {
		installWg.Add(1)
		go func(n NodeConfig) {
			defer installWg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检测操作系统类型
			osType := s.detectNodeOSType(ctx, n.Host)
			if osType == "" {
				log.Printf("[WARN] 无法检测节点 %s 的操作系统类型，尝试rocky", n.Host)
				osType = "rocky" // 默认使用rocky
			}

			log.Printf("[INFO] 在节点 %s 上安装slurmd (OS: %s)", n.Host, osType)

			// 准备SSH连接参数（设置默认值）
			sshPort := n.Port
			if sshPort == 0 {
				sshPort = 22
			}
			sshUser := n.User
			if sshUser == "" {
				sshUser = "root"
			}

			// 调用安装方法，传递SSH连接参数（包括密钥）
			installReq := InstallSlurmNodeRequest{
				NodeName:   n.Host,
				OSType:     osType,
				SSHHost:    n.Host,
				SSHPort:    sshPort,
				SSHUser:    sshUser,
				Password:   n.Password,
				KeyPath:    n.KeyPath,
				PrivateKey: n.PrivateKey,
			}

			installResp, err := s.InstallSlurmNode(ctx, installReq)
			if err != nil {
				installResp = &InstallSlurmNodeResponse{
					Success: false,
					Message: fmt.Sprintf("安装失败: %v", err),
					Logs:    "",
				}
			}

			// 保存结果
			installMu.Lock()
			installResults[n.Host] = installResp
			installMu.Unlock()

			if installResp.Success {
				log.Printf("[SUCCESS] 节点 %s slurmd安装成功", n.Host)
			} else {
				log.Printf("[ERROR] 节点 %s slurmd安装失败: %s", n.Host, installResp.Message)
			}
		}(node)
	}

	// 等待所有安装任务完成
	installWg.Wait()
	log.Printf("[DEBUG] ScaleUp: 所有节点安装任务已完成")

	// 更新结果消息
	for i := range result.Results {
		nodeID := result.Results[i].NodeID
		if installResp, ok := installResults[nodeID]; ok {
			if installResp.Success {
				result.Results[i].Message = "节点已成功添加到SLURM集群并安装slurmd服务"
			} else {
				result.Results[i].Success = false
				result.Results[i].Message = fmt.Sprintf("节点添加成功但slurmd安装失败: %s", installResp.Message)
			}
		}
	}

	// 3. 生成并更新SLURM配置
	log.Printf("[DEBUG] ScaleUp: 开始更新SLURM配置")
	if err := s.updateSlurmConfigAndReload(ctx); err != nil {
		log.Printf("[ERROR] 更新SLURM配置失败: %v", err)
		// 不返回错误，因为节点状态已更新，可以手动重新加载配置
		result.Success = false
		for i := range result.Results {
			if result.Results[i].Success {
				result.Results[i].Message += " (SLURM配置更新失败，需要手动reload)"
			}
		}
	} else {
		log.Printf("[DEBUG] ScaleUp: SLURM配置更新成功")
	}

	return result, nil
}

// updateSlurmConfigAndReload 更新SLURM配置并重新加载
func (s *SlurmService) updateSlurmConfigAndReload(ctx context.Context) error {
	// 获取所有active状态的节点
	var nodes []models.SlurmNode
	if err := s.db.Where("status = ?", "active").Find(&nodes).Error; err != nil {
		return fmt.Errorf("获取active节点失败: %w", err)
	}

	log.Printf("[DEBUG] 找到 %d 个active节点", len(nodes))

	if len(nodes) == 0 {
		return fmt.Errorf("没有active状态的节点")
	}

	// 生成SLURM配置
	config := s.generateSlurmConfig(nodes)
	log.Printf("[DEBUG] 生成的SLURM配置:\n%s", config)

	// 通过Salt API更新SLURM master的配置文件
	if err := s.updateSlurmMasterConfig(ctx, config); err != nil {
		return fmt.Errorf("更新SLURM master配置失败: %w", err)
	}

	// 重新加载SLURM配置
	if err := s.reloadSlurmConfig(ctx); err != nil {
		return fmt.Errorf("重新加载SLURM配置失败: %w", err)
	}

	// 等待一秒让配置生效
	time.Sleep(1 * time.Second)

	// 将所有新节点设置为 DOWN 状态，并检测节点健康状态
	log.Printf("[DEBUG] 开始设置新添加节点的初始状态并进行健康检测...")
	for i := range nodes {
		node := &nodes[i]
		nodeName := s.ensureNodeHasValidName(node)
		if nodeName == "" {
			log.Printf("[WARN] 节点ID=%d 缺少合法的 NodeName，跳过 scontrol 初始化", node.ID)
			continue
		}
		if node.NodeType == "compute" || node.NodeType == "node" {
			if err := s.ensureNodeRegistered(ctx, nodeName); err != nil {
				log.Printf("[ERROR] 节点 %s 未能在SLURM控制器注册: %v", nodeName, err)
				continue
			}
			// 将节点设置为 DOWN 状态
			downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"新添加节点，正在检测状态\"", nodeName)
			output, err := s.ExecuteSlurmCommand(ctx, downCmd)
			if err != nil {
				log.Printf("[WARNING] 设置节点 %s 为 DOWN 状态失败: %v, output: %s", nodeName, err, output)
				continue
			}
			log.Printf("[INFO] 节点 %s 已设置为 DOWN 状态，开始健康检测...", nodeName)

			// 尝试检测和修复节点状态（最多3次）
			if err := s.DetectAndFixNodeState(ctx, nodeName, 3); err != nil {
				log.Printf("[ERROR] 节点 %s 健康检测失败: %v", nodeName, err)
			}
		}
	}

	log.Printf("[INFO] 所有新节点已完成初始化和健康检测")

	return nil
}

// DetectAndFixNodeState 检测节点状态并尝试修复异常（导出方法）
// maxRetries: 最大重试次数
// 返回 nil 表示节点正常或已成功修复，返回 error 表示修复失败
func (s *SlurmService) DetectAndFixNodeState(ctx context.Context, nodeName string, maxRetries int) error {
	log.Printf("[DEBUG] 开始检测节点 %s 的健康状态（最多重试 %d 次）", nodeName, maxRetries)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 等待节点状态稳定
		if attempt > 1 {
			waitTime := time.Duration(attempt) * 2 * time.Second
			log.Printf("[DEBUG] 第 %d 次检测前等待 %v...", attempt, waitTime)
			time.Sleep(waitTime)
		} else {
			time.Sleep(2 * time.Second)
		}

		// 检查节点状态
		checkCmd := fmt.Sprintf("scontrol show node %s", nodeName)
		output, err := s.ExecuteSlurmCommand(ctx, checkCmd)
		if err != nil {
			log.Printf("[WARNING] 第 %d/%d 次：无法检查节点 %s 状态: %v", attempt, maxRetries, nodeName, err)
			continue
		}

		reason := parseSlurmNodeReason(output)
		reasonLower := strings.ToLower(reason)

		// 分析节点状态
		if strings.Contains(output, "State=IDLE") || strings.Contains(output, "State=ALLOCATED") || strings.Contains(output, "State=MIXED") {
			// 节点状态正常
			log.Printf("[SUCCESS] 节点 %s 状态正常: IDLE/ALLOCATED/MIXED", nodeName)
			return nil
		}

		needsRestart := strings.Contains(output, "NOT_RESPONDING") || strings.Contains(output, "State=UNKNOWN") ||
			strings.Contains(reasonLower, "未响应") || strings.Contains(reasonLower, "not responding")

		if needsRestart {
			log.Printf("[WARNING] 第 %d/%d 次：节点 %s 未响应 (Reason=%s)，尝试重新拉起 slurmd", attempt, maxRetries, nodeName, reason)
			downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"节点未响应，第%d次检测失败\"", nodeName, attempt)
			if _, err := s.ExecuteSlurmCommand(ctx, downCmd); err != nil {
				log.Printf("[WARNING] 第 %d/%d 次：设置节点 %s 为 DOWN 失败: %v", attempt, maxRetries, nodeName, err)
			}
			if err := s.restartSlurmdForNode(ctx, nodeName); err != nil {
				log.Printf("[WARNING] 自动重启节点 %s slurmd 失败: %v", nodeName, err)
			} else {
				if err := s.resumeNode(ctx, nodeName); err != nil {
					log.Printf("[WARNING] 重置节点 %s 状态失败: %v", nodeName, err)
				}
				log.Printf("[INFO] 节点 %s slurmd 重启命令已执行，等待状态同步", nodeName)
				continue
			}
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("节点未响应或状态未知，可能原因：slurmd未运行、网络不可达或配置错误。请手动检查节点状态并执行: scontrol update NodeName=%s State=RESUME", nodeName)
		}

		if strings.Contains(output, "State=DOWN") {
			// 节点处于 DOWN 状态但可以响应，尝试激活
			log.Printf("[INFO] 第 %d/%d 次：节点 %s 处于 DOWN 状态，尝试激活...", attempt, maxRetries, nodeName)
			if err := s.resumeNode(ctx, nodeName); err != nil {
				log.Printf("[WARNING] 第 %d/%d 次：激活节点 %s 失败: %v", attempt, maxRetries, nodeName, err)
				continue
			}
			log.Printf("[INFO] 第 %d/%d 次：节点 %s 激活命令已执行", attempt, maxRetries, nodeName)
			continue // 下一次循环检查激活是否成功
		}

		// 其他未预期的状态
		log.Printf("[INFO] 第 %d/%d 次：节点 %s 状态: %s", attempt, maxRetries, nodeName, output)
	}

	// 所有重试都失败
	return fmt.Errorf("节点状态检测失败，已重试 %d 次。请手动检查节点状态并执行: scontrol update NodeName=%s State=RESUME", maxRetries, nodeName)
}

func (s *SlurmService) restartSlurmdForNode(ctx context.Context, nodeName string) error {
	if s.db == nil {
		return fmt.Errorf("数据库未就绪，无法定位节点 %s", nodeName)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	node, err := s.lookupNodeByName(ctx, nodeName)
	if err != nil {
		return err
	}
	user := strings.TrimSpace(node.Username)
	if user == "" {
		user = "root"
	}
	port := node.Port
	if port == 0 {
		port = 22
	}
	host := strings.TrimSpace(node.Host)
	if host == "" {
		host = nodeName
	}
	password := strings.TrimSpace(node.Password)
	privateKey := strings.TrimSpace(node.KeyPath)
	if privateKey == "" {
		privateKey = backendDefaultSSHKeyPath()
	}
	if privateKey != "" {
		if _, statErr := os.Stat(privateKey); statErr != nil {
			log.Printf("[DEBUG] SSH私钥 %s 不可用: %v，回退为空", privateKey, statErr)
			privateKey = ""
		}
	}
	var buf bytes.Buffer
	if err := s.startSlurmServicesViaSSH(ctx, host, port, user, password, privateKey, &buf); err != nil {
		return fmt.Errorf("远程重启 slurmd 失败: %w (输出: %s)", err, strings.TrimSpace(buf.String()))
	}
	log.Printf("[INFO] 节点 %s slurmd 已重新启动: %s", nodeName, strings.TrimSpace(buf.String()))
	return nil
}

func (s *SlurmService) lookupNodeByName(ctx context.Context, nodeName string) (*models.SlurmNode, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未配置")
	}
	normalized := strings.ToLower(strings.TrimSpace(nodeName))
	if normalized == "" {
		return nil, fmt.Errorf("节点名称为空")
	}
	var node models.SlurmNode
	query := s.db.WithContext(ctx).
		Where("LOWER(node_name) = ? OR LOWER(host) = ?", normalized, normalized)
	if err := query.First(&node).Error; err != nil {
		return nil, fmt.Errorf("无法根据名称 %s 定位节点: %w", nodeName, err)
	}
	return &node, nil
}

func backendDefaultSSHKeyPath() string {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return ""
	}
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	if _, err := os.Stat(keyPath); err == nil {
		return keyPath
	}
	return ""
}

func (s *SlurmService) resumeNode(ctx context.Context, nodeName string) error {
	resumeCmd := fmt.Sprintf("scontrol update NodeName=%s State=RESUME", nodeName)
	if _, err := s.ExecuteSlurmCommand(ctx, resumeCmd); err != nil {
		return err
	}
	return nil
}

func parseSlurmNodeReason(output string) string {
	idx := strings.Index(output, "Reason=")
	if idx == -1 {
		return ""
	}
	reason := output[idx+len("Reason="):]
	newline := strings.IndexByte(reason, '\n')
	if newline >= 0 {
		reason = reason[:newline]
	}
	reason = strings.TrimSpace(reason)
	reason = strings.Trim(reason, "\"")
	return reason
}

func (s *SlurmService) ensureNodeRegistered(ctx context.Context, nodeName string) error {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		checkCmd := fmt.Sprintf("scontrol show node %s", nodeName)
		output, err := s.ExecuteSlurmCommand(ctx, checkCmd)
		if err == nil && strings.Contains(output, fmt.Sprintf("NodeName=%s", nodeName)) {
			return nil
		}
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "invalid node name") {
			log.Printf("[WARN] 节点 %s 未在SLURM注册 (第 %d/%d 次)，尝试重新加载配置", nodeName, attempt, maxAttempts)
			if reloadErr := s.reloadSlurmConfig(ctx); reloadErr != nil {
				log.Printf("[ERROR] 重新加载SLURM配置失败: %v", reloadErr)
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if err != nil {
			return err
		}
		// 没有错误但输出不包含节点信息，短暂等待后重试
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("节点 %s 仍未在SLURM控制器中注册", nodeName)
}

func (s *SlurmService) detectUnregisteredNodes(ctx context.Context, nodes []models.SlurmNode) ([]string, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	output, err := s.executeSlurmCommand(ctx, "sinfo -h -N -o '%N'")
	if err != nil {
		return nil, err
	}
	active := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		active[strings.ToLower(name)] = struct{}{}
	}
	missing := make([]string, 0)
	for _, node := range nodes {
		name := normalizeNodeNameValue(node.NodeName, node.Host, node.ID)
		if name == "" {
			continue
		}
		if _, ok := active[strings.ToLower(name)]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

// getSlurmMasterSSHConfig 获取SLURM master的SSH连接配置
func (s *SlurmService) getSlurmMasterSSHConfig() (host string, port int, user, password, keyPath string) {
	host = os.Getenv("SLURM_MASTER_HOST")
	if host == "" {
		host = "ai-infra-slurm-master"
	}

	user = os.Getenv("SLURM_MASTER_USER")
	if user == "" {
		user = "root"
	}

	password = os.Getenv("SLURM_MASTER_PASSWORD")

	portStr := os.Getenv("SLURM_MASTER_PORT")
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}
	if port == 0 {
		port = 22
	}

	keyPath = strings.TrimSpace(os.Getenv("SLURM_MASTER_KEY_PATH"))
	if keyPath == "" {
		home := os.Getenv("HOME")
		if home != "" {
			defaultKey := filepath.Join(home, ".ssh", "id_rsa")
			if _, err := os.Stat(defaultKey); err == nil {
				keyPath = defaultKey
			}
		}
	}
	return
}

// updateSlurmMasterConfig 通过SSH动态更新SLURM master的配置文件
func (s *SlurmService) updateSlurmMasterConfig(ctx context.Context, config string) error {
	log.Printf("[DEBUG] 开始通过SSH更新SLURM配置")

	// 获取SLURM master的SSH连接信息
	slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath := s.getSlurmMasterSSHConfig()

	// 1. 首先读取当前的基础配置（保留非节点相关的配置）
	log.Printf("[DEBUG] 读取SLURM master现有配置...")
	readCmd := "cat /etc/slurm/slurm.conf"
	currentConfig, err := s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, readCmd)
	if err != nil {
		log.Printf("[WARNING] 无法读取现有配置，将使用新配置: %v", err)
	}

	// 2. 生成完整的配置文件
	// 如果能读取到现有配置，则提取基础部分并合并节点配置
	finalConfig := s.mergeConfigs(currentConfig, config)

	// 3. 通过SSH写入新配置
	log.Printf("[DEBUG] 通过SSH写入新的SLURM配置（长度: %d 字节）", len(finalConfig))

	// 使用here-doc方式写入，避免特殊字符问题
	writeCmd := fmt.Sprintf("cat > /etc/slurm/slurm.conf << 'SLURM_CONFIG_EOF'\n%s\nSLURM_CONFIG_EOF", finalConfig)
	_, err = s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, writeCmd)
	if err != nil {
		return fmt.Errorf("SSH写入配置文件失败: %w", err)
	}

	// 4. 验证配置文件已正确写入
	verifyCmd := "wc -l /etc/slurm/slurm.conf"
	verifyOutput, err := s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, verifyCmd)
	if err != nil {
		log.Printf("[WARNING] 无法验证配置文件: %v", err)
	} else {
		log.Printf("[DEBUG] 配置文件已验证: %s", strings.TrimSpace(verifyOutput))
	}

	log.Printf("[DEBUG] SLURM配置文件已通过SSH成功更新")
	return nil
}

// mergeConfigs 合并基础配置和节点配置
func (s *SlurmService) mergeConfigs(currentConfig, newNodesConfig string) string {
	// 如果无法读取当前配置，直接使用新生成的配置
	if currentConfig == "" {
		return newNodesConfig
	}

	// 提取当前配置中的基础部分（非NodeName和PartitionName行）
	var baseLines []string
	lines := strings.Split(currentConfig, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过节点和分区定义行
		if strings.HasPrefix(trimmed, "NodeName=") ||
			strings.HasPrefix(trimmed, "PartitionName=") {
			continue
		}
		baseLines = append(baseLines, line)
	}

	// 提取新配置中的节点和分区定义（允许注释文案不同）
	var nodeLines []string
	newLines := strings.Split(newNodesConfig, "\n")
	for _, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "NodeName=") || strings.HasPrefix(trimmed, "PartitionName=") {
			nodeLines = append(nodeLines, line)
		}
	}

	// 合并：基础配置 + 节点配置注释 + 节点定义
	result := strings.Join(baseLines, "\n")
	if len(nodeLines) > 0 {
		result += "\n\n# 节点配置（动态生成）\n"
		result += strings.Join(nodeLines, "\n")
	}

	return result
}

// reloadSlurmConfig 通过SSH重新加载SLURM配置
func (s *SlurmService) reloadSlurmConfig(ctx context.Context) error {
	log.Printf("[DEBUG] 开始通过SSH重新加载SLURM配置")

	// 获取SLURM master的SSH连接信息
	slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath := s.getSlurmMasterSSHConfig()

	// 执行scontrol reconfigure
	reconfigCmd := "scontrol reconfigure"
	output, err := s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, reconfigCmd)
	if err != nil {
		// 如果错误是"Zero Bytes were transmitted"，这实际上是成功的（命令执行了但没有输出）
		if strings.Contains(output, "Zero Bytes were transmitted") || strings.TrimSpace(output) == "" {
			log.Printf("[DEBUG] scontrol reconfigure执行成功（无输出）")
		} else {
			return fmt.Errorf("SSH执行scontrol reconfigure失败: %w, output: %s", err, output)
		}
	} else {
		log.Printf("[DEBUG] scontrol reconfigure成功: %s", strings.TrimSpace(output))
	}

	// 验证配置重新加载成功
	verifyCmd := "scontrol ping"
	verifyOutput, err := s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, verifyCmd)
	if err != nil {
		log.Printf("[WARNING] 无法验证slurmctld状态: %v", err)
	} else {
		log.Printf("[DEBUG] slurmctld状态: %s", strings.TrimSpace(verifyOutput))
	}

	return nil
}

// executeSlurmCommand 执行SLURM命令的辅助函数（通过SSH连接到SLURM master）
func (s *SlurmService) executeSlurmCommand(ctx context.Context, command string) (string, error) {
	// 获取SLURM master的SSH连接信息
	slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath := s.getSlurmMasterSSHConfig()

	// 使用密钥认证连接slurm-master
	return s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, command)
}

// ExecuteSlurmCommand 公开的SLURM命令执行方法（供Controller调用）
func (s *SlurmService) ExecuteSlurmCommand(ctx context.Context, command string) (string, error) {
	return s.executeSlurmCommand(ctx, command)
}

// executeSSHCommand 执行SSH命令的辅助函数（支持密码和密钥认证）
func (s *SlurmService) executeSSHCommand(host string, port int, user, password, command string) (string, error) {
	return s.executeSSHCommandWithKey(host, port, user, password, "", command)
}

// executeSSHCommandWithKey 执行SSH命令（支持密码和密钥认证）
func (s *SlurmService) executeSSHCommandWithKey(host string, port int, user, password, privateKey, command string) (string, error) {
	// 创建SSH客户端配置
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应使用正确的host key验证
		Timeout:         30 * time.Second,
	}

	// 设置认证方法
	var authMethods []ssh.AuthMethod

	// 优先使用私钥认证
	trimmedKey := strings.TrimSpace(privateKey)
	if trimmedKey != "" {
		var signer ssh.Signer
		var err error
		if strings.HasPrefix(trimmedKey, "-----BEGIN") {
			signer, err = ssh.ParsePrivateKey([]byte(trimmedKey))
		} else {
			keyContent, readErr := os.ReadFile(trimmedKey)
			if readErr != nil {
				if errors.Is(readErr, os.ErrNotExist) {
					log.Printf("[DEBUG] SSH私钥文件不存在(%s)，跳过密钥认证", trimmedKey)
				} else {
					return "", fmt.Errorf("读取私钥文件失败: %v", readErr)
				}
			} else {
				signer, err = ssh.ParsePrivateKey(keyContent)
			}
		}
		if err != nil {
			return "", fmt.Errorf("解析私钥失败: %v", err)
		}
		if signer != nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	// 如果提供了密码，添加密码认证
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return "", fmt.Errorf("未提供任何认证方式（密码或私钥）")
	}

	config.Auth = authMethods

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %w", err)
	}
	defer client.Close()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 执行命令
	output, err := session.CombinedOutput(command)
	outputStr := string(output)

	if err != nil {
		// 检查是否是退出码错误
		if exitErr, ok := err.(*ssh.ExitError); ok {
			// 返回更详细的错误信息，包含输出内容
			if outputStr != "" {
				return outputStr, fmt.Errorf("SSH命令执行失败 (退出码 %d): %s", exitErr.ExitStatus(), outputStr)
			}
			return outputStr, fmt.Errorf("SSH命令执行失败 (退出码 %d)", exitErr.ExitStatus())
		}
		// 其他SSH错误
		return outputStr, fmt.Errorf("SSH命令执行失败: %w (输出: %s)", err, outputStr)
	}

	return outputStr, nil
}

// getSaltAPIToken 获取Salt API认证token
func (s *SlurmService) getSaltAPIToken(ctx context.Context) (string, error) {
	saltAPIURL := os.Getenv("SALTSTACK_MASTER_URL")
	if saltAPIURL == "" {
		saltAPIURL = "http://saltstack:8002"
	}

	username := os.Getenv("SALT_API_USERNAME")
	if username == "" {
		username = "saltapi"
	}
	password := os.Getenv("SALT_API_PASSWORD")
	if password == "" {
		password = "your-salt-api-password"
	}
	eauth := os.Getenv("SALT_API_EAUTH")
	if eauth == "" {
		eauth = "file"
	}

	// 认证请求
	authPayload := map[string]string{
		"username": username,
		"password": password,
		"eauth":    eauth,
	}

	jsonData, err := json.Marshal(authPayload)
	if err != nil {
		return "", fmt.Errorf("序列化认证数据失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", saltAPIURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("认证失败 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var authResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResult); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}

	// 提取token
	var token string
	if returnData, ok := authResult["return"].([]interface{}); ok && len(returnData) > 0 {
		if tokenData, ok := returnData[0].(map[string]interface{}); ok {
			if t, ok := tokenData["token"].(string); ok {
				token = t
			}
		}
	}

	if token == "" {
		return "", fmt.Errorf("未能从认证响应中获取token")
	}

	return token, nil
}

// executeSaltCommand 执行Salt命令
func (s *SlurmService) executeSaltCommand(ctx context.Context, token string, cmd map[string]interface{}) (map[string]interface{}, error) {
	saltAPIURL := os.Getenv("SALTSTACK_MASTER_URL")
	if saltAPIURL == "" {
		saltAPIURL = "http://saltstack:8002"
	}

	jsonData, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("序列化命令数据失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", saltAPIURL+"/", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result, nil
} // UpdateSlurmConfig 更新SLURM配置文件并重新加载
func (s *SlurmService) UpdateSlurmConfig(ctx context.Context, sshSvc SSHServiceInterface) error {
	// 获取当前所有活跃节点
	var nodes []models.SlurmNode
	if err := s.db.Where("status = ?", "active").Find(&nodes).Error; err != nil {
		return fmt.Errorf("failed to get active nodes: %w", err)
	}

	// 生成新的slurm.conf内容
	config := s.generateSlurmConfig(nodes)

	// 获取SLURM控制器信息
	controllerHost, controllerPort, err := s.getSlurmControllerInfo()
	if err != nil {
		return fmt.Errorf("failed to get SLURM controller info: %w", err)
	}

	// 上传新的配置文件到控制器
	configPath := "/etc/slurm/slurm.conf"
	if err := sshSvc.UploadFile(controllerHost, controllerPort, "root", "", []byte(config), configPath); err != nil {
		return fmt.Errorf("failed to upload slurm.conf: %w", err)
	}

	// 重新加载SLURM配置
	reloadCmd := "scontrol reconfigure"
	if _, err := sshSvc.ExecuteCommand(controllerHost, controllerPort, "root", "", reloadCmd); err != nil {
		return fmt.Errorf("failed to reload SLURM config: %w", err)
	}

	return nil
}

// generateSlurmConfig 生成SLURM配置文件内容（保留全局配置，只更新节点和分区部分）
func (s *SlurmService) generateSlurmConfig(nodes []models.SlurmNode) string {
	globalConfig := strings.TrimSpace(s.extractGlobalSlurmConfig(s.loadBaseSlurmConfig()))
	if globalConfig == "" {
		if currentConfig, err := s.fetchRemoteSlurmConfig(); err == nil {
			globalConfig = strings.TrimSpace(s.extractGlobalSlurmConfig(currentConfig))
		} else {
			log.Printf("[WARN] 无法读取远程slurm.conf: %v", err)
		}
	}
	if globalConfig == "" {
		log.Printf("[WARN] 使用内置SLURM配置模板作为兜底")
		globalConfig = defaultSlurmGlobalConfig
	}

	var builder strings.Builder
	builder.WriteString(globalConfig)
	builder.WriteString("\n\n# 计算节点配置（动态生成）\n")

	// 添加节点定义
	computeNodes := []string{}
	log.Printf("[DEBUG] generateSlurmConfig: 处理 %d 个节点", len(nodes))
	for i := range nodes {
		node := &nodes[i]
		nodeName := s.ensureNodeHasValidName(node)
		log.Printf("[DEBUG] 节点 #%d: NodeName=%s, NodeType=%s, Host=%s", i, nodeName, node.NodeType, node.Host)
		if nodeName == "" {
			log.Printf("[WARN] 跳过节点 #%d，缺少合法 NodeName (Host=%s)", i, node.Host)
			continue
		}
		if node.NodeType == "compute" || node.NodeType == "node" {
			// 使用节点配置的硬件参数，如果为0则使用默认值
			cpus := node.CPUs
			if cpus == 0 {
				cpus = 2 // 默认2核
			}

			memory := node.Memory
			if memory == 0 {
				memory = 1000 // 默认1000MB
			}

			sockets := node.Sockets
			if sockets == 0 {
				sockets = 1 // 默认1个插槽
			}

			coresPerSocket := node.CoresPerSocket
			if coresPerSocket == 0 {
				coresPerSocket = cpus / sockets // 自动计算
				if coresPerSocket == 0 {
					coresPerSocket = 1
				}
			}

			threadsPerCore := node.ThreadsPerCore
			if threadsPerCore == 0 {
				threadsPerCore = 1 // 默认1个线程
			}

			// 构建节点配置字符串
			nodeConfig := fmt.Sprintf("NodeName=%s NodeAddr=%s CPUs=%d Sockets=%d CoresPerSocket=%d ThreadsPerCore=%d RealMemory=%d",
				nodeName, node.Host, cpus, sockets, coresPerSocket, threadsPerCore, memory)

			// 添加 GPU 配置（如果有）
			if node.GPUs > 0 {
				nodeConfig += fmt.Sprintf(" Gres=gpu:%d", node.GPUs)
			}

			// 添加 XPU 配置（如果有）
			if node.XPUs > 0 {
				if node.GPUs > 0 {
					// 如果已经有GPU配置，追加XPU
					nodeConfig += fmt.Sprintf(",xpu:%d", node.XPUs)
				} else {
					nodeConfig += fmt.Sprintf(" Gres=xpu:%d", node.XPUs)
				}
			}

			builder.WriteString(nodeConfig)
			builder.WriteString("\n")
			computeNodes = append(computeNodes, nodeName)
			log.Printf("[DEBUG] 已添加计算节点: %s (地址: %s, CPU:%d, 内存:%dMB, GPU:%d, XPU:%d)",
				nodeName, node.Host, cpus, memory, node.GPUs, node.XPUs)
		} else {
			log.Printf("[WARNING] 跳过节点 %s，类型不匹配: %s", nodeName, node.NodeType)
		}
	}

	// 添加分区配置
	log.Printf("[DEBUG] 计算节点列表: %v", computeNodes)
	if len(computeNodes) > 0 {
		partitionConfig := fmt.Sprintf("PartitionName=compute Nodes=%s Default=YES MaxTime=INFINITE State=UP",
			strings.Join(computeNodes, ","))
		builder.WriteString(partitionConfig)
		builder.WriteString("\n")
		log.Printf("[DEBUG] 已添加分区配置: %s", partitionConfig)
	} else {
		log.Printf("[WARNING] 没有计算节点，跳过分区配置")
	}

	return builder.String()
}

func (s *SlurmService) loadBaseSlurmConfig() string {
	candidates := s.resolveSlurmConfigTemplatePaths()
	userProvided := strings.TrimSpace(os.Getenv("SLURM_BASE_CONFIG_PATH"))
	for idx, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if idx == 0 && userProvided != "" {
				log.Printf("[WARN] 无法读取自定义的SLURM模板 %s: %v", candidate, err)
			}
			continue
		}
		return string(data)
	}
	return ""
}

func (s *SlurmService) resolveSlurmConfigTemplatePaths() []string {
	userProvided := strings.TrimSpace(os.Getenv("SLURM_BASE_CONFIG_PATH"))
	candidates := make([]string, 0, 10)
	if userProvided != "" {
		candidates = append(candidates, userProvided)
	}
	defaultRelPaths := []string{
		"config/slurm/slurm.conf.base",
		"src/backend/config/slurm/slurm.conf.base",
		"../config/slurm/slurm.conf.base",
		"../../config/slurm/slurm.conf.base",
	}
	candidates = append(candidates, defaultRelPaths...)
	candidates = append(candidates,
		"/app/config/slurm/slurm.conf.base",
		"/opt/ai-infra/config/slurm/slurm.conf.base",
	)

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "config", "slurm", "slurm.conf.base"))
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "config", "slurm", "slurm.conf.base"),
			filepath.Join(filepath.Dir(exeDir), "config", "slurm", "slurm.conf.base"),
		)
	}

	return uniquePaths(candidates)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		clean := filepath.Clean(p)
		if clean == "." || clean == ".." {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

func (s *SlurmService) extractGlobalSlurmConfig(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	globalLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "NodeName=") || strings.HasPrefix(trimmed, "PartitionName=") {
			break
		}
		if strings.Contains(trimmed, "节点配置（动态生成）") || strings.Contains(trimmed, "计算节点配置") || strings.Contains(trimmed, "Placeholder for dynamically generated node and partition blocks") {
			break
		}
		globalLines = append(globalLines, line)
	}
	return strings.TrimSpace(strings.Join(globalLines, "\n"))
}

var defaultSlurmGlobalConfig = strings.TrimSpace(`
#
# AI Infrastructure Matrix SLURM Configuration
# (fallback template)
#
ClusterName=ai-infra-cluster
ControlMachine=slurm-master
ControlAddr=slurm-master

# Authentication
AuthType=auth/munge
AuthAltTypes=auth/jwt
AuthAltParameters=jwt_key=/etc/slurm/jwt_hs256.key

# Accounting
AccountingStorageType=accounting_storage/slurmdbd
AccountingStorageHost=slurm-master
AccountingStoragePort=6818
AccountingStorageUser=root
AccountingStorageEnforce=associations,limits,qos
AccountingStoreFlags=job_comment,job_script,job_env

# Scheduling
SchedulerType=sched/backfill
SelectType=select/cons_tres
SelectTypeParameters=CR_Core_Memory

# Networking / state
SlurmctldPort=6817
SlurmdPort=6818
SlurmdSpoolDir=/var/spool/slurm/slurmd
StateSaveLocation=/var/lib/slurm/slurmctld
SlurmctldParameters=enable_configless

# Logging
SlurmctldLogFile=/var/log/slurm/slurmctld.log
SlurmdLogFile=/var/log/slurm/slurmd.log
SlurmctldDebug=info
SlurmdDebug=info

# Service config
SlurmUser=slurm
SlurmdUser=root
SlurmctldPidFile=/var/run/slurm/slurmctld.pid
SlurmdPidFile=/var/run/slurm/slurmd.pid

# Timeouts
MessageTimeout=60
KillWait=30
MinJobAge=300
Waittime=0
ReturnToService=1

# Plugin lookup
PluginDir=/usr/lib/slurm-wlm:/usr/lib/slurm:/usr/lib64/slurm:/usr/lib/x86_64-linux-gnu/slurm:/usr/lib/aarch64-linux-gnu/slurm

# Task / MPI
TaskPlugin=task/affinity
ProctrackType=proctrack/linuxproc
MpiDefault=pmix

# Job completion
JobCompType=jobcomp/filetxt
JobCompLoc=/var/log/slurm/jobcomp.log

# Limits
MaxJobCount=10000
MaxArraySize=10000
`)

func (s *SlurmService) fetchRemoteSlurmConfig() (string, error) {
	slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath := s.getSlurmMasterSSHConfig()
	return s.executeSSHCommandWithKey(slurmMasterHost, slurmMasterPort, slurmMasterUser, slurmMasterPassword, keyPath, "cat /etc/slurm/slurm.conf")
}

// StartAutoRegisterLoop 自动同步已有SLURM节点到数据库并刷新配置
func (s *SlurmService) StartAutoRegisterLoop() {
	if !slurmAutoRegisterEnabled() {
		log.Printf("[INFO] SLURM自动注册已禁用")
		return
	}
	if s.db == nil {
		log.Printf("[INFO] 数据库未就绪，跳过SLURM自动注册")
		return
	}
	delay := autoRegisterInitialDelay()
	log.Printf("[INFO] SLURM自动注册将在 %s 后执行", delay)
	go func() {
		time.Sleep(delay)
		for attempt := 1; attempt <= autoRegisterMaxAttempts; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			err := s.autoDiscoverAndRegisterNodes(ctx)
			cancel()
			if err == nil {
				log.Printf("[INFO] SLURM自动注册完成")
				return
			}
			log.Printf("[WARN] SLURM自动注册第 %d 次失败: %v", attempt, err)
			time.Sleep(time.Duration(autoRegisterRetrySecond) * time.Second)
		}
		log.Printf("[ERROR] SLURM自动注册多次失败，已放弃")
	}()
}

func (s *SlurmService) autoDiscoverAndRegisterNodes(ctx context.Context) error {
	nodes, demo, err := s.GetNodes(ctx)
	if err != nil {
		return fmt.Errorf("获取SLURM节点失败: %w", err)
	}
	if demo {
		return fmt.Errorf("SLURM命令不可用，无法自动注册节点")
	}
	if len(nodes) == 0 {
		return fmt.Errorf("未从SLURM读取到任何节点")
	}
	seen := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		name := strings.TrimSpace(node.Name)
		if name == "" {
			continue
		}
		if err := s.upsertNodeFromSlurm(ctx, node); err != nil {
			log.Printf("[WARN] 写入节点 %s 失败: %v", name, err)
			continue
		}
		seen[name] = struct{}{}
	}
	if len(seen) == 0 {
		return fmt.Errorf("没有节点写入数据库")
	}
	if err := s.markMissingNodesInactive(ctx, seen); err != nil {
		log.Printf("[WARN] 标记缺失节点失败: %v", err)
	}
	if err := s.UpdateSlurmConfig(ctx, NewSSHService()); err != nil {
		return fmt.Errorf("刷新SLURM配置失败: %w", err)
	}
	return nil
}

func (s *SlurmService) upsertNodeFromSlurm(ctx context.Context, node SlurmNode) error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	name := strings.TrimSpace(node.Name)
	status := mapSlurmStateToStatus(node.State)
	cpus := parseNumericField(node.CPUs)
	memory := parseNumericField(node.MemoryMB)
	data := map[string]interface{}{
		"status":     status,
		"host":       name,
		"cpus":       cpus,
		"memory":     memory,
		"updated_at": time.Now(),
	}
	var existing models.SlurmNode
	result := s.db.WithContext(ctx).Where("node_name = ?", name).First(&existing)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newNode := models.SlurmNode{
				ClusterID: defaultSlurmClusterID,
				NodeName:  name,
				NodeType:  "compute",
				Host:      name,
				Port:      22,
				Username:  "root",
				Status:    status,
				CPUs:      cpus,
				Memory:    memory,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return s.db.WithContext(ctx).Create(&newNode).Error
		}
		return result.Error
	}
	return s.db.WithContext(ctx).Model(&existing).Updates(data).Error
}

func (s *SlurmService) markMissingNodesInactive(ctx context.Context, seen map[string]struct{}) error {
	if s.db == nil || len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return s.db.WithContext(ctx).
		Model(&models.SlurmNode{}).
		Where("cluster_id = ?", defaultSlurmClusterID).
		Where("node_name NOT IN ?", names).
		Updates(map[string]interface{}{
			"status":     "inactive",
			"updated_at": time.Now(),
		}).
		Error
}

func mapSlurmStateToStatus(state string) string {
	upper := strings.ToUpper(strings.TrimSpace(state))
	switch {
	case strings.Contains(upper, "IDLE"), strings.Contains(upper, "ALLOC"), strings.Contains(upper, "MIXED"), strings.Contains(upper, "COMPLETING"):
		return "active"
	case strings.Contains(upper, "DOWN"), strings.Contains(upper, "DRAIN"), strings.Contains(upper, "FAIL"), strings.Contains(upper, "POWER"):
		return "inactive"
	default:
		return "pending"
	}
}

func parseNumericField(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var builder strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			break
		}
	}
	if builder.Len() == 0 {
		return 0
	}
	value, err := strconv.Atoi(builder.String())
	if err != nil {
		return 0
	}
	return value
}

func slurmAutoRegisterEnabled() bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv("SLURM_AUTO_REGISTER")))
	return val == "" || val == "true" || val == "1" || val == "yes"
}

func autoRegisterInitialDelay() time.Duration {
	raw := strings.TrimSpace(os.Getenv("SLURM_AUTO_REGISTER_DELAY_SECONDS"))
	if raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(defaultAutoRegisterDelaySecond) * time.Second
}

// getSlurmControllerInfo 获取SLURM控制器连接信息
func (s *SlurmService) getSlurmControllerInfo() (string, int, error) {
	// 从SLURM master SSH配置中获取
	host, port, _, _, _ := s.getSlurmMasterSSHConfig()
	return host, port, nil
} // SSHServiceInterface 定义SSH服务接口以便测试
type SSHServiceInterface interface {
	UploadFile(host string, port int, user, password string, content []byte, remotePath string) error
	ExecuteCommand(host string, port int, user, password, command string) (string, error)
}

// ScaleDown 执行缩容操作
func (s *SlurmService) ScaleDown(ctx context.Context, nodeIDs []string) (*ScalingResult, error) {
	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	// 获取SLURM Master的连接信息
	masterHost, masterPort, masterUser, masterPassword, _ := s.getSlurmMasterSSHConfig()

	if masterPassword == "" {
		return nil, fmt.Errorf("未配置SLURM_MASTER_PASSWORD环境变量，无法连接SLURM Master")
	}

	// 建立SSH连接
	sshConfig := &ssh.ClientConfig{
		User: masterUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(masterPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", masterHost, masterPort), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SLURM Master失败: %v", err)
	}
	defer client.Close()

	// 对每个节点执行缩容操作
	for _, nodeID := range nodeIDs {
		cleanNodeID := normalizeNodeNameValue(nodeID, "", 0)
		nodeResult := NodeScalingResult{
			NodeID:  cleanNodeID,
			Success: false,
			Message: "",
		}
		if cleanNodeID == "" {
			nodeResult.Message = "节点名称无效，无法执行缩容"
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 步骤1: 将节点状态设置为DOWN
		downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason='缩容移除节点_%s'",
			cleanNodeID, time.Now().Format("20060102_150405"))

		session, err := client.NewSession()
		if err != nil {
			nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		output, err := session.CombinedOutput(downCmd)
		session.Close()

		if err != nil {
			nodeResult.Message = fmt.Sprintf("设置节点DOWN状态失败: %v, 输出: %s", err, string(output))
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 步骤2: 从slurm.conf中移除节点
		configPath := "/etc/slurm/slurm.conf"

		// 读取配置文件
		session, err = client.NewSession()
		if err != nil {
			nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		configData, err := session.CombinedOutput(fmt.Sprintf("cat %s", configPath))
		session.Close()

		if err != nil {
			nodeResult.Message = fmt.Sprintf("读取slurm.conf失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 移除包含该节点的行
		lines := strings.Split(string(configData), "\n")
		var newLines []string
		removed := false
		for _, line := range lines {
			// 跳过包含该节点名称的NodeName行
			if strings.Contains(line, "NodeName="+nodeID) ||
				(strings.HasPrefix(line, "NodeName=") && strings.Contains(line, nodeID)) {
				removed = true
				continue
			}
			newLines = append(newLines, line)
		}

		if removed {
			// 写回配置文件
			newConfig := strings.Join(newLines, "\n")

			session, err = client.NewSession()
			if err != nil {
				nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			// 使用临时文件并移动的方式更新配置
			tmpPath := "/tmp/slurm.conf.tmp"
			writeCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF\nmv %s %s", tmpPath, newConfig, tmpPath, configPath)
			output, err = session.CombinedOutput(writeCmd)
			session.Close()

			if err != nil {
				nodeResult.Message = fmt.Sprintf("更新slurm.conf失败: %v, 输出: %s", err, string(output))
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			// 步骤3: 重新加载SLURM配置
			session, err = client.NewSession()
			if err != nil {
				nodeResult.Message = fmt.Sprintf("创建SSH会话失败: %v", err)
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}

			output, err = session.CombinedOutput("scontrol reconfigure")
			session.Close()

			if err != nil {
				nodeResult.Message = fmt.Sprintf("重新加载SLURM配置失败: %v, 输出: %s", err, string(output))
				result.Results = append(result.Results, nodeResult)
				result.Success = false
				continue
			}
		} else {
			nodeResult.Message = fmt.Sprintf("在slurm.conf中未找到节点 %s", nodeID)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 成功
		nodeResult.Success = true
		nodeResult.Message = "节点已成功从SLURM集群中移除"
		result.Results = append(result.Results, nodeResult)
	}

	// 如果所有操作都失败，整体标记为失败
	allFailed := true
	for _, r := range result.Results {
		if r.Success {
			allFailed = false
			break
		}
	}
	if allFailed {
		result.Success = false
	}

	return result, nil
}

// GetNodeTemplates 获取节点模板列表
func (s *SlurmService) GetNodeTemplates(ctx context.Context) ([]NodeTemplate, error) {
	// 如果数据库没有数据，返回一些默认模板
	var templates []models.NodeTemplate
	if err := s.db.Find(&templates).Error; err != nil {
		// 如果数据库查询失败，返回默认模板
		defaultTemplates := []NodeTemplate{
			{
				ID:          "small",
				Name:        "小型计算节点",
				Description: "2核4GB内存，适合轻量级计算任务",
				Config: NodeConfig{
					Host:     "compute-small",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"small", "compute"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "medium",
				Name:        "中型计算节点",
				Description: "4核8GB内存，适合中等规模计算任务",
				Config: NodeConfig{
					Host:     "compute-medium",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"medium", "compute"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "large",
				Name:        "大型计算节点",
				Description: "8核16GB内存，适合大规模计算任务",
				Config: NodeConfig{
					Host:     "compute-large",
					Port:     22,
					User:     "root",
					KeyPath:  "",
					Password: "",
					MinionID: "",
				},
				Tags:      []string{"large", "compute", "gpu"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		return defaultTemplates, nil
	}

	// 转换数据库中的模板为服务层结构
	result := make([]NodeTemplate, len(templates))
	for i, t := range templates {
		// 安全地转换模型层的 NodeConfig
		serviceConfig := NodeConfig{
			Host:     "localhost", // 默认主机
			Port:     22,          // 默认SSH端口
			User:     "root",      // 默认用户
			KeyPath:  "",
			Password: "",
			MinionID: "",
		}

		// 如果配置中有分区信息，使用第一个作为主机名
		if len(t.Config.Partitions) > 0 {
			serviceConfig.Host = t.Config.Partitions[0]
		}

		result[i] = NodeTemplate{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Config:      serviceConfig,
			Tags:        t.Tags,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		}
	}
	return result, nil
}

// CreateNodeTemplate 创建节点模板
func (s *SlurmService) CreateNodeTemplate(ctx context.Context, template *NodeTemplate) error {
	// 获取当前用户ID（需要从context中获取，这里暂时使用默认值）
	userID := uint(1) // TODO: 从JWT中获取用户ID

	// 将服务层的 NodeConfig 转换为模型层的 NodeConfig
	modelConfig := models.NodeConfig{
		Partitions:     []string{template.Config.Host}, // 使用主机名作为分区
		Features:       []string{},
		CustomSettings: map[string]string{},
		Mounts:         []models.MountConfig{},
	}

	model := &models.NodeTemplate{
		ID:          generateTemplateID(),
		Name:        template.Name,
		Description: template.Description,
		Config:      modelConfig,
		Tags:        template.Tags,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	template.ID = model.ID
	template.CreatedAt = model.CreatedAt
	template.UpdatedAt = model.UpdatedAt
	return nil
}

// UpdateNodeTemplate 更新节点模板
func (s *SlurmService) UpdateNodeTemplate(ctx context.Context, id string, template *NodeTemplate) error {
	var model models.NodeTemplate
	if err := s.db.Where("id = ?", id).First(&model).Error; err != nil {
		return err
	}

	// 将服务层的 NodeConfig 转换为模型层的 NodeConfig
	modelConfig := models.NodeConfig{
		Partitions:     []string{template.Config.Host}, // 使用主机名作为分区
		Features:       []string{},
		CustomSettings: map[string]string{},
		Mounts:         []models.MountConfig{},
	}

	model.Name = template.Name
	model.Description = template.Description
	model.Config = modelConfig
	model.Tags = template.Tags
	model.UpdatedAt = time.Now()

	return s.db.Save(&model).Error
}

// DeleteNodeTemplate 删除节点模板
func (s *SlurmService) DeleteNodeTemplate(ctx context.Context, id string) error {
	return s.db.Where("id = ?", id).Delete(&models.NodeTemplate{}).Error
}

// generateOperationID 生成操作ID
func generateOperationID() string {
	return fmt.Sprintf("op-%d", time.Now().Unix())
}

// SLURM REST API 相关结构体和方法

// SlurmNodeSpec REST API节点规格
type SlurmNodeSpec struct {
	NodeName string `json:"name"`
	CPUs     int    `json:"cpus"`
	Memory   int    `json:"real_memory"` // MB
	Features string `json:"features,omitempty"`
	Gres     string `json:"gres,omitempty"`
	State    string `json:"state,omitempty"`
}

// SlurmAPIResponse REST API响应
type SlurmAPIResponse struct {
	Meta   map[string]interface{} `json:"meta,omitempty"`
	Errors []SlurmAPIError        `json:"errors,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
}

// SlurmAPIError REST API错误
type SlurmAPIError struct {
	Error       string `json:"error"`
	ErrorNumber int    `json:"error_number"`
	Source      string `json:"source,omitempty"`
}

// SlurmNodeUpdate 节点更新请求
type SlurmNodeUpdate struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// getJWTToken 获取SLURM JWT token
func (s *SlurmService) getJWTToken(ctx context.Context) (string, error) {
	// 尝试从环境变量获取预设的token
	if token := os.Getenv("SLURM_JWT_TOKEN"); token != "" {
		return token, nil
	}

	// 通过SSH连接获取token
	masterHost := os.Getenv("SLURM_MASTER_HOST")
	if masterHost == "" {
		masterHost = "slurm-master"
	}

	masterPortStr := os.Getenv("SLURM_MASTER_PORT")
	masterPort := 22
	if masterPortStr != "" {
		if port, err := strconv.Atoi(masterPortStr); err == nil {
			masterPort = port
		}
	}

	masterUser := os.Getenv("SLURM_MASTER_USER")
	if masterUser == "" {
		masterUser = "root"
	}

	var authMethods []ssh.AuthMethod

	// 优先使用私钥（与容器默认配置一致）
	keyPath := os.Getenv("SLURM_MASTER_PRIVATE_KEY")
	if keyPath == "" {
		keyPath = "/root/.ssh/id_rsa"
	}
	if keyData, err := os.ReadFile(keyPath); err == nil {
		if passphrase := os.Getenv("SLURM_MASTER_PRIVATE_KEY_PASSPHRASE"); passphrase != "" {
			if signer, err := ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase)); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else {
				log.Printf("[WARN] 无法解析带口令的SLURM私钥 %s: %v", keyPath, err)
			}
		} else {
			if signer, err := ssh.ParsePrivateKey(keyData); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else {
				log.Printf("[WARN] 无法解析SLURM私钥 %s: %v", keyPath, err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("[WARN] 读取SLURM私钥失败 (%s): %v", keyPath, err)
	}

	// 兼容旧配置：支持密码认证
	if masterPassword := os.Getenv("SLURM_MASTER_PASSWORD"); masterPassword != "" {
		authMethods = append(authMethods, ssh.Password(masterPassword))
	}

	if len(authMethods) == 0 {
		return "", fmt.Errorf("未配置可用的SLURM Master认证方式 (私钥或密码)")
	}

	sshConfig := &ssh.ClientConfig{
		User:            masterUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", masterHost, masterPort), sshConfig)
	if err != nil {
		return "", fmt.Errorf("连接SLURM Master失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// 获取token
	output, err := session.CombinedOutput("scontrol token lifespan=3600")
	if err != nil {
		return "", fmt.Errorf("获取JWT token失败: %v, 输出: %s", err, string(output))
	}

	// 解析token输出 (格式: SLURM_JWT=xxxxx)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SLURM_JWT=") {
			return strings.TrimPrefix(line, "SLURM_JWT="), nil
		}
	}

	return "", fmt.Errorf("未找到有效的JWT token")
}

// callSlurmAPI 调用SLURM REST API
func (s *SlurmService) callSlurmAPI(ctx context.Context, method, endpoint string, body interface{}) (*SlurmAPIResponse, error) {
	// 获取JWT token
	token, err := s.getJWTToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取JWT token失败: %v", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/slurm/v0.0.41%s", s.restAPIURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %v", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("X-SLURM-USER-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp SlurmAPIResponse
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		// 如果不是JSON，返回原始响应
		log.Printf("[WARN] 解析JSON响应失败: %v, 响应: %s", err, string(respData))
		return &SlurmAPIResponse{
			Data: string(respData),
		}, nil
	}

	// 检查API错误
	if len(apiResp.Errors) > 0 {
		return &apiResp, fmt.Errorf("SLURM API错误: %s", apiResp.Errors[0].Error)
	}

	return &apiResp, nil
}

// AddNodeViaAPI 通过REST API添加节点
func (s *SlurmService) AddNodeViaAPI(ctx context.Context, nodeSpec SlurmNodeSpec) error {
	log.Printf("[DEBUG] AddNodeViaAPI: 添加节点 %s", nodeSpec.NodeName)

	// 调用API添加节点
	resp, err := s.callSlurmAPI(ctx, "POST", "/nodes", map[string]interface{}{
		"nodes": []SlurmNodeSpec{nodeSpec},
	})

	if err != nil {
		return fmt.Errorf("添加节点失败: %v", err)
	}

	log.Printf("[DEBUG] AddNodeViaAPI: 节点 %s 添加成功, 响应: %+v", nodeSpec.NodeName, resp)
	return nil
}

// UpdateNodeViaAPI 通过REST API更新节点状态
func (s *SlurmService) UpdateNodeViaAPI(ctx context.Context, nodeName string, update SlurmNodeUpdate) error {
	log.Printf("[DEBUG] UpdateNodeViaAPI: 更新节点 %s 状态为 %s", nodeName, update.State)

	// 调用API更新节点
	resp, err := s.callSlurmAPI(ctx, "POST", fmt.Sprintf("/node/%s", nodeName), update)
	if err != nil {
		return fmt.Errorf("更新节点状态失败: %v", err)
	}

	log.Printf("[DEBUG] UpdateNodeViaAPI: 节点 %s 状态更新成功, 响应: %+v", nodeName, resp)
	return nil
}

// CancelJobViaAPI 通过REST API取消作业
func (s *SlurmService) CancelJobViaAPI(ctx context.Context, jobID string, signal string) error {
	log.Printf("[DEBUG] CancelJobViaAPI: 取消作业 %s (信号: %s)", jobID, signal)

	payload := map[string]interface{}{
		"job_id": jobID,
	}
	if signal != "" {
		payload["signal"] = signal
	}

	resp, err := s.callSlurmAPI(ctx, "DELETE", fmt.Sprintf("/job/%s", jobID), payload)
	if err != nil {
		return fmt.Errorf("取消作业失败: %v", err)
	}

	log.Printf("[DEBUG] CancelJobViaAPI: 作业 %s 取消成功, 响应: %+v", jobID, resp)
	return nil
}

// UpdateJobViaAPI 通过REST API更新作业状态
func (s *SlurmService) UpdateJobViaAPI(ctx context.Context, jobID string, action string) error {
	log.Printf("[DEBUG] UpdateJobViaAPI: 对作业 %s 执行 %s 操作", jobID, action)

	payload := map[string]interface{}{
		"job_id": jobID,
		"action": action,
	}

	resp, err := s.callSlurmAPI(ctx, "POST", fmt.Sprintf("/job/%s", jobID), payload)
	if err != nil {
		return fmt.Errorf("更新作业状态失败: %v", err)
	}

	log.Printf("[DEBUG] UpdateJobViaAPI: 作业 %s 操作 %s 成功, 响应: %+v", jobID, action, resp)
	return nil
}

// DeleteNodeViaAPI 通过REST API删除节点
func (s *SlurmService) DeleteNodeViaAPI(ctx context.Context, nodeName string) error {
	log.Printf("[DEBUG] DeleteNodeViaAPI: 删除节点 %s", nodeName)

	// 先将节点设置为DOWN状态
	err := s.UpdateNodeViaAPI(ctx, nodeName, SlurmNodeUpdate{
		State:  "DOWN",
		Reason: fmt.Sprintf("节点删除_%s", time.Now().Format("20060102_150405")),
	})
	if err != nil {
		return fmt.Errorf("设置节点DOWN状态失败: %v", err)
	}

	// 调用API删除节点
	resp, err := s.callSlurmAPI(ctx, "DELETE", fmt.Sprintf("/node/%s", nodeName), nil)
	if err != nil {
		return fmt.Errorf("删除节点失败: %v", err)
	}

	log.Printf("[DEBUG] DeleteNodeViaAPI: 节点 %s 删除成功, 响应: %+v", nodeName, resp)
	return nil
}

// ScaleUpViaAPI 通过REST API扩容
func (s *SlurmService) ScaleUpViaAPI(ctx context.Context, nodes []NodeConfig) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleUpViaAPI: 开始基于REST API的扩容操作，节点数量: %d", len(nodes))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	for _, node := range nodes {
		nodeResult := NodeScalingResult{
			NodeID:  node.Host,
			Success: false,
			Message: "",
		}

		// 创建节点规格 (从配置读取，提供默认值)
		cpus := node.CPUs
		if cpus == 0 {
			cpus = 4 // 默认4核
		}

		memory := node.Memory
		if memory == 0 {
			memory = 8192 // 默认8GB内存 (MB)
		}

		features := node.Features
		if features == "" {
			features = "compute" // 默认特性
		}

		nodeSpec := SlurmNodeSpec{
			NodeName: node.Host,
			CPUs:     cpus,
			Memory:   memory,
			Features: features,
			State:    "DOWN", // 新节点初始状态设置为 DOWN，等待手动激活
		}

		log.Printf("[DEBUG] ScaleUpViaAPI: 节点 %s 配置: CPUs=%d, Memory=%d MB, Features=%s, State=DOWN (需手动激活)",
			node.Host, cpus, memory, features)

		// 通过API添加节点
		if err := s.AddNodeViaAPI(ctx, nodeSpec); err != nil {
			nodeResult.Message = fmt.Sprintf("API添加节点失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 更新数据库状态
		if s.db != nil {
			var dbNode models.SlurmNode
			err := s.db.Where("host = ? OR node_name = ?", node.Host, node.MinionID).First(&dbNode).Error

			if err == gorm.ErrRecordNotFound {
				// 创建新节点记录
				dbNode = models.SlurmNode{
					NodeName:     node.Host,
					Host:         node.Host,
					CPUs:         4,    // 默认4核
					Memory:       8192, // 默认8GB内存
					Status:       "active",
					SaltMinionID: node.MinionID,
				}
				if err := s.db.Create(&dbNode).Error; err != nil {
					log.Printf("[WARN] 创建节点 %s 数据库记录失败: %v", node.Host, err)
				}
			} else if err == nil {
				// 更新现有记录
				if err := s.db.Model(&dbNode).Updates(map[string]interface{}{
					"status":         "active",
					"salt_minion_id": node.MinionID,
				}).Error; err != nil {
					log.Printf("[WARN] 更新节点 %s 数据库记录失败: %v", node.Host, err)
				}
			}
		}

		nodeResult.Success = true
		nodeResult.Message = "节点已添加到SLURM集群(状态: DOWN)，请确认slurmd运行后执行: scontrol update NodeName=" + node.Host + " State=RESUME"
		result.Results = append(result.Results, nodeResult)

		log.Printf("[DEBUG] 节点 %s 通过REST API扩容成功", node.Host)
	}

	log.Printf("[INFO] ScaleUpViaAPI 完成，所有节点已添加为 DOWN 状态，需手动激活")

	return result, nil
}

// ScaleDownViaAPI 通过REST API缩容
func (s *SlurmService) ScaleDownViaAPI(ctx context.Context, nodeIDs []string) (*ScalingResult, error) {
	log.Printf("[DEBUG] ScaleDownViaAPI: 开始基于REST API的缩容操作，节点数量: %d", len(nodeIDs))

	result := &ScalingResult{
		OperationID: generateOperationID(),
		Success:     true,
		Results:     []NodeScalingResult{},
	}

	for _, nodeID := range nodeIDs {
		nodeResult := NodeScalingResult{
			NodeID:  nodeID,
			Success: false,
			Message: "",
		}

		// 通过API删除节点
		if err := s.DeleteNodeViaAPI(ctx, nodeID); err != nil {
			nodeResult.Message = fmt.Sprintf("API删除节点失败: %v", err)
			result.Results = append(result.Results, nodeResult)
			result.Success = false
			continue
		}

		// 更新数据库状态
		if s.db != nil {
			if err := s.db.Model(&models.SlurmNode{}).Where("node_name = ? OR host = ?", nodeID, nodeID).
				Updates(map[string]interface{}{
					"status": "removed",
				}).Error; err != nil {
				log.Printf("[WARN] 更新节点 %s 数据库状态失败: %v", nodeID, err)
			}
		}

		nodeResult.Success = true
		nodeResult.Message = "节点成功从SLURM集群移除"
		result.Results = append(result.Results, nodeResult)

		log.Printf("[DEBUG] 节点 %s 通过REST API缩容成功", nodeID)
	}

	return result, nil
}

// ReloadSlurmConfig 重新加载SLURM配置
func (s *SlurmService) ReloadSlurmConfig(ctx context.Context) error {
	log.Printf("[DEBUG] ReloadSlurmConfig: 重新加载SLURM配置")

	// 调用API重新加载配置
	resp, err := s.callSlurmAPI(ctx, "POST", "/reconfigure", nil)
	if err != nil {
		return fmt.Errorf("重新加载SLURM配置失败: %v", err)
	}

	log.Printf("[DEBUG] SLURM配置重新加载成功, 响应: %+v", resp)
	return nil
}

// generateTemplateID 生成模板ID
func generateTemplateID() string {
	return fmt.Sprintf("tmpl-%d", time.Now().Unix())
}

// InstallSlurmNodeRequest 安装SLURM节点请求
type InstallSlurmNodeRequest struct {
	NodeName   string `json:"node_name"`   // 节点名称（容器名）
	OSType     string `json:"os_type"`     // 操作系统类型：rocky 或 ubuntu
	SSHHost    string `json:"ssh_host"`    // SSH连接地址（IP或主机名）
	SSHPort    int    `json:"ssh_port"`    // SSH端口，默认22
	SSHUser    string `json:"ssh_user"`    // SSH用户，默认root
	Password   string `json:"password"`    // 初始密码（用于首次连接和部署SSH密钥）
	KeyPath    string `json:"key_path"`    // SSH私钥文件路径
	PrivateKey string `json:"private_key"` // SSH私钥内容（内联）
}

// InstallSlurmNodeResponse 安装SLURM节点响应
type InstallSlurmNodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Logs    string `json:"logs"` // 安装日志
}

// InstallSlurmNode 在指定节点上安装SLURM客户端和slurmd
func (s *SlurmService) InstallSlurmNode(ctx context.Context, req InstallSlurmNodeRequest) (*InstallSlurmNodeResponse, error) {
	log.Printf("[INFO] 开始在节点 %s 上安装SLURM (OS: %s, Host: %s)", req.NodeName, req.OSType, req.SSHHost)

	var logBuffer bytes.Buffer
	logWriter := io.MultiWriter(os.Stdout, &logBuffer)

	// 设置默认值
	if req.SSHPort == 0 {
		req.SSHPort = 22
	}
	if req.SSHUser == "" {
		req.SSHUser = "root"
	}
	if req.SSHHost == "" {
		req.SSHHost = req.NodeName // 默认使用节点名作为主机名
	}

	// 创建任务记录到数据库（如果数据库可用）
	var taskID string
	var dbTask *models.NodeInstallTask
	if s.db != nil {
		taskID = fmt.Sprintf("install-%s-%d", req.NodeName, time.Now().Unix())

		// 查找或创建节点记录以获取NodeID
		var nodeRecord models.SlurmNode
		result := s.db.Where("node_name = ? OR host = ?", req.NodeName, req.SSHHost).First(&nodeRecord)
		if result.Error != nil {
			// 如果节点不存在，创建一个新节点记录
			nodeRecord = models.SlurmNode{
				ClusterID: 1, // 默认集群ID
				NodeName:  req.NodeName,
				NodeType:  "compute",
				Host:      req.SSHHost,
				Port:      req.SSHPort,
				Username:  req.SSHUser,
				AuthType:  "key",
				Status:    "installing",
				CPUs:      0, // 安装后会更新
				Memory:    0,
				Storage:   0,
			}
			if err := s.db.Create(&nodeRecord).Error; err != nil {
				log.Printf("[WARN] 创建节点记录失败: %v", err)
			}
		}

		dbTask = &models.NodeInstallTask{
			TaskID:       taskID,
			NodeID:       nodeRecord.ID,
			TaskType:     "scale-up",
			Status:       "running",
			Progress:     0,
			ErrorMessage: "",
		}

		// 设置开始时间
		now := time.Now()
		dbTask.StartedAt = &now

		if err := s.db.Create(dbTask).Error; err != nil {
			log.Printf("[WARN] 创建任务记录失败: %v", err)
			// 继续安装，不因为任务记录创建失败而中断
		} else {
			fmt.Fprintf(logWriter, "[INFO] 任务ID: %s\n", taskID)
		}
	}

	// 定义更新任务进度的辅助函数
	updateTaskProgress := func(progress int, status string, errorMsg string) {
		if s.db != nil && dbTask != nil {
			updates := map[string]interface{}{
				"progress": progress,
				"status":   status,
			}
			if errorMsg != "" {
				updates["error_message"] = errorMsg
			}
			if status == "completed" || status == "failed" {
				now := time.Now()
				updates["completed_at"] = &now
			}
			s.db.Model(dbTask).Updates(updates)
		}
	}

	// 1. 获取slurm.conf和munge.key (进度: 0-10%)
	log.Printf("[INFO] 从slurm-master获取配置文件...")
	fmt.Fprintf(logWriter, "[INFO] [10%%] 获取配置文件...\n")
	updateTaskProgress(10, "running", "")

	slurmConf, err := s.getSlurmMasterConfig(ctx)
	if err != nil {
		updateTaskProgress(10, "failed", fmt.Sprintf("获取slurm.conf失败: %v", err))
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("获取slurm.conf失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	mungeKey, err := s.getMungeKey(ctx)
	if err != nil {
		log.Printf("[WARN] 获取munge.key失败: %v，将继续安装", err)
		mungeKey = nil
	}

	// 2. 部署SSH密钥到新节点（进度: 10-25%）
	fmt.Fprintf(logWriter, "[INFO] [25%%] 部署SSH密钥...\n")
	updateTaskProgress(25, "running", "")

	if err := s.deploySSHKeyToNode(ctx, req, logWriter); err != nil {
		updateTaskProgress(25, "failed", fmt.Sprintf("部署SSH密钥失败: %v", err))
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("部署SSH密钥失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 2.5. 安装 Salt-Minion（进度: 25-40%）
	fmt.Fprintf(logWriter, "[INFO] [35%%] 安装 Salt-Minion...\n")
	updateTaskProgress(35, "running", "")

	if err := s.installSaltMinionOnNode(ctx, req, logWriter); err != nil {
		// Salt-Minion 安装失败不阻塞 SLURM 安装，只记录警告
		log.Printf("[WARN] 安装 Salt-Minion 失败: %v，继续 SLURM 安装", err)
		fmt.Fprintf(logWriter, "[WARN] Salt-Minion 安装失败: %v，继续 SLURM 安装...\n", err)
	}

	// 3. 根据OS类型安装SLURM（进度: 40-65%）
	fmt.Fprintf(logWriter, "[INFO] [50%%] 安装SLURM包...\n")
	updateTaskProgress(50, "running", "")

	if err := s.installSlurmPackages(ctx, req, logWriter); err != nil {
		updateTaskProgress(50, "failed", fmt.Sprintf("安装SLURM包失败: %v", err))
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("安装SLURM包失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 4. 配置文件（进度: 65-80%）
	fmt.Fprintf(logWriter, "[INFO] [75%%] 配置节点...\n")
	updateTaskProgress(75, "running", "")

	if err := s.configureSlurmNode(ctx, req, slurmConf, mungeKey, logWriter); err != nil {
		updateTaskProgress(75, "failed", fmt.Sprintf("配置节点失败: %v", err))
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("配置节点失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 5. 启动服务（进度: 80-95%）
	fmt.Fprintf(logWriter, "[INFO] [90%%] 启动服务...\n")
	updateTaskProgress(90, "running", "")

	if err := s.startSlurmServices(ctx, req, logWriter); err != nil {
		updateTaskProgress(90, "failed", fmt.Sprintf("启动服务失败: %v", err))
		return &InstallSlurmNodeResponse{
			Success: false,
			Message: fmt.Sprintf("启动服务失败: %v", err),
			Logs:    logBuffer.String(),
		}, err
	}

	// 6. 安装完成（进度: 100%）
	fmt.Fprintf(logWriter, "[INFO] [100%%] 安装完成!\n")
	updateTaskProgress(100, "completed", "")

	// 更新节点状态为active
	if s.db != nil && dbTask != nil {
		s.db.Model(&models.SlurmNode{}).Where("id = ?", dbTask.NodeID).Update("status", "active")
	}

	log.Printf("[INFO] 节点 %s SLURM安装完成", req.NodeName)
	return &InstallSlurmNodeResponse{
		Success: true,
		Message: "SLURM安装成功",
		Logs:    logBuffer.String(),
	}, nil
}

// getSlurmMasterConfig 从slurm-master获取slurm.conf
func (s *SlurmService) getSlurmMasterConfig(ctx context.Context) ([]byte, error) {
	// 使用SSH连接到slurm-master读取配置文件
	masterHost, masterPort, _, _, keyPath := s.getSlurmMasterSSHConfig()

	output, err := s.executeSSHCommandWithKey(masterHost, masterPort, "root", "", keyPath, "cat /etc/slurm/slurm.conf")
	if err != nil {
		return nil, fmt.Errorf("读取slurm.conf失败: %v", err)
	}
	return []byte(output), nil
}

// getMungeKey 从slurm-master获取munge.key
func (s *SlurmService) getMungeKey(ctx context.Context) ([]byte, error) {
	// 使用SSH连接到slurm-master读取munge密钥
	masterHost, masterPort, _, _, keyPath := s.getSlurmMasterSSHConfig()

	output, err := s.executeSSHCommandWithKey(masterHost, masterPort, "root", "", keyPath, "cat /etc/munge/munge.key")
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// deploySSHKeyToNode 部署统一SSH公钥到新节点（支持密码或已有密钥认证）
func (s *SlurmService) deploySSHKeyToNode(ctx context.Context, req InstallSlurmNodeRequest, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 部署SSH公钥到节点 %s@%s:%d...\n", req.SSHUser, req.SSHHost, req.SSHPort)

	// 读取backend的SSH公钥（统一密钥）
	pubKeyPath := os.Getenv("HOME") + "/.ssh/id_rsa.pub"
	pubKeyContent, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("读取SSH公钥失败: %w", err)
	}

	// 准备部署命令
	setupCmd := fmt.Sprintf(`
		mkdir -p /root/.ssh && \
		chmod 700 /root/.ssh && \
		echo '%s' >> /root/.ssh/authorized_keys && \
		chmod 600 /root/.ssh/authorized_keys && \
		sort -u /root/.ssh/authorized_keys -o /root/.ssh/authorized_keys && \
		echo "SSH公钥已部署"
	`, strings.TrimSpace(string(pubKeyContent)))

	sshTarget := fmt.Sprintf("%s@%s", req.SSHUser, req.SSHHost)
	sshPort := fmt.Sprintf("%d", req.SSHPort)

	var cmd *exec.Cmd
	var authMethod string

	// 尝试方式1: 使用密码认证（如果提供了密码）
	if req.Password != "" {
		authMethod = "password"
		fmt.Fprintf(logWriter, "[INFO] 使用密码认证部署SSH密钥...\n")
		cmd = exec.CommandContext(ctx, "sshpass", "-p", req.Password,
			"ssh", "-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-p", sshPort,
			sshTarget,
			setupCmd)
	} else {
		// 方式2: 使用现有的SSH密钥认证
		authMethod = "existing-key"
		fmt.Fprintf(logWriter, "[INFO] 使用现有SSH密钥认证部署公钥...\n")
		cmd = exec.CommandContext(ctx, "ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-p", sshPort,
			sshTarget,
			setupCmd)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("部署SSH公钥失败 (认证方式: %s): %v, output: %s", authMethod, err, string(output))
	}

	fmt.Fprintf(logWriter, "[INFO] ✓ SSH公钥部署成功 (认证方式: %s)\n", authMethod)
	fmt.Fprintf(logWriter, "[DEBUG] %s\n", strings.TrimSpace(string(output)))

	// 测试密钥认证是否可用
	testCmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "PasswordAuthentication=no",
		"-p", sshPort,
		sshTarget,
		"echo 'SSH密钥认证OK'")

	if testOutput, testErr := testCmd.CombinedOutput(); testErr == nil {
		fmt.Fprintf(logWriter, "[INFO] ✓ SSH密钥认证测试成功: %s\n", strings.TrimSpace(string(testOutput)))
	} else {
		fmt.Fprintf(logWriter, "[WARN] SSH密钥认证测试失败: %v, 但继续进行\n", testErr)
	}

	return nil
}

// installSaltMinionOnNode 在节点上安装 Salt-Minion（通过 SSH 执行脚本）
func (s *SlurmService) installSaltMinionOnNode(ctx context.Context, req InstallSlurmNodeRequest, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 在 %s 上安装 Salt-Minion (OS: %s)\n", req.SSHHost, req.OSType)

	// 获取 Salt Master 配置
	saltMasterHost := os.Getenv("SALT_MASTER_HOST")
	if saltMasterHost == "" {
		saltMasterHost = "saltstack"
	}

	// 确定使用哪个安装脚本
	var scriptPath string
	if req.OSType == "ubuntu" || req.OSType == "debian" {
		scriptPath = "/app/scripts/install-salt-minion-deb.sh"
	} else if req.OSType == "rocky" || req.OSType == "centos" || req.OSType == "rhel" {
		scriptPath = "/app/scripts/install-salt-minion-rpm.sh"
	} else {
		return fmt.Errorf("不支持的操作系统类型: %s", req.OSType)
	}

	// 读取安装脚本
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取 Salt-Minion 安装脚本失败: %v", err)
	}

	// 准备远程执行脚本
	sshTarget := fmt.Sprintf("%s@%s", req.SSHUser, req.SSHHost)
	sshPort := fmt.Sprintf("%d", req.SSHPort)
	tmpScriptPath := "/tmp/install-salt-minion.sh"

	// 步骤1: 上传脚本到节点
	fmt.Fprintf(logWriter, "[INFO] 上传 Salt-Minion 安装脚本...\n")
	uploadCmd := fmt.Sprintf("cat > %s << 'EOF_SCRIPT'\n%s\nEOF_SCRIPT\nchmod +x %s",
		tmpScriptPath, string(scriptContent), tmpScriptPath)

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		uploadCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("上传 Salt-Minion 脚本失败: %v", err)
	}

	// 步骤2: 执行安装脚本
	fmt.Fprintf(logWriter, "[INFO] 执行 Salt-Minion 安装脚本...\n")
	executeCmd := fmt.Sprintf("SALT_MASTER_HOST=%s %s", saltMasterHost, tmpScriptPath)
	cmd = exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		executeCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 Salt-Minion 安装脚本失败: %v", err)
	}

	// 步骤3: 清理临时脚本
	cleanCmd := fmt.Sprintf("rm -f %s", tmpScriptPath)
	cmd = exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		cleanCmd)
	cmd.Run() // 忽略清理错误

	fmt.Fprintf(logWriter, "[INFO] ✓ Salt-Minion 安装成功\n")
	return nil
}

// installSlurmPackages 安装SLURM包（通过SSH执行）
func (s *SlurmService) installSlurmPackages(ctx context.Context, req InstallSlurmNodeRequest, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 在 %s 上安装SLURM包 (OS: %s)\n", req.SSHHost, req.OSType)

	// 读取安装脚本
	scriptPath := "/app/scripts/install-slurm-node.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取安装脚本失败 %s: %w", scriptPath, err)
	}

	// 获取apphub URL
	apphubURL := os.Getenv("APPHUB_URL")
	if apphubURL == "" {
		apphubURL = "http://ai-infra-apphub"
	}

	fmt.Fprintf(logWriter, "[INFO] AppHub URL: %s\n", apphubURL)

	// SSH目标
	sshTarget := fmt.Sprintf("%s@%s", req.SSHUser, req.SSHHost)
	sshPort := fmt.Sprintf("%d", req.SSHPort)

	// 步骤1: 上传脚本到节点
	tmpScriptPath := fmt.Sprintf("/tmp/install-slurm-%s.sh", req.NodeName)
	uploadCmd := fmt.Sprintf("cat > %s && chmod +x %s", tmpScriptPath, tmpScriptPath)

	fmt.Fprintf(logWriter, "[INFO] 上传安装脚本到节点...\n")
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		uploadCmd)
	cmd.Stdin = bytes.NewReader(scriptContent)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("上传脚本失败: %v", err)
	}

	// 步骤2: 执行安装脚本
	executeCmd := fmt.Sprintf("%s %s compute", tmpScriptPath, apphubURL)

	fmt.Fprintf(logWriter, "[INFO] 开始执行安装脚本...\n")
	cmd = exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		executeCmd)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装SLURM包失败: %v", err)
	}

	// 步骤3: 清理临时脚本
	cleanCmd := fmt.Sprintf("rm -f %s", tmpScriptPath)
	cmd = exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		cleanCmd)
	cmd.Run() // 忽略清理错误

	fmt.Fprintf(logWriter, "[INFO] ✓ SLURM包安装成功\n")
	return nil
}

// configureSlurmNode 配置SLURM节点（通过SCP和SSH）
func (s *SlurmService) configureSlurmNode(ctx context.Context, req InstallSlurmNodeRequest, slurmConf, mungeKey []byte, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 配置 %s 节点\n", req.NodeName)

	// SSH目标
	sshTarget := fmt.Sprintf("%s@%s", req.SSHUser, req.SSHHost)
	sshPort := fmt.Sprintf("%d", req.SSHPort)

	// 确定配置文件路径
	slurmConfPath := "/etc/slurm/slurm.conf"
	if req.OSType == "ubuntu" || req.OSType == "debian" {
		slurmConfPath = "/etc/slurm-llnl/slurm.conf"
	}

	// 写入本地临时文件
	tmpSlurmConf := fmt.Sprintf("/tmp/slurm.conf.%s", req.NodeName)
	if err := os.WriteFile(tmpSlurmConf, slurmConf, 0644); err != nil {
		return fmt.Errorf("写入临时slurm.conf失败: %v", err)
	}
	defer os.Remove(tmpSlurmConf)

	// 使用SCP复制slurm.conf到节点
	fmt.Fprintf(logWriter, "[INFO] 复制slurm.conf到节点...\n")
	scpCmd := exec.CommandContext(ctx, "scp",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-P", sshPort,
		tmpSlurmConf,
		fmt.Sprintf("%s:%s", sshTarget, slurmConfPath))
	scpCmd.Stdout = logWriter
	scpCmd.Stderr = logWriter
	if err := scpCmd.Run(); err != nil {
		return fmt.Errorf("复制slurm.conf失败: %v", err)
	}
	fmt.Fprintf(logWriter, "[INFO] slurm.conf复制成功 -> %s\n", slurmConfPath)

	// 修复PluginDir路径
	fixPluginDirScript := "src/backend/scripts/fix-slurm-plugindir.sh"
	scriptContent, err := os.ReadFile(fixPluginDirScript)
	if err == nil {
		fmt.Fprintf(logWriter, "[INFO] 修复PluginDir路径...\n")
		cmd := exec.CommandContext(ctx, "ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-p", sshPort,
			sshTarget,
			"bash -s")
		cmd.Stdin = bytes.NewReader(scriptContent)
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
		if err := cmd.Run(); err == nil {
			fmt.Fprintf(logWriter, "[INFO] PluginDir路径兼容性已修复\n")
		}
	}

	// 配置munge.key
	if mungeKey != nil && len(mungeKey) > 0 {
		tmpMungeKey := fmt.Sprintf("/tmp/munge.key.%s", req.NodeName)
		if err := os.WriteFile(tmpMungeKey, mungeKey, 0400); err != nil {
			return fmt.Errorf("写入临时munge.key失败: %v", err)
		}
		defer os.Remove(tmpMungeKey)

		fmt.Fprintf(logWriter, "[INFO] 复制munge.key到节点...\n")
		scpCmd = exec.CommandContext(ctx, "scp",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-P", sshPort,
			tmpMungeKey,
			fmt.Sprintf("%s:/etc/munge/munge.key", sshTarget))
		scpCmd.Stdout = logWriter
		scpCmd.Stderr = logWriter
		if err := scpCmd.Run(); err == nil {
			// 设置权限：munge.key必须属于munge用户且权限为400
			chownCmd := "chown munge:munge /etc/munge/munge.key && chmod 400 /etc/munge/munge.key"
			cmd := exec.CommandContext(ctx, "ssh",
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-p", sshPort,
				sshTarget,
				chownCmd)
			cmd.Run()
			fmt.Fprintf(logWriter, "[INFO] munge.key配置成功\n")
		}
	}

	return nil
}

// startSlurmServices 启动SLURM服务（通过SSH）
func (s *SlurmService) startSlurmServices(ctx context.Context, req InstallSlurmNodeRequest, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 启动 %s 上的服务\n", req.NodeName)

	// SSH目标
	sshTarget := fmt.Sprintf("%s@%s", req.SSHUser, req.SSHHost)
	sshPort := fmt.Sprintf("%d", req.SSHPort)

	// 读取启动脚本
	scriptPath := "/root/scripts/start-slurmd.sh"
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取启动脚本失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[INFO] 使用脚本启动 slurmd...\n")

	// 通过SSH执行脚本
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		"bash -s")
	cmd.Stdin = bytes.NewReader(scriptContent)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[SUCCESS] 服务启动完成\n")

	// 验证服务状态
	time.Sleep(2 * time.Second)

	checkCmd := "pgrep -x slurmd >/dev/null && echo 'slurmd running' || echo 'slurmd not running'"
	cmd = exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", sshPort,
		sshTarget,
		checkCmd)
	output, _ := cmd.Output()
	fmt.Fprintf(logWriter, "[INFO] 验证结果: %s\n", strings.TrimSpace(string(output)))

	return nil
}

// executeScriptViaSSH 通过SSH执行脚本（支持密码和密钥认证）
func (s *SlurmService) executeScriptViaSSH(ctx context.Context, host string, port int, user, password, privateKey, scriptPath string) (string, error) {
	// 读取脚本内容
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("读取脚本失败: %v", err)
	}

	// 通过SSH执行脚本
	return s.executeSSHCommandWithKey(host, port, user, password, privateKey, string(scriptContent))
}

// startSlurmServicesViaSSH 通过真实SSH启动SLURM服务（非docker exec）
func (s *SlurmService) startSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string, logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "[INFO] 通过SSH启动 %s 上的服务\n", host)

	// 执行启动脚本
	scriptPath := "/root/scripts/start-slurmd.sh"
	output, err := s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)

	if err != nil {
		fmt.Fprintf(logWriter, "[ERROR] 执行启动脚本失败: %v\n", err)
		fmt.Fprintf(logWriter, "[ERROR] 输出: %s\n", output)
		return fmt.Errorf("启动服务失败: %v", err)
	}

	fmt.Fprintf(logWriter, "[SUCCESS] 脚本执行输出:\n%s\n", output)
	return nil
}

// checkSlurmServicesViaSSH 通过SSH检查SLURM服务状态
func (s *SlurmService) checkSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string) (string, error) {
	scriptPath := "/root/scripts/check-slurmd.sh"
	return s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)
}

// stopSlurmServicesViaSSH 通过SSH停止SLURM服务
func (s *SlurmService) stopSlurmServicesViaSSH(ctx context.Context, host string, port int, user, password, privateKey string) (string, error) {
	scriptPath := "/root/scripts/stop-slurmd.sh"
	return s.executeScriptViaSSH(ctx, host, port, user, password, privateKey, scriptPath)
}

// detectNodeOSType 检测节点的操作系统类型
func (s *SlurmService) detectNodeOSType(ctx context.Context, nodeName string) string {
	// 尝试检测操作系统类型
	cmd := exec.CommandContext(ctx, "docker", "exec", nodeName, "cat", "/etc/os-release")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[WARN] 检测节点 %s 操作系统失败: %v", nodeName, err)
		return ""
	}

	outputStr := string(output)
	// 检查是否是 Rocky Linux
	if strings.Contains(outputStr, "Rocky") || strings.Contains(outputStr, "rocky") {
		return "rocky"
	}
	// 检查是否是 CentOS
	if strings.Contains(outputStr, "CentOS") || strings.Contains(outputStr, "centos") {
		return "centos"
	}
	// 检查是否是 Ubuntu
	if strings.Contains(outputStr, "Ubuntu") || strings.Contains(outputStr, "ubuntu") {
		return "ubuntu"
	}
	// 检查是否是 Debian
	if strings.Contains(outputStr, "Debian") || strings.Contains(outputStr, "debian") {
		return "debian"
	}

	log.Printf("[WARN] 无法识别节点 %s 的操作系统类型", nodeName)
	return ""
}

// BatchInstallSlurmNodes 批量安装SLURM节点（并发）
func (s *SlurmService) BatchInstallSlurmNodes(ctx context.Context, nodes []InstallSlurmNodeRequest) (map[string]*InstallSlurmNodeResponse, error) {
	results := make(map[string]*InstallSlurmNodeResponse)
	var mu sync.Mutex // 保护 results map
	var wg sync.WaitGroup

	// 使用 channel 来限制并发数，避免过多并发导致系统负载过高
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, node := range nodes {
		wg.Add(1)
		go func(n InstallSlurmNodeRequest) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resp, err := s.InstallSlurmNode(ctx, n)
			if err != nil {
				resp = &InstallSlurmNodeResponse{
					Success: false,
					Message: err.Error(),
				}
			}

			// 写入结果时加锁
			mu.Lock()
			results[n.NodeName] = resp
			mu.Unlock()
		}(node)
	}

	// 等待所有安装任务完成
	wg.Wait()

	return results, nil
}
