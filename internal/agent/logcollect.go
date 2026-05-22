package agent

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"vaultfleet/pkg/redact"
)

type logSource int

const (
	logSourceJournalctl logSource = iota
	logSourceFile
	logSourceNone
)

func detectLogSource(fallbackLogFile string) logSource {
	if _, err := exec.LookPath("journalctl"); err == nil {
		out, err := exec.Command("systemctl", "is-active", "vaultfleet-agent").Output()
		if err == nil && strings.TrimSpace(string(out)) == "active" {
			return logSourceJournalctl
		}
	}
	if _, err := os.Stat(fallbackLogFile); err == nil {
		return logSourceFile
	}
	return logSourceNone
}

const defaultLogFile = "/var/log/vaultfleet-agent.log"

func collectLogs(logFile string, maxBytes int) (string, error) {
	source := detectLogSource(logFile)
	switch source {
	case logSourceJournalctl:
		return collectLogsFromJournalctl(maxBytes)
	case logSourceFile:
		return collectLogsFromFile(logFile, maxBytes)
	default:
		return "", fmt.Errorf("no agent log source found")
	}
}

func collectLogsFromJournalctl(maxBytes int) (string, error) {
	cmd := exec.Command("journalctl", "-u", "vaultfleet-agent", "--since", "24 hours ago", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("collect journalctl logs failed: %v", err)
		return "", fmt.Errorf("collect journalctl logs: %w", err)
	}
	return redactAndLimit(string(out), maxBytes), nil
}

func collectLogsFromFile(path string, maxBytes int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read log file %s: %w", path, err)
	}
	return redactAndLimit(string(data), maxBytes), nil
}

func redactAndLimit(text string, maxBytes int) string {
	text = redact.Text(text)
	if maxBytes > 0 && len(text) > maxBytes {
		text = text[len(text)-maxBytes:]
	}
	return text
}
