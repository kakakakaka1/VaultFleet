package ws

import (
	"sync"

	"vaultfleet/pkg/protocol"
)

type BackupProgressCache struct {
	mu       sync.RWMutex
	progress map[backupProgressKey]protocol.BackupProgressPayload
}

type backupProgressKey struct {
	agentID   string
	messageID string
}

func NewBackupProgressCache() *BackupProgressCache {
	return &BackupProgressCache{
		progress: make(map[backupProgressKey]protocol.BackupProgressPayload),
	}
}

func (c *BackupProgressCache) Set(agentID string, messageID string, payload *protocol.BackupProgressPayload) {
	if c == nil || payload == nil || agentID == "" || messageID == "" {
		return
	}
	copied := *payload

	c.mu.Lock()
	if c.progress == nil {
		c.progress = make(map[backupProgressKey]protocol.BackupProgressPayload)
	}
	c.progress[backupProgressKey{agentID: agentID, messageID: messageID}] = copied
	c.mu.Unlock()
}

func (c *BackupProgressCache) Get(agentID string, messageID string) *protocol.BackupProgressPayload {
	if c == nil || agentID == "" || messageID == "" {
		return nil
	}

	c.mu.RLock()
	payload, ok := c.progress[backupProgressKey{agentID: agentID, messageID: messageID}]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	copied := payload
	return &copied
}

func (c *BackupProgressCache) Delete(agentID string, messageID string) {
	if c == nil || agentID == "" || messageID == "" {
		return
	}

	c.mu.Lock()
	delete(c.progress, backupProgressKey{agentID: agentID, messageID: messageID})
	c.mu.Unlock()
}

func (c *BackupProgressCache) DeleteAgent(agentID string) {
	if c == nil || agentID == "" {
		return
	}

	c.mu.Lock()
	for key := range c.progress {
		if key.agentID == agentID {
			delete(c.progress, key)
		}
	}
	c.mu.Unlock()
}
