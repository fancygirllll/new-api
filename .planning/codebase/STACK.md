# 技术栈

**分析日期：** 2026-07-01

## 编程语言

**主语言：**
- Go 1.25.1（在 `go.mod` 中声明；Docker 构建使用 `golang:1.26.1-alpine`，见 `Dockerfile` 第 23 行）—— 后端 API 网关、中继引擎、计费、定时任务、支付回调。模块路径：`github.com/QuantumNous/new-api`（`go.mod` 第 1 行）。

**副语言：**
- TypeScript —— 前端应用（`web/default/`、`web/classic/`）。配置文件：`web/default/tsconfig.json`、`web/default/tsconfig.app.json`、`web/default/tsconfig.node.json`。
- JavaScript —— Electron 桌面壳（`electron/main.js`、`electron/preload.js`）。
- YAML —— i18n 语言包（`i18n/locales/zh-CN.yaml`、`zh-TW.yaml`、`en.yaml`）。
- Dockerfile / docker-compose —— 容器构建与编排。

## 运行时

**环境：**
- Go 运行时（单一静态二进制；`Dockerfile` 第 24 行 `CGO_ENABLED=0` 构建；第 29 行通过 `GOEXPERIMENT=greenteagc` 启用实验性 `greenteagc` GC）。
- 生产容器基础镜像：`debian:bookworm-slim`（`Dockerfile` 第 41 行）。
- 仅在前端构建阶段使用可选的 Node.js/Bun 工具链（`oven/bun:1` 构建阶段，`Dockerfile` 第 1 行与第 12 行）。
- 桌面发行版使用 Electron 39.8.5 运行时（`electron/package.json` 第 28 行）。

**包管理器：**
- Go modules —— `go.mod` / `go.sum`（已含 lockfile）。
- Bun —— `web/bun.lock`（前端 monorepo workspace，`web/package.json` 第 3-5 行）。
- npm —— `electron/package-lock.json`（仅桌面应用）。

## 框架

**核心（后端）：**
- Gin `v1.9.1` —— HTTP 框架。在 `main.go` 第 167-203 行初始化；路由通过 `router/main.go` → `SetApiRouter`、`SetRelayRouter`、`SetVideoRouter`、`SetWebRouter` 注册。
- GORM `v1.25.2` —— ORM。在 `model/main.go` `InitDB()`（第 181 行）初始化；驱动：mysql `v1.4.3`、postgres `v1.5.2`、clickhouse `v0.6.0`、sqlite 通过 `glebarez/sqlite v1.9.0`。
- Casbin `v2.135.0` —— RBAC 鉴权。在 `main.go` 第 291 行（`authz.Init`）初始化；策略同步在 `main.go` 第 105 行启动。

**核心（默认前端 —— `web/default`）：**
- React `^19.2.6`（`web/default/package.json` 第 60 行）。
- TanStack Router `^1.170.8` + Router 插件（基于文件的路由位于 `web/default/src/routes/`）。
- TanStack Query `^5.100.14`、TanStack Table `^8.21.3`、TanStack Virtual `^3.13.25`。
- Tailwind CSS `^4.3.0` + `@base-ui/react` + shadcn `^4.8.0`。
- Rsbuild `^2.0.7`（构建/开发服务器；配置：`web/default/rsbuild.config.ts`）。
- Vercel AI SDK `ai ^6.0.191` + `sse.js`（流式对话）。

**核心（Classic 前端 —— `web/classic`）：**
- React `^19.2.6`（`web/classic/package.json` 第 26 行）。
- Semi UI `@douyinfe/semi-ui ^2.69.1`（组件库）。
- React Router DOM `^6.3.0`。
- Rsbuild `^2.0.7`。

**桌面端：**
- Electron `39.8.5` + `electron-builder ^26.7.0`（`electron/package.json` 第 28-29 行）。

**测试：**
- `github.com/stretchr/testify v1.11.1` —— Go 断言库（例如 `model/usedata_flow_test.go`、`relay/channel/openai/chat_via_responses_test.go`）。

**构建/开发：**
- Bun —— 前端安装与构建（`makefile` `build-frontend` target）。
- `make` —— 编排开发栈（`makefile`：`dev`、`dev-api`、`dev-web`）。
- Docker Compose —— `docker-compose.yml`（类生产）、`docker-compose.dev.yml`（开发数据库服务）。
- `go:embed` —— 前端 `dist/` 嵌入二进制（`main.go` 第 39-49 行）。

## 关键依赖

**关键（后端）：**
- `github.com/gin-gonic/gin v1.9.1` —— HTTP 服务器与中间件管道。
- `gorm.io/gorm v1.25.2` —— 所有数据库访问（`model/main.go`）。
- `github.com/go-redis/redis/v8 v8.11.5` —— 缓存与分布式限流（`common/redis.go`）。
- `github.com/casbin/casbin/v2 v2.135.0` —— 角色/权限执行（`service/authz/`）。
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.50.4` —— AWS Bedrock 中继（`relay/channel/aws/adaptor.go`）。
- `github.com/stripe/stripe-go/v81 v81.4.0` —— Stripe 结账与回调（`controller/topup_stripe.go`）。
- `github.com/Calcium-Ion/go-epay v0.0.4` —— EPay 网关（`controller/topup.go`）。
- `github.com/waffo-com/waffo-pancake-sdk-go v0.3.1` —— Waffo Pancake 托管结账（`service/waffo_pancake.go`）。
- `github.com/go-webauthn/webauthn v0.14.0` —— Passkey/WebAuthn 登录（`controller/passkey.go`）。
- `github.com/pquerna/otp v1.5.0` —— TOTP 2FA（`model/twofa.go`）。
- `github.com/golang-jwt/jwt/v5 v5.3.0` —— Vertex AI 服务账号 JWT（`relay/channel/vertex/service_account.go`）。
- `github.com/gorilla/websocket v1.5.0` —— OpenAI Realtime API 中继（`relay/websocket.go`）。
- `github.com/tidwall/gjson v1.18.0` + `sjson v1.2.5` —— 上游 payload 的 JSON 路径读写。
- `github.com/tiktoken-go/tokenizer v0.6.2` —— 用于计费的 token 计数（`service/tokenizer.go`）。
- `github.com/shopspring/decimal v1.4.0` —— 金额/配额运算（`controller/topup_waffo_pancake.go`、`service/billing.go`）。
- `github.com/expr-lang/expr v1.17.8` —— 计费表达式求值（`relay/helper/billing_expr_request.go`）。
- `github.com/anknown/ahocorasick v0.0.0-...` —— 敏感词匹配（`service/sensitive.go`）。
- `github.com/bytedance/gopkg v0.1.3` —— 协程池（`gopool`），用于各类后台任务。
- `github.com/nicksnyder/go-i18n/v2 v2.6.1` —— 服务端本地化（`i18n/i18n.go`）。
- `github.com/joho/godotenv v1.5.1` —— 加载 `.env`（`main.go` 第 266 行）。
- `github.com/grafana/pyroscope-go v1.2.7` —— 持续性能剖析（`common/pyro.go`）。
- `github.com/samber/lo v1.52.0` + `samber/hot v0.11.0` —— 函数式工具与 hot 缓存。

**音频解码（用于多模态 token 计数）：**
- `github.com/go-audio/wav v1.1.0`、`go-audio/aiff v1.1.0`、`jfreymuth/oggvorbis v1.0.5`、`mewkiz/flac v1.0.13`、`tcolgate/mp3`、`abema/go-mp4 v1.4.1`、`yapingcat/gomedia`。

**基础设施（后端）：**
- `github.com/gin-contrib/cors v1.7.2`、`gzip v0.0.6`、`sessions v0.0.5`（cookie 存储）、`static v0.0.1`。
- `github.com/go-playground/validator/v10 v10.20.0` —— 请求校验。
- `github.com/google/uuid v1.6.0`。
- `github.com/thanhpk/randstr v1.0.6` —— 交易号/验证码。
- `github.com/jinzhu/copier v0.4.0` —— 结构体拷贝。
- `github.com/Azure/go-ntlmssp v0.1.1` —— NTLM SMTP 认证（`common/email_ntlm_auth.go`）。
- `github.com/shirou/gopsutil v3.21.11+incompatible` —— 系统监控（`common/system_monitor.go`）。
- Prometheus 客户端库（间接依赖，经由 pyroscope）—— 指标。

**关键（默认前端）：**
- `@tanstack/react-router` `^1.170.8` —— 基于文件的路由（`web/default/src/routes/`）。
- `@tanstack/react-query` `^5.100.14` —— 服务端状态。
- `tailwindcss` `^4.3.0` + `tw-animate-css` + `tailwind-merge` —— 样式。
- `@base-ui/react` `^1.5.0` + `shadcn` `^4.8.0` —— 无样式组件原语。
- `@visactor/vchart` `^2.0.22` —— 图表（`dashboard`、`usage-logs`）。
- `@lobehub/icons` `^5.10.0` —— 模型/厂商图标集。
- `ai` `^6.0.191` + `sse.js` `^2.8.0` —— 流式对话客户端（`playground`）。
- `react-hook-form` `^7.76.1` + `@hookform/resolvers` + `zod` `^4.4.3` —— 表单。
- `zustand` `^5.0.13` —— 客户端 store（`web/default/src/stores/`）。
- `i18next` `^26.2.0` + `react-i18next` `^17.0.8` —— 本地化。
- `@codemirror/*` —— markdown 编辑器；`marked`、`shiki`、`katex` —— 渲染。
- `axios`（目录） —— API 客户端。

## 配置

**环境：**
- `.env` 文件通过 `main.go` 第 266 行的 `godotenv.Load(".env")` 加载（仅记录存在；内容从不被工具读取）。
- 进程环境变量在 `common/init.go` `InitEnv()`（第 31 行）与 `common/constants.go` 中解析。
- 运行时选项镜像到基于 DB 的 `OptionMap`（`model/option.go` `InitOptionMap` 第 30 行）—— 仅 root 可通过 `/api/option` 编辑。

**所需关键配置：**
- `SQL_DSN` —— 主数据库连接（MySQL/PostgreSQL/SQLite；`model/main.go` `chooseDB` 第 127 行）。
- `LOG_SQL_DSN` —— 可选独立日志库（支持 ClickHouse；`model/main.go` 第 222 行）。
- `REDIS_CONN_STRING` —— 可选 Redis（`common/redis.go` 第 25 行）。
- `SESSION_SECRET` —— cookie 存储密钥（`common/init.go` 第 49 行；未设置时回退为随机值并告警）。
- `CRYPTO_SECRET` —— token 加密密钥（默认为 `Session_SECRET`；`common/init.go` 第 59 行）。
- `NODE_TYPE=slave` —— 标记非 master 节点（跳过迁移与 cron；`common/init.go` 第 84 行）。
- `NODE_NAME` —— 节点身份，用于审计日志（`common/constants.go` 第 168 行）。
- `SQLITE_PATH` —— SQLite 文件位置（默认 `one-api.db?_busy_timeout=30000`；`common/database.go` 第 44 行）。

**构建配置文件：**
- `Dockerfile`、`Dockerfile.dev` —— 多阶段容器构建。
- `docker-compose.yml`、`docker-compose.dev.yml` —— 编排。
- `makefile` —— 开发/构建目标。
- `new-api.service` —— systemd 单元。
- `web/default/rsbuild.config.ts` —— 前端打包器配置（开发代理 `/api`、`/mj`、`/pg` → `localhost:3000`）。
- `web/default/tsconfig*.json`、`web/default/.oxlintrc.json` —— TS/lint 配置。
- `electron/package.json` `build` 块 —— electron-builder 目标（mac/win/linux）。

## 平台要求

**开发：**
- Go 1.25+ 工具链（`go.mod` 第 4 行）。
- Bun（前端；`makefile` 使用 `bun install`）。
- Docker（可选，用于 `make dev-api` 的数据库服务）。
- 端口：后端 `3000`（默认，`common/init.go` 第 18 行），默认前端开发 `5173`，Classic 前端开发 `5174`（`makefile` 第 4-5 行）。

**生产：**
- 单一静态二进制（`new-api`）、容器镜像 `calciumion/new-api:latest`，或 Electron 桌面包（`electron-builder` mac/win/linux）。
- 默认监听端口 `3000`（`Dockerfile` 第 50 行 `EXPOSE 3000`）。
- 数据卷 `/data`（`Dockerfile` 第 51 行）。
- 外部依赖：关系型数据库（MySQL 8+ / PostgreSQL 15+ / SQLite），可选 Redis，可选 ClickHouse 用于日志。
- 多实例通过 `NODE_TYPE` master/slave + 共享 `SESSION_SECRET` 支持（`docker-compose.yml` 注释第 41-42 行）。

---

*技术栈分析：2026-07-01*
