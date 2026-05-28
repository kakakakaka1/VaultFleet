# VaultFleet

<!-- markdownlint-disable MD013 -->

> 像探针一样部署 Agent，像备份平台一样统一编排多台服务器的备份、快照和恢复。

**语言 / Language:** 中文 | [English](#english)

VaultFleet 是一个面向多台 VPS / Linux 服务器的集中式备份管理系统。它采用 **Master + Agent** 架构：Master 提供 Web UI、API、策略管理和任务状态汇总；Agent 安装在每台服务器上，主动连接 Master，接收备份策略，并使用 `restic` + `rclone` 将数据直接写入对象存储、WebDAV、SFTP、网盘等后端。

这个项目的部署模型参考了 Komari、哪吒监控这类探针项目的体验：节点无需开放入站端口，只需一条安装命令完成注册和上线。不同的是，VaultFleet 的核心不是监控面板，而是可恢复、可审计、可集中管理的备份系统。

## 特性

- **Master + Agent 架构**：Agent 主动连接 Master，服务器侧不需要暴露额外端口。
- **集中策略管理**：在 Master 中维护每台节点的备份目录、排除规则、调度周期、保留策略、任务超时和存储配置。
- **现代 Web 控制台**：提供仪表盘、节点管理、存储配置、备份策略、任务历史、快照浏览、通知设置和系统管理页面。
- **一次性注册令牌**：安装时使用 `enrollment token`，注册后换发长期 `agent_token`，减少安装命令泄露后的重放风险。
- **WebSocket 控制面**：心跳、目录浏览、目录大小计算、策略下发、立即备份、任务取消、快照刷新、快照内容浏览、选择性恢复和 Agent 版本信息都通过统一协议传递。
- **备份数据不经过 Master**：Agent 直接通过 `rclone` 将 `restic` 仓库写入存储端，Master 只保存配置、任务和快照元数据。
- **端到端加密备份**：每台 Agent 使用独立的 restic 仓库密码，Master 侧敏感配置使用 AES-256-GCM 加密存储。
- **任务进度与取消**：运行中的备份任务会上报阶段、文件数、字节数和传输速度；长任务支持取消和策略级超时配置。
- **快照浏览与选择性恢复**：支持浏览快照内容、选择部分路径恢复，并通过 Agent 本地缓存降低大快照反复浏览成本。
- **rclone 高级传输参数**：策略可配置 `transfers`、`tpslimit`、`retries`、`timeout` 等参数，WebDAV 场景内置更保守的默认值。
- **Agent 版本感知与自更新**：Master 可在心跳中识别 Agent 版本差异，Agent 支持通过 GitHub Release 下载新版二进制并重启。
- **通知与诊断**：支持 Telegram、Webhook、健康检查、系统诊断和日志收集，便于备份失败、节点离线和安装问题排障。
- **轻量部署**：Master 可用 Docker 运行；Agent 是单二进制，安装脚本会自动准备 `restic` 和 `rclone`。

## 界面预览

**仪表盘** — 系统状态、节点在线数、策略同步、24h 任务统计和最近任务一览。

![仪表盘](docs/screenshots/dashboard.png)

**节点管理** — 查看所有 Agent 节点的在线状态、系统信息，一键生成安装命令。

![节点管理](docs/screenshots/nodes.png)

**节点详情** — 单个节点的系统信息、连接状态、策略、快照、任务历史、命令队列和远程文件浏览器。

![节点详情](docs/screenshots/node-detail.png)

**备份策略** — 选择节点和存储，配置备份目录、Cron 调度、保留策略、任务超时和 rclone 高级传输参数。

![备份策略](docs/screenshots/policy-create.png)

**存储配置** — 支持 S3 / R2 / MinIO、WebDAV、SFTP、本地路径等多种 rclone 后端，内置连接测试。

![存储配置](docs/screenshots/storage-create.png)

## 架构

```text
┌──────────────────────────────────────────────┐
│                   Master                      │
│  Web UI / API / SQLite / Policy / Notify      │
└──────────────────────┬───────────────────────┘
                       │ WebSocket 控制面
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │ Agent A │    │ Agent B │    │ Agent C │
   └────┬────┘    └────┬────┘    └────┬────┘
        │              │              │
        └──────────────┼──────────────┘
                       ▼
        S3 / R2 / MinIO / WebDAV / SFTP / rclone 后端
```

关键设计原则：

- Master 负责控制面，不接收、不中转备份数据。
- Agent 保存本地策略副本，Master 短暂离线不影响已下发的定时备份。
- 每台服务器拥有独立仓库路径和独立 restic 密码。
- Agent 只主动向外连接 Master 和存储端，不要求公网入站访问。

## 组件

| 组件 | 说明 |
| --- | --- |
| Master | Web UI、认证、Agent 管理、存储配置、备份策略、任务历史、快照、通知、导出恢复 |
| Agent | 注册、心跳、目录浏览、策略接收、本地调度、restic/rclone 执行、结果上报 |
| Storage | 由 rclone 适配的远端存储，保存每台 Agent 的 restic 仓库 |

## 快速开始

### 1. 启动 Master

使用 Docker Compose：

```bash
docker compose up -d
```

默认会拉取公开镜像 `ghcr.io/momo-z/vaultfleet:latest`。服务监听
`http://localhost:8080`，数据保存在当前目录的 `./data`：

```text
data/
├── vaultfleet.db
├── master.key
└── rollback/
```

也可以使用 Docker 直接启动：

```bash
docker run -d \
  --name vaultfleet \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  --restart unless-stopped \
  ghcr.io/momo-z/vaultfleet:latest
```

首次访问 Web UI 时，需要初始化管理员账号。

### 2. 添加节点并安装 Agent

在 Web UI 中创建节点后，Master 会生成一次性注册令牌。然后在目标服务器上执行：

```bash
curl -fsSL http://MASTER_HOST:8080/install.sh | bash -s -- \
  --server http://MASTER_HOST:8080 \
  --token ek_xxxxxxxxxxxxxxxxxxxxxxxx
```

安装脚本默认从 GitHub Releases 下载对应架构的 Agent：

```text
https://github.com/momo-z/VaultFleet/releases/latest/download/vaultfleet-agent-linux-amd64
https://github.com/momo-z/VaultFleet/releases/latest/download/vaultfleet-agent-linux-arm64
```

如果服务器访问 GitHub 较慢，可以像 Komari 一样指定 GitHub 代理前缀：

```bash
curl -fsSL http://MASTER_HOST:8080/install.sh | bash -s -- \
  --server http://MASTER_HOST:8080 \
  --token ek_xxxxxxxxxxxxxxxxxxxxxxxx \
  --github-proxy https://gh-proxy.example.com
```

`--agent-url` 是高级覆盖参数，用于指定完整的 Agent 二进制下载地址。普通用户
不需要填写；它主要用于测试未发布版本、内网镜像、自建 CDN 或临时下载源。

安装脚本会：

1. 检测 Linux 架构，目前支持 `amd64` 和 `arm64`。
2. 下载 `vaultfleet-agent` 到 `/usr/local/bin/`。
3. 安装 `restic` 和 `rclone`。
4. 创建 `/etc/vaultfleet/` 配置目录。
5. 使用一次性 token 向 Master 注册。
6. 创建并启动 systemd / OpenRC 服务；没有受支持的 init system 时使用 `nohup` 启动。

### 3. 卸载 Agent

如果需要从服务器上卸载 Agent，运行：

```bash
curl -fsSL https://raw.githubusercontent.com/momo-z/VaultFleet/main/build/uninstall.sh | bash
```

该脚本会停止服务、删除二进制文件（vaultfleet-agent、restic、rclone）和配置目录。

### 4. 配置备份策略

典型流程：

1. 在 Master 中添加存储配置，例如 S3、Cloudflare R2、MinIO、WebDAV 或其他 rclone 后端。
2. 为节点创建备份策略，选择备份目录、排除规则、cron 调度和保留策略。
3. Agent 在线后自动接收策略并确认同步。
4. 等待定时任务触发，或在节点详情中执行立即备份。

## 典型使用流程

1. 在 **存储配置** 中添加 S3 / R2 / MinIO、WebDAV、SFTP、本地路径或其他 rclone 后端，并执行连接测试。
2. 在 **节点管理** 中创建节点，复制安装命令到目标服务器执行，等待 Agent 注册上线。
3. 在 **备份策略** 中选择节点和存储，设置仓库子路径、备份目录、排除规则、Cron 调度、保留策略和任务超时。
4. 如使用 WebDAV、AList 代理或限流存储，在策略的 **高级传输参数** 中调整 rclone 并发、请求频率、重试和超时。
5. 在 **任务历史** 中查看手动备份、定时备份、恢复任务和运行中备份进度；必要时取消仍在运行的任务。
6. 在 **快照浏览** 中刷新快照、浏览快照内容，并选择整份快照或部分路径恢复到目标目录。
7. 需要迁移到新节点时，在新节点上创建使用相同存储和仓库子路径的策略，策略同步后即可看到原有快照。

## 常用命令

```bash
# 构建 Master
make build-master

# 构建 Agent
make build-agent

# 构建全部二进制
make build-all

# 运行测试
make test

# 构建 Master Docker 镜像
make docker-build
```

Agent 支持的主要参数：

```bash
vaultfleet-agent --config /etc/vaultfleet/agent.yaml

vaultfleet-agent --enroll-only \
  --server http://MASTER_HOST:8080 \
  --token ek_xxxxxxxxxxxxxxxxxxxxxxxx \
  --config /etc/vaultfleet/agent.yaml
```

## 通信协议

Master 和 Agent 通过 JSON WebSocket 消息通信，消息统一使用：

```json
{
  "type": "message_type",
  "id": "request_or_event_id",
  "payload": {}
}
```

当前协议包含：

| 类型 | 方向 | 用途 |
| --- | --- | --- |
| `heartbeat` | Agent -> Master | 上报在线状态、CPU、内存、磁盘和工具版本 |
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
| `dir_size_req` | Master -> Agent | 请求计算目录大小 |
| `dir_size_resp` | Agent -> Master | 返回目录大小 |
| `version_info` | Master -> Agent | 告知 Master 当前版本和下载信息 |
| `update_agent` | Master -> Agent | 请求 Agent 立即执行版本更新 |
| `backup_progress` | Agent -> Master | 上报运行中备份进度 |
| `cancel_task` | Master -> Agent | 请求取消运行中的任务 |

## 安全模型

- 生产环境建议通过反向代理启用 HTTPS / WSS。
- Agent 注册使用一次性 `ek_` token，注册成功后保存长期 `agent_token`。
- Master 使用 bcrypt 保存管理员密码哈希。
- rclone 配置、restic 仓库密码和通知凭据在 Master 数据库中加密保存。
- `/data/master.key` 是 Master 加密主密钥，必须和数据库一起备份并妥善保护。
- Agent 本地配置默认位于 `/etc/vaultfleet/agent.yaml`，安装目录权限限制为 root 可读写。

## 数据导出与恢复

Master 提供数据导出接口，可将 `/data` 打包为 zip。恢复时，将 `backup.zip` 放入数据目录，Master 启动时会自动检测并恢复，同时把恢复前数据保存到 `rollback/`。

这只包含 Master 的配置、元数据、密钥和任务记录；实际备份数据仍位于远端 restic 仓库。

## 项目结构

```text
.
├── cmd/
│   ├── master/              # Master 入口
│   └── agent/               # Agent 入口
├── internal/
│   ├── master/
│   │   ├── api/             # HTTP API、认证、前端入口、下载、诊断和健康检查
│   │   ├── backup/          # Master 数据导出和恢复
│   │   ├── commands/        # 命令队列、任务取消和离线投递
│   │   ├── db/              # SQLite / GORM / 加密
│   │   ├── events/          # 内部事件总线
│   │   ├── logbuf/          # Master 运行日志环形缓冲
│   │   ├── notify/          # Telegram / Webhook 通知
│   │   ├── storagecheck/    # rclone provider 和存储连接测试
│   │   └── ws/              # WebSocket Hub、进度缓存和离线检测
│   └── agent/
│       ├── connect/         # WebSocket 客户端、重连和心跳
│       ├── enroll/          # 注册流程
│       ├── executor/        # restic / rclone 执行封装
│       ├── filebrowse/      # 目录浏览和目录大小计算
│       ├── policy/          # 本地策略和待上报结果
│       ├── scheduler/       # cron 调度
│       └── selfupdate/      # Agent 自更新
├── pkg/protocol/            # Master / Agent 共享消息协议
├── web/                     # React / Vite / Tailwind Web UI
├── build/                   # Dockerfile、安装脚本和卸载脚本
├── docs/                    # 发布、支持、截图和设计/实施文档
├── docker-compose.yml
├── Makefile
└── go.mod
```

## 开发状态

VaultFleet 当前是可部署的早期版本，核心备份/恢复链路、Master + Agent 通信、React Web 控制台、安装脚本、Docker 镜像、GitHub Release 产物和自动化测试已经具备。

项目仍在快速迭代，建议生产使用时：

- 使用 HTTPS / WSS 反向代理保护 Master。
- 定期备份 `/data`，并确保 `/data/master.key` 与数据库一起保存。
- 先在非关键节点上验证存储后端、恢复流程和 Agent 更新流程。
- 遇到问题时按 [问题反馈和日志收集指南](docs/support.md) 收集日志并脱敏。

更多背景文档：

- [发布流程](docs/release.md)
- [问题反馈和日志收集指南](docs/support.md)
- [核心设计](docs/superpowers/specs/2026-05-18-vaultfleet-design.md)
- [端到端验收测试方案](docs/superpowers/specs/2026-05-21-vaultfleet-e2e-acceptance-test-v2.md)
- [任务取消设计](docs/superpowers/specs/2026-05-27-task-cancellation-design.md)
- [快照浏览缓存实施计划](docs/superpowers/plans/2026-05-28-snapshot-browse-cache.md)

## 反馈问题 / Report an issue

遇到 bug 或需要排障支持时，请先阅读 [问题反馈和日志收集指南](docs/support.md)。

提交问题时使用 GitHub Issue 表单：

- [选择 Issue 类型](https://github.com/momo-z/VaultFleet/issues/new/choose)
- [Bug report](https://github.com/momo-z/VaultFleet/issues/new?template=bug_report.yml)
- [Support request](https://github.com/momo-z/VaultFleet/issues/new?template=support_request.yml)

VaultFleet 不会连接或保存你的 GitHub 账号；提交账号由浏览器里的 GitHub 登录态决定。发布日志前请按指南脱敏 token、密码和存储凭据。

## 参考

- [Komari Monitor](https://github.com/komari-monitor/komari)：Agent 注册、WebSocket 通信、任务下发等探针式部署体验。
- [Nezha](https://github.com/nezhahq/nezha)：Dashboard + Agent 的监控面板项目形态。
- [restic](https://restic.net/)：加密备份引擎。
- [rclone](https://rclone.org/)：多存储后端适配。

## English

VaultFleet is a centralized backup management system for multiple VPS or Linux servers. It uses a **Master + Agent** architecture: the Master provides the Web UI, API, policy management, task history, snapshots, and notifications; each Agent runs on a server, connects back to the Master, receives backup policies, and writes encrypted `restic` repositories to remote storage through `rclone`.

The deployment experience is inspired by probe-style projects such as Komari and Nezha: nodes do not need inbound ports, and a server can be enrolled with a single installation command. VaultFleet focuses on backups, snapshots, and restore workflows rather than general-purpose monitoring.

## Features

- **Master + Agent architecture** with outbound-only Agent connections.
- **Centralized policy management** for backup paths, excludes, schedules, retention, task timeout, and storage settings.
- **React Web console** for dashboard, nodes, storage, policies, tasks, snapshots, notifications, and system management.
- **One-time enrollment tokens** followed by long-lived Agent tokens.
- **WebSocket control plane** for heartbeat, directory browsing, directory size calculation, policy push, backup triggers, task cancellation, snapshots, selective restore, and Agent version messages.
- **No backup data through Master**: Agents write directly to storage using `restic` and `rclone`.
- **Encrypted backups** with per-agent restic repository passwords and AES-256-GCM encrypted Master-side secrets.
- **Backup progress and cancellation** for long-running jobs, with policy-level timeout settings.
- **Snapshot browsing and selective restore** with Agent-side snapshot browse cache for large repositories.
- **rclone advanced transfer parameters** for storage-specific tuning, including conservative WebDAV defaults.
- **Agent version reporting and self-update** through GitHub Release assets.
- **Telegram, Webhook, health, diagnostics, and log collection** for operations and troubleshooting.
- **Docker-friendly Master** and single-binary Linux Agent installation.

## Screenshots

**Dashboard** — System status, online nodes, policy sync, 24h task stats, and recent task list.

![Dashboard](docs/screenshots/dashboard.png)

**Node Management** — View all Agent nodes with online status, system info, and one-click install command generation.

![Node Management](docs/screenshots/nodes.png)

**Node Detail** — System info, connection status, policies, snapshots, task history, command queue, and remote file browser for a single node.

![Node Detail](docs/screenshots/node-detail.png)

**Backup Policy** — Select node and storage, configure backup directories, cron schedule, retention, timeout, and rclone transfer parameters.

![Backup Policy](docs/screenshots/policy-create.png)

**Storage Configuration** — Supports S3 / R2 / MinIO, WebDAV, SFTP, local paths, and other rclone backends with built-in connection testing.

![Storage Configuration](docs/screenshots/storage-create.png)

## Architecture

```text
┌──────────────────────────────────────────────┐
│                   Master                      │
│  Web UI / API / SQLite / Policy / Notify      │
└──────────────────────┬───────────────────────┘
                       │ WebSocket control plane
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │ Agent A │    │ Agent B │    │ Agent C │
   └────┬────┘    └────┬────┘    └────┬────┘
        │              │              │
        └──────────────┼──────────────┘
                       ▼
        S3 / R2 / MinIO / WebDAV / SFTP / rclone backends
```

Design rules:

- Master manages the control plane only. It does not receive or relay backup data.
- Agents keep a local policy copy, so scheduled backups can continue during temporary Master downtime.
- Each server uses its own repository path and restic password.
- Agents only make outbound connections to the Master and storage backend.

## Quick Start

Start the Master with Docker Compose:

```bash
docker compose up -d
```

This pulls the public image `ghcr.io/momo-z/vaultfleet:latest`.
The service listens on `http://localhost:8080` and stores persistent data in `./data`.

Or run the container directly:

```bash
docker run -d \
  --name vaultfleet \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  --restart unless-stopped \
  ghcr.io/momo-z/vaultfleet:latest
```

After creating a node in the Web UI, install the Agent on the target server:

```bash
curl -fsSL http://MASTER_HOST:8080/install.sh | bash -s -- \
  --server http://MASTER_HOST:8080 \
  --token ek_xxxxxxxxxxxxxxxxxxxxxxxx
```

The installer downloads the Agent from GitHub Releases by default:

```text
https://github.com/momo-z/VaultFleet/releases/latest/download/vaultfleet-agent-linux-amd64
https://github.com/momo-z/VaultFleet/releases/latest/download/vaultfleet-agent-linux-arm64
```

If GitHub is slow from the target server, add a proxy prefix:

```bash
curl -fsSL http://MASTER_HOST:8080/install.sh | bash -s -- \
  --server http://MASTER_HOST:8080 \
  --token ek_xxxxxxxxxxxxxxxxxxxxxxxx \
  --github-proxy https://gh-proxy.example.com
```

`--agent-url` is an advanced override for a full Agent binary URL.
It is useful for unpublished builds, private mirrors, internal CDNs, or temporary download sources.

The installer downloads the Agent, installs `restic` and `rclone`, enrolls the node, and starts the Agent with systemd, OpenRC, or `nohup`.

### Uninstall Agent

To remove the Agent from a server:

```bash
curl -fsSL https://raw.githubusercontent.com/momo-z/VaultFleet/main/build/uninstall.sh | bash
```

This stops the service and removes binaries (vaultfleet-agent, restic, rclone) and the configuration directory.

## Typical Workflow

1. Add a storage backend and run the connection test.
2. Create a node, copy the generated install command, and wait for the Agent to enroll.
3. Create a backup policy with repository path, backup directories, excludes, cron schedule, retention, and timeout.
4. Tune rclone transfer parameters when using WebDAV, AList proxies, or rate-limited storage.
5. Track backup, restore, queued, cancelled, failed, and running tasks from the task history page.
6. Browse snapshots, inspect snapshot contents, and restore either a full snapshot or selected paths.
7. For cross-node restore, create a policy on the new node with the same storage and repository path.

## Common Commands

```bash
make build-master    # Build Master
make build-agent     # Build Agent
make build-all       # Build both binaries
make test            # Run tests with race detector
make docker-build    # Build Master Docker image
```

## Security

- Use HTTPS / WSS in production, typically behind a reverse proxy.
- One-time `ek_` enrollment tokens are exchanged for long-lived Agent tokens.
- Admin passwords are stored as bcrypt hashes.
- Storage credentials, restic passwords, and notification secrets are encrypted in the Master database.
- Keep `/data/master.key` safe; it is required to decrypt Master-side secrets.
- Agent configuration is stored under `/etc/vaultfleet/` and should be readable only by root.

## Development Status

VaultFleet is a deployable early-stage project. The core backup and restore flow, Master + Agent protocol, React Web console, installer, Docker image, GitHub Release artifacts, and automated tests are in place.

For production use:

- Put the Master behind HTTPS / WSS.
- Back up `/data`, including `/data/master.key`.
- Validate storage, restore, and Agent update flows on non-critical nodes first.
- Use the [support and log collection guide](docs/support.md) when reporting issues.

More docs:

- [Release guide](docs/release.md)
- [Support and log collection guide](docs/support.md)
- [Core design](docs/superpowers/specs/2026-05-18-vaultfleet-design.md)
- [E2E acceptance test plan](docs/superpowers/specs/2026-05-21-vaultfleet-e2e-acceptance-test-v2.md)
- [Task cancellation design](docs/superpowers/specs/2026-05-27-task-cancellation-design.md)
- [Snapshot browse cache plan](docs/superpowers/plans/2026-05-28-snapshot-browse-cache.md)

## Report an issue

For bugs or troubleshooting support, read the [issue reporting and log collection guide](docs/support.md), then open a GitHub Issue:

- [Choose an Issue type](https://github.com/momo-z/VaultFleet/issues/new/choose)
- [Bug report](https://github.com/momo-z/VaultFleet/issues/new?template=bug_report.yml)
- [Support request](https://github.com/momo-z/VaultFleet/issues/new?template=support_request.yml)

VaultFleet does not connect to or store your GitHub account. GitHub uses your browser login, and logs should be redacted before posting tokens, passwords, or storage credentials.

## References

- [Komari Monitor](https://github.com/komari-monitor/komari)
- [Nezha](https://github.com/nezhahq/nezha)
- [restic](https://restic.net/)
- [rclone](https://rclone.org/)

<!-- markdownlint-enable MD013 -->
