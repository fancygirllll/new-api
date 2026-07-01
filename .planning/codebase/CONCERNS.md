# 代码库关注事项

**分析日期：** 2026-07-01

## 技术债务

**未实现的 relay adaptor 方法（100+ 个桩函数）：**
- 问题：`relay/channel/*/adaptor.go` 下的几乎每个 channel adaptor 都为可选协议转换（`ConvertClaudeRequest`、`ConvertGeminiRequest`、`ConvertAudioRequest`、`ConvertImageRequest`、`ConvertEmbeddingRequest`、`ConvertOpenAIResponsesRequest`、`ConvertRerankRequest`）包含 `//TODO implement me` 桩函数。部分桩函数返回 `errors.New("not implemented")`，但有几个会 panic。
- 文件：[zhipu/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/zhipu/adaptor.go), [baidu/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/baidu/adaptor.go), [deepseek/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/deepseek/adaptor.go), [xunfei/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/xunfei/adaptor.go), [cohere/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/cohere/adaptor.go), [dify/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/dify/adaptor.go), [mokaai/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/mokaai/adaptor.go), [cloudflare/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/cloudflare/adaptor.go), [mistral/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/mistral/adaptor.go), [claude/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/claude/adaptor.go), [aws/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/aws/adaptor.go), [moonshot/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/moonshot/adaptor.go), [vertex/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/vertex/adaptor.go), [tencent/adaptor.go](file:///e:/AI%20Project/new-api/relay/channel/tencent/adaptor.go)
- 影响：各 channel 之间的功能面不一致。部分桩函数 `panic("implement me")`（例如 `relay/channel/zhipu/adaptor.go:28`、`relay/channel/jina/adaptor.go:30`、`relay/channel/xunfei/adaptor.go:28`、`relay/channel/palm/adaptor.go:28`、`relay/channel/baidu/adaptor.go:29`、`relay/channel/cohere/adaptor.go:28`），一旦命中该代码路径就会使请求 goroutine 崩溃。该 panic 会被 `middleware/recover.go` 和 `main.go:168` 的 CustomRecovery 捕获，但会向客户端返回 500 而非干净的"不支持"错误。
- 修复方案：通过共享的 `channel.ErrNotSupported` sentinel 将所有桩函数标准化为返回 `errors.New("channel type X does not support Y")`。移除所有 `panic("implement me")`。审计哪些桩函数应返回真实错误（以便路由器返回 404/501），哪些应不可达（在 distributor 层做防护）。

**计费中脆弱的字符串匹配错误分类：**
- 问题：订阅资金错误通过对错误消息字符串做子串匹配来分类，而非使用类型化的 sentinel 错误。
- 文件：[service/billing_session.go:216-220](file:///e:/AI%20Project/new-api/service/billing_session.go#L216-L220)
- 影响：若 model 层错误消息措辞发生变化（例如 "no active subscription" → "subscription inactive"），计费路径会静默地错误分类错误，返回带重试的通用 `ErrorCodeUpdateDataError`，而非预期的 `ErrorCodeInsufficientUserQuota`（跳过重试）。用户可能被错误计费，或在不该重试时重试。
- 修复方案：定义 `model.ErrNoActiveSubscription` 和 `model.ErrSubscriptionQuotaInsufficient` sentinel 错误（代码内 TODO 中已提及）。由资金层包装/返回它们。用 `errors.Is(err, model.ErrNoActiveSubscription)` 替换 `strings.Contains(errMsg, ...)`。

**双前端维护负担（classic + default）：**
- 问题：仓库内置两套完整的 React 前端（`web/default/` 和 `web/classic/`），均在 `main.go:39-49` 嵌入二进制。Classic 处于弃用状态，多个 TODO 提及移除。
- 文件：[main.go:45-49](file:///e:/AI%20Project/new-api/main.go#L45-L49), [controller/log.go:156](file:///e:/AI%20Project/new-api/controller/log.go#L156), [router/api-router.go:268](file:///e:/AI%20Project/new-api/router/api-router.go#L268)
- 影响：前端维护工作翻倍。遗留路由和 handler 仅为 classic 保留，例如 `controller/log.go` 保留了一个 `/api/log/...` handler，并带有明确的"remove once the classic frontend is removed" TODO。任何 UI bug 修复都必须应用两次。
- 修复方案：为 classic 选定一个移除里程碑。在该里程碑删除 `web/classic/`，移除 `classicBuildFS`/`classicIndexPage` 嵌入及 `common/constants.go` 中 `SetTheme("classic")` 的代码路径，并删除 TODO 标记的 classic 专用 API handler。

**超过 1000 行的 God files：**
- 问题：多个源文件体积过大，难以安全地审查、测试和扩展。
- 文件：[controller/channel.go](file:///e:/AI%20Project/new-api/controller/channel.go)（1960 行）、[relay/common/override.go](file:///e:/AI%20Project/new-api/relay/common/override.go)（1937 行）、[relay/channel/gemini/relay-gemini.go](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go)（1652 行）、[controller/user.go](file:///e:/AI%20Project/new-api/controller/user.go)（1332 行）、[model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go)（1282 行）、[model/channel.go](file:///e:/AI%20Project/new-api/model/channel.go)（1014 行）、[dto/openai_request.go](file:///e:/AI%20Project/new-api/dto/openai_request.go)（970 行）、[relay/channel/claude/relay-claude.go](file:///e:/AI%20Project/new-api/relay/channel/claude/relay-claude.go)（942 行）、[service/convert.go](file:///e:/AI%20Project/new-api/service/convert.go)（938 行）、[service/channel_affinity.go](file:///e:/AI%20Project/new-api/service/channel_affinity.go)（923 行）
- 影响：任何变更的认知负担都很高。合并冲突频繁。难以编写聚焦的单元测试。`relay/common/override.go` 是最严重的，1937 行 override/transform 逻辑包含深层嵌套分支，并在第 918、935、949 行有 `err = nil` 吞错误。
- 修复方案：按关注点拆分 —— `controller/channel.go` 拆为 `channel_crud.go`、`channel_test.go`、`channel_models.go`；`override.go` 按请求类型拆为多个 override 文件。仅在因其他原因触及该文件时执行（不要冷重构）。

**悬空的迁移 TODO：**
- 问题：注释掉的 `ALTER TABLE channels MODIFY model_mapping TEXT;` 留在 SQLite/MySQL 迁移路径中，附带"delete this line when most users have upgraded" TODO。
- 文件：[model/main.go:211](file:///e:/AI%20Project/new-api/model/main.go#L211)
- 影响：死迁移代码；不清楚何时才算"most users have upgraded"。使迁移 bootstrap 杂乱，并可能让未来维护者困惑。
- 修复方案：在 `common.Version` 历史中定义一个版本截止线，在该截止线发布后删除此行；或将其转换为带已应用记录标志的受防护一次性迁移。

**未统一的 api_version 上下文分发：**
- 问题：`middleware/distributor.go` 有一个 `switch channel.Type` 从被重载的 `channel.Other` 字段设置上下文键（`api_version`、`region`、`plugin`、`bot_id`）。字段名 `Other` 在不同 channel 类型下含义不同。
- 文件：[middleware/distributor.go:485-503](file:///e:/AI%20Project/new-api/middleware/distributor.go#L485-L503)
- 影响：新增需要自身 version 字段的 channel 类型必须编辑此 switch。`channel.Other` 字段是无类型 `string` —— 无校验，无文档说明各 channel 类型期望什么。
- 修复方案：将 `channel.Other` 替换为类型化的 `ChannelSettings` 结构字段（或小 map），并将 context-key 映射移入各 adaptor 的 `Init()`，使 distributor 不再需要该 switch。

## 已知 Bug

**Channel 计费不支持 Azure：**
- 症状：`UpdateAllChannelsBalance` 跳过 Azure channel 余额更新 —— `controller/channel-billing.go:466` 的 TODO 显示类型过滤器被注释掉。
- 文件：[controller/channel-billing.go:466-469](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L466-L469)
- 触发：任何 Azure channel；余额永远不会被上报/低余额自动封禁。
- 变通：手动余额监控。修复：实现 Azure 余额抓取，或文档化 Azure 不受支持并通过显式排除而非死代码处理。

**`UpdateAllChannelsBalance` 阻塞请求：**
- 症状：HTTP handler 内联运行完整的同步余额更新循环。
- 文件：[controller/channel-billing.go:484-486](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L484-L486)
- 触发：管理员触发"update all channel balances"。channel 数量多时会导致 HTTP 请求超时。
- 变通：无。修复：通过系统任务 runner（已在 `service.StartSystemTaskRunner` 可用）分发，而非内联运行。

## 安全考虑

**Session cookie `Secure: false`：**
- 风险：Session cookie 通过明文 HTTP 传输，在非 TLS 网络/错误配置的代理上可被截获。
- 文件：[main.go:189](file:///e:/AI%20Project/new-api/main.go#L189)
- 当前缓解：设置了 `SameSite: http.SameSiteStrictMode` 和 `HttpOnly: true`。
- 建议：生产环境（`GIN_MODE=release`）下默认 `Secure` 为 `true`；通过环境变量（`SESSION_COOKIE_SECURE`）允许本地 HTTP 开发覆盖。

**重启时静默使用随机 `SessionSecret` / `CryptoSecret`：**
- 风险：若 `SESSION_SECRET` 未设置，`common.SessionSecret` 默认为 `common/constants.go:75` 在 `init()` 中生成的随机 UUID。未设置场景无 fatal 警告 —— 只有字面字符串 `"random_string"` 触发 `common/init.go:51-54` 的 `log.Fatal`。后果：（1）session 重启后不持久；（2）多实例部署中各实例生成不同 secret，用户 session cookie 在负载均衡后的其他实例上无效；（3）`CryptoSecret` 在 `common/init.go:62` 回退到 `SessionSecret`，因此用它加密的任何 access-token/凭据在重启后无法解密。
- 文件：[common/constants.go:75-76](file:///e:/AI%20Project/new-api/common/constants.go#L75-L76), [common/init.go:49-63](file:///e:/AI%20Project/new-api/common/init.go#L49-L63)
- 当前缓解：拒绝 `random_string` 字面量；文档大概会告知运维设置 `SESSION_SECRET`。
- 建议：当 `GIN_MODE != debug` 且 `SESSION_SECRET` 未设置时，记录明显警告。检测到多实例时（SystemInstanceReporter 注册多于一个节点），若未显式设置 `SESSION_SECRET` 则拒绝启动。清晰文档化 `CRYPTO_SECRET` 依赖 —— 在 token 已签发后轮换 `SESSION_SECRET` 会使加密凭据失效。

**密码最大长度 20 字符：**
- 风险：`User.Password` 字段被校验为 `min=8,max=20` —— 这主动阻止用户使用强口令短语（NIST 800-63B 建议允许 64+ 字符）。该上限还迫使用户使用更弱密码。
- 文件：[model/user.go:27](file:///e:/AI%20Project/new-api/model/user.go#L27)
- 当前缓解：在 `common/crypto.go:25` 使用 bcrypt 哈希（良好 —— bcrypt 自身有 72 字节限制）。
- 建议：将 `max` 提升至至少 64（bcrypt 最多接受 72 字节）。保持 `min=8` 或提升到 10–12。

**全局不安全 TLS 配置：**
- 风险：`common.InsecureTLSConfig = &tls.Config{InsecureSkipVerify: true}` 是包级单例。当 `TLS_INSECURE_SKIP_VERIFY=true` 时应用（在 `common/init.go:86` 环境门控）。若运维设置此标志，所有上游 model 调用的出站 TLS 验证被禁用 —— 使 API key 传输面临 MITM 风险。
- 文件：[common/constants.go:117-118](file:///e:/AI%20Project/new-api/common/constants.go#L117-L118)
- 当前缓解：默认 `false`；SMTP 变体由管理员控制，在 `common/email.go:40` 有 `#nosec G402` 注解。
- 建议：当 `TLSInsecureSkipVerify` 为 true 时发出启动警告。避免共享单例 —— 为每个 client 构造独立配置，使标志不会泄漏到意外的 client。

**pprof 绑定所有网络接口：**
- 风险：当 `ENABLE_PPROF=true` 时，debug 服务器监听 `0.0.0.0:8005` —— 主机所在的任何网络均可访问。pprof 暴露 goroutine dump、heap profile 以及 `common.Monitor()` 运行时信息。
- 文件：[main.go:153-159](file:///e:/AI%20Project/new-api/main.go#L153-L159)
- 当前缓解：默认关闭。
- 建议：默认绑定 `127.0.0.1:8005`；为显式需要远程访问的运维添加 `PPROF_HOST` 覆盖。

**敏感字段清理依赖开发者自觉：**
- 风险：`model/user.go:22-23` 的注释警告，任何新增敏感字段必须在 `setupLogin` 中手动清理，否则会以明文持久化到浏览器 localStorage。
- 文件：[model/user.go:22-23](file:///e:/AI%20Project/new-api/model/user.go#L22-L23)
- 当前缓解：人工审查。
- 建议：替换为登录响应 DTO（`UserBase` / 专用 `UserSessionView`）中的显式 allowlist，使新字段默认被排除而非默认被包含。

## 性能瓶颈

**同步 channel 余额刷新循环：**
- 问题：`updateAllChannelsBalance` 顺序遍历每个 channel，每个之间 `time.Sleep(common.RequestInterval)`，阻塞调用者。
- 文件：[controller/channel-billing.go:455-482](file:///e:/AI%20Project/new-api/controller/channel-billing.go#L455-L482)
- 原因：无并发；循环是 O(n)，每个 channel 一次 sleep。
- 改进路径：迁移到现有系统任务 runner（`service.StartSystemTaskRunner`、`main.go:144-145` 的 `controller.RegisterScheduledSystemTasks`），后者已支持 DB 租约去重和历史记录。将余额检查作为周期性定时任务运行，而非按需 HTTP。

**重试路径使 body 内存翻倍：**
- 问题：relay override 路径为每次重试克隆请求 body，在高并发下每次重试约消耗 2× body 大小的临时内存。
- 文件：[relay/common/override.go:722](file:///e:/AI%20Project/new-api/relay/common/override.go#L722)
- 原因：为支持重试时的请求改写而完整缓冲 body。
- 改进路径：未配置 override 时流式传输原始 body；仅在 `info.ChannelSetting` 实际请求 body 修改时缓冲。考虑为缓冲使用 `sync.Pool`。

**Gemini relay 热路径字符串抖动：**
- 问题：高并发下 Gemini relay 路径执行重复的 `+` 字符串拼接后接 `strings.Join`，放大堆驻留。
- 文件：[relay/channel/gemini/relay-gemini.go:1086](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go#L1086), [relay/channel/gemini/relay-gemini.go:1212](file:///e:/AI%20Project/new-api/relay/channel/gemini/relay-gemini.go#L1212)
- 原因：在最终 join 之前增量组装文本片段。
- 改进路径：为累积缓冲使用 `strings.Builder`（已是 Go 惯用法）；基于上游 chunk 的长度提示预分配。

## 脆弱区域

**`relay/common/override.go`（1937 行）：**
- 文件：[relay/common/override.go](file:///e:/AI%20Project/new-api/relay/common/override.go)
- 为何脆弱：单文件包含所有 channel 的请求/响应 override 转换。第 918、935、949 行有 `err = nil` 吞错误语句，在 override 应用期间静默上游错误。每个 channel 类型有深层嵌套条件。为一个 channel 的变更可能引发另一个 channel 的回归。
- 安全修改：始终运行 `relay/common/override_test.go`（2096 行 —— 最大测试文件，表明团队已将其视为关键）。编辑前为触及的具体 override 添加测试用例。不要一次重构整个文件。
- 测试覆盖：由 `override_test.go` 充分覆盖；风险在于吞错误（918/935/949）可能隐藏测试未覆盖的错误。

**`middleware/distributor.go` channel 选择：**
- 文件：[middleware/distributor.go](file:///e:/AI%20Project/new-api/middleware/distributor.go)
- 为何脆弱：中央请求路由逻辑。`switch channel.Type` 块（485-503 行）加多 key 处理（472-478 行）以及 model-request 解析路径都汇经此处。此处任何 bug 影响每个 relay 请求。
- 安全修改：`common/gin.go:144` 的 model-request 提取有"someday non-JSON requests have variant model" TODO —— 注意非 JSON body 会静默跳过 model 提取。修改 switch 前在 channel-type 分发周围添加集成测试。
- 测试覆盖：无 `distributor_test.go`。高优先级缺口。

**`model/subscription.go`（1282 行）+ `service/billing_session.go`：**
- 文件：[model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go), [service/billing_session.go](file:///e:/AI%20Project/new-api/service/billing_session.go)
- 为何脆弱：订阅配额生命周期（pre-consume → reserve → settle → reset）跨 model + service 层，错误分类采用字符串匹配（见上文技术债务）。资金路径。
- 安全修改：编辑前追踪完整请求生命周期穿过 `BillingSession.PreConsume` / `reserveFunding` / settle。绝不修改 `model/subscription.go` 中的错误消息而不更新 `billing_session.go:218` 的字符串匹配。
- 测试覆盖：`service/tiered_settle_test.go`（688 行）和 `service/task_billing_test.go` 覆盖部分；无 `model/subscription_test.go`。

## 扩展限制

**单进程定时任务：**
- 当前容量：一个 master 节点运行 `service.StartSystemTaskRunner()`（`main.go:145`）和 `controller.RegisterScheduledSystemTasks()`（`main.go:144`）。
- 限制：若 master 节点宕机，在选出新 master 之前没有定时任务（channel 测试、上游 model 更新、异步任务轮询、订阅配额重置）运行。DB 租约去重防止重复执行，但其本身不提供故障转移 —— 必须由另一个实例接管 master 角色。
- 扩展路径：确保多实例部署运行 ≥2 个能成为 master 的节点。监控 `model/system_instance.go` reporter 输出。若正常运行时间要求超过单 master 可靠性，考虑外部化调度器（例如 DB 支持的工作队列）。

**内存内 channel/token 缓存一致性：**
- 当前容量：`model.InitChannelCache()` + `model.SyncChannelCache(common.SyncFrequency)` 在 `MemoryCacheEnabled` 时运行（`main.go:79-99`）。同步频率是单一全局间隔。
- 限制：缓存是进程内的。多实例部署中，实例 A 上的管理员变更在下一个 `SyncFrequency` tick（`common.SyncFrequency` 默认值）之前对实例 B 不可见。`authz.StartPolicySync` 循环（`main.go:105`）对策略遵循相同模型。
- 扩展路径：为更紧的一致性，使用 Redis pub/sub 在写入时失效（Redis 已是依赖）。向运维文档化最终一致性窗口。

## 风险依赖

**`github.com/go-redis/redis/v8`（v8.11.5）—— EOL 主版本：**
- 风险：`go-redis/redis/v8` 模块路径是遗留导入；项目已迁移到 `redis/go-redis/v9`。v8 不再获得新功能，仅有有限安全修复。
- 影响：`common/redis.go` 和所有缓存层（`model/channel_cache.go`、`model/token_cache.go`、`model/user_cache.go`）固定在 v8 API。
- 迁移计划：迁移到 `github.com/redis/go-redis/v9`。API 大体兼容；主要破坏性变更是 pipeline/context 语义和 `Cmdable` 接口。审计 `common.InitRedisClient` 和缓存调用方。

**`github.com/gin-gonic/gin` v1.9.1 —— 落后当前版本：**
- 风险：v1.9.1（2023 年 5 月）早于若干 CVE 修复和 v1.10 线。`gin-contrib/cors`、`gin-contrib/sessions`、`gin-contrib/gzip`、`gin-contrib/static` 均固定到匹配此线的版本。
- 影响：gin 本身的安全公告；目前无功能性阻塞。
- 迁移计划：升级到 gin v1.10.x；验证 gin-contrib 子包兼容。升级后运行 `controller/*_test.go` 和 relay 集成测试。

**`github.com/shirou/gopsutil` v3.21.11+incompatible —— 陈旧/不正确的模块路径：**
- 风险：`+incompatible` 后缀意味着 v3 tag 在没有 go.mod 的情况下发布。当前推荐路径是 `github.com/shirou/gopsutil/v3` 或 `v4`。v3.21.11 来自 2021 年。
- 影响：由 `common.Monitor()` 和系统信息页使用。较新 OS 上进程/磁盘指标陈旧。
- 迁移计划：切换到 `github.com/shirou/gopsutil/v4`（或使用正确模块路径的 `v3/cpu`、`v3/mem` 子包）。

**`gorm.io/gorm` v1.25.2 及驱动 —— 落后当前版本：**
- 风险：v1.25.2 不是最新 v1.25.x 补丁。`gorm.io/driver/postgres` v1.5.2 和 `gorm.io/driver/mysql` v1.4.3 更旧。
- 影响：可能缺失迁移和 SQLite 列添加路径（`model/main.go:564`）的 bug 修复。
- 迁移计划：在 v1.25.x 线内升级（低风险）。针对新驱动版本验证 `model/main.go` 中的自定义迁移 helper（`ensureSubscriptionPlanTableSQLite`、`checkAndMigrateDB`）。

## 缺失的关键功能

**除 channel-disable 外，上游 5xx 无自动重试/断路器：**
- 问题：当上游 provider 返回 5xx，relay 通过 channel affinity 重试（`service/channel_affinity.go`，923 行），但没有 per-provider 断路器。慢上游会占用请求 goroutine。
- 阻塞：部分上游中断下的可靠行为。

**无管理员操作的结构化审计日志：**
- 问题：`middleware/audit.go` 存在，但审计覆盖不全面 —— `model/user.go:22-23` 的 `User` 结构注释暗示敏感字段处理是手动的。代码库图中没有可见的配额授予、角色变更或 channel-key 轮换的只追加审计轨迹。
- 阻塞：计费争议的合规/取证。

## 测试覆盖缺口

**`controller/user.go`（1332 行）—— 无专门测试：**
- 未测试内容：User CRUD、密码修改、角色分配、access-token 生成（`controller/user.go:357` 使用 `common.GenerateRandomKey`）、OAuth 绑定。
- 文件：[controller/user.go](file:///e:/AI%20Project/new-api/controller/user.go)
- 风险：认证/管理路径 bug 未被察觉即发布。密码修改边界情况（错误旧密码、角色变更时的配额）未测试。
- 优先级：High。

**`middleware/auth.go` + `middleware/distributor.go` —— 无测试：**
- 未测试内容：Access-token 校验流程、角色门控（`authHelper` min-role）、channel 选择分发、多 key 索引处理。
- 文件：[middleware/auth.go](file:///e:/AI%20Project/new-api/middleware/auth.go), [middleware/distributor.go](file:///e:/AI%20Project/new-api/middleware/distributor.go)
- 风险：每个请求都流经这些。此处回归是全面中断。
- 优先级：High。

**`model/user.go`（981 行）、`model/subscription.go`（1282 行）、`model/channel.go`（1014 行）—— 无 model 层测试：**
- 未测试内容：配额原子性（`IncreaseUserQuota`、`DecreaseUserQuota`）、订阅生命周期状态转换、channel 缓存失效。
- 文件：[model/user.go](file:///e:/AI%20Project/new-api/model/user.go), [model/subscription.go](file:///e:/AI%20Project/new-api/model/subscription.go), [model/channel.go](file:///e:/AI%20Project/new-api/model/channel.go)
- 风险：资金路径数据完整性 bug。注意 `model/redemption.go` 和 `model/topup.go` 已使用 `recover()` 守卫（第 36、69、177、213、246、286 行），暗示这些路径过去发生过 panic。
- 优先级：High（资金路径），Medium（user/channel）。

**`service/convert.go`（938 行）—— 无测试：**
- 未测试内容：OpenAI/Claude/Gemini 等之间的请求/响应格式转换。
- 文件：[service/convert.go](file:///e:/AI%20Project/new-api/service/convert.go)
- 风险：静默的格式转换回归导致上游请求畸形。
- 优先级：Medium。（注：`service/relayconvert/` 子包已测试 —— 未测试的是较旧的 `service/convert.go`。）

**`oauth/*.go` —— 无测试：**
- 未测试内容：GitHub/Discord/OIDC/LinuxDO/generic OAuth 流程（`oauth/github.go`、`oauth/discord.go`、`oauth/oidc.go`、`oauth/linuxdo.go`、`oauth/generic.go`）。
- 风险：OAuth provider API 变更静默破坏登录。`NewOAuthErrorWithRaw` 包装器被广泛使用，但无测试断言错误分类正确性。
- 优先级：Medium。

**`controller/relay.go`（593 行）—— 无测试：**
- 未测试内容：连接 distributor → adaptor → 响应的主 relay 入口 handler。
- 文件：[controller/relay.go](file:///e:/AI%20Project/new-api/controller/relay.go)
- 风险：Relay 路由回归。部分由 relay adaptor 测试缓解，但顶层 handler 未测试。
- 优先级：Medium。

---

*Concerns audit: 2026-07-01*
