package ws

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultfleet/pkg/protocol"
)

func TestBackupProgressCacheSetGetDeleteCopiesPayload(t *testing.T) {
	cache := NewBackupProgressCache()
	progress := &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "backup",
		PercentDone: 42.5,
		TotalFiles:  100,
		FilesDone:   42,
		TotalBytes:  2048,
		BytesDone:   1024,
		BytesPerSec: 512,
		CurrentFile: "/srv/data.db",
	}

	cache.Set("agent-1", "msg-1", progress)
	progress.Phase = "mutated-after-set"

	got := cache.Get("agent-1", "msg-1")
	require.NotNil(t, got)
	assert.Equal(t, "backup", got.Phase)
	got.Phase = "mutated-after-get"

	gotAgain := cache.Get("agent-1", "msg-1")
	require.NotNil(t, gotAgain)
	assert.Equal(t, "backup", gotAgain.Phase)

	cache.Delete("agent-1", "msg-1")
	assert.Nil(t, cache.Get("agent-1", "msg-1"))
}

func TestBackupProgressCacheConcurrentAccess(t *testing.T) {
	cache := NewBackupProgressCache()
	var wg sync.WaitGroup

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			agentID := "agent-concurrent"
			cache.Set(agentID, "msg-concurrent", &protocol.BackupProgressPayload{
				AgentID:     agentID,
				Phase:       "backup",
				PercentDone: float64(i),
			})
			_ = cache.Get(agentID, "msg-concurrent")
		}(i)
	}

	wg.Wait()
	require.NotNil(t, cache.Get("agent-concurrent", "msg-concurrent"))
}

func TestBackupProgressCacheSetOverwritesExistingMessageProgress(t *testing.T) {
	cache := NewBackupProgressCache()

	cache.Set("agent-1", "msg-1", &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "initial",
		PercentDone: 10,
	})
	cache.Set("agent-1", "msg-1", &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "updated",
		PercentDone: 75,
		BytesDone:   4096,
	})

	progress := cache.Get("agent-1", "msg-1")
	require.NotNil(t, progress)
	assert.Equal(t, "updated", progress.Phase)
	assert.Equal(t, 75.0, progress.PercentDone)
	assert.Equal(t, int64(4096), progress.BytesDone)
}

func TestBackupProgressCacheKeepsMessagesSeparateForSameAgent(t *testing.T) {
	cache := NewBackupProgressCache()

	cache.Set("agent-1", "msg-1", &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "backup",
		PercentDone: 20,
	})
	cache.Set("agent-1", "msg-2", &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "stats",
		PercentDone: 80,
	})

	first := cache.Get("agent-1", "msg-1")
	second := cache.Get("agent-1", "msg-2")
	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.Equal(t, "backup", first.Phase)
	assert.Equal(t, "stats", second.Phase)
}

func TestBackupProgressCacheDeleteAgentClearsAllAgentMessages(t *testing.T) {
	cache := NewBackupProgressCache()
	cache.Set("agent-1", "msg-1", &protocol.BackupProgressPayload{AgentID: "agent-1", Phase: "backup"})
	cache.Set("agent-1", "msg-2", &protocol.BackupProgressPayload{AgentID: "agent-1", Phase: "stats"})
	cache.Set("agent-2", "msg-1", &protocol.BackupProgressPayload{AgentID: "agent-2", Phase: "backup"})

	cache.DeleteAgent("agent-1")

	assert.Nil(t, cache.Get("agent-1", "msg-1"))
	assert.Nil(t, cache.Get("agent-1", "msg-2"))
	assert.NotNil(t, cache.Get("agent-2", "msg-1"))
}

func TestBackupProgressCacheIgnoresEmptyMessageID(t *testing.T) {
	cache := NewBackupProgressCache()

	cache.Set("agent-1", "", &protocol.BackupProgressPayload{
		AgentID: "agent-1",
		Phase:   "backup",
	})

	assert.Nil(t, cache.Get("agent-1", ""))
}

func TestBackupProgressCacheZeroValueIsUsable(t *testing.T) {
	var cache BackupProgressCache

	cache.Set("agent-1", "msg-1", &protocol.BackupProgressPayload{
		AgentID:     "agent-1",
		Phase:       "backup",
		PercentDone: 10,
	})

	progress := cache.Get("agent-1", "msg-1")
	require.NotNil(t, progress)
	assert.Equal(t, "backup", progress.Phase)
}
