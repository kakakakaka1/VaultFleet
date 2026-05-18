package executor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateRcloneConfS3SortedOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rclone.conf")
	config := RcloneConfig{
		Type: "s3",
		Params: map[string]string{
			"secret_access_key": "secret",
			"provider":          "AWS",
			"access_key_id":     "key",
			"region":            "us-east-1",
		},
	}

	if err := WriteRcloneConf(path, config); err != nil {
		t.Fatalf("WriteRcloneConf() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}

	want := "[vaultfleet]\n" +
		"type = s3\n" +
		"access_key_id = key\n" +
		"provider = AWS\n" +
		"region = us-east-1\n" +
		"secret_access_key = secret\n"
	if string(got) != want {
		t.Fatalf("generated config mismatch\nwant:\n%s\ngot:\n%s", want, string(got))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat generated config: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o600 {
		t.Fatalf("config mode = %o, want 600", gotMode)
	}
}

func TestGenerateRcloneConfWebDAVContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rclone.conf")
	config := RcloneConfig{
		Type: "webdav",
		Params: map[string]string{
			"url":    "https://dav.example.test/remote.php/dav/files/user",
			"vendor": "nextcloud",
			"user":   "user@example.test",
			"pass":   "encrypted-pass",
		},
	}

	if err := WriteRcloneConf(path, config); err != nil {
		t.Fatalf("WriteRcloneConf() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}

	for _, want := range []string{
		"[vaultfleet]\n",
		"type = webdav\n",
		"pass = encrypted-pass\n",
		"url = https://dav.example.test/remote.php/dav/files/user\n",
		"user = user@example.test\n",
		"vendor = nextcloud\n",
	} {
		if !containsLine(string(got), want) {
			t.Fatalf("generated config missing %q in:\n%s", want, string(got))
		}
	}
}

func containsLine(content, line string) bool {
	return len(line) == 0 || (len(content) >= len(line) && contains(content, line))
}

func contains(content, substr string) bool {
	for i := 0; i+len(substr) <= len(content); i++ {
		if content[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
