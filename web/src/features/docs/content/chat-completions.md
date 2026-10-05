Chat Completions 是 OpenAI 兼容的对话接口，绝大多数聊天客户端、SDK 和 AI 框架都支持这种格式。在 {{SITE_NAME}} 调用时 Base URL 填 `{{BASE_URL}}/v1`，令牌使用 [API 密钥](/keys) 页面创建的 `sk-` 令牌。

## 接口地址

| 方法 | 路径                   | 说明                                            |
| ---- | ---------------------- | ----------------------------------------------- |
| POST | `/v1/chat/completions` | 对话补全，支持流式输出                          |
| POST | `/v1/completions`      | 旧版文本补全（`prompt` 字段），新项目不建议使用 |

认证使用请求头 `Authorization: Bearer sk-xxxxxxxx`。其他认证写法和通用约定见 [API 概览](/docs/api-overview)。

## 支持的分组与模型

端点类型包含 `openai` 的分组都可以调用本接口：

| 分组                                     | 示例模型                                 |
| ---------------------------------------- | ---------------------------------------- |
| `DeepSeek`                               | `deepseek-v4-pro`、`deepseek-v4.1-flash` |
| `codex-plus`、`codex-pro`、`codex-官key` | `gpt-5.5`、`gpt-5.6-sol`、`gpt-6-sol`    |
| `glm`                                    | `glm-5.3`                                |
| `grok`                                   | `grok-4.7`                               |
| `kimi`                                   | `kimi-k3`                                |
| `minimax`                                | `MiniMax-M3`                             |
| `qwen`                                   | `qwen3.8-max`                            |
| `豆包`                                   | `doubao-seed-2-1-pro`                    |
| `hunyuan`                                | `hy3`                                    |
| `mimo`                                   | `mimo-v2.6-pro`                          |
| `酒馆`                                   | `[c]` 前缀的按次计费模型                 |

- 模型名必须与 [模型广场](/pricing) 中显示的完全一致，区分大小写。
- 令牌只能调用所属分组内的模型，用 `GET /v1/models` 可以查看当前令牌可用的模型。
- `酒馆` 分组按次计费，禁止用于 Vibe Code（AI 编程）。

> [!IMPORTANT]
> `claude-kiro`、`claude-kiro正价`、`claude-max` 只支持 Anthropic 格式，请改用 [Anthropic Messages](/docs/anthropic-messages)；`gemini` 分组只支持 Gemini 原生格式，见 [Gemini 原生接口](/docs/gemini-native)；生图和视频分组分别见 [图片生成](/docs/images) 和 [视频生成](/docs/videos)。

## 请求示例

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      {"role": "system", "content": "你是一个简洁的助手"},
      {"role": "user", "content": "用一句话介绍你自己"}
    ]
  }'
```

```powershell
$body = @{
  model    = "deepseek-v4-pro"
  messages = @(
    @{ role = "system"; content = "你是一个简洁的助手" }
    @{ role = "user"; content = "用一句话介绍你自己" }
  )
} | ConvertTo-Json -Depth 10

Invoke-RestMethod -Uri "{{BASE_URL}}/v1/chat/completions" -Method Post `
  -Headers @{ Authorization = "Bearer sk-xxxxxxxx" } `
  -ContentType "application/json; charset=utf-8" `
  -Body ([Text.Encoding]::UTF8.GetBytes($body))
```

```python
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")
completion = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=[
        {"role": "system", "content": "你是一个简洁的助手"},
        {"role": "user", "content": "用一句话介绍你自己"},
    ],
)
print(completion.choices[0].message.content)
```

```javascript
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: '{{BASE_URL}}/v1', apiKey: 'sk-xxxxxxxx' })
const completion = await client.chat.completions.create({
  model: 'deepseek-v4-pro',
  messages: [
    { role: 'system', content: '你是一个简洁的助手' },
    { role: 'user', content: '用一句话介绍你自己' },
  ],
})
console.log(completion.choices[0].message.content)
```

<!-- /code-group -->

## 响应示例

非流式请求返回一个完整的 JSON 对象：

```json
{
  "id": "chatcmpl-xxxxxxxx",
  "object": "chat.completion",
  "created": 1767225600,
  "model": "deepseek-v4-pro",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "我是一个乐于助人的 AI 助手。"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 21,
    "completion_tokens": 12,
    "total_tokens": 33
  }
}
```

- `finish_reason` 常见取值：`stop`（正常结束）、`length`（达到输出上限）、`tool_calls`（模型请求调用工具）、`content_filter`（被内容过滤）。
- `usage` 是本次请求的 token 用量，按量计费的模型据此计费。实际扣费以控制台 [使用日志](/usage-logs) 为准。
- 推理模型的思考过程可能放在 `message.reasoning_content` 中，是否返回取决于模型。

## 主要参数

| 参数                                    | 类型             | 说明                                                                     |
| --------------------------------------- | ---------------- | ------------------------------------------------------------------------ |
| `model`                                 | string           | 必填，模型名                                                             |
| `messages`                              | array            | 必填，对话消息列表，常用 `role` 为 `system`、`user`、`assistant`、`tool` |
| `stream`                                | boolean          | 是否流式输出，默认 `false`                                               |
| `stream_options`                        | object           | 流式选项，`include_usage` 控制流末尾是否返回用量                         |
| `max_tokens`                            | integer          | 最大输出 token 数                                                        |
| `max_completion_tokens`                 | integer          | 最大输出 token 数（含推理 token），推理模型通常用这个字段                |
| `temperature`                           | number           | 采样温度，越高越随机                                                     |
| `top_p`                                 | number           | 核采样概率阈值                                                           |
| `stop`                                  | string 或 array  | 停止序列                                                                 |
| `n`                                     | integer          | 生成的候选回复数量                                                       |
| `tools`                                 | array            | 可供模型调用的工具（函数）定义                                           |
| `tool_choice`                           | string 或 object | 工具选择策略，如 `auto`、`none`，或指定某个函数                          |
| `parallel_tool_calls`                   | boolean          | 是否允许一次返回多个工具调用                                             |
| `response_format`                       | object           | 结构化输出，如 `{"type": "json_object"}` 或 `json_schema`                |
| `reasoning_effort`                      | string           | 推理强度，如 `low`、`medium`、`high`，仅推理模型有效                     |
| `seed`                                  | integer          | 随机种子                                                                 |
| `frequency_penalty`、`presence_penalty` | number           | 重复惩罚                                                                 |

> [!NOTE]
> 参数最终由上游模型解释，不支持的参数可能被忽略或返回 400。网关也会透传部分厂商扩展字段，例如 `thinking`、`enable_thinking`，能否生效以对应模型的官方文档为准。

## 流式输出

请求体加 `"stream": true` 后，响应改为 SSE（`Content-Type: text/event-stream`）。每个数据块以 `data: ` 开头，内容是 `chat.completion.chunk` 对象，最后一行是 `data: [DONE]`：

```text
data: {"id":"chatcmpl-xxxxxxxx","object":"chat.completion.chunk","created":1767225600,"model":"deepseek-v4-pro","choices":[{"index":0,"delta":{"role":"assistant","content":"我是"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxxxxxxx","object":"chat.completion.chunk","created":1767225600,"model":"deepseek-v4-pro","choices":[{"index":0,"delta":{"content":"一个 AI 助手。"},"finish_reason":"stop"}]}

data: {"id":"chatcmpl-xxxxxxxx","object":"chat.completion.chunk","created":1767225600,"model":"deepseek-v4-pro","choices":[],"usage":{"prompt_tokens":21,"completion_tokens":12,"total_tokens":33}}

data: [DONE]
```

<!-- code-group -->

```bash
curl -N {{BASE_URL}}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "stream": true,
    "stream_options": {"include_usage": true},
    "messages": [{"role": "user", "content": "写一首关于秋天的短诗"}]
  }'
```

```python
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")
stream = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=[{"role": "user", "content": "写一首关于秋天的短诗"}],
    stream=True,
    stream_options={"include_usage": True},
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
    if chunk.usage:
        print("\n用量:", chunk.usage.total_tokens)
```

```javascript
import OpenAI from 'openai'

const client = new OpenAI({ baseURL: '{{BASE_URL}}/v1', apiKey: 'sk-xxxxxxxx' })
const stream = await client.chat.completions.create({
  model: 'deepseek-v4-pro',
  messages: [{ role: 'user', content: '写一首关于秋天的短诗' }],
  stream: true,
  stream_options: { include_usage: true },
})
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content ?? '')
  if (chunk.usage) console.log('\n用量:', chunk.usage.total_tokens)
}
```

<!-- /code-group -->

- 用量块的 `choices` 为空数组，只携带 `usage`。需要统计用量时建议显式传 `"stream_options": {"include_usage": true}`；不需要时传 `false`。
- 读取 `choices[0]` 前先判断数组是否为空，否则处理用量块时会报错。
- 网关可能发送以冒号开头的注释行（例如 `: PING`）保持连接，标准 SSE 客户端会自动忽略。
- 上游长时间没有返回任何数据时，网关会按站点设定的空闲超时结束这次流式响应。

## 工具调用

在 `tools` 中声明函数，模型需要调用时会返回函数名和参数：

```json
{
  "model": "deepseek-v4-pro",
  "messages": [{ "role": "user", "content": "北京今天天气怎么样？" }],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "查询城市天气",
        "parameters": {
          "type": "object",
          "properties": { "city": { "type": "string" } },
          "required": ["city"]
        }
      }
    }
  ]
}
```

- 模型决定调用工具时，`finish_reason` 为 `tool_calls`，`message.tool_calls` 中包含调用 `id`、函数名和 JSON 字符串形式的 `arguments`。
- 执行函数后，把结果作为 `role` 为 `tool` 的消息（带上对应的 `tool_call_id`）追加到 `messages`，再请求一次即可得到最终回复。
- 是否支持工具调用、并行调用和结构化输出取决于具体模型，请在 [模型广场](/pricing) 查看模型说明。

## 常见错误

| 状态码 | 典型提示                                           | 处理方法                                                          |
| ------ | -------------------------------------------------- | ----------------------------------------------------------------- |
| 400    | `未指定模型名称，模型名称不能为空`，或参数格式错误 | 检查请求体是否为合法 JSON，`model` 是否填写                       |
| 401    | `无效的令牌`                                       | 令牌填错、已禁用、已过期或令牌额度用尽，到 [API 密钥](/keys) 检查 |
| 403    | `该令牌无权访问模型 ...`                           | 令牌开启了模型限制，在令牌设置里放开该模型                        |
| 403    | `用户额度不足`                                     | 账户余额不足，到 [钱包](/wallet) 充值                             |
| 429    | `您已达到请求数限制：...`                          | 请求过于频繁，等待时间窗口过去后重试                              |
| 503    | `分组 xxx 下模型 yyy 无可用渠道`                   | 模型不属于令牌所在分组或线路暂不可用，核对模型名或改用其他分组    |
| 5xx    | 上游返回的错误                                     | 稍后重试；持续出现时带上请求 ID 反馈                              |

错误响应结构为 `{"error": {"message", "type", "param", "code"}}`，`message` 末尾的 `(request id: ...)` 与响应头 `X-Oneapi-Request-Id` 一致。完整说明见 [错误处理](/docs/errors)，通用约定见 [API 概览](/docs/api-overview)。
