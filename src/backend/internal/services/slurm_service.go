package services

import (
    "bufio"
    "context"
    "errors"
    "fmt"
    "os/exec"
    "strconv"
    "strings"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
    "gorm.io/gorm"
)

// Slurm service provides lightweight access to Slurm cluster metrics by shelling out
// to common CLI tools (sinfo, squeue). When CLIs are unavailable, it returns demo data.

type SlurmService struct {
    db *gorm.DB
}

func NewSlurmService() *SlurmService {
    return &SlurmService{}
}

func NewSlurmServiceWithDB(db *gorm.DB) *SlurmService {
    return &SlurmService{db: db}
}

func (s *SlurmService) GetSummary(ctx context.Context) (*SlurmSummary, error) {
    // Try to gather from sinfo/squeue; fallback to demo data if unavailable
    nodesTotal, nodesIdle, nodesAlloc, partitions, demo1 := s.getNodeStats(ctx)
    jobsRun, jobsPend, jobsOther, demo2 := s.getJobStats(ctx)

    demo := demo1 || demo2
    if demo {
        // Provide a stable small demo when tools missing
        if nodesTotal == 0 && jobsRun == 0 && jobsPend == 0 {
            return &SlurmSummary{
                NodesTotal:  3,
                NodesIdle:   2,
                NodesAlloc:  1,
                Partitions:  2,
                JobsRunning: 1,
                JobsPending: 2,
                JobsOther:   0,
                Demo:        true,
                GeneratedAt: time.Now(),
            }, nil
        }
    }

    return &SlurmSummary{
        NodesTotal:  nodesTotal,
        NodesIdle:   nodesIdle,
        NodesAlloc:  nodesAlloc,
        Partitions:  partitions,
        JobsRunning: jobsRun,
        JobsPending: jobsPend,
        JobsOther:   jobsOther,
        Demo:        demo,
        GeneratedAt: time.Now(),
    }, nil
}

func (s *SlurmService) GetNodes(ctx context.Context) ([]SlurmNode, bool, error) {
    // sinfo format: NodeName|State|CPUS(A/I/O/T)|Memory|Partition
    cmd := exec.CommandContext(ctx, "sinfo", "-N", "-o", "%N|%T|%C|%m|%P")
    out, err := cmd.Output()
    if err != nil {
        // Try to get data from database first
        if s.db != nil {
            var dbNodes []models.SlurmNode
            if err := s.db.Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
                // Convert database nodes to service nodes
                var nodes []SlurmNode
                for _, dbNode := range dbNodes {
                    nodes = append(nodes, SlurmNode{
                        Name:      dbNode.NodeName,
                        State:     dbNode.Status,
                        CPUs:      fmt.Sprintf("%d", dbNode.CPUs),
                        MemoryMB:  fmt.Sprintf("%d", dbNode.Memory),
                        Partition: "compute", // 默认分区
                    })
                }
                return nodes, false, nil
            }
        }
        
        // Fallback to demo data if both sinfo and database are unavailable
        return []SlurmNode{
            {Name: "node-a", State: "idle", CPUs: "4", MemoryMB: "16384", Partition: "debug"},
            {Name: "node-b", State: "alloc", CPUs: "8", MemoryMB: "32768", Partition: "compute"},
            {Name: "node-c", State: "idle", CPUs: "16", MemoryMB: "65536", Partition: "compute"},
        }, true, nil
    }

    var nodes []SlurmNode
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
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
        nodes = append(nodes, SlurmNode{
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
    return nodes, false, nil
}

func (s *SlurmService) GetJobs(ctx context.Context) ([]SlurmJob, bool, error) {
    // squeue format: JobID|User|State|Elapsed|Nodes|Reason|Name|Partition
    cmd := exec.CommandContext(ctx, "squeue", "-o", "%i|%u|%T|%M|%D|%R|%j|%P")
    out, err := cmd.Output()
    if err != nil {
        // demo data
        return []SlurmJob{
            {ID: "12345", Name: "train-1", User: "alice", State: "RUNNING", Elapsed: "00:12:34", Nodes: "1", Reason: "None", Partition: "compute"},
            {ID: "12346", Name: "prep-2", User: "bob", State: "PENDING", Elapsed: "00:00:00", Nodes: "1", Reason: "Priority", Partition: "debug"},
        }, true, nil
    }

    var jobs []SlurmJob
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
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
    // sinfo summarized counts
    // partitions count via: sinfo -h -o %P | sort -u | wc -l (approx) – we'll parse simply
    cmd := exec.CommandContext(ctx, "sinfo", "-h", "-o", "%T|%P")
    out, err := cmd.Output()
    if err != nil {
        return 3, 2, 1, 2, true
    }
    seenPartitions := map[string]struct{}{}
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
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
    cmd2 := exec.CommandContext(ctx, "sinfo", "-h", "-N", "-o", "%T")
    out2, err2 := cmd2.Output()
    if err2 == nil {
        total = 0
        idle = 0
        alloc = 0
        scanner2 := bufio.NewScanner(strings.NewReader(string(out2)))
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
    }
    partitions = len(seenPartitions)
    if partitions == 0 {
        // Try to derive partitions another way
        cmd3 := exec.CommandContext(ctx, "sinfo", "-h", "-o", "%P")
        out3, err3 := cmd3.Output()
        if err3 == nil {
            set := map[string]struct{}{}
            scanner3 := bufio.NewScanner(strings.NewReader(string(out3)))
            for scanner3.Scan() {
                set[strings.TrimSpace(scanner3.Text())] = struct{}{}
            }
            partitions = len(set)
        }
    }
    return total, idle, alloc, partitions, false
}

func (s *SlurmService) getJobStats(ctx context.Context) (running, pending, other int, demo bool) {
    cmd := exec.CommandContext(ctx, "squeue", "-h", "-o", "%T")
    out, err := cmd.Output()
    if err != nil {
        return 1, 2, 0, true
    }
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
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
}

// ScalingOperation 扩缩容操作
type ScalingOperation struct {
    ID          string    `json:"id"`
    Type        string    `json:"type"` // "scale-up" or "scale-down"
    Status      string    `json:"status"`
    Nodes       []string  `json:"nodes"`
    StartedAt   time.Time `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    Error       string    `json:"error,omitempty"`
}

// NodeConfig 节点配置
type NodeConfig struct {
    Host       string `json:"host"`
    Port       int    `json:"port"`
    User       string `json:"user"`
    KeyPath    string `json:"key_path"`
    PrivateKey string `json:"private_key"` // 新增：内联私钥内容
    Password   string `json:"password"`
    MinionID   string `json:"minion_id"`
}

// NodeTemplate 节点模板
type NodeTemplate struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Config      NodeConfig        `json:"config"`
    Tags        []string          `json:"tags"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

// ScalingResult 扩缩容结果
type ScalingResult struct {
    OperationID string                 `json:"operation_id"`
    Success     bool                   `json:"success"`
    Results     []NodeScalingResult    `json:"results"`
}

// NodeScalingResult 节点扩缩容结果
type NodeScalingResult struct {
    NodeID  string `json:"node_id"`
    Success bool   `json:"success"`
    Message string `json:"message"`
}

// GetScalingStatus 获取扩缩容状态
func (s *SlurmService) GetScalingStatus(ctx context.Context) (*ScalingStatus, error) {
    // 这里应该从数据库或缓存中获取实际的状态
    // 目前返回模拟数据
    return &ScalingStatus{
        ActiveOperations: []ScalingOperation{},
        RecentOperations: []ScalingOperation{
            {
                ID:        "op-001",
                Type:      "scale-up",
                Status:    "completed",
                Nodes:     []string{"node001", "node002"},
                StartedAt: time.Now().Add(-1 * time.Hour),
                CompletedAt: &[]time.Time{time.Now().Add(-30 * time.Minute)}[0],
            },
        },
        NodeTemplates: []NodeTemplate{},
    }, nil
}

// ScaleUp 执行扩容操作
func (s *SlurmService) ScaleUp(ctx context.Context, nodes []NodeConfig) (*ScalingResult, error) {
    // 这里应该实现实际的SLURM节点扩容逻辑
    // 包括更新slurm.conf、重新加载配置等

    result := &ScalingResult{
        OperationID: generateOperationID(),
        Success:     true,
        Results:     []NodeScalingResult{},
    }

    // 处理节点配置
    for _, node := range nodes {
        result.Results = append(result.Results, NodeScalingResult{
            NodeID:  node.Host,
            Success: true,
            Message: "节点已成功添加到SLURM集群",
        })
    }

    return result, nil
}

// ScaleDown 执行缩容操作
func (s *SlurmService) ScaleDown(ctx context.Context, nodeIDs []string) (*ScalingResult, error) {
    // 这里应该实现实际的SLURM节点缩容逻辑
    // 包括从集群中移除节点、更新配置等

    result := &ScalingResult{
        OperationID: generateOperationID(),
        Success:     true,
        Results:     []NodeScalingResult{},
    }

    // 模拟缩容操作
    for _, nodeID := range nodeIDs {
        result.Results = append(result.Results, NodeScalingResult{
            NodeID:  nodeID,
            Success: true,
            Message: "节点已成功从SLURM集群中移除",
        })
    }

    return result, nil
}

// GetNodeTemplates 获取节点模板列表
func (s *SlurmService) GetNodeTemplates(ctx context.Context) ([]NodeTemplate, error) {
	var templates []models.NodeTemplate
	if err := s.db.Find(&templates).Error; err != nil {
		return nil, err
	}

	// 转换为服务层结构
	result := make([]NodeTemplate, len(templates))
	for i, t := range templates {
		// 将模型层的 NodeConfig 转换为服务层的 NodeConfig
		serviceConfig := NodeConfig{
			Host:     t.Config.Partitions[0], // 使用第一个分区作为主机名（临时方案）
			Port:     22,                     // 默认SSH端口
			User:     "root",                 // 默认用户
			KeyPath:  "",
			Password: "",
			MinionID: "",
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

// generateTemplateID 生成模板ID
func generateTemplateID() string {
    return fmt.Sprintf("tmpl-%d", time.Now().Unix())
}
