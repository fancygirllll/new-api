# 代码库结构

**分析日期：** 2026-07-01

## 目录布局

```
new-api/
├── main.go                 # 进程入口；内嵌前端，构建 Gin 服务器
├── go.mod / go.sum          # Go 1.25.1 module github.com/QuantumNous/new-api
├── makefile                 # 构建/测试目标
├── Dockerfile / Dockerfile.dev / docker-compose*.yml  # 容器构建
├── VERSION                  # 在构建时注入 common.Version 的 semver 字符串
├── .env.example             # 已文档化的环境变量（不要读取 .env 内容）
├── AGENTS.md / CLAUDE.md    # 项目约定（导入到 agent 上下文中）
├── bin/                     # 迁移 SQL + shell 辅助脚本
├── docs/                    # 额外文档
├── data/                    # SQLite DB + 磁盘缓存文件（运行时数据）
├── logs/                    # 应用日志文件（运行时输出）
├── router/                  # HTTP 路由注册（所有 Gin 分组）
├── middleware/              # 请求流水线（认证、分发、限流、审计）
├── controller/              # 请求处理器（API + 中继编排）
├── service/                 # 业务逻辑 + authz/、passkey/、relayconvert/ 子包
├── model/                   # GORM 实体、DB 初始化、内存缓存
├── relay/                   # AI 代理核心：handler + channel/<provider>/ 适配器
│   └── channel/             # 40+ 供应商适配器子包
├── dto/                     # 请求/响应传输结构体
├── types/                   # 核心类型定义（RelayFormat、NewAPIError、PriceData）
├── constant/                # 常量（渠道类型、api 类型、上下文键）
├── common/                  # 共享工具（json、redis、crypto、env、limiter/）
├── setting/                # 配置域（ratio_、model_、operation_、...）
├── oauth/                   # OAuth provider 实现 + 注册表
├── i18n/                    # 后端 i18n（go-i18n）+ locales/
├── logger/                  # 自定义分级 logger
├── pkg/                     # 内部库（billingexpr、cachex、ionet、perf_metrics）
├── web/                     # 前端容器
│   ├── default/             # 默认主题（React 19、Rsbuild、Base UI、Tailwind）
│   └── classic/             # 经典主题（React 18、Vite、Semi Design）
└── electron/                # Electron 桌面壳
```

## 目录用途

**`router/`：**
- 用途：HTTP 路由 —— 注册所有 Gin 路由分组和每分组的中间件。
- 包含：`main.go`（SetRouter 装配）、`api-router.go`、`relay-router.go`、`dashboard.go`、`web-router.go`、`video-router.go`、`channel-router.go`、`authz-router.go`、`channel_router_test.go`
- 关键文件：`router/main.go`、`router/relay-router.go`、`router/api-router.go`

**`middleware/`：**
- 用途：横切请求流水线 —— 认证、分发、限流、CORS、gzip、审计、i18n。
- 包含：`auth.go`、`distributor.go`、`rate-limit.go`、`model-rate-limit.go`、`cors.go`、`gzip.go`、`audit.go`、`i18n.go`、`request-id.go`、`request_body_limit.go`、`secure_verification.go`、`turnstile-check.go`、`cache.go`、`disable-cache.go`、`stats.go`、`performance.go`、`header_nav.go`、`kling_adapter.go`、`jimeng_adapter.go`、`body_cleanup.go`、`recover.go`、`utils.go`
- 关键文件：`middleware/auth.go`、`middleware/distributor.go`、`middleware/rate-limit.go`

**`controller/`：**
- 用途：请求处理器 —— 解析输入、调用 service/model、格式化 HTTP 响应。
- 包含：80+ 个 handler 文件，覆盖 relay、user、channel、token、billing、topup、subscription、log、pricing、model、playground、passkey、2FA、OAuth、system task、部署（io.net）、视频代理等。
- 关键文件：`controller/relay.go`（Relay/RelayTask/RelayMidjourney）、`controller/user.go`、`controller/channel.go`、`controller/billing.go`、`controller/token.go`、`controller/system_task*.go`

**`service/`：**
- 用途：可跨 handler 复用的业务逻辑 —— 计费、渠道选择、敏感词检查、token 计数、支付、通知、任务轮询。
- 包含：`billing.go`、`billing_session.go`、`pre_consume_quota.go`、`channel_select.go`、`channel_affinity.go`、`channel.go`、`quota.go`、`text_quota.go`、`task_billing.go`、`tiered_settle.go`、`sensitive.go`、`token_counter.go`、`tokenizer.go`、`epay.go`、`webhook.go`、`system_task.go`、`task_polling.go`、`error.go`、`http_client.go`、codex 凭证刷新、订阅重置等。
- 子包：`service/authz/`（casbin RBAC：enforcer、permission、role、resources、registry）、`service/passkey/`（WebAuthn）、`service/relayconvert/`（请求转换辅助）
- 关键文件：`service/billing.go`、`service/channel_select.go`、`service/system_task.go`、`service/authz/enforcer.go`

**`model/`：**
- 用途：GORM 实体、DB 连接管理、多方言辅助函数、内存缓存。
- 包含：`main.go`（DB 初始化 + `commonGroupCol`/`commonKeyCol` 方言辅助函数）、`user.go`、`token.go`、`channel.go`、`ability.go`、`channel_cache.go`、`user_cache.go`、`token_cache.go`、`log.go`、`task.go`、`system_task.go`、`pricing.go`、`subscription.go`、`option.go`、`casbin_rule.go`、`authz_role.go`、`midjourney.go`、`vendor_meta.go`、`model_meta.go`、`perf_metric.go`、`setup.go`、`errors.go`
- 关键文件：`model/main.go`、`model/channel_cache.go`、`model/ability.go`、`model/user.go`

**`relay/`：**
- 用途：AI 代理核心 —— 格式转换、上游传输、流式、计费集成。
- 包含：`relay_adaptor.go`（工厂）、`compatible_handler.go`（文本）、`claude_handler.go`、`gemini_handler.go`、`image_handler.go`、`audio_handler.go`、`embedding_handler.go`、`rerank_handler.go`、`responses_handler.go`、`mjproxy_handler.go`、`relay_task.go`、`websocket.go`、`chat_completions_via_responses.go`、`param_override_error.go`
- 子包：`relay/channel/`（供应商适配器）、`relay/common/`（RelayInfo、billing、override）、`relay/helper/`（请求校验、计价）、`relay/constant/`（RelayMode）、`relay/common_handler/`、`relay/reasonmap/`
- 关键文件：`relay/relay_adaptor.go`、`relay/channel/adapter.go`、`relay/common/relay_info.go`、`relay/helper/valid_request.go`、`relay/helper/price.go`

**`relay/channel/<provider>/`：**
- 用途：每个上游 AI 供应商一个子包，实现 `channel.Adaptor`（可选再实现 `channel.TaskAdaptor`）。
- 包含（40+ 供应商）：`openai/`、`claude/`、`gemini/`、`aws/`、`vertex/`、`azure`（通过 openai）、`ali/`、`baidu/`、`baidu_v2/`、`tencent/`、`xunfei/`、`zhipu/`、`zhipu_4v/`、`ollama/`、`perplexity/`、`cohere/`、`dify/`、`jina/`、`cloudflare/`、`siliconflow/`、`mistral/`、`deepseek/`、`mokaai/`、`volcengine/`、`xai/`、`coze/`、`jimeng/`、`moonshot/`、`minimax/`、`replicate/`、`codex/`、`advancedcustom/`、`submodel/`、`palm/`、`ai360/`、`lingyiwanwu/`（由 openai 导入）、`openrouter/`、`xinference/`（由 openai 导入）、`task/`（异步任务适配器：`ali/`、`doubao/`、`gemini/`、`hailuo/`、`jimeng/`、`kling/`、`sora/`、`suno/`、`vertex/`、`vidu/`）
- 每个供应商子包通常包含：`adaptor.go`、`constants.go`、`dto.go`，以及供应商专属的中继文件（如 `relay-zhipu.go`、`image.go`）。
- 关键文件：`relay/channel/adapter.go`（接口契约）、`relay/channel/openai/adaptor.go`

**`dto/`：**
- 用途：数据传输对象 —— 从客户端 JSON 解析得到、再重新序列化发给上游的请求/响应结构体。
- 包含：`openai_request.go`、`claude.go`、`gemini.go`、`openai_response.go`、`openai_image.go`、`openai_responses_compaction_request.go`、`audio.go`、`embedding.go`、`rerank.go`、`task.go`、`midjourney.go`、`suno.go`、`video.go`、`playground.go`、`pricing.go`、`ratio_sync.go`、`channel_settings.go`、`request_common.go`、`realtime.go`、`error.go`、`notify.go`、`user_settings.go`、`values.go`、`sensitive.go`、`openai_video.go`、`openai_compaction.go`
- 关键文件：`dto/openai_request.go`、`dto/claude.go`、`dto/gemini.go`

**`types/`：**
- 用途：不依赖任何单一中继格式的核心类型定义。
- 包含：`relay_format.go`（RelayFormat 枚举）、`error.go`（NewAPIError）、`channel_error.go`、`price_data.go`、`request_meta.go`、`file_source.go`、`file_data.go`、`set.go`、`rw_map.go`
- 关键文件：`types/relay_format.go`、`types/error.go`

**`constant/`：**
- 用途：项目级常量 —— 渠道类型、api 类型、上下文键、缓存键、环境标志。
- 包含：`channel.go`（ChannelType* 枚举）、`api_type.go`（APIType* 枚举）、`context_key.go`、`cache_key.go`、`env.go`、`task.go`、`midjourney.go`、`finish_reason.go`、`multi_key_mode.go`、`endpoint_type.go`、`endpoint_defaults.go`、`azure.go`、`setup.go`、`waffo_pay_method.go`
- 关键文件：`constant/channel.go`、`constant/api_type.go`、`constant/context_key.go`

**`common/`：**
- 用途：被各层共享的工具 —— JSON 编解码器、Redis、crypto、env、IP/CIDR、email、限流 limiter、磁盘缓存、SSRF 防护、gopool、系统监控。
- 包含：`json.go`、`redis.go`、`crypto.go`、`hash.go`、`env.go`、`init.go`、`ip.go`、`email.go`、`rate-limit.go`、`quota.go`、`database.go`、`constants.go`、`gin.go`、`utils.go`、`validate.go`、`verification.go`、`totp.go`、`ssrf_protection.go`、`request_body_limit.go`、`body_storage.go`、`disk_cache.go`、`gopool.go`、`go-channel.go`、`embed-file-system.go`、`model.go`、`page_info.go`、`node_identity.go`、`performance_config.go`、`pyro.go`、`pprof.go`、`sys_log.go`、`system_monitor*.go`、`str.go`、`copy.go`、`custom-event.go`、`audio.go`、`endpoint_type.go`、`endpoint_defaults.go`、`topup-ratio.go`、`url_validator.go`、`disk_cache_config.go`、`json_test.go`、`email_test.go`、`url_validator_test.go`
- 子包：`common/limiter/`（`limiter.go`、`lua/rate_limit.lua`）
- 关键文件：`common/json.go`、`common/constants.go`、`common/redis.go`、`common/init.go`

**`setting/`：**
- 用途：从 `model.OptionMap` 加载的类型化配置域，每个域持有一片运行时设置。
- 包含（顶层）：`auto_group.go`、`chat.go`、`midjourney.go`、`payment_creem.go`、`payment_stripe.go`、`payment_waffo.go`、`payment_waffo_pancake.go`、`rate_limit.go`、`sensitive.go`、`user_usable_group.go`
- 子包：`ratio_setting/`（group_ratio、model_ratio、cache_ratio、compact_suffix、exposed_cache）、`model_setting/`、`operation_setting/`、`system_setting/`、`performance_setting/`（在 main.go 中通过 `_` 自导入）、`perf_metrics_setting/`、`billing_setting/`、`console_setting/`、`reasoning/`、`config/`
- 关键文件：`setting/ratio_setting/model_ratio.go`、`setting/ratio_setting/group_ratio.go`、`setting/operation_setting/*.go`

**`oauth/`：**
- 用途：OAuth provider 实现 + 一个通过 `init()` 自动注册 provider 的注册表。
- 包含：`provider.go`（接口）、`registry.go`、`types.go`、`github.go`、`discord.go`、`oidc.go`、`linuxdo.go`、`generic.go`
- 关键文件：`oauth/provider.go`、`oauth/registry.go`

**`i18n/`：**
- 用途：基于 `nicksnyder/go-i18n/v2` 的后端国际化。
- 包含：`Init`、消息目录、`locales/`（en、zh JSON）
- 关键文件：`i18n/` 根目录文件、`i18n/locales/`

**`logger/`：**
- 用途：写入 `logs/` 和 stdout 的自定义分级 logger。
- 关键文件：`logger/` 根目录文件

**`pkg/`：**
- 用途：自带稳定 API 的自包含内部库，可跨层使用。
- 包含：`billingexpr/`（分级计费表达式引擎 —— 改动前先读 `pkg/billingexpr/expr.md`）、`cachex/`（带 codec + 命名空间的内存/Redis 混合缓存）、`ionet/`（io.net 部署客户端）、`perf_metrics/`（中继采样记录 + 系统指标）
- 关键文件：`pkg/billingexpr/expr.md`、`pkg/cachex/hybrid_cache.go`、`pkg/perf_metrics/`

**`web/default/`：**
- 用途：默认前端主题 —— React 19 + TypeScript + Rsbuild + Base UI + Tailwind CSS。
- 包含：`package.json`、`rsbuild.config.*`、`tsconfig*.json`、`src/`（应用代码）、`dist/`（构建产物，由 Go 内嵌）
- `src/` 下的关键目录：`routes/`（TanStack Router 基于文件的路由）、`features/`（功能模块）、`components/`（`ui/`、`layout/`、`data-table/`、`ai-elements/`）、`stores/`（Zustand）、`hooks/`、`lib/`、`config/`、`context/`、`i18n/`（i18next + `locales/`）、`styles/`、`assets/`

**`web/classic/`：**
- 用途：旧版前端主题 —— React 18 + Vite + Semi Design。构建产物由 Go 内嵌。
- 注意：旧版的同步日志删除路由 `/api/log` DELETE 只为这个主题存在（见 `router/api-router.go:269` 的 TODO）。

**`electron/`：**
- 用途：包在 web 前端外的 Electron 桌面壳。

**`bin/`：**
- 用途：SQL 迁移脚本（`migration_v0.2-v0.3.sql`、`migration_v0.3-v0.4.sql`）和 shell 辅助脚本（`time_test.sh`）。

## 关键文件位置

**入口点：**
- `main.go`：进程引导、Gin 服务器装配、前端内嵌、后台 goroutine
- `router/main.go`：通过 `SetRouter` 装配所有路由分组
- `controller/relay.go:68`（`Relay`）：中继请求编排器
- `controller/relay.go:486`（`RelayTask`）：异步任务编排器
- `web/default/src/routes/__root.tsx`：前端根路由

**配置：**
- `.env.example`：已文档化的环境变量（仅存在性 —— 不要读取 `.env`）
- `common/env.go` / `common/init.go`：env 加载 + 全局标志初始化
- `model/main.go:30-51`：DB 方言感知的列/值辅助函数
- `setting/ratio_setting/`：计价倍率配置
- `VERSION`：构建时注入 `common.Version` 的 semver 字符串

**核心逻辑：**
- `relay/relay_adaptor.go:54`：适配器工厂（`GetAdaptor`）
- `relay/relay_adaptor.go:138`：任务适配器工厂（`GetTaskAdaptor`）
- `relay/channel/adapter.go`：Adaptor + TaskAdaptor 接口契约
- `relay/common/relay_info.go`：`RelayInfo` 请求上下文载体
- `relay/helper/valid_request.go:21`：请求解析 + 校验分发
- `relay/helper/price.go`：模型计价计算
- `service/billing.go` / `service/billing_session.go`：计费会话（预扣/结算/退还）
- `service/channel_select.go` / `service/channel_affinity.go`：渠道选择逻辑
- `middleware/distributor.go:32`：渠道分发中间件
- `middleware/auth.go:303`（`TokenAuth`）、`:181`（`UserAuth`）、`:187`（`AdminAuth`）、`:193`（`RootAuth`）、`:199`（`RequirePermission`）
- `service/authz/enforcer.go`：casbin enforcer 初始化 + 模型
- `model/channel_cache.go:26`（`InitChannelCache`）：内存渠道索引
- `model/ability.go`：Group→Model→Channel 能力实体

**测试：**
- Go 测试：与被测代码同目录放置的 `*_test.go`（如 `relay/common/relay_info_test.go`、`service/task_billing_test.go`、`model/system_task_test.go`、`router/channel_router_test.go`）
- 迁移 SQL：`bin/migration_*.sql`
- 前端测试：在 `web/default/` 下（见 `web/default/AGENTS.md`）

**文档：**
- `AGENTS.md`：项目约定（在本仓库工作前先读这个）
- `CLAUDE.md`：导入 `AGENTS.md`
- `pkg/billingexpr/expr.md`：计费表达式引擎设计（改动计费表达式前先读）
- `README*.md`：多语言 README（en、zh_CN、zh_TW、ja、fr）

## 命名约定

**文件：**
- Go：大多数文件用 `snake_case.go`；当多词 handler 文件按功能聚合时用连字符（`channel-router.go`、`channel-billing.go`、`channel-test.go`、`topup-stripe.go`、`subscription_payment_stripe.go`）。混用是历史遗留 —— 新文件优先用 `snake_case`。
- `relay/channel/<provider>/` 内的供应商中继文件用 `relay-<provider>.go`（如 `relay-zhipu.go`、`relay-xunfei.go`）。
- 适配器文件始终是 `adaptor.go`；供应商常量在 `constants.go`；供应商 DTO 在 `dto.go`。
- 测试：`*_test.go` 与被测单元同目录放置。
- 前端（默认主题）：文件用 `kebab-case.tsx` / `kebab-case.ts`；路由文件遵循 TanStack Router 约定（动态段用 `$param.tsx`，布局/无路径路由用 `_authenticated/` 和 `_route` 前缀）。

**目录：**
- Go 包：小写，优先单词（`controller`、`model`、`service`）；供应商包与供应商 slug 匹配（`relay/channel/openai/`、`relay/channel/aws/`）。
- 配置子包：`<domain>_setting/`（如 `ratio_setting/`、`model_setting/`）。
- 前端功能目录：`web/default/src/features/<feature>/` 与路由段匹配（如 `channels/`、`keys/`、`usage-logs/`）。

**标识符：**
- 导出的 Go 类型/函数：`PascalCase`（`RelayInfo`、`GetAdaptor`、`TokenAuth`）。
- 渠道/API 类型常量：`ChannelType<Pascal>` / `APIType<Pascal>`（`constant/channel.go`、`constant/api_type.go`）。
- 上下文键：`ContextKey<Pascal>`（`constant/context_key.go`）。
- 中继模式：`RelayMode<Pascal>`（`relay/constant/relay_mode.go`）。

## 在哪里添加新代码

**新增上游 AI 供应商（channel）：**
1. 创建 `relay/channel/<provider>/`，包含 `adaptor.go`（实现 `relay/channel/adapter.go` 中的 `channel.Adaptor`）、`constants.go`、`dto.go`，以及任意 `relay-<provider>.go` 辅助文件。
2. 在 `constant/channel.go` 中加一个渠道类型常量（放在 `ChannelTypeDummy` 之前），并在 `ChannelBaseURLs` 中加 base URL。
3. 在 `constant/api_type.go` 中加一个 `APIType<Pascal>` 常量，以及 channel→api-type 映射。
4. 在 `relay/relay_adaptor.go:54` 的 `GetAdaptor` 中加一个 `case constant.APIType<Pascal>: return &<provider>.Adaptor{}`。
5. 如果该供应商支持 `StreamOptions`，在 `streamSupportedChannels` 中注册（见 `AGENTS.md` "Relay and provider behavior"）。
6. 测试：在供应商包中加 `*_test.go`。

**新增异步任务供应商（Midjourney/Suno/视频类）：**
1. 创建 `relay/channel/task/<provider>/`，实现 `channel.TaskAdaptor`（`relay/channel/adapter.go:34`）。
2. 在 `relay/relay_adaptor.go:138` 的 `GetTaskAdaptor` 中按 `constant.ChannelType*` 加一个 `case`。
3. 在 `router/relay-router.go`（或 `router/video-router.go`）中接上 submit/fetch 路由，调用 `controller.RelayTask` / `controller.RelayTaskFetch`。

**新增管理 API 端点：**
1. 在 `controller/` 中加 handler 函数（选一个拥有该资源的现有文件，如渠道就放 `controller/channel.go`；只在资源截然不同时才新建文件）。
2. 在 `router/api-router.go` 的相应分组中注册路由（权限门控的渠道路由放 `router/channel-router.go` —— 加到 `channelPermissionRoutes` 并附一个 `authz.Permission`）。
3. 套上合适的中间件：`UserAuth` / `AdminAuth` / `RootAuth` / `RequirePermission` / `CriticalRateLimit` / `AnonymousRequestBodyLimit`，按需选择。
4. 在 `model/` 中加 DB 方法（扩展相关实体文件）。

**新增业务逻辑 / service 函数：**
- 放在 `service/<area>.go`（如计费逻辑放 `service/billing*.go`，渠道逻辑放 `service/channel*.go`）。保持可被 `controller` 和 `middleware` 导入。不要从 `service` 导入 `relay` —— 任务适配器用注入的 `service.GetTaskAdaptorFunc`。

**新增 GORM 实体：**
1. 在 `model/<entity>.go` 中定义结构体，使用跨库安全的 tag（不要 `AUTO_INCREMENT`/`SERIAL`，不要 `default:true` 业务规则布尔值，`group`/`key` 等保留字列用 `commonGroupCol`/`commonKeyCol`）。
2. 在 `model/main.go` 迁移列表中注册以自动迁移。
3. 如果热路径读取需要缓存，在 `model/<entity>_cache.go` 中加缓存方法。

**新增前端路由（默认主题）：**
1. 加 `web/default/src/routes/<segment>/index.tsx`（需登录则放 `web/default/src/routes/_authenticated/<segment>/index.tsx`）。动态段用 `$param.tsx`，分页子路由用 `$section.tsx`。
2. 在 `web/default/src/features/<segment>/` 下加功能模块，承载页面的组件/hook。
3. 在 `web/default/src/i18n/locales/{lang}.json` 中以英文源串形式加 i18n key（在 `web/default/` 下用 `bun run i18n:sync`）。
4. 重新跑 `bun run build`（或 `bun run dev`）—— `web/default/src/routeTree.gen.ts` 会自动生成。

**新增共享工具：**
- 后端：加到 `common/<area>.go`（如 `common/utils.go`、`common/str.go`）。任何 JSON 都用 `common.Marshal`/`Unmarshal`。
- 前端：加到 `web/default/src/lib/`。

**新增中间件：**
- 加 `middleware/<name>.go`，导出一个构造函数返回 `gin.HandlerFunc`（匹配 `UserAuth`/`AdminAuth` 用的 `func() func(c *gin.Context)` 形态，闭包捕获配置）。
- 在相关路由文件中接入（`router/api-router.go`、`router/relay-router.go` 等）。

**新增配置 / 选项：**
- 在 `setting/<domain>_setting/` 下选一个归属子包；加类型化的 getter/setter 并注册 option key 以便从 `model.OptionMap` 加载。`setting/performance_setting/` 包在 `main.go:27` 通过空导入注册其 init —— 如果某个配置包需要副作用注册，照此模式做。

## 特殊目录

**`data/`：**
- 用途：SQLite 数据库文件、磁盘缓存文件及其他运行时数据。
- 生成：是（运行时）
- 提交：否（gitignored）

**`logs/`：**
- 用途：由 `logger/` 写入的应用日志文件。
- 生成：是（运行时）
- 提交：否（gitignored）

**`web/default/dist/` 和 `web/classic/dist/`：**
- 用途：通过 `//go:embed` 内嵌进 Go 二进制的前端构建产物（`main.go:39-49`）。`go build` 前必须存在。
- 生成：是（由 `bun run build` / Vite 生成）
- 提交：是（内嵌的 `dist/index.html` 在编译时读取；提交依赖前端更新的 Go 改动前需重新构建）

**`web/default/src/routeTree.gen.ts`：**
- 用途：自动生成的 TanStack Router 路由树。不要手动编辑。
- 生成：是（由路由插件在 `bun run dev`/`build` 时生成）
- 提交：是

**`.planning/`：**
- 用途：GSD 工作流产物 —— 代码库地图、路线图、阶段计划。
- 生成：是（由 GSD 命令生成）
- 提交：可选（项目自定）

**`.agents/skills/` 和 `.trae/skills/`：**
- 用途：项目级 agent skill（如 `vercel-react-best-practices`）。以轻量 `SKILL.md` 索引加载，而非完整 `AGENTS.md`。
- 生成：否
- 提交：是

**`bin/`：**
- 用途：一次性 SQL 迁移脚本和 shell 辅助脚本，在版本升级时手动运行。
- 生成：否
- 提交：是

---

*结构分析：2026-07-01*
