package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/services"
)

// JobController 作业管理控制器
type JobController struct {
	jobService *services.JobService
}

// NewJobController 创建作业控制器
func NewJobController(jobService *services.JobService) *JobController {
	return &JobController{
		jobService: jobService,
	}
}

// ListJobs 获取作业列表
// @Summary 获取作业列表
// @Description 获取用户的作业列表
// @Tags 作业管理
// @Accept json
// @Produce json
// @Param cluster query string false "集群ID"
// @Param status query string false "作业状态"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} models.Response{data=models.JobListResponse}
// @Router /api/jobs [get]
func (jc *JobController) ListJobs(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	cluster := c.Query("cluster")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	jobs, total, err := jc.jobService.ListJobs(c.Request.Context(), userID, cluster, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取作业列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Message: "success",
		Data: models.JobListResponse{
			Jobs:  jobs,
			Total: total,
			Page:  page,
			PageSize: pageSize,
		},
	})
}

// SubmitJob 提交作业
// @Summary 提交作业
// @Description 提交新的作业
// @Tags 作业管理
// @Accept json
// @Produce json
// @Param request body models.SubmitJobRequest true "作业提交请求"
// @Success 200 {object} models.Response{data=models.Job}
// @Router /api/jobs [post]
func (jc *JobController) SubmitJob(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	var req models.SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	req.UserID = userID

	job, err := jc.jobService.SubmitJob(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "提交作业失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    job,
	})
}

// CancelJob 取消作业
// @Summary 取消作业
// @Description 取消指定的作业
// @Tags 作业管理
// @Accept json
// @Produce json
// @Param jobId path int true "作业ID"
// @Param cluster query string true "集群ID"
// @Success 200 {object} models.Response
// @Router /api/jobs/{jobId}/cancel [post]
func (jc *JobController) CancelJob(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的作业ID",
		})
		return
	}

	cluster := c.Query("cluster")
	if cluster == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "集群ID不能为空",
		})
		return
	}

	err = jc.jobService.CancelJob(c.Request.Context(), userID, cluster, uint32(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "取消作业失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
	})
}

// GetJobDetail 获取作业详情
// @Summary 获取作业详情
// @Description 获取指定作业的详细信息
// @Tags 作业管理
// @Accept json
// @Produce json
// @Param jobId path int true "作业ID"
// @Param cluster query string true "集群ID"
// @Success 200 {object} models.Response{data=models.Job}
// @Router /api/jobs/{jobId} [get]
func (jc *JobController) GetJobDetail(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的作业ID",
		})
		return
	}

	cluster := c.Query("cluster")
	if cluster == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "集群ID不能为空",
		})
		return
	}

	job, err := jc.jobService.GetJobDetail(c.Request.Context(), userID, cluster, uint32(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取作业详情失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    job,
	})
}

// GetJobOutput 获取作业输出
// @Summary 获取作业输出
// @Description 获取作业的标准输出和错误输出
// @Tags 作业管理
// @Accept json
// @Produce json
// @Param jobId path int true "作业ID"
// @Param cluster query string true "集群ID"
// @Success 200 {object} models.Response{data=models.JobOutput}
// @Router /api/jobs/{jobId}/output [get]
func (jc *JobController) GetJobOutput(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "无效的作业ID",
		})
		return
	}

	cluster := c.Query("cluster")
	if cluster == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "集群ID不能为空",
		})
		return
	}

	output, err := jc.jobService.GetJobOutput(c.Request.Context(), userID, cluster, uint32(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取作业输出失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    output,
	})
}

// GetDashboardStats 获取仪表板统计信息
// @Summary 获取仪表板统计信息
// @Description 获取用户的作业和集群统计信息
// @Tags 作业管理
// @Accept json
// @Produce json
// @Success 200 {object} models.Response{data=models.JobDashboardStats}
// @Router /api/dashboard/stats [get]
func (jc *JobController) GetDashboardStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.Response{
			Code:    401,
			Message: "用户未认证",
		})
		return
	}

	stats, err := jc.jobService.GetDashboardStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "获取统计信息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "success",
		Data:    stats,
	})
}