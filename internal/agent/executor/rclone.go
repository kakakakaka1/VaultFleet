package executor

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type RcloneConfig struct {
	Type   string
	Params map[string]string
}

func WriteRcloneConf(path string, config RcloneConfig) error {
	var builder strings.Builder
	builder.WriteString("[vaultfleet]\n")
	builder.WriteString(fmt.Sprintf("type = %s\n", config.Type))

	keys := make([]string, 0, len(config.Params))
	for key := range config.Params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s = %s\n", key, config.Params[key]))
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if _, err := file.WriteString(builder.String()); err != nil {
		return err
	}
	return nil
}
