package api

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultfleet/internal/master/db"
	"vaultfleet/pkg/protocol"
)

func TestBackupNowSendsAgentCommand(t *testing.T) {
	setup := setupTasksAPI(t)
	agent := createTasksTestAgent(t, setup.database, "online")
	setup.hub.online[agent.ID] = true

	w := postAnyJSON(t, setup.router, "/api/agents/"+agent.ID+"/backup-now", map[string]any{})

	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	body := parseJSON(t, w)
	assert.Equal(t, true, body["ok"])
	data := requireMap(t, body["data"])
	assert.NotEmpty(t, data["message_id"])
	require.Len(t, setup.hub.sent, 1)
	assert.Equal(t, agent.ID, setup.hub.sent[0].agentID)
	assert.Equal(t, protocol.TypeBackupNow, setup.hub.sent[0].message.Type)
	assert.Equal(t, data["message_id"], setup.hub.sent[0].message.ID)
	payload, err := protocol.ParsePayload[protocol.BackupNowPayload](&setup.hub.sent[0].message)
	require.NoError(t, err)
	assert.Equal(t, agent.ID, payload.AgentID)
}

func TestBackupNowRejectsOfflineAgent(t *testing.T) {
	setup := setupTasksAPI(t)
	agent := createTasksTestAgent(t, setup.database, "offline")

	w := postAnyJSON(t, setup.router, "/api/agents/"+agent.ID+"/backup-now", map[string]any{})

	require.Equal(t, http.StatusBadGateway, w.Code)
	body := parseJSON(t, w)
	assert.Equal(t, false, body["ok"])
	assert.Equal(t, "agent offline", body["error"])
}

func TestListTasksFiltersAndLimitsHistory(t *testing.T) {
	setup := setupTasksAPI(t)
	agentA := createTasksTestAgent(t, setup.database, "online")
	agentB := createTasksTestAgent(t, setup.database, "online")
	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	seedTaskHistory(t, setup.database, agentA.ID, "backup", "success", "snap-a-old", now.Add(-2*time.Hour))
	seedTaskHistory(t, setup.database, agentA.ID, "backup", "failed", "snap-a-new", now)
	seedTaskHistory(t, setup.database, agentA.ID, "restore", "success", "snap-restore", now.Add(-time.Hour))
	seedTaskHistory(t, setup.database, agentB.ID, "backup", "success", "snap-b", now.Add(time.Hour))

	w := getJSON(t, setup.router, "/api/tasks?agent_id="+agentA.ID+"&type=backup&limit=1")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := parseJSON(t, w)
	assert.Equal(t, true, body["ok"])
	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)
	task := requireMap(t, data[0])
	assert.Equal(t, agentA.ID, task["agent_id"])
	assert.Equal(t, "backup", task["type"])
	assert.Equal(t, "snap-a-new", task["snapshot_id"])
}

type tasksAPISetup struct {
	database *db.Database
	hub      *fakeCommandHub
	handler  *TaskHandler
	router   *gin.Engine
}

func setupTasksAPI(t *testing.T) tasksAPISetup {
	t.Helper()

	gin.SetMode(gin.TestMode)
	database, err := db.New(t.TempDir())
	require.NoError(t, err)

	hub := &fakeCommandHub{online: map[string]bool{}}
	handler := NewTaskHandler(database, hub)
	router := gin.New()
	RegisterTaskRoutes(router.Group("/api"), handler)

	return tasksAPISetup{database: database, hub: hub, handler: handler, router: router}
}

type sentCommandMessage struct {
	agentID string
	message protocol.Message
}

type fakeCommandHub struct {
	online  map[string]bool
	sendErr error
	sent    []sentCommandMessage
}

func (h *fakeCommandHub) IsOnline(agentID string) bool {
	return h.online[agentID]
}

func (h *fakeCommandHub) Send(agentID string, msg interface{}) error {
	if h.sendErr != nil {
		return h.sendErr
	}
	message, ok := msg.(protocol.Message)
	if !ok {
		return errors.New("message is not protocol.Message")
	}
	h.sent = append(h.sent, sentCommandMessage{agentID: agentID, message: message})
	return nil
}

func createTasksTestAgent(t *testing.T, database *db.Database, status string) db.Agent {
	t.Helper()

	agent := db.Agent{Name: "Task Agent", Status: status}
	require.NoError(t, database.DB.Create(&agent).Error)
	return agent
}

func seedTaskHistory(t *testing.T, database *db.Database, agentID string, taskType string, status string, snapshotID string, createdAt time.Time) {
	t.Helper()

	startedAt := createdAt.Add(-time.Minute)
	finishedAt := createdAt
	history := db.TaskHistory{
		AgentID:    agentID,
		Type:       taskType,
		Status:     status,
		SnapshotID: snapshotID,
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
		DurationMs: 60000,
		CreatedAt:  createdAt,
	}
	require.NoError(t, database.DB.Create(&history).Error)
}
