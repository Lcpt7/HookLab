# HookLab MVP 实施记录

用户要求直接在当前仓库实施，按四阶段推进，不等待额外方案审批。

架构：React/TypeScript/Vite/Tailwind 前端；Gin + database/sql 后端；MySQL 8.4；Compose 以 Go 为公开同源入口，Nginx 仅提供内部静态资源。单应用，不引入缓存或队列。

1. 初始化：backend/cmd/server、internal/app、migrations；版本化 SQL 迁移、健康/就绪检查；Compose 数据库健康检查。Go 测试验证健康接口及迁移落库。
2. 业务：Argon2id、服务端 Session（随机凭证 SHA256 后入库）、同源与 CSRF 校验；所有资源 SQL 限定 user_id；256 位 Endpoint Token；原始二进制请求体、脱敏 Headers、原始 Query、真实连接 IP。Go + MySQL 集成覆盖认证、隔离、持久化、体积限制。
3. 工作台：真实认证、Endpoint 增删改、轮询历史、搜索与方法筛选、分页、详情、JSON/Raw。Playwright 覆盖真实数据库端到端流程、空状态、错误与移动布局。
4. 重放：仅 HTTP/HTTPS，限制端口 80/443，拒绝私网/保留地址；解析后固定拨号 IP 防 DNS 重绑定；禁止代理与重定向，限制超时及响应体；脱敏凭证不转发，记录结果。Go 安全测试覆盖地址分类及重定向。

决策：本地 HTTP 允许 COOKIE_SECURE=false；生产必须 HTTPS+true。Body 以 base64 API 返回保证任意字节无损；UI 解码文本展示。Webhook 不依赖 CSRF，但管理 API 要求精确 Origin 与 Session 绑定 CSRF。请求历史按递增 ID 倒序及 before 游标分页，搜索仅搜索路径和 Body（上限 200 字节）。

阶段结果随开发追加。

阶段 1：Go 健康检查、MySQL 迁移幂等性已通过。
阶段 2：Go + MySQL 集成覆盖隔离、脱敏、原始字节、停用、级联、CSRF、退出，已通过。
阶段 3：TypeScript/Vite 构建与 Playwright 全链路（4.7s）已通过，桌面/移动截图已检查。


最终架构修正：容器实测确认 Nginx 拒绝 TRACE/CONNECT、lowercase 及部分 RFC token 方法。公开同源入口改为现有 Go API，Webhook 直接处理，页面请求反向代理到内部 Nginx。三个服务不变，去掉固定代理网段，默认不信任 X-Forwarded-For。新增真实 HTTP 前端代理、Cookie 剥离和 12 种方法持久化测试。

独立审查：修复大载荷无界 DOM（最多 200 行/24k 字符，超过 12k 不高亮），保留完整下载；修复忙碌弹窗 Escape 隐藏后无法展示失败的问题。两项浏览器回归均已先观察失败，再观察通过。

阶段 4：重放、审计落库、SSRF、DNS 固定拨号、禁止代理/重定向、来源 IP 测试已通过。
初次 MVP Go 检查：15 个顶层测试，含子测试共 38 项；真实 MySQL、race、go vet、静态二进制编译均通过。容器路由与 Playwright 复验结果如下。

最终容器检查通过：两个正式 Dockerfile 构建成功，MySQL/API/静态 Web 全部 healthy。生产同源入口接收并落库 GET/POST/PUT/PATCH/DELETE/OPTIONS/HEAD/TRACE/CONNECT/BREW/custom/M!X；静态页面与 CSP、来源 IP、防伪造头、脱敏、SSRF 拒绝、删除和退出烟雾测试通过。最终正式镜像上 Playwright 4/4 通过（9.1s）。

环境处理：仅 8 GB 磁盘且 Docker 使用 vfs，构建阶段改用轻量 Alpine、临时源码挂载和 Go 编译临时 tmpfs，顺序构建并清理本任务构建缓存；不删除 MySQL 数据卷。容器 wget 本机健康检查明确关闭环境代理，避免被代理误报 403。Go 公网入口对回环健康检查正常。

初次 MVP 完成时，代码留在 work 分支。v1.0 收尾已备份并逐文件校验工作区，基于最新 origin/main 建立 feature/hooklab-v1，保留原有 Git 提交；不合并或覆盖 main。实际提交与推送结果以 Git 历史为准。


## v1.0 最终收尾

本轮范围仅限三个已确认问题、交付文件整理、测试和 Git 提交推送，不新增功能或依赖。

- Endpoint PATCH 使用可选布尔字段与 SQL COALESCE：缺少 active 保留原值，显式 false 停用；响应返回数据库实际状态。回归覆盖活动/暂停状态下的改名及显式停用/恢复。
- 捕获 Header 在脱敏后合入真实 Request.Host（包含端口），覆盖 Header map 中可能出现的同名值；沿用现有详情与 Headers 查看器展示。回归验证详情和 MySQL 持久化，并检查敏感头仍脱敏。
- Method 筛选对参数做二进制比较，与已有 MySQL 不区分大小写的默认排序规则兼容；不用重写已部署迁移。回归分别验证 BREW 与 brew。
- 新增三个独立 Go 回归测试，并在已有 Playwright 流程中验证 Host 展示。构建产物 *.tsbuildinfo 等已加入忽略规则；环境变量示例仅含空凭证占位。

自动清理、Token 轮换、存储配额等留待未来版本，本轮交付后不继续扩展。

v1.0 最终验证：真实 MySQL 上 Go/race 18 个主测试（含子测试共 47 项）全部通过，测试用例无跳过；go vet、TypeScript、Vite 生产构建通过；新镜像上 Playwright 4/4 通过（含 Host 展示）。两个正式 Dockerfile 均构建成功，默认 Compose 的数据库、API、Web 全部 healthy，公开入口三项修复复验通过。环境磁盘限制通过清理构建缓存和释放旧容器 vfs 副本解决，原 MySQL 数据卷和业务表行数均保留并校验一致。
