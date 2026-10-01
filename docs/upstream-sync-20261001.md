# new-api fork 上游同步（2026-10-01）

本次在 `main` 普通合并 QuantumNous/new-api 的 `56758edf95ec162033a6b73b553ac789404f87c9`（2026-09-30），保留双方历史。合并前的 fork 提交为 `4e73fb3cbd6d4d83fae6b5dc4a369d08b47d8adf`，备份引用为 `backup/main-before-upstream-20261001-v2`。

## 适配结果

- 保留 RS Gateway 渠道编号 **61**；上游 Task Plugin 在本 fork 使用 **64**，vLLM/SGLang 为 62/63。从上游实例导入渠道时，不能直接把其 Task Plugin 的编号 61 写入本 fork。
- 协议转换、工具调用及厂商行为由独立的 `realseek-gateway` 管理。new-api 保留透传、身份头、分发、管理和计费功能，网关错误不触发 new-api 的重复重试与自动封禁。
- 以 `plugins/tasks/rs-gateway/plugin.js` 接入上游新任务系统，保留任意视频模型、自定义请求字段、模型映射、轮询和内容下载。插件不复制 Sora 或 Responses 的厂商转换逻辑。
- 启动迁移把历史网关任务的 `platform=61` 改为 `rs-gateway` 并补充插件快照，保留上游任务 ID、渠道密钥、原始任务数据和计费快照。迁移可重复执行。
- 保留 `RSGatewaySettlement` 的原子资金事务、退款和结算幂等；视频提交也进入该账本。上游已接受的任务在客户端断开后继续落库和结算。
- 保留视频按秒/分辨率、图片档位、分辨率模型别名、OnlyArt SSO、批量兑换、在线充值导航和日志信息。自定义 `VideoPrice` / `ImagePrice` 进入新版价格管理的事务和版本检查，避免编辑其他模型时清空原有配置。
- 吸收上游新 UI、访问令牌权限、安全功能、任务插件及计费表达式。预扣沿用上游按输入费用乘预扣倍率的新规则，实际结算仍按实际用量。
- 七种语言已同步，缺失翻译键均为零。
- 原定时任务改为只读检查上游更新并生成 Actions 摘要，取消自动合并及推送 main。未来上游渠道编号和任务机制变动需要先完成适配及回归。

## 验证

工具版本固定在项目 `mise.toml`：Go 1.26.1、Bun 1.4.0；前端依赖按锁文件安装。

- 根 Go 模块构建、relaykit 在 `GOWORK=off` 下独立构建，以及 relaykit DTO 测试通过。
- 网关 HTTP/SSE、取消、资金事务、退款、任务迁移、完整视频提交计费、模型价格保存/重置、时长校验、图片价格及别名的核心回归通过。
- model、service、relay、middleware、router、插件运行时、价格配置及 common 的相应包测试通过；controller 全包检查发现的插件错误文案预期及测试夹具问题已修复，并增量复验相关案例。
- 前端类型检查及生产构建通过。首轮 26 个相关测试文件、389 个案例通过；新增适配后的 6 个测试文件、345 个案例通过，最终日志徽标修改单独复验通过。
- 本次适配文件 lint 通过。全项目 lint 仍存在上游继承的问题，本次未批量修改无关文件。

SQLite 回归已运行；未连接生产数据库，也未运行真实 MySQL/PostgreSQL 环境验证。本次仅完成代码合并，不包含远程推送或部署。

## 合并后修复（2026-10-02）

- 网关视频拒绝顶层字段与 metadata 的时长冲突，同时检查 multipart 中 JSON 字符串形式的 metadata。只提供 metadata/durationSeconds 时长、未提供网关识别的顶层 seconds/duration 时拒绝提交，避免本地按指定时长扣费而网关使用默认时长。
- 创建视频任务时保存本次选中的网关密钥，轮询与内容下载沿用同一身份。已经创建但没有保存密钥的多密钥任务无法自动推断创建身份，需要通过网关记录核对。
- 失败视频的资金、令牌、任务额度、用量统计与退款账本在主库同一事务提交。任一步失败均回滚，任务保留额度，轮询继续重试；重复任务副本不会重复退款。新任务通过已有 Execution.RequestID 更新原结算账本；缺少请求来源的历史迁移任务以 task-refund:<数据库任务ID> 创建独立退款记录，不猜测原请求账目。独立日志库的退款日志在资金事务提交后写入，不与主库形成跨库事务。
- 渠道 61 可开启 Responses WebSocket；每轮请求独立建账、预扣、按网关实际用量结算或退款，包括零价请求。保留请求扩展字段、零值和 false，网关连接失败不触发重试或自动封禁。连接第二轮不再因被忽略的请求头覆盖配置报错，上一轮迟到用量不会进入下一轮账单。普通渠道预扣后重试选中网关时，要求客户端重连，以重新建立网关计费会话。
- Playground 视频账本不记录未预扣的 API 令牌，避免失败退款时凭空增加令牌余额。

本轮验证使用 Go 1.26.1、Bun 1.4.0、真实 SQLite **3.50.4**：

- 根模块 `go build ./...` 通过；本次没有修改 relaykit。
- `go test ./controller ./service ./relay ./model -run 'TestRSGateway|TestResponsesWebSocket|TestRefundTaskQuota|TestRecalculateTaskQuota|TestRunTaskPolling|TestTaskPolling|TestPrepareRequestBilling' -count=1 -timeout=150s` 通过。之后新增的免费请求用例以及时序/Playground 修复单独增量测试通过。
- 覆盖钱包/订阅退款的令牌写入故障回滚、任务写入故障回滚、自动重试、退款幂等、迁移任务退款、视频真实提交计费、HTTP/WS 对照、连续两轮请求、旧用量隔离、拒绝/失败/断开退款以及免费请求账本结束。
- 前端 Responses WebSocket 设置测试 **27 个用例通过**；`bun run typecheck`、相关前端文件及网关插件 lint 通过。

**尚未完成的验证：** 没有可用的真实 MySQL/PostgreSQL 测试实例，用户已确认先修复并记录缺口。当前不能宣称三个数据库的事务与锁行为均已验证；部署前仍需补齐。现有控制器数据库夹具可使用 `TEST_TASK_DB_DIALECT=mysql/postgres` 和 `TEST_MYSQL_DSN` / `TEST_POSTGRES_DSN` 连接隔离测试数据库，再运行 `go test ./controller -run TestRSGatewayVideoSubmissionPersistsAndChargesOnce -count=1 -v`；订阅退款及并发锁场景也应在两种数据库补验。

## 部署步骤

1. 备份主库、独立日志库、环境配置和当前部署镜像；先在数据库副本上启动新版，确认迁移、登录、OnlyArt、价格和历史任务查询/下载。
2. 确认任务插件总开关开启，内置 `rs-gateway` 未被禁用；检查渠道 61 仍为网关，新增 Task Plugin 为 64。
3. 停止旧版所有实例及任务轮询进程，再启动新版。历史任务 platform 会被迁移，不能让不识别 `rs-gateway` 的旧实例与新版并行处理任务。
4. 用实际 gateway 验证 Claude/Responses 工具图片、流式请求、视频生成/下载、按秒及图片价格，并核对钱包、令牌、账本和消费日志。
5. 确认新增访问令牌设置及旧令牌退役期限符合使用安排，再切换流量。

回滚应同时恢复旧镜像和升级前数据库备份；仅回退代码不能撤销任务 platform 等数据库迁移。
