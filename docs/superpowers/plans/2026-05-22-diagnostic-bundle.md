# 诊断包功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在系统页面新增一键生成诊断包功能，自动收集 Master 日志、Agent 日志、系统状态并打包为 ZIP 下载。

**Architecture:** 新增共享脱敏工具 `pkg/redact`，Master 端内存环形日志缓冲区 `internal/master/logbuf`，通过新 WebSocket 命令 `collect_logs` / `collect_logs_resp` 从 Agent 收集日志，新增 `GET /api/system/diagnostic` 端点生成 ZIP 流，前端在 `/system` 页面新增诊断包卡片。

**Tech Stack:** Go (Gin, GORM, gorilla/websocket, archive/zip), React 18, TypeScript, TanStack Query, shadcn/ui, Sonner toast

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `pkg/redact/redact.go` | 新建 | 共享日志脱敏工具（Master + Agent 共用） |
| `pkg/redact/redact_test.go` | 新建 | 脱敏工具测试 |
| `internal/master/logbuf/logbuf.go` | 新建 | 内存环形日志缓冲区 |
| `internal/master/logbuf/logbuf_test.go` | 新建 | 环形缓冲区测试 |
| `pkg/protocol/message.go` | 修改 | 新增 `TypeCollectLogsReq` / `TypeCollectLogsResp` 常量和 payload 类型 |
| `internal/master/ws/hub.go` | 修改 | `expectedResponseType` 中增加 `collect_logs_req` 的映射 |
| `internal/master/ws/handler.go` | 修改 | `dispatch` 中增加 `collect_logs_resp` 路由 |
| `internal/agent/handler.go` | 修改 | `Handle` 中增加 `collect_logs_req` 分支，实现日志收集 |
| `internal/agent/logcollect.go` | 新建 | Agent 端日志收集：检测 init system，读取日志，截断，脱敏 |
| `internal/agent/logcollect_test.go` | 新建 | Agent 日志收集测试 |
| `internal/master/api/diagnostic.go` | 新建 | 诊断包 API handler：收集数据，生成 ZIP |
| `internal/master/api/diagnostic_test.go` | 新建 | 诊断包 API 测试 |
| `internal/master/api/router.go` | 修改 | 注册诊断包路由 |
| `cmd/master/main.go` | 修改 | 初始化 logbuf，设置 `log.SetOutput` |
| `web/src/services/diagnostic.ts` | 新建 | 前端诊断包 API 调用 |
| `web/src/pages/system/system-page.tsx` | 修改 | 新增诊断包卡片 UI |

---

### Task 1: 共享脱敏工具 `pkg/redact`

**Files:**
- Create: `pkg/redact/redact.go`
- Test: `pkg/redact/redact_test.go`

- [ ] **Step 1: 编写脱敏测试**

```go
// pkg/redact/redact_test.go
package redact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactText_PasswordKeyValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"password=value", `password=secret123`, `password=[REDACTED]`},
		{"PASSWORD=value", `PASSWORD=secret123`, `PASSWORD=[REDACTED]`},
		{"token: value", `token: abc-xyz-123`, `token: [REDACTED]`},
		{"secret_key = value", `secret_key = my-secret`, `secret_key = [REDACTED]`},
		{"api_key=value", `api_key=AKIAIOSFODNN7EXAMPLE`, `api_key=[REDACTED]`},
		{"access_key=value", `access_key=AKIAIOSFODNN7EXAMPLE`, `access_key=[REDACTED]`},
		{"private_key=val", `private_key=pk_live_xxx`, `private_key=[REDACTED]`},
		{"credential=val", `credential=cred_abc`, `credential=[REDACTED]`},
		{"cookie=val", `cookie=sess_abc123`, `cookie=[REDACTED]`},
		{"auth=val", `auth=bearer_token_here`, `auth=[REDACTED]`},
		{"bearer token", `Authorization: Bearer eyJhbGciOiJIUzI1`, `Authorization: Bearer [REDACTED]`},
		{"no match", `this is a normal log line`, `this is a normal log line`},
		{"mixed line", `connecting to server password=abc123 on port 8080`, `connecting to server password=[REDACTED] on port 8080`},
		{"passwd=val", `passwd=hunter2`, `passwd=[REDACTED]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Text(tt.input))
		})
	}
}

func TestRedactText_MultiLine(t *testing.T) {
	input := "line1 normal\npassword=secret\nline3 token: abc\n"
	want := "line1 normal\npassword=[REDACTED]\nline3 token: [REDACTED]\n"
	assert.Equal(t, want, Text(input))
}

func TestRedactJSON_StorageFields(t *testing.T) {
	input := map[string]any{
		"name":       "my-storage",
		"endpoint":   "https://s3.example.com",
		"access_key": "AKIAIOSFODNN7EXAMPLE",
		"secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE",
		"bucket":     "my-bucket",
		"password":   "p@ssw0rd",
		"region":     "us-east-1",
	}
	result := JSONFields(input, "access_key", "secret_key", "password", "endpoint")
	assert.Equal(t, "[REDACTED]", result["access_key"])
	assert.Equal(t, "[REDACTED]", result["secret_key"])
	assert.Equal(t, "[REDACTED]", result["password"])
	assert.Equal(t, "[REDACTED]", result["endpoint"])
	assert.Equal(t, "my-storage", result["name"])
	assert.Equal(t, "my-bucket", result["bucket"])
	assert.Equal(t, "us-east-1", result["region"])
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./pkg/redact/ -v`
Expected: 编译失败，`package redact` 不存在

- [ ] **Step 3: 实现脱敏工具**

```go
// pkg/redact/redact.go
package redact

import (
	"regexp"
	"strings"
)

const Placeholder = "[REDACTED]"

var sensitiveKV = regexp.MustCompile(
	`(?i)(token|password|passwd|secret|cookie|credential|api_key|access_key|secret_key|private_key|auth)(\s*[=:]\s*)(\S+)`)

var bearerToken = regexp.MustCompile(`(?i)(Bearer\s+)\S+`)

func Text(s string) string {
	s = sensitiveKV.ReplaceAllString(s, "${1}${2}"+Placeholder)
	s = bearerToken.ReplaceAllString(s, "${1}"+Placeholder)
	return s
}

func JSONFields(m map[string]any, fields ...string) map[string]any {
	redactSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		redactSet[strings.ToLower(f)] = true
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		if redactSet[strings.ToLower(k)] {
			result[k] = Placeholder
		} else {
			result[k] = v
		}
	}
	return result
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./pkg/redact/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/redact/redact.go pkg/redact/redact_test.go
git commit -m "feat: add shared log redaction utility (pkg/redact)"
```

---

### Task 2: Master 日志环形缓冲区 `internal/master/logbuf`

**Files:**
- Create: `internal/master/logbuf/logbuf.go`
- Test: `internal/master/logbuf/logbuf_test.go`

- [ ] **Step 1: 编写环形缓冲区测试**

```go
// internal/master/logbuf/logbuf_test.go
package logbuf

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_WriteAndRead(t *testing.T) {
	buf := New(64)
	n, err := buf.Write([]byte("hello world\n"))
	require.NoError(t, err)
	assert.Equal(t, 12, n)

	data := buf.Bytes()
	assert.Equal(t, "hello world\n", string(data))
}

func TestRingBuffer_Overflow(t *testing.T) {
	buf := New(16)
	buf.Write([]byte("AAAAAAAABBBBBBBB"))  // fills to capacity (16 bytes)
	buf.Write([]byte("CCCC"))               // overwrites first 4 bytes

	data := string(buf.Bytes())
	assert.Equal(t, 16, len(data))
	// oldest data dropped: should contain tail of ring
	assert.True(t, strings.Contains(data, "CCCC"))
	assert.False(t, strings.HasPrefix(data, "AAAA"))
}

func TestRingBuffer_ExactCapacity(t *testing.T) {
	buf := New(8)
	buf.Write([]byte("12345678"))
	assert.Equal(t, "12345678", string(buf.Bytes()))
}

func TestRingBuffer_Empty(t *testing.T) {
	buf := New(64)
	assert.Equal(t, 0, len(buf.Bytes()))
}

func TestRingBuffer_MultipleSmallWrites(t *testing.T) {
	buf := New(10)
	buf.Write([]byte("aa"))
	buf.Write([]byte("bb"))
	buf.Write([]byte("cc"))
	assert.Equal(t, "aabbcc", string(buf.Bytes()))
}

func TestRingBuffer_ConcurrentSafety(t *testing.T) {
	buf := New(1024)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			buf.Write([]byte("write\n"))
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = buf.Bytes()
	}
	<-done
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./internal/master/logbuf/ -v`
Expected: 编译失败

- [ ] **Step 3: 实现环形缓冲区**

```go
// internal/master/logbuf/logbuf.go
package logbuf

import (
	"io"
	"os"
	"sync"
)

type RingBuffer struct {
	mu       sync.Mutex
	buf      []byte
	pos      int
	full     bool
	capacity int
}

func New(capacity int) *RingBuffer {
	return &RingBuffer{
		buf:      make([]byte, capacity),
		capacity: capacity,
	}
}

func (r *RingBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(p)
	if n >= r.capacity {
		copy(r.buf, p[n-r.capacity:])
		r.pos = 0
		r.full = true
		return n, nil
	}

	spaceToEnd := r.capacity - r.pos
	if n <= spaceToEnd {
		copy(r.buf[r.pos:], p)
	} else {
		copy(r.buf[r.pos:], p[:spaceToEnd])
		copy(r.buf, p[spaceToEnd:])
	}

	if !r.full && r.pos+n >= r.capacity {
		r.full = true
	}
	r.pos = (r.pos + n) % r.capacity
	return n, nil
}

func (r *RingBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.full {
		out := make([]byte, r.pos)
		copy(out, r.buf[:r.pos])
		return out
	}

	out := make([]byte, r.capacity)
	copy(out, r.buf[r.pos:])
	copy(out[r.capacity-r.pos:], r.buf[:r.pos])
	return out
}

func (r *RingBuffer) MultiWriter() io.Writer {
	return io.MultiWriter(os.Stdout, r)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./internal/master/logbuf/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/master/logbuf/logbuf.go internal/master/logbuf/logbuf_test.go
git commit -m "feat: add in-memory ring buffer for Master log capture"
```

---

### Task 3: WebSocket 协议扩展 — `collect_logs` 命令

**Files:**
- Modify: `pkg/protocol/message.go`
- Modify: `internal/master/ws/hub.go:143-152`
- Modify: `internal/master/ws/handler.go:138-186`
- Test: `pkg/protocol/message_test.go` (验证新类型可用)

- [ ] **Step 1: 在 `pkg/protocol/message.go` 中添加新类型和 payload**

在 `message.go` 的常量块（第 12-24 行）末尾添加两个新常量，然后在文件末尾添加 payload 结构体：

```go
// 在常量块中追加（第 23 行之后）：
TypeCollectLogsReq  = "collect_logs_req"
TypeCollectLogsResp = "collect_logs_resp"
```

在文件末尾追加 payload 类型：

```go
// CollectLogsReqPayload requests recent logs from an agent.
type CollectLogsReqPayload struct {
	MaxBytes int `json:"max_bytes"`
}

// CollectLogsRespPayload returns collected log text from an agent.
type CollectLogsRespPayload struct {
	Logs  string `json:"logs"`
	Error string `json:"error,omitempty"`
}
```

- [ ] **Step 2: 在 `internal/master/ws/hub.go` 的 `expectedResponseType` 中添加映射**

在 `hub.go` 第 143-152 行的 `expectedResponseType` switch 中添加新 case：

```go
// 在 case protocol.TypeSnapshotListReq 之后添加：
case protocol.TypeCollectLogsReq:
	return protocol.TypeCollectLogsResp, nil
```

- [ ] **Step 3: 在 `internal/master/ws/handler.go` 的 `dispatch` 中添加路由**

在 `handler.go` 第 178-185 行的 `TypeDirBrowseResp, TypeSnapshotListResp` case 中，追加 `TypeCollectLogsResp`：

将第 178 行：
```go
case protocol.TypeDirBrowseResp, protocol.TypeSnapshotListResp:
```
改为：
```go
case protocol.TypeDirBrowseResp, protocol.TypeSnapshotListResp, protocol.TypeCollectLogsResp:
```

- [ ] **Step 4: 运行现有测试确认无回归**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./pkg/protocol/ ./internal/master/ws/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add pkg/protocol/message.go internal/master/ws/hub.go internal/master/ws/handler.go
git commit -m "feat: add collect_logs WebSocket command type"
```

---

### Task 4: Agent 端日志收集器

**Files:**
- Create: `internal/agent/logcollect.go`
- Create: `internal/agent/logcollect_test.go`
- Modify: `internal/agent/handler.go:100-113`

- [ ] **Step 1: 编写 Agent 日志收集测试**

```go
// internal/agent/logcollect_test.go
package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectLogs_FromFile(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "agent.log")

	now := time.Now()
	lines := now.Add(-2*time.Hour).Format(time.RFC3339) + " early line\n" +
		now.Add(-30*time.Minute).Format(time.RFC3339) + " recent line password=secret123\n" +
		now.Add(-5*time.Minute).Format(time.RFC3339) + " latest line\n"
	require.NoError(t, os.WriteFile(logFile, []byte(lines), 0644))

	result := collectLogsFromFile(logFile, 1024*1024)
	assert.Contains(t, result, "latest line")
	assert.Contains(t, result, "[REDACTED]")
	assert.NotContains(t, result, "secret123")
}

func TestCollectLogs_Truncation(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "agent.log")

	data := make([]byte, 100)
	for i := range data {
		data[i] = 'A'
	}
	data[99] = '\n'
	require.NoError(t, os.WriteFile(logFile, data, 0644))

	result := collectLogsFromFile(logFile, 50)
	assert.LessOrEqual(t, len(result), 50)
}

func TestCollectLogs_MissingFile(t *testing.T) {
	result := collectLogsFromFile("/nonexistent/path/agent.log", 1024)
	assert.Equal(t, "", result)
}

func TestDetectLogSource_Fallback(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "agent.log")
	require.NoError(t, os.WriteFile(logFile, []byte("test\n"), 0644))

	source := detectLogSource(logFile)
	assert.Equal(t, logSourceFile, source)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./internal/agent/ -run TestCollectLogs -v`
Expected: 编译失败

- [ ] **Step 3: 实现 Agent 日志收集器**

```go
// internal/agent/logcollect.go
package agent

import (
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

const defaultLogFile = "/var/log/vaultfleet-agent.log"

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

func collectLogs(maxBytes int) string {
	source := detectLogSource(defaultLogFile)
	switch source {
	case logSourceJournalctl:
		return collectLogsFromJournalctl(maxBytes)
	case logSourceFile:
		return collectLogsFromFile(defaultLogFile, maxBytes)
	default:
		return ""
	}
}

func collectLogsFromJournalctl(maxBytes int) string {
	cmd := exec.Command("journalctl", "-u", "vaultfleet-agent", "--since", "24 hours ago", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("collect journalctl logs failed: %v", err)
		return ""
	}
	text := string(out)
	if len(text) > maxBytes {
		text = text[len(text)-maxBytes:]
	}
	return redact.Text(text)
}

func collectLogsFromFile(path string, maxBytes int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(data)
	if len(text) > maxBytes {
		text = text[len(text)-maxBytes:]
	}
	return redact.Text(text)
}
```

- [ ] **Step 4: 在 `internal/agent/handler.go` 中添加命令处理**

在 `handler.go` 第 100-113 行的 `Handle` 方法 switch 中添加新 case：

```go
// 在 case protocol.TypeSnapshotListReq 之后（第 111 行之后）添加：
case protocol.TypeCollectLogsReq:
	h.handleCollectLogsReq(msg)
```

在 `handler.go` 文件末尾添加处理方法：

```go
func (h *Handler) handleCollectLogsReq(msg protocol.Message) {
	req, err := protocol.ParsePayload[protocol.CollectLogsReqPayload](&msg)
	maxBytes := 5 * 1024 * 1024 // 5MB default
	if err == nil && req.MaxBytes > 0 && req.MaxBytes < maxBytes {
		maxBytes = req.MaxBytes
	}

	logs := collectLogs(maxBytes)
	payload := protocol.CollectLogsRespPayload{Logs: logs}
	resp, err := protocol.NewMessage(protocol.TypeCollectLogsResp, payload)
	if err != nil {
		log.Printf("create collect_logs response failed: %v", err)
		return
	}
	resp.ID = msg.ID
	if err := h.sendMessage(*resp); err != nil {
		log.Printf("send collect_logs response failed: %v", err)
	}
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./internal/agent/ -v`
Expected: 全部 PASS（包括新测试和已有测试）

- [ ] **Step 6: 提交**

```bash
git add internal/agent/logcollect.go internal/agent/logcollect_test.go internal/agent/handler.go
git commit -m "feat: add Agent-side log collection with auto-detection and redaction"
```

---

### Task 5: Master 诊断包 API

**Files:**
- Create: `internal/master/api/diagnostic.go`
- Create: `internal/master/api/diagnostic_test.go`
- Modify: `internal/master/api/router.go`

- [ ] **Step 1: 编写诊断包 API 测试**

```go
// internal/master/api/diagnostic_test.go
package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vaultfleet/internal/master/db"
	"vaultfleet/internal/master/logbuf"
)

type diagnosticTestSetup struct {
	database *db.Database
	router   *gin.Engine
	logBuf   *logbuf.RingBuffer
}

func setupDiagnosticAPI(t *testing.T) diagnosticTestSetup {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := db.New(t.TempDir())
	require.NoError(t, err)

	buf := logbuf.New(1024)
	h := &DiagnosticHandler{
		DB:      database,
		LogBuf:  buf,
		Version: "v0.3.2",
	}

	router := gin.New()
	RegisterDiagnosticRoutes(router.Group("/api/system"), h)
	return diagnosticTestSetup{database: database, router: router, logBuf: buf}
}

func seedDiagnosticAgent(t *testing.T, database *db.Database, name, status string) string {
	t.Helper()
	agent := db.Agent{Name: name, Status: status}
	require.NoError(t, database.DB.Create(&agent).Error)
	return agent.ID
}

func seedFailedTask(t *testing.T, database *db.Database, agentID, errorLog string) {
	t.Helper()
	now := time.Now()
	task := db.TaskHistory{
		AgentID:    agentID,
		Type:       "backup",
		Status:     "failed",
		ErrorLog:   errorLog,
		FinishedAt: &now,
	}
	require.NoError(t, database.DB.Create(&task).Error)
}

func readZipFiles(t *testing.T, body *bytes.Buffer) map[string]string {
	t.Helper()
	zipReader, err := zip.NewReader(bytes.NewReader(body.Bytes()), int64(body.Len()))
	require.NoError(t, err)

	files := make(map[string]string)
	for _, f := range zipReader.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		files[f.Name] = string(data)
	}
	return files
}

func TestDiagnosticHandler_GenerateZip(t *testing.T) {
	setup := setupDiagnosticAPI(t)
	agentID := seedDiagnosticAgent(t, setup.database, "Test-Agent-1", "online")
	seedDiagnosticAgent(t, setup.database, "Test-Agent-2", "offline")
	seedFailedTask(t, setup.database, agentID, "backup failed: connection refused")

	setup.logBuf.Write([]byte("2026-05-22 master log line 1\n"))
	setup.logBuf.Write([]byte("2026-05-22 master log line 2 password=secret\n"))

	req := httptest.NewRequest(http.MethodGet, "/api/system/diagnostic", nil)
	w := httptest.NewRecorder()
	setup.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "vaultfleet-diagnostic-")

	files := readZipFiles(t, w.Body)

	assert.Contains(t, files, "meta.json")
	assert.Contains(t, files, "master/logs.txt")
	assert.Contains(t, files, "master/nodes.json")
	assert.Contains(t, files, "master/storage.json")
	assert.Contains(t, files, "master/policies.json")
	assert.Contains(t, files, "master/recent_errors.json")

	assert.Contains(t, files["master/logs.txt"], "master log line 1")
	assert.NotContains(t, files["master/logs.txt"], "secret")
	assert.Contains(t, files["master/logs.txt"], "[REDACTED]")

	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(files["meta.json"]), &meta))
	assert.Equal(t, "v0.3.2", meta["version"])

	assert.Contains(t, files["master/recent_errors.json"], "connection refused")
}
```

- [ ] **Step 2: 实现诊断包 Handler**

```go
// internal/master/api/diagnostic.go
package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"vaultfleet/internal/master/db"
	"vaultfleet/internal/master/logbuf"
	"vaultfleet/pkg/protocol"
	"vaultfleet/pkg/redact"
)

type DiagnosticHub interface {
	SendAndWait(agentID string, msg protocol.Message, timeout time.Duration) (<-chan protocol.Message, error)
	IsOnline(agentID string) bool
}

type DiagnosticHandler struct {
	DB      *db.Database
	Hub     DiagnosticHub
	LogBuf  *logbuf.RingBuffer
	Version string
}

func NewDiagnosticHandler(database *db.Database, hub DiagnosticHub, logBuf *logbuf.RingBuffer) *DiagnosticHandler {
	return &DiagnosticHandler{DB: database, Hub: hub, LogBuf: logBuf}
}

func RegisterDiagnosticRoutes(rg *gin.RouterGroup, h *DiagnosticHandler) {
	rg.GET("/diagnostic", h.Generate)
}

func (h *DiagnosticHandler) Generate(c *gin.Context) {
	agentIDs := parseAgentIDs(c.Query("agents"))

	timestamp := time.Now().Format("20060102T150405")
	filename := fmt.Sprintf("vaultfleet-diagnostic-%s.zip", timestamp)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/zip")
	c.Status(http.StatusOK)

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	h.writeMeta(zw)
	h.writeMasterLogs(zw)
	h.writeNodes(zw)
	h.writeStorage(zw)
	h.writePolicies(zw)
	h.writeRecentErrors(zw)
	h.collectAgentLogs(zw, agentIDs)
}

func parseAgentIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	var ids []string
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *DiagnosticHandler) writeMeta(zw *zip.Writer) {
	meta := map[string]any{
		"version":      h.Version,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
	}
	writeJSONFile(zw, "meta.json", meta)
}

func (h *DiagnosticHandler) writeMasterLogs(zw *zip.Writer) {
	if h.LogBuf == nil {
		return
	}
	data := h.LogBuf.Bytes()
	writeTextFile(zw, "master/logs.txt", redact.Text(string(data)))
}

func (h *DiagnosticHandler) writeNodes(zw *zip.Writer) {
	var agents []db.Agent
	if err := h.DB.DB.Find(&agents).Error; err != nil {
		log.Printf("diagnostic: query agents failed: %v", err)
		return
	}

	type nodeInfo struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		Status     string     `json:"status"`
		LastSeenAt *time.Time `json:"last_seen_at"`
		SystemInfo string     `json:"system_info,omitempty"`
	}
	nodes := make([]nodeInfo, 0, len(agents))
	for _, a := range agents {
		nodes = append(nodes, nodeInfo{
			ID:         a.ID,
			Name:       a.Name,
			Status:     a.Status,
			LastSeenAt: a.LastSeenAt,
			SystemInfo: a.SystemInfo,
		})
	}
	writeJSONFile(zw, "master/nodes.json", nodes)
}

func (h *DiagnosticHandler) writeStorage(zw *zip.Writer) {
	var configs []db.StorageConfig
	if err := h.DB.DB.Find(&configs).Error; err != nil {
		log.Printf("diagnostic: query storage failed: %v", err)
		return
	}

	type storageInfo struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		RcloneType string `json:"rclone_type"`
	}
	items := make([]storageInfo, 0, len(configs))
	for _, s := range configs {
		items = append(items, storageInfo{
			ID:         s.ID,
			Name:       s.Name,
			RcloneType: s.RcloneType,
		})
	}
	writeJSONFile(zw, "master/storage.json", items)
}

func (h *DiagnosticHandler) writePolicies(zw *zip.Writer) {
	var policies []db.BackupPolicy
	if err := h.DB.DB.Find(&policies).Error; err != nil {
		log.Printf("diagnostic: query policies failed: %v", err)
		return
	}

	type policyInfo struct {
		ID        string `json:"id"`
		AgentID   string `json:"agent_id"`
		StorageID string `json:"storage_id"`
		Schedule  string `json:"schedule"`
		Synced    bool   `json:"synced"`
	}
	items := make([]policyInfo, 0, len(policies))
	for _, p := range policies {
		items = append(items, policyInfo{
			ID:        p.ID,
			AgentID:   p.AgentID,
			StorageID: p.StorageID,
			Schedule:  p.Schedule,
			Synced:    p.Synced,
		})
	}
	writeJSONFile(zw, "master/policies.json", items)
}

func (h *DiagnosticHandler) writeRecentErrors(zw *zip.Writer) {
	var tasks []db.TaskHistory
	if err := h.DB.DB.Where("status = ?", "failed").
		Order("created_at DESC").
		Limit(50).
		Find(&tasks).Error; err != nil {
		log.Printf("diagnostic: query failed tasks failed: %v", err)
		return
	}

	type errorInfo struct {
		ID         string     `json:"id"`
		AgentID    string     `json:"agent_id"`
		Type       string     `json:"type"`
		ErrorLog   string     `json:"error_log"`
		CreatedAt  time.Time  `json:"created_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}
	items := make([]errorInfo, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, errorInfo{
			ID:         t.ID,
			AgentID:    t.AgentID,
			Type:       t.Type,
			ErrorLog:   redact.Text(t.ErrorLog),
			CreatedAt:  t.CreatedAt,
			FinishedAt: t.FinishedAt,
		})
	}
	writeJSONFile(zw, "master/recent_errors.json", items)
}

func (h *DiagnosticHandler) collectAgentLogs(zw *zip.Writer, agentIDs []string) {
	if h.Hub == nil || len(agentIDs) == 0 {
		return
	}

	agentNames := h.loadAgentNames(agentIDs)
	for _, agentID := range agentIDs {
		name := agentNames[agentID]
		if name == "" {
			name = agentID
		}
		dirName := fmt.Sprintf("agents/%s", name)

		if !h.Hub.IsOnline(agentID) {
			writeTextFile(zw, dirName+"/error.txt", "agent offline at collection time")
			continue
		}

		msg, err := protocol.NewMessage(protocol.TypeCollectLogsReq, protocol.CollectLogsReqPayload{
			MaxBytes: 5 * 1024 * 1024,
		})
		if err != nil {
			writeTextFile(zw, dirName+"/error.txt", fmt.Sprintf("create message failed: %v", err))
			continue
		}

		respCh, err := h.Hub.SendAndWait(agentID, *msg, 30*time.Second)
		if err != nil {
			writeTextFile(zw, dirName+"/error.txt", fmt.Sprintf("send failed: %v", err))
			continue
		}

		resp, ok := <-respCh
		if !ok {
			writeTextFile(zw, dirName+"/timeout.txt", "agent did not respond within 30 seconds")
			continue
		}

		payload, err := protocol.ParsePayload[protocol.CollectLogsRespPayload](&resp)
		if err != nil {
			writeTextFile(zw, dirName+"/error.txt", fmt.Sprintf("parse response failed: %v", err))
			continue
		}

		if payload.Error != "" {
			writeTextFile(zw, dirName+"/error.txt", payload.Error)
		}
		if payload.Logs != "" {
			writeTextFile(zw, dirName+"/logs.txt", payload.Logs)
		}
	}
}

func (h *DiagnosticHandler) loadAgentNames(agentIDs []string) map[string]string {
	names := make(map[string]string, len(agentIDs))
	var agents []db.Agent
	if err := h.DB.DB.Where("id IN ?", agentIDs).Find(&agents).Error; err != nil {
		return names
	}
	for _, a := range agents {
		names[a.ID] = a.Name
	}
	return names
}

func writeJSONFile(zw *zip.Writer, name string, data any) {
	w, err := zw.Create(name)
	if err != nil {
		log.Printf("diagnostic: create zip entry %s failed: %v", name, err)
		return
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Printf("diagnostic: write zip entry %s failed: %v", name, err)
	}
}

func writeTextFile(zw *zip.Writer, name string, content string) {
	w, err := zw.Create(name)
	if err != nil {
		log.Printf("diagnostic: create zip entry %s failed: %v", name, err)
		return
	}
	if _, err := w.Write([]byte(content)); err != nil {
		log.Printf("diagnostic: write zip entry %s failed: %v", name, err)
	}
}
```

- [ ] **Step 3: 在 `router.go` 中注册诊断包路由**

在 `internal/master/api/router.go` 的 `NewRouter` 函数中，在 `RegisterSystemRoutes` 调用附近（第 174 行之后），添加以下内容。

首先，`RouterConfig` 需要新增 `LogBuf` 字段。在 `router.go` 文件顶部的 `RouterConfig` 结构体（第 29-36 行）中添加：

```go
// 在 Version string 之后添加：
LogBuf *logbuf.RingBuffer
```

在 import 块中添加：
```go
"vaultfleet/internal/master/logbuf"
```

然后在 `NewRouter` 函数中（第 174 行 `RegisterSystemRoutes` 之后）添加路由注册：

```go
diagnosticHandler := NewDiagnosticHandler(cfg.Database, cfg.Hub, cfg.LogBuf)
diagnosticHandler.Version = cfg.Version
RegisterDiagnosticRoutes(protected.Group("/system"), diagnosticHandler)
```

- [ ] **Step 4: 运行测试**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./internal/master/api/ -run TestDiagnostic -v`
Expected: PASS（如果测试辅助函数需要调整，在此步骤中同步调整）

注意：测试可能需要使用已有的测试辅助函数。查看现有 API 测试文件中的 `setupTestDB` 等函数模式。如果没有现成的，需要创建简化版本。关键是验证 ZIP 生成逻辑正确。如果测试基础设施复杂，可以先跳过集成测试，手动验证后补充。

- [ ] **Step 5: 提交**

```bash
git add internal/master/api/diagnostic.go internal/master/api/diagnostic_test.go internal/master/api/router.go
git commit -m "feat: add diagnostic bundle API endpoint (GET /api/system/diagnostic)"
```

---

### Task 6: Master 启动时初始化日志缓冲区

**Files:**
- Modify: `cmd/master/main.go`

- [ ] **Step 1: 在 `cmd/master/main.go` 中初始化 logbuf**

添加 import：
```go
"vaultfleet/internal/master/logbuf"
```

在 `masterRuntime` 结构体（第 25-32 行）中添加字段：
```go
logBuf *logbuf.RingBuffer
```

在 `main()` 函数中，在 `log.Printf("starting VaultFleet master...")`（第 43 行）**之前**添加：
```go
logRing := logbuf.New(2 * 1024 * 1024) // 2MB
log.SetOutput(logRing.MultiWriter())
```

在 `buildRuntimeWithOptions` 函数中（第 136 行 `api.NewRouter` 调用处），将 `RouterConfig` 更新为传入 `LogBuf`：
```go
// 在 RouterConfig 中添加：
LogBuf: options.logBuf,
```

更新 `runtimeOptions` 结构体添加 `logBuf *logbuf.RingBuffer`。

在 `buildRuntime` 中传入该选项。或者，更简洁的做法：在 `main()` 中创建 `logRing`，然后将其传给 `buildRuntimeWithOptions` 通过 `RouterConfig`。

简化做法——直接在 `main()` 中处理：

在 `main()` 函数中，`runtime := buildRuntime(ctx, database)` 之前创建 logRing，然后修改 `buildRuntime` 签名接受它：

```go
// main() 中：
logRing := logbuf.New(2 * 1024 * 1024)
log.SetOutput(logRing.MultiWriter())

// ... database init ...

runtime := buildRuntime(ctx, database, logRing)
```

更新 `buildRuntime` 和 `buildRuntimeWithOptions`：

```go
func buildRuntime(ctx context.Context, database *db.Database, logRing *logbuf.RingBuffer) masterRuntime {
	return buildRuntimeWithOptions(ctx, database, logRing, runtimeOptions{
		commandTimeoutScanInterval: time.Minute,
	})
}

func buildRuntimeWithOptions(ctx context.Context, database *db.Database, logRing *logbuf.RingBuffer, options runtimeOptions) masterRuntime {
	// ... 现有代码 ...

	// 在 api.NewRouter 的 RouterConfig 中添加：
	// LogBuf: logRing,
}
```

在 `masterRuntime` 中也保存 `logBuf`:
```go
type masterRuntime struct {
	// ... existing fields ...
	logBuf *logbuf.RingBuffer
}
```

在 return 时赋值：`logBuf: logRing`

- [ ] **Step 2: 运行编译确认无误**

Run: `cd /home/nstar/code_temp/VaultFleet && go build ./cmd/master/`
Expected: 编译成功

- [ ] **Step 3: 运行已有的 master 测试确认无回归**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./cmd/master/ -v`
Expected: 全部 PASS（如果有测试调用 `buildRuntime`，需要同步更新签名）

- [ ] **Step 4: 提交**

```bash
git add cmd/master/main.go
git commit -m "feat: initialize log ring buffer at Master startup"
```

---

### Task 7: 前端诊断包服务和 UI

**Files:**
- Create: `web/src/services/diagnostic.ts`
- Modify: `web/src/pages/system/system-page.tsx`

- [ ] **Step 1: 创建前端 API 服务**

```typescript
// web/src/services/diagnostic.ts
export async function downloadDiagnosticBundle(agentIds: string[]): Promise<Blob> {
  const params = agentIds.length > 0 ? `?agents=${agentIds.join(",")}` : "";
  const response = await fetch(`/api/system/diagnostic${params}`, {
    credentials: "same-origin",
  });
  if (!response.ok) {
    throw new Error(`诊断包生成失败: ${response.status}`);
  }
  return response.blob();
}
```

- [ ] **Step 2: 修改系统页面，添加诊断包卡片**

在 `web/src/pages/system/system-page.tsx` 中进行以下修改：

添加 import：

```typescript
// 在现有 import 中添加：
import { Download, Loader2 } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { downloadDiagnosticBundle } from "@/services/diagnostic";
import { listAgents } from "@/services/agents";
import type { Agent } from "@/types/agent";
```

在 `SystemPage` 组件中，添加 agents 查询和诊断包状态（在现有 hooks 之后）：

```typescript
const { data: agents } = useQuery({
  queryKey: ["agents"],
  queryFn: listAgents,
});

const [selectedAgents, setSelectedAgents] = useState<string[]>([]);
const [isGenerating, setIsGenerating] = useState(false);

const toggleAgent = (id: string) => {
  setSelectedAgents((prev) =>
    prev.includes(id) ? prev.filter((a) => a !== id) : [...prev, id]
  );
};

const handleGenerateDiagnostic = async () => {
  setIsGenerating(true);
  try {
    const blob = await downloadDiagnosticBundle(selectedAgents);
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `vaultfleet-diagnostic-${new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19)}.zip`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
    toast.success("诊断包已生成");
  } catch (error: any) {
    toast.error("生成诊断包失败", { description: error.message });
  } finally {
    setIsGenerating(false);
  }
};
```

在 JSX 中，在"问题反馈"卡片 **之前** 添加诊断包卡片：

```tsx
<Card>
  <CardHeader>
    <CardTitle className="text-lg">诊断包</CardTitle>
    <CardDescription>
      自动收集系统信息和日志，用于问题排查。
    </CardDescription>
  </CardHeader>
  <CardContent className="space-y-3">
    {agents && agents.length > 0 && (
      <div className="space-y-2">
        <p className="text-sm text-muted-foreground">
          选择需要收集日志的 Agent（可选）：
        </p>
        {agents.map((agent: Agent) => (
          <label
            key={agent.id}
            className="flex items-center gap-2 text-sm"
          >
            <Checkbox
              checked={selectedAgents.includes(agent.id)}
              onCheckedChange={() => toggleAgent(agent.id)}
              disabled={agent.status !== "online" || isGenerating}
            />
            <span
              className={
                agent.status !== "online"
                  ? "text-muted-foreground"
                  : ""
              }
            >
              {agent.name}
            </span>
            {agent.status !== "online" && (
              <span className="text-xs text-muted-foreground">
                （离线）
              </span>
            )}
          </label>
        ))}
      </div>
    )}
  </CardContent>
  <CardFooter>
    <Button
      variant="outline"
      onClick={handleGenerateDiagnostic}
      disabled={isGenerating}
    >
      {isGenerating ? (
        <>
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          正在生成...
        </>
      ) : (
        <>
          <Download className="mr-2 h-4 w-4" />
          生成诊断包
        </>
      )}
    </Button>
  </CardFooter>
</Card>
```

也需要在文件顶部添加 `useState` import（如果不存在）。

- [ ] **Step 3: 运行前端类型检查**

Run: `cd /home/nstar/code_temp/VaultFleet/web && npx tsc --noEmit`
Expected: 无类型错误

- [ ] **Step 4: 启动开发服务器，在浏览器中测试**

Run: `cd /home/nstar/code_temp/VaultFleet/web && npm run dev`

在浏览器中访问 `/system` 页面，验证：
1. 诊断包卡片显示正常
2. Agent 列表显示正确（在线可勾选，离线灰显）
3. 点击"生成诊断包"能下载 ZIP 文件
4. ZIP 中包含 meta.json、master/logs.txt 等文件
5. 加载状态显示正确

- [ ] **Step 5: 提交**

```bash
git add web/src/services/diagnostic.ts web/src/pages/system/system-page.tsx
git commit -m "feat(ui): add diagnostic bundle card to system page"
```

---

### Task 8: 端到端验证

- [ ] **Step 1: 运行全部后端测试**

Run: `cd /home/nstar/code_temp/VaultFleet && go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 2: 运行前端类型检查和构建**

Run: `cd /home/nstar/code_temp/VaultFleet/web && npx tsc --noEmit && npm run build`
Expected: 无错误

- [ ] **Step 3: 手动验证完整流程**

启动 Master（`go run ./cmd/master/`），在浏览器中：

1. 访问 `/system` 页面
2. 验证诊断包卡片出现在"问题反馈"上方
3. 不勾选任何 Agent，点击"生成诊断包"
4. 确认 ZIP 下载，解压验证包含 `meta.json`、`master/logs.txt`、`master/nodes.json`、`master/storage.json`、`master/policies.json`、`master/recent_errors.json`
5. 如果有在线 Agent，勾选后再次生成，验证 `agents/<name>/logs.txt` 存在

- [ ] **Step 4: 最终提交（如有修复）**

```bash
git add -A
git commit -m "fix: diagnostic bundle integration fixes"
```
