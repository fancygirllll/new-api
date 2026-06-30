# Codebase Concerns

**Analysis Date:** 2026-07-01

## Tech Debt

**Unimplemented relay adaptor methods (100+ stubs):**
- Issue: Nearly every channel adaptor under `relay/channel/*/adaptor.go` contains `//TODO implement me` stubs for optional protocol conversions (`ConvertClaudeRequest`, `ConvertGeminiRequest`, `ConvertAudioRequest`, `ConvertImageRequest`, `ConvertEmbeddingRequest`, `ConvertOpenAIResponsesRequest`, `ConvertRerankRequest`). Some stubs return `errors.New("not implemented")`, but several panic instead.
- Files: [zhipu/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/zhipu/adaptor.go), [baidu/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/baidu/adaptor.go), [deepseek/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/deepseek/adaptor.go), [xunfei/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/xunfei/adaptor.go), [cohere/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/cohere/adaptor.go), [dify/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/dify/adaptor.go), [mokaai/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/mokaai/adaptor.go), [cloudflare/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/cloudflare/adaptor.go), [mistral/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/mistral/adaptor.go), [claude/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/claude/adaptor.go), [aws/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/aws/adaptor.go), [moonshot/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor.go), [vertex/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/vertex/adaptor.go), [tencent/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/tencent/adaptor.go)
- Impact: Feature surface is inconsistent across channels. Some stubs `panic("implement me")` (e.g. `relay/channel/zhipu/adaptor.go:28`, `relay/channel/jina/adaptor.go:30`, `relay/channel/xunfei/adaptor.go:28`, `relay/channel/palm/adaptor.go:28`, `relay/channel/baidu/adaptor.go:29`, `relay/channel/cohere/adaptor.go:28`) which crashes the request goroutine if the code path is ever hit. The panic is caught by `middleware/recover.go` and `main.go:168` CustomRecovery, but it returns a 500 to the client instead of a clean "not supported" error.
- Fix approach: Standardize all stubs to return `errors.New("channel type X does not support Y")` via a shared `channel.ErrNotSupported` sentinel. Remove every `panic("implement me")`. Audit which stubs should return real errors (so the router can 404/501) vs which should be unreachable (guard at distributor level).

**Fragile string-matching error classification in billing:**
- Issue: Subscription funding errors are classified by substring matching on the error message string instead of typed sentinel errors.
- Files: [service/billing_session.go:216-220](file:///e:/AI%20Project/new-api/service/billing_session.go#L216-L220)
- Impact: If a model-layer error message wording changes (e.g. "no active subscription" → "subscription inactive"), the billing path silently misclassifies the error and returns a generic `ErrorCodeUpdateDataError` with retry enabled, instead of the intended `ErrorCodeInsufficientUserQuota` with skip-retry. Users may be charged incorrectly or retried when they shouldn't be.
- Fix approach: Define `model.ErrNoActiveSubscription` and `model.ErrSubscriptionQuotaInsufficient` sentinel errors (already noted in the in-code TODO). Have the funding layer wrap/return them. Replace `strings.Contains(errMsg, ...)` with `errors.Is(err, model.ErrNoActiveSubscription)`.

**Dual-frontend maintenance burden (classic + default):**
- Issue: The repo ships two complete React frontends (`web/default/` and `web/classic/`), both embedded into the binary at `main.go:39-49`. Classic is in deprecation; multiple TODOs reference removal.
- Files: [main.go:45-49](file:///e:/AI%20Project/new-api/main.go#L45-L49), [controller/log.go:156](file:///e:/AI%20Project/new-api/controller/log.go#L156), [router/api-router.go:268](file:///e:/AI%20Project/new-api/router/api-router.go#L268)
- Impact: Doubles frontend maintenance. Legacy routes and handlers are kept alive solely for classic, e.g. `controller/log.go` keeps a `/api/log/...` handler with an explicit "remove once the classic frontend is removed" TODO. Any UI bug fix must be applied twice.
- Fix approach: Pick a removal milestone for classic. At that milestone delete `web/classic/`, remove `classicBuildFS`/`classicIndexPage` embedding and the `SetTheme("classic")` code path in `common/constants.go`, and delete the classic-only API handlers flagged by the TODOs.

**God files exceeding 1000 lines:**
- Issue: Several source files have grown to sizes that make them hard to review, test, and extend safely.
- Files: [controller/channel.go](file:///e:/AI%20Project/new-api/controller/channel.go) (1960 lines), [relay/common/override.go](file:///e:/AI%20Project/new-api/relay/common/override.go) (1937 lines), [relay/channel/gemini/relay-gemini.go](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go) (1652 lines), [controller/user.go](file:///e:/AI%20Project/new-api/controller/user.go) (1332 lines), [model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go) (1282 lines), [model/channel.go](file:///e:/AI%20Project/new-api/model/channel.go) (1014 lines), [dto/openai_request.go](file:///e:/AI%20Project/new-api/dto/openai_request.go) (970 lines), [relay/channel/claude/relay-claude.go](file:///e:/AI%20Project/new-api/relay/channel/claude/relay-claude.go) (942 lines), [service/convert.go](file:///e:/AI%20Project/new-api/service/convert.go) (938 lines), [service/channel_affinity.go](file:///e:/AI%20Project/new-api/service/channel_affinity.go) (923 lines)
- Impact: High cognitive load for any change. Merge conflicts are frequent. Hard to write focused unit tests. `relay/common/override.go` is the worst offender at 1937 lines of override/transform logic with deeply nested branches and `err = nil` swallows at lines 918, 935, 949.
- Fix approach: Split by concern — `controller/channel.go` into `channel_crud.go`, `channel_test.go`, `channel_models.go`; `override.go` into per-request-type override files. Do this only when touching the file for another reason (do not refactor cold).

**Dangling migration TODO:**
- Issue: Commented-out `ALTER TABLE channels MODIFY model_mapping TEXT;` left in the SQLite/MySQL migration path with a "delete this line when most users have upgraded" TODO.
- Files: [model/main.go:211](file:///e:/AI%20Project/new-api/model/main.go#L211)
- Impact: Dead migration code; unclear when "most users have upgraded" is satisfied. Clutters the migration bootstrap and risks confusion for future maintainers.
- Fix approach: Define a version cutoff in `common.Version` history and delete the line once that cutoff ships, or convert it to a guarded one-time migration with a recorded-applied flag.

**Ununified api_version context dispatch:**
- Issue: `middleware/distributor.go` has a `switch channel.Type` that sets context keys (`api_version`, `region`, `plugin`, `bot_id`) from the overloaded `channel.Other` field. The field name `Other` is overloaded to mean different things per channel type.
- Files: [middleware/distributor.go:485-503](file:///e:/AI%20Project/new-api/middleware/distributor.go#L485-L503)
- Impact: Adding a new channel type that needs its own version field requires editing this switch. The `channel.Other` field is untyped `string` — no validation, no documentation of which channel types expect what.
- Fix approach: Replace `channel.Other` with a typed `ChannelSettings` struct field (or a small map) and move the context-key mapping into each adaptor's `Init()` so the distributor no longer needs the switch.

## Known Bugs

**Channel billing does not support Azure:**
- Symptoms: `UpdateAllChannelsBalance` skips Azure channel balance updates — the TODO at `controller/channel-billing.go:466` shows the type filter is commented out.
- Files: [controller/channel-billing.go:466-469](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L466-L469)
- Trigger: Any Azure channel; balance will never be reported/auto-banned on low balance.
- Workaround: Manual balance monitoring. Fix: implement Azure balance fetch or document that Azure is unsupported and exclude it explicitly rather than via dead code.

**`UpdateAllChannelsBalance` blocks the request:**
- Symptoms: The HTTP handler runs the full synchronous balance-update loop inline.
- Files: [controller/channel-billing.go:484-486](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L484-L486)
- Trigger: Admin triggers "update all channel balances". With many channels this times out the HTTP request.
- Workaround: None. Fix: dispatch via the system-task runner (already available at `service.StartSystemTaskRunner`) instead of running inline.

## Security Considerations

**Session cookie `Secure: false`:**
- Risk: Session cookies are transmitted over plain HTTP, allowing interception on non-TLS networks/misconfigured proxies.
- Files: [main.go:189](file:///e:/AI%20Project/new-api/main.go#L189)
- Current mitigation: `SameSite: http.SameSiteStrictMode` and `HttpOnly: true` are set.
- Recommendations: Default `Secure` to `true` for production (`GIN_MODE=release`); allow override via env (`SESSION_COOKIE_SECURE`) for local dev on HTTP.

**Silent random `SessionSecret` / `CryptoSecret` on restart:**
- Risk: If `SESSION_SECRET` is unset, `common.SessionSecret` defaults to a random UUID generated in `init()` at `common/constants.go:75`. There is no fatal warning for the unset case — only the literal string `"random_string"` triggers `log.Fatal` at `common/init.go:51-54`. Consequences: (1) sessions do not persist across restarts; (2) in multi-instance deployments each instance generates a different secret, so a user's session cookie is invalid on other instances behind the load balancer; (3) `CryptoSecret` falls back to `SessionSecret` at `common/init.go:62`, so any access-token/credential encrypted with it becomes undecryptable after a restart.
- Files: [common/constants.go:75-76](file:///e:/AI%20Project/new-api/common/constants.go#L75-L76), [common/init.go:49-63](file:///e:/AI%20Project/new-api/common/init.go#L49-L63)
- Current mitigation: `random_string` literal is rejected; docs presumably tell operators to set `SESSION_SECRET`.
- Recommendations: When `GIN_MODE != debug` and `SESSION_SECRET` is unset, log a loud warning. When multi-instance is detected (SystemInstanceReporter registers more than one node), refuse to start unless `SESSION_SECRET` is explicitly set. Document the `CRYPTO_SECRET` dependency clearly — rotating `SESSION_SECRET` after tokens are issued invalidates encrypted credentials.

**Password maximum length of 20 characters:**
- Risk: The `User.Password` field is validated as `min=8,max=20` — this actively prevents users from using strong passphrases (NIST 800-63B recommends allowing 64+ characters). The cap also forces weaker passwords.
- Files: [model/user.go:27](file:///e:/AI%20Project/new-api/model/user.go#L27)
- Current mitigation: bcrypt is used for hashing at `common/crypto.go:25` (good — bcrypt has its own 72-byte limit).
- Recommendations: Raise `max` to at least 64 (bcrypt accepts up to 72 bytes). Keep `min=8` or raise to 10–12.

**Global insecure TLS config:**
- Risk: `common.InsecureTLSConfig = &tls.Config{InsecureSkipVerify: true}` is a package-level singleton. It is applied when `TLS_INSECURE_SKIP_VERIFY=true` (env-gated at `common/init.go:86`). If an operator sets this flag, all outbound TLS verification is disabled for upstream model calls — enabling MITM on API keys transit.
- Files: [common/constants.go:117-118](file:///e:/AI%20Project/new-api/common/constants.go#L117-L118)
- Current mitigation: Defaults to `false`; SMTP variant is admin-controlled with a `#nosec G402` annotation at `common/email.go:40`.
- Recommendations: When `TLSInsecureSkipVerify` is true, emit a startup warning. Avoid sharing the singleton — construct per-client configs so the flag cannot leak into unexpected clients.

**pprof binds to all interfaces:**
- Risk: When `ENABLE_PPROF=true`, the debug server listens on `0.0.0.0:8005` — reachable from any network the host is on. pprof exposes goroutine dumps, heap profiles, and the `common.Monitor()` runtime info.
- Files: [main.go:153-159](file:///e:/AI%20Project/new-api/main.go#L153-L159)
- Current mitigation: Off by default.
- Recommendations: Bind to `127.0.0.1:8005` by default; add a `PPROF_HOST` override for operators who explicitly want remote access.

**Sensitive-field cleanup relies on developer discipline:**
- Risk: A comment at `model/user.go:22-23` warns that any new sensitive field must be manually cleaned in `setupLogin` or it will be persisted in plain text in browser localStorage.
- Files: [model/user.go:22-23](file:///e:/AI%20Project/new-api/model/user.go#L22-L23)
- Current mitigation: Manual review.
- Recommendations: Replace with an explicit allowlist in the login response DTO (`UserBase` / a dedicated `UserSessionView`) so new fields are excluded by default rather than included by default.

## Performance Bottlenecks

**Synchronous channel-balance refresh loop:**
- Problem: `updateAllChannelsBalance` iterates every channel sequentially with `time.Sleep(common.RequestInterval)` between each, blocking the caller.
- Files: [controller/channel-billing.go:455-482](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L455-L482)
- Cause: No concurrency; the loop is O(n) with a sleep per channel.
- Improvement path: Move to the existing system-task runner (`service.StartSystemTaskRunner`, `controller.RegisterScheduledSystemTasks` at `main.go:144-145`) which already supports DB-lease dedup and history. Run balance checks as periodic scheduled tasks, not on-demand HTTP.

**Retry path doubles body memory:**
- Problem: The relay override path clones the request body for each retry, costing ~2× body size of temp memory per retry under high concurrency.
- Files: [relay/common/override.go:722](file:///e:/AI%20Project/new-api/relay/common/override.go#L722)
- Cause: Full body buffering to support request rewriting on retry.
- Improvement path: Stream the original body when no override is configured; only buffer when `info.ChannelSetting` actually requests a body mutation. Consider a `sync.Pool` for the buffer.

**Gemini relay hot-path string thrash:**
- Problem: Under high concurrency the Gemini relay path performs repeated `+` string concatenations followed by `strings.Join`, amplifying heap residency.
- Files: [relay/channel/gemini/relay-gemini.go:1086](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go#L1086), [relay/channel/gemini/relay-gemini.go:1212](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go#L1212)
- Cause: Incremental text-fragment assembly before a final join.
- Improvement path: Use `strings.Builder` (already idiomatic in Go) for the accumulating buffer; preallocate based on a length hint from the upstream chunk.

## Fragile Areas

**`relay/common/override.go` (1937 lines):**
- Files: [relay/common/override.go](file:///e:/AI%20Project/new-api/relay/common/override.go)
- Why fragile: Single file containing all request/response override transforms for every channel. Contains `err = nil` swallow statements at lines 918, 935, 949 that silence upstream errors during override application. Deeply nested conditionals per channel type. Changes for one channel risk regressions in another.
- Safe modification: Always run `relay/common/override_test.go` (2096 lines — the largest test file, indicating the team already treats this as critical). Add a test case for the specific override being touched before editing. Do not refactor the whole file at once.
- Test coverage: Well-covered by `override_test.go`; the risk is in the swallows (918/935/949) which may hide errors the tests do not exercise.

**`middleware/distributor.go` channel selection:**
- Files: [middleware/distributor.go](file:///e:/AI%20Project/new-api/middleware/distributor.go)
- Why fragile: Central request-routing logic. The `switch channel.Type` block (lines 485-503) plus multi-key handling (lines 472-478) and the model-request parsing path all funnel through here. Any bug here affects every relay request.
- Safe modification: The model-request extraction at `common/gin.go:144` has a "someday non-JSON requests have variant model" TODO — be aware non-JSON bodies silently skip model extraction. Add integration tests around channel-type dispatch before changing the switch.
- Test coverage: No `distributor_test.go` exists. High priority gap.

**`model/subscription.go` (1282 lines) + `service/billing_session.go`:**
- Files: [model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go), [service/billing_session.go](file:///e:/AI%20Project/new-api/service/billing_session.go)
- Why fragile: Subscription quota lifecycle (pre-consume → reserve → settle → reset) spans model + service layers with string-matched error classification (see Tech Debt above). Money path.
- Safe modification: Trace a full request lifecycle through `BillingSession.PreConsume` / `reserveFunding` / settle before editing. Never change error messages in `model/subscription.go` without updating the string matches in `billing_session.go:218`.
- Test coverage: `service/tiered_settle_test.go` (688 lines) and `service/task_billing_test.go` cover parts; no `model/subscription_test.go`.

## Scaling Limits

**Single-process scheduled tasks:**
- Current capacity: One master node runs `service.StartSystemTaskRunner()` (`main.go:145`) and `controller.RegisterScheduledSystemTasks()` (`main.go:144`).
- Limit: If the master node dies, no scheduled tasks (channel tests, upstream model updates, async task polling, subscription quota resets) run until a new master is elected. The DB-lease dedup prevents double-execution but does not itself provide failover — another instance must take the master role.
- Scaling path: Ensure multi-instance deployments run ≥2 nodes capable of becoming master. Monitor `model/system_instance.go` reporter output. Consider externalizing the scheduler (e.g., to a DB-backed work queue) if uptime requirements exceed single-master reliability.

**In-memory channel/token cache coherency:**
- Current capacity: `model.InitChannelCache()` + `model.SyncChannelCache(common.SyncFrequency)` run when `MemoryCacheEnabled` (`main.go:79-99`). Sync frequency is a single global interval.
- Limit: Cache is per-process. In a multi-instance deployment, an admin change on instance A is invisible to instance B until the next `SyncFrequency` tick (default value in `common.SyncFrequency`). The `authz.StartPolicySync` loop (`main.go:105`) follows the same model for policy.
- Scaling path: For tighter consistency, use Redis pub/sub to invalidate on write (Redis is already a dependency). Document the eventual-consistency window for operators.

## Dependencies at Risk

**`github.com/go-redis/redis/v8` (v8.11.5) — EOL major version:**
- Risk: The `go-redis/redis/v8` module path is the legacy import; the project moved to `redis/go-redis/v9`. v8 receives no new features and limited security fixes.
- Impact: `common/redis.go` and all cache layers (`model/channel_cache.go`, `model/token_cache.go`, `model/user_cache.go`) are pinned to v8 APIs.
- Migration plan: Move to `github.com/redis/go-redis/v9`. API is largely compatible; the main breaking changes are pipeline/context semantics and the `Cmdable` interface. Audit `common.InitRedisClient` and the cache callers.

**`github.com/gin-gonic/gin` v1.9.1 — behind current:**
- Risk: v1.9.1 (May 2023) predates several CVE fixes and the v1.10 line. `gin-contrib/cors`, `gin-contrib/sessions`, `gin-contrib/gzip`, `gin-contrib/static` are all pinned to versions matching this line.
- Impact: Security advisories on gin itself; no functional blocker today.
- Migration plan: Bump to gin v1.10.x; verify gin-contrib subpackages are compatible. Run `controller/*_test.go` and the relay integration tests after bump.

**`github.com/shirou/gopsutil` v3.21.11+incompatible — stale/incorrect module path:**
- Risk: The `+incompatible` suffix means the v3 tag was published without a go.mod. The current recommended path is `github.com/shirou/gopsutil/v3` or `v4`. v3.21.11 is from 2021.
- Impact: Used by `common.Monitor()` and the system-info page. Stale process/disk metrics on newer OSes.
- Migration plan: Switch to `github.com/shirou/gopsutil/v4` (or `v3/cpu`, `v3/mem` subpackages with proper module path).

**`gorm.io/gorm` v1.25.2 and drivers — behind current:**
- Risk: v1.25.2 is not the latest v1.25.x patch. `gorm.io/driver/postgres` v1.5.2 and `gorm.io/driver/mysql` v1.4.3 are older still.
- Impact: Bug fixes in migrations and the SQLite column-add path (`model/main.go:564`) may be missing.
- Migration plan: Bump within the v1.25.x line (minor risk). Validate the custom migration helpers in `model/main.go` (`ensureSubscriptionPlanTableSQLite`, `checkAndMigrateDB`) against the new driver versions.

## Missing Critical Features

**No automated retry/circuit-breaker for upstream 5xx beyond channel-disable:**
- Problem: When an upstream provider returns 5xx, the relay retries via channel affinity (`service/channel_affinity.go`, 923 lines) but there is no per-provider circuit breaker. A slow upstream can tie up request goroutines.
- Blocks: Reliable behavior under partial upstream outage.

**No structured audit log of admin actions:**
- Problem: `middleware/audit.go` exists but audit coverage is not comprehensive — the `User` struct comment at `model/user.go:22-23` implies sensitive-field handling is manual. There is no append-only audit trail of quota grants, role changes, or channel-key rotations visible in the codebase map.
- Blocks: Compliance / forensics for billing disputes.

## Test Coverage Gaps

**`controller/user.go` (1332 lines) — no dedicated test:**
- What's not tested: User CRUD, password change, role assignment, access-token generation (`controller/user.go:357` uses `common.GenerateRandomKey`), OAuth binding.
- Files: [controller/user.go](file:///e:/AI%20Project/new-api/controller/user.go)
- Risk: Auth/admin path bugs ship undetected. Password-change edge cases (wrong old password, quota on role change) are untested.
- Priority: High.

**`middleware/auth.go` + `middleware/distributor.go` — no tests:**
- What's not tested: Access-token validation flow, role-gating (`authHelper` min-role), channel selection dispatch, multi-key index handling.
- Files: [middleware/auth.go](file:///e:/AI%20Project/new-api/middleware/auth.go), [middleware/distributor.go](file:///e:/AI%20Project/new-api/middleware/distributor.go)
- Risk: Every request flows through these. A regression here is a total outage.
- Priority: High.

**`model/user.go` (981 lines), `model/subscription.go` (1282 lines), `model/channel.go` (1014 lines) — no model-layer tests:**
- What's not tested: Quota atomicity (`IncreaseUserQuota`, `DecreaseUserQuota`), subscription lifecycle transitions, channel cache invalidation.
- Files: [model/user.go](file:///e:/AI%20Project/new-api/model/user.go), [model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go), [model/channel.go](file:///e:/AI%20Project/new-api/model/channel.go)
- Risk: Money-path data integrity bugs. Note `model/redemption.go` and `model/topup.go` already use `recover()` guards (lines 36, 69, 177, 213, 246, 286) suggesting past panic incidents in these paths.
- Priority: High (money path), Medium (user/channel).

**`service/convert.go` (938 lines) — no test:**
- What's not tested: Request/response format conversion between OpenAI/Claude/Gemini/etc.
- Files: [service/convert.go](file:///e:/AI%20Project/new-api/service/convert.go)
- Risk: Silent format-conversion regressions cause malformed upstream requests.
- Priority: Medium. (Note: `service/relayconvert/` subpackage is tested — the untested code is the older `service/convert.go`.)

**`oauth/*.go` — no tests:**
- What's not tested: GitHub/Discord/OIDC/LinuxDO/generic OAuth flows (`oauth/github.go`, `oauth/discord.go`, `oauth/oidc.go`, `oauth/linuxdo.go`, `oauth/generic.go`).
- Risk: OAuth provider API changes break login silently. The `NewOAuthErrorWithRaw` wrapper is used extensively but no tests assert error-classification correctness.
- Priority: Medium.

**`controller/relay.go` (593 lines) — no test:**
- What's not tested: The main relay entry handler that wires distributor → adaptor → response.
- Files: [controller/relay.go](file:///e:/AI%20Project/new-api/controller/relay.go)
- Risk: Relay routing regressions. Partially mitigated by relay adaptor tests, but the top-level handler is untested.
- Priority: Medium.

---

*Concerns audit: 2026-07-01*
