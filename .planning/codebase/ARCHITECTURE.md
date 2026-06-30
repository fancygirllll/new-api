<!-- refreshed: 2026-07-01 -->
# Architecture

**Analysis Date:** 2026-07-01

## System Overview

new-api is an AI API gateway/proxy built in Go (Gin) that aggregates 40+ upstream AI providers (OpenAI, Anthropic Claude, Gemini, Azure, AWS Bedrock, Vertex AI, etc.) behind a unified OpenAI/Claude/Gemini-compatible API. It adds user management, token-based billing, rate limiting, channel selection, an admin dashboard, and a React frontend.

```text
┌──────────────────────────────────────────────────────────────────────┐
│                         HTTP Clients / Frontend                        │
│   (OpenAI SDK, Claude SDK, Gemini SDK, React default+classic themes)   │
└──────────────┬───────────────────────────────────────┬───────────────┘
               │ /v1/* /v1beta/* /mj /suno  (relay)     │ /api/* /pg/* /dashboard/*
               ▼                                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          Router Layer (Gin)                           │
│  `router/main.go` → SetApiRouter / SetRelayRouter / SetDashboardRouter │
│                    / SetVideoRouter / SetWebRouter                     │
├──────────────────┬───────────────────────┬───────────────────────────┤
│  Middleware Chain │  Controller Handlers  │   Embedded Static Frontend │
│ `middleware/*.go` │   `controller/*.go`   │  `web/default/dist`        │
└────────┬─────────┴───────────┬───────────┴──────────┬────────────────┘
         │                      │                       │
         ▼                      ▼                       │
┌──────────────────────────────────────────────────────┐│
│                  Service Layer (business logic)       ││
│  `service/billing.go` `service/channel_select.go`     ││
│  `service/authz/` (casbin RBAC) `service/passkey/`    ││
└────────┬─────────────────────────────────┬───────────┘│
         │                                 │            │
         ▼                                 ▼            ▼
┌─────────────────────────┐   ┌──────────────────────────────────────┐
│   Relay / Proxy Core    │   │        Model Layer (GORM v2)          │
│      `relay/*.go`       │   │       `model/*.go`                    │
│  Adaptor interface +    │◄──┤  User, Token, Channel, Ability,       │
│  40+ provider adapters  │   │  Log, Task, Subscription, Pricing      │
│  `relay/channel/<prov>/`│   │  + in-memory channel cache             │
└────────────┬────────────┘   └───────────────┬──────────────────────┘
             │                                  │
             ▼                                  ▼
┌─────────────────────────┐   ┌──────────────────────────────────────┐
│  Upstream AI Providers  │   │  SQLite / MySQL / PostgreSQL + Redis │
│  (OpenAI, Claude, etc.) │   │  (+ optional ClickHouse log DB)      │
└─────────────────────────┘   └──────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Entry point / bootstrap | Loads env, inits DB/Redis/authz/i18n, builds Gin server, embeds frontend | `main.go` |
| Router wiring | Registers all route groups + theme-aware static serving | `router/main.go` |
| Relay API routes | OpenAI/Claude/Gemini/MJ/Suno compatible endpoints | `router/relay-router.go` |
| Admin/management API | Users, channels, tokens, billing, options, subscriptions | `router/api-router.go` |
| Auth middleware | Session + token + passkey auth, role gating, audit | `middleware/auth.go` |
| Channel distributor | Selects upstream channel per group+model+affinity | `middleware/distributor.go` |
| Relay orchestrator | Request parse → price → pre-charge → retry loop → settle | `controller/relay.go` |
| Adaptor factory | Maps `apiType` → provider adaptor implementation | `relay/relay_adaptor.go` |
| Adaptor contract | Provider-agnostic interface every channel implements | `relay/channel/adapter.go` |
| Request context carrier | `RelayInfo` holds token/user/channel/billing state | `relay/common/relay_info.go` |
| Billing session | Pre-consume → settle/refund quota flow | `service/billing.go`, `service/billing_session.go` |
| Channel selection cache | In-memory group→model→channels index | `model/channel_cache.go` |
| Ability mapping | DB-backed group+model+channel enabled matrix | `model/ability.go` |
| RBAC enforcement | Casbin permissions for channel/admin ops | `service/authz/enforcer.go` |
| Database init | Multi-DB dialect selection, column quoting, migrations | `model/main.go` |
| JSON wrapper | Enforces single JSON codec across codebase | `common/json.go` |

## Pattern Overview

**Overall:** Layered architecture (Router → Controller → Service → Model) with a Strategy/Adapter pattern for provider abstraction and a factory for adaptor instantiation.

**Key Characteristics:**
- Layered: each layer only calls downward (Router → Controller → Service → Model), with one injected callback to break a service↔relay cycle (see Architectural Constraints).
- Provider-agnostic relay: every upstream provider implements the `channel.Adaptor` interface; `relay.GetAdaptor(apiType)` returns the concrete instance. Adding a provider = new subpackage under `relay/channel/` + a case in the factory switch.
- Retry-with-channel-fallback: the relay loop retries failed requests against alternative channels from the same group, governed by `shouldRetry` / `shouldRetryTaskRelay`.
- Billing as a session: quota is pre-consumed before the upstream call, then settled (supplement/refund) after the response; failures refund via `relayInfo.Billing.Refund`.
- Multi-DB dialect-agnostic: all DB code must run on SQLite, MySQL, and PostgreSQL simultaneously (ClickHouse only for logs).
- Embedded SPA: both frontend themes are compiled to `dist/` and embedded into the Go binary with `//go:embed`.
- Master/worker nodes: scheduled tasks and seeding run only on the master node (`common.IsMasterNode`).

## Layers

**Router:**
- Purpose: Define HTTP routes and per-group middleware chains; serve embedded frontend.
- Location: `router/`
- Contains: `main.go`, `api-router.go`, `relay-router.go`, `dashboard.go`, `web-router.go`, `video-router.go`, `channel-router.go`, `authz-router.go`
- Depends on: `controller`, `middleware`
- Used by: `main.go` (the only caller of `SetRouter`)

**Middleware:**
- Purpose: Cross-cutting request processing — auth, rate limiting, CORS, gzip, distribution, audit, i18n, request-id, body limits.
- Location: `middleware/`
- Contains: `auth.go` (TokenAuth/UserAuth/AdminAuth/RootAuth/RequirePermission), `distributor.go`, `rate-limit.go`, `model-rate-limit.go`, `cors.go`, `gzip.go`, `audit.go`, `i18n.go`, `request-id.go`, `request_body_limit.go`, `secure_verification.go`, `turnstile-check.go`, `cache.go`, `stats.go`, `performance.go`
- Depends on: `model`, `service`, `common`, `i18n`
- Used by: router definitions and route groups via `router.Use(...)`

**Controller:**
- Purpose: Request handlers — parse input, call service/model, format HTTP response.
- Location: `controller/`
- Contains: `relay.go` (Relay/RelayTask/RelayMidjourney orchestration), `user.go`, `channel.go`, `token.go`, `billing.go`, `topup_*.go`, `subscription*.go`, `log.go`, `pricing.go`, `model.go`, `playground.go`, `passkey.go`, `twofa.go`, `oauth.go`, `system_task*.go`, etc.
- Depends on: `service`, `model`, `relay`, `dto`, `types`, `common`
- Used by: router handlers

**Service:**
- Purpose: Business logic that doesn't belong to a single handler — billing, channel selection, sensitive checks, token counting, notifications, payments.
- Location: `service/` plus subpackages `authz/`, `passkey/`, `relayconvert/`
- Contains: `billing.go`, `billing_session.go`, `pre_consume_quota.go`, `channel_select.go`, `channel_affinity.go`, `channel.go`, `quota.go`, `text_quota.go`, `task_billing.go`, `tiered_settle.go`, `sensitive.go`, `token_counter.go`, `tokenizer.go`, `epay.go`, `webhook.go`, `system_task.go`, `task_polling.go`, `error.go`, etc.
- Depends on: `model`, `relay` (via injected `GetTaskAdaptorFunc`), `common`, `setting`
- Used by: `controller`, `middleware`

**Model:**
- Purpose: GORM entities, DB access, connection management, in-memory caches.
- Location: `model/`
- Contains: `main.go` (DB init + dialect helpers), `user.go`, `token.go`, `channel.go`, `ability.go`, `channel_cache.go`, `user_cache.go`, `token_cache.go`, `log.go`, `task.go`, `system_task.go`, `pricing.go`, `subscription.go`, `option.go`, `casbin_rule.go`, `authz_role.go`, etc.
- Depends on: `common`, `constant`, `dto`, GORM drivers
- Used by: all upper layers

**Relay:**
- Purpose: The AI-proxy core — convert client requests to upstream format, execute, convert responses back, stream, bill.
- Location: `relay/` plus `relay/channel/<provider>/`, `relay/common/`, `relay/helper/`, `relay/constant/`, `relay/common_handler/`, `relay/reasonmap/`
- Contains: `relay_adaptor.go` (factory), `compatible_handler.go` (text), `claude_handler.go`, `gemini_handler.go`, `image_handler.go`, `audio_handler.go`, `embedding_handler.go`, `rerank_handler.go`, `responses_handler.go`, `mjproxy_handler.go`, `relay_task.go`, `websocket.go`; `channel/adapter.go` (interfaces); `common/relay_info.go` (RelayInfo); `helper/valid_request.go`, `helper/price.go`, `helper/common.go`
- Depends on: `dto`, `types`, `model`, `service`, `common`, `setting`
- Used by: `controller` (Relay entry points), `service` (task polling via injected factory)

## Data Flow

### Primary Request Path (Relay — e.g. `POST /v1/chat/completions`)

1. Gin receives request → global middleware: `RequestId`, `PoweredBy`, `I18n`, logger, sessions (`main.go:179-192`)
2. Relay router group middleware: `CORS`, `DecompressRequestMiddleware`, `BodyStorageCleanup`, `StatsMiddleware`, `RouteTag("relay")`, `SystemPerformanceCheck`, `TokenAuth`, `ModelRequestRateLimit`, `Distribute` (`router/relay-router.go:69-85`)
3. `TokenAuth` validates `sk-...` token, loads user cache, sets context keys (`middleware/auth.go:303-434`)
4. `Distribute` selects an upstream channel by group+model+affinity and seeds channel context (`middleware/distributor.go:32`)
5. `controller.Relay(c, RelayFormatOpenAI)` entry (`controller/relay.go:68`)
6. `helper.GetAndValidateRequest` parses+validates body into a `dto.Request` (`relay/helper/valid_request.go:21`)
7. `relaycommon.GenRelayInfo` builds the `RelayInfo` carrying token/user/channel/billing state (`relay/common/relay_info.go`)
8. Optional sensitive-word check + `service.EstimateRequestToken` for prompt token count
9. `helper.ModelPriceHelper` computes `PriceData` (model ratio × group ratio × tokens) (`relay/helper/price.go`)
10. `service.PreConsumeBilling` creates a `BillingSession` and pre-deducts quota (`service/billing.go:19`)
11. Retry loop (`controller/relay.go:191-237`):
    a. `getChannel` → picks/refreshes channel from cache (`controller/relay.go:293`)
    b. `relay.GetAdaptor(info.ApiType)` returns provider adaptor (`relay/relay_adaptor.go:54`)
    c. `adaptor.Init` → `ConvertOpenAIRequest` → `GetRequestURL` → `SetupRequestHeader` → `DoRequest` → `DoResponse`
    d. On success: return; on error: `processChannelError` (may auto-ban channel) and `shouldRetry` decides next attempt
12. On response: `service.PostTextConsumeQuota` settles billing (supplement/refund delta) and logs consumption
13. On terminal failure: `relayInfo.Billing.Refund` returns pre-consumed quota; error is JSON-formatted per `relayFormat` (OpenAI/Claude shape)

### Async Task Flow (Midjourney / Suno / Video — e.g. `POST /suno/submit/:action`)

1. Router → `controller.RelayTask` (`controller/relay.go:486`)
2. `relaycommon.GenRelayInfo` (RelayFormatTask) → `relay.ResolveOriginTask`
3. `relay.GetTaskAdaptor(platform)` returns a `channel.TaskAdaptor` (`relay/relay_adaptor.go:138`)
4. Retry loop: `BuildRequestURL/Header/Body` → `DoRequest` → `DoResponse` returns `taskID`
5. On success: `service.SettleBilling`, `service.LogTaskConsumption`, `model.InitTask(...).Insert()` persists the task
6. Background polling: `service.StartSystemTaskRunner` periodically calls `service.RunTaskPollingOnce` → adaptor `FetchTask` + `ParseTaskResult` → `AdjustBillingOnComplete` settles quota delta → updates task row

### Management/Dashboard Flow (e.g. `GET /api/channel/`)

1. `SetApiRouter` registers `/api/channel/` group with `AdminAuth` (`router/channel-router.go:19-37`)
2. Each route additionally requires `middleware.RequirePermission(permission)` (casbin check, `middleware/auth.go:199-213`)
3. `authHelper` enforces session/access-token + role + audit (`middleware/auth.go:37-168`)
4. Controller (e.g. `controller.GetAllChannels`) reads via `model.*` and returns JSON

**State Management:**
- Request-scoped state lives on `*gin.Context` (context keys defined in `constant/context_key.go`).
- Long-lived mutable state: `model.OptionMap` (settings, RWMutex-guarded), in-memory channel cache (`model/channel_cache.go` `group2model2channels` + `channelsIDM`), casbin `SyncedEnforcer` (`service/authz/enforcer.go`).
- Background goroutines refresh state periodically (`model.SyncChannelCache`, `model.SyncOptions`, `authz.StartPolicySync`, system task runner) — see `main.go:98-145`.

## Key Abstractions

**Adaptor interface:**
- Purpose: Abstracts a single upstream AI provider behind a uniform set of conversion + transport methods.
- Examples: `relay/channel/openai/adaptor.go`, `relay/channel/claude/adaptor.go`, `relay/channel/gemini/adaptor.go`, `relay/channel/aws/adaptor.go`, plus 30+ others under `relay/channel/`
- Pattern: Strategy + Adapter. `relay/relay_adaptor.go:54-128` (`GetAdaptor`) is the factory switch.

**TaskAdaptor interface:**
- Purpose: Async task providers (Midjourney, Suno, Kling, Sora, Vidu, etc.) — submit + poll lifecycle.
- Examples: `relay/channel/task/suno/`, `relay/channel/task/kling/`, `relay/channel/task/sora/`, `relay/channel/task/vertex/`
- Pattern: Strategy. Factory in `relay/relay_adaptor.go:138-167` (`GetTaskAdaptor`).

**RelayInfo:**
- Purpose: Carries everything the relay pipeline needs about a single request — token, user, group, channel meta, pricing, billing session, retry index, stream state.
- Examples: `relay/common/relay_info.go`
- Pattern: Context object passed through the pipeline.

**BillingSession:**
- Purpose: Encapsulates pre-consume → settle/refund so wallet and subscription billing share one code path.
- Examples: `service/billing_session.go`, `service/billing.go`
- Pattern: Strategy over `BillingSource` (wallet vs subscription).

**Ability / channel cache:**
- Purpose: Maps (group, model) → ordered list of enabled channel IDs, used by the distributor for weighted-priority selection.
- Examples: `model/ability.go` (DB entity), `model/channel_cache.go` (in-memory index)
- Pattern: Read-through cache with periodic sync.

**Casbin enforcer:**
- Purpose: RBAC permission checks (`authz.Can(userID, role, permission)`) wired into `middleware.RequirePermission`.
- Examples: `service/authz/enforcer.go`, `service/authz/permission.go`, `service/authz/resources_channel.go`
- Pattern: Policy-as-data (gorm adapter) with periodic cross-node sync.

## Entry Points

**Process entry (`main.go:51`):**
- Location: `main.go`
- Triggers: `go run` / compiled binary / Docker
- Responsibilities: `InitResources` (env, logger, ratio settings, HTTP client, token encoders, DB, authz, options, cache, i18n, OAuth providers); start background goroutines (channel cache sync, option sync, policy sync, quota data, system instance reporter, codex credential refresh, subscription reset, system task runner, batch updater, pprof, pyroscope); build Gin server with recovery/request-id/i18n/logger/session middleware; `router.SetRouter`; `server.Run`.

**HTTP entry (`router/main.go:15`):**
- Location: `router/main.go`
- Triggers: every inbound HTTP request
- Responsibilities: Wires the five router groups (API, Dashboard, Relay, Video, Web) and the theme-aware static fallback.

**Relay entry (`controller/relay.go:68`):**
- Location: `controller.Relay`, `controller.RelayTask`, `controller.RelayMidjourney`
- Triggers: relay router handlers
- Responsibilities: Orchestrate parse → price → pre-charge → retry loop → settle.

## Architectural Constraints

- **Multi-database compatibility:** All DB code MUST work on SQLite, MySQL >= 5.7.8, and PostgreSQL >= 9.6 simultaneously (ClickHouse only for the log DB via `LOG_SQL_DSN`). Use GORM methods over raw SQL; use `commonGroupCol`/`commonKeyCol`/`commonTrueVal`/`commonFalseVal` from `model/main.go` for reserved-word columns and booleans; branch with `common.UsingMainDatabase(...)` / `common.UsingLogDatabase(...)`. See `AGENTS.md` "Database compatibility".
- **JSON codec:** All marshal/unmarshal MUST go through `common.Marshal` / `common.Unmarshal` / `common.UnmarshalJsonStr` / `common.DecodeJsonStr` (`common/json.go`). Do NOT call `encoding/json` directly in business code.
- **Relay DTO zero values:** Optional scalar fields in client-parsed request structs MUST use pointer types with `omitempty` so explicit `0`/`false` are preserved upstream (`AGENTS.md` "Relay and provider behavior").
- **Service → relay import cycle:** `service` cannot import `relay` directly. The cycle is broken by injecting `service.GetTaskAdaptorFunc` from `main.go:131-137` — do not add a direct `relay` import from `service`.
- **Master/worker execution:** Background scheduled tasks (channel test, upstream model update, async task polling) run only on the master node and use a DB lease for cross-instance dedup (`controller.RegisterScheduledSystemTasks`, `service.StartSystemTaskRunner`, `common.IsMasterNode`).
- **Threading:** Single-process, multi-goroutine. CPU-bound work and fire-and-forget side effects use `gopool.Go` (`github.com/bytedance/gopkg/util/gopool`). The casbin enforcer is a `SyncedEnforcer`; channel cache and option map are guarded by `sync.RWMutex`.
- **Global state:** `model.OptionMap` + `model.OptionMapRWMutex` (`model/option.go`), `model.group2model2channels` / `model.channelsIDM` / `model.channel2advancedCustomConfig` + `model.channelSyncLock` (`model/channel_cache.go`), `service.authz.enforcer` + `enforcerMu` (`service/authz/enforcer.go`), `common.themeValue` (`common/constants.go`), `model.DB` / `model.LOG_DB` (`model/main.go`).
- **Embedded frontend:** The binary embeds `web/default/dist` and `web/classic/dist` via `//go:embed` (`main.go:39-49`). Frontend must be built before the Go build.
- **Protected identifiers:** Project name (new-api) and organization (QuantumNous) identifiers — module path, branding, copyright — are protected and must not be renamed (`AGENTS.md` "Project Governance").

## Anti-Patterns

### Calling `encoding/json` directly in business code

**What happens:** A new handler uses `json.Marshal` / `json.Unmarshal` from `encoding/json` instead of the wrapper.
**Why it's wrong:** The codebase routes all JSON through `common/json.go` (which uses `sonic`/`goccy` variants configured centrally) for consistent behavior and performance. Direct calls bypass this and can diverge on edge cases.
**Do this instead:** Use `common.Marshal(v)`, `common.Unmarshal(data, &v)`, `common.UnmarshalJsonStr(s, &v)`, `common.DecodeJsonStr(reader, &v)` from `common/json.go`. Reference `encoding/json` types only as types.

### Database-specific SQL without cross-DB fallback

**What happens:** Raw SQL uses MySQL backtick quoting or PostgreSQL-only operators, or a model relies on `gorm:"default:true"` boolean tags.
**Why it's wrong:** The same code must run on SQLite, MySQL, and PostgreSQL. Mismatched quoting or boolean normalization causes GORM `AutoMigrate` to issue spurious `ALTER TABLE` on every restart.
**Do this instead:** Use `commonGroupCol`/`commonKeyCol`/`commonTrueVal`/`commonFalseVal` from `model/main.go` for reserved words and booleans; prefer GORM methods; set business-rule defaults in model hooks or service code, not in `default:` tags (see `AGENTS.md` "Database compatibility").

### Adding a direct `relay` import from `service`

**What happens:** A new service function needs a task adaptor and imports `relay` directly.
**Why it's wrong:** This reintroduces the `service` ↔ `relay` import cycle that `main.go:131-137` was written to break.
**Do this instead:** Use the injected `service.GetTaskAdaptorFunc(platform)` (already set in `main.go`) which returns a `service.TaskPollingAdaptor` without importing `relay`.

### Hardcoding provider selection outside the adaptor factory

**What happens:** A handler special-cases a provider with `if channelType == constant.ChannelTypeX` instead of going through the adaptor.
**Why it's wrong:** Bypasses the `Adaptor` interface and `GetAdaptor` factory, so the provider won't get conversion/stream/billing handling and the next developer won't find the code path.
**Do this instead:** Add a case to `GetAdaptor` in `relay/relay_adaptor.go:54` returning a new `<provider>.Adaptor{}` that implements `relay/channel/adapter.go`.

## Error Handling

**Strategy:** Typed errors (`types.NewAPIError`) carry an HTTP status code, error code, error type, and retry hint; they are converted to provider-shaped responses (OpenAI / Claude / Gemini / task) at the controller boundary.

**Patterns:**
- `types.NewError(err, code, opts...)` builds the canonical relay error (`types/error.go`).
- `ErrOptionWithSkipRetry()` marks errors that must not trigger a channel retry.
- `service.NormalizeViolationFeeError` and `service.ChargeViolationFeeIfNeeded` wrap billing-specific error semantics around relay errors.
- `gin.CustomRecovery` (`main.go:168-176`) catches panics and returns a `new_api_panic` JSON error so a single request failure cannot crash the process.
- Channel errors are recorded via `processChannelError` → `model.RecordErrorLog` and may trigger `service.DisableChannel` (auto-ban) when `ShouldDisableChannel` returns true.
- Task errors use `dto.TaskError` with `LocalError` flag (no retry, no upstream report).

## Cross-Cutting Concerns

**Logging:** Custom logger in `logger/` (leveled: `SysLog`, `LogInfo`, `LogWarn`, `LogError`, `LogDebug`) writing to files under `logs/` and stdout. Request-scoped logging carries the request id. Sensitive data is masked via `common.LocalLogPreview` and `err.MaskSensitiveErrorWithStatusCode`.

**Validation:** Request validation in `relay/helper/valid_request.go` per relay format (OpenAI/Claude/Gemini/Responses/Image/Audio/Embedding/Rerank). Additional field validation uses `github.com/go-playground/validator/v10`. Anonymous request body size is capped by `middleware.AnonymousRequestBodyLimit`.

**Authentication:** Three parallel mechanisms in `middleware/auth.go` — session cookie (dashboard users via `gin-contrib/sessions` cookie store), API token `sk-...` (`TokenAuth`, supports Anthropic `x-api-key`, Gemini `x-goog-api-key`/`?key=`, MJ `mj-api-secret`, WebSocket `Sec-WebSocket-Protocol`), and access token (`New-Api-User` header). Fine-grained authorization via casbin `RequirePermission`. Passkey/WebAuthn in `service/passkey/`. 2FA TOTP in `controller/twofa.go` + `model/twofa.go`. OAuth in `oauth/` (GitHub, Discord, OIDC, LinuxDO, WeChat, Telegram) with custom providers loaded from DB at startup.

**i18n:** Backend uses `nicksnyder/go-i18n/v2` with `i18n/locales/` (en, zh); messages are translated per-request via `common.TranslateMessage(c, ...)`. Frontend uses `i18next` + `react-i18next` with flat JSON in `web/default/src/i18n/locales/`. Middleware `middleware/i18n.go` detects the language from the `Accept-Language` header / query.

**Rate limiting:** Global API rate limit (`middleware.GlobalAPIRateLimit`), global web rate limit (`middleware.GlobalWebRateLimit`), critical rate limit (`middleware.CriticalRateLimit`), search rate limit, email-verification rate limit, model request rate limit (`middleware/model-rate-limit.go`). Backed by Redis when available, else in-process limiter (`common/limiter/` with a Lua script for Redis).

**Audit:** `middleware/audit.go` (`beginAdminAudit`/`finishAdminAudit`) is wired inside `authHelper` so every `AdminAuth`/`RootAuth` write operation is audited without per-route middleware. Handlers can set `ContextKeyAuditLogged` to skip double-logging when they audit manually.

**Performance monitoring:** `pkg/perf_metrics/` records relay samples and system metrics; `middleware/performance.go` (`SystemPerformanceCheck`) can reject requests under load. Pyroscope profiling is optional (`common.StartPyroScope`); pprof on `:8005` when `ENABLE_PPROF=true`.

---

*Architecture analysis: 2026-07-01*
