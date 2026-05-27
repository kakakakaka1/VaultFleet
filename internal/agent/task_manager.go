package agent

import (
	"context"
	"errors"
	"sync"
)

const (
	taskTypeBackup  = "backup"
	taskTypeRestore = "restore"
	taskTypeQuery   = "query"
)

var (
	errBackupAlreadyRunning  = errors.New("backup already running")
	errRestoreAlreadyRunning = errors.New("restore already running")
)

type runningTask struct {
	messageID string
	taskType  string
	cancel    context.CancelFunc
}

type taskManager struct {
	mu          sync.Mutex
	tasks       map[string]*runningTask
	backupSlot  bool
	restoreSlot bool
}

func newTaskManager() *taskManager {
	return &taskManager{
		tasks: make(map[string]*runningTask),
	}
}

func (tm *taskManager) Start(messageID string, taskType string, fn func(ctx context.Context)) error {
	tm.mu.Lock()
	switch taskType {
	case taskTypeBackup:
		if tm.backupSlot {
			tm.mu.Unlock()
			return errBackupAlreadyRunning
		}
		tm.backupSlot = true
	case taskTypeRestore:
		if tm.restoreSlot {
			tm.mu.Unlock()
			return errRestoreAlreadyRunning
		}
		tm.restoreSlot = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	tm.tasks[messageID] = &runningTask{
		messageID: messageID,
		taskType:  taskType,
		cancel:    cancel,
	}
	tm.mu.Unlock()

	go func() {
		defer func() {
			tm.mu.Lock()
			delete(tm.tasks, messageID)
			switch taskType {
			case taskTypeBackup:
				tm.backupSlot = false
			case taskTypeRestore:
				tm.restoreSlot = false
			}
			tm.mu.Unlock()
		}()

		fn(ctx)
	}()

	return nil
}

func (tm *taskManager) Cancel(messageID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[messageID]
	if !ok {
		return false
	}
	task.cancel()
	return true
}
