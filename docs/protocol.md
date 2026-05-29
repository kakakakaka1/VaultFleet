# VaultFleet Protocol Reference

**语言 / Language:** 中文 | [English](#english)

VaultFleet Master 和 Agent 通过 JSON WebSocket 消息通信。WebSocket 控制面用于心跳、策略下发、目录浏览、快照、恢复、任务取消、诊断日志收集和 Agent 更新。

## 消息格式

```json
{
  "type": "message_type",
  "id": "request_or_event_id",
  "payload": {}
}
```

## 消息类型

| 类型 | 方向 | 用途 |
| --- | --- | --- |
| `heartbeat` | Agent -> Master | 上报在线状态、CPU、内存、磁盘、工具版本和 Agent 能力 |
| `dir_browse_req` | Master -> Agent | 请求浏览目录 |
| `dir_browse_resp` | Agent -> Master | 返回目录列表 |
| `policy_push` | Master -> Agent | 下发完整备份策略 |
| `policy_ack` | Agent -> Master | 确认策略接收结果 |
| `backup_now` | Master -> Agent | 立即执行备份 |
| `task_result` | Agent -> Master | 上报备份、恢复等任务结果 |
| `restore_req` | Master -> Agent | 请求恢复指定快照 |
| `selective_restore_req` | Master -> Agent | 请求恢复快照中的指定路径 |
| `restore_progress` | Agent -> Master | 上报恢复进度 |
| `snapshot_list_req` | Master -> Agent | 请求刷新快照列表 |
| `snapshot_list_resp` | Agent -> Master | 返回快照列表 |
| `snapshot_browse_req` | Master -> Agent | 请求浏览单个快照内容 |
| `snapshot_browse_resp` | Agent -> Master | 返回快照文件树条目 |
| `collect_logs_req` | Master -> Agent | 请求收集 Agent 近期日志 |
| `collect_logs_resp` | Agent -> Master | 返回 Agent 日志或日志收集错误 |
| `dir_size_req` | Master -> Agent | 请求计算目录大小 |
| `dir_size_resp` | Agent -> Master | 返回目录大小 |
| `version_info` | Master -> Agent | 告知 Agent 当前 Master 版本和下载仓库 |
| `update_agent` | Master -> Agent | 请求 Agent 执行版本更新 |
| `backup_progress` | Agent -> Master | 上报运行中备份进度 |
| `cancel_task` | Master -> Agent | 请求取消运行中的任务 |

## 安全说明

- 生产环境应通过 HTTPS/WSS 暴露 Master。
- `policy_push` 包含存储配置、仓库路径和 restic 仓库密码，因此 Master 是信任边界。
- Agent token 是长期凭据，泄露后应在 Web UI 中重新生成。
- 诊断日志可能包含路径、主机名和错误上下文，提交 issue 前应按 `docs/support.md` 和 `SECURITY.md` 脱敏。

## English

VaultFleet Master and Agents communicate with JSON WebSocket messages. The control plane handles heartbeat, policy push, directory browsing, snapshots, restore, task cancellation, diagnostic log collection, and Agent updates.

## Message Envelope

```json
{
  "type": "message_type",
  "id": "request_or_event_id",
  "payload": {}
}
```

## Message Types

| Type | Direction | Purpose |
| --- | --- | --- |
| `heartbeat` | Agent -> Master | Reports online state, CPU, memory, disk, tool versions, and Agent capabilities |
| `dir_browse_req` | Master -> Agent | Requests a directory listing |
| `dir_browse_resp` | Agent -> Master | Returns a directory listing |
| `policy_push` | Master -> Agent | Sends the full backup policy |
| `policy_ack` | Agent -> Master | Acknowledges policy receipt |
| `backup_now` | Master -> Agent | Starts an immediate backup |
| `task_result` | Agent -> Master | Reports backup, restore, or maintenance task results |
| `restore_req` | Master -> Agent | Requests a snapshot restore |
| `selective_restore_req` | Master -> Agent | Requests restore of selected snapshot paths |
| `restore_progress` | Agent -> Master | Reports restore progress |
| `snapshot_list_req` | Master -> Agent | Requests a snapshot refresh |
| `snapshot_list_resp` | Agent -> Master | Returns the snapshot list |
| `snapshot_browse_req` | Master -> Agent | Requests entries from one snapshot |
| `snapshot_browse_resp` | Agent -> Master | Returns snapshot file-tree entries |
| `collect_logs_req` | Master -> Agent | Requests recent Agent logs |
| `collect_logs_resp` | Agent -> Master | Returns Agent logs or a collection error |
| `dir_size_req` | Master -> Agent | Requests directory size calculation |
| `dir_size_resp` | Agent -> Master | Returns directory size |
| `version_info` | Master -> Agent | Sends Master version and download repository information |
| `update_agent` | Master -> Agent | Requests an Agent version update |
| `backup_progress` | Agent -> Master | Reports running backup progress |
| `cancel_task` | Master -> Agent | Requests cancellation of a running task |

## Security Notes

- Expose the Master over HTTPS/WSS in production.
- `policy_push` contains storage configuration, repository path, and the restic repository password, so the Master is part of the trust boundary.
- In protocol terms, policy_push contains sensitive fields and should be treated as privileged control-plane data.
- Agent tokens are long-lived credentials and should be regenerated from the Web UI after exposure.
- Diagnostic logs can contain paths, hostnames, and error context. Redact them according to `docs/support.md` and `SECURITY.md` before posting issues.
