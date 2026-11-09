# HookLab

**Webhook Inspector & Event Replay Platform**

HookLab 是一个面向开发者的 Webhook 调试与事件管理平台。它可以接收 HTTP Webhook 请求，将请求数据持久化到 MySQL，并提供请求检索、详情检查和事件重放能力。

项目采用前后端分离架构，提供 Docker Compose 部署方式，适合本地开发、接口调试和学习 Web 后端工程实践。

> **项目状态：** v1.0 MVP 已完成。当前版本面向个人开发与受控环境使用；公开部署前请阅读下方的[安全说明](#安全说明)和[当前限制](#当前限制)。

## 功能特性

- **Webhook Endpoint 管理**：创建、重命名、暂停、恢复和删除接收地址；每个 Endpoint 使用独立的随机 Token。
- **HTTP 请求捕获**：接收标准及自定义 HTTP Method，记录请求方法、路径、Host、Headers、Query、原始 Body、来源 IP 和接收时间。
- **事件检索**：按 Endpoint、HTTP Method 和内容搜索事件，支持游标分页及自动刷新。
- **请求检查**：查看 JSON、Raw Body、Headers 和 Query；支持下载原始请求体。
- **事件重放**：将历史请求发送到指定目标，记录响应状态、耗时和失败原因。
- **账户与访问控制**：支持注册、登录、退出和基于用户的资源隔离。
- **审计与安全防护**：记录重放结果；提供 CSRF 防护、请求体限制、敏感请求头脱敏及 SSRF 防护。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React、TypeScript、Vite、Tailwind CSS |
| 后端 | Go、Gin |
| 数据库 | MySQL 8.4 |
| 数据访问 | `database/sql`、参数化 SQL、SQL migrations |
| Web 服务 | Nginx（静态前端） |
| 容器化 | Docker、Docker Compose |
| 测试 | Go Test、Race Detector、Playwright |

## 系统架构

```text
浏览器 ───────┐
              ├── Go + Gin API ─── MySQL 8.4
Webhook 客户端┘         │
                        └── 内部 Nginx（React 静态资源）
```

Go 服务负责 API、身份认证、Webhook 接收、事件持久化和重放。Nginx 仅提供前端静态资源；Webhook 请求由 Go 服务直接处理。

### 项目结构

```text
.
├── backend/
│   ├── cmd/server/       # 服务入口、配置、生命周期与健康检查
│   ├── internal/app/     # 认证、Endpoint、事件、重放及安全逻辑
│   ├── migrations/       # SQL 数据库迁移
│   └── Dockerfile
├── web/
│   ├── src/              # React + TypeScript 前端
│   ├── tests/             # Playwright 端到端测试
│   ├── nginx.conf
│   └── Dockerfile
├── scripts/               # 初始化与测试脚本
├── docs/                  # 项目文档
├── compose.yaml            # 默认部署配置
├── compose.dev.yaml        # 本地开发数据库端口映射
└── .env.example            # 环境变量模板
```

数据库迁移由应用启动时自动执行。当前包含用户、会话、Endpoint、Webhook 事件、重放记录和迁移版本等数据表。

## 快速开始

### 环境要求

- Docker Engine
- Docker Compose v2（支持 `--wait`）
- OpenSSL
- Bash 环境

Windows 用户可以通过 Docker Desktop + WSL2，在 WSL2 终端中执行以下命令。

### 启动服务

克隆仓库并进入项目目录：

```bash
git clone https://github.com/Lcpt7/HookLab.git
cd HookLab
```

初始化本地环境并启动：

```bash
./scripts/setup.sh
docker compose up -d --build --wait
```

查看容器状态：

```bash
docker compose ps
```

服务启动后访问：

**http://localhost:8080**

首次使用时注册账户并创建 Endpoint。项目不提供预置账户或演示事件。

### 发送测试请求

在 HookLab 工作台复制 Endpoint URL，或使用以下示例（将 Token 替换为实际值）：

```bash
curl -X POST 'http://localhost:8080/h/YOUR_TOKEN/orders?version=1' \
  -H 'Content-Type: application/json' \
  -d '{"event":"order.created","amount":2048}'
```

成功接收时返回 HTTP `202`。默认请求体上限为 1 MiB，超过限制的请求返回 `413`；暂停或不存在的 Endpoint 返回 `404`。

### 常用命令

```bash
# 查看服务日志
docker compose logs -f api web

# 停止服务（保留数据库卷）
docker compose stop

# 停止并移除容器和网络（保留数据库卷）
docker compose down
```

MySQL 数据保存在 Docker 命名卷 `mysql_data` 中。**不要在需要保留数据时执行 `docker compose down -v`，该命令会删除数据卷。**

## 配置说明

`./scripts/setup.sh` 会生成随机数据库密码并创建本地 `.env` 文件；如果文件已存在，不会覆盖。实际凭证不应提交到 Git。

| 变量 | 说明 |
| --- | --- |
| `MYSQL_PASSWORD` | 必填，应用数据库用户密码 |
| `MYSQL_ROOT_PASSWORD` | 必填，MySQL root 密码；API 不使用 root 账户 |
| `APP_ORIGIN` | 浏览器访问来源，默认 `http://localhost:8080`；必须与实际访问来源完全一致，不带尾部斜杠 |
| `COOKIE_SECURE` | 本地 HTTP 使用 `false`；HTTPS 部署必须设为 `true` |
| `PORT` | 宿主机 Web 端口，默认 `8080`；修改时同步调整 `APP_ORIGIN` |
| `MAX_BODY_BYTES` | 请求体大小上限，默认 `1048576`，允许范围为 1–8388608 字节 |
| `FRONTEND_URL` | Compose 内部前端地址，默认由 Compose 配置为 `http://web:80` |
| `DATABASE_DSN` | 原生启动 API 时使用的 MySQL 连接串；Compose 部署时自动组装 |
| `TRUSTED_PROXIES` | 可选，可信代理 IP/CIDR 列表；默认不信任转发头 |
| `TEST_DATABASE_DSN` | Go 集成测试使用的数据库连接串 |
| `GO_BIN` | 可选，Go 测试脚本使用的 Go 可执行文件路径 |
| `BASE_URL` | Playwright 测试目标地址，默认 `http://localhost:5173` |
| `CHROMIUM_PATH` | 可选，指定本机 Chromium 可执行文件路径 |

生产环境应在 Go 服务前配置 HTTPS，设置正确的 `APP_ORIGIN`，并启用 `COOKIE_SECURE=true`。不要直接将默认本地配置用于公网部署。

## 本地开发

### 依赖

- Go 1.25+
- Node.js 22.12+
- npm
- Docker Compose

先初始化环境并启动开发用 MySQL：

```bash
./scripts/setup.sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --wait db
```

终端一：启动 Go API（在仓库根目录执行）

```bash
set -a
. ./.env
set +a
export APP_ORIGIN=http://localhost:5173
export DATABASE_DSN="hooklab:${MYSQL_PASSWORD}@tcp(127.0.0.1:3306)/hooklab?parseTime=true&charset=utf8mb4&loc=UTC"
cd backend
go run ./cmd/server
```

终端二：启动前端开发服务器

```bash
cd web
npm ci
npm run dev
```

访问 **http://localhost:5173**。Vite 会将 API 和 Webhook 请求代理到本机 Go 服务的 `8080` 端口。若完整 Compose 应用已经占用该端口，请先停止对应服务，避免端口冲突。

## 测试

### Go 测试

集成测试需要真实 MySQL 数据库。建议使用独立测试数据库，不要将测试指向生产数据。

```bash
./scripts/setup.sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --wait db
./scripts/test-go.sh
cd backend
go vet ./...
```

`scripts/test-go.sh` 会设置测试连接并运行：

```bash
go test -race ./... -count=1
```

如果未设置 `TEST_DATABASE_DSN`，直接运行部分 Go 测试可能跳过 MySQL 集成测试；不能仅凭这种结果认定集成测试已通过。

### 前端与端到端测试

先启动完整 Compose 应用，再执行：

```bash
cd web
npm ci
npx playwright install --with-deps chromium
BASE_URL=http://localhost:8080 npm test
npm run build
```

测试使用真实 API 和本地数据库，不使用 API Mock。失败时会保留 Playwright 截图和 Trace。

## 安全说明

HookLab 包含多项基础安全控制，但它仍是一个 MVP 工具，不应被视为经过独立安全认证或渗透测试的生产级 Webhook 基础设施。

- **密码与会话**：密码使用 Argon2id；Session 使用随机值，数据库仅保存其 SHA-256 摘要。Cookie 设置 HttpOnly、SameSite=Lax，并可配置 Secure。
- **CSRF 与来源校验**：注册、登录及管理写请求校验 Origin；登录后的管理请求还需要 Session 绑定的 CSRF Token。
- **数据访问控制**：管理 API 在查询资源时同时校验所属用户；SQL 使用参数化查询。
- **请求捕获**：限制请求体大小；默认对 Authorization、Cookie、Token、Secret、Password 等敏感请求头脱敏。Headers 脱敏不可恢复，但 Body 和 Query 会原样保存。
- **来源 IP**：默认使用实际 TCP 连接来源，忽略未经信任的转发头。仅在代理配置可信且代理会覆盖外部转发头时设置 `TRUSTED_PROXIES`。
- **事件重放与 SSRF 防护**：仅允许公网 HTTP/HTTPS 目标及标准端口；校验 DNS 解析结果并固定到已验证 IP；拒绝私网、回环和保留地址；不跟随重定向、不使用环境代理，并限制超时及响应大小。
- **限流**：认证、Webhook 捕获和重放均有按来源 IP 的限流。当前限流为单进程内存实现，进程重启后重置。

### 敏感数据注意事项

Webhook Body 和 Query 会原样保存，可能包含业务数据或凭证。请勿将生产密钥、密码或个人敏感信息发送到不受信任的 HookLab 实例。

当前 Endpoint Token 会以明文形式存储，以便工作台展示完整接收 URL。因此应限制数据库和备份的访问权限，并将数据库备份视为敏感数据。

## API 概览

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/healthz`、`/readyz` | 存活与就绪检查 |
| `POST` | `/api/auth/register`、`/api/auth/login`、`/api/auth/logout` | 用户认证 |
| `GET` | `/api/auth/me` | 当前用户与 CSRF Token |
| `GET`、`POST` | `/api/endpoints` | Endpoint 列表与创建 |
| `PATCH`、`DELETE` | `/api/endpoints/:id` | 更新与删除 Endpoint |
| 任意 HTTP Method | `/h/:token[/path]` | 接收 Webhook |
| `GET` | `/api/events` | 事件列表与筛选 |
| `GET` | `/api/events/:id` | 事件详情 |
| `POST` | `/api/events/:id/replay` | 重放事件 |
| `GET` | `/api/events/:id/replays` | 查看最近重放记录 |

事件列表支持 `endpoint_id`、`method`、`q` 和 `before` 参数；每页最多返回 50 条。HTTP Method 筛选区分大小写。事件详情中的原始 Body 以 Base64 字段返回。

## 当前限制与后续规划

当前版本适合个人开发、学习和受控测试环境。以下能力尚未纳入 v1.0：

- 自动事件保留期清理与用户存储配额
- Endpoint Token 轮换
- 密码找回与邮箱验证
- 批量或计划重放、自动重试队列
- 大规模全文检索与多实例共享限流
- WebSocket 实时推送

在增加数据保留策略、存储配额和多实例限流前，不建议将服务作为无约束的公网接收端开放给不特定用户。

## 版本验证

v1.0 MVP 于 2026-11-09 完成验证，测试环境为 MySQL 8.4 和 Docker Compose：

- Go：18 个顶层测试，含子测试共 47 项通过；Race Detector、`go vet` 通过。
- 前端：TypeScript 检查、Vite 生产构建通过。
- Playwright：4 项端到端测试通过。
- 容器：MySQL、API、Web 服务构建成功，健康检查通过。
- 回归测试：覆盖 PATCH 状态保留、Host 持久化及 HTTP Method 大小写筛选。

---

**Repository:** [github.com/Lcpt7/HookLab](https://github.com/Lcpt7/HookLab)
