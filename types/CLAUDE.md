[根目录](../CLAUDE.md) > **types**

# types — 跨层共享类型（错误体系 / 计费乘数 / 中继格式 / 文件来源）

> 更新时间：2026-07-09 23:51:43 ｜ 面包屑：根目录 / types
>
> `package types`，定义跨层复用的核心类型：统一错误体系、计费价格数据、中继格式枚举、文件来源抽象与并发容器。仅依赖 `common`，被 `relay/`、`service/`、`dto/` 等大量引用。**改动 `price_data.go` 前必须先读根 `AGENTS.md` 的计费安全不变式。**

## 模块职责

为「统一请求格式 → 多 provider」的中继链路提供公共类型底座：错误在各协议（OpenAI/Claude/Gemini）间转换、计费乘数（OtherRatios）安全累积、文件（图片/音频/视频）来源统一抽象、中继格式枚举等。

## 文件清单与职责

| 文件 | 职责 |
| --- | --- |
| `relay_format.go` | `RelayFormat`（字符串枚举）：`openai` / `claude` / `gemini` / `openai_responses` / `openai_responses_compaction` / `openai_audio` / `openai_image` / `openai_realtime` / `rerank` / `embedding` / `task` / `mj_proxy`。理解「统一请求格式」的入口。 |
| `price_data.go` | ⭐ `PriceData`（模型价/倍率/缓存倍率/图片音频倍率等）+ **`OtherRatios` 计费乘数安全入口**。 |
| `error.go` | ⭐ `NewAPIError` 统一错误类型 + `ErrorType` / `ErrorCode` 枚举 + 构造器与协议互转。 |
| `channel_error.go` | `ChannelError`（渠道错误上下文：channelId/type/name/isMultiKey/usingKey/autoBan），用于渠道级错误归因与自动禁用。 |
| `file_source.go` | `FileSource` 接口（`URLSource` / `Base64Source`）+ `CachedFileData`（内存/磁盘两级缓存，懒加载、带清理注册）。 |
| `file_data.go` | `LocalFileData`（MimeType/Base64Data/Url/Size 的轻量载体）。 |
| `request_meta.go` | `FileType` / `TokenType` 枚举、`TokenCountMeta`（计 token 的元信息）、`FileMeta`、`RequestMeta`。 |
| `rw_map.go` | `RWMap[K,V]` 读写锁并发 map（`MarshalJSON`/`UnmarshalJSON` 走 `common.*`）。 |
| `set.go` | `Set[T]` 泛型集合（Add/Remove/Contains/Items…）。 |

## ⭐ 计费乘数安全入口：`PriceData.OtherRatios`（`price_data.go`）

- `otherRatios` 字段**私有**，只能通过 `AddOtherRatio(key, ratio)` / `ReplaceOtherRatios(...)` 写入；`OtherRatios()` 只读导出。**禁止绕过方法直接写 map**（根 `AGENTS.md` 明确要求）。
- 守卫函数 `isValidOtherRatio` 拒绝 **非正数、NaN、+Inf**：这些值会污染下游 quota 乘法（`int(NaN * quota)` 会回绕成负计费/贷记）。**不得削弱该守卫。**
- 应用乘数用 `OtherRatioMultiplier()` / `ApplyOtherRatiosToFloat()` / `ApplyOtherRatiosToDecimal()`；不要在别处重写乘法。
- 配合 `common/quota_math.go`（`QuotaFromFloat`/`QuotaRound`/`QuotaFromDecimal` 及 `*Checked` 变体）完成最终 quota 换算，**禁止裸 `int(...)` 强转**。
- 计费全链路（校验 → EstimateBilling/OtherRatios → quota 换算 → 预扣 → 结算/退款）不变式详见 [pkg/billingexpr/CLAUDE.md](../pkg/billingexpr/CLAUDE.md)、`pkg/billingexpr/expr.md` 与根 `AGENTS.md`。

## 错误体系：`NewAPIError`（`error.go`）

- 核心类型 `NewAPIError`：内含 `errorType`（`new_api_error`/`openai_error`/`claude_error`/`gemini_error`/…）、`errorCode`、`StatusCode`、可选 `RelayError`（原始 provider 错误）。
- 构造器：`NewError` / `NewErrorWithStatusCode` / `NewOpenAIError` / `WithOpenAIError` / `WithClaudeError` / `InitOpenAIError`；`errors.As` 会保留深层已包装的 `NewAPIError`。
- 协议互转：`ToOpenAIError()` / `ToClaudeError()`（把上游错误转成客户端期望格式）。
- 脱敏：`MaskSensitiveError()` / `ToOpenAIError` 内部对非 `count_token_failed` 错误调用 `common.MaskSensitiveInfo`。
- 语义判定 / 选项：`IsChannelError`（`channel:` 前缀）、`IsSkipRetryError`、`ErrOptionWithSkipRetry`、`ErrOptionWithNoRecordErrorLog`、`ErrOptionWithHideErrMsg` 等。

## 文件来源抽象：`FileSource`（`file_source.go`）

- 统一处理图片/音频/视频等多模态输入的两种来源：URL 与 base64。`NewFileSourceFromData` 按 `http(s)://` 前缀自动选择 `URLSource` / `Base64Source`。
- `CachedFileData` 支持内存（小文件）与磁盘（大文件）两种缓存，`Close()` 负责删磁盘临时文件并回调统计扣减；配合 `ContextKeyFileSourcesToCleanup`（见 `constant/context_key.go`）在请求结束时清理。

## 常见问题 (FAQ)

- **Q：想给某请求加一个计费倍率？** A：走 `PriceData.AddOtherRatio(key, ratio)`，并先确认该数量已在请求校验处限制上界（防溢出成负计费）。
- **Q：上游返回的错误如何回给不同格式的客户端？** A：构造 `NewAPIError` 后按目标格式调 `ToOpenAIError()` / `ToClaudeError()`。
- **Q：`RWMap` / `Set` 的 JSON 序列化？** A：`RWMap` 已实现 `Marshal/UnmarshalJSON` 且走 `common.*` 包装；直接用即可。

## 相关文件清单

- 计费：`price_data.go` ↔ `common/quota_math.go` ↔ `pkg/billingexpr/`
- 错误：`error.go` / `channel_error.go` ↔ `relay/`、`middleware/`
- 文件：`file_source.go` / `request_meta.go` ↔ `relay/` 多模态与 token 计数

## 变更记录 (Changelog)

- **2026-07-09 23:51:43**：初始化 types 共享类型文档（RelayFormat 枚举、PriceData/OtherRatios 计费乘数安全入口、NewAPIError 错误体系、FileSource 文件来源抽象）。
