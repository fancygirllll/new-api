# Testing Patterns

**Analysis Date:** 2026-07-01

## Test Framework

**Runner:**
- Go standard `testing` package (no external runner)
- Go 1.25.1 toolchain (see [go.mod](file:///e:/AI%20Project/new-api/go.mod))
- No `Makefile`; tests run via `go test`

**Assertion Library:**
- `github.com/stretchr/testify` v1.11.1 — both `assert` and `require` subpackages
  - `require.NoError(t, err)` / `require.Equal(t, expected, actual)` — fail-fast (stops test)
  - `assert.Equal(t, expected, actual)` / `assert.Contains(t, str, sub)` — record failure, continue
- `github.com/glebarez/sqlite` v1.9.0 — in-memory SQLite for DB-backed tests
- `net/http/httptest` — HTTP request/response recording and fake servers
- `github.com/gin-gonic/gin` test mode via `gin.CreateTestContext`

**Run Commands:**
```bash
go test ./...                      # Run all tests
go test ./controller/...            # Single package
go test -run TestGetAllTokens ./controller/   # Single test
go test -v ./...                    # Verbose
go test -race ./...                 # With race detector
go test -cover ./...                # Coverage report
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out   # HTML coverage

# External DB migration tests (skip by default without DSN):
TEST_MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/testdb" go test ./controller/ -run TestTokenMigrationFromChar48ToVarchar128MySQL
TEST_POSTGRES_DSN="user=postgres password=pass dbname=testdb sslmode=disable" go test ./controller/ -run TestTokenMigrationFromChar48ToVarchar128Postgres
```

**CI:** No dedicated test workflow found in `.github/workflows/` — the `pr-check.yml` workflow only enforces PR template / anti-slop rules. Tests are expected to run locally and via Docker build. Build verification happens in the `docker-build.yml` pipeline.

## Test File Organization

**Location:**
- Co-located with source: `foo.go` → `foo_test.go` in the **same package** (white-box access to unexported symbols)
- Internal tests use the production package name (`package controller`, `package model`, `package common`) — not `package controller_test`
- A few files are `_internal_test.go` to emphasize white-box scope: [controller/channel_test_internal_test.go](file:///e:/AI%20Project/new-api/controller/channel_test_internal_test.go)

**Naming:**
- `Test<Unit><Scenario>` (e.g., `TestGetAllTokensMasksKeyInResponse`, `TestTokenAutoMigrateUsesVarchar128KeyColumn`)
- Table-driven subtests named via the `name` field (e.g., `"exact domain match with https"`)
- Helper functions: `setupXxx`, `seedXxx`, `newXxx`, `insertXxx`, `clearXxx`, `truncateXxx`, `decodeXxxResponse`

**Structure:**
```
controller/
├── token.go
├── token_test.go                  # token controller tests + helpers
├── channel.go
├── channel_test_internal_test.go  # white-box channel tests
├── channel_authz.go
├── channel_authz_test.go
model/
├── token.go
├── task_cas_test.go               # package-wide TestMain + truncateTables helper
├── system_task_test.go
├── model_owner_test.go
relay/
├── chat_completions_via_responses_test.go
└── channel/<provider>/<provider>_test.go or adaptor_test.go
```

## Test Structure

**Suite Organization:**
Standard `func TestXxx(t *testing.T)` top-level functions. Table-driven tests use anonymous structs + `t.Run`:

```go
// From controller/url_validator_test.go (common/ package)
func TestValidateRedirectURL(t *testing.T) {
    originalDomains := constant.TrustedRedirectDomains
    defer func() { constant.TrustedRedirectDomains = originalDomains }()

    tests := []struct {
        name           string
        url            string
        trustedDomains []string
        wantErr        bool
        errContains    string
    }{
        {name: "exact domain match with https", url: "https://example.com/success", trustedDomains: []string{"example.com"}, wantErr: false},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            constant.TrustedRedirectDomains = tt.trustedDomains
            err := ValidateRedirectURL(tt.url)
            if tt.wantErr {
                // assert error
            } else {
                // assert no error
            }
        })
    }
}
```

**Patterns:**
- **Setup helper:** `t.Helper()` is mandatory on test helpers (marks the caller's line on failure)
- **Teardown:** `t.Cleanup(func(){ ... })` (preferred over `defer` because it composes across helpers)
- **Parallel:** `t.Parallel()` on independent tests/subtests; remember `tc := tc` capture before the subtest closure (see [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go))
- **Skip for external deps:** `t.Skip("set TEST_MYSQL_DSN to run mysql migration compatibility test")` when env var absent
- **Fatal vs. non-fatal:** `require.Xxx` for setup invariants (DB open, migrate); `assert.Xxx` for assertions you want all of

## Mocking

**Framework:** No mocking framework (no `gomock`, `mockery`, `testify/mock`). Mocking is done by:
1. **In-memory SQLite** replacing the production DB
2. **`httptest.NewRequest` + `httptest.NewRecorder`** replacing real HTTP
3. **Fake servers** built on `net.Listen` / `httptest.NewServer` for upstream protocols (SMTP, HTTP)
4. **Global-state save/restore** for package-level config vars

**Patterns:**

In-memory DB + gin context (the dominant pattern for controller/service/model tests):

```go
// From controller/token_test.go
func setupTokenControllerTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    gin.SetMode(gin.TestMode)
    common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
    common.RedisEnabled = false

    dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open sqlite db: %v", err)
    }
    model.DB = db
    model.LOG_DB = db

    t.Cleanup(func() {
        sqlDB, err := db.DB()
        if err == nil { _ = sqlDB.Close() }
    })
    return db
}

func newAuthenticatedContext(t *testing.T, method, target string, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
    t.Helper()
    recorder := httptest.NewRecorder()
    ctx, _ := gin.CreateTestContext(recorder)
    ctx.Request = httptest.NewRequest(method, target, requestBody)
    ctx.Set("id", userID)   // simulate auth middleware
    return ctx, recorder
}
```

Global-state save/restore (for config/setting tests):

```go
// From controller/payment_webhook_availability_test.go
func TestStripeWebhookEnabledRequiresTopUpAndWebhookConfig(t *testing.T) {
    originalAPISecret := setting.StripeApiSecret
    originalWebhookSecret := setting.StripeWebhookSecret
    t.Cleanup(func() {
        setting.StripeApiSecret = originalAPISecret
        setting.StripeWebhookSecret = originalWebhookSecret
    })

    setting.StripeWebhookSecret = ""
    setting.StripeApiSecret = "sk_test_123"
    require.False(t, isStripeWebhookEnabled())
    // ...
}
```

Writer swap (for log-capture tests):

```go
// From service/error_test.go
var logBuffer bytes.Buffer
common.LogWriterMu.Lock()
oldWriter := gin.DefaultErrorWriter
gin.DefaultErrorWriter = &logBuffer
common.LogWriterMu.Unlock()
t.Cleanup(func() {
    common.LogWriterMu.Lock()
    gin.DefaultErrorWriter = oldWriter
    common.LogWriterMu.Unlock()
})
```

Fake upstream server (for protocol-level tests):

```go
// From common/email_test.go — fake SMTP server on a random port
listener, err := net.Listen("tcp", "127.0.0.1:0")
// fakeSMTPServer handles SMTP dialogue, captures commands into channels
```

**What to Mock:**
- The database — always replace with in-memory SQLite (`sqlite.Open(":memory:")` or `file:...?mode=memory&cache=shared`)
- Redis — disable by setting `common.RedisEnabled = false` (and `common.BatchUpdateEnabled = false`)
- Gin context — construct with `gin.CreateTestContext(httptest.NewRecorder())` and set context values that middleware would normally populate (`id`, `token_id`, `group`)
- Upstream HTTP/SMTP servers — fake servers on `127.0.0.1:0`
- Global config vars (`common.DebugEnabled`, `common.IsMasterNode`, `setting.*`) — save, mutate, restore via `t.Cleanup`

**What NOT to Mock:**
- Adaptor logic itself — instantiate the real `Adaptor{}` and call `ConvertOpenAIRequest` directly (see [relay/channel/moonshot/adaptor_test.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor_test.go))
- GORM query behavior — let it run against real SQLite; do not stub `DB.Create`/`DB.Find`
- The response envelope decoder — use `common.Unmarshal` on the recorder body

## Fixtures and Factories

**Test Data:**
Inline struct literals are the norm; no factory library. Helper functions seed minimal rows:

```go
// From controller/token_test.go
func seedToken(t *testing.T, db *gorm.DB, userID int, name string, rawKey string) *model.Token {
    t.Helper()
    token := &model.Token{
        UserId:         userID,
        Name:           name,
        Key:            rawKey,
        Status:         common.TokenStatusEnabled,
        CreatedTime:    1,
        AccessedTime:   1,
        ExpiredTime:    -1,
        RemainQuota:    100,
        UnlimitedQuota: true,
        Group:          "default",
    }
    if err := db.Create(token).Error; err != nil {
        t.Fatalf("failed to create token: %v", err)
    }
    return token
}
```

```go
// From model/model_owner_test.go
func insertPreferredOwnerCandidate(t *testing.T, channelID int, modelName, group string, channelType int, priority int64, weight uint, channelStatus int, abilityEnabled bool) {
    t.Helper()
    require.NoError(t, DB.Create(&Channel{Id: channelID, Type: channelType, Key: fmt.Sprintf("key-%d", channelID), Status: channelStatus, Name: fmt.Sprintf("channel-%d", channelID)}).Error)
    require.NoError(t, DB.Create(&Ability{Group: group, Model: modelName, ChannelId: channelID, Enabled: abilityEnabled, Priority: &priority, Weight: weight}).Error)
}
```

**Location:**
- Helpers live at the top of the test file that uses them (no shared `testutil/` package)
- The one package-wide fixture is `truncateTables(t)` in [model/task_cas_test.go](file:///e:/AI%20Project/new-api/model/task_cas_test.go)

## Test Database Setup

**Package-wide `TestMain`** (when the package needs a shared DB across all tests):

```go
// From model/task_cas_test.go
func TestMain(m *testing.M) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil { panic("failed to open test db: " + err.Error()) }
    DB = db
    LOG_DB = db

    common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
    common.RedisEnabled = false
    common.BatchUpdateEnabled = false
    common.LogConsumeEnabled = true
    initCol()

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(1)   // serialize to avoid SQLite locking issues

    if err := db.AutoMigrate(&Task{}, &User{}, &Token{}, /* ... */); err != nil {
        panic("failed to migrate: " + err.Error())
    }
    os.Exit(m.Run())
}

func truncateTables(t *testing.T) {
    t.Helper()
    t.Cleanup(func() {
        DB.Exec("DELETE FROM tasks")
        DB.Exec("DELETE FROM users")
        // ... one DELETE per table
    })
}
```

**Per-test DB (when isolation matters more than reuse):**
- `openTokenControllerTestDB(t)` uses `file:<TestName>?mode=memory&cache=shared` so each test gets a fresh named DB
- `common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)` must be called so GORM dialect selection matches
- Always set `model.DB = db` and `model.LOG_DB = db` (the production code reads these package globals)
- `sqlDB.SetMaxOpenConns(1)` is recommended to avoid SQLite "database is locked" errors

## Coverage

**Requirements:** None enforced. No `go test -cover` threshold in CI.

**View Coverage:**
```bash
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out      # per-function
go tool cover -html=cover.out      # opens browser
```

**Existing coverage is uneven:** tests concentrate on billing, relay adaptors, authz, token masking, and migration compatibility. Many controller handlers and model functions have no direct test (verified by the 60 `*_test.go` files against 200+ source files).

## Test Types

**Unit Tests:**
- Pure logic: [common/json_test.go](file:///e:/AI%20Project/new-api/common/json_test.go), [relay/chat_completions_via_responses_test.go](file:///e:/AI%20Project/new-api/relay/chat_completions_via_responses_test.go)
- Table-driven, no external deps, `t.Parallel()` where safe

**Integration Tests (handler + DB):**
- Construct a gin context with `gin.CreateTestContext`, set auth context values, call the real handler, assert on the recorder body
- Examples: [controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go), [controller/channel_test_internal_test.go](file:///e:/AI%20Project/new-api/controller/channel_test_internal_test.go), [model/system_task_test.go](file:///e:/AI%20Project/new-api/model/system_task_test.go)

**Migration Compatibility Tests:**
- Run legacy schema → `AutoMigrate` → assert column type changed and data preserved
- Against SQLite by default; MySQL/Postgres gated by `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN` env vars
- Example: `TestTokenMigrationFromChar48ToVarchar128*` in [controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go)

**Fake-Server Tests:**
- Spin up a real TCP listener for protocols Gin can't easily stub (SMTP, raw HTTP upstream)
- Example: `fakeSMTPServer` in [common/email_test.go](file:///e:/AI%20Project/new-api/common/email_test.go), `httptest.NewServer` in relay helper tests

**E2E Tests:** Not used. No Playwright/Selenium. Frontend has its own `bun run`-driven checks (see i18n sync scripts) but no Go-level end-to-end harness.

## Common Patterns

**Async Testing:**
Go tests are synchronous by default. For goroutine/timer logic, use `require.Eventually`:

```go
require.Eventually(t, func() bool {
    // poll condition
}, 2*time.Second, 50*time.Millisecond)
```

**Error Testing:**
```go
// Assert error present and inspect message
require.Error(t, err)
assert.Contains(t, err.Error(), "not in the trusted domains list")

// Assert error type
var newAPIErr *types.NewAPIError
require.ErrorAs(t, err, &newAPIErr)

// Assert no error (fail-fast)
require.NoError(t, db.AutoMigrate(&model.Token{}).Error)
```

**HTTP Handler Testing (the canonical flow):**
```go
// 1. Setup DB
db := setupTokenControllerTestDB(t)
token := seedToken(t, db, 1, "list-token", "abcd1234efgh5678")

// 2. Build context + recorder
ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/?p=1&size=10", nil, 1)

// 3. Call the real handler (no router, no middleware)
GetAllTokens(ctx)

// 4. Decode the envelope and assert
response := decodeAPIResponse(t, recorder)
require.True(t, response.Success)
require.NotContains(t, recorder.Body.String(), token.Key)  // no secret leak
```

**Gin test mode:** Always call `gin.SetMode(gin.TestMode)` at the start of handler-touching tests to suppress debug logging. Some packages do this in a test-file `func init()` (e.g., [relay/helper/stream_scanner_test.go](file:///e:/AI%20Project/new-api/relay/helper/stream_scanner_test.go)).

**Test isolation rules:**
- Each test that mutates `model.DB` or global config must restore via `t.Cleanup`
- `truncateTables(t)` runs in `t.Cleanup` (after the test) so failures still see a clean slate for the next run — note: this means within a single test you must seed your own data; tables are NOT pre-truncated
- `common.RedisEnabled = false` is mandatory in DB tests (Redis is not available in CI)
- `common.BatchUpdateEnabled = false` prevents async batch writers from racing the test

## Test Helpers Reference

| Helper | Location | Purpose |
|--------|----------|---------|
| `setupTokenControllerTestDB(t)` | [controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go) | In-memory SQLite + set `model.DB` |
| `setupModelListControllerTestDB(t)` | controller | Controller test DB (channel list) |
| `newAuthenticatedContext(t, ...)` | controller/token_test.go | Gin context with `id` set |
| `decodeAPIResponse(t, recorder)` | controller/token_test.go | Decode `{success,message,data}` envelope |
| `truncateTables(t)` | [model/task_cas_test.go](file:///e:/AI%20Project/new-api/model/task_cas_test.go) | Wipe all tables in `t.Cleanup` |
| `insertTask(t, db, task)` | model/task_cas_test.go | Seed a task row |
| `insertPreferredOwnerCandidate(t, ...)` | [model/model_owner_test.go](file:///e:/AI%20Project/new-api/model/model_owner_test.go) | Seed channel + ability |
| `clearPreferredOwnerTables(t)` | model/model_owner_test.go | Delete abilities + channels |
| `newAuthzTestDB(t)` | [service/authz/authz_test.go](file:///e:/AI%20Project/new-api/service/authz/authz_test.go) | In-memory DB + master-node flag |
| `confirmPaymentComplianceForTest(t)` | [controller/payment_webhook_availability_test.go](file:///e:/AI%20Project/new-api/controller/payment_webhook_availability_test.go) | Toggle compliance setting + restore |
| `withDebugEnabled(t, bool)` | [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go) | Toggle + restore `common.DebugEnabled` |
| `setupStreamTest(t, body)` | [relay/helper/stream_scanner_test.go](file:///e:/AI%20Project/new-api/relay/helper/stream_scanner_test.go) | Gin ctx + fake `*http.Response` for SSE |

---

*Testing analysis: 2026-07-01*
