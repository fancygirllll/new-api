<!-- refreshed: 2026-07-01 -->
# 架构

**分析日期：** 2026-07-01

## 系统概览

new-api 是一个用 Go (Gin) 构建的 AI API 网关/代理，在统一的 OpenAI/Claude/Gemini 兼容 API 之后聚合了 40+ 上游 AI 供应商（OpenAI、Anthropic Claude、Gemini、Azure、AWS Bedrock、Vertex AI 等）。它附加了用户管理、基于 token 的计费、限流、渠道选择、管理后台以及 React 前端。

```text
┌──────────────────────────────────────────────────────────────────────┐
│                          HTTP 客户端 / 前端                            │
│   (OpenAI SDK, Claude SDK, Gemini SDK, React default+classic 主题)    │
└──────────────┬───────────────────────────────────────┬───────────────┘
               │ /v1/* /v1beta/* /mj /suno  (relay)     │ /api/* /pg/* /dashboard/*
               ▼                                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                             路由层 (Gin)                              │
│  `router/main.go` → SetApiRouter / SetRelayRouter / SetDashboardRouter │
│                    / SetVideoRouter / SetWebRouter                     │
├──────────────────┬───────────────────────┬───────────────────────────┤
│  中间件链        │   控制器处理器        │   内嵌静态前端            │
│ `middleware/*.go`│   `controller/*.go`   │  `web/default/dist`      │
└────────┬─────────┴───────────┬───────────┴──────────┬────────────────┘
         │                      │                       │
         ▼                      ▼                       │
┌──────────────────────────────────────────────────────┐│
│                  服务层 (业务逻辑)                    ││
│  `service/billing.go` `service/channel_select.go`    ││
│  `service/authz/` (casbin RBAC) `service/passkey/`   ││
└────────┬─────────────────────────────────┬───────────┘│
         │                                 │            │
         ▼                                 ▼            ▼
┌─────────────────────────┐   ┌──────────────────────────────────────┐
│   中继 / 代理核心       │   │        模型层 (GORM v2)               │
│      `relay/*.go`       │   │       `model/*.go`                    │
│  Adaptor 接口 +         │◄──┤  User, Token, Channel, Ability,       │
│  40+ 供应商适配器       │   │  Log, Task, Subscription, Pricing      │
│  `relay/channel/<prov>/`│   │  + 内存态渠道缓存                     │
└────────────┬────────────┘   └───────────────┬──────────────────────┘
             │                                  │
             ▼                                  ▼
┌─────────────────────────┐   ┌──────────────────────────────────────┐
│  上游 AI 供应商         │   │  SQLite / MySQL / PostgreSQL + Redis │
│  (OpenAI, Claude, 等)   │   │  (+ 可选 ClickHouse 日志库)           │
└─────────────────────────┘   └──────────────────────────────────────┘
```

## 组件职责

| 组件 | 职责 | 文件 |
|------|------|------|
| 入口 / 引导 | 加载环境变量，初始化 DB/Redis/authz/i18n，构建 Gin 服务器，内嵌前端 | `main.go` |
| 路由装配 | 注册所有路由分组 + 主题感知的静态资源服务 | `router/main.go` |
| 中继 API 路由 | OpenAI/Claude/Gemini/MJ/Suno 兼容端点 | `router/relay-router.go` |
| 管理 / 后台 API | 用户、渠道、令牌、计费、选项、订阅 | `router/api-router.go` |
| 认证中间件 | 会话 + 令牌 + passkey 认证、角色门控、审计 | `middleware/auth.go` |
| 渠道分发器 | 按 group+model+affinity 选择上游渠道 | `middleware/distributor.go` |
| 中继编排器 | 请求解析 → 计价 → 预扣 → 重试循环 → 结算 | `controller/relay.go` |
| 适配器工厂 | 将 `apiType` 映射到供应商适配器实现 | `relay/relay_adaptor.go` |
| 适配器契约 | 每个渠道都需实现的、与供应商无关的接口 | `relay/channel/adapter.go` |
| 请求上下文载体 | `RelayInfo` 持有 token/user/channel/billing 状态 | `relay/common/relay_info.go` |
| 计费会话 | 预扣 → 结算/退还配额流程 | `service/billing.go`, `service/billing_session.go` |
| 渠道选择缓存 | 内存中的 group→model→channels 索引 | `model/channel_cache.go` |
| 能力映射 | 基于 DB 的 group+model+channel 启用矩阵 | `model/ability.go` |
| RBAC 权限执行 | 用于渠道/管理操作的 casbin 权限 | `service/authz/enforcer.go` |
| 数据库初始化 | 多数据库方言选择、列引用、迁移 | `model/main.go` |
| JSON 包装器 | 在整个代码库中强制使用统一的 JSON 编解码器 | `common/json.go` |

## 模式概览

**总体：** 分层架构（Router → Controller → Service → Model），使用策略/适配器模式做供应商抽象，并使用工厂模式做适配器实例化。

**关键特征：**
- 分层：每层只向下调用（Router → Controller → Service → Model），存在一个被注入的回调用于打破 service↔relay 循环依赖（见"架构约束"）。
- 与供应商无关的中继：每个上游供应商都实现 `channel.Adaptor` 接口；`relay.GetAdaptor(apiType)` 返回具体实例。新增供应商 = 在 `relay/channel/` 下新建子包 + 在工厂 switch 中加一个 case。
- 失败重试与渠道降级：中继循环对同一分组内的其他渠道进行失败重试，由 `shouldRetry` / `shouldRetryTaskRelay` 控制。
- 计费即会话：在上游调用前预扣配额，响应后再结算（补扣/退还）；失败时通过 `relayInfo.Billing.Refund` 退款。
- 多数据库方言无关：所有 DB 代码必须同时能在 SQLite、MySQL、PostgreSQL 上运行（ClickHouse 仅用于日志）。
- 内嵌 SPA：两套前端主题都编译为 `dist/` 并通过 `//go:embed` 嵌入到 Go 二进制中。
- Master/worker 节点：定时任务与初始化只在 master 节点运行（`common.IsMasterNode`）。

## 分层

**Router：**
- 目的：定义 HTTP 路由和每个分组的中间件链；服务内嵌前端。
- 位置：`router/`
- 包含：`main.go`、`api-router.go`、`relay-router.go`、`dashboard.go`、`web-router.go`、`video-router.go`、`channel-router.go`、`authz-router.go`
- 依赖：`controller`、`middleware`
- 被谁调用：`main.go`（`SetRouter` 的唯一调用方）

**Middleware：**
- 目的：横切请求处理 —— 认证、限流、CORS、gzip、分发、审计、i18n、request-id、body 大小限制。
- 位置：`middleware/`
- 包含：`auth.go`（TokenAuth/UserAuth/AdminAuth/RootAuth/RequirePermission）、`distributor.go`、`rate-limit.go`、`model-rate-limit.go`、`cors.go`、`gzip.go`、`audit.go`、`i18n.go`、`request-id.go`、`request_body_limit.go`、`secure_verification.go`、`turnstile-check.go`、`cache.go`、`stats.go`、`performance.go`
- 依赖：`model`、`service`、`common`、`i18n`
- 被谁调用：路由定义和路由分组通过 `router.Use(...)`

**Controller：**
- 目的：请求处理器 —— 解析输入、调用 service/model、格式化 HTTP 响应。
- 位置：`controller/`
- 包含：`relay.go`（Relay/RelayTask/RelayMidjourney 编排）、`user.go`、`channel.go`、`token.go`、`billing.go`、`topup_*.go`、`subscription*.go`、`log.go`、`pricing.go`、`model.go`、`playground.go`、`passkey.go`、`twofa.go`、`oauth.go`、`system_task*.go` 等
- 依赖：`service`、`model`、`relay`、`dto`、`types`、`common`
- 被谁调用：路由 handler

**Service：**
- 目的：不属于单一 handler 的业务逻辑 —— 计费、渠道选择、敏感词检查、token 计数、通知、支付。
- 位置：`service/` 加子包 `authz/`、`passkey/`、`relayconvert/`
- 包含：`billing.go`、`billing_session.go`、`pre_consume_quota.go`、`channel_select.go`、`channel_affinity.go`、`channel.go`、`quota.go`、`text_quota.go`、`task_billing.go`、`tiered_settle.go`、`sensitive.go`、`token_counter.go`、`tokenizer.go`、`epay.go`、`webhook.go`、`system_task.go`、`task_polling.go`、`error.go` 等
- 依赖：`model`、`relay`（通过注入的 `GetTaskAdaptorFunc`）、`common`、`setting`
- 被谁调用：`controller`、`middleware`

**Model：**
- 目的：GORM 实体、DB 访问、连接管理、内存缓存。
- 位置：`model/`
- 包含：`main.go`（DB 初始化 + 方言辅助函数）、`user.go`、`token.go`、`channel.go`、`ability.go`、`channel_cache.go`、`user_cache.go`、`token_cache.go`、`log.go`、`task.go`、`system_task.go`、`pricing.go`、`subscription.go`、`option.go`、`casbin_rule.go`、`authz_role.go` 等
- 依赖：`common`、`constant`、`dto`、GORM 驱动
- 被谁调用：所有上层

**Relay：**
- 目的：AI 代理核心 —— 将客户端请求转换为上游格式、执行、转回响应、流式、计费。
- 位置：`relay/` 加 `relay/channel/<provider>/`、`relay/common/`、`relay/helper/`、`relay/constant/`、`relay/common_handler/`、`relay/reasonmap/`
- 包含：`relay_adaptor.go`（工厂）、`compatible_handler.go`（文本）、`claude_handler.go`、`gemini_handler.go`、`image_handler.go`、`audio_handler.go`、`embedding_handler.go`、`rerank_handler.go`、`responses_handler.go`、`mjproxy_handler.go`、`relay_task.go`、`websocket.go`；`channel/adapter.go`（接口）；`common/relay_info.go`（RelayInfo）；`helper/valid_request.go`、`helper/price.go`、`helper/common.go`
- 依赖：`dto`、`types`、`model`、`service`、`common`、`setting`
- 被谁调用：`controller`（Relay 入口）、`service`（通过注入的工厂进行任务轮询）

## 数据流

### 主请求路径（Relay —— 例如 `POST /v1/chat/completions`）

1. Gin 接收请求 → 全局中间件：`RequestId`、`PoweredBy`、`I18n`、logger、sessions（`main.go:179-192`）
2. Relay 路由分组中间件：`CORS`、`DecompressRequestMiddleware`、`BodyStorageCleanup`、`StatsMiddleware`、`RouteTag("relay")`、`SystemPerformanceCheck`、`TokenAuth`、`ModelRequestRateLimit`、`Distribute`（`router/relay-router.go:69-85`）
3. `TokenAuth` 校验 `sk-...` token，加载用户缓存，设置上下文键（`middleware/auth.go:303-434`）
4. `Distribute` 按 group+model+affinity 选择上游渠道并播种渠道上下文（`middleware/distributor.go:32`）
5. `controller.Relay(c, RelayFormatOpenAI)` 入口（`controller/relay.go:68`）
6. `helper.GetAndValidateRequest` 将 body 解析+校验为 `dto.Request`（`relay/helper/valid_request.go:21`）
7. `relaycommon.GenRelayInfo` 构建承载 token/user/channel/billing 状态的 `RelayInfo`（`relay/common/relay_info.go`）
8. 可选的敏感词检查 + `service.EstimateRequestToken` 估算 prompt token 数
9. `helper.ModelPriceHelper` 计算 `PriceData`（模型倍率 × 分组倍率 × tokens）（`relay/helper/price.go`）
10. `service.PreConsumeBilling` 创建 `BillingSession` 并预扣配额（`service/billing.go:19`）
11. 重试循环（`controller/relay.go:191-237`）：
    a. `getChannel` → 从缓存中取/刷新渠道（`controller/relay.go:293`）
    b. `relay.GetAdaptor(info.ApiType)` 返回供应商适配器（`relay/relay_adaptor.go:54`）
    c. `adaptor.Init` → `ConvertOpenAIRequest` → `GetRequestURL` → `SetupRequestHeader` → `DoRequest` → `DoResponse`
    d. 成功：返回；失败：`processChannelError`（可能自动禁用渠道）+ `shouldRetry` 决定是否重试
12. 响应时：`service.PostTextConsumeQuota` 结算计费（补扣/退还差额）并记录消费
13. 终态失败时：`relayInfo.Billing.Refund` 退还预扣配额；错误按 `relayFormat`（OpenAI/Claude 形态）格式化为 JSON

### 异步任务流（Midjourney / Suno / Video —— 例如 `POST /suno/submit/:action`）

1. Router → `controller.RelayTask`（`controller/relay.go:486`）
2. `relaycommon.GenRelayInfo`（RelayFormatTask）→ `relay.ResolveOriginTask`
3. `relay.GetTaskAdaptor(platform)` 返回 `channel.TaskAdaptor`（`relay/relay_adaptor.go:138`）
4. 重试循环：`BuildRequestURL/Header/Body` → `DoRequest` → `DoResponse` 返回 `taskID`
5. 成功时：`service.SettleBilling`、`service.LogTaskConsumption`、`model.InitTask(...).Insert()` 持久化任务
6. 后台轮询：`service.StartSystemTaskRunner` 定期调用 `service.RunTaskPollingOnce` → 适配器 `FetchTask` + `ParseTaskResult` → `AdjustBillingOnComplete` 结算配额差额 → 更新任务行

### 管理 / 后台流程（例如 `GET /api/channel/`）

1. `SetApiRouter` 注册 `/api/channel/` 分组并附加 `AdminAuth`（`router/channel-router.go:19-37`）
2. 每个路由还额外要求 `middleware.RequirePermission(permission)`（casbin 校验，`middleware/auth.go:199-213`）
3. `authHelper` 强制执行 session/access-token + 角色 + 审计（`middleware/auth.go:37-168`）
4. Controller（如 `controller.GetAllChannels`）通过 `model.*` 读取并返回 JSON

**状态管理：**
- 请求级状态存放在 `*gin.Context`（上下文键定义在 `constant/context_key.go`）。
- 长期可变状态：`model.OptionMap`（设置项，由 RWMutex 保护）、内存渠道缓存（`model/channel_cache.go` 的 `group2model2channels` + `channelsIDM`）、casbin `SyncedEnforcer`（`service/authz/enforcer.go`）。
- 后台 goroutine 定期刷新状态（`model.SyncChannelCache`、`model.SyncOptions`、`authz.StartPolicySync`、系统任务 runner）—— 见 `main.go:98-145`。

## 关键抽象

**Adaptor 接口：**
- 目的：用一组统一的转换 + 传输方法把单个上游 AI 供应商抽象掉。
- 示例：`relay/channel/openai/adaptor.go`、`relay/channel/claude/adaptor.go`、`relay/channel/gemini/adaptor.go`、`relay/channel/aws/adaptor.go`，以及 `relay/channel/` 下的其他 30+ 个
- 模式：Strategy + Adapter。`relay/relay_adaptor.go:54-128`（`GetAdaptor`）是工厂 switch。

**TaskAdaptor 接口：**
- 目的：异步任务供应商（Midjourney、Suno、Kling、Sora、Vidu 等）—— 提交 + 轮询生命周期。
- 示例：`relay/channel/task/suno/`、`relay/channel/task/kling/`、`relay/channel/task/sora/`、`relay/channel/task/vertex/`
- 模式：Strategy。工厂在 `relay/relay_adaptor.go:138-167`（`GetTaskAdaptor`）。

**RelayInfo：**
- 目的：承载中继流水线处理单个请求所需的一切 —— token、user、group、channel 元信息、计价、计费会话、重试索引、流式状态。
- 示例：`relay/common/relay_info.go`
- 模式：在流水线中传递的 Context 对象。

**BillingSession：**
- 目的：封装"预扣 → 结算/退还"，让钱包和订阅计费共享同一套代码路径。
- 示例：`service/billing_session.go`、`service/billing.go`
- 模式：基于 `BillingSource`（钱包 vs 订阅）的 Strategy。

**Ability / 渠道缓存：**
- 目的：将 (group, model) 映射到已启用渠道 ID 的有序列表，供分发器做加权优先级选择。
- 示例：`model/ability.go`（DB 实体）、`model/channel_cache.go`（内存索引）
- 模式：带周期性同步的 read-through 缓存。

**Casbin enforcer：**
- 目的：RBAC 权限检查（`authz.Can(userID, role, permission)`），接入 `middleware.RequirePermission`。
- 示例：`service/authz/enforcer.go`、`service/authz/permission.go`、`service/authz/resources_channel.go`
- 模式：Policy-as-data（gorm adapter），带跨节点周期性同步。

## 入口点

**进程入口（`main.go:51`）：**
- 位置：`main.go`
- 触发：`go run` / 编译后的二进制 / Docker
- 职责：`InitResources`（环境变量、logger、倍率设置、HTTP client、token encoder、DB、authz、options、cache、i18n、OAuth provider）；启动后台 goroutine（渠道缓存同步、option 同步、策略同步、配额数据、系统实例上报、codex 凭证刷新、订阅重置、系统任务 runner、批量更新器、pprof、pyroscope）；用 recovery/request-id/i18n/logger/session 中间件构建 Gin 服务器；`router.SetRouter`；`server.Run`。

**HTTP 入口（`router/main.go:15`）：**
- 位置：`router/main.go`
- 触发：每一个入站 HTTP 请求
- 职责：装配五个路由分组（API、Dashboard、Relay、Video、Web）以及主题感知的静态资源回退。

**Relay 入口（`controller/relay.go:68`）：**
- 位置：`controller.Relay`、`controller.RelayTask`、`controller.RelayMidjourney`
- 触发：relay 路由 handler
- 职责：编排 解析 → 计价 → 预扣 → 重试循环 → 结算。

## 架构约束

- **多数据库兼容：** 所有 DB 代码必须同时在 SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6 上运行（ClickHouse 仅用于日志库，通过 `LOG_SQL_DSN`）。优先使用 GORM 方法而非裸 SQL；保留字列和布尔值使用 `model/main.go` 中的 `commonGroupCol`/`commonKeyCol`/`commonTrueVal`/`commonFalseVal`；用 `common.UsingMainDatabase(...)` / `common.UsingLogDatabase(...)` 做分支。详见 `AGENTS.md` "Database compatibility"。
- **JSON 编解码器：** 所有 marshal/unmarshal 必须走 `common.Marshal` / `common.Unmarshal` / `common.UnmarshalJsonStr` / `common.DecodeJsonStr`（`common/json.go`）。不要在业务代码中直接调用 `encoding/json`。
- **Relay DTO 零值：** 客户端解析的请求结构体中的可选标量字段必须使用指针类型加 `omitempty`，以便把显式的 `0`/`false` 原样传给上游（`AGENTS.md` "Relay and provider behavior"）。
- **Service → relay 导入循环：** `service` 不能直接导入 `relay`。该循环通过 `main.go:131-137` 注入 `service.GetTaskAdaptorFunc` 来打破 —— 不要在 `service` 中加直接的 `relay` import。
- **Master/worker 执行：** 后台定时任务（渠道测试、上游模型更新、异步任务轮询）只在 master 节点运行，并用 DB 租约做跨实例去重（`controller.RegisterScheduledSystemTasks`、`service.StartSystemTaskRunner`、`common.IsMasterNode`）。
- **线程模型：** 单进程、多 goroutine。CPU 密集工作和"即发即弃"副作用使用 `gopool.Go`（`github.com/bytedance/gopkg/util/gopool`）。casbin enforcer 是 `SyncedEnforcer`；渠道缓存和 option map 由 `sync.RWMutex` 保护。
- **全局状态：** `model.OptionMap` + `model.OptionMapRWMutex`（`model/option.go`）、`model.group2model2channels` / `model.channelsIDM` / `model.channel2advancedCustomConfig` + `model.channelSyncLock`（`model/channel_cache.go`）、`service.authz.enforcer` + `enforcerMu`（`service/authz/enforcer.go`）、`common.themeValue`（`common/constants.go`）、`model.DB` / `model.LOG_DB`（`model/main.go`）。
- **内嵌前端：** 二进制通过 `//go:embed` 内嵌 `web/default/dist` 和 `web/classic/dist`（`main.go:39-49`）。Go 构建前必须先构建前端。
- **受保护的标识符：** 项目名（new-api）和组织名（QuantumNous）标识符 —— 模块路径、品牌、版权 —— 受保护，不得改名（`AGENTS.md` "Project Governance"）。

## 反模式

### 在业务代码中直接调用 `encoding/json`

**现象：** 某个新 handler 用了 `encoding/json` 的 `json.Marshal` / `json.Unmarshal`，而不是包装器。
**为何不对：** 代码库把所有 JSON 都路由到 `common/json.go`（中心化配置了 `sonic`/`goccy` 变体）以保证一致的行为和性能。直接调用会绕过它，在边缘情况下可能出现行为分歧。
**应该这样做：** 使用 `common/json.go` 的 `common.Marshal(v)`、`common.Unmarshal(data, &v)`、`common.UnmarshalJsonStr(s, &v)`、`common.DecodeJsonStr(reader, &v)`。`encoding/json` 的类型只作为类型引用。

### 写了只针对单一数据库的 SQL，没有跨库兜底

**现象：** 裸 SQL 用了 MySQL 反引号引用，或 PostgreSQL 专有的操作符；或某个 model 依赖 `gorm:"default:true"` 布尔 tag。
**为何不对：** 同一份代码必须同时跑在 SQLite、MySQL、PostgreSQL 上。引用不一致或布尔值归一化不对，会让 GORM `AutoMigrate` 在每次重启时都发出多余的 `ALTER TABLE`。
**应该这样做：** 保留字和布尔值使用 `model/main.go` 的 `commonGroupCol`/`commonKeyCol`/`commonTrueVal`/`commonFalseVal`；优先用 GORM 方法；业务规则的默认值在 model hook 或 service 代码中设置，不要写在 `default:` tag 里（见 `AGENTS.md` "Database compatibility"）。

### 从 `service` 直接 import `relay`

**现象：** 某个新的 service 函数需要任务适配器，于是直接 import 了 `relay`。
**为何不对：** 这会重新引入 `service` ↔ `relay` 导入循环，而 `main.go:131-137` 就是为了打破这个循环才写的。
**应该这样做：** 使用已注入的 `service.GetTaskAdaptorFunc(platform)`（已在 `main.go` 中设置），它返回 `service.TaskPollingAdaptor`，无需 import `relay`。

### 在适配器工厂之外硬编码供应商选择

**现象：** 某个 handler 用 `if channelType == constant.ChannelTypeX` 给某个供应商开特例，而不是走适配器。
**为何不对：** 绕过了 `Adaptor` 接口和 `GetAdaptor` 工厂，导致该供应商拿不到转换/流式/计费处理，下一位开发者也找不到这条代码路径。
**应该这样做：** 在 `relay/relay_adaptor.go:54` 的 `GetAdaptor` 中加一个 case，返回一个实现了 `relay/channel/adapter.go` 的新 `<provider>.Adaptor{}`。

## 错误处理

**策略：** 类型化错误（`types.NewAPIError`）携带 HTTP 状态码、错误码、错误类型以及重试提示；在 controller 边界被转换为对应供应商形态的响应（OpenAI / Claude / Gemini / task）。

**模式：**
- `types.NewError(err, code, opts...)` 构造规范化的中继错误（`types/error.go`）。
- `ErrOptionWithSkipRetry()` 标记不允许触发渠道重试的错误。
- `service.NormalizeViolationFeeError` 和 `service.ChargeViolationFeeIfNeeded` 在中继错误外包裹计费专属的错误语义。
- `gin.CustomRecovery`（`main.go:168-176`）捕获 panic 并返回 `new_api_panic` JSON 错误，保证单个请求失败不会让进程崩溃。
- 渠道错误通过 `processChannelError` → `model.RecordErrorLog` 记录，并在 `ShouldDisableChannel` 返回 true 时触发 `service.DisableChannel`（自动禁用）。
- 任务错误使用 `dto.TaskError` 加 `LocalError` 标志（不重试，不上报上游）。

## 横切关注点

**日志：** `logger/` 中的自定义 logger（分级：`SysLog`、`LogInfo`、`LogWarn`、`LogError`、`LogDebug`），写入 `logs/` 下的文件和 stdout。请求级日志会携带 request id。敏感数据通过 `common.LocalLogPreview` 和 `err.MaskSensitiveErrorWithStatusCode` 脱敏。

**校验：** 请求校验在 `relay/helper/valid_request.go` 中按中继格式（OpenAI/Claude/Gemini/Responses/Image/Audio/Embedding/Rerank）分发。额外的字段校验使用 `github.com/go-playground/validator/v10`。匿名请求体大小由 `middleware.AnonymousRequestBodyLimit` 限制。

**认证：** `middleware/auth.go` 中有三套并行机制 —— session cookie（后台用户，通过 `gin-contrib/sessions` cookie store）、API token `sk-...`（`TokenAuth`，支持 Anthropic `x-api-key`、Gemini `x-goog-api-key`/`?key=`、MJ `mj-api-secret`、WebSocket `Sec-WebSocket-Protocol`）、access token（`New-Api-User` header）。细粒度授权通过 casbin `RequirePermission` 完成。Passkey/WebAuthn 在 `service/passkey/`。2FA TOTP 在 `controller/twofa.go` + `model/twofa.go`。OAuth 在 `oauth/`（GitHub、Discord、OIDC、LinuxDO、WeChat、Telegram），启动时从 DB 加载自定义 provider。

**i18n：** 后端使用 `nicksnyder/go-i18n/v2`，目录在 `i18n/locales/`（en、zh）；消息按请求通过 `common.TranslateMessage(c, ...)` 翻译。前端使用 `i18next` + `react-i18next`，flat JSON 在 `web/default/src/i18n/locales/`。中间件 `middleware/i18n.go` 从 `Accept-Language` header / query 检测语言。

**限流：** 全局 API 限流（`middleware.GlobalAPIRateLimit`）、全局 web 限流（`middleware.GlobalWebRateLimit`）、关键操作限流（`middleware.CriticalRateLimit`）、搜索限流、邮件验证限流、模型请求限流（`middleware/model-rate-limit.go`）。可用 Redis 时基于 Redis，否则用进程内限流器（`common/limiter/`，Redis 用 Lua 脚本）。

**审计：** `middleware/audit.go`（`beginAdminAudit`/`finishAdminAudit`）接入 `authHelper` 内部，使得每个 `AdminAuth`/`RootAuth` 写操作都会被审计，无需逐路由加中间件。Handler 在自己手动审计时可以设置 `ContextKeyAuditLogged` 跳过重复记录。

**性能监控：** `pkg/perf_metrics/` 记录中继采样和系统指标；`middleware/performance.go`（`SystemPerformanceCheck`）可在高负载时拒绝请求。Pyroscope profiling 可选（`common.StartPyroScope`）；`ENABLE_PPROF=true` 时 pprof 监听 `:8005`。

---

*架构分析：2026-07-01*
