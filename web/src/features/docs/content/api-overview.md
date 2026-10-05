{{SITE_NAME}} 在同一个站点下兼容 OpenAI、Anthropic 和 Gemini 三种接口格式，同一个令牌可以按所属分组调用不同格式的接口。本页汇总 Base URL、认证方式、接口列表、流式输出和请求 ID 等通用约定。

## Base URL

| 接口格式       | Base URL          | 完整地址示例                                                |
| -------------- | ----------------- | ----------------------------------------------------------- |
| OpenAI 兼容    | `{{BASE_URL}}/v1` | `{{BASE_URL}}/v1/chat/completions`                          |
| Anthropic 兼容 | `{{BASE_URL}}`    | `{{BASE_URL}}/v1/messages`                                  |
| Gemini 原生    | `{{BASE_URL}}`    | `{{BASE_URL}}/v1beta/models/gemini-3.1-pro:generateContent` |

> [!IMPORTANT]
> OpenAI SDK 和大多数 OpenAI 兼容客户端填写以 `/v1` 结尾的地址；Anthropic SDK、Claude Code 和 Gemini SDK 会自动拼接 `/v1/...` 或 `/v1beta/...`，直接填写站点地址即可。

## 认证

所有接口都使用在 [API 密钥](/keys) 页面创建的令牌（以 `sk-` 开头）。网关支持以下几种传递方式：

| 方式             | 写法                                                          | 适用接口                                          |
| ---------------- | ------------------------------------------------------------- | ------------------------------------------------- |
| Bearer           | `Authorization: Bearer sk-xxxxxxxx`                           | 所有接口                                          |
| Anthropic 风格   | `x-api-key: sk-xxxxxxxx`                                      | `/v1/messages`、`/v1/models`                      |
| Gemini 风格      | `x-goog-api-key: sk-xxxxxxxx` 或 URL 参数 `?key=sk-xxxxxxxx`  | `/v1beta/models/...`、`/v1/models`                |
| WebSocket 子协议 | `Sec-WebSocket-Protocol: openai-insecure-api-key.sk-xxxxxxxx` | `/v1/realtime`、`/v1/responses` 的 WebSocket 连接 |

- 令牌所属的分组决定能调用哪些模型和接口格式，见 [分组说明](/docs/groups)。
- URL 参数里的令牌可能出现在代理日志和浏览器历史中，非必要不要使用。
- 建议每个客户端使用单独的令牌，便于统计用量和单独停用，见 [创建令牌](/docs/api-keys)。

## 接口列表

### 对话与文本

| 方法 | 路径                                           | 格式      | 说明                                                               |
| ---- | ---------------------------------------------- | --------- | ------------------------------------------------------------------ |
| POST | `/v1/chat/completions`                         | OpenAI    | 对话补全，见 [Chat Completions](/docs/chat-completions)            |
| POST | `/v1/responses`                                | OpenAI    | Responses 接口，Codex 使用，见 [Responses](/docs/responses)        |
| GET  | `/v1/responses`                                | OpenAI    | Responses 的 WebSocket 连接                                        |
| POST | `/v1/responses/compact`                        | OpenAI    | Responses 上下文压缩                                               |
| POST | `/v1/alpha/search`                             | OpenAI    | Codex 独立网页搜索                                                 |
| POST | `/v1/completions`                              | OpenAI    | 旧版文本补全                                                       |
| POST | `/v1/messages`                                 | Anthropic | Claude Messages，见 [Anthropic Messages](/docs/anthropic-messages) |
| POST | `/v1beta/models/{model}:generateContent`       | Gemini    | 非流式生成，见 [Gemini 原生接口](/docs/gemini-native)              |
| POST | `/v1beta/models/{model}:streamGenerateContent` | Gemini    | 流式生成                                                           |

### 图片、视频与音频

| 方法 | 路径                       | 说明                                      |
| ---- | -------------------------- | ----------------------------------------- |
| POST | `/v1/images/generations`   | 图片生成，见 [图片生成](/docs/images)     |
| POST | `/v1/images/edits`         | 图片编辑，支持 JSON 和 multipart          |
| POST | `/v1/videos`               | 创建视频任务，见 [视频生成](/docs/videos) |
| GET  | `/v1/videos/{id}`          | 查询视频任务                              |
| GET  | `/v1/videos/{id}/content`  | 下载成片                                  |
| POST | `/v1/audio/speech`         | 语音合成                                  |
| POST | `/v1/audio/transcriptions` | 语音转文字                                |
| POST | `/v1/audio/translations`   | 语音翻译                                  |

### 模型与其他

| 方法 | 路径                 | 说明                   |
| ---- | -------------------- | ---------------------- |
| GET  | `/v1/models`         | 当前令牌可用的模型列表 |
| GET  | `/v1/models/{model}` | 查询单个模型           |
| GET  | `/v1beta/models`     | Gemini 格式的模型列表  |
| POST | `/v1/embeddings`     | 向量嵌入               |
| POST | `/v1/rerank`         | 重排序                 |
| POST | `/v1/moderations`    | 内容审核               |
| GET  | `/v1/realtime`       | Realtime WebSocket     |

- 接口是否可用取决于令牌所属分组是否包含对应类型的模型。各分组支持的端点类型见 [分组说明](/docs/groups)，例如 Claude 相关分组只支持 `/v1/messages`。
- `GET /v1/models` 默认返回 OpenAI 格式；同时带 `x-api-key` 和 `anthropic-version` 请求头时返回 Anthropic 格式，带 `x-goog-api-key` 请求头或 `?key=` 参数时返回 Gemini 格式。
- `/v1/files`、`/v1/fine-tunes`、`/v1/images/variations` 未实现，会返回 HTTP 501 和 `api_not_implemented`。
- `/v1/messages/count_tokens` 和 Gemini 的 `:countTokens` 不可用，会返回 HTTP 404。

## 流式输出

三种格式都支持流式输出，响应使用 SSE（`Content-Type: text/event-stream`），客户端需要逐行读取：

| 格式                    | 开启方式                                         | 结束标志                  |
| ----------------------- | ------------------------------------------------ | ------------------------- |
| OpenAI Chat Completions | 请求体 `"stream": true`                          | 最后一行 `data: [DONE]`   |
| OpenAI Responses        | 请求体 `"stream": true`                          | `response.completed` 事件 |
| Anthropic Messages      | 请求体 `"stream": true`                          | `message_stop` 事件       |
| Gemini                  | 调用 `:streamGenerateContent`，通常加 `?alt=sse` | 流结束                    |

<!-- code-group -->

```bash
curl -N {{BASE_URL}}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "stream": true,
    "messages": [{"role": "user", "content": "用一句话介绍你自己"}]
  }'
```

```python
from openai import OpenAI

client = OpenAI(base_url="{{BASE_URL}}/v1", api_key="sk-xxxxxxxx")
stream = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=[{"role": "user", "content": "用一句话介绍你自己"}],
    stream=True,
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

<!-- /code-group -->

> [!TIP]
> 编程工具和长回复建议始终开启流式输出：首个字到达得更快，也能避免长时间无数据导致客户端或代理超时断开。

## 限流与超时

- 管理员可以按时间窗口限制请求次数。超过限制时返回 HTTP 429，提示类似「您已达到请求数限制：N分钟内最多请求M次」，等待窗口过去后再请求即可。
- 上游号池或模型方繁忙时，也可能返回 429 或 5xx，这类错误与本站限流无关，稍后重试或换用其他分组。
- 长时间运行的生成请求请使用流式输出；视频生成是异步任务，创建后轮询任务状态，不要长时间挂起同一个连接。
- 网关检测到服务器负载过高时会暂时返回 HTTP 503，稍后重试即可。

各类错误的含义和处理方法见 [错误处理](/docs/errors)。

## 请求 ID

每个响应都带有响应头 `X-Oneapi-Request-Id`，网关返回的错误信息末尾也会附带 `(request id: ...)`。遇到问题时，请把请求 ID、请求时间和使用的模型一起反馈，便于在日志中定位。

用 curl 查看响应头：

```bash
curl -s -D - -o /dev/null {{BASE_URL}}/v1/models \
  -H "Authorization: Bearer sk-xxxxxxxx" | grep -i x-oneapi-request-id
```

> [!NOTE]
> 每次调用的模型、令牌、分组、用量和费用都可以在控制台 [使用日志](/usage-logs) 中查看，筛选条件里也可以直接按「请求 ID」查找。
