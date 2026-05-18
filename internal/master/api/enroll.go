package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"vaultfleet/internal/master/db"
)

type enrollAgentRequest struct {
	EnrollToken string `json:"enroll_token" binding:"required"`
	SystemInfo  string `json:"system_info"`
}

func (h *AgentHandler) Enroll(c *gin.Context) {
	var request enrollAgentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request"})
		return
	}

	var agent db.Agent
	var agentToken string
	err := withGeneratedToken("ak_", func(token string) error {
		result := h.DB.DB.Model(&db.Agent{}).
			Where("enroll_token = ? AND agent_token = ?", request.EnrollToken, "").
			Select("agent_token", "enroll_token", "system_info").
			Updates(map[string]any{
				"agent_token":  token,
				"enroll_token": "",
				"system_info":  request.SystemInfo,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		agentToken = token
		return h.DB.DB.First(&agent, "agent_token = ?", token).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			h.writeEnrollmentTokenRejected(c, request.EnrollToken)
		case isTokenGenerationError(err):
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "token generation failed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": gin.H{
			"agent_id":    agent.ID,
			"agent_token": agentToken,
		},
	})
}

func (h *AgentHandler) writeEnrollmentTokenRejected(c *gin.Context, enrollToken string) {
	var agent db.Agent
	err := h.DB.DB.First(&agent, "enroll_token = ?", enrollToken).Error
	if err == nil && agent.AgentToken != "" {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "error": "agent already enrolled"})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid enrollment token"})
}
