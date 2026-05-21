package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBuildInitCmdIncludesRepoPasswordAndRcloneConfigEnv(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "backups/agent-1",
	}

	cmd := runner.buildInitCmd()

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"init",
		"-r",
		"rclone:vaultfleet:backups/agent-1",
		"--password-file",
		"/tmp/.restic-password",
	})
	assertEnvContains(t, cmd.Env, "RCLONE_CONFIG=/tmp/rclone.conf")
}

func TestBuildInitCmdProvidesCacheDirWhenServiceEnvironmentOmitsHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	configDir := t.TempDir()
	runner := ResticRunner{
		RcloneConfPath: filepath.Join(configDir, "rclone.conf"),
		PasswordFile:   filepath.Join(configDir, ".restic-password"),
		RepoPath:       "backups/agent-1",
	}

	cmd := runner.buildInitCmd()

	assertEnvContains(t, cmd.Env, "XDG_CACHE_HOME="+filepath.Join(configDir, ".cache"))
}

func TestBuildBackupCmdIncludesExcludesAndDirectories(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	cmd := runner.buildBackupCmd([]string{"/home/alice", "/etc"}, []string{"*.tmp", "/home/alice/cache"})

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"backup",
		"-r",
		"rclone:vaultfleet:repo",
		"--password-file",
		"/tmp/.restic-password",
		"--exclude=*.tmp",
		"--exclude=/home/alice/cache",
		"/home/alice",
		"/etc",
	})
	assertEnvContains(t, cmd.Env, "RCLONE_CONFIG=/tmp/rclone.conf")
}

func TestBuildForgetCmdIncludesPruneAndNonZeroRetention(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	cmd := runner.buildForgetCmd(RetentionPolicy{
		KeepLast:    3,
		KeepDaily:   7,
		KeepMonthly: 12,
	})

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"forget",
		"-r",
		"rclone:vaultfleet:repo",
		"--password-file",
		"/tmp/.restic-password",
		"--prune",
		"--keep-last=3",
		"--keep-daily=7",
		"--keep-monthly=12",
	})
}

func TestBuildSnapshotsCmdRequestsJSON(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	cmd := runner.buildSnapshotsCmd()

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"snapshots",
		"--json",
		"-r",
		"rclone:vaultfleet:repo",
		"--password-file",
		"/tmp/.restic-password",
	})
}

func TestBuildStatsCmdRequestsRawRepositorySizeAsJSON(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	cmd := runner.buildStatsCmd()

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"stats",
		"--mode",
		"raw-data",
		"--json",
		"-r",
		"rclone:vaultfleet:repo",
		"--password-file",
		"/tmp/.restic-password",
	})
}

func TestBuildRestoreCmdIncludesSnapshotAndTarget(t *testing.T) {
	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	cmd := runner.buildRestoreCmd("abc123", "/restore/target")

	assertArgsEqual(t, cmd.Args, []string{
		"restic",
		"restore",
		"abc123",
		"--target",
		"/restore/target",
		"-r",
		"rclone:vaultfleet:repo",
		"--password-file",
		"/tmp/.restic-password",
	})
}

func TestInitRepoIgnoresAlreadyInitializedError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
	}{
		{
			name:   "restic already initialized",
			stderr: "repository already initialized\n",
		},
		{
			name: "rclone config already exists",
			stderr: "Fatal: create repository at rclone:vaultfleet:repo failed: " +
				"Fatal: unable to open repository at rclone:vaultfleet:repo: config file already exists\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFakeRestic(t, dir, fakeResticScript{
				Stdout: "",
				Stderr: tt.stderr,
				Exit:   1,
			})
			prependPath(t, dir)

			runner := ResticRunner{
				RcloneConfPath: "/tmp/rclone.conf",
				PasswordFile:   "/tmp/.restic-password",
				RepoPath:       "repo",
			}

			if err := runner.InitRepo(context.Background()); err != nil {
				t.Fatalf("InitRepo() error = %v, want nil", err)
			}
		})
	}
}

func TestInitRepoSkipsInitWhenSnapshotsCanListExistingRepository(t *testing.T) {
	dir := t.TempDir()
	writeFakeResticRouter(t, dir, map[string]fakeResticScript{
		"cat":       {Stderr: "repository does not exist\n", Exit: 10},
		"snapshots": {Stdout: `[]` + "\n"},
		"init":      {Stderr: "init should not be called\n", Exit: 1},
	})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	if err := runner.InitRepo(context.Background()); err != nil {
		t.Fatalf("InitRepo() error = %v, want nil", err)
	}
}

func TestRunBackupReturnsStdoutAndIncludesStderrOnFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeRestic(t, dir, fakeResticScript{Stdout: "snapshot abc123 saved\n"})
		prependPath(t, dir)

		runner := ResticRunner{
			RcloneConfPath: "/tmp/rclone.conf",
			PasswordFile:   "/tmp/.restic-password",
			RepoPath:       "repo",
		}

		got, err := runner.RunBackup(context.Background(), []string{"/data"}, nil)
		if err != nil {
			t.Fatalf("RunBackup() error = %v", err)
		}
		if got != "snapshot abc123 saved\n" {
			t.Fatalf("RunBackup() stdout = %q", got)
		}
	})

	t.Run("failure", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeRestic(t, dir, fakeResticScript{Stderr: "backup failed for /data\n", Exit: 2})
		prependPath(t, dir)

		runner := ResticRunner{
			RcloneConfPath: "/tmp/rclone.conf",
			PasswordFile:   "/tmp/.restic-password",
			RepoPath:       "repo",
		}

		_, err := runner.RunBackup(context.Background(), []string{"/data"}, nil)
		if err == nil {
			t.Fatal("RunBackup() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "backup failed for /data") {
			t.Fatalf("RunBackup() error = %q, want stderr included", err.Error())
		}
	})
}

func TestListSnapshotsParsesResticJSON(t *testing.T) {
	dir := t.TempDir()
	writeFakeRestic(t, dir, fakeResticScript{
		Stdout: `[{"id":"abc123","time":"2026-05-18T12:34:56Z","paths":["/data"],"hostname":"agent-1","size":4096}]` + "\n",
	})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	got, err := runner.ListSnapshots(context.Background())
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListSnapshots() returned %d snapshots, want 1", len(got))
	}
	wantTime := time.Date(2026, 5, 18, 12, 34, 56, 0, time.UTC)
	if got[0].ID != "abc123" || got[0].Hostname != "agent-1" || got[0].Size != 4096 || !got[0].Time.Equal(wantTime) {
		t.Fatalf("ListSnapshots()[0] = %+v", got[0])
	}
	if len(got[0].Paths) != 1 || got[0].Paths[0] != "/data" {
		t.Fatalf("ListSnapshots()[0].Paths = %#v", got[0].Paths)
	}
}

func TestRepositorySizeParsesResticStatsJSON(t *testing.T) {
	dir := t.TempDir()
	writeFakeRestic(t, dir, fakeResticScript{
		Stdout: `{"total_size":987654,"total_file_count":12}` + "\n",
	})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	got, err := runner.RepositorySize(context.Background())
	if err != nil {
		t.Fatalf("RepositorySize() error = %v", err)
	}
	if got != 987654 {
		t.Fatalf("RepositorySize() = %d, want 987654", got)
	}
}

func TestRepositorySizeReturnsStderrOnFailure(t *testing.T) {
	dir := t.TempDir()
	writeFakeRestic(t, dir, fakeResticScript{Stderr: "stats failed\n", Exit: 1})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	_, err := runner.RepositorySize(context.Background())
	if err == nil {
		t.Fatal("RepositorySize() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "stats failed") {
		t.Fatalf("RepositorySize() error = %q, want stderr included", err.Error())
	}
}

func TestRestoreSnapshotReturnsStderrOnFailure(t *testing.T) {
	dir := t.TempDir()
	writeFakeRestic(t, dir, fakeResticScript{Stderr: "restore failed\n", Exit: 1})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}

	err := runner.RestoreSnapshot(context.Background(), "abc123", "/restore")
	if err == nil {
		t.Fatal("RestoreSnapshot() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "restore failed") {
		t.Fatalf("RestoreSnapshot() error = %q, want stderr included", err.Error())
	}
}

func TestRunForgetHonorsContextCancellation(t *testing.T) {
	dir := t.TempDir()
	writeFakeRestic(t, dir, fakeResticScript{SleepSeconds: 2})
	prependPath(t, dir)

	runner := ResticRunner{
		RcloneConfPath: "/tmp/rclone.conf",
		PasswordFile:   "/tmp/.restic-password",
		RepoPath:       "repo",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runner.RunForget(ctx, RetentionPolicy{KeepLast: 1})
	if err == nil {
		t.Fatal("RunForget() error = nil, want context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunForget() error = %v, want context.Canceled", err)
	}
}

func assertArgsEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args length = %d, want %d\nargs: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q\nargs: %#v", i, got[i], want[i], got)
		}
	}
}

func assertEnvContains(t *testing.T, env []string, want string) {
	t.Helper()
	for _, entry := range env {
		if entry == want {
			return
		}
	}
	t.Fatalf("env missing %q in %#v", want, env)
}

type fakeResticScript struct {
	Stdout       string
	Stderr       string
	Exit         int
	SleepSeconds int
}

func writeFakeRestic(t *testing.T, dir string, script fakeResticScript) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake restic shell script is not supported on windows")
	}

	path := filepath.Join(dir, "restic")
	content := "#!/bin/sh\n"
	if script.SleepSeconds > 0 {
		content += "sleep " + strconv.Itoa(script.SleepSeconds) + "\n"
	}
	if script.Stdout != "" {
		content += "printf '%s' " + shellQuote(script.Stdout) + "\n"
	}
	if script.Stderr != "" {
		content += "printf '%s' " + shellQuote(script.Stderr) + " >&2\n"
	}
	content += "exit " + strconv.Itoa(script.Exit) + "\n"

	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write fake restic: %v", err)
	}
}

func writeFakeResticRouter(t *testing.T, dir string, scripts map[string]fakeResticScript) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake restic shell script is not supported on windows")
	}

	path := filepath.Join(dir, "restic")
	content := "#!/bin/sh\n"
	content += "case \"$1\" in\n"
	keys := make([]string, 0, len(scripts))
	for key := range scripts {
		keys = append(keys, key)
	}
	for _, key := range keys {
		script := scripts[key]
		content += key + ")\n"
		if script.SleepSeconds > 0 {
			content += "sleep " + strconv.Itoa(script.SleepSeconds) + "\n"
		}
		if script.Stdout != "" {
			content += "printf '%s' " + shellQuote(script.Stdout) + "\n"
		}
		if script.Stderr != "" {
			content += "printf '%s' " + shellQuote(script.Stderr) + " >&2\n"
		}
		content += "exit " + strconv.Itoa(script.Exit) + "\n"
		content += ";;\n"
	}
	content += "*) echo unexpected command \"$1\" >&2; exit 99;;\n"
	content += "esac\n"

	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write fake restic router: %v", err)
	}
}

func prependPath(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
