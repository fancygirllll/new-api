# Coding Conventions

**Analysis Date:** 2026-07-01

## Project Context

- **Module:** `github.com/QuantumNous/new-api` (see [go.mod](file:///e:/AI%20Project/new-api/go.mod))
- **Language:** Go 1.25.1 (backend), React + Vite + TypeScript (frontend, embedded via `//go:embed`)
- **Framework:** Gin (`github.com/gin-gonic/gin` v1.9.1) + GORM (`gorm.io/gorm` v1.25.2)
- **Two frontend themes:** `web/default` (modern) and `web/classic` (legacy), both built with bun and embedded into the Go binary via `buildFS`/`classicBuildFS` in [main.go](file:///e:/AI%20Project/new-api/main.go)

## Naming Patterns

**Files:**
- `snake_case.go` for all Go source: `sys_log.go`, `url_validator.go`, `topup_stripe.go`, `channel_authz.go`
- Test files co-located with source as `*_test.go`: `token.go` + `token_test.go`, `url_validator.go` + `url_validator_test.go`
- Interface file uses the canonical spelling `adapter.go` at [relay/channel/adapter.go](file:///e:/AI%20Project/new-api/relay/channel/adapter.go) (note: directory is `relay/channel`, type is `Adaptor`)

**Packages:**
- Single lowercase word at top level: `common`, `controller`, `model`, `dto`, `types`, `constant`, `middleware`, `relay`, `service`, `router`, `setting`, `i18n`, `logger`, `oauth`, `pkg`
- Sub-packages use `snake_case`: `relay/channel/openai`, `service/authz`, `setting/operation_setting`, `setting/ratio_setting`, `pkg/billingexpr`, `pkg/perf_metrics`
- Import aliases: `relaycommon` for `relay/common`, `relayconstant` for `relay/constant`, `channelconstant` for `constant`, `perfmetrics` for `pkg/perf_metrics`

**Functions:**
- Exported: `PascalCase` (`GetAllTokens`, `ApiErrorI18n`, `ConvertOpenAIRequest`)
- Unexported: `camelCase` (`buildMaskedTokenResponse`, `detectLanguage`, `isTemperatureOneOnlyModel`)
- Constructors: `NewXxx` (`NewError`, `NewOpenAIError`, `NewLocalizer`)
- HTTP handlers are exported free functions in `controller/` taking `*gin.Context`

**Types/Structs:**
- `PascalCase`: `Token`, `Channel`, `RelayInfo`, `PageInfo`, `NewAPIError`, `OpenAIError`
- DTO structs in `dto/` mirror external API shapes (`GeneralOpenAIRequest`, `ClaudeRequest`, `GeminiChatRequest`)
- Model structs in `model/` use GORM tags + JSON tags (see [model/token.go](file:///e:/AI%20Project/new-api/model/token.go))

**Constants:**
- Grouped in `const (...)` blocks, `PascalCase`
- i18n message keys are namespaced string constants in [i18n/keys.go](file:///e:/AI%20Project/new-api/i18n/keys.go): `MsgInvalidParams = "common.invalid_params"`, `MsgAuthNotLoggedIn = "auth.not_logged_in"`
- Error codes in [types/error.go](file:///e:/AI%20Project/new-api/types/error.go) as typed `ErrorCode` constants: `ErrorCodeInvalidRequest`, `ErrorCodeChannelNoAvailableKey`

**Context Keys:**
- Typed `constant.ContextKey string` (see [constant/context_key.go](file:///e:/AI%20Project/new-api/constant/context_key.go)) — never use raw string keys
- Constants prefixed by domain: `ContextKeyToken...`, `ContextKeyChannel...`, `ContextKeyUser...`

## Code Style

**Formatting:**
- `gofmt` / `goimports` defaults (no `.golangci.yml` or `.editorconfig` present at repo root)
- Tabs for indentation, no trailing whitespace

**Linting:**
- No golangci-lint config in the repo. CI focuses on build + tests.
- Frontend (web/default) uses ESLint (`DISABLE_ESLINT_PLUGIN='true'` is set during Docker build per [Dockerfile](file:///e:/AI%20Project/new-api/Dockerfile))

**Generics:**
- Generics used where they reduce boilerplate:
  - `common.GetContextKeyType[T any]` in [common/gin.go](file:///e:/AI%20Project/new-api/common/gin.go)
  - `common.GetPointer[T any](v T) *T` in [common/utils.go](file:///e:/AI%20Project/new-api/common/utils.go)

## Import Organization

Standard Go grouping with blank lines between groups (run `goimports`):

```go
import (
    // 1. Standard library
    "errors"
    "fmt"
    "net/http"

    // 2. Third-party
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"

    // 3. Internal (github.com/QuantumNous/new-api/...)
    "github.com/QuantumNous/new-api/common"
    "github.com/QuantumNous/new-api/constant"
    "github.com/QuantumNous/new-api/model"
)
```

**Blank imports for side effects:**
- `_ "github.com/QuantumNous/new-api/setting/performance_setting"` in [main.go](file:///e:/AI%20Project/new-api/main.go) (runs package `init()`)
- `_ "github.com/QuantumNous/new-api/oauth"` in [router/api-router.go](file:///e:/AI%20Project/new-api/router/api-router.go) (registers OAuth providers)
- `_ "net/http/pprof"` in [main.go](file:///e:/AI%20Project/new-api/main.go)

**Path Aliases:**
- `relaycommon` → `relay/common`
- `relayconstant` → `relay/constant`
- `channelconstant` → `constant` (when `constant` would clash)
- `perfmetrics` → `pkg/perf_metrics`

## Error Handling

**Central error type:** `types.NewAPIError` ([types/error.go](file:///e:/AI%20Project/new-api/types/error.go)) carries `Err`, `RelayError`, `errorType`, `errorCode`, `StatusCode`, `Metadata`, plus internal `skipRetry`/`recordErrorLog` flags.

**Constructors (use these, do not build the struct directly):**
- `types.NewError(err, errorCode, opts...)` — generic, preserves wrapped `*NewAPIError` via `errors.As`
- `types.NewOpenAIError(err, errorCode, statusCode, opts...)` — wraps as OpenAI-style error
- `types.WithOpenAIError(openAIError, statusCode, opts...)` — from a parsed `OpenAIError`
- `types.WithClaudeError(claudeError, statusCode, opts...)` — from a parsed `ClaudeError`
- `types.NewErrorWithStatusCode(err, errorCode, statusCode, opts...)`
- `types.InitOpenAIError(errorCode, statusCode, opts...)` — no underlying err yet

**Functional options** (`NewAPIErrorOptions`):
- `types.ErrOptionWithSkipRetry()` — mark as non-retryable
- `types.ErrOptionWithNoRecordErrorLog()` — suppress error log persistence
- `types.ErrOptionWithStatusCode(int)`
- `types.ErrOptionWithHideErrMsg(replaceStr)` — replace message (debug-logged when `common.DebugEnabled`)

**Controller response helpers** (in [common/gin.go](file:///e:/AI%20Project/new-api/common/gin.go)):
- `common.ApiSuccess(c, data)` → `{"success":true,"message":"","data":...}` HTTP 200
- `common.ApiError(c, err)` → `{"success":false,"message":err.Error()}` HTTP 200
- `common.ApiErrorMsg(c, msg)` → `{"success":false,"message":msg}` HTTP 200
- `common.ApiErrorI18n(c, key, args...)` → translated message (preferred for user-facing errors)
- `common.ApiSuccessI18n(c, key, data, args...)` → translated success message

**API envelope convention:** All `/api/*` JSON responses return HTTP 200 with `{success, message, data}`. Do NOT use HTTP 4xx for application errors on the `/api` group. Relay (OpenAI-compatible) endpoints may return proper HTTP status codes via `*types.NewAPIError`.

**Error wrapping:**
- Use `errors.Is` / `errors.As` (the type implements `Unwrap()`)
- Wrap with `fmt.Errorf("...: %w", err)` or `github.com/pkg/errors` (`errors.Wrap`, `errors.New`)
- Sentinel errors: `common.ErrRequestBodyTooLarge`, `model.ErrDatabase`, `model.ErrUserEmptyCredentials`

**Sensitive data masking:**
- `common.MaskSensitiveInfo(str)` strips keys/secrets from error messages
- `NewAPIError.MaskSensitiveError()` applied before returning to clients
- Exception: `ErrorCodeCountTokenFailed` is returned raw (debugging value)

## Logging

**Framework:** Custom thin layer over `gin.DefaultWriter` / `gin.DefaultErrorWriter` (see [common/sys_log.go](file:///e:/AI%20Project/new-api/common/sys_log.go) and [logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go)).

**Functions:**
- `common.SysLog(s string)` — `[SYS] timestamp | msg` to stdout
- `common.SysError(s string)` — `[SYS] timestamp | msg` to stderr
- `common.FatalLog(v ...any)` — `[FATAL] ...` then `os.Exit(1)`
- `logger.LogInfo(ctx, msg)` / `logger.LogWarn(ctx, msg)` / `logger.LogError(ctx, msg)` — `[LEVEL] timestamp | request_id | msg`
- `logger.LogDebug(ctx, msg, args...)` — only when `common.DebugEnabled == true` (printf-style)
- `logger.LogJson(ctx, msg, obj)` — debug-only JSON dump

**Patterns:**
- Always acquire `common.LogWriterMu.RLock()` when writing through `gin.DefaultWriter`/`gin.DefaultErrorWriter` (writer can be swapped during log rotation). `SysLog`/`SysError`/`FatalLog` already do this.
- Log file rotation: [logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go) `SetupLogger()` rotates at `maxLogCount = 1000000` lines and on startup; protected by `setupLogLock`.
- Truncate large bodies: relay error handler truncates upstream response bodies to `common.LocalLogContentLimit` with `[truncated]` + `original_length=N` markers (see [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go) for the contract).
- Include request id: `logger` pulls `common.RequestIdKey` from context; `middleware.RequestId()` sets it.
- Never log raw secrets/keys — use `common.MaskSensitiveInfo` or `Token.GetMaskedKey()`.

## i18n Conventions

**Backend (Go):**
- Library: `github.com/nicksnyder/go-i18n/v2` + `golang.org/x/text/language` (see [i18n/i18n.go](file:///e:/AI%20Project/new-api/i18n/i18n.go))
- Locale files embedded: `i18n/locales/{en,zh-CN,zh-TW}.yaml` via `//go:embed locales/*.yaml`
- **Default language:** `en` (fallback when unsupported)
- Message keys are constants in [i18n/keys.go](file:///e:/AI%20Project/new-api/i18n/keys.go), namespaced by domain:
  - `common.*`, `auth.*`, `token.*`, `redemption.*`, `user.*`, `quota.*`, etc.
- **Translate at the boundary:** use `i18n.T(c, key, args...)` or `common.ApiErrorI18n(c, i18n.MsgXxx)`. Never hardcode user-facing English in handlers.
- Language detection order (see [middleware/i18n.go](file:///e:/AI%20Project/new-api/middleware/i18n.go) + `GetLangFromContext`):
  1. User setting (`dto.UserSetting.Language` set by `TokenAuth`)
  2. Lazy-loaded user language from DB/cache (`i18n.SetUserLangLoader`)
  3. `Accept-Language` header (parsed by `i18n.ParseAcceptLanguage`)
  4. Default `en`
- `common.TranslateMessage` is a `var` function set by `i18n.Init()` to avoid a `common`→`i18n` import cycle. **Do not replace it after init.**
- `i18n.IsSupported(lang)`, `i18n.NormalizeLang(lang)`, `i18n.SupportedLanguages()` are the only supported-language gates.
- YAML key format: `domain.message_id: "Text with {{.Var}} placeholder"` (see [i18n/locales/en.yaml](file:///e:/AI%20Project/new-api/i18n/locales/en.yaml)).

**Frontend (React):**
- Separate i18n system in `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json` (six locales) under a flat `"translation"` object
- Keys are English source strings; `en.json` is the base, `zh` is fallback
- **NEVER edit locale JSON directly** — all writes go through the `add-missing-keys.mjs` script then `bun run i18n:sync` (enforced by the `i18n-translate` skill at [.agents/skills/i18n-translate/SKILL.md](file:///e:/AI%20Project/new-api/.agents/skills/i18n-translate/SKILL.md))
- Run from `web/default/`: `bun run i18n:sync` normalizes ordering and writes `_reports/_sync-report.json`

## Comments

**When to Comment:**
- Chinese comments acceptable for domain notes (`// 热更新配置`, `// 兼容`); English for public API/interface contracts
- Every exported function in `types/error.go`, `i18n/i18n.go`, `relay/channel/adapter.go` has an English doc comment
- `// TODO implement me` / `// not supported` markers for unimplemented adaptor methods (e.g., [relay/channel/moonshot/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor.go))

**JSDoc/TSDoc:**
- Go uses standard `// FunctionName ...` doc comments on exported identifiers
- No formal TSDoc convention enforced in frontend

## Function Design

**Size:** No hard limit; controllers often 50-150 lines. Extract helpers (`buildMaskedTokenResponse`, `decodeAPIResponse`) when logic is reused.

**Parameters:**
- Handlers always take `c *gin.Context` as the sole parameter
- Adaptor methods take `(c *gin.Context, info *relaycommon.RelayInfo, req *dto.XxxRequest)` (see [relay/channel/adapter.go](file:///e:/AI%20Project/new-api/relay/channel/adapter.go))
- Prefer `*T` for optional fields (e.g., `*string` for `Token.AllowIps`) and use `common.GetPointer[T](v)` to take addresses

**Return Values:**
- `(result, error)` standard Go
- Relay adaptors return `(any, error)` from `Convert*` and `(usage any, err *types.NewAPIError)` from `DoResponse` (typed error, not `error`)
- Controller helpers do not return — they write directly to `c` via `common.ApiSuccess`/`common.ApiErrorI18n`

## Module Design

**Exports:**
- One package per concern; no barrel files
- `controller/` exports handler functions, not types
- `model/` exports GORM model structs + data-access functions
- `dto/` exports request/response DTOs and `Validate()` methods
- `types/` exports cross-cutting types (`NewAPIError`, `RelayFormat`, `PriceData`)

**Init pattern:**
- Packages that need startup use `func init()` (e.g., `common.Validate`, `common.TranslateMessage` default)
- `i18n.Init()` is idempotent via `sync.Once` (`initOnce`)
- `model.InitDB()`, `model.InitOptionMap()`, `model.InitChannelCache()` called explicitly from `main.InitResources()` in dependency order

**Adaptor interface:** Every upstream provider implements `channel.Adaptor` (and optionally `TaskAdaptor` for async tasks) in `relay/channel/<provider>/adaptor.go`. New providers go in `relay/channel/<provider>/` and register via the central adaptor factory.

## Response & Pagination Conventions

**Envelope:** `{"success": bool, "message": string, "data": any}` at HTTP 200 for `/api/*`.

**Pagination:** Use `common.PageInfo` ([common/page_info.go](file:///e:/AI%20Project/new-api/common/page_info.go)):
```go
pageInfo := common.GetPageQuery(c)   // reads ?p=&page_size=&ps=&size=
items, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
pageInfo.SetTotal(int(total))
pageInfo.SetItems(items)
common.ApiSuccess(c, pageInfo)
```
- `page_size` capped at 100, defaults to `common.ItemsPerPage` (10)
- `GetStartIdx() = (Page-1) * PageSize`

## Concurrency Conventions

- **Goroutines:** use `gopool.Go(func(){...})` from `github.com/bytedance/gopkg/util/gopool` (managed pool, not raw `go`). See [main.go](file:///e:/AI%20Project/new-api/main.go) and [logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go).
- **Shared maps:** guard with `sync.RWMutex` (e.g., `common.OptionMapRWMutex`, `i18n.mu`)
- **Atomic singletons:** `atomic.Value` for `themeValue` in [common/constants.go](file:///e:/AI%20Project/new-api/common/constants.go)
- **Once:** `sync.Once` for idempotent init (`i18n.initOnce`)
- **Context cancellation:** pass `context.Context` to long-running ops; respect `ctx.Done()`

## Configuration Conventions

- **Env vars:** read via `common.GetEnvOrDefault(name, default)`, `GetEnvOrDefaultString`, `GetEnvOrDefaultBool` in [common/env.go](file:///e:/AI%20Project/new-api/common/env.go). All env initialization goes in `common.InitEnv()` + `common.initConstantEnv()` ([common/init.go](file:///e:/AI%20Project/new-api/common/init.go)).
- **Flags:** `flag.Int("port", ...)`, `flag.String("log-dir", ...)` defined in [common/init.go](file:///e:/AI%20Project/new-api/common/init.go)
- **`.env` file:** loaded via `github.com/joho/godotenv` in `main.InitResources()` — `.env.example` documents available vars (do not read `.env` contents)
- **Runtime settings:** stored in DB via `model.OptionMap` and exposed through `setting/*` packages (e.g., `operation_setting`, `ratio_setting`)

---

*Convention analysis: 2026-07-01*
