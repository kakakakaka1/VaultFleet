package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"vaultfleet/internal/master/db"
	"vaultfleet/pkg/protocol"
)

type RestoreHandler struct {
	DB  *db.Database
	Hub RestoreHub
}

type RestoreHub interface {
	IsOnline(agentID string) bool
	Send(agentID string, msg interface{}) error
}

type restoreRequest struct {
	SnapshotID string `json:"snapshot_id" binding:"required"`
	TargetPath string `json:"target_path"`
	Target     string `json:"target"`
}

func NewRestoreHandler(database *db.Database, hub RestoreHub) *RestoreHandler {
	return &RestoreHandler{DB: database, Hub: hub}
}

func RegisterRestoreRoutes(rg *gin.RouterGroup, h *RestoreHandler) {
	rg.POST("/agents/:id/restore", h.Restore)
}

func (h *RestoreHandler) Restore(c *gin.Context) {
	agentID := c.Param("id")
	if !agentExistsByID(c, h.DB, agentID) {
		return
	}

	var request restoreRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeErrorResponse(c, http.StatusBadRequest, "invalid request")
		return
	}
	targetPath := request.TargetPath
	if targetPath == "" {
		targetPath = request.Target
	}
	if targetPath == "" {
		writeErrorResponse(c, http.StatusBadRequest, "invalid request")
		return
	}
	if h.Hub == nil || !h.Hub.IsOnline(agentID) {
		writeErrorResponse(c, http.StatusBadGateway, "agent offline")
		return
	}

	msg, err := protocol.NewMessage(protocol.TypeRestoreReq, protocol.RestoreReqPayload{
		SnapshotID: request.SnapshotID,
		Target:     targetPath,
	})
	if err != nil {
		writeErrorResponse(c, http.StatusInternalServerError, "encode restore request")
		return
	}

	startedAt := time.Now()
	history := db.TaskHistory{
		AgentID:    agentID,
		Type:       "restore",
		Status:     "running",
		SnapshotID: request.SnapshotID,
		MessageID:  msg.ID,
		StartedAt:  &startedAt,
	}
	if err := h.DB.DB.Create(&history).Error; err != nil {
		writeErrorResponse(c, http.StatusInternalServerError, "database error")
		return
	}

	if err := h.Hub.Send(agentID, *msg); err != nil {
		finishedAt := time.Now()
		updates := map[string]interface{}{
			"status":      "failed",
			"finished_at": &finishedAt,
			"duration_ms": finishedAt.Sub(startedAt).Milliseconds(),
			"error_log":   err.Error(),
		}
		if updateErr := h.DB.DB.Model(&history).Updates(updates).Error; updateErr != nil {
			writeErrorResponse(c, http.StatusInternalServerError, "database error")
			return
		}
		writeErrorResponse(c, http.StatusBadGateway, "agent offline")
		return
	}

	writeDataResponse(c, http.StatusAccepted, gin.H{
		"message":    "restore started",
		"message_id": msg.ID,
	})
}
