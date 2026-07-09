[根目录](../CLAUDE.md) > **relay**

# relay — AI 中继与 provider 适配器（对接核心）⭐

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / relay
>
> ⭐ 这是**对接其他网关最可复用的核心模块**。无论是把 new-api 接到其他上游，还是把其他网关的能力接入 new-api，都从这里的 `Adaptor` 接口 + adaptor 注册表入手。

## 模块职责

把客户端发来的**统一格式请求**（OpenAI / Claude / Gemini / OpenAI-Responses / Embedding / Rerank / Audio / Image / 异步 Task）转换为各上游 provider 的私有协议，转发上游，再把上游响应/流式数据回转为客户端期望的格式，并在过程中统计 usage 供计费。

## 目录结构

```
relay/
├── *_handler.go             # 按请求格式的中继 handler（编排入口）
│   ├── claude_handler.go / gemini_handler.go / responses_handler.go
│   ├── image_handler.go / audio_handler.go / embedding_handler.go
│   ├── rerank_handler.go / compatible_handler.go / mjproxy_handler.go
│   └── relay_task.go        #   异步任务（视频/音乐）编排
├── relay_adaptor.go         # ⭐ adaptor 注册表：GetAdaptor / GetTaskAdaptor
├── channel/
│   ├── adapter.go           # ⭐ Adaptor / TaskAdaptor / OpenAIVideoConverter 接口定义
│   ├── api_request.go       #   通用上游请求
│   ├── {provider}/          #   各 provider 适配器（openai/claude/gemini/aws/ali/...）
│   │   ├── adaptor.go       #     实现 channel.Adaptor
│   │   ├── constants.go     #     ModelList、ChannelName
│   │   ├── dto.go           #     provider 私有请求/响应结构
│   │   └── relay-*.go       #     请求转换 / 响应解析 / 流式处理
│   └── task/{platform}/     #   异步任务适配器（kling/suno/sora/vidu/hailuo/...）
├── common/                  # RelayInfo、billing、outbound_body、relay_utils、stream_status
├── helper/                  # valid_request（含 maxTokensLimit）、price、stream_scanner、model_mapped
└── constant/                # relay 层常量（relay mode 等）
```

## 核心接口：`channel.Adaptor`（`relay/channel/adapter.go`）

每个同步 provider 适配器都实现该接口。请求生命周期方法：

| 方法 | 职责 |
| --- | --- |
| `Init(info)` | 初始化，读取 `RelayInfo`（是否流式、渠道信息等） |
| `GetRequestURL(info)` | 构造上游请求 URL |
| `SetupRequestHeader(c, header, info)` | 设置上游鉴权/自定义头 |
| `ConvertOpenAIRequest(...)` | OpenAI Chat 请求 → 上游格式 |
| `ConvertClaudeRequest(...)` | Claude Messages 请求 → 上游格式 |
| `ConvertGeminiRequest(...)` | Gemini 请求 → 上游格式 |
| `ConvertOpenAIResponsesRequest(...)` | OpenAI Responses 请求 → 上游格式 |
| `ConvertRerankRequest / ConvertEmbeddingRequest / ConvertAudioRequest / ConvertImageRequest` | 其余端点的请求转换 |
| `DoRequest(c, info, body)` | 实际发起上游 HTTP 请求 |
| `DoResponse(c, resp, info)` | 解析上游响应/流，回写客户端，返回 `usage` 供计费 |
| `GetModelList()` / `GetChannelName()` | 声明支持的模型与渠道名 |

> 许多 provider 通过「先转成 OpenAI 格式再复用 openai 逻辑」实现：如 `openai.Adaptor.ConvertClaudeRequest` 内部调 `service.ClaudeToOpenAIRequest` 再走 `ConvertOpenAIRequest`。跨格式互转逻辑集中在 `service/`（`convert.go`、`relayconvert/`、`openaicompat`）。

## 异步任务接口：`channel.TaskAdaptor`

用于视频/音乐等**异步任务**（先提交、后轮询）。除 `BuildRequestURL/Header/Body`、`DoRequest/DoResponse`、`FetchTask/ParseTaskResult` 外，含三个**计费钩子**（务必理解，改动需遵守计费安全不变式）：

- `EstimateBilling(c, info) map[string]float64` — 提交前按请求参数（如 `seconds`、分辨率）估算 OtherRatios，用于**预扣费**。
- `AdjustBillingOnSubmit(info, taskData) map[string]float64` — 上游返回实际参数后调整 ratios，结算差额。
- `AdjustBillingOnComplete(task, taskResult) int` — 任务达终态时返回实际 quota，触发补扣/退款。

## 请求/响应转换流程（简）

```
client 请求
  → middleware（distributor 选渠道，写入 channel_type/apiType 到 gin.Context）
  → relay/*_handler.go（按格式编排）
  → GetAdaptor(apiType)                     // relay_adaptor.go
  → Init → Convert*Request → SetupRequestHeader → GetRequestURL
  → DoRequest（发上游）
  → DoResponse（解析/流式回写，产出 usage）
  → 计费结算（service/quota、pkg/billingexpr）
```

- 同步 adaptor 选择：`GetAdaptor(apiType int)`（`relay/relay_adaptor.go`），`apiType` 来自 `constant/api_type.go`。
- 异步 task adaptor 选择：`GetTaskAdaptor(platform)`（同文件），按 `constant.ChannelType*` 分发。

## 如何新增一个 provider 适配器

1. **定义常量**：在 `constant/api_type.go` 加 `APIType{X}`，在 `constant/channel.go` 加 `ChannelType{X}`。
2. **建目录** `relay/channel/{provider}/`：
   - `adaptor.go` 实现 `channel.Adaptor` 全部方法；
   - `constants.go` 提供 `GetModelList()`/`GetChannelName()` 数据；
   - `dto.go` 定义上游私有请求/响应结构；
   - `relay-*.go` 写请求转换、响应解析、流式处理。
3. **注册**：在 `relay/relay_adaptor.go` 的 `GetAdaptor()` switch 增加 `case constant.APIType{X}: return &{provider}.Adaptor{}`。
4. **StreamOptions**：确认 provider 是否支持 `stream_options.include_usage`；支持则将其加入 `streamSupportedChannels`（使 `info.SupportStreamOptions` 为真，adaptor 内据此设置 `StreamOptions{IncludeUsage: true}`）。
5. **异步任务** provider：改实现 `channel.TaskAdaptor`，放 `relay/channel/task/{platform}/`，在 `GetTaskAdaptor()` 注册，并实现三个计费钩子。

## DTO / 计费安全约束（务必遵守，来自根 AGENTS.md）

- **可选标量字段用指针 + `omitempty`**（`*int`/`*uint`/`*float64`/`*bool`）：客户端缺省 → `nil` 省略；显式 `0`/`false` → 保留并发上游。禁止非指针标量配 `omitempty`（会静默丢零值）。
- **JSON 收发一律走 `common.Marshal/Unmarshal` 等包装**，禁止直接 `encoding/json`。
- **计费乘数必须先校验上界**：max-tokens 类走 `relay/helper/valid_request.go` 的 `maxTokensLimit`；图片 `n` 走 `dto.MaxImageN`；视频时长走 `relaycommon.MaxTaskDurationSeconds`。
- **quota 换算禁止裸 `int(...)` 强转**，统一用 `common/quota_math.go`（`QuotaFromFloat`/`QuotaRound`/`QuotaFromDecimal`，或 `*Checked` 变体）。
- 详见 [pkg/billingexpr/CLAUDE.md](../pkg/billingexpr/CLAUDE.md) 与 `pkg/billingexpr/expr.md`。

## 已注册 provider（节选，见 `relay_adaptor.go`）

openai · anthropic(claude) · gemini · ali · baidu / baidu_v2 · aws · tencent · xunfei · zhipu / zhipu_4v · ollama · perplexity · cohere · dify · jina · cloudflare · siliconflow · vertex · mistral · deepseek · mokaai · volcengine · openrouter · xinference · xai · coze · jimeng · moonshot(Claude API) · submodel · minimax · replicate · codex · advancedcustom
异步任务：suno · kling · sora · vidu · hailuo · doubao · vertex · gemini(video) · ali · jimeng

## 常见问题 (FAQ)

- **Q：新增渠道流式不计费/usage 为 0？** A：确认是否支持 `StreamOptions` 并加入 `streamSupportedChannels`，否则上游不回传 usage。
- **Q：显式传的 `0`/`false` 被丢了？** A：该字段用了非指针 + `omitempty`；改为 `*T` + `omitempty`。
- **Q：想复用现有 openai 逻辑？** A：在新 adaptor 的 `Convert*Request` 里先转成 `dto.GeneralOpenAIRequest` 再委托 openai 转换（参考 `openai/adaptor.go`）。

## 相关文件清单

- 接口：`relay/channel/adapter.go`
- 注册表：`relay/relay_adaptor.go`
- 校验/计费：`relay/helper/valid_request.go`、`relay/helper/price.go`、`relay/common/billing.go`
- 中继上下文：`relay/common/relay_info.go`
- 格式互转：`service/convert.go`、`service/relayconvert/`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 relay 中继与 adaptor 机制文档（接口约定、转换流程、新增 provider 步骤、DTO/计费约束）。
