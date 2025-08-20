package controllers

import (
    "context"
    "net/http"
    "time"

    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
    "github.com/gin-gonic/gin"
)

type SlurmController struct {
    svc *services.SlurmService
}

func NewSlurmController() *SlurmController {
    return &SlurmController{svc: services.NewSlurmService()}
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
