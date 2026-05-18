package executor

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewExecutorBuildsRunnerAndCopiesConfig(t *testing.T) {
	cfg := ExecutorConfig{
		ConfigDir:  "/var/lib/vaultfleet",
		RepoPath:   "repo/agent-1",
		BackupDirs: []string{"/home/alice", "/etc"},
		Excludes:   []string{"*.tmp"},
		Retention: RetentionPolicy{
			KeepLast:  3,
			KeepDaily: 7,
		},
	}

	executor := NewExecutor(cfg)

	if executor.restic == nil {
		t.Fatal("NewExecutor() restic runner is nil")
	}
	runner, ok := executor.restic.(ResticRunner)
	if !ok {
		t.Fatalf("NewExecutor() restic runner type = %T, want ResticRunner", executor.restic)
	}
	if runner.RcloneConfPath != filepath.Join(cfg.ConfigDir, "rclone.conf") {
		t.Fatalf("RcloneConfPath = %q", runner.RcloneConfPath)
	}
	if runner.PasswordFile != filepath.Join(cfg.ConfigDir, ".restic-password") {
		t.Fatalf("PasswordFile = %q", runner.PasswordFile)
	}
	if runner.RepoPath != cfg.RepoPath {
		t.Fatalf("RepoPath = %q, want %q", runner.RepoPath, cfg.RepoPath)
	}

	cfg.BackupDirs[0] = "/mutated"
	cfg.Excludes[0] = "mutated"
	if executor.backupDirs[0] != "/home/alice" {
		t.Fatalf("backup dirs were not copied: %#v", executor.backupDirs)
	}
	if executor.excludes[0] != "*.tmp" {
		t.Fatalf("excludes were not copied: %#v", executor.excludes)
	}
	if executor.retention.KeepLast != 3 || executor.retention.KeepDaily != 7 {
		t.Fatalf("retention = %+v", executor.retention)
	}
}

func TestRunBackupJobSuccessReturnsLatestSnapshotAndSnapshots(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	runner := &recordingRunner{
		backupDelay: 10 * time.Millisecond,
		snapshots: []SnapshotInfo{
			{ID: "old", Time: now.Add(-time.Hour)},
			{ID: "new", Time: now},
		},
	}
	executor := &Executor{
		restic:     runner,
		backupDirs: []string{"/data"},
		excludes:   []string{"*.tmp"},
		retention:  RetentionPolicy{KeepLast: 2},
	}

	result := executor.RunBackupJob(context.Background())

	if result.Type != "backup" {
		t.Fatalf("Type = %q, want backup", result.Type)
	}
	if result.Status != "success" {
		t.Fatalf("Status = %q, want success; error log: %q", result.Status, result.ErrorLog)
	}
	if result.SnapshotID != "new" {
		t.Fatalf("SnapshotID = %q, want new", result.SnapshotID)
	}
	if len(result.Snapshots) != 2 {
		t.Fatalf("Snapshots length = %d, want 2", len(result.Snapshots))
	}
	if result.DurationMs <= 0 {
		t.Fatalf("DurationMs = %d, want positive duration", result.DurationMs)
	}
	assertRunnerCalls(t, runner.calls, []string{"init", "backup", "forget", "snapshots"})
}

func TestRunBackupJobFailureStopsAtStageAndReturnsErrorLog(t *testing.T) {
	runner := &recordingRunner{backupErr: errors.New("disk read failed")}
	executor := &Executor{
		restic:     runner,
		backupDirs: []string{"/data"},
		retention:  RetentionPolicy{KeepLast: 1},
	}

	result := executor.RunBackupJob(context.Background())

	if result.Status != "failed" {
		t.Fatalf("Status = %q, want failed", result.Status)
	}
	if !strings.Contains(result.ErrorLog, "backup: disk read failed") {
		t.Fatalf("ErrorLog = %q, want backup stage and error", result.ErrorLog)
	}
	assertRunnerCalls(t, runner.calls, []string{"init", "backup"})
}

func TestTaskResultStructureAllowsSnapshotMetadata(t *testing.T) {
	result := TaskResult{
		Type:       "backup",
		Status:     "success",
		DurationMs: 123,
		SnapshotID: "abc123",
		RepoSize:   4096,
		Snapshots: []SnapshotInfo{
			{ID: "abc123", Hostname: "agent-1", Paths: []string{"/data"}},
		},
		ErrorLog: "",
	}

	if result.Type != "backup" || result.Status != "success" || result.SnapshotID != "abc123" {
		t.Fatalf("TaskResult basic fields = %+v", result)
	}
	if result.RepoSize != 4096 || len(result.Snapshots) != 1 {
		t.Fatalf("TaskResult metadata fields = %+v", result)
	}
}

type recordingRunner struct {
	calls       []string
	initErr     error
	backupOut   string
	backupErr   error
	backupDelay time.Duration
	forgetErr   error
	snapshots   []SnapshotInfo
	snapshotErr error
	restoreErr  error
}

func (r *recordingRunner) InitRepo(context.Context) error {
	r.calls = append(r.calls, "init")
	return r.initErr
}

func (r *recordingRunner) RunBackup(_ context.Context, dirs []string, excludes []string) (string, error) {
	r.calls = append(r.calls, "backup")
	if r.backupDelay > 0 {
		time.Sleep(r.backupDelay)
	}
	return r.backupOut, r.backupErr
}

func (r *recordingRunner) RunForget(_ context.Context, retention RetentionPolicy) error {
	r.calls = append(r.calls, "forget")
	return r.forgetErr
}

func (r *recordingRunner) ListSnapshots(context.Context) ([]SnapshotInfo, error) {
	r.calls = append(r.calls, "snapshots")
	return r.snapshots, r.snapshotErr
}

func (r *recordingRunner) RestoreSnapshot(context.Context, string, string) error {
	r.calls = append(r.calls, "restore")
	return r.restoreErr
}

func assertRunnerCalls(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("runner calls = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("runner calls = %#v, want %#v", got, want)
		}
	}
}
