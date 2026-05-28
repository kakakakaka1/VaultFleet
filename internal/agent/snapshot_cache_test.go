package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultfleet/internal/agent/executor"
)

func TestSnapshotCachePutGetHasAndPermissions(t *testing.T) {
	configDir := t.TempDir()
	cache := newSnapshotCache(configDir)
	entries := []executor.SnapshotFileEntry{
		{Path: "/srv", Type: "dir", Size: 0, Mtime: "2026-05-28T08:00:00Z"},
		{Path: "/srv/app.db", Type: "file", Size: 4096, Mtime: "2026-05-28T08:01:00Z"},
	}

	assert.False(t, cache.Has("snap-1"))

	require.NoError(t, cache.Put("snap-1", entries))

	assert.True(t, cache.Has("snap-1"))
	got, ok, err := cache.Get("snap-1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, entries, got)

	cacheDirInfo, err := os.Stat(filepath.Join(configDir, "snapshot-cache"))
	require.NoError(t, err)
	assert.True(t, cacheDirInfo.IsDir())
	assert.Equal(t, os.FileMode(0o700), cacheDirInfo.Mode().Perm())

	cachePath, err := cache.path("snap-1")
	require.NoError(t, err)
	assertFileMode(t, cachePath, 0o600)
}

func TestSnapshotCacheGetMissingAndCorrupt(t *testing.T) {
	cache := newSnapshotCache(t.TempDir())

	got, ok, err := cache.Get("missing")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)

	require.NoError(t, os.MkdirAll(cache.dir, 0o700))
	cachePath, err := cache.path("snap-1")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cachePath, []byte("{not-json"), 0o600))

	got, ok, err = cache.Get("snap-1")
	require.Error(t, err)
	assert.True(t, ok)
	assert.Nil(t, got)
}

func TestSnapshotCacheSyncRemovesStaleJSONOnly(t *testing.T) {
	cache := newSnapshotCache(t.TempDir())
	require.NoError(t, cache.Put("snap-live", []executor.SnapshotFileEntry{{Path: "/live", Type: "dir"}}))
	require.NoError(t, cache.Put("snap-stale", []executor.SnapshotFileEntry{{Path: "/stale", Type: "dir"}}))
	require.NoError(t, os.WriteFile(filepath.Join(cache.dir, "scratch.tmp"), []byte("tmp"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(cache.dir, "notes.txt"), []byte("notes"), 0o600))

	require.NoError(t, cache.Sync([]string{"snap-live"}))

	assert.True(t, cache.Has("snap-live"))
	assert.False(t, cache.Has("snap-stale"))
	_, err := os.Stat(filepath.Join(cache.dir, "scratch.tmp"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(cache.dir, "notes.txt"))
	require.NoError(t, err)
}

func TestSnapshotCacheRejectsUnsafeSnapshotIDs(t *testing.T) {
	cache := newSnapshotCache(t.TempDir())
	unsafeIDs := []string{"", ".", "..", "../snap", "nested/snap", `nested\snap`}

	for _, snapshotID := range unsafeIDs {
		t.Run(snapshotID, func(t *testing.T) {
			require.Error(t, cache.Put(snapshotID, nil))
			got, ok, err := cache.Get(snapshotID)
			require.Error(t, err)
			assert.False(t, ok)
			assert.Nil(t, got)
		})
	}

	require.Error(t, cache.Sync([]string{"../snap"}))
}
