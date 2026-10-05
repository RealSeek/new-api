Anthropic Messages 是 Claude 官方的对话接口格式，Claude Code、Anthropic SDK 以及很多 Agent 工具默认使用它。在 {{SITE_NAME}} 调用时 Base URL 填 `{{BASE_URL}}`（SDK 会自动拼接 `/v1/messages`），令牌使用 [API 密钥](/keys) 页面创建的 `sk-` 令牌。

## 接口地址

| 方法 | 路径           | 说明                   |
| ---- | -------------- | ---------------------- |
| POST | `/v1/messages` | 创建消息，支持流式输出 |

> [!NOTE]
> `/v1/messages/count_tokens` 目前未开放，调用会返回 404。需要估算用量时，可以参考响应里的 `usage` 字段。

## 认证与请求头

| 请求头                              | 说明                                     |
| ----------------------------------- | ---------------------------------------- |
| `x-api-key: sk-xxxxxxxx`            | Anthropic 官方写法，SDK 默认使用         |
| `Authorization: Bearer sk-xxxxxxxx` | 也可以使用，二选一即可                   |
| `anthropic-version`                 | 可选，不传时网关按 `2023-06-01` 转发     |
| `anthropic-beta`                    | 可选，原样转发给上游，用于开启 Beta 功能 |
| `Content-Type: application/json`    | 必填                                     |

## 支持的分组与模型

端点类型包含 `anthropic` 的分组都可以调用本接口：

| 分组                                           | 示例模型                                                            |
| ---------------------------------------------- | ------------------------------------------------------------------- |
| `claude-kiro`、`claude-kiro正价`、`claude-max` | `claude-sonnet-4-6`、`claude-opus-5-5`、`claude-haiku-4-5-20251001` |
| `DeepSeek`                                     | `deepseek-v4-pro`、`deepseek-v4.1-flash`                            |
| `glm`                                          | `glm-5.3`                                                           |
| `kimi`                                         | `kimi-k3`                                                           |
| `minimax`                                      | `MiniMax-M3`                                                        |
| `grok`                                         | `grok-4.7`                                                          |
| `codex-plus`、`codex-pro`、`codex-官key`       | `gpt-5.5`                                                           |

- 模型名必须与 [模型广场](/pricing) 中显示的完全一致，区分大小写。
- 令牌所属分组决定能调用哪些模型，分组说明见 [分组说明](/docs/groups)。

> [!IMPORTANT]
> Claude 分组只支持 Anthropic 格式，请通过本接口调用，用 `/v1/chat/completions` 等其他格式调用不保证可用。

## 请求示例

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/messages \
  -H "x-api-key: sk-xxxxxxxx" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 1024,
    "system": "你是一个简洁的助手",
    "messages": [
      { "role": "user", "content": "用一句话介绍你自己" }
    ]
  }'
```

```python
from anthropic import Anthropic

client = Anthropic(
    api_key="sk-xxxxxxxx",
    base_url="{{BASE_URL}}",
)

message = client.messages.create(
    model="claude-sonnet-4-6",
    max_tokens=1024,
    system="你是一个简洁的助手",
    messages=[{"role": "user", "content": "用一句话介绍你自己"}],
)
print(message.content[0].text)
```

```javascript
import Anthropic from '@anthropic-ai/sdk'

const client = new Anthropic({
  apiKey: 'sk-xxxxxxxx',
  baseURL: '{{BASE_URL}}',
})

const message = await client.messages.create({
  model: 'claude-sonnet-4-6',
  max_tokens: 1024,
  system: '你是一个简洁的助手',
  messages: [{ role: 'user', content: '用一句话介绍你自己' }],
})
console.log(message.content[0].text)
```

<!-- /code-group -->

SDK 的 `base_url` 不要带 `/v1`，否则会请求到 `/v1/v1/messages` 并返回 404。

## 响应示例

```json
{
  "id": "msg_xxxxxxxx",
  "type": "message",
  "role": "assistant",
  "model": "claude-sonnet-4-6",
  "content": [
    {
      "type": "text",
      "text": "我是一个简洁的 AI 助手，可以回答问题和协助编程。"
    }
  ],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 24,
    "output_tokens": 18
  }
}
```

- `content` 是数组，可能同时包含 `text`、`thinking`、`tool_use` 等多种块，读取文本时请按 `type` 过滤。
- `stop_reason` 常见取值：`end_turn`（正常结束）、`max_tokens`（达到输出上限）、`tool_use`（模型请求调用工具）、`stop_sequence`（命中停止序列）。
- `usage` 是本次请求的 Token 用量，命中提示词缓存时还会出现 `cache_read_input_tokens`、`cache_creation_input_tokens`。实际扣费以 [使用日志](/usage-logs) 为准。

## 主要参数

| 参数                   | 类型           | 说明                                                        |
| ---------------------- | -------------- | ----------------------------------------------------------- |
| `model`                | string         | 必填，模型名                                                |
| `messages`             | array          | 必填，对话消息，`role` 为 `user` 或 `assistant`，交替出现   |
| `max_tokens`           | integer        | 最多生成的 Token 数，建议每次都显式传入                     |
| `system`               | string / array | 系统提示，写在顶层，不放进 `messages`                       |
| `stream`               | boolean        | 是否流式输出                                                |
| `temperature`          | number         | 采样温度，取值 0–1                                          |
| `top_p`、`top_k`       | number         | 采样参数，一般只调其中一个                                  |
| `stop_sequences`       | array          | 停止序列                                                    |
| `tools`、`tool_choice` | array / object | 工具定义与调用策略                                          |
| `thinking`             | object         | 扩展思考，例如 `{"type": "enabled", "budget_tokens": 4096}` |
| `metadata`             | object         | 元数据，例如 `user_id`                                      |

`content` 可以是字符串，也可以是内容块数组。图片用 `image` 块传入：

```json
{
  "role": "user",
  "content": [
    { "type": "text", "text": "描述这张图片" },
    {
      "type": "image",
      "source": { "type": "url", "url": "https://example.com/cat.png" }
    }
  ]
}
```

也可以用 `"source": {"type": "base64", "media_type": "image/png", "data": "..."}` 内联图片，体积较大时建议先压缩。

> [!NOTE]
> 参数最终由上游模型解释。非 Claude 分组的模型对 `thinking`、`cache_control`、图片等能力的支持各不相同，不支持的参数可能被忽略或返回 400，请在 [模型广场](/pricing) 查看模型说明。

## 流式输出

请求体加 `"stream": true` 后，响应按 SSE 推送，每个事件都有 `event:` 行和对应的 `data:` 行：

```text
event: message_start
data: {"type":"message_start","message":{"id":"msg_xxxxxxxx","role":"assistant","content":[],"usage":{"input_tokens":24,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"我是"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":18}}

event: message_stop
data: {"type":"message_stop"}
```

<!-- code-group -->

```python
from anthropic import Anthropic

client = Anthropic(api_key="sk-xxxxxxxx", base_url="{{BASE_URL}}")

with client.messages.stream(
    model="claude-sonnet-4-6",
    max_tokens=1024,
    messages=[{"role": "user", "content": "写一首关于秋天的短诗"}],
) as stream:
    for text in stream.text_stream:
        print(text, end="", flush=True)
```

```javascript
import Anthropic from '@anthropic-ai/sdk'

const client = new Anthropic({ apiKey: 'sk-xxxxxxxx', baseURL: '{{BASE_URL}}' })

const stream = client.messages.stream({
  model: 'claude-sonnet-4-6',
  max_tokens: 1024,
  messages: [{ role: 'user', content: '写一首关于秋天的短诗' }],
})
stream.on('text', (text) => process.stdout.write(text))
await stream.finalMessage()
```

<!-- /code-group -->

- 文本增量在 `content_block_delta` 事件的 `delta.text` 中，思考内容的增量类型为 `thinking_delta`。
- 收到 `message_stop` 表示本次响应结束，最终用量在 `message_delta` 的 `usage` 中。
- 流中可能出现 `ping` 事件用于保持连接，可以直接忽略。

## 扩展思考

支持思考的模型可以通过 `thinking` 参数开启，`budget_tokens` 需小于 `max_tokens`：

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 8192,
  "thinking": { "type": "enabled", "budget_tokens": 4096 },
  "messages": [{ "role": "user", "content": "证明根号 2 是无理数" }]
}
```

响应的 `content` 中会先出现 `thinking` 块，再出现 `text` 块。多轮对话时请把上一轮的 `thinking` 块原样放回 `assistant` 消息。思考产生的 Token 按输出计费。

## 工具调用

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1024,
  "tools": [
    {
      "name": "get_weather",
      "description": "查询城市天气",
      "input_schema": {
        "type": "object",
        "properties": { "city": { "type": "string" } },
        "required": ["city"]
      }
    }
  ],
  "messages": [{ "role": "user", "content": "北京今天天气怎么样？" }]
}
```

模型需要调用工具时，`stop_reason` 为 `tool_use`，`content` 中包含 `tool_use` 块（带 `id`、`name`、`input`）。执行工具后，把结果作为 `user` 消息发回：

```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_xxxxxxxx",
      "content": "晴，22°C"
    }
  ]
}
```

## 常见错误

转发阶段的错误按 Anthropic 结构返回：

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "... (request id: 20261005xxxxxxxxxxxxxxxx)"
  }
}
```

| 状态码 | 常见原因                                                             | 处理方法                                                     |
| ------ | -------------------------------------------------------------------- | ------------------------------------------------------------ |
| 400    | 缺少 `model`、`messages` 格式不对、`budget_tokens` 超过 `max_tokens` | 对照上文检查请求体                                           |
| 401    | 令牌无效、已过期或被禁用                                             | 检查 `x-api-key` 是否完整，到 [API 密钥](/keys) 页面确认状态 |
| 403    | 令牌分组无权调用该模型                                               | 换用分组内的模型，或新建对应分组的令牌                       |
| 404    | `base_url` 多写了 `/v1`，或调用了未开放的 `count_tokens`             | 检查请求地址                                                 |
| 429    | 请求过于频繁或上游限流                                               | 降低并发，稍后重试                                           |
| 503    | 当前分组下该模型暂无可用线路                                         | 稍后重试或换用其他分组                                       |

鉴权、分组和额度校验阶段的错误仍使用 OpenAI 结构，末尾带有 `(request id: ...)`。更多状态码和排查方法见 [错误码说明](/docs/errors)。
