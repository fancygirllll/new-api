# Codebase Structure

**Analysis Date:** 2026-07-01

## Directory Layout

```
new-api/
├── main.go                 # Process entry point; embeds frontend, builds Gin server
├── go.mod / go.sum          # Go 1.25.1 module github.com/QuantumNous/new-api
├── makefile                 # Build/test targets
├── Dockerfile / Dockerfile.dev / docker-compose*.yml  # Container builds
├── VERSION                  # Semver string injected into common.Version at build
├── .env.example             # Documented env vars (do NOT read .env contents)
├── AGENTS.md / CLAUDE.md    # Project conventions (imported into agent context)
├── bin/                     # Migration SQL + shell helper scripts
├── docs/                    # Additional documentation
├── data/                    # SQLite DB + on-disk cache files (runtime data)
├── logs/                    # Application log files (runtime output)
├── router/                  # HTTP route registration (all Gin groups)
├── middleware/              # Request pipeline (auth, distribution, rate limit, audit)
├── controller/              # Request handlers (API + relay orchestration)
├── service/                 # Business logic + authz/, passkey/, relayconvert/ subpkgs
├── model/                   # GORM entities, DB init, in-memory caches
├── relay/                   # AI-proxy core: handlers + channel/<provider>/ adapters
│   └── channel/             # 40+ provider adapter subpackages
├── dto/                     # Request/response transfer structs
├── types/                   # Core type defs (RelayFormat, NewAPIError, PriceData)
├── constant/                # Constants (channel types, api types, context keys)
├── common/                  # Shared utilities (json, redis, crypto, env, limiter/)
├── setting/                # Configuration domains (ratio_, model_, operation_, ...)
├── oauth/                   # OAuth provider implementations + registry
├── i18n/                    # Backend i18n (go-i18n) + locales/
├── logger/                  # Custom leveled logger
├── pkg/                     # Internal libs (billingexpr, cachex, ionet, perf_metrics)
├── web/                     # Frontend container
│   ├── default/             # Default theme (React 19, Rsbuild, Base UI, Tailwind)
│   └── classic/             # Classic theme (React 18, Vite, Semi Design)
└── electron/                # Electron desktop wrapper
```

## Directory Purposes

**`router/`:**
- Purpose: HTTP routing — registers all Gin route groups and per-group middleware.
- Contains: `main.go` (SetRouter wiring), `api-router.go`, `relay-router.go`, `dashboard.go`, `web-router.go`, `video-router.go`, `channel-router.go`, `authz-router.go`, `channel_router_test.go`
- Key files: `router/main.go`, `router/relay-router.go`, `router/api-router.go`

**`middleware/`:**
- Purpose: Cross-cutting request pipeline — auth, distribution, rate limiting, CORS, gzip, audit, i18n.
- Contains: `auth.go`, `distributor.go`, `rate-limit.go`, `model-rate-limit.go`, `cors.go`, `gzip.go`, `audit.go`, `i18n.go`, `request-id.go`, `request_body_limit.go`, `secure_verification.go`, `turnstile-check.go`, `cache.go`, `disable-cache.go`, `stats.go`, `performance.go`, `header_nav.go`, `kling_adapter.go`, `jimeng_adapter.go`, `body_cleanup.go`, `recover.go`, `utils.go`
- Key files: `middleware/auth.go`, `middleware/distributor.go`, `middleware/rate-limit.go`

**`controller/`:**
- Purpose: Request handlers — parse input, call service/model, format HTTP response.
- Contains: 80+ handler files covering relay, users, channels, tokens, billing, topup, subscriptions, logs, pricing, models, playground, passkey, 2FA, OAuth, system tasks, deployment (io.net), video proxy, etc.
- Key files: `controller/relay.go` (Relay/RelayTask/RelayMidjourney), `controller/user.go`, `controller/channel.go`, `controller/billing.go`, `controller/token.go`, `controller/system_task*.go`

**`service/`:**
- Purpose: Business logic reusable across handlers — billing, channel selection, sensitive checks, token counting, payments, notifications, task polling.
- Contains: `billing.go`, `billing_session.go`, `pre_consume_quota.go`, `channel_select.go`, `channel_affinity.go`, `channel.go`, `quota.go`, `text_quota.go`, `task_billing.go`, `tiered_settle.go`, `sensitive.go`, `token_counter.go`, `tokenizer.go`, `epay.go`, `webhook.go`, `system_task.go`, `task_polling.go`, `error.go`, `http_client.go`, codex credential refresh, subscription reset, etc.
- Subpackages: `service/authz/` (casbin RBAC: enforcer, permission, role, resources, registry), `service/passkey/` (WebAuthn), `service/relayconvert/` (request conversion helpers)
- Key files: `service/billing.go`, `service/channel_select.go`, `service/system_task.go`, `service/authz/enforcer.go`

**`model/`:**
- Purpose: GORM entities, DB connection management, multi-dialect helpers, in-memory caches.
- Contains: `main.go` (DB init + `commonGroupCol`/`commonKeyCol` dialect helpers), `user.go`, `token.go`, `channel.go`, `ability.go`, `channel_cache.go`, `user_cache.go`, `token_cache.go`, `log.go`, `task.go`, `system_task.go`, `pricing.go`, `subscription.go`, `option.go`, `casbin_rule.go`, `authz_role.go`, `midjourney.go`, `vendor_meta.go`, `model_meta.go`, `perf_metric.go`, `setup.go`, `errors.go`
- Key files: `model/main.go`, `model/channel_cache.go`, `model/ability.go`, `model/user.go`

**`relay/`:**
- Purpose: AI-proxy core — format conversion, upstream transport, streaming, billing integration.
- Contains: `relay_adaptor.go` (factory), `compatible_handler.go` (text), `claude_handler.go`, `gemini_handler.go`, `image_handler.go`, `audio_handler.go`, `embedding_handler.go`, `rerank_handler.go`, `responses_handler.go`, `mjproxy_handler.go`, `relay_task.go`, `websocket.go`, `chat_completions_via_responses.go`, `param_override_error.go`
- Subpackages: `relay/channel/` (provider adapters), `relay/common/` (RelayInfo, billing, override), `relay/helper/` (request validation, pricing), `relay/constant/` (RelayMode), `relay/common_handler/`, `relay/reasonmap/`
- Key files: `relay/relay_adaptor.go`, `relay/channel/adapter.go`, `relay/common/relay_info.go`, `relay/helper/valid_request.go`, `relay/helper/price.go`

**`relay/channel/<provider>/`:**
- Purpose: One subpackage per upstream AI provider implementing `channel.Adaptor` (and optionally `channel.TaskAdaptor`).
- Contains (40+ providers): `openai/`, `claude/`, `gemini/`, `aws/`, `vertex/`, `azure` (via openai), `ali/`, `baidu/`, `baidu_v2/`, `tencent/`, `xunfei/`, `zhipu/`, `zhipu_4v/`, `ollama/`, `perplexity/`, `cohere/`, `dify/`, `jina/`, `cloudflare/`, `siliconflow/`, `mistral/`, `deepseek/`, `mokaai/`, `volcengine/`, `xai/`, `coze/`, `jimeng/`, `moonshot/`, `minimax/`, `replicate/`, `codex/`, `advancedcustom/`, `submodel/`, `palm/`, `ai360/`, `lingyiwanwu/` (imported by openai), `openrouter/`, `xinference/` (imported by openai), `task/` (async task adapters: `ali/`, `doubao/`, `gemini/`, `hailuo/`, `jimeng/`, `kling/`, `sora/`, `suno/`, `vertex/`, `vidu/`)
- Each provider subpackage typically contains: `adaptor.go`, `constants.go`, `dto.go`, plus provider-specific relay files (e.g. `relay-zhipu.go`, `image.go`).
- Key files: `relay/channel/adapter.go` (interface contract), `relay/channel/openai/adaptor.go`

**`dto/`:**
- Purpose: Data transfer objects — request/response structs parsed from client JSON and re-marshaled to upstream.
- Contains: `openai_request.go`, `claude.go`, `gemini.go`, `openai_response.go`, `openai_image.go`, `openai_responses_compaction_request.go`, `audio.go`, `embedding.go`, `rerank.go`, `task.go`, `midjourney.go`, `suno.go`, `video.go`, `playground.go`, `pricing.go`, `ratio_sync.go`, `channel_settings.go`, `request_common.go`, `realtime.go`, `error.go`, `notify.go`, `user_settings.go`, `values.go`, `sensitive.go`, `openai_video.go`, `openai_compaction.go`
- Key files: `dto/openai_request.go`, `dto/claude.go`, `dto/gemini.go`

**`types/`:**
- Purpose: Core type definitions independent of any single relay format.
- Contains: `relay_format.go` (RelayFormat enum), `error.go` (NewAPIError), `channel_error.go`, `price_data.go`, `request_meta.go`, `file_source.go`, `file_data.go`, `set.go`, `rw_map.go`
- Key files: `types/relay_format.go`, `types/error.go`

**`constant/`:**
- Purpose: Project-wide constants — channel types, API types, context keys, cache keys, env flags.
- Contains: `channel.go` (ChannelType* enum), `api_type.go` (APIType* enum), `context_key.go`, `cache_key.go`, `env.go`, `task.go`, `midjourney.go`, `finish_reason.go`, `multi_key_mode.go`, `endpoint_type.go`, `endpoint_defaults.go`, `azure.go`, `setup.go`, `waffo_pay_method.go`
- Key files: `constant/channel.go`, `constant/api_type.go`, `constant/context_key.go`

**`common/`:**
- Purpose: Shared utilities used across all layers — JSON codec, Redis, crypto, env, IP/CIDR, email, rate-limit limiter, disk cache, SSRF protection, gopool, system monitor.
- Contains: `json.go`, `redis.go`, `crypto.go`, `hash.go`, `env.go`, `init.go`, `ip.go`, `email.go`, `rate-limit.go`, `quota.go`, `database.go`, `constants.go`, `gin.go`, `utils.go`, `validate.go`, `verification.go`, `totp.go`, `ssrf_protection.go`, `request_body_limit.go`, `body_storage.go`, `disk_cache.go`, `gopool.go`, `go-channel.go`, `embed-file-system.go`, `model.go`, `page_info.go`, `node_identity.go`, `performance_config.go`, `pyro.go`, `pprof.go`, `sys_log.go`, `system_monitor*.go`, `str.go`, `copy.go`, `custom-event.go`, `audio.go`, `endpoint_type.go`, `endpoint_defaults.go`, `topup-ratio.go`, `url_validator.go`, `disk_cache_config.go`, `json_test.go`, `email_test.go`, `url_validator_test.go`
- Subpackage: `common/limiter/` (`limiter.go`, `lua/rate_limit.lua`)
- Key files: `common/json.go`, `common/constants.go`, `common/redis.go`, `common/init.go`

**`setting/`:**
- Purpose: Typed configuration domains loaded from `model.OptionMap`, each owning a slice of runtime settings.
- Contains (top-level): `auto_group.go`, `chat.go`, `midjourney.go`, `payment_creem.go`, `payment_stripe.go`, `payment_waffo.go`, `payment_waffo_pancake.go`, `rate_limit.go`, `sensitive.go`, `user_usable_group.go`
- Subpackages: `ratio_setting/` (group_ratio, model_ratio, cache_ratio, compact_suffix, exposed_cache), `model_setting/`, `operation_setting/`, `system_setting/`, `performance_setting/` (self-importing via `_` in main.go), `perf_metrics_setting/`, `billing_setting/`, `console_setting/`, `reasoning/`, `config/`
- Key files: `setting/ratio_setting/model_ratio.go`, `setting/ratio_setting/group_ratio.go`, `setting/operation_setting/*.go`

**`oauth/`:**
- Purpose: OAuth provider implementations + a registry that auto-registers providers via `init()`.
- Contains: `provider.go` (interface), `registry.go`, `types.go`, `github.go`, `discord.go`, `oidc.go`, `linuxdo.go`, `generic.go`
- Key files: `oauth/provider.go`, `oauth/registry.go`

**`i18n/`:**
- Purpose: Backend internationalization with `nicksnyder/go-i18n/v2`.
- Contains: `Init`, message catalogs, `locales/` (en, zh JSON)
- Key files: `i18n/` root files, `i18n/locales/`

**`logger/`:**
- Purpose: Custom leveled logger writing to `logs/` and stdout.
- Key files: `logger/` root files

**`pkg/`:**
- Purpose: Self-contained internal libraries with stable APIs, usable across layers.
- Contains: `billingexpr/` (tiered billing expression engine — read `pkg/billingexpr/expr.md` before changes), `cachex/` (hybrid in-memory/Redis cache with codec + namespacing), `ionet/` (io.net deployment client), `perf_metrics/` (relay sample recording + system metrics)
- Key files: `pkg/billingexpr/expr.md`, `pkg/cachex/hybrid_cache.go`, `pkg/perf_metrics/`

**`web/default/`:**
- Purpose: Default frontend theme — React 19 + TypeScript + Rsbuild + Base UI + Tailwind CSS.
- Contains: `package.json`, `rsbuild.config.*`, `tsconfig*.json`, `src/` (application code), `dist/` (build output, embedded by Go)
- Key dirs under `src/`: `routes/` (TanStack Router file-based routes), `features/` (feature modules), `components/` (`ui/`, `layout/`, `data-table/`, `ai-elements/`), `stores/` (Zustand), `hooks/`, `lib/`, `config/`, `context/`, `i18n/` (i18next + `locales/`), `styles/`, `assets/`

**`web/classic/`:**
- Purpose: Legacy frontend theme — React 18 + Vite + Semi Design. Build output embedded by Go.
- Note: Legacy synchronous log-delete route `/api/log` DELETE exists only for this theme (see `router/api-router.go:269` TODO).

**`electron/`:**
- Purpose: Electron desktop wrapper around the web frontend.

**`bin/`:**
- Purpose: SQL migration scripts (`migration_v0.2-v0.3.sql`, `migration_v0.3-v0.4.sql`) and shell helpers (`time_test.sh`).

## Key File Locations

**Entry Points:**
- `main.go`: Process bootstrap, Gin server setup, frontend embedding, background goroutines
- `router/main.go`: Wires all route groups via `SetRouter`
- `controller/relay.go:68` (`Relay`): Relay request orchestrator
- `controller/relay.go:486` (`RelayTask`): Async task orchestrator
- `web/default/src/routes/__root.tsx`: Frontend root route

**Configuration:**
- `.env.example`: Documented environment variables (existence only — do not read `.env`)
- `common/env.go` / `common/init.go`: Env loading + global flag initialization
- `model/main.go:30-51`: DB dialect-aware column/value helpers
- `setting/ratio_setting/`: Pricing ratio configuration
- `VERSION`: Semver string injected at build time into `common.Version`

**Core Logic:**
- `relay/relay_adaptor.go:54`: Adaptor factory (`GetAdaptor`)
- `relay/relay_adaptor.go:138`: Task adaptor factory (`GetTaskAdaptor`)
- `relay/channel/adapter.go`: Adaptor + TaskAdaptor interface contracts
- `relay/common/relay_info.go`: `RelayInfo` request context carrier
- `relay/helper/valid_request.go:21`: Request parse + validate dispatch
- `relay/helper/price.go`: Model price computation
- `service/billing.go` / `service/billing_session.go`: Billing session (pre-consume/settle/refund)
- `service/channel_select.go` / `service/channel_affinity.go`: Channel selection logic
- `middleware/distributor.go:32`: Channel distribution middleware
- `middleware/auth.go:303` (`TokenAuth`), `:181` (`UserAuth`), `:187` (`AdminAuth`), `:193` (`RootAuth`), `:199` (`RequirePermission`)
- `service/authz/enforcer.go`: Casbin enforcer init + model
- `model/channel_cache.go:26` (`InitChannelCache`): In-memory channel index
- `model/ability.go`: Group→Model→Channel ability entity

**Testing:**
- Go tests: co-located `*_test.go` next to the code under test (e.g. `relay/common/relay_info_test.go`, `service/task_billing_test.go`, `model/system_task_test.go`, `router/channel_router_test.go`)
- Migration SQL: `bin/migration_*.sql`
- Frontend tests: under `web/default/` (see `web/default/AGENTS.md`)

**Documentation:**
- `AGENTS.md`: Project conventions (read first when working in this repo)
- `CLAUDE.md`: Imports `AGENTS.md`
- `pkg/billingexpr/expr.md`: Billing expression engine design (read before touching billing expressions)
- `README*.md`: Multi-language READMEs (en, zh_CN, zh_TW, ja, fr)

## Naming Conventions

**Files:**
- Go: `snake_case.go` for most files; multi-word handler files use hyphens when they group a feature (`channel-router.go`, `channel-billing.go`, `channel-test.go`, `topup-stripe.go`, `subscription_payment_stripe.go`). Mixed style is historical — prefer `snake_case` for new files.
- Provider relay files inside `relay/channel/<provider>/` use `relay-<provider>.go` (e.g. `relay-zhipu.go`, `relay-xunfei.go`).
- Adaptor file is always `adaptor.go`; provider constants in `constants.go`; provider DTOs in `dto.go`.
- Tests: `*_test.go` co-located with the unit under test.
- Frontend (default theme): `kebab-case.tsx` / `kebab-case.ts` for files; route files use TanStack Router conventions (`$param.tsx` for dynamic segments, `_authenticated/` and `_route` prefixes for layout/pathless routes).

**Directories:**
- Go packages: lowercase, single word preferred (`controller`, `model`, `service`); provider packages match the provider slug (`relay/channel/openai/`, `relay/channel/aws/`).
- Setting subpackages: `<domain>_setting/` (e.g. `ratio_setting/`, `model_setting/`).
- Frontend feature dirs: `web/default/src/features/<feature>/` matching the route segment (e.g. `channels/`, `keys/`, `usage-logs/`).

**Identifiers:**
- Exported Go types/functions: `PascalCase` (`RelayInfo`, `GetAdaptor`, `TokenAuth`).
- Channel/API type constants: `ChannelType<Pascal>` / `APIType<Pascal>` (`constant/channel.go`, `constant/api_type.go`).
- Context keys: `ContextKey<Pascal>` (`constant/context_key.go`).
- Relay modes: `RelayMode<Pascal>` (`relay/constant/relay_mode.go`).

## Where to Add New Code

**New upstream AI provider (channel):**
1. Create `relay/channel/<provider>/` with `adaptor.go` (implement `channel.Adaptor` from `relay/channel/adapter.go`), `constants.go`, `dto.go`, and any `relay-<provider>.go` helper.
2. Add a channel-type constant to `constant/channel.go` (before `ChannelTypeDummy`) and a base URL to `ChannelBaseURLs`.
3. Add an `APIType<Pascal>` constant to `constant/api_type.go` and the channel→api-type mapping.
4. Add a `case constant.APIType<Pascal>: return &<provider>.Adaptor{}` to `GetAdaptor` in `relay/relay_adaptor.go:54`.
5. If the provider supports `StreamOptions`, register it in `streamSupportedChannels` (see `AGENTS.md` "Relay and provider behavior").
6. Tests: add `*_test.go` in the provider package.

**New async task provider (Midjourney/Suno/video-style):**
1. Create `relay/channel/task/<provider>/` implementing `channel.TaskAdaptor` (`relay/channel/adapter.go:34`).
2. Add a `case` to `GetTaskAdaptor` in `relay/relay_adaptor.go:138` keyed by `constant.ChannelType*`.
3. Wire the submit/fetch routes in `router/relay-router.go` (or `router/video-router.go`) calling `controller.RelayTask` / `controller.RelayTaskFetch`.

**New management API endpoint:**
1. Add the handler function in `controller/` (pick the existing file that owns the resource, e.g. `controller/channel.go` for channels; create a new file only for a distinct resource).
2. Register the route in the appropriate group in `router/api-router.go` (or `router/channel-router.go` for permission-gated channel routes — add to `channelPermissionRoutes` with an `authz.Permission`).
3. Apply the right middleware: `UserAuth` / `AdminAuth` / `RootAuth` / `RequirePermission` / `CriticalRateLimit` / `AnonymousRequestBodyLimit` as appropriate.
4. Add the DB method to `model/` (extend the relevant entity file).

**New business logic / service function:**
- Put it in `service/<area>.go` (e.g. billing logic in `service/billing*.go`, channel logic in `service/channel*.go`). Keep it importable by `controller` and `middleware`. Do NOT import `relay` from `service` — use the injected `service.GetTaskAdaptorFunc` for task adaptors.

**New GORM entity:**
1. Define the struct in `model/<entity>.go` with cross-DB-safe tags (no `AUTO_INCREMENT`/`SERIAL`, no `default:true` business-rule booleans, use `commonGroupCol`/`commonKeyCol` for reserved `group`/`key` columns).
2. Register it for auto-migration in `model/main.go` migration list.
3. Add cache methods in `model/<entity>_cache.go` if hot-path read caching is needed.

**New frontend route (default theme):**
1. Add `web/default/src/routes/<segment>/index.tsx` (or `web/default/src/routes/_authenticated/<segment>/index.tsx` for auth-required). Use `$param.tsx` for dynamic segments, `$section.tsx` for tabbed sub-routes.
2. Add a feature module under `web/default/src/features/<segment>/` for the page's components/hooks.
3. Add i18n keys as English source strings in `web/default/src/i18n/locales/{lang}.json` (use `bun run i18n:sync` from `web/default/`).
4. Re-run `bun run build` (or `bun run dev`) — `web/default/src/routeTree.gen.ts` is generated.

**New shared utility:**
- Backend: add to `common/<area>.go` (e.g. `common/utils.go`, `common/str.go`). Use `common.Marshal`/`Unmarshal` for any JSON.
- Frontend: add to `web/default/src/lib/`.

**New middleware:**
- Add `middleware/<name>.go` exposing a constructor func returning `gin.HandlerFunc` (match the `func() func(c *gin.Context)` shape used by `UserAuth`/`AdminAuth` where a closure captures config).
- Wire it in the relevant router file (`router/api-router.go`, `router/relay-router.go`, etc.).

**New setting / option:**
- Pick the owning subpackage under `setting/<domain>_setting/`; add the typed getter/setter and register the option key so it loads from `model.OptionMap`. The `setting/performance_setting/` package is blank-imported in `main.go:27` to register its init — follow that pattern if a setting package needs side-effecting registration.

## Special Directories

**`data/`:**
- Purpose: SQLite database file, on-disk cache files, and other runtime data.
- Generated: Yes (runtime)
- Committed: No (gitignored)

**`logs/`:**
- Purpose: Application log files written by `logger/`.
- Generated: Yes (runtime)
- Committed: No (gitignored)

**`web/default/dist/` and `web/classic/dist/`:**
- Purpose: Built frontend assets embedded into the Go binary via `//go:embed` (`main.go:39-49`). Must exist before `go build`.
- Generated: Yes (by `bun run build` / Vite)
- Committed: Yes (the embedded `dist/index.html` is read at compile time; rebuild before committing Go changes that depend on frontend updates)

**`web/default/src/routeTree.gen.ts`:**
- Purpose: Auto-generated TanStack Router route tree. Do not edit by hand.
- Generated: Yes (by the router plugin during `bun run dev`/`build`)
- Committed: Yes

**`.planning/`:**
- Purpose: GSD workflow artifacts — codebase maps, roadmaps, phase plans.
- Generated: Yes (by GSD commands)
- Committed: Optional (project's choice)

**`.agents/skills/` and `.trae/skills/`:**
- Purpose: Project-scoped agent skills (e.g. `vercel-react-best-practices`). Loaded as lightweight `SKILL.md` indexes, not full `AGENTS.md`.
- Generated: No
- Committed: Yes

**`bin/`:**
- Purpose: One-off SQL migration scripts and shell helpers, run manually during version upgrades.
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-07-01*
