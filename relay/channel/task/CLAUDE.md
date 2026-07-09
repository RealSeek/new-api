[根目录](../../../CLAUDE.md) > [relay](../../CLAUDE.md) > channel > **task**

# relay/channel/task — 异步任务适配器（视频 / 音乐）

> 更新时间：2026-07-09 23:51:43 ｜ 面包屑：根目录 / relay / channel / task
>
> 本目录承接 [relay/CLAUDE.md](../../CLAUDE.md) 中的 **`channel.TaskAdaptor`** 接口，收录所有「先提交、后轮询」的异步任务平台适配器（视频生成、音乐生成）。先读 relay 主文档理解 adaptor 全貌，再看这里的各平台实现与新增步骤。

## 模块职责

把统一的异步任务请求（`RelayFormatTask`）转换为各上游平台的提交协议，提交后由轮询循环拉取任务状态直至终态，并在提交/轮询过程中通过三个**计费钩子**完成预扣、差额结算与最终 quota 结算。

## 核心接口：`channel.TaskAdaptor`（定义于 `relay/channel/adapter.go`）

| 阶段 | 方法 | 职责 |
| --- | --- | --- |
| 初始化/校验 | `Init(info)`、`ValidateRequestAndSetAction(c, info)` | 读取 `RelayInfo`；校验请求并设定 action（**含时长等参数上界校验**）。 |
| 计费 | `EstimateBilling(c, info) map[string]float64` | 提交前按请求参数（`seconds`、分辨率等）估算 OtherRatios，用于**预扣费**。 |
| 计费 | `AdjustBillingOnSubmit(info, taskData) map[string]float64` | 上游提交返回实际参数后调整 ratios，结算差额；无需调整返回 `nil`。 |
| 计费 | `AdjustBillingOnComplete(task, taskResult) int` | 任务达终态时返回实际 quota，触发补扣/退款；返回 `0` 表示维持预扣。 |
| 请求 | `BuildRequestURL/Header/Body`、`DoRequest`、`DoResponse` | 构造并发起上游提交请求，解析出 `taskID` + `taskData`。 |
| 轮询 | `FetchTask`、`ParseTaskResult` | 拉取任务状态并解析为 `relaycommon.TaskInfo`。 |
| 元信息 | `GetModelList`、`GetChannelName` | 声明支持的模型与渠道名。 |

> 不需要自定义计费的平台可**嵌入 `taskcommon.BaseBilling`**（`helpers.go`），它为三个计费钩子提供 no-op 默认实现（`EstimateBilling`→nil、`AdjustBillingOnSubmit`→nil、`AdjustBillingOnComplete`→0）。

## 各平台适配器

| 子目录 | 平台 | 类别 | 分发键（`GetTaskAdaptor`） |
| --- | --- | --- | --- |
| `kling/` | 快手 Kling | 视频 | `ChannelTypeKling`(50) |
| `sora/` | OpenAI Sora | 视频 | `ChannelTypeSora`(55) / `ChannelTypeOpenAI`(1) |
| `vidu/` | Vidu | 视频 | `ChannelTypeVidu`(52) |
| `hailuo/` | 海螺（MiniMax） | 视频 | `ChannelTypeMiniMax`(35) |
| `doubao/` | 豆包/火山 | 视频 | `ChannelTypeDoubaoVideo`(54) / `ChannelTypeVolcEngine`(45) |
| `gemini/` | Google Gemini（视频/图像） | 视频 | `ChannelTypeGemini`(24) |
| `vertex/` | Google Vertex AI | 视频 | `ChannelTypeVertexAi`(41) |
| `ali/` | 阿里云 | 视频 | `ChannelTypeAli`(17) |
| `jimeng/` | 即梦 Jimeng | 视频 | `ChannelTypeJimeng`(51) |
| `suno/` | Suno | 音乐 | `TaskPlatformSuno`("suno") |
| `taskcommon/` | 公共辅助（非平台） | — | 被各平台复用 |

- 注册入口：`relay/relay_adaptor.go` 的 **`GetTaskAdaptor(platform constant.TaskPlatform)`**。`suno` 走 `TaskPlatformSuno`；其余把 `platform` 解析为 `ChannelType` 数字后 switch 分发。
- 平台判定：`GetTaskPlatform(c)` 优先取 `channel_type`（数字），否则取 `platform` 字符串。

## `taskcommon/helpers.go` 公共辅助

- `UnmarshalMetadata(metadata, target)` — 把任务 `metadata` map 经 JSON round-trip 填入结构体；**内部 `delete(metadata, "model")` 防止 metadata 覆盖 model 字段造成计费绕过**。
- `EncodeLocalTaskID` / `DecodeLocalTaskID` — base64 编解码上游 operation name（Gemini/Vertex 用作 taskID）。
- `BuildProxyURL(taskID)` — 生成 `.../v1/videos/{taskID}/content` 视频代理地址。
- `DefaultString` / `DefaultInt`、进度常量（`ProgressSubmitted`…`ProgressComplete`）、`BaseBilling`（可嵌入的 no-op 计费）。

## ⚠️ 计费安全（务必遵守，来自根 `AGENTS.md`）

- **时长上界**：用户传入的视频时长是计费乘数（OtherRatio `"seconds"`），必须校验上界。`relay/common/relay_utils.go` 定义 `const MaxTaskDurationSeconds = 3600`，`validateTaskDurationBounds` 要求 `0 <= seconds <= MaxTaskDurationSeconds`，否则 400。**新增平台的校验必须复用该常量，不要另造上界。**
- **谨防绕过路径**：`metadata` map、passthrough 字段可能夹带同类数量，任何从这些路径读乘数的 adaptor 必须本地施加同样上界/钳制。
- **quota 换算**：三个计费钩子产出的比率/额度换算统一走 `common/quota_math.go`（`QuotaFromFloat`/`QuotaRound`/`QuotaFromDecimal` 及 `*Checked` 变体），**禁止裸 `int(...)` 强转**；乘数写入必须经 `types.PriceData.AddOtherRatio`（拒绝非正/NaN/+Inf）。
- 预扣（EstimateBilling）与结算（AdjustBilling*）都要安全：超大 quota 应在预扣阶段以「余额不足」失败，绝不静默回绕。完整链路不变式见 [pkg/billingexpr/CLAUDE.md](../../../pkg/billingexpr/CLAUDE.md) 与 `pkg/billingexpr/expr.md`。

## 如何新增一个异步任务平台

1. **常量**：在 `constant/channel.go` 加 `ChannelType{X}`（视频/音乐任务用 `ChannelType` 分发，通常无需 `APIType`）。
2. **建目录** `relay/channel/task/{platform}/`：`adaptor.go` 实现 `channel.TaskAdaptor`；可嵌入 `taskcommon.BaseBilling` 省去无关计费钩子。
3. **校验**：在 `ValidateRequestAndSetAction` 里做参数校验；涉及视频时长务必复用 `relaycommon.MaxTaskDurationSeconds` 做上界。
4. **计费钩子**：按需实现 `EstimateBilling`（预扣）/ `AdjustBillingOnSubmit`（差额）/ `AdjustBillingOnComplete`（终态 quota），并遵守上面的计费安全约束。
5. **提交/轮询**：实现 `BuildRequestURL/Header/Body`、`DoRequest`、`DoResponse`（产出 `taskID`+`taskData`）、`FetchTask`、`ParseTaskResult`。
6. **注册**：在 `relay/relay_adaptor.go` 的 `GetTaskAdaptor()` 增加 `case constant.ChannelType{X}: return &{platform}.TaskAdaptor{}`。

## 常见问题 (FAQ)

- **Q：任务提交成功但一直不结算？** A：检查 `ParseTaskResult` 是否正确映射终态，以及 `AdjustBillingOnComplete` 是否返回实际 quota。
- **Q：一个 provider 既有同步 chat 又有异步视频怎么办？** A：同步走 `Adaptor`（`GetAdaptor`/`APIType`），异步走 `TaskAdaptor`（`GetTaskAdaptor`/`ChannelType`），两套分别实现与注册（如 gemini/vertex/ali 都是这样）。
- **Q：时长可以不校验吗？** A：不可以。未校验的时长会成为计费乘数并可能溢出成负计费，必须用 `MaxTaskDurationSeconds` 限制。

## 相关文件清单

- 接口定义：`relay/channel/adapter.go`（`TaskAdaptor`）
- 注册分发：`relay/relay_adaptor.go`（`GetTaskAdaptor` / `GetTaskPlatform`）
- 公共辅助：`relay/channel/task/taskcommon/helpers.go`
- 时长上界：`relay/common/relay_utils.go`（`MaxTaskDurationSeconds`）
- 任务编排：`relay/relay_task.go`

## 变更记录 (Changelog)

- **2026-07-09 23:51:43**：初始化异步任务适配器文档（TaskAdaptor 提交→轮询模式与三计费钩子、各平台一览、taskcommon 辅助、MaxTaskDurationSeconds 时长上界、新增平台步骤）。
