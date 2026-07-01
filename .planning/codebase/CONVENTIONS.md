# 编码规范

**分析日期：** 2026-07-01

## 项目背景

- **模块：** `github.com/QuantumNous/new-api`（见 [go.mod](file:///e:/AI%20Project/new-api/go.mod)）
- **语言：** Go 1.25.1（后端），React + Vite + TypeScript（前端，通过 `//go:embed` 内嵌）
- **框架：** Gin（`github.com/gin-gonic/gin` v1.9.1）+ GORM（`gorm.io/gorm` v1.25.2）
- **两套前端主题：** `web/default`（现代）和 `web/classic`（旧版），均通过 bun 构建并通过 [main.go](file:///e:/AI%20Project/new-api/main.go) 中的 `buildFS`/`classicBuildFS` 内嵌进 Go 二进制

## 命名规范

**文件：**
- 所有 Go 源文件采用 `snake_case.go`：`sys_log.go`、`url_validator.go`、`topup_stripe.go`、`channel_authz.go`
- 测试文件与源文件同目录，命名为 `*_test.go`：`token.go` + `token_test.go`，`url_validator.go` + `url_validator_test.go`
- 接口文件使用规范拼写 `adapter.go`，位于 [relay/channel/adapter.go](file:///e:/AI%20Project/new-api/relay/channel/adapter.go)（注意：目录为 `relay/channel`，类型为 `Adaptor`）

**包：**
- 顶层包为单个单词小写：`common`、`controller`、`model`、`dto`、`types`、`constant`、`middleware`、`relay`、`service`、`router`、`setting`、`i18n`、`logger`、`oauth`、`pkg`
- 子包采用 `snake_case`：`relay/channel/openai`、`service/authz`、`setting/operation_setting`、`setting/ratio_setting`、`pkg/billingexpr`、`pkg/perf_metrics`
- 导入别名：`relaycommon` 对应 `relay/common`，`relayconstant` 对应 `relay/constant`，`channelconstant` 对应 `constant`，`perfmetrics` 对应 `pkg/perf_metrics`

**函数：**
- 导出函数：`PascalCase`（`GetAllTokens`、`ApiErrorI18n`、`ConvertOpenAIRequest`）
- 未导出函数：`camelCase`（`buildMaskedTokenResponse`、`detectLanguage`、`isTemperatureOneOnlyModel`）
- 构造函数：`NewXxx`（`NewError`、`NewOpenAIError`、`NewLocalizer`）
- HTTP 处理函数为 `controller/` 下导出的自由函数，接收 `*gin.Context`

**类型/结构体：**
- `PascalCase`：`Token`、`Channel`、`RelayInfo`、`PageInfo`、`NewAPIError`、`OpenAIError`
- `dto/` 中的 DTO 结构体与外部 API 形状对应（`GeneralOpenAIRequest`、`ClaudeRequest`、`GeminiChatRequest`）
- `model/` 中的模型结构体使用 GORM tag + JSON tag（见 [model/token.go](file:///e:/AI%20Project/new-api/model/token.go)）

**常量：**
- 分组在 `const (...)` 块中，`PascalCase`
- i18n 消息键为 [i18n/keys.go](file:///e:/AI%20Project/new-api/i18n/keys.go) 中的命名空间化字符串常量：`MsgInvalidParams = "common.invalid_params"`，`MsgAuthNotLoggedIn = "auth.not_logged_in"`
- 错误码定义在 [types/error.go](file:///e:/AI%20Project/new-api/types/error.go) 中作为类型化的 `ErrorCode` 常量：`ErrorCodeInvalidRequest`、`ErrorCodeChannelNoAvailableKey`

**上下文键：**
- 类型化为 `constant.ContextKey string`（见 [constant/context_key.go](file:///e:/AI%20Project/new-api/constant/context_key.go)）——绝不使用原始字符串键
- 常量按域前缀：`ContextKeyToken...`、`ContextKeyChannel...`、`ContextKeyUser...`

## 代码风格

**格式化：**
- `gofmt` / `goimports` 默认配置（仓库根目录下没有 `.golangci.yml` 或 `.editorconfig`）
- 使用 Tab 缩进，无行尾空格

**Lint：**
- 仓库中没有 golangci-lint 配置。CI 重点在构建 + 测试。
- 前端（web/default）使用 ESLint（Docker 构建时设置 `DISABLE_ESLINT_PLUGIN='true'`，见 [Dockerfile](file:///e:/AI%20Project/new-api/Dockerfile)）

**泛型：**
- 在能减少样板代码处使用泛型：
  - `common.GetContextKeyType[T any]`，位于 [common/gin.go](file:///e:/AI%20Project/new-api/common/gin.go)
  - `common.GetPointer[T any](v T) *T`，位于 [common/utils.go](file:///e:/AI%20Project/new-api/common/utils.go)

## 导入组织

标准 Go 分组，组间用空行分隔（运行 `goimports`）：

```go
import (
    // 1. 标准库
    "errors"
    "fmt"
    "net/http"

    // 2. 第三方
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"

    // 3. 内部（github.com/QuantumNous/new-api/...）
    "github.com/QuantumNous/new-api/common"
    "github.com/QuantumNous/new-api/constant"
    "github.com/QuantumNous/new-api/model"
)
```

**为副作用使用的空白导入：**
- `_ "github.com/QuantumNous/new-api/setting/performance_setting"`，位于 [main.go](file:///e:/AI%20Project/new-api/main.go)（运行包的 `init()`）
- `_ "github.com/QuantumNous/new-api/oauth"`，位于 [router/api-router.go](file:///e:/AI%20Project/new-api/router/api-router.go)（注册 OAuth 提供者）
- `_ "net/http/pprof"`，位于 [main.go](file:///e:/AI%20Project/new-api/main.go)

**路径别名：**
- `relaycommon` → `relay/common`
- `relayconstant` → `relay/constant`
- `channelconstant` → `constant`（当 `constant` 会冲突时）
- `perfmetrics` → `pkg/perf_metrics`

## 错误处理

**集中错误类型：** `types.NewAPIError`（[types/error.go](file:///e:/AI%20Project/new-api/types/error.go)）携带 `Err`、`RelayError`、`errorType`、`errorCode`、`StatusCode`、`Metadata`，以及内部 `skipRetry`/`recordErrorLog` 标志。

**构造函数（使用这些，不要直接构造结构体）：**
- `types.NewError(err, errorCode, opts...)` —— 通用，通过 `errors.As` 保留被包装的 `*NewAPIError`
- `types.NewOpenAIError(err, errorCode, statusCode, opts...)` —— 包装为 OpenAI 风格错误
- `types.WithOpenAIError(openAIError, statusCode, opts...)` —— 从解析出的 `OpenAIError` 构造
- `types.WithClaudeError(claudeError, statusCode, opts...)` —— 从解析出的 `ClaudeError` 构造
- `types.NewErrorWithStatusCode(err, errorCode, statusCode, opts...)`
- `types.InitOpenAIError(errorCode, statusCode, opts...)` —— 尚无底层 err

**函数式选项**（`NewAPIErrorOptions`）：
- `types.ErrOptionWithSkipRetry()` —— 标记为不可重试
- `types.ErrOptionWithNoRecordErrorLog()` —— 抑制错误日志持久化
- `types.ErrOptionWithStatusCode(int)`
- `types.ErrOptionWithHideErrMsg(replaceStr)` —— 替换消息（当 `common.DebugEnabled` 时输出到 debug 日志）

**控制器响应辅助函数**（位于 [common/gin.go](file:///e:/AI%20Project/new-api/common/gin.go)）：
- `common.ApiSuccess(c, data)` → `{"success":true,"message":"","data":...}` HTTP 200
- `common.ApiError(c, err)` → `{"success":false,"message":err.Error()}` HTTP 200
- `common.ApiErrorMsg(c, msg)` → `{"success":false,"message":msg}` HTTP 200
- `common.ApiErrorI18n(c, key, args...)` → 翻译后的消息（面向用户错误时优先使用）
- `common.ApiSuccessI18n(c, key, data, args...)` → 翻译后的成功消息

**API 信封约定：** 所有 `/api/*` JSON 响应均返回 HTTP 200 与 `{success, message, data}`。不要在 `/api` 组上为应用层错误使用 HTTP 4xx。中转（OpenAI 兼容）端点可通过 `*types.NewAPIError` 返回正确的 HTTP 状态码。

**错误包装：**
- 使用 `errors.Is` / `errors.As`（该类型实现了 `Unwrap()`）
- 用 `fmt.Errorf("...: %w", err)` 或 `github.com/pkg/errors`（`errors.Wrap`、`errors.New`）包装
- 哨兵错误：`common.ErrRequestBodyTooLarge`、`model.ErrDatabase`、`model.ErrUserEmptyCredentials`

**敏感数据脱敏：**
- `common.MaskSensitiveInfo(str)` 从错误消息中剥离 key/密钥
- `NewAPIError.MaskSensitiveError()` 在返回客户端前应用
- 例外：`ErrorCodeCountTokenFailed` 原样返回（用于调试）

## 日志

**框架：** 在 `gin.DefaultWriter` / `gin.DefaultErrorWriter` 之上的自定义薄层（见 [common/sys_log.go](file:///e:/AI%20Project/new-api/common/sys_log.go) 和 [logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go)）。

**函数：**
- `common.SysLog(s string)` —— `[SYS] timestamp | msg` 输出到 stdout
- `common.SysError(s string)` —— `[SYS] timestamp | msg` 输出到 stderr
- `common.FatalLog(v ...any)` —— 输出 `[FATAL] ...` 后 `os.Exit(1)`
- `logger.LogInfo(ctx, msg)` / `logger.LogWarn(ctx, msg)` / `logger.LogError(ctx, msg)` —— `[LEVEL] timestamp | request_id | msg`
- `logger.LogDebug(ctx, msg, args...)` —— 仅当 `common.DebugEnabled == true` 时（printf 风格）
- `logger.LogJson(ctx, msg, obj)` —— 仅 debug 的 JSON 转储

**模式：**
- 通过 `gin.DefaultWriter`/`gin.DefaultErrorWriter` 写入时总是先获取 `common.LogWriterMu.RLock()`（日志轮转时写入器可能被替换）。`SysLog`/`SysError`/`FatalLog` 已内置此操作。
- 日志文件轮转：[logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go) 的 `SetupLogger()` 在 `maxLogCount = 1000000` 行时和启动时轮转，受 `setupLogLock` 保护。
- 截断大体积内容：中转错误处理器将上游响应体截断为 `common.LocalLogContentLimit`，并加上 `[truncated]` + `original_length=N` 标记（约定见 [service/error_test.go](file:///e:/AI%20Project/new-api/service/error_test.go)）。
- 包含 request id：`logger` 从上下文中读取 `common.RequestIdKey`；`middleware.RequestId()` 设置它。
- 绝不记录原始密钥/key —— 使用 `common.MaskSensitiveInfo` 或 `Token.GetMaskedKey()`。

## i18n 约定

**后端（Go）：**
- 库：`github.com/nicksnyder/go-i18n/v2` + `golang.org/x/text/language`（见 [i18n/i18n.go](file:///e:/AI%20Project/new-api/i18n/i18n.go)）
- 内嵌语言文件：`i18n/locales/{en,zh-CN,zh-TW}.yaml`，通过 `//go:embed locales/*.yaml`
- **默认语言：** `en`（不支持时回退）
- 消息键为 [i18n/keys.go](file:///e:/AI%20Project/new-api/i18n/keys.go) 中的常量，按域命名空间化：
  - `common.*`、`auth.*`、`token.*`、`redemption.*`、`user.*`、`quota.*` 等
- **在边界处翻译：** 使用 `i18n.T(c, key, args...)` 或 `common.ApiErrorI18n(c, i18n.MsgXxx)`。绝不在处理函数中硬编码面向用户的英文。
- 语言检测顺序（见 [middleware/i18n.go](file:///e:/AI%20Project/new-api/middleware/i18n.go) + `GetLangFromContext`）：
  1. 用户设置（`dto.UserSetting.Language`，由 `TokenAuth` 设置）
  2. 从 DB/缓存懒加载的用户语言（`i18n.SetUserLangLoader`）
  3. `Accept-Language` 头（由 `i18n.ParseAcceptLanguage` 解析）
  4. 默认 `en`
- `common.TranslateMessage` 是一个 `var` 函数，由 `i18n.Init()` 设置以避免 `common`→`i18n` 导入循环。**初始化后不要替换它。**
- `i18n.IsSupported(lang)`、`i18n.NormalizeLang(lang)`、`i18n.SupportedLanguages()` 是仅有的支持语言判断入口。
- YAML 键格式：`domain.message_id: "Text with {{.Var}} placeholder"`（见 [i18n/locales/en.yaml](file:///e:/AI%20Project/new-api/i18n/locales/en.yaml)）。

**前端（React）：**
- 独立的 i18n 系统，位于 `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi}.json`（六种语言），扁平的 `"translation"` 对象
- 键为英文源字符串；`en.json` 是基础，`zh` 为回退
- **绝不直接编辑 locale JSON** —— 所有写入都通过 `add-missing-keys.mjs` 脚本然后 `bun run i18n:sync`（由 [.agents/skills/i18n-translate/SKILL.md](file:///e:/AI%20Project/new-api/.agents/skills/i18n-translate/SKILL.md) 的 `i18n-translate` skill 强制执行）
- 在 `web/default/` 下运行：`bun run i18n:sync` 规范化排序并写入 `_reports/_sync-report.json`

## 注释

**何时写注释：**
- 中文注释可用于领域说明（`// 热更新配置`、`// 兼容`）；公开 API/接口契约使用英文
- `types/error.go`、`i18n/i18n.go`、`relay/channel/adapter.go` 中每个导出函数都有英文文档注释
- 使用 `// TODO implement me` / `// not supported` 标记未实现的适配器方法（例如 [relay/channel/moonshot/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor.go)）

**JSDoc/TSDoc：**
- Go 在导出标识符上使用标准 `// FunctionName ...` 文档注释
- 前端未强制 TSDoc 约定

## 函数设计

**大小：** 无硬性上限；控制器通常 50-150 行。当逻辑被复用时抽取辅助函数（`buildMaskedTokenResponse`、`decodeAPIResponse`）。

**参数：**
- 处理函数总是以 `c *gin.Context` 作为唯一参数
- 适配器方法接收 `(c *gin.Context, info *relaycommon.RelayInfo, req *dto.XxxRequest)`（见 [relay/channel/adapter.go](file:///e:/AI%20Project/new-api/relay/channel/adapter.go)）
- 可选字段优先用 `*T`（例如 `Token.AllowIps` 用 `*string`），使用 `common.GetPointer[T](v)` 取地址

**返回值：**
- `(result, error)` 标准 Go
- 中转适配器的 `Convert*` 返回 `(any, error)`，`DoResponse` 返回 `(usage any, err *types.NewAPIError)`（类型化错误，而非 `error`）
- 控制器辅助函数不返回 —— 它们通过 `common.ApiSuccess`/`common.ApiErrorI18n` 直接写入 `c`

## 模块设计

**导出：**
- 每个关注点一个包；无 barrel 文件
- `controller/` 导出处理函数，而非类型
- `model/` 导出 GORM 模型结构体 + 数据访问函数
- `dto/` 导出请求/响应 DTO 和 `Validate()` 方法
- `types/` 导出横切类型（`NewAPIError`、`RelayFormat`、`PriceData`）

**初始化模式：**
- 需要启动的包使用 `func init()`（例如 `common.Validate`、`common.TranslateMessage` 默认值）
- `i18n.Init()` 通过 `sync.Once`（`initOnce`）保证幂等
- `model.InitDB()`、`model.InitOptionMap()`、`model.InitChannelCache()` 在 `main.InitResources()` 中按依赖顺序显式调用

**适配器接口：** 每个上游提供者在 `relay/channel/<provider>/adaptor.go` 中实现 `channel.Adaptor`（异步任务额外实现 `TaskAdaptor`）。新提供者放在 `relay/channel/<provider>/` 下，通过中央适配器工厂注册。

## 响应与分页约定

**信封：** `/api/*` 返回 HTTP 200 与 `{"success": bool, "message": string, "data": any}`。

**分页：** 使用 `common.PageInfo`（[common/page_info.go](file:///e:/AI%20Project/new-api/common/page_info.go)）：
```go
pageInfo := common.GetPageQuery(c)   // 读取 ?p=&page_size=&ps=&size=
items, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
pageInfo.SetTotal(int(total))
pageInfo.SetItems(items)
common.ApiSuccess(c, pageInfo)
```
- `page_size` 上限 100，默认为 `common.ItemsPerPage`（10）
- `GetStartIdx() = (Page-1) * PageSize`

## 并发约定

- **协程：** 使用 `github.com/bytedance/gopkg/util/gopool` 的 `gopool.Go(func(){...})`（托管池，而非原始 `go`）。见 [main.go](file:///e:/AI%20Project/new-api/main.go) 和 [logger/logger.go](file:///e:/AI%20Project/new-api/logger/logger.go)。
- **共享 map：** 用 `sync.RWMutex` 保护（例如 `common.OptionMapRWMutex`、`i18n.mu`）
- **原子单例：** [common/constants.go](file:///e:/AI%20Project/new-api/common/constants.go) 中用 `atomic.Value` 存 `themeValue`
- **Once：** `sync.Once` 用于幂等初始化（`i18n.initOnce`）
- **上下文取消：** 长时间运行操作传递 `context.Context`；尊重 `ctx.Done()`

## 配置约定

- **环境变量：** 通过 [common/env.go](file:///e:/AI%20Project/new-api/common/env.go) 的 `common.GetEnvOrDefault(name, default)`、`GetEnvOrDefaultString`、`GetEnvOrDefaultBool` 读取。所有环境变量初始化在 `common.InitEnv()` + `common.initConstantEnv()`（[common/init.go](file:///e:/AI%20Project/new-api/common/init.go)）中。
- **标志：** `flag.Int("port", ...)`、`flag.String("log-dir", ...)` 定义在 [common/init.go](file:///e:/AI%20Project/new-api/common/init.go)
- **`.env` 文件：** 通过 `github.com/joho/godotenv` 在 `main.InitResources()` 中加载 —— `.env.example` 文档了可用变量（不要读取 `.env` 内容）
- **运行时设置：** 通过 `model.OptionMap` 存储在 DB 中，通过 `setting/*` 包暴露（例如 `operation_setting`、`ratio_setting`）

---

*规范分析：2026-07-01*
