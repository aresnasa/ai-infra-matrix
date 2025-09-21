package controllers

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
    "github.com/gin-gonic/gin"
)

// SlurmController 提供Slurm集群初始化与仓库配置能力，并包含扩缩容与SaltStack集成
type SlurmController struct {
    svc     *services.SlurmService
    sshSvc  *services.SSHService
    saltSvc *services.SaltStackService
}

func NewSlurmController() *SlurmController {
    return &SlurmController{
        svc:     services.NewSlurmServiceWithDB(database.DB),
        sshSvc:  services.NewSSHService(),
        saltSvc: services.NewSaltStackService(),
    }
}

// --- 请求与响应结构 ---

type SSHAuth struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    User     string `json:"user"`
    Password string `json:"password"`
}

type RepoSetupRequest struct {
    RepoHost    SSHAuth `json:"repoHost"`
    BasePath    string  `json:"basePath"`   // e.g. /var/www/html/deb/slurm
    BaseURL     string  `json:"baseURL"`    // e.g. http://repo-host/deb/slurm
    EnableIndex bool    `json:"enableIndex"`
}

type RepoSetupResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    BaseURL string `json:"baseURL"`
}

type InitNode struct {
    SSH  SSHAuth `json:"ssh"`
    Role string  `json:"role"` // controller | node
}

type InitNodesRequest struct {
    RepoURL string     `json:"repoURL"`
    Nodes   []InitNode `json:"nodes"`
}

type HostResult struct {
    Host    string `json:"host"`
    Success bool   `json:"success"`
    Output  string `json:"output"`
    Error   string `json:"error"`
    TookMS  int64  `json:"tookMs"`
}

type InitNodesResponse struct {
    Success bool         `json:"success"`
    Results []HostResult `json:"results"`
}

// GET /api/slurm/summary
func (c *SlurmController) GetSummary(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 4*time.Second)
    defer cancel()
    sum, err := c.svc.GetSummary(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": sum})
}

// GET /api/slurm/nodes
func (c *SlurmController) GetNodes(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()
    nodes, demo, err := c.svc.GetNodes(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": nodes, "demo": demo})
}

// GET /api/slurm/jobs
func (c *SlurmController) GetJobs(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()
    jobs, demo, err := c.svc.GetJobs(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": jobs, "demo": demo})
}

// ScalingRequest 扩缩容请求
type ScalingRequest struct {
    Nodes []services.NodeConfig `json:"nodes" binding:"required"`
}

// ScaleDownRequest 缩容请求
type ScaleDownRequest struct {
    NodeIDs []string `json:"node_ids" binding:"required"`
}

// GET /api/slurm/scaling/status
func (c *SlurmController) GetScalingStatus(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()

    status, err := c.svc.GetScalingStatus(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": status})
}

// POST /api/slurm/scaling/scale-up/async
func (c *SlurmController) ScaleUpAsync(ctx *gin.Context) {
    var req ScalingRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    pm := services.GetProgressManager()
    op := pm.Start("slurm:scale-up", "开始扩容Slurm节点")

    go func(opID string, r ScalingRequest) {
        failed := false
        defer func() {
            pm.Complete(opID, failed, "扩容完成")
        }()

        // 转换节点配置
        connections := make([]services.SSHConnection, len(r.Nodes))
        for i, node := range r.Nodes {
            connections[i] = services.SSHConnection{
                Host:     node.Host,
                Port:     node.Port,
                User:     node.User,
                KeyPath:  node.KeyPath,
                Password: node.Password,
            }
        }

        // SaltStack部署配置
        saltConfig := services.SaltStackDeploymentConfig{
            MasterHost: "salt-master", // 从配置中获取
            MasterPort: 4506,
            AutoAccept: true,
        }

        total := float64(len(connections))
        for i, conn := range connections {
            host := conn.Host
            pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: fmt.Sprintf("deploy-minion-%s", host), Message: "部署SaltStack Minion", Host: host, Progress: float64(i)/total})
        }

        // 并发部署SaltStack Minion
        results, err := c.sshSvc.DeploySaltMinion(context.Background(), connections, saltConfig)
        if err != nil {
            failed = true
            pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "deploy-minions", Message: err.Error()})
            return
        }

        for i, result := range results {
            host := connections[i].Host
            if result.Success {
                pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: fmt.Sprintf("deploy-minion-%s", host), Message: "Minion部署成功", Host: host, Progress: float64(i+1)/total})
            } else {
                failed = true
                pm.Emit(opID, services.ProgressEvent{Type: "error", Step: fmt.Sprintf("deploy-minion-%s", host), Message: result.Error, Host: host, Progress: float64(i+1)/total})
            }
        }

        pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "scale-up-slurm", Message: "执行Slurm扩容"})
        // 执行扩容操作
        scaleResults, err := c.svc.ScaleUp(context.Background(), r.Nodes)
        if err != nil {
            failed = true
            pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "scale-up-slurm", Message: err.Error()})
            return
        }

        pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "scale-up-slurm", Message: "Slurm扩容完成", Data: scaleResults})
    }(op.ID, req)

    ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// POST /api/slurm/scaling/scale-up
func (c *SlurmController) ScaleUp(ctx *gin.Context) {
    var req ScalingRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 300*time.Second)
    defer cancel()

    // 转换节点配置
    connections := make([]services.SSHConnection, len(req.Nodes))
    for i, node := range req.Nodes {
        connections[i] = services.SSHConnection{
            Host:     node.Host,
            Port:     node.Port,
            User:     node.User,
            KeyPath:  node.KeyPath,
            Password: node.Password,
        }
    }

    // SaltStack部署配置
    saltConfig := services.SaltStackDeploymentConfig{
        MasterHost: "salt-master", // 从配置中获取
        MasterPort: 4506,
        AutoAccept: true,
    }

    // 并发部署SaltStack Minion
    results, err := c.sshSvc.DeploySaltMinion(ctxWithTimeout, connections, saltConfig)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 执行扩容操作
    scaleResults, err := c.svc.ScaleUp(ctxWithTimeout, req.Nodes)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{
        "saltstack_deployment": results,
        "scale_results": scaleResults,
    })
}

// POST /api/slurm/scaling/scale-down
func (c *SlurmController) ScaleDown(ctx *gin.Context) {
    var req ScaleDownRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
    defer cancel()

    results, err := c.svc.ScaleDown(ctxWithTimeout, req.NodeIDs)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"data": results})
}

// GET /api/slurm/node-templates
func (c *SlurmController) GetNodeTemplates(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()

    templates, err := c.svc.GetNodeTemplates(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": templates})
}

// POST /api/slurm/node-templates
func (c *SlurmController) CreateNodeTemplate(ctx *gin.Context) {
    var template services.NodeTemplate
    if err := ctx.ShouldBindJSON(&template); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()

    err := c.svc.CreateNodeTemplate(ctxWithTimeout, &template)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusCreated, gin.H{"data": template})
}

// PUT /api/slurm/node-templates/:id
func (c *SlurmController) UpdateNodeTemplate(ctx *gin.Context) {
    id := ctx.Param("id")
    var template services.NodeTemplate
    if err := ctx.ShouldBindJSON(&template); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()

    err := c.svc.UpdateNodeTemplate(ctxWithTimeout, id, &template)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"data": template})
}

// DELETE /api/slurm/node-templates/:id
func (c *SlurmController) DeleteNodeTemplate(ctx *gin.Context) {
    id := ctx.Param("id")

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
    defer cancel()

    err := c.svc.DeleteNodeTemplate(ctxWithTimeout, id)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"message": "模板删除成功"})
}

// GET /api/slurm/saltstack/integration
func (c *SlurmController) GetSaltStackIntegration(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
    defer cancel()

    status, err := c.saltSvc.GetStatus(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"data": status})
}

// POST /api/slurm/saltstack/deploy-minion
func (c *SlurmController) DeploySaltMinion(ctx *gin.Context) {
    var req services.NodeConfig
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 120*time.Second)
    defer cancel()

    connection := services.SSHConnection{
        Host:     req.Host,
        Port:     req.Port,
        User:     req.User,
        KeyPath:  req.KeyPath,
        Password: req.Password,
    }

    saltConfig := services.SaltStackDeploymentConfig{
        MasterHost: "salt-master", // 从配置中获取
        MasterPort: 4506,
        MinionID:   req.MinionID,
        AutoAccept: true,
    }

    results, err := c.sshSvc.DeploySaltMinion(ctxWithTimeout, []services.SSHConnection{connection}, saltConfig)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if len(results) > 0 {
        ctx.JSON(http.StatusOK, gin.H{"data": results[0]})
    } else {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "部署失败"})
    }
}

// POST /api/slurm/saltstack/execute/async
func (c *SlurmController) ExecuteSaltCommandAsync(ctx *gin.Context) {
    var req struct {
        Command string `json:"command" binding:"required"`
        Targets []string `json:"targets"`
    }
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    pm := services.GetProgressManager()
    op := pm.Start("slurm:execute-command", "开始执行SaltStack命令")

    go func(opID string, r struct {
        Command string   `json:"command" binding:"required"`
        Targets []string `json:"targets"`
    }) {
        failed := false
        defer func() {
            pm.Complete(opID, failed, "命令执行完成")
        }()

        pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "execute", Message: "执行命令: " + r.Command})
        result, err := c.saltSvc.ExecuteCommand(context.Background(), r.Command, r.Targets)
        if err != nil {
            failed = true
            pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "execute", Message: err.Error()})
            return
        }

        pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "execute", Message: "命令执行成功", Data: result})
    }(op.ID, req)

    ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// POST /api/slurm/saltstack/execute
func (c *SlurmController) ExecuteSaltCommand(ctx *gin.Context) {
    var req struct {
        Command string `json:"command" binding:"required"`
        Targets []string `json:"targets"`
    }
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
    defer cancel()

    result, err := c.saltSvc.ExecuteCommand(ctxWithTimeout, req.Command, req.Targets)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"data": result})
}

// GET /api/slurm/saltstack/jobs
func (c *SlurmController) GetSaltJobs(ctx *gin.Context) {
    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
    defer cancel()

    jobs, err := c.saltSvc.GetJobs(ctxWithTimeout)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    ctx.JSON(http.StatusOK, gin.H{"data": jobs})
}

// --- 通用进度：SSE 与快照 ---

// GET /api/slurm/progress/:opId
func (c *SlurmController) GetProgress(ctx *gin.Context) {
    opID := ctx.Param("opId")
    pm := services.GetProgressManager()
    snap, ok := pm.Snapshot(opID)
    if !ok {
        ctx.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"data": snap})
}

// GET /api/slurm/tasks - 获取所有任务列表
func (c *SlurmController) GetTasks(ctx *gin.Context) {
    pm := services.GetProgressManager()
    tasks := pm.ListOperations()
    
    // 转换任务数据为前端友好的格式
    taskList := make([]gin.H, 0, len(tasks))
    for _, task := range tasks {
        taskData := gin.H{
            "id":          task.ID,
            "name":        task.Name,
            "status":      string(task.Status),
            "started_at":  task.StartedAt.Unix(),
            "completed_at": task.CompletedAt.Unix(),
            "duration":    0,
        }
        
        if !task.CompletedAt.IsZero() {
            taskData["duration"] = task.CompletedAt.Sub(task.StartedAt).Seconds()
        } else {
            taskData["duration"] = time.Since(task.StartedAt).Seconds()
        }
        
        // 获取最新进度事件
        if len(task.Events) > 0 {
            latestEvent := task.Events[len(task.Events)-1]
            taskData["current_step"] = latestEvent.Step
            taskData["progress"] = latestEvent.Progress
            taskData["last_message"] = latestEvent.Message
        }
        
        taskList = append(taskList, taskData)
    }
    
    ctx.JSON(http.StatusOK, gin.H{"data": taskList})
}

// GET /api/slurm/progress/:opId/stream (SSE)
func (c *SlurmController) StreamProgress(ctx *gin.Context) {
    opID := ctx.Param("opId")
    pm := services.GetProgressManager()
    snap, ok := pm.Snapshot(opID)
    if !ok {
        ctx.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
        return
    }
    // Setup SSE headers
    ctx.Writer.Header().Set("Content-Type", "text/event-stream")
    ctx.Writer.Header().Set("Cache-Control", "no-cache")
    ctx.Writer.Header().Set("Connection", "keep-alive")
    flusher, okf := ctx.Writer.(http.Flusher)
    if !okf {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "stream not supported"})
        return
    }
    // Send existing events first
    for _, ev := range snap.Events {
        b, _ := json.Marshal(ev)
        fmt.Fprintf(ctx.Writer, "data: %s\n\n", string(b))
    }
    flusher.Flush()

    // Subscribe to future events
    ch, ok := pm.Subscribe(opID)
    if !ok {
        return
    }
    defer func() {
        // channel will be closed by manager on completion; nothing else to do
    }()

    notify := ctx.Writer.CloseNotify()
    for {
        select {
        case <-notify:
            return
        case ev, more := <-ch:
            if !more {
                return
            }
            b, _ := json.Marshal(ev)
            fmt.Fprintf(ctx.Writer, "data: %s\n\n", string(b))
            flusher.Flush()
        }
    }
}

// --- 异步：仓库准备 ---

// POST /api/slurm/repo/setup/async
func (c *SlurmController) SetupRepoAsync(ctx *gin.Context) {
    var req RepoSetupRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    pm := services.GetProgressManager()
    op := pm.Start("slurm:setup-repo", "开始准备仓库")

    go func(opID string, r RepoSetupRequest) {
        failed := false
        defer func() {
            pm.Complete(opID, failed, "仓库准备完成")
        }()
        if r.RepoHost.Port == 0 { r.RepoHost.Port = 22 }
        if r.BasePath == "" { r.BasePath = "/var/www/html/deb/slurm" }

        pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "connect", Message: "连接到仓库主机", Host: r.RepoHost.Host})
        // We assume connection happens within next call; no explicit check here.

        pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "install", Message: "安装Nginx与dpkg-dev"})
        if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, `/bin/sh -lc 'if command -v apt-get >/dev/null 2>&1; then export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y nginx dpkg-dev && systemctl enable nginx || true && systemctl restart nginx || true; elif command -v yum >/dev/null 2>&1; then yum install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; elif command -v dnf >/dev/null 2>&1; then dnf install -y nginx createrepo; systemctl enable nginx || true; systemctl restart nginx || true; else echo "Unsupported distro"; exit 1; fi'`); err != nil {
            failed = true
            pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "install", Message: err.Error()})
            return
        }
        pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "install", Message: "组件安装完成"})

        pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "mkdir", Message: "创建仓库目录"})
        if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, "/bin/sh -lc 'mkdir -p "+r.BasePath+"'" ); err != nil {
            failed = true
            pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "mkdir", Message: err.Error()})
            return
        }
        pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "mkdir", Message: "目录已创建"})

        if r.EnableIndex {
            pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: "index", Message: "生成Packages索引（Deb系）"})
            if _, err := c.sshSvc.ExecuteCommand(r.RepoHost.Host, r.RepoHost.Port, r.RepoHost.User, r.RepoHost.Password, "/bin/sh -lc 'if command -v apt-ftparchive >/dev/null 2>&1; then (cd "+r.BasePath+" && apt-ftparchive packages . > Packages && gzip -f Packages); fi'" ); err != nil {
                failed = true
                pm.Emit(opID, services.ProgressEvent{Type: "error", Step: "index", Message: err.Error()})
                return
            }
            pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: "index", Message: "索引生成完成"})
        }
    }(op.ID, req)

    ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// --- 异步：节点初始化 ---

// POST /api/slurm/init-nodes/async
func (c *SlurmController) InitNodesAsync(ctx *gin.Context) {
    var req InitNodesRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if len(req.Nodes) == 0 {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "nodes is empty"})
        return
    }
    pm := services.GetProgressManager()
    op := pm.Start("slurm:init-nodes", "开始初始化节点")

    go func(opID string, r InitNodesRequest) {
        failed := false
        defer func() {
            pm.Complete(opID, failed, "节点初始化完成")
        }()
        total := float64(len(r.Nodes))
        for i, n := range r.Nodes {
            host := n.SSH.Host
            stepBase := fmt.Sprintf("node-%s", host)
            pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: stepBase+":config-repo", Message: "配置APT仓库", Host: host, Progress: float64(i)/total})
            if err := c.sshSvc.ConfigureAptRepo(host, n.SSH.Port, n.SSH.User, n.SSH.Password, r.RepoURL); err != nil {
                failed = true
                pm.Emit(opID, services.ProgressEvent{Type: "error", Step: stepBase+":config-repo", Message: err.Error(), Host: host, Progress: float64(i)/total})
                continue
            }
            pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: stepBase+":config-repo", Message: "APT仓库配置完成", Host: host, Progress: float64(i)/total})

            pm.Emit(opID, services.ProgressEvent{Type: "step-start", Step: stepBase+":install-slurm", Message: "安装Slurm组件", Host: host, Progress: float64(i)/total})
            if out, err := c.sshSvc.InstallSlurm(host, n.SSH.Port, n.SSH.User, n.SSH.Password, n.Role); err != nil {
                failed = true
                pm.Emit(opID, services.ProgressEvent{Type: "error", Step: stepBase+":install-slurm", Message: err.Error(), Host: host, Data: out, Progress: float64(i)/total})
                continue
            } else {
                pm.Emit(opID, services.ProgressEvent{Type: "step-done", Step: stepBase+":install-slurm", Message: "Slurm安装完成", Host: host, Progress: float64(i+1)/total})
            }
        }
    }(op.ID, req)

    ctx.JSON(http.StatusAccepted, gin.H{"opId": op.ID})
}

// SetupRepo 在指定主机上安装nginx + dpkg-dev 并创建简易的 deb 仓库路径
func (c *SlurmController) SetupRepo(ctx *gin.Context) {
    var req RepoSetupRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.RepoHost.Port == 0 {
        req.RepoHost.Port = 22
    }
    if req.BasePath == "" {
        req.BasePath = "/var/www/html/deb/slurm"
    }

    if err := c.sshSvc.SetupSimpleDebRepo(
        req.RepoHost.Host, req.RepoHost.Port, req.RepoHost.User, req.RepoHost.Password,
        req.BasePath, req.EnableIndex,
    ); err != nil {
        ctx.JSON(http.StatusInternalServerError, RepoSetupResponse{Success: false, Message: err.Error(), BaseURL: req.BaseURL})
        return
    }

    ctx.JSON(http.StatusOK, RepoSetupResponse{Success: true, Message: "Repo prepared", BaseURL: req.BaseURL})
}

// InitNodes 将所有节点配置为使用给定repo并安装slurm，依据角色安装 slurmctld/slurmd
func (c *SlurmController) InitNodes(ctx *gin.Context) {
    var req InitNodesRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if len(req.Nodes) == 0 {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "nodes is empty"})
        return
    }

    results := make([]HostResult, 0, len(req.Nodes))
    for _, n := range req.Nodes {
        start := time.Now()
        host := n.SSH.Host
        port := n.SSH.Port
        if port == 0 {
            port = 22
        }
        // 配置APT源
        if err := c.sshSvc.ConfigureAptRepo(host, port, n.SSH.User, n.SSH.Password, req.RepoURL); err != nil {
            results = append(results, HostResult{Host: host, Success: false, Error: err.Error(), TookMS: time.Since(start).Milliseconds()})
            continue
        }
        // 安装Slurm
        out, err := c.sshSvc.InstallSlurm(host, port, n.SSH.User, n.SSH.Password, n.Role)
        if err != nil {
            results = append(results, HostResult{Host: host, Success: false, Output: out, Error: err.Error(), TookMS: time.Since(start).Milliseconds()})
            continue
        }
        results = append(results, HostResult{Host: host, Success: true, Output: out, TookMS: time.Since(start).Milliseconds()})
    }

    ok := true
    for _, r := range results {
        if !r.Success {
            ok = false
            break
        }
    }
    ctx.JSON(http.StatusOK, InitNodesResponse{Success: ok, Results: results})
}
