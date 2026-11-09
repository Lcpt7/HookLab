# HookLab

面向开发者的 Webhook 调试与事件管理平台。真实接收请求、持久化到 MySQL，在深色工作台中搜索、检查和重放事件。前后端分离，无 ORM、Redis 或额外微服务。

## 已实现

- 注册、登录、退出；Argon2id 密码哈希，服务端 Session。
- Endpoint 创建、改名、暂停/恢复和永久删除；独立的 256 位随机 Token。PATCH 省略 `active` 时保持当前状态，显式传入 `false`/`true` 时停用/恢复。
- 任意 HTTP Method 接收，包括自定义方法；保存方法、路径、Headers（含真实 Host 及端口）、原始 Query、原始 Body、来源 IP 和 UTC 时间。
- 请求体以 MySQL `MEDIUMBLOB` 存储，API 通过 Base64 传输，保留二进制原始字节。
- Endpoint 工作区、全部事件、区分大小写的方法筛选、路径/Body 搜索、游标分页，每 5 秒自动刷新。
- JSON 高亮和格式化、Raw 文本、Headers、Query、原始 Body 下载。页面预览最多 200 行 / 24,000 字符，大文本跳过语法高亮，完整数据仍可下载。
- 真实 HTTP 重放；保存目标、状态码、耗时与失败原因，在事件详情查看最近 50 次结果。
- 响应式桌面/移动布局、空状态、加载反馈、操作错误和删除确认。

## 技术架构与目录

```text
浏览器 / Webhook → Go + Gin :8080 → MySQL 8.4
                       └→ 内部 Nginx（静态 React）
```

```text
backend/
  cmd/server/           配置、迁移启动、HTTP 生命周期和健康检查
  internal/app/         认证、CSRF、Endpoint、捕获、查询、重放和安全策略
  migrations/           嵌入式 SQL 迁移、版本记录与 MySQL advisory lock
  Dockerfile            Go 构建 → 非 root distroless 运行镜像
web/
  src/                  React + TypeScript + Tailwind 工作台
  tests/                Playwright 真实 API 端到端测试
  nginx.conf            内部静态资源、CSP
  Dockerfile            Vite 构建 → Nginx
scripts/
  setup.sh              生成随机数据库密码，不覆盖已有 .env
  test-go.sh            使用真实 MySQL 跑 Go/race 测试
compose.yaml            三个服务、持久化卷、健康检查和隔离网络
compose.dev.yaml        仅供本机开发的 MySQL 端口映射
.env.example            环境变量说明模板，无可用凭证
```

数据库包含 `users`、`sessions`、`endpoints`、`events`、`replays` 和 `schema_migrations`。外键级联删除；邮箱和 Endpoint Token 唯一；索引支持用户 Endpoint、Endpoint 事件 ID/方法、Session 过期时间和事件重放记录查询。查询使用参数绑定，资源查询同时限定所属用户。请求时间存储为 UTC，浏览器按本地时区显示。

## Docker 启动

需要 Docker Engine、支持 BuildKit 的 Docker Compose v2，以及本机的 OpenSSL。

```bash
./scripts/setup.sh
docker compose up -d --build --wait
docker compose ps
curl -f http://localhost:8080/healthz
curl -f http://localhost:8080/readyz
```

打开 **http://localhost:8080**，创建自己的账户，再创建 Endpoint。没有预设账户或演示事件。将工作台给出的 URL 填入 Webhook 提供方，或使用界面中的 curl 命令。

```bash
curl -X POST 'http://localhost:8080/h/替换为你的TOKEN/orders?version=1' \
  -H 'Content-Type: application/json' \
  -d '{"event":"order.created","amount":2048}'
```

接收成功返回 `202 {"id":...,"status":"captured"}`；请求体默认限 1 MiB，超限返回 413。暂停或不存在的 Endpoint 返回 404。

```bash
docker compose logs -f api web
docker compose stop
```

MySQL 数据保存在 `mysql_data` 卷中。启动 API 时自动运行尚未应用的迁移；重复启动不会重复应用。MySQL DDL 会自动提交，初始迁移可幂等重试；新增变更请添加递增 SQL 文件，不修改已部署迁移。当前不提供破坏性的自动 down 迁移。

### 环境变量

| 变量 | 用途 / 默认值 |
| --- | --- |
| `MYSQL_PASSWORD` | 必填；setup 脚本生成随机密码，Compose app 数据库用户使用 |
| `MYSQL_ROOT_PASSWORD` | 必填；独立随机 root 密码，API 不使用 root |
| `APP_ORIGIN` | 浏览器完整来源，默认 `http://localhost:8080`，无尾部 `/`；必须与访问地址完全一致 |
| `COOKIE_SECURE` | 本地 HTTP 用 `false`；HTTPS 部署必须 `true` |
| `PORT` | 宿主机 Web 端口，默认 `8080`；改端口时同时修改 `APP_ORIGIN` |
| `MAX_BODY_BYTES` | 默认 `1048576`，允许 1–8388608；由 Go 入口直接限制 |
| `FRONTEND_URL` | Compose 设为 `http://web:80`；Go 将页面请求转发到内部静态服务；原生 Vite 开发无需设置 |
| `DATABASE_DSN` | 原生启动 API 必填，包含 `parseTime=true&charset=utf8mb4&loc=UTC`；Compose 自动组装 |
| `TRUSTED_PROXIES` | 原生 API 可选，逗号分隔的可信 IP/CIDR；默认不信任代理 |
| `TEST_DATABASE_DSN` | Go 集成测试连接串；test-go.sh 默认使用本机 3306 的 HookLab 数据库 |
| `GO_BIN` | test-go.sh 可选 Go 可执行文件路径，默认 `go` |
| `BASE_URL` | Playwright 测试地址，默认 `http://localhost:5173` |
| `CHROMIUM_PATH` | Playwright 可选浏览器可执行路径；不设置时使用 Playwright 管理的 Chromium |

`.env` 已被 Git 忽略。不要提交它或把数据库密码写入版本控制。生产部署请在 Go 入口前配置 HTTPS，设置真实 `APP_ORIGIN` 和 `COOKIE_SECURE=true`；默认配置面向本地开发。

如果构建环境使用企业 HTTPS 代理，可用 Compose build secret `proxy_ca` 为两个构建阶段传入 CA PEM；普通本地构建不需要该 Secret。构建过程保持 TLS 验证，不将代理 CA 写入运行镜像。

## 原生开发

需要 Go 1.25+、Node.js 22.12+、npm 和 Docker Compose。

```bash
./scripts/setup.sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --wait db
```

终端 1（仓库根目录）：

```bash
set -a
. ./.env
set +a
export APP_ORIGIN=http://localhost:5173
export DATABASE_DSN="hooklab:${MYSQL_PASSWORD}@tcp(127.0.0.1:3306)/hooklab?parseTime=true&charset=utf8mb4&loc=UTC"
cd backend
go run ./cmd/server
```

终端 2：

```bash
cd web
npm ci
npm run dev
```

打开 **http://localhost:5173**。Vite 将 API/Webhook 代理到本机 8080，浏览器仍使用同源会话。如果已经在 8080 运行 Compose 应用，请先停止其 Web/API 服务以避免端口冲突。

## 测试

### Go 单元测试与真实 MySQL 集成测试

测试会创建并清理各自的用户数据，建议使用独立开发/测试数据库，不要指向生产库。

```bash
./scripts/setup.sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --wait db
# 等待 docker compose ps 中数据库显示 healthy
./scripts/test-go.sh
cd backend
go vet ./...
```

未设置 `TEST_DATABASE_DSN` 时直接执行 `go test ./...` 会跳过 MySQL 集成测试，不能据此声称集成测试通过。`test-go.sh` 会明确设置连接串并执行 `go test -race ./... -count=1`。

覆盖健康/数据库就绪、迁移幂等、认证校验与退出、Session 哈希存储/过期/轮换、Argon2id、CSRF、跨用户隔离、自定义方法、二进制 Body、请求体限制、脱敏、过滤搜索、暂停/删除、来源 IP、重放投递/持久化、SSRF 地址/DNS/端口策略、重定向与凭证剥离。重放成功测试使用真实本地 HTTP 接收器，通过受控 DNS/拨号注入验证公网策略后的投递，不依赖外部服务。

### Playwright

先启动完整 Compose 应用，再运行：

```bash
cd web
npm ci
npx playwright install --with-deps chromium
BASE_URL=http://localhost:8080 npm test
npm run build
```

如果系统已安装 Chromium，可用 `CHROMIUM_PATH=/usr/bin/chromium BASE_URL=http://localhost:8080 npm test`。原生开发服务运行时可直接 `npm test` 使用 5173。

测试走真实注册、创建 Endpoint、发送 Webhook、查看 JSON/Raw/Query/脱敏头、拒绝元数据地址重放、搜索、改名、退出并重新登录、移动导航与删除。失败保留截图和 trace；浏览器测试数据仅写入配置的本地数据库，不使用 API mock。

## 安全约束

- Argon2id 参数：64 MiB、3 次迭代、2 个线程，随机 16 字节盐。
- Session 使用 256 位随机值；数据库仅存其 SHA-256，7 天到期。Cookie 为 HttpOnly、SameSite=Lax，可配置 Secure；登录轮换已有 Cookie，退出销毁服务端会话。
- 注册、登录及所有管理写请求要求精确 `Origin`；登录后还必须携带 Session 绑定的 `X-CSRF-Token`。Webhook 接收路由单独使用 URL Token，不依赖浏览器 Session。
- 默认脱敏 Authorization、Cookie、API-Key、Token、Secret、Signature、Credential、Password、Auth 等敏感请求头。Headers 脱敏不可恢复；Body 和 Query 原样保存，避免发送生产凭证。
- API 日志仅记录方法、状态与耗时，不记录包含 Token 的请求 URL 或 Body。Nginx 仅接收静态页面请求且关闭访问日志，Webhook 不经过 Nginx。
- 默认使用实际连接的来源 IP，忽略伪造转发头。如果外层另有 TLS 代理，显式配置 `TRUSTED_PROXIES` 并让该代理覆写外来转发头。
- 重放只允许公网 HTTP:80、HTTPS:443，无 URL 凭证、无片段。拒绝私网、回环、链路本地、文档/基准/保留地址、IPv4 映射私网及 IPv6 隧道地址；全部 DNS 结果都必须安全，并固定到已校验 IP 拨号。
- 重放不用环境 HTTP 代理，不跟随重定向，保留 TLS 验证；最长 10 秒，响应头最多 32 KiB，响应体最多读取 64 KiB，不保存响应 Body。
- 重放转发原方法、Body 与非敏感非 hop-by-hop 头；不自动附加原始 Query。目标 URL 中填写需要的 Query。结果失败也会记录，外部请求可能产生真实业务副作用。
- 认证每 IP 每分钟 20 次且最多 4 个并发哈希；捕获每 IP 每分钟 600 次，重放每 IP 每分钟 30 次。限流为单进程内存实现，重启重置。

## API 摘要

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz`, `/readyz` | 存活 / 数据库就绪 |
| POST | `/api/auth/register`, `/api/auth/login`, `/api/auth/logout` | 认证 |
| GET | `/api/auth/me` | 用户和 CSRF Token |
| GET / POST | `/api/endpoints` | 列表 / 创建 |
| PATCH / DELETE | `/api/endpoints/:id` | 修改名称（`active` 可省略，省略保留原状态）/ 级联删除 |
| 任意方法 | `/h/:token[/path]` | 接收 Webhook |
| GET | `/api/events` | `endpoint_id`、`method`、`q`、`before`；方法按二进制比较区分大小写；每页 50 条，返回 `next_cursor` |
| GET | `/api/events/:id` | 详情，Body 在 `body_base64` |
| POST | `/api/events/:id/replay` | JSON `{"target":"https://..."}` |
| GET | `/api/events/:id/replays` | 最近 50 条重放结果 |

## MVP 边界

本版本已实现四阶段核心流程。为支持 TRACE、CONNECT、大小写与标点自定义方法，Go 直接接收公网 HTTP，Nginx 只提供内部静态页面。暂未加入密码找回/邮箱验证、Token 轮换、自动保留期清理、大数据量全文索引、WebSocket 推送、批量或计划重放、重试队列和多实例共享限流。当前搜索是用户范围内的参数化 SQL 内容匹配，适合 MVP 数据规模；Endpoint Token 明文存储用于展示完整接收 URL，数据库备份也应按敏感数据保护。

## 本次验证记录

2026-11-09，在真实 MySQL 8.4 与正式 Compose 镜像上完成：

- Go：18 个顶层测试，含子测试共 47 项通过；`-race`、`go vet`、静态二进制构建通过。新增回归覆盖 PATCH 状态保留/显式停用、Host 持久化和 HTTP Method 大小写筛选。
- 前端：TypeScript 检查、Vite 生产构建、代码格式检查通过。
- Playwright：4 项测试通过，包括完整业务链路、未登录/失败登录、大载荷预览和忙碌弹窗。
- 容器：MySQL、API、Web 健康检查通过；12 种 HTTP 方法通过公网入口接收并持久化，CSP、来源 IP 与安全检查通过。
