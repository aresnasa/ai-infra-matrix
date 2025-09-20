package controllers

import (
    "context"
    "net/http"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
    "github.com/gin-gonic/gin"
)

type SlurmController struct {
    svc      *services.SlurmService
    sshSvc   *services.SSHService
    saltSvc  *services.SaltStackService
}

func NewSlurmController() *SlurmController {
    return &SlurmController{
        svc:     services.NewSlurmService(),
        sshSvc:  services.NewSSHService(),
        saltSvc: services.NewSaltStackService(),
    }
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

// POST /api/slurm/scaling/scale-up
func (c *SlurmController) ScaleUp(ctx *gin.Context) {
    var req ScalingRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 300*time.Second) // 5分钟超时
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
        "data": gin.H{
            "deployment_results": results,
            "scaling_results":    scaleResults,
        },
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
