[根目录](../CLAUDE.md) > **constant**

# constant — 全局常量（对接新 provider 的常量入口）

> 更新时间：2026-07-09 23:51:43 ｜ 面包屑：根目录 / constant
>
> 纯常量定义包（`package constant`），无业务逻辑，被 `relay/`、`model/`、`controller/`、`middleware/` 等广泛引用。**对接新的上游 provider 时，第一站通常就是这里**（新增 `APIType` / `ChannelType`）。

## 模块职责

集中定义全项目共享的枚举与常量：适配器类型（`APIType`）、渠道类型（`ChannelType`）、端点类型（`EndpointType`）、`gin.Context` 键（`ContextKey`）、异步任务平台（`TaskPlatform`）、多密钥模式、缓存键、结束原因、环境派生变量等。仅少量查表函数（如 `GetChannelTypeName`）。

## 文件清单与职责

| 文件 | 职责 |
| --- | --- |
| `api_type.go` | ⭐ `APIType`（iota 枚举，`APITypeOpenAI`…`APITypeAdvancedCustom`）。relay 层 `GetAdaptor(apiType)` 据此分发同步 adaptor。末尾 `APITypeDummy` 仅用于计数，**其后不得再加**。 |
| `channel.go` | ⭐ `ChannelType`（显式整数值 1…58）、`ChannelBaseURLs`（各渠道默认 BaseURL）、`ChannelTypeNames`（展示名 map）、`GetChannelTypeName()`、`ChannelSpecialBases`（glm/kimi/doubao coding-plan 等特殊 Claude/OpenAI BaseURL）。末尾 `ChannelTypeDummy` 仅用于计数。 |
| `endpoint_type.go` | `EndpointType`（字符串枚举）：`openai` / `openai-response` / `openai-response-compact` / `anthropic` / `gemini` / `jina-rerank` / `image-generation` / `embeddings` / `openai-video`。 |
| `context_key.go` | `ContextKey`（字符串枚举）：贯穿请求生命周期写入 `gin.Context` 的键，分 token / channel / user / auto-group 等组（如 `ContextKeyChannelType`、`ContextKeyOriginalModel`、`ContextKeyUserGroup`、`ContextKeyIsStream`）。 |
| `task.go` | `TaskPlatform`（`suno` / `mj`）、Suno/Task action 常量、`SunoModel2Action` 映射。 |
| `midjourney.go` | MJ 错误码、MJ action 常量（`IMAGINE`/`UPSCALE`/…）、`MidjourneyModel2Action`。 |
| `multi_key_mode.go` | `MultiKeyMode`：`random`（随机）/ `polling`（轮询）。 |
| `azure.go` | `AzureNoRemoveDotTime`（2025-05-10，Azure 模型名去点逻辑的时间分界）。 |
| `cache_key.go` | 用户相关缓存键格式（`user_group:%d` 等）与 token 字段名。 |
| `finish_reason.go` | OpenAI 风格结束原因字符串（`stop`/`tool_calls`/`length`/…）。 |
| `env.go` | 环境派生的全局可变变量（`StreamingTimeout`、`MaxRequestBodyMB`、`TaskTimeoutMinutes`、`TrustedRedirectDomains` 等），启动时由 env 赋值。 |
| `setup.go` | `Setup bool`（系统是否已完成初始化引导）。 |
| `waffo_pay_method.go` | Waffo 支付方式结构与 `DefaultWaffoPayMethods`（Card / Apple Pay / Google Pay）。 |

## 对接参考：两套编号体系（务必分清）

- **`ChannelType`（`channel.go`）** — 渠道/provider 类型，**存数据库、前端下拉展示**。前端 `web/default/src/features/channels/constants.ts` 的 `CHANNEL_TYPES` 与此一一对应。
- **`APIType`（`api_type.go`）** — **适配器类型**，relay 层 `GetAdaptor(apiType)`（`relay/relay_adaptor.go`）据此选择同步 adaptor。
- 二者编号不同，一个 provider 通常两者都要新增；从 `ChannelType` 到 `APIType` 的映射在 model/relay 层完成。
- **异步任务**走 `TaskPlatform`：`GetTaskAdaptor(platform)` 对 `suno` 走 `TaskPlatformSuno`，其余按 `ChannelType` 数字分发（详见 [relay/channel/task/CLAUDE.md](../relay/channel/task/CLAUDE.md)）。

## 如何新增一个 provider 的常量

1. **加渠道类型**：`channel.go` 中给 `ChannelType{X}` 分配未占用整数（放在 `ChannelTypeDummy` 之前），并同步补 `ChannelBaseURLs`、`ChannelTypeNames`（必要时 `ChannelSpecialBases`）。
2. **加适配器类型**：`api_type.go` 中在 `APITypeDummy` 之前追加 `APIType{X}`。
3. **同步前端**：在 `web/default/src/features/channels/constants.ts` 的 `CHANNEL_TYPES` / 展示顺序中补相同 `ChannelType`。
4. **异步任务平台**：若为视频/音乐类，改在 `GetTaskAdaptor` 用 `ChannelType` 分发，无需 `APIType`。
5. 具体 adaptor 实现步骤见 [relay/CLAUDE.md](../relay/CLAUDE.md)。

> 二开提示：`constant/` 是上游频繁改动的文件区（新增渠道会追加枚举），本地若也在此新增常量，容易与每日自动同步产生 merge 冲突。可行时把新常量集中成小改动、靠近文件末尾追加，降低冲突命中率。

## 常见问题 (FAQ)

- **Q：新增渠道后 relay 找不到 adaptor？** A：确认 `APIType` 已加且 `GetAdaptor` 有对应 `case`；`ChannelType` 与 `APIType` 是两套编号，别只加了一个。
- **Q：前端渠道下拉没有新类型？** A：`constant/channel.go` 只是后端；前端要另在 `channels/constants.ts` 的 `CHANNEL_TYPES` 补同号。
- **Q：`ContextKey` 直接用字符串行不行？** A：统一用本包常量，避免各处硬编码键名不一致。

## 相关文件清单

- 适配器分发：`api_type.go` → `relay/relay_adaptor.go`
- 渠道类型：`channel.go` → 前端 `channels/constants.ts`
- 任务平台：`task.go` → `relay/channel/task/`
- 上下文键：`context_key.go`（贯穿 `middleware/` → `relay/`）

## 变更记录 (Changelog)

- **2026-07-09 23:51:43**：初始化 constant 常量包文档（APIType/ChannelType/EndpointType/ContextKey/TaskPlatform 等，及对接新 provider 的常量入口指引）。
