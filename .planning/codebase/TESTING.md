# 测试模式

**分析日期：** 2026-07-01

## 测试框架

**运行器：**
- Go 标准 `testing` 包（无外部运行器）
- Go 1.25.1 工具链（见 [go.mod](file:///e:/AI%20Project/new-api/go.mod)）
- 无 `Makefile`；通过 `go test` 运行测试

**断言库：**
- `github.com/stretchr/testify` v1.11.1 —— 同时使用 `assert` 和 `require` 子包
  - `require.NoError(t, err)` / `require.Equal(t, expected, actual)` —— 快速失败（停止测试）
  - `assert.Equal(t, expected, actual)` / `assert.Contains(t, str, sub)` —— 记录失败，继续执行
- `github.com/glebarez/sqlite` v1.9.0 —— 用于 DB 相关测试的内存 SQLite
- `net/http/httptest` —— HTTP 请求/响应录制与伪服务器
- `github.com/gin-gonic/gin` 测试模式通过 `gin.CreateTestContext`

**运行命令：**
```bash
go test ./...                      # 运行全部测试
go test ./controller/...            # 单个包
go test -run TestGetAllTokens ./controller/   # 单个测试
go test -v ./...                    # 详细输出
go test -race ./...                 # 启用竞态检测
go test -cover ./...                # 覆盖率报告
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out   # HTML 覆盖率

# 外部 DB 迁移测试（无 DSN 时默认跳过）：
TEST_MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/testdb" go test ./controller/ -run TestTokenMigrationFromChar48ToVarchar128MySQL
TEST_POSTGRES_DSN="user=postgres password=pass dbname=testdb sslmode=disable" go test ./controller/ -run TestTokenMigrationFromChar48ToVarchar128Postgres
```

**CI：** `.github/workflows/` 中未发现专门的测试工作流 —— `pr-check.yml` 工作流仅强制 PR 模板 / 反 slop 规则。测试应在本地和通过 Docker 构建运行。构建校验在 `docker-build.yml` 流水线中完成。

## 测试文件组织

**位置：**
- 与源文件同目录：`foo.go` → `foo_test.go`，位于**同一包**中（白盒访问未导出符号）
- 内部测试使用生产包名（`package controller`、`package model`、`package common`）—— 而非 `package controller_test`
- 部分文件名为 `_internal_test.go` 以强调白盒范围：[controller/channel_test_internal_test.go](file:///e:/AI%20Project/new-api/controller/channel_test_internal_test.go)

**命名：**
- `Test<Unit><Scenario>`（例如 `TestGetAllTokensMasksKeyInResponse`、`TestTokenAutoMigrateUsesVarchar128KeyColumn`）
- 表驱动子测试通过 `name` 字段命名（例如 `"exact domain match with https"`）
- 辅助函数：`setupXxx`、`seedXxx`、`newXxx`、`insertXxx`、`clearXxx`、`truncateXxx`、`decodeXxxResponse`

**结构：**
```
controller/
├── token.go
├── token_test.go                  # token 控制器测试 + 辅助函数
├── channel.go
├── channel_test_internal_test.go  # 白盒 channel 测试
├── channel_authz.go
├── channel_authz_test.go
model/
├── token.go
├── task_cas_test.go               # 包级 TestMain + truncateTables 辅助函数
├── system_task_test.go
├── model_owner_test.go
relay/
├── chat_completions_via_responses_test.go
└── channel/<provider>/<provider>_test.go or adaptor_test.go
```

## 测试结构

**套件组织：**
标准 `func TestXxx(t *testing.T)` 顶层函数。表驱动测试使用匿名结构体 + `t.Run`：

```go
// 摘自 controller/url_validator_test.go（common/ 包）
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
                // 断言有错误
            } else {
                // 断言无错误
            }
        })
    }
}
```

**模式：**
- **Setup 辅助函数：** 测试辅助函数上必须调用 `t.Helper()`（失败时标记调用者行）
- **Teardown：** `t.Cleanup(func(){ ... })`（优先于 `defer`，因为它能跨辅助函数组合）
- **并行：** 在相互独立的测试/子测试上加 `t.Parallel()`；记得在子测试闭包前用 `tc := tc` 捕获（见 [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go)）
- **跳过外部依赖：** 当环境变量缺失时 `t.Skip("set TEST_MYSQL_DSN to run mysql migration compatibility test")`
- **致命 vs 非致命：** `require.Xxx` 用于 setup 不变量（DB 打开、迁移）；`assert.Xxx` 用于希望全部跑完的断言

## 模拟（Mocking）

**框架：** 无 mocking 框架（无 `gomock`、`mockery`、`testify/mock`）。Mocking 通过以下方式完成：
1. **内存 SQLite** 替换生产 DB
2. **`httptest.NewRequest` + `httptest.NewRecorder`** 替换真实 HTTP
3. **伪服务器** 基于 `net.Listen` / `httptest.NewServer` 构建上游协议（SMTP、HTTP）
4. **全局状态保存/恢复** 用于包级配置变量

**模式：**

内存 DB + gin 上下文（控制器/服务/模型测试的主导模式）：

```go
// 摘自 controller/token_test.go
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
    ctx.Set("id", userID)   // 模拟鉴权中间件
    return ctx, recorder
}
```

全局状态保存/恢复（用于配置/设置测试）：

```go
// 摘自 controller/payment_webhook_availability_test.go
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

写入器替换（用于日志捕获测试）：

```go
// 摘自 service/error_test.go
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

伪上游服务器（用于协议级测试）：

```go
// 摘自 common/email_test.go —— 在随机端口上的伪 SMTP 服务器
listener, err := net.Listen("tcp", "127.0.0.1:0")
// fakeSMTPServer 处理 SMTP 对话，将命令捕获到 channel 中
```

**Mock 什么：**
- 数据库 —— 总是用内存 SQLite 替换（`sqlite.Open(":memory:")` 或 `file:...?mode=memory&cache=shared`）
- Redis —— 通过 `common.RedisEnabled = false` 禁用（以及 `common.BatchUpdateEnabled = false`）
- Gin 上下文 —— 用 `gin.CreateTestContext(httptest.NewRecorder())` 构造，并设置中间件通常会填充的上下文值（`id`、`token_id`、`group`）
- 上游 HTTP/SMTP 服务器 —— 在 `127.0.0.1:0` 上的伪服务器
- 全局配置变量（`common.DebugEnabled`、`common.IsMasterNode`、`setting.*`）—— 保存、修改、通过 `t.Cleanup` 恢复

**不要 Mock 什么：**
- 适配器逻辑本身 —— 实例化真实的 `Adaptor{}` 并直接调用 `ConvertOpenAIRequest`（见 [relay/channel/moonshot/adaptor_test.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor_test.go)）
- GORM 查询行为 —— 让它对真实 SQLite 执行；不要 stub `DB.Create`/`DB.Find`
- 响应信封解码器 —— 对 recorder body 使用 `common.Unmarshal`

## 固定数据与工厂

**测试数据：**
内联结构体字面量是常态；无工厂库。辅助函数种入最小行：

```go
// 摘自 controller/token_test.go
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
// 摘自 model/model_owner_test.go
func insertPreferredOwnerCandidate(t *testing.T, channelID int, modelName, group string, channelType int, priority int64, weight uint, channelStatus int, abilityEnabled bool) {
    t.Helper()
    require.NoError(t, DB.Create(&Channel{Id: channelID, Type: channelType, Key: fmt.Sprintf("key-%d", channelID), Status: channelStatus, Name: fmt.Sprintf("channel-%d", channelID)}).Error)
    require.NoError(t, DB.Create(&Ability{Group: group, Model: modelName, ChannelId: channelID, Enabled: abilityEnabled, Priority: &priority, Weight: weight}).Error)
}
```

**位置：**
- 辅助函数位于使用它们的测试文件顶部（无共享 `testutil/` 包）
- 唯一的包级 fixture 是 [model/task_cas_test.go](file:///e:/AI%20Project/new-api/model/task_cas_test.go) 中的 `truncateTables(t)`

## 测试数据库设置

**包级 `TestMain`**（当包内所有测试需要共享 DB 时）：

```go
// 摘自 model/task_cas_test.go
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
    sqlDB.SetMaxOpenConns(1)   // 串行化以避免 SQLite 锁问题

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
        // ... 每张表一条 DELETE
    })
}
```

**单测试 DB（当隔离比复用更重要时）：**
- `openTokenControllerTestDB(t)` 使用 `file:<TestName>?mode=memory&cache=shared`，让每个测试得到一个全新的命名 DB
- 必须调用 `common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)`，以便 GORM 方言选择匹配
- 总是设置 `model.DB = db` 和 `model.LOG_DB = db`（生产代码读取这些包级全局变量）
- 推荐设置 `sqlDB.SetMaxOpenConns(1)` 以避免 SQLite "database is locked" 错误

## 覆盖率

**要求：** 无强制要求。CI 中没有 `go test -cover` 阈值。

**查看覆盖率：**
```bash
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out      # 按函数
go tool cover -html=cover.out      # 在浏览器中打开
```

**现有覆盖率不均衡：** 测试集中在计费、中转适配器、authz、token 脱敏和迁移兼容性。许多控制器处理函数和模型函数没有直接测试（已通过 60 个 `*_test.go` 文件对比 200+ 个源文件验证）。

## 测试类型

**单元测试：**
- 纯逻辑：[common/json_test.go](file:///e:/AI%20Project/new-api/common/json_test.go)、[relay/chat_completions_via_responses_test.go](file:///e:/AI%20Project/new-api/relay/chat_completions_via_responses_test.go)
- 表驱动，无外部依赖，安全处加 `t.Parallel()`

**集成测试（处理函数 + DB）：**
- 用 `gin.CreateTestContext` 构造 gin 上下文，设置鉴权上下文值，调用真实处理函数，对 recorder body 做断言
- 示例：[controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go)、[controller/channel_test_internal_test.go](file:///e:/AI%20Project/new-api/controller/channel_test_internal_test.go)、[model/system_task_test.go](file:///e:/AI%20Project/new-api/model/system_task_test.go)

**迁移兼容性测试：**
- 运行旧 schema → `AutoMigrate` → 断言列类型已变更且数据保留
- 默认对 SQLite；MySQL/Postgres 由 `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN` 环境变量门控
- 示例：[controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go) 中的 `TestTokenMigrationFromChar48ToVarchar128*`

**伪服务器测试：**
- 为 Gin 难以 stub 的协议（SMTP、原始 HTTP 上游）启动真实 TCP 监听器
- 示例：[common/email_test.go](file:///e:/AI%20Project/new-api/common/email_test.go) 中的 `fakeSMTPServer`，中转辅助测试中的 `httptest.NewServer`

**E2E 测试：** 未使用。无 Playwright/Selenium。前端有自己的 `bun run` 驱动的检查（见 i18n 同步脚本），但无 Go 级端到端框架。

## 常见模式

**异步测试：**
Go 测试默认同步。对于协程/定时器逻辑，使用 `require.Eventually`：

```go
require.Eventually(t, func() bool {
    // 轮询条件
}, 2*time.Second, 50*time.Millisecond)
```

**错误测试：**
```go
// 断言有错误并检查消息
require.Error(t, err)
assert.Contains(t, err.Error(), "not in the trusted domains list")

// 断言错误类型
var newAPIErr *types.NewAPIError
require.ErrorAs(t, err, &newAPIErr)

// 断言无错误（快速失败）
require.NoError(t, db.AutoMigrate(&model.Token{}).Error)
```

**HTTP 处理函数测试（标准流程）：**
```go
// 1. setup DB
db := setupTokenControllerTestDB(t)
token := seedToken(t, db, 1, "list-token", "abcd1234efgh5678")

// 2. 构造上下文 + recorder
ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/?p=1&size=10", nil, 1)

// 3. 调用真实处理函数（无路由器，无中间件）
GetAllTokens(ctx)

// 4. 解码信封并断言
response := decodeAPIResponse(t, recorder)
require.True(t, response.Success)
require.NotContains(t, recorder.Body.String(), token.Key)  // 无密钥泄露
```

**Gin 测试模式：** 在涉及处理函数的测试开始处总是调用 `gin.SetMode(gin.TestMode)` 以抑制 debug 日志。部分包在测试文件的 `func init()` 中完成此设置（例如 [relay/helper/stream_scanner_test.go](file:///e:/AI%20Project/new-api/relay/helper/stream_scanner_test.go)）。

**测试隔离规则：**
- 每个修改 `model.DB` 或全局配置的测试必须通过 `t.Cleanup` 恢复
- `truncateTables(t)` 在 `t.Cleanup` 中执行（在测试之后），这样失败用例仍能为下一次运行看到干净状态 —— 注意：这意味着在单个测试内你必须自己种入数据；表不会被预先清空
- DB 测试中必须设置 `common.RedisEnabled = false`（CI 中 Redis 不可用）
- `common.BatchUpdateEnabled = false` 阻止异步批量写入器与测试竞态

## 测试辅助函数参考

| 辅助函数 | 位置 | 用途 |
|--------|----------|---------|
| `setupTokenControllerTestDB(t)` | [controller/token_test.go](file:///e:/AI%20Project/new-api/controller/token_test.go) | 内存 SQLite + 设置 `model.DB` |
| `setupModelListControllerTestDB(t)` | controller | 控制器测试 DB（channel 列表） |
| `newAuthenticatedContext(t, ...)` | controller/token_test.go | 设置了 `id` 的 gin 上下文 |
| `decodeAPIResponse(t, recorder)` | controller/token_test.go | 解码 `{success,message,data}` 信封 |
| `truncateTables(t)` | [model/task_cas_test.go](file:///e:/AI%20Project/new-api/model/task_cas_test.go) | 在 `t.Cleanup` 中清空所有表 |
| `insertTask(t, db, task)` | model/task_cas_test.go | 种入一条 task 行 |
| `insertPreferredOwnerCandidate(t, ...)` | [model/model_owner_test.go](file:///e:/AI%20Project/new-api/model/model_owner_test.go) | 种入 channel + ability |
| `clearPreferredOwnerTables(t)` | model/model_owner_test.go | 删除 abilities + channels |
| `newAuthzTestDB(t)` | [service/authz/authz_test.go](file:///e:/AI%20Project/new-api/service/authz/authz_test.go) | 内存 DB + master-node 标志 |
| `confirmPaymentComplianceForTest(t)` | [controller/payment_webhook_availability_test.go](file:///e:/AI%20Project/new-api/controller/payment_webhook_availability_test.go) | 切换合规设置 + 恢复 |
| `withDebugEnabled(t, bool)` | [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go) | 切换 + 恢复 `common.DebugEnabled` |
| `setupStreamTest(t, body)` | [relay/helper/stream_scanner_test.go](file:///e:/AI%20Project/new-api/relay/helper/stream_scanner_test.go) | gin ctx + 用于 SSE 的伪 `*http.Response` |

---

*测试分析：2026-07-01*
