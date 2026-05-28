package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vaultfleet/internal/agent/executor"
)

type snapshotCache struct {
	dir string
}

func newSnapshotCache(configDir string) *snapshotCache {
	return &snapshotCache{dir: filepath.Join(configDir, "snapshot-cache")}
}

func (c *snapshotCache) Has(snapshotID string) bool {
	path, err := c.path(snapshotID)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func (c *snapshotCache) Get(snapshotID string) ([]executor.SnapshotFileEntry, bool, error) {
	path, err := c.path(snapshotID)
	if err != nil {
		return nil, false, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var entries []executor.SnapshotFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, true, fmt.Errorf("decode snapshot cache %s: %w", snapshotID, err)
	}
	return entries, true, nil
}

func (c *snapshotCache) Put(snapshotID string, entries []executor.SnapshotFileEntry) error {
	path, err := c.path(snapshotID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}

	file, err := os.CreateTemp(c.dir, ".snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if err := json.NewEncoder(file).Encode(entries); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (c *snapshotCache) Sync(liveSnapshotIDs []string) error {
	liveFiles := make(map[string]struct{}, len(liveSnapshotIDs))
	for _, snapshotID := range liveSnapshotIDs {
		fileName, err := c.fileName(snapshotID)
		if err != nil {
			return err
		}
		liveFiles[fileName] = struct{}{}
	}

	dirEntries, err := os.ReadDir(c.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if _, ok := liveFiles[entry.Name()]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (c *snapshotCache) path(snapshotID string) (string, error) {
	fileName, err := c.fileName(snapshotID)
	if err != nil {
		return "", err
	}
	return filepath.Join(c.dir, fileName), nil
}

func (c *snapshotCache) fileName(snapshotID string) (string, error) {
	if snapshotID == "" {
		return "", errors.New("snapshot id is empty")
	}
	if snapshotID == "." || snapshotID == ".." || strings.Contains(snapshotID, "/") || strings.Contains(snapshotID, `\`) {
		return "", fmt.Errorf("invalid snapshot id %q", snapshotID)
	}
	return snapshotID + ".json", nil
}
