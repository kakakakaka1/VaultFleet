package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTaskManagerStartAndComplete(t *testing.T) {
	tm := newTaskManager()
	done := make(chan struct{})

	err := tm.Start("msg-1", taskTypeBackup, func(ctx context.Context) {
		close(done)
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("task did not run")
	}

	waitForTaskManagerState(t, tm, func() bool {
		return !tm.backupSlot && len(tm.tasks) == 0
	})
}

func TestTaskManagerBackupSlotMutex(t *testing.T) {
	tm := newTaskManager()
	started := make(chan struct{})
	release := make(chan struct{})

	err := tm.Start("msg-1", taskTypeBackup, func(ctx context.Context) {
		close(started)
		<-release
	})
	if err != nil {
		t.Fatalf("first Start error: %v", err)
	}
	<-started

	err = tm.Start("msg-2", taskTypeBackup, func(ctx context.Context) {})
	if err != errBackupAlreadyRunning {
		t.Fatalf("expected errBackupAlreadyRunning, got: %v", err)
	}

	close(release)
	waitForTaskManagerState(t, tm, func() bool {
		return !tm.backupSlot && len(tm.tasks) == 0
	})

	err = tm.Start("msg-3", taskTypeBackup, func(ctx context.Context) {})
	if err != nil {
		t.Fatalf("Start after release error: %v", err)
	}
	waitForTaskManagerState(t, tm, func() bool {
		return !tm.backupSlot && len(tm.tasks) == 0
	})
}

func TestTaskManagerRestoreSlotMutex(t *testing.T) {
	tm := newTaskManager()
	started := make(chan struct{})
	release := make(chan struct{})

	err := tm.Start("msg-1", taskTypeRestore, func(ctx context.Context) {
		close(started)
		<-release
	})
	if err != nil {
		t.Fatalf("first Start error: %v", err)
	}
	<-started

	err = tm.Start("msg-2", taskTypeRestore, func(ctx context.Context) {})
	if err != errRestoreAlreadyRunning {
		t.Fatalf("expected errRestoreAlreadyRunning, got: %v", err)
	}

	close(release)
	waitForTaskManagerState(t, tm, func() bool {
		return !tm.restoreSlot && len(tm.tasks) == 0
	})
}

func TestTaskManagerQueryNoSlotLimit(t *testing.T) {
	tm := newTaskManager()
	var wg sync.WaitGroup
	count := 5
	started := make(chan struct{}, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		msgID := "msg-" + string(rune('a'+i))
		err := tm.Start(msgID, taskTypeQuery, func(ctx context.Context) {
			started <- struct{}{}
			wg.Done()
		})
		if err != nil {
			t.Fatalf("Start query %d error: %v", i, err)
		}
	}
	wg.Wait()
	if len(started) != count {
		t.Fatalf("expected %d queries started, got %d", count, len(started))
	}
	waitForTaskManagerState(t, tm, func() bool {
		return len(tm.tasks) == 0
	})
}

func TestTaskManagerCancelRunningTask(t *testing.T) {
	tm := newTaskManager()
	started := make(chan struct{})
	cancelled := make(chan struct{})

	err := tm.Start("msg-1", taskTypeBackup, func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(cancelled)
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	<-started

	found := tm.Cancel("msg-1")
	if !found {
		t.Fatal("Cancel returned false, expected true")
	}

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("task was not cancelled")
	}
}

func TestTaskManagerCancelUnknownReturnsFalse(t *testing.T) {
	tm := newTaskManager()
	if tm.Cancel("nonexistent") {
		t.Fatal("Cancel returned true for nonexistent task")
	}
}

func TestTaskManagerBackupAndRestoreIndependent(t *testing.T) {
	tm := newTaskManager()
	backupStarted := make(chan struct{})
	restoreStarted := make(chan struct{})
	release := make(chan struct{})

	err := tm.Start("msg-b", taskTypeBackup, func(ctx context.Context) {
		close(backupStarted)
		<-release
	})
	if err != nil {
		t.Fatalf("backup Start error: %v", err)
	}
	<-backupStarted

	err = tm.Start("msg-r", taskTypeRestore, func(ctx context.Context) {
		close(restoreStarted)
		<-release
	})
	if err != nil {
		t.Fatalf("restore Start error: %v (should not conflict with backup)", err)
	}
	<-restoreStarted

	close(release)
	waitForTaskManagerState(t, tm, func() bool {
		return !tm.backupSlot && !tm.restoreSlot && len(tm.tasks) == 0
	})
}

func waitForTaskManagerState(t *testing.T, tm *taskManager, ready func() bool) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		tm.mu.Lock()
		ok := ready()
		tm.mu.Unlock()
		if ok {
			return
		}

		select {
		case <-deadline:
			t.Fatal("task manager state did not settle")
		case <-ticker.C:
		}
	}
}
