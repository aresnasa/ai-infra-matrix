package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AnsibleHandler struct {
}

func NewAnsibleHandler() *AnsibleHandler {
	return &AnsibleHandler{}
}

// GetAnsibleStatus 获取Ansible状态
func (h *AnsibleHandler) GetAnsibleStatus(c *gin.Context) {
	logrus.Info("Getting Ansible status")
	
	c.JSON(http.StatusOK, gin.H{
		"status": "operational",
		"message": "Ansible service is running",
	})
}

// ExecutePlaybook 执行Ansible Playbook
func (h *AnsibleHandler) ExecutePlaybook(c *gin.Context) {
	var req struct {
		PlaybookID uint   `json:"playbook_id" binding:"required"`
		Hosts      []string `json:"hosts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"playbook_id": req.PlaybookID,
		"hosts": req.Hosts,
	}).Info("Executing Ansible playbook")

	// TODO: Implement actual playbook execution logic
	c.JSON(http.StatusOK, gin.H{
		"message": "Playbook execution started",
		"playbook_id": req.PlaybookID,
		"status": "running",
	})
}
