# 外部集成

**分析日期：** 2026-07-01

new-api 是一个 LLM API 网关/代理。其主要集成为上游 AI 模型提供商（通过按提供商划分的适配器中继），以及支付、认证、邮件、监控与分析服务。

## API 与外部服务

### LLM / AI 模型提供商（中继适配器）

每个上游提供商实现 `channel.Adaptor` 接口（`relay/channel/adapter.go` 第 15 行）或用于异步视频/图像/音频任务的 `channel.TaskAdaptor`（第 34 行）。适配器由 `relay/relay_adaptor.go` `GetAdaptor()`（第 54 行）按 `apiType` 解析。渠道类型 ID 与默认 base URL 在 `constant/channel.go` 中定义。凭证按渠道存储于 `channels` 数据库表（管理员配置），不存于环境变量。

**对话 / 补全 / 嵌入 / 图像 / 重排 / 音频提供商：**

| 提供商 | 渠道类型 | 适配器 | 默认 Base URL |
|----------|-------------|---------|------------------|
| OpenAI | `ChannelTypeOpenAI` (1) | `relay/channel/openai/adaptor.go` | `https://api.openai.com` |
| Azure OpenAI | `ChannelTypeAzure` (3) | `relay/channel/openai/adaptor.go`（Azure 模式） | （按部署） |
| Anthropic Claude | `ChannelTypeAnthropic` (14) | `relay/channel/claude/adaptor.go` | `https://api.anthropic.com` |
| Google Gemini | `ChannelTypeGemini` (24) | `relay/channel/gemini/adaptor.go` | `https://generativelanguage.googleapis.com` |
| Google PaLM | `ChannelTypePaLM` (11) | `relay/channel/palm/adaptor.go` | — |
| Google Vertex AI | `ChannelTypeVertexAi` (41) | `relay/channel/vertex/adaptor.go`（服务账号 JWT 认证，`relay/channel/vertex/service_account.go`） | — |
| AWS Bedrock | `ChannelTypeAws` (33) | `relay/channel/aws/adaptor.go`（AWS SDK v2 `bedrockruntime`；API-key 或 AK/SK 模式） | — |
| Cohere | `ChannelTypeCohere` (34) | `relay/channel/cohere/adaptor.go` | `https://api.cohere.ai` |
| Mistral | `ChannelTypeMistral` (42) | `relay/channel/mistral/adaptor.go` | `https://api.mistral.ai` |
| DeepSeek | `ChannelTypeDeepSeek` (43) | `relay/channel/deepseek/adaptor.go` | `https://api.deepseek.com` |
| xAI (Grok) | `ChannelTypeXai` (48) | `relay/channel/xai/adaptor.go` | `https://api.x.ai` |
| Cloudflare Workers AI | `ChannelCloudflare` (39) | `relay/channel/cloudflare/adaptor.go` | `https://api.cloudflare.com` |
| Ollama | `ChannelTypeOllama` (4) | `relay/channel/ollama/adaptor.go` | `http://localhost:11434` |
| OpenRouter | `ChannelTypeOpenRouter` (20) | `relay/channel/openai/adaptor.go` | `https://openrouter.ai/api` |
| Perplexity | `ChannelTypePerplexity` (27) | `relay/channel/perplexity/adaptor.go` | `https://api.perplexity.ai` |
| Replicate | `ChannelTypeReplicate` (56) | `relay/channel/replicate/adaptor.go` | `https://api.replicate.com` |
| SiliconFlow | `ChannelTypeSiliconFlow` (40) | `relay/channel/siliconflow/adaptor.go` | `https://api.siliconflow.cn` |
| MiniMax | `ChannelTypeMiniMax` (35) | `relay/channel/minimax/adaptor.go` | `https://api.minimax.chat` |
| Moonshot | `ChannelTypeMoonshot` (25) | `relay/channel/moonshot/adaptor.go` | `https://api.moonshot.cn` |
| Dify | `ChannelTypeDify` (37) | `relay/channel/dify/adaptor.go` | `https://api.dify.ai` |
| Jina | `ChannelTypeJina` (38) | `relay/channel/jina/adaptor.go` | `https://api.jina.ai` |
| Coze | `ChannelTypeCoze` (49) | `relay/channel/coze/adaptor.go` | `https://api.coze.cn` |
| 腾讯混元 | `ChannelTypeTencent` (23) | `relay/channel/tencent/adaptor.go` | `https://hunyuan.tencentcloudapi.com` |
| 百度（文心一言） | `ChannelTypeBaidu` (15) / `BaiduV2` (46) | `relay/channel/baidu/adaptor.go`、`baidu_v2/adaptor.go` | `https://aip.baidubce.com` / `https://qianfan.baidubce.com` |
| 阿里（通义千问 / DashScope） | `ChannelTypeAli` (17) | `relay/channel/ali/adaptor.go` | `https://dashscope.aliyuncs.com` |
| 讯飞（星火） | `ChannelTypeXunfei` (18) | `relay/channel/xunfei/adaptor.go` | — |
| 智谱（GLM） | `ChannelTypeZhipu` (16) / `ZhipuV4` (26) | `relay/channel/zhipu/adaptor.go`、`zhipu_4v/adaptor.go` | `https://open.bigmodel.cn` |
| 火山引擎（豆包） | `ChannelTypeVolcEngine` (45) | `relay/channel/volcengine/adaptor.go` | `https://ark.cn-beijing.volces.com` |
| Xinference | `ChannelTypeXinference` (47) | `relay/channel/openai/adaptor.go` | — |
| LingYiWanWu | `ChannelTypeLingYiWanWu` (31) | `relay/channel/openai/adaptor.go` | `https://api.lingyiwanwu.com` |
| MokaAI | `ChannelTypeMokaAI` (44) | `relay/channel/mokaai/adaptor.go` | `https://api.moka.ai` |
| Submodel | `ChannelTypeSubmodel` (53) | `relay/channel/submodel/adaptor.go` | `https://llm.submodel.ai` |
| ChatGPT 订阅（Codex） | `ChannelTypeCodex` (57) | `relay/channel/codex/adaptor.go`（OAuth refresh-token 流程；`service/codex_oauth.go`） | `https://chatgpt.com` |
| 高级自定义 | `ChannelTypeAdvancedCustom` (58) | `relay/channel/advancedcustom/adaptor.go` | （运营者自定义） |

中继入口路由：`router/relay-router.go` —— `/v1/chat/completions`、`/v1/messages`（Claude）、`/v1/responses`、`/v1/embeddings`、`/v1/audio/*`、`/v1/images/*`、`/v1/rerank`、`/v1beta/models/*`（Gemini）、`/v1/realtime`（WebSocket）。

**异步任务提供商（视频 / 图像 / 音乐）—— `channel.TaskAdaptor`：**

| 提供商 | 平台 | 适配器 |
|----------|----------|---------|
| Midjourney | `:mode/mj/*` 路由（`router/relay-router.go` `registerMjRouterGroup`） | `relay/mjproxy_handler.go` |
| Suno（音乐） | `/suno/submit/*` | `relay/channel/task/suno/adaptor.go` |
| Kling（视频） | `/kling/v1/videos/*`（`router/video-router.go`） | `relay/channel/task/kling/adaptor.go` |
| Jimeng（图像/视频） | `/jimeng` | `relay/channel/task/jimeng/adaptor.go` |
| Vertex AI（视频） | `/v1/video/generations` | `relay/channel/task/vertex/adaptor.go` |
| Vidu（视频） | `/v1/video/generations` | `relay/channel/task/vidu/adaptor.go` |
| 豆包视频 / 火山引擎 | `/v1/video/generations` | `relay/channel/task/doubao/adaptor.go` |
| Sora（OpenAI 视频） | `/v1/videos` | `relay/channel/task/sora/adaptor.go` |
| Gemini（图像生成） | task 适配器 | `relay/channel/task/gemini/adaptor.go` |
| 海螺（MiniMax 视频） | task 适配器 | `relay/channel/task/hailuo/adaptor.go` |
| 阿里（万象图像） | task 适配器 | `relay/channel/task/ali/adaptor.go` |

认证：按渠道的 API key / OAuth token 存于 DB `channels` 表；通过各适配器的 `SetupRequestHeader` 传递。Codex 适配器自动刷新 OpenAI OAuth token（`service/codex_credential_refresh_task.go`，在 `main.go` 第 119 行启动）。

### 支付提供商

| 提供商 | SDK / 客户端 | 初始化 | Webhook 路由 | Controller |
|----------|--------------|------|---------------|------------|
| Stripe | `github.com/stripe/stripe-go/v81` + `webhook` | `setting/payment_stripe.go`（`StripeApiSecret`、`StripeWebhookSecret`、`StripePriceId`） | `POST /api/stripe/webhook`（`router/api-router.go` 第 57 行） | `controller/topup_stripe.go`、`controller/subscription_payment_stripe.go` |
| EPay | `github.com/Calcium-Ion/go-epay v0.0.4` | 管理员配置的商户密钥 | `GET/POST /api/user/epay/notify`、`/api/subscription/epay/notify`（第 76-77、180-181 行） | `controller/topup.go`（`RequestEpay`）、`controller/subscription_payment_epay.go` |
| Creem | 自定义 HTTP + HMAC-SHA256（`controller/topup_creem.go` `verifyCreemSignature`） | `setting/payment_creem.go`（`CreemApiKey`、`CreemWebhookSecret`、`CreemTestMode`） | `POST /api/creem/webhook`（第 58 行） | `controller/topup_creem.go`、`controller/subscription_payment_creem.go` |
| Waffo（Global） | `github.com/waffo-com/waffo-go v1.3.2` | `setting/payment_waffo.go` | `POST /api/waffo/webhook`（第 59 行） | `controller/topup_waffo.go` |
| Waffo Pancake | `github.com/waffo-com/waffo-pancake-sdk-go v0.3.1` | `setting/payment_waffo_pancake.go`（`WaffoPancakeMerchantID`、`WaffoPancakePrivateKey`、`WaffoPancakeStoreID`、`WaffoPancakeProductID`） | `POST /api/waffo-pancake/webhook/:env`（第 62 行；`:env` 区分 test/prod） | `controller/topup_waffo_pancake.go`、`controller/subscription_payment_waffo_pancake.go`、`service/waffo_pancake.go` |

支付端点（已认证）：`/api/user/stripe/pay`、`/api/user/pay`（epay）、`/api/user/creem/pay`、`/api/user/waffo/pay`、`/api/user/waffo-pancake/pay`（`router/api-router.go` 第 99-107 行）；订阅对应项位于 `/api/subscription/*`（第 157-161 行）。

### OAuth / 社交登录提供商

所有标准提供商实现 `oauth.Provider` 接口（`oauth/provider.go`）并在 `init()` 中自注册。统一路由：`GET /api/oauth/:provider` → `controller.HandleOAuth`（`router/api-router.go` 第 54 行）。

| 提供商 | Slug | 实现 | 配置 |
|----------|------|----------------|----------|
| GitHub | `github` | `oauth/github.go`（token 交换 `github.com/login/oauth/access_token`，用户信息 `api.github.com/user`） | `common.GitHubClientId/Secret` |
| Discord | `discord` | `oauth/discord.go` | `setting/system_setting/discord.go`（`GetDiscordSettings()`） |
| OIDC（通用） | `oidc` | `oauth/oidc.go`（基于发现） | `setting/system_setting/oidc.go` |
| LinuxDO | `linuxdo` | `oauth/linuxdo.go` | `common.LinuxDOClientId/Secret` |
| 自定义（DB 驱动） | 任意 slug | `oauth/generic.go` + `oauth/registry.go` `LoadCustomProviders()`（从 `custom_oauth_providers` 表加载，`main.go` 第 336 行） | `model/custom_oauth_provider.go` |
| 微信 | `wechat` | `controller/wechat.go`（通过 `WeChatServerAddress` + `WeChatServerToken` 代理） | `common.WeChatServerAddress/Token` |
| Telegram | `telegram` | `controller/telegram.go`（针对 `TelegramBotToken` 的 HMAC 校验） | `common.TelegramBotToken` |

绑定：`/api/oauth/wechat/bind`、`/api/oauth/telegram/bind`、`/api/oauth/email/bind`（第 50-52 行）。管理员自定义提供商 CRUD：`/api/custom-oauth-provider/*`（第 202-211 行）。

### 其他外部服务

- **Cloudflare Turnstile** —— 注册/登录/重置/签到的人机验证。中间件 `middleware/turnstile-check.go`；密钥 `common.TurnstileSiteKey/SecretKey`。
- **io.net** —— GPU 集群/模型部署管理。客户端 `pkg/ionet`；API key 来自 `OptionMap["model_deployment.ionet.api_key"]`（`controller/deployment.go` 第 19 行）。路由 `/api/deployments/*`。
- **Uptime Kuma** —— 外部状态页抓取。`controller/uptime_kuma.go` 轮询 `/api/status-page/` 与 `/api/status-page/heartbeat/`（常量第 22-23 行）。
- **Pyroscope** —— 持续 Go 性能剖析（可选）。`common/pyro.go` `StartPyroScope()`（环境变量 `PYROSCOPE_URL`）。
- **pprof** —— `ENABLE_PPROF=true` 时在 `:8005` 上的 Go 运行时性能剖析（`main.go` 第 153-159 行）。
- **Prometheus** —— 客户端库经由 pyroscope 间接引入（`go.mod` 第 143-146 行）。
- **OpenAI Codex OAuth** —— 对 `https://auth.openai.com/oauth/token` 刷新 token，客户端 ID 为 `app_EMoamEEZ73f0CkXaXp7hrann`（`service/codex_oauth.go` 第 17-19 行）。
- **Umami Analytics** —— 当 `UMAMI_WEBSITE_ID` 设置时注入脚本到嵌入的 `index.html`（`main.go` `InjectUmamiAnalytics` 第 218 行）。
- **Google Analytics (GA4)** —— 当 `GOOGLE_ANALYTICS_ID` 设置时注入 gtag（`main.go` `InjectGoogleAnalytics` 第 239 行）。

## 数据存储

**数据库：**
- 主库：SQLite（默认，文件 `one-api.db`）、MySQL 或 PostgreSQL —— 由 `SQL_DSN` scheme 在 `model/main.go` `chooseDB`（第 127 行）选择。
  - 客户端：GORM（`gorm.io/gorm v1.25.2`）+ 驱动包（`mysql`、`postgres`、`glebarez/sqlite`、`clickhouse`）。
  - 连接：环境变量 `SQL_DSN`。连接池：`SQL_MAX_IDLE_CONNS`（100）、`SQL_MAX_OPEN_CONNS`（1000）、`SQL_MAX_LIFETIME`（60s）—— `model/main.go` 第 203-205 行。
  - 迁移：`migrateDB()`（`model/main.go` 第 263 行）自动迁移约 30 个模型，包括 `Channel`、`Token`、`User`、`Log`、`Task`、`SubscriptionPlan`、`CasbinRule`、`AuthzRole`、`PerfMetric`、`SystemInstance`、`SystemTask`。
- 日志库（可选、独立）：`LOG_SQL_DSN` —— 支持 MySQL、PostgreSQL 或 **ClickHouse**（`model/main.go` `InitLogDB` 第 222 行；ClickHouse DDL 见 `clickHouseLogCreateTableSQL` 第 429 行）。ClickHouse TTL 通过 `LOG_SQL_CLICKHOUSE_TTL_DAYS`。

**文件存储：**
- 仅本地文件系统（无 S3/OSS）。磁盘缓存位于配置的缓存目录（`common/disk_cache.go`、`common/disk_cache_config.go`）。容器 `/data` 卷（`docker-compose.yml` 第 26 行）。

**缓存：**
- Redis（可选）—— `REDIS_CONN_STRING`；客户端 `github.com/go-redis/redis/v8`（`common/redis.go`）。连接池大小 `REDIS_POOL_SIZE`（10）。启用渠道/选项缓存 + 限流。
- 内存缓存 —— `MEMORY_CACHE_ENABLED=true`（启用 Redis 时自动开启；`main.go` 第 75-99 行）。`SyncFrequency`（默认 60s）刷新。
- 磁盘缓存 —— `common/disk_cache.go`，用于大响应体。
- 异步 token 缓存 —— `bytedance/gopkg/cache/asynccache`（Vertex AI token，`relay/channel/vertex/service_account.go` 第 31 行；30 分钟过期）。

## 认证与身份

**认证提供商：** 自建（内置）。

**实现：**
- Web session —— `gin-contrib/sessions` cookie 存储，密钥来自 `SESSION_SECRET`（`main.go` 第 184-192 行）。30 天 MaxAge，HttpOnly，SameSite=Strict。
- API access token —— `Authorization` 头由 `model.ValidateAccessToken` 校验（`middleware/auth.go` `authHelper` 第 37 行）。
- API key（中继）—— `TokenAuth()` 中间件针对 `tokens` 表校验 `sk-...` 密钥（`middleware/auth.go`）。
- 角色 —— Guest(0) / Common(1) / Admin(10) / Root(100)（`common/constants.go` 第 200-205 行）。由 `UserAuth`、`AdminAuth`、`RootAuth` 中间件执行。
- 鉴权 —— Casbin RBAC（`service/authz/`、`model/casbin_rule.go`、`model/authz_role.go`）；策略重载在节点间同步（`main.go` 第 105 行）。
- 2FA —— 通过 `github.com/pquerna/otp` 的 TOTP（`model/twofa.go`、`controller/twofa.go`）。
- Passkey/WebAuthn —— `github.com/go-webauthn/webauthn`（`controller/passkey.go`、`model/passkey.go`）。
- OAuth —— 见上方提供商；绑定存储于 `user_oauth_bindings` 表。

## 监控与可观测性

**错误追踪：** 无（无 Sentry/bugsnag）。错误进入文件日志（`logger/logger.go`）+ `common.SysError`。

**日志：**
- 自定义文件日志位于 `./logs`（`logger/logger.go`、`common/init.go` `LogDir` 标志默认 `./logs`）。
- 请求/消费日志持久化到 DB `logs` 表（或 ClickHouse）。
- 可选 `ERROR_LOG_ENABLED` 标志。

**性能剖析：**
- Pyroscope（持续）—— `common/pyro.go`。
- pprof（按需）—— `ENABLE_PPROF=true` 时在 `:8005` 上的 `net/http/pprof`。
- `github.com/shirou/gopsutil` —— 系统指标（`common/system_monitor.go`）。
- 内置性能统计 API —— `/api/performance/*`（`controller/performance.go`），性能指标 `/api/perf-metrics/*`（`controller/perf_metrics.go`），`pkg/perf_metrics`。

## CI/CD 与部署

**托管：**
- 容器镜像 `calciumion/new-api:latest`（Dockerfile）。
- Electron 桌面包（mac dmg/zip、win nsis/portable、linux AppImage/deb）通过 `electron-builder`（`electron/package.json`）。
- systemd 服务（`new-api.service`）。
- 裸二进制通过 `go build`（`Dockerfile` 第 39 行）。

**CI 流水线：** 未检测到（扫描范围内无 `.github/workflows`；构建由 `make`/Docker 驱动）。

## 环境配置

**所需环境变量（生产）：**
- `SQL_DSN` —— 主数据库（多节点推荐 PostgreSQL；`docker-compose.yml` 第 29 行）。
- `SESSION_SECRET` —— 多节点必需；若保持默认 `random_string` 则 panic（`common/init.go` 第 51 行）。
- `CRYPTO_SECRET` —— token 加密（默认为 `SESSION_SECRET`）。

**可选环境变量：**
- `LOG_SQL_DSN`、`LOG_SQL_CLICKHOUSE_TTL_DAYS` —— 独立日志库 / ClickHouse TTL。
- `REDIS_CONN_STRING`、`REDIS_POOL_SIZE` —— 缓存与限流。
- `NODE_TYPE=slave`、`NODE_NAME` —— 多实例身份。
- `PORT`、`GIN_MODE=debug`、`DEBUG=true`。
- `SQLITE_PATH`、`SYNC_FREQUENCY`、`BATCH_UPDATE_ENABLED`、`BATCH_UPDATE_INTERVAL`。
- `RELAY_TIMEOUT`、`RELAY_IDLE_CONN_TIMEOUT`、`RELAY_MAX_IDLE_CONNS`、`RELAY_MAX_IDLE_CONNS_PER_HOST`。
- `STREAMING_TIMEOUT`、`MAX_FILE_DOWNLOAD_MB`、`MAX_REQUEST_BODY_MB`、`FORCE_STREAM_OPTION`、`GET_MEDIA_TOKEN`。
- `TLS_INSECURE_SKIP_VERIFY`、`HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`（中继客户端遵循代理环境变量；`service/http_client.go` 第 42 行）。
- `GLOBAL_API_RATE_LIMIT*`、`GLOBAL_WEB_RATE_LIMIT*`、`CRITICAL_RATE_LIMIT*`、`SEARCH_RATE_LIMIT*`。
- `ENABLE_PPROF`、`PYROSCOPE_URL`、`PYROSCOPE_APP_NAME`、`PYROSCOPE_BASIC_AUTH_*`。
- `GOOGLE_ANALYTICS_ID`、`UMAMI_WEBSITE_ID`、`UMAMI_SCRIPT_URL`。
- `CHANNEL_UPDATE_FREQUENCY`、`GENERATE_DEFAULT_TOKEN`、`ERROR_LOG_ENABLED`。
- `GEMINI_SAFETY_SETTING`、`COHERE_SAFETY_SETTING`、`AZURE_DEFAULT_API_VERSION`。
- `TRUSTED_REDIRECT_DOMAINS` —— 支付成功/取消跳转 URL 的白名单（`common/init.go` 第 174 行）。
- `FRONTEND_BASE_URL` —— 单独提供前端并跳转（`router/main.go` 第 20 行）。

完整环境变量解析：`common/init.go` `InitEnv()`（第 31 行）+ `initConstantEnv()`（第 134 行）。SMTP/OAuth/支付提供商凭证由管理员在运行时通过基于 DB 的 options 表配置（`model/option.go`），不通过环境变量。

**密钥位置：**
- `.env` 文件（由 `godotenv` 加载；仅记录存在 —— 从不被工具读取）。
- DB `options` 表（`model/option.go`）—— 管理员通过 UI 录入；包含 `Secret`/`Token` 的键在 `GetOptions` 中被掩码（`common/constants.go` 第 73 行注释）。
- 渠道 API key —— `channels` DB 表。

## Webhook 与回调

**入站（本服务器接收）：**
- `POST /api/stripe/webhook` —— Stripe 结账事件（`router/api-router.go` 第 57 行；签名校验在 `controller/topup_stripe.go`）。
- `POST /api/creem/webhook` —— Creem 支付事件（第 58 行；HMAC-SHA256 校验 `controller/topup_creem.go`）。
- `POST /api/waffo/webhook` —— Waffo 传统网关（第 59 行）。
- `POST /api/waffo-pancake/webhook/:env` —— Waffo Pancake 结账事件（第 62 行；`:env` = test|prod）。
- `GET/POST /api/user/epay/notify` —— EPay 异步通知（第 76-77 行）。
- `GET/POST /api/subscription/epay/notify`、`/api/subscription/epay/return` —— EPay 订阅回调（第 180-183 行）。
- `GET /api/oauth/:provider` —— OAuth 提供商带 `code` 回跳（第 54 行）。

**出站（本服务器发送）：**
- 出站 webhook 通知到管理员配置的 URL —— `service/webhook.go` `SendWebhookNotify`（HMAC-SHA256 签名；第 35 行）。触发于配额超限、渠道更新、渠道测试（`dto/notify.go`）。
- 邮件通知通过 SMTP —— `common/email.go` `SendEmail`；`service/user_notify.go` `NotifyUser` / `NotifyRootUser` / `NotifyUpstreamModelUpdateWatchers`。
- 中继请求到所有上游 LLM 提供商（见上方提供商表）—— 通过共享 HTTP 客户端 `service/http_client.go` `GetHttpClient()`，含 SSRF 防护（`common/ssrf_protection.go`、`common/url_validator.go`）。
- OAuth token 交换与用户信息抓取到 GitHub、Discord、OIDC、LinuxDO、微信代理、Telegram。
- Codex OAuth token 刷新到 `https://auth.openai.com/oauth/token`（`service/codex_oauth.go`）。
- Vertex AI 服务账号 JWT 交换到 Google OAuth2 token 端点（`relay/channel/vertex/service_account.go`）。
- io.net 部署 API 调用（`controller/deployment.go` 通过 `pkg/ionet.Client`）。
- Uptime Kuma 状态轮询（`controller/uptime_kuma.go`）。
- Pyroscope 性能剖析数据上传（`common/pyro.go`）。

---

*集成审计：2026-07-01*
