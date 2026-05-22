package redact

import (
	"regexp"
	"strings"
)

const Placeholder = "[REDACTED]"

var sensitiveKV = regexp.MustCompile(`(?i)(token|password|passwd|secret|cookie|credential|api_key|access_key|secret_key|private_key|auth)(\s*[=:]\s*)(\S+)`)
var bearerToken = regexp.MustCompile(`(?i)(Bearer\s+)\S+`)

func Text(s string) string {
	s = sensitiveKV.ReplaceAllString(s, "${1}${2}"+Placeholder)
	return bearerToken.ReplaceAllString(s, "${1}"+Placeholder)
}

func JSONFields(m map[string]any, fields ...string) map[string]any {
	redactSet := make(map[string]bool, len(fields))
	for _, field := range fields {
		redactSet[strings.ToLower(field)] = true
	}

	result := make(map[string]any, len(m))
	for key, value := range m {
		if redactSet[strings.ToLower(key)] {
			result[key] = Placeholder
			continue
		}
		result[key] = value
	}
	return result
}
