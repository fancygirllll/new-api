# Technology Stack

**Analysis Date:** 2026-07-01

## Languages

**Primary:**
- Go 1.25.1 (declared in `go.mod`; Docker build uses `golang:1.26.1-alpine` in `Dockerfile` line 23) — backend API gateway, relay engine, billing, scheduled tasks, payment webhooks. Module path: `github.com/QuantumNous/new-api` (`go.mod` line 1).

**Secondary:**
- TypeScript — frontend apps (`web/default/`, `web/classic/`). Configs: `web/default/tsconfig.json`, `web/default/tsconfig.app.json`, `web/default/tsconfig.node.json`.
- JavaScript — Electron desktop wrapper (`electron/main.js`, `electron/preload.js`).
- YAML — i18n locale bundles (`i18n/locales/zh-CN.yaml`, `zh-TW.yaml`, `en.yaml`).
- Dockerfile / docker-compose — container build & orchestration.

## Runtime

**Environment:**
- Go runtime (single static binary; `CGO_ENABLED=0` build in `Dockerfile` line 24; experimental `greenteagc` GC via `GOEXPERIMENT=greenteagc` line 29).
- Production container base: `debian:bookworm-slim` (`Dockerfile` line 41).
- Optional Node.js/Bun toolchain only at frontend build stage (`oven/bun:1` builder stages, `Dockerfile` lines 1 & 12).
- Electron 39.8.5 runtime for the desktop distribution (`electron/package.json` line 28).

**Package Manager:**
- Go modules — `go.mod` / `go.sum` (lockfile present).
- Bun — `web/bun.lock` (frontend monorepo workspace, `web/package.json` lines 3-5).
- npm — `electron/package-lock.json` (desktop app only).

## Frameworks

**Core (Backend):**
- Gin `v1.9.1` — HTTP framework. Setup in `main.go` lines 167-203; routes registered via `router/main.go` → `SetApiRouter`, `SetRelayRouter`, `SetVideoRouter`, `SetWebRouter`.
- GORM `v1.25.2` — ORM. Initialized in `model/main.go` `InitDB()` (line 181); drivers: mysql `v1.4.3`, postgres `v1.5.2`, clickhouse `v0.6.0`, sqlite via `glebarez/sqlite v1.9.0`.
- Casbin `v2.135.0` — RBAC authorization. Initialized in `main.go` line 291 (`authz.Init`); policy sync started at `main.go` line 105.

**Core (Frontend Default — `web/default`):**
- React `^19.2.6` (`web/default/package.json` line 60).
- TanStack Router `^1.170.8` + Router Plugin (file-based routes in `web/default/src/routes/`).
- TanStack Query `^5.100.14`, TanStack Table `^8.21.3`, TanStack Virtual `^3.13.25`.
- Tailwind CSS `^4.3.0` + `@base-ui/react` + shadcn `^4.8.0`.
- Rsbuild `^2.0.7` (build/dev server; config: `web/default/rsbuild.config.ts`).
- Vercel AI SDK `ai ^6.0.191` + `sse.js` (streaming chat).

**Core (Frontend Classic — `web/classic`):**
- React `^19.2.6` (`web/classic/package.json` line 26).
- Semi UI `@douyinfe/semi-ui ^2.69.1` (component library).
- React Router DOM `^6.3.0`.
- Rsbuild `^2.0.7`.

**Desktop:**
- Electron `39.8.5` + `electron-builder ^26.7.0` (`electron/package.json` lines 28-29).

**Testing:**
- `github.com/stretchr/testify v1.11.1` — Go assertions (e.g. `model/usedata_flow_test.go`, `relay/channel/openai/chat_via_responses_test.go`).

**Build/Dev:**
- Bun — frontend install & build (`makefile` `build-frontend` target).
- `make` — orchestrates dev stack (`makefile`: `dev`, `dev-api`, `dev-web`).
- Docker Compose — `docker-compose.yml` (production-like), `docker-compose.dev.yml` (dev DB services).
- `go:embed` — frontend `dist/` embedded into binary (`main.go` lines 39-49).

## Key Dependencies

**Critical (Backend):**
- `github.com/gin-gonic/gin v1.9.1` — HTTP server & middleware pipeline.
- `gorm.io/gorm v1.25.2` — all DB access (`model/main.go`).
- `github.com/go-redis/redis/v8 v8.11.5` — cache & distributed rate limit (`common/redis.go`).
- `github.com/casbin/casbin/v2 v2.135.0` — role/permission enforcement (`service/authz/`).
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.50.4` — AWS Bedrock relay (`relay/channel/aws/adaptor.go`).
- `github.com/stripe/stripe-go/v81 v81.4.0` — Stripe checkout & webhooks (`controller/topup_stripe.go`).
- `github.com/Calcium-Ion/go-epay v0.0.4` — EPay gateway (`controller/topup.go`).
- `github.com/waffo-com/waffo-pancake-sdk-go v0.3.1` — Waffo Pancake hosted checkout (`service/waffo_pancake.go`).
- `github.com/go-webauthn/webauthn v0.14.0` — Passkey/WebAuthn login (`controller/passkey.go`).
- `github.com/pquerna/otp v1.5.0` — TOTP 2FA (`model/twofa.go`).
- `github.com/golang-jwt/jwt/v5 v5.3.0` — Vertex AI service-account JWTs (`relay/channel/vertex/service_account.go`).
- `github.com/gorilla/websocket v1.5.0` — OpenAI Realtime API relay (`relay/websocket.go`).
- `github.com/tidwall/gjson v1.18.0` + `sjson v1.2.5` — JSON path read/write for upstream payloads.
- `github.com/tiktoken-go/tokenizer v0.6.2` — token counting for billing (`service/tokenizer.go`).
- `github.com/shopspring/decimal v1.4.0` — money/quota math (`controller/topup_waffo_pancake.go`, `service/billing.go`).
- `github.com/expr-lang/expr v1.17.8` — billing expression evaluation (`relay/helper/billing_expr_request.go`).
- `github.com/anknown/ahocorasick v0.0.0-...` — sensitive-word matching (`service/sensitive.go`).
- `github.com/bytedance/gopkg v0.1.3` — goroutine pool (`gopool`) used across background tasks.
- `github.com/nicksnyder/go-i18n/v2 v2.6.1` — server-side localization (`i18n/i18n.go`).
- `github.com/joho/godotenv v1.5.1` — `.env` loading (`main.go` line 266).
- `github.com/grafana/pyroscope-go v1.2.7` — continuous profiling (`common/pyro.go`).
- `github.com/samber/lo v1.52.0` + `samber/hot v0.11.0` — functional helpers & hot cache.

**Audio Decoding (for multimodal token counting):**
- `github.com/go-audio/wav v1.1.0`, `go-audio/aiff v1.1.0`, `jfreymuth/oggvorbis v1.0.5`, `mewkiz/flac v1.0.13`, `tcolgate/mp3`, `abema/go-mp4 v1.4.1`, `yapingcat/gomedia`.

**Infrastructure (Backend):**
- `github.com/gin-contrib/cors v1.7.2`, `gzip v0.0.6`, `sessions v0.0.5` (cookie store), `static v0.0.1`.
- `github.com/go-playground/validator/v10 v10.20.0` — request validation.
- `github.com/google/uuid v1.6.0`.
- `github.com/thanhpk/randstr v1.0.6` — trade IDs / verification codes.
- `github.com/jinzhu/copier v0.4.0` — struct copying.
- `github.com/Azure/go-ntlmssp v0.1.1` — NTLM SMTP auth (`common/email_ntlm_auth.go`).
- `github.com/shirou/gopsutil v3.21.11+incompatible` — system monitor (`common/system_monitor.go`).
- Prometheus client libs (indirect, via pyroscope) — metrics.

**Critical (Frontend Default):**
- `@tanstack/react-router` `^1.170.8` — file-based routing (`web/default/src/routes/`).
- `@tanstack/react-query` `^5.100.14` — server state.
- `tailwindcss` `^4.3.0` + `tw-animate-css` + `tailwind-merge` — styling.
- `@base-ui/react` `^1.5.0` + `shadcn` `^4.8.0` — unstyled component primitives.
- `@visactor/vchart` `^2.0.22` — charts (`dashboard`, `usage-logs`).
- `@lobehub/icons` `^5.10.0` — model/vendor icon set.
- `ai` `^6.0.191` + `sse.js` `^2.8.0` — streaming chat client (`playground`).
- `react-hook-form` `^7.76.1` + `@hookform/resolvers` + `zod` `^4.4.3` — forms.
- `zustand` `^5.0.13` — client stores (`web/default/src/stores/`).
- `i18next` `^26.2.0` + `react-i18next` `^17.0.8` — localization.
- `@codemirror/*` — markdown editor; `marked`, `shiki`, `katex` — rendering.
- `axios` (catalog) — API client.

## Configuration

**Environment:**
- `.env` file loaded via `godotenv.Load(".env")` in `main.go` line 266 (existence noted; contents never read by tooling).
- Process env vars parsed in `common/init.go` `InitEnv()` (line 31) and `common/constants.go`.
- Runtime options mirrored into DB-backed `OptionMap` (`model/option.go` `InitOptionMap` line 30) — admin-editable via `/api/option` (root only).

**Key configs required:**
- `SQL_DSN` — primary DB connection (MySQL/PostgreSQL/SQLite; `model/main.go` `chooseDB` line 127).
- `LOG_SQL_DSN` — optional separate log DB (supports ClickHouse; `model/main.go` line 222).
- `REDIS_CONN_STRING` — optional Redis (`common/redis.go` line 25).
- `SESSION_SECRET` — cookie store secret (`common/init.go` line 49; random fallback warning if unset).
- `CRYPTO_SECRET` — token encryption secret (defaults to `Session_SECRET`; `common/init.go` line 59).
- `NODE_TYPE=slave` — designates non-master node (skips migrations & cron; `common/init.go` line 84).
- `NODE_NAME` — node identity for audit logs (`common/constants.go` line 168).
- `SQLITE_PATH` — SQLite file location (default `one-api.db?_busy_timeout=30000`; `common/database.go` line 44).

**Build config files:**
- `Dockerfile`, `Dockerfile.dev` — multi-stage container build.
- `docker-compose.yml`, `docker-compose.dev.yml` — orchestration.
- `makefile` — dev/build targets.
- `new-api.service` — systemd unit.
- `web/default/rsbuild.config.ts` — frontend bundler config (dev proxy `/api`, `/mj`, `/pg` → `localhost:3000`).
- `web/default/tsconfig*.json`, `web/default/.oxlintrc.json` — TS/lint config.
- `electron/package.json` `build` block — electron-builder targets (mac/win/linux).

## Platform Requirements

**Development:**
- Go 1.25+ toolchain (`go.mod` line 4).
- Bun (frontend; `makefile` uses `bun install`).
- Docker (optional, for `make dev-api` DB services).
- Ports: backend `3000` (default, `common/init.go` line 18), default frontend dev `5173`, classic frontend dev `5174` (`makefile` lines 4-5).

**Production:**
- Single static binary (`new-api`), container image `calciumion/new-api:latest`, or Electron desktop bundles (`electron-builder` mac/win/linux).
- Default listen port `3000` (`Dockerfile` line 50 `EXPOSE 3000`).
- Data volume `/data` (`Dockerfile` line 51).
- External: relational DB (MySQL 8+ / PostgreSQL 15+ / SQLite), optional Redis, optional ClickHouse for logs.
- Multi-instance supported via `NODE_TYPE` master/slave + `SESSION_SECRET` sharing (`docker-compose.yml` comments lines 41-42).

---

*Stack analysis: 2026-07-01*
