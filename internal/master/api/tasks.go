package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"vaultfleet/internal/master/db"
	"vaultfleet/pkg/protocol"
)

const defaultTaskListLimit = 50
const maxTaskListLimit = 200

type TaskHandler struct {
	DB  *db.Database
	Hub CommandHub
}

type CommandHub interface {
	IsOnline(agentID string) bool
	Send(agentID string, msg interface{}) error
}

type taskResponse struct {
	ID         string     `json:"id"`
	AgentID    string     `json:"agent_id"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	SnapshotID string     `json:"snapshot_id"`
	MessageID  string     `json:"message_id,omitempty"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	DurationMs int64      `json:"duration_ms"`
	RepoSize   int64      `json:"repo_size"`
	ErrorLog   string     `json:"error_log,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func NewTaskHandler(database *db.Database, hub CommandHub) *TaskHandler {
	return &TaskHandler{DB: database, Hub: hub}
}

func RegisterTaskRoutes(rg *gin.RouterGroup, h *TaskHandler) {
	rg.GET("/tasks", h.List)
	rg.POST("/agents/:id/backup-now", h.BackupNow)
}

func (h *TaskHandler) BackupNow(c *gin.Context) {
	agentID := c.Param("id")
	if !agentExistsByID(c, h.DB, agentID) {
		return
	}
	if h.Hub == nil || !h.Hub.IsOnline(agentID) {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "agent offline"})
		return
	}

	msg, err := protocol.NewMessage(protocol.TypeBackupNow, protocol.BackupNowPayload{AgentID: agentID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "encode backup request"})
		return
	}
	if err := h.Hub.Send(agentID, *msg); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": "agent offline"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"ok": true,
		"data": gin.H{
			"message_id": msg.ID,
		},
	})
}

func (h *TaskHandler) List(c *gin.Context) {
	limit := parseTaskLimit(c.Query("limit"))
	query := h.DB.DB.Order("created_at DESC").Limit(limit)
	if agentID := c.Query("agent_id"); agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if taskType := c.Query("type"); taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var histories []db.TaskHistory
	if err := query.Find(&histories).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"ok": true, "data": []taskResponse{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "database error"})
		return
	}

	responses := make([]taskResponse, 0, len(histories))
	for _, history := range histories {
		responses = append(responses, newTaskResponse(history))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": responses})
}

func parseTaskLimit(raw string) int {
	if raw == "" {
		return defaultTaskListLimit
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return defaultTaskListLimit
	}
	if limit > maxTaskListLimit {
		return maxTaskListLimit
	}
	return limit
}

func newTaskResponse(history db.TaskHistory) taskResponse {
	return taskResponse{
		ID:         history.ID,
		AgentID:    history.AgentID,
		Type:       history.Type,
		Status:     history.Status,
		SnapshotID: history.SnapshotID,
		MessageID:  history.MessageID,
		StartedAt:  history.StartedAt,
		FinishedAt: history.FinishedAt,
		DurationMs: history.DurationMs,
		RepoSize:   history.RepoSize,
		ErrorLog:   history.ErrorLog,
		CreatedAt:  history.CreatedAt,
	}
}
