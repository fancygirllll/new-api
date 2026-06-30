# External Integrations

**Analysis Date:** 2026-07-01

new-api is an LLM API gateway/proxy. Its primary integrations are upstream AI model providers (relayed via per-provider adaptors) plus payment, auth, email, monitoring, and analytics services.

## APIs & External Services

### LLM / AI Model Providers (relay adaptors)

Every upstream provider implements the `channel.Adaptor` interface (`relay/channel/adapter.go` line 15) or `channel.TaskAdaptor` (line 34) for async video/image/audio tasks. Adaptors are resolved by `apiType` in `relay/relay_adaptor.go` `GetAdaptor()` (line 54). Channel type IDs and default base URLs are defined in `constant/channel.go`. Credentials are stored per-channel in the `channels` DB table (admin-configured), not in env vars.

**Chat / Completion / Embedding / Image / Rerank / Audio Providers:**

| Provider | Channel Type | Adaptor | Default Base URL |
|----------|-------------|---------|------------------|
| OpenAI | `ChannelTypeOpenAI` (1) | `relay/channel/openai/adaptor.go` | `https://api.openai.com` |
| Azure OpenAI | `ChannelTypeAzure` (3) | `relay/channel/openai/adaptor.go` (Azure mode) | (per-deployment) |
| Anthropic Claude | `ChannelTypeAnthropic` (14) | `relay/channel/claude/adaptor.go` | `https://api.anthropic.com` |
| Google Gemini | `ChannelTypeGemini` (24) | `relay/channel/gemini/adaptor.go` | `https://generativelanguage.googleapis.com` |
| Google PaLM | `ChannelTypePaLM` (11) | `relay/channel/palm/adaptor.go` | — |
| Google Vertex AI | `ChannelTypeVertexAi` (41) | `relay/channel/vertex/adaptor.go` (service-account JWT auth, `relay/channel/vertex/service_account.go`) | — |
| AWS Bedrock | `ChannelTypeAws` (33) | `relay/channel/aws/adaptor.go` (AWS SDK v2 `bedrockruntime`; API-key or AK/SK modes) | — |
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
| Tencent Hunyuan | `ChannelTypeTencent` (23) | `relay/channel/tencent/adaptor.go` | `https://hunyuan.tencentcloudapi.com` |
| Baidu (ERNIE) | `ChannelTypeBaidu` (15) / `BaiduV2` (46) | `relay/channel/baidu/adaptor.go`, `baidu_v2/adaptor.go` | `https://aip.baidubce.com` / `https://qianfan.baidubce.com` |
| Ali (Qwen / DashScope) | `ChannelTypeAli` (17) | `relay/channel/ali/adaptor.go` | `https://dashscope.aliyuncs.com` |
| Xunfei (Spark) | `ChannelTypeXunfei` (18) | `relay/channel/xunfei/adaptor.go` | — |
| Zhipu (GLM) | `ChannelTypeZhipu` (16) / `ZhipuV4` (26) | `relay/channel/zhipu/adaptor.go`, `zhipu_4v/adaptor.go` | `https://open.bigmodel.cn` |
| VolcEngine (Doubao) | `ChannelTypeVolcEngine` (45) | `relay/channel/volcengine/adaptor.go` | `https://ark.cn-beijing.volces.com` |
| Xinference | `ChannelTypeXinference` (47) | `relay/channel/openai/adaptor.go` | — |
| LingYiWanWu | `ChannelTypeLingYiWanWu` (31) | `relay/channel/openai/adaptor.go` | `https://api.lingyiwanwu.com` |
| MokaAI | `ChannelTypeMokaAI` (44) | `relay/channel/mokaai/adaptor.go` | `https://api.moka.ai` |
| Submodel | `ChannelTypeSubmodel` (53) | `relay/channel/submodel/adaptor.go` | `https://llm.submodel.ai` |
| ChatGPT Subscription (Codex) | `ChannelTypeCodex` (57) | `relay/channel/codex/adaptor.go` (OAuth refresh-token flow; `service/codex_oauth.go`) | `https://chatgpt.com` |
| Advanced Custom | `ChannelTypeAdvancedCustom` (58) | `relay/channel/advancedcustom/adaptor.go` | (operator-defined) |

Relay entry routes: `router/relay-router.go` — `/v1/chat/completions`, `/v1/messages` (Claude), `/v1/responses`, `/v1/embeddings`, `/v1/audio/*`, `/v1/images/*`, `/v1/rerank`, `/v1beta/models/*` (Gemini), `/v1/realtime` (WebSocket).

**Async Task Providers (video / image / music) — `channel.TaskAdaptor`:**

| Provider | Platform | Adaptor |
|----------|----------|---------|
| Midjourney | `:mode/mj/*` routes (`router/relay-router.go` `registerMjRouterGroup`) | `relay/mjproxy_handler.go` |
| Suno (music) | `/suno/submit/*` | `relay/channel/task/suno/adaptor.go` |
| Kling (video) | `/kling/v1/videos/*` (`router/video-router.go`) | `relay/channel/task/kling/adaptor.go` |
| Jimeng (image/video) | `/jimeng` | `relay/channel/task/jimeng/adaptor.go` |
| Vertex AI (video) | `/v1/video/generations` | `relay/channel/task/vertex/adaptor.go` |
| Vidu (video) | `/v1/video/generations` | `relay/channel/task/vidu/adaptor.go` |
| Doubao Video / VolcEngine | `/v1/video/generations` | `relay/channel/task/doubao/adaptor.go` |
| Sora (OpenAI video) | `/v1/videos` | `relay/channel/task/sora/adaptor.go` |
| Gemini (image gen) | task adaptor | `relay/channel/task/gemini/adaptor.go` |
| Hailuo (MiniMax video) | task adaptor | `relay/channel/task/hailuo/adaptor.go` |
| Ali (Wanx image) | task adaptor | `relay/channel/task/ali/adaptor.go` |

Auth: per-channel API key / OAuth token stored in DB `channels` table; passed via `SetupRequestHeader` per adaptor. Codex adaptor auto-refreshes OpenAI OAuth tokens (`service/codex_credential_refresh_task.go`, started in `main.go` line 119).

### Payment Providers

| Provider | SDK / Client | Init | Webhook Route | Controller |
|----------|--------------|------|---------------|------------|
| Stripe | `github.com/stripe/stripe-go/v81` + `webhook` | `setting/payment_stripe.go` (`StripeApiSecret`, `StripeWebhookSecret`, `StripePriceId`) | `POST /api/stripe/webhook` (`router/api-router.go` line 57) | `controller/topup_stripe.go`, `controller/subscription_payment_stripe.go` |
| EPay | `github.com/Calcium-Ion/go-epay v0.0.4` | admin-configured merchant key | `GET/POST /api/user/epay/notify`, `/api/subscription/epay/notify` (lines 76-77, 180-181) | `controller/topup.go` (`RequestEpay`), `controller/subscription_payment_epay.go` |
| Creem | custom HTTP + HMAC-SHA256 (`controller/topup_creem.go` `verifyCreemSignature`) | `setting/payment_creem.go` (`CreemApiKey`, `CreemWebhookSecret`, `CreemTestMode`) | `POST /api/creem/webhook` (line 58) | `controller/topup_creem.go`, `controller/subscription_payment_creem.go` |
| Waffo (Global) | `github.com/waffo-com/waffo-go v1.3.2` | `setting/payment_waffo.go` | `POST /api/waffo/webhook` (line 59) | `controller/topup_waffo.go` |
| Waffo Pancake | `github.com/waffo-com/waffo-pancake-sdk-go v0.3.1` | `setting/payment_waffo_pancake.go` (`WaffoPancakeMerchantID`, `WaffoPancakePrivateKey`, `WaffoPancakeStoreID`, `WaffoPancakeProductID`) | `POST /api/waffo-pancake/webhook/:env` (line 62; `:env` separates test/prod) | `controller/topup_waffo_pancake.go`, `controller/subscription_payment_waffo_pancake.go`, `service/waffo_pancake.go` |

Pay endpoints (authenticated): `/api/user/stripe/pay`, `/api/user/pay` (epay), `/api/user/creem/pay`, `/api/user/waffo/pay`, `/api/user/waffo-pancake/pay` (`router/api-router.go` lines 99-107); subscription equivalents under `/api/subscription/*` (lines 157-161).

### OAuth / Social Login Providers

All standard providers implement the `oauth.Provider` interface (`oauth/provider.go`) and self-register in `init()`. Unified route: `GET /api/oauth/:provider` → `controller.HandleOAuth` (`router/api-router.go` line 54).

| Provider | Slug | Implementation | Settings |
|----------|------|----------------|----------|
| GitHub | `github` | `oauth/github.go` (token exchange `github.com/login/oauth/access_token`, user info `api.github.com/user`) | `common.GitHubClientId/Secret` |
| Discord | `discord` | `oauth/discord.go` | `setting/system_setting/discord.go` (`GetDiscordSettings()`) |
| OIDC (generic) | `oidc` | `oauth/oidc.go` (discovery-based) | `setting/system_setting/oidc.go` |
| LinuxDO | `linuxdo` | `oauth/linuxdo.go` | `common.LinuxDOClientId/Secret` |
| Custom (DB-driven) | any slug | `oauth/generic.go` + `oauth/registry.go` `LoadCustomProviders()` (loaded from `custom_oauth_providers` table in `main.go` line 336) | `model/custom_oauth_provider.go` |
| WeChat | `wechat` | `controller/wechat.go` (proxied via `WeChatServerAddress` + `WeChatServerToken`) | `common.WeChatServerAddress/Token` |
| Telegram | `telegram` | `controller/telegram.go` (HMAC validation against `TelegramBotToken`) | `common.TelegramBotToken` |

Bindings: `/api/oauth/wechat/bind`, `/api/oauth/telegram/bind`, `/api/oauth/email/bind` (lines 50-52). Admin custom-provider CRUD: `/api/custom-oauth-provider/*` (lines 202-211).

### Other External Services

- **Cloudflare Turnstile** — bot protection on register/login/reset/checkin. Middleware `middleware/turnstile-check.go`; keys `common.TurnstileSiteKey/SecretKey`.
- **io.net** — GPU cluster / model deployment management. Client `pkg/ionet`; API key from `OptionMap["model_deployment.ionet.api_key"]` (`controller/deployment.go` line 19). Routes `/api/deployments/*`.
- **Uptime Kuma** — external status page fetch. `controller/uptime_kuma.go` polls `/api/status-page/` and `/api/status-page/heartbeat/` (constants lines 22-23).
- **Pyroscope** — continuous Go profiling (optional). `common/pyro.go` `StartPyroScope()` (env `PYROSCOPE_URL`).
- **pprof** — Go runtime profiling on `:8005` when `ENABLE_PPROF=true` (`main.go` lines 153-159).
- **Prometheus** — client libs pulled in transitively via pyroscope (`go.mod` lines 143-146).
- **OpenAI Codex OAuth** — token refresh against `https://auth.openai.com/oauth/token` with client ID `app_EMoamEEZ73f0CkXaXp7hrann` (`service/codex_oauth.go` lines 17-19).
- **Umami Analytics** — script injected into embedded `index.html` when `UMAMI_WEBSITE_ID` set (`main.go` `InjectUmamiAnalytics` line 218).
- **Google Analytics (GA4)** — gtag injected when `GOOGLE_ANALYTICS_ID` set (`main.go` `InjectGoogleAnalytics` line 239).

## Data Storage

**Databases:**
- Primary: SQLite (default, file `one-api.db`), MySQL, or PostgreSQL — chosen by `SQL_DSN` scheme in `model/main.go` `chooseDB` (line 127).
  - Client: GORM (`gorm.io/gorm v1.25.2`) + driver packages (`mysql`, `postgres`, `glebarez/sqlite`, `clickhouse`).
  - Connection: env `SQL_DSN`. Pool: `SQL_MAX_IDLE_CONNS` (100), `SQL_MAX_OPEN_CONNS` (1000), `SQL_MAX_LIFETIME` (60s) — `model/main.go` lines 203-205.
  - Migration: `migrateDB()` (`model/main.go` line 263) auto-migrates ~30 models including `Channel`, `Token`, `User`, `Log`, `Task`, `SubscriptionPlan`, `CasbinRule`, `AuthzRole`, `PerfMetric`, `SystemInstance`, `SystemTask`.
- Log DB (optional, separate): `LOG_SQL_DSN` — supports MySQL, PostgreSQL, or **ClickHouse** (`model/main.go` `InitLogDB` line 222; ClickHouse DDL at `clickHouseLogCreateTableSQL` line 429). ClickHouse TTL via `LOG_SQL_CLICKHOUSE_TTL_DAYS`.

**File Storage:**
- Local filesystem only (no S3/OSS). Disk cache under configured cache dir (`common/disk_cache.go`, `common/disk_cache_config.go`). Container `/data` volume (`docker-compose.yml` line 26).

**Caching:**
- Redis (optional) — `REDIS_CONN_STRING`; client `github.com/go-redis/redis/v8` (`common/redis.go`). Pool size `REDIS_POOL_SIZE` (10). Enables channel/option caching + rate limiting.
- In-memory cache — `MEMORY_CACHE_ENABLED=true` (auto-on when Redis enabled; `main.go` lines 75-99). `SyncFrequency` (60s default) refresh.
- Disk cache — `common/disk_cache.go` for large response bodies.
- Async token cache — `bytedance/gopkg/cache/asynccache` (Vertex AI tokens, `relay/channel/vertex/service_account.go` line 31; 30-min expiry).

## Authentication & Identity

**Auth Provider:** Custom (built-in).

**Implementation:**
- Web sessions — `gin-contrib/sessions` cookie store, secret from `SESSION_SECRET` (`main.go` lines 184-192). 30-day MaxAge, HttpOnly, SameSite=Strict.
- API access tokens — `Authorization` header validated by `model.ValidateAccessToken` (`middleware/auth.go` `authHelper` line 37).
- API keys (relay) — `TokenAuth()` middleware validates `sk-...` keys against `tokens` table (`middleware/auth.go`).
- Roles — Guest(0) / Common(1) / Admin(10) / Root(100) (`common/constants.go` lines 200-205). Enforced by `UserAuth`, `AdminAuth`, `RootAuth` middleware.
- Authorization — Casbin RBAC (`service/authz/`, `model/casbin_rule.go`, `model/authz_role.go`); policy reload synced across nodes (`main.go` line 105).
- 2FA — TOTP via `github.com/pquerna/otp` (`model/twofa.go`, `controller/twofa.go`).
- Passkey/WebAuthn — `github.com/go-webauthn/webauthn` (`controller/passkey.go`, `model/passkey.go`).
- OAuth — see providers above; bindings stored in `user_oauth_bindings` table.

## Monitoring & Observability

**Error Tracking:** None (no Sentry/bugsnag). Errors go to file logger (`logger/logger.go`) + `common.SysError`.

**Logs:**
- Custom file logger in `./logs` (`logger/logger.go`, `common/init.go` `LogDir` flag default `./logs`).
- Request/consume logs persisted to DB `logs` table (or ClickHouse).
- Optional `ERROR_LOG_ENABLED` flag.

**Profiling:**
- Pyroscope (continuous) — `common/pyro.go`.
- pprof (on-demand) — `net/http/pprof` on `:8005` when `ENABLE_PPROF=true`.
- `github.com/shirou/gopsutil` — system metrics (`common/system_monitor.go`).
- Built-in performance stats API — `/api/performance/*` (`controller/performance.go`), perf metrics `/api/perf-metrics/*` (`controller/perf_metrics.go`), `pkg/perf_metrics`.

## CI/CD & Deployment

**Hosting:**
- Container image `calciumion/new-api:latest` (Dockerfile).
- Electron desktop bundles (mac dmg/zip, win nsis/portable, linux AppImage/deb) via `electron-builder` (`electron/package.json`).
- Systemd service (`new-api.service`).
- Bare binary via `go build` (`Dockerfile` line 39).

**CI Pipeline:** Not detected (no `.github/workflows` in scanned scope; build is `make`/Docker-driven).

## Environment Configuration

**Required env vars (production):**
- `SQL_DSN` — primary DB (PostgreSQL recommended for multi-node; `docker-compose.yml` line 29).
- `SESSION_SECRET` — mandatory for multi-node; panics if left as default `random_string` (`common/init.go` line 51).
- `CRYPTO_SECRET` — token encryption (defaults to `SESSION_SECRET`).

**Optional env vars:**
- `LOG_SQL_DSN`, `LOG_SQL_CLICKHOUSE_TTL_DAYS` — separate log DB / ClickHouse TTL.
- `REDIS_CONN_STRING`, `REDIS_POOL_SIZE` — cache & rate limit.
- `NODE_TYPE=slave`, `NODE_NAME` — multi-instance identity.
- `PORT`, `GIN_MODE=debug`, `DEBUG=true`.
- `SQLITE_PATH`, `SYNC_FREQUENCY`, `BATCH_UPDATE_ENABLED`, `BATCH_UPDATE_INTERVAL`.
- `RELAY_TIMEOUT`, `RELAY_IDLE_CONN_TIMEOUT`, `RELAY_MAX_IDLE_CONNS`, `RELAY_MAX_IDLE_CONNS_PER_HOST`.
- `STREAMING_TIMEOUT`, `MAX_FILE_DOWNLOAD_MB`, `MAX_REQUEST_BODY_MB`, `FORCE_STREAM_OPTION`, `GET_MEDIA_TOKEN`.
- `TLS_INSECURE_SKIP_VERIFY`, `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` (relay client respects proxy env; `service/http_client.go` line 42).
- `GLOBAL_API_RATE_LIMIT*`, `GLOBAL_WEB_RATE_LIMIT*`, `CRITICAL_RATE_LIMIT*`, `SEARCH_RATE_LIMIT*`.
- `ENABLE_PPROF`, `PYROSCOPE_URL`, `PYROSCOPE_APP_NAME`, `PYROSCOPE_BASIC_AUTH_*`.
- `GOOGLE_ANALYTICS_ID`, `UMAMI_WEBSITE_ID`, `UMAMI_SCRIPT_URL`.
- `CHANNEL_UPDATE_FREQUENCY`, `GENERATE_DEFAULT_TOKEN`, `ERROR_LOG_ENABLED`.
- `GEMINI_SAFETY_SETTING`, `COHERE_SAFETY_SETTING`, `AZURE_DEFAULT_API_VERSION`.
- `TRUSTED_REDIRECT_DOMAINS` — allowlist for payment success/cancel redirect URLs (`common/init.go` line 174).
- `FRONTEND_BASE_URL` — serve frontend separately & redirect (`router/main.go` line 20).

Full env parsing: `common/init.go` `InitEnv()` (line 31) + `initConstantEnv()` (line 134). SMTP/OAuth/payment provider credentials are admin-configured at runtime via the DB-backed options table (`model/option.go`), not env vars.

**Secrets location:**
- `.env` file (loaded by `godotenv`; existence noted only — never read by tooling).
- DB `options` table (`model/option.go`) — admin-entered via UI; keys containing `Secret`/`Token` are masked from `GetOptions` (`common/constants.go` line 73 comment).
- Channel API keys — `channels` DB table.

## Webhooks & Callbacks

**Incoming (this server receives):**
- `POST /api/stripe/webhook` — Stripe checkout events (`router/api-router.go` line 57; signature-verified in `controller/topup_stripe.go`).
- `POST /api/creem/webhook` — Creem payment events (line 58; HMAC-SHA256 verified `controller/topup_creem.go`).
- `POST /api/waffo/webhook` — Waffo legacy gateway (line 59).
- `POST /api/waffo-pancake/webhook/:env` — Waffo Pancake checkout events (line 62; `:env` = test|prod).
- `GET/POST /api/user/epay/notify` — EPay async notify (lines 76-77).
- `GET/POST /api/subscription/epay/notify`, `/api/subscription/epay/return` — EPay subscription callbacks (lines 180-183).
- `GET /api/oauth/:provider` — OAuth provider redirects back with `code` (line 54).

**Outgoing (this server sends):**
- Outbound webhook notifications to admin-configured URL — `service/webhook.go` `SendWebhookNotify` (HMAC-SHA256 signed; line 35). Triggered for quota exceed, channel updates, channel tests (`dto/notify.go`).
- Email notifications via SMTP — `common/email.go` `SendEmail`; `service/user_notify.go` `NotifyUser` / `NotifyRootUser` / `NotifyUpstreamModelUpdateWatchers`.
- Relay requests to all upstream LLM providers (see provider table above) — via shared HTTP client `service/http_client.go` `GetHttpClient()` with SSRF protection (`common/ssrf_protection.go`, `common/url_validator.go`).
- OAuth token exchange & user-info fetches to GitHub, Discord, OIDC, LinuxDO, WeChat proxy, Telegram.
- Codex OAuth token refresh to `https://auth.openai.com/oauth/token` (`service/codex_oauth.go`).
- Vertex AI service-account JWT exchange to Google OAuth2 token endpoint (`relay/channel/vertex/service_account.go`).
- io.net deployment API calls (`controller/deployment.go` via `pkg/ionet.Client`).
- Uptime Kuma status polling (`controller/uptime_kuma.go`).
- Pyroscope profile uploads (`common/pyro.go`).

---

*Integration audit: 2026-07-01*
