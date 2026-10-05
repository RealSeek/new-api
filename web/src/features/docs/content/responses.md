Responses 是 OpenAI 新一代的对话接口，Codex CLI 等编程工具默认使用这种格式。它用 `input` 代替 `messages`，用 `instructions` 传系统提示，流式输出按事件类型推送。在 {{SITE_NAME}} 调用时 Base URL 填 `{{BASE_URL}}/v1`，令牌使用 [API 密钥](/keys) 页面创建的 `sk-` 令牌。

## 接口地址

| 方法 | 路径                    | 说明                                                    |
| ---- | ----------------------- | ------------------------------------------------------- |
| POST | `/v1/responses`         | 创建响应，支持流式输出                                  |
| POST | `/v1/responses/compact` | 压缩对话上下文（Codex 使用）                            |
| GET  | `/v1/responses`         | WebSocket 模式，握手后按 `response.create` 消息发起请求 |

认证方式与其他 OpenAI 格式接口相同：请求头 `Authorization: Bearer sk-xxxxxxxx`。通用约定见 [接口概览](/docs/api-overview)。

## 支持的分组与模型

端点类型包含 `openai-response` 的分组可以调用本接口：

| 分组                                     | 示例模型                                 |
| ---------------------------------------- | ---------------------------------------- |
| `codex-plus`、`codex-pro`、`codex-官key` | `gpt-5.5`、`gpt-5.6-sol`、`gpt-6-sol`    |
| `DeepSeek`                               | `deepseek-v4-pro`、`deepseek-v4.1-flash` |
| `glm`                                    | `glm-5.3`                                |
| `grok`                                   | `grok-4.7`                               |
| `kimi`                                   | `kimi-k3`                                |
| `minimax`                                | `MiniMax-M3`                             |

- `/v1/responses/compact` 只有 codex 分组支持（端点类型 `openai-response-compact`）。
- 模型名必须与 [模型广场](/pricing) 中显示的完全一致，区分大小写。
- `qwen`、`豆包`、`hunyuan`、`mimo`、`酒馆` 只支持 `openai` 端点类型，请改用 [Chat Completions](/docs/chat-completions)。

> [!IMPORTANT]
> `claude-kiro`、`claude-kiro正价`、`claude-max` 只支持 Anthropic 格式，请改用 [Anthropic Messages](/docs/anthropic-messages)；`gemini` 分组见 [Gemini 原生接口](/docs/gemini-native)。

## 请求示例

<!-- code-group -->

```bash title="curl"
curl {{BASE_URL}}/v1/responses \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.5",
    "instructions": "你是一个简洁的助手",
    "input": "用一句话介绍你自己"
  }'
```

```python title="Python"
from openai import OpenAI

client = OpenAI(
    base_url="{{BASE_URL}}/v1",
    api_key="sk-xxxxxxxx",
)

response = client.responses.create(
    model="gpt-5.5",
    instructions="你是一个简洁的助手",
    input="用一句话介绍你自己",
)
print(response.output_text)
```

```javascript title="Node.js"
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: '{{BASE_URL}}/v1',
  apiKey: 'sk-xxxxxxxx',
})

const response = await client.responses.create({
  model: 'gpt-5.5',
  instructions: '你是一个简洁的助手',
  input: '用一句话介绍你自己',
})
console.log(response.output_text)
```

<!-- /code-group -->

`input` 可以是字符串，也可以是消息数组，用于多轮对话或传入图片：

```json
{
  "model": "gpt-5.5",
  "input": [
    {
      "role": "user",
      "content": [
        { "type": "input_text", "text": "这张图里有什么？" },
        { "type": "input_image", "image_url": "https://example.com/cat.png" }
      ]
    }
  ]
}
```

## 响应示例

```json
{
  "id": "resp_xxxxxxxx",
  "object": "response",
  "status": "completed",
  "model": "gpt-5.5",
  "output": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "我是一个 AI 助手，可以回答问题和处理文本。"
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 24,
    "output_tokens": 15,
    "total_tokens": 39
  }
}
```

- `output` 是数组，可能同时包含推理（`reasoning`）、消息（`message`）和工具调用（`function_call`）等条目。SDK 的 `output_text` 会把所有文本拼接好。
- `usage` 是本次请求的 token 用量，计费以 [使用日志](/usage-logs) 中的记录为准。

## 主要参数

| 参数                   | 类型           | 说明                                   |
| ---------------------- | -------------- | -------------------------------------- |
| `model`                | string         | 必填，模型名                           |
| `input`                | string / array | 输入内容，字符串或消息数组             |
| `instructions`         | string         | 系统提示                               |
| `stream`               | boolean        | 是否流式输出，默认 `false`             |
| `max_output_tokens`    | integer        | 输出 token 上限                        |
| `temperature`、`top_p` | number         | 采样参数                               |
| `reasoning`            | object         | 推理设置，例如 `{"effort": "high"}`    |
| `tools`、`tool_choice` | array / string | 工具定义与调用策略                     |
| `text`                 | object         | 输出格式，例如结构化输出的 JSON Schema |
| `previous_response_id` | string         | 接续上一次响应的上下文                 |
| `metadata`             | object         | 自定义元数据                           |

> [!NOTE]
> 参数最终由上游模型解释，不支持的参数可能被忽略或返回 400。`previous_response_id` 依赖上游保存的会话，不同线路的支持情况不同；需要稳定的多轮对话时，建议在 `input` 中自行带上历史消息。

## 流式输出

请求体加 `"stream": true` 后，响应为 SSE 事件流，每个事件带 `event` 类型和 JSON 数据：

```text
event: response.created
data: {"type":"response.created","response":{"id":"resp_xxxxxxxx","status":"in_progress"}}

event: response.output_text.delta
data: {"type":"response.output_text.delta","delta":"我是"}

event: response.completed
data: {"type":"response.completed","response":{"id":"resp_xxxxxxxx","status":"completed","usage":{"input_tokens":24,"output_tokens":15,"total_tokens":39}}}
```

<!-- code-group -->

```python title="Python"
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")

stream = client.responses.create(
    model="gpt-5.5",
    input="写一首关于秋天的短诗",
    stream=True,
)
for event in stream:
    if event.type == "response.output_text.delta":
        print(event.delta, end="", flush=True)
```

```javascript title="Node.js"
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: '{{BASE_URL}}/v1', apiKey: 'sk-xxxxxxxx' })

const stream = await client.responses.create({
  model: 'gpt-5.5',
  input: '写一首关于秋天的短诗',
  stream: true,
})
for await (const event of stream) {
  if (event.type === 'response.output_text.delta') {
    process.stdout.write(event.delta)
  }
}
```

<!-- /code-group -->

- 文本增量在 `response.output_text.delta` 事件里，用量在最后的 `response.completed` 事件里。
- 与 Chat Completions 不同，Responses 流不以 `data: [DONE]` 结尾，收到 `response.completed`（或 `response.failed`）即表示结束。
- 上游长时间没有返回任何数据时，网关会按站点设定的空闲超时结束这次流式响应。

## 工具调用

工具定义直接写在 `tools` 数组里，不需要外层的 `function` 包装：

```json
{
  "model": "gpt-5.5",
  "input": "上海今天天气怎么样？",
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "查询城市天气",
      "parameters": {
        "type": "object",
        "properties": { "city": { "type": "string" } },
        "required": ["city"]
      }
    }
  ]
}
```

模型需要调用工具时，`output` 中会出现 `function_call` 条目，包含 `call_id`、`name` 和 `arguments`。执行工具后，把结果作为 `function_call_output` 条目（带相同的 `call_id`）连同之前的条目一起放进 `input`，再次请求即可得到最终回复。

## 在 Codex 中使用

Codex CLI 通过本接口调用模型，`wire_api` 需保持 `responses`，`base_url` 填 `{{BASE_URL}}/v1`。完整配置见 [Codex](/docs/codex)。

## 常见错误

| 状态码 | 常见原因                                        | 处理方法                                 |
| ------ | ----------------------------------------------- | ---------------------------------------- |
| 400    | 请求体格式错误、缺少 `model` 或参数不被模型支持 | 检查 JSON 和参数，去掉模型不支持的字段   |
| 401    | 令牌无效、已过期或未传                          | 检查 `Authorization` 请求头和令牌状态    |
| 403    | 令牌无权使用该模型或分组                        | 检查令牌的分组和模型限制                 |
| 404    | 路径写错，例如 `/v1/v1/responses`               | Base URL 只写到 `/v1`，不要重复拼接      |
| 429    | 请求过于频繁或上游限流                          | 降低并发，稍后重试                       |
| 503    | 当前分组下没有可用渠道提供该模型                | 确认模型属于令牌所在分组，或换用其他分组 |

错误响应使用 OpenAI 格式，末尾带有 `(request id: ...)`。更多状态码和排查方法见 [错误码说明](/docs/errors)。
