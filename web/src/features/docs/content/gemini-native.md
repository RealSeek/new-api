Gemini 原生接口兼容 Google Gemini API 的请求格式，适合 Google Gen AI SDK、Gemini CLI 等原生客户端。模型名写在 URL 路径中，请求体使用 `contents`、`generationConfig` 等 Gemini 字段。

## 接口地址

| 方法 | 路径                                           | 说明                             |
| ---- | ---------------------------------------------- | -------------------------------- |
| POST | `/v1beta/models/{model}:generateContent`       | 生成内容，一次返回完整结果       |
| POST | `/v1beta/models/{model}:streamGenerateContent` | 流式生成，建议加 `?alt=sse`      |
| POST | `/v1beta/models/{model}:embedContent`          | 文本向量，分组提供向量模型时可用 |
| POST | `/v1beta/models/{model}:batchEmbedContents`    | 批量文本向量                     |
| GET  | `/v1beta/models`                               | Gemini 格式的模型列表            |

- Base URL 填 `{{BASE_URL}}`，不要加 `/v1beta`，SDK 会自动拼接路径。
- 把 `{model}` 换成令牌分组内的模型名，例如 `/v1beta/models/gemini-3.1-pro:generateContent`。
- `/v1/models/{model}:generateContent` 这类以 `/v1` 开头的路径也会按 Gemini 格式处理。
- `:countTokens` 暂不支持，会返回 HTTP 404。

## 认证

以下三种方式任选其一：

| 方式     | 写法                                | 说明                                                      |
| -------- | ----------------------------------- | --------------------------------------------------------- |
| 请求头   | `x-goog-api-key: sk-xxxxxxxx`       | 推荐，Google Gen AI SDK 默认使用这种方式                  |
| URL 参数 | `?key=sk-xxxxxxxx`                  | 令牌会出现在 URL 中，可能被代理和日志记录，非必要不要使用 |
| Bearer   | `Authorization: Bearer sk-xxxxxxxx` | 与其他接口相同                                            |

令牌在 [API 密钥](/keys) 页面创建，详见 [创建令牌](/docs/api-keys)。

## 支持的分组与模型

| 分组                                     | 说明                                              | 示例模型                                                                     |
| ---------------------------------------- | ------------------------------------------------- | ---------------------------------------------------------------------------- |
| `gemini`                                 | Gemini 反重力逆向号池，只支持 Gemini 原生格式     | `gemini-3.1-pro`、`gemini-3.8-flash`、`gemini-4-preview`、`gemini-2.5-flash` |
| `codex-plus`、`codex-pro`、`codex-官key` | 端点类型包含 gemini，分组内的模型也能用本格式调用 | 见 [分组说明](/docs/groups)                                                  |
| `kimi`                                   | 端点类型包含 gemini                               | `kimi-k3`                                                                    |

> [!IMPORTANT]
> `gemini` 分组只支持 Gemini 原生格式，不能用 `/v1/chat/completions` 等 OpenAI 格式调用。各分组实际可用的模型以 [分组说明](/docs/groups) 的实时列表和 [模型广场](/pricing) 为准。

## 请求示例

<!-- code-group -->

```bash
curl "{{BASE_URL}}/v1beta/models/gemini-3.1-pro:generateContent" \
  -H "x-goog-api-key: sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      { "role": "user", "parts": [{ "text": "用一句话介绍你自己" }] }
    ]
  }'
```

```powershell
$body = '{"contents":[{"role":"user","parts":[{"text":"用一句话介绍你自己"}]}]}'
[IO.File]::WriteAllText("$PWD\body.json", $body)
curl.exe "{{BASE_URL}}/v1beta/models/gemini-3.1-pro:generateContent" `
  -H "x-goog-api-key: sk-xxxxxxxx" `
  -H "Content-Type: application/json" `
  --data-binary "@body.json"
```

```python
from google import genai
from google.genai import types

client = genai.Client(
    api_key="sk-xxxxxxxx",
    http_options=types.HttpOptions(base_url="{{BASE_URL}}"),
)

response = client.models.generate_content(
    model="gemini-3.1-pro",
    contents="用一句话介绍你自己",
)
print(response.text)
```

```javascript
import { GoogleGenAI } from '@google/genai'

const ai = new GoogleGenAI({
  apiKey: 'sk-xxxxxxxx',
  httpOptions: { baseUrl: '{{BASE_URL}}' },
})

const response = await ai.models.generateContent({
  model: 'gemini-3.1-pro',
  contents: '用一句话介绍你自己',
})
console.log(response.text)
```

<!-- /code-group -->

- SDK 安装：Python 用 `pip install google-genai`，Node.js 用 `npm install @google/genai`。SDK 的参数名以官方文档为准。
- PowerShell 示例先把请求体写入 UTF-8 文件再发送，避免中文乱码。

## 响应示例

```json
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [{ "text": "我是一个 AI 助手……" }]
      },
      "finishReason": "STOP",
      "index": 0
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 8,
    "candidatesTokenCount": 20,
    "totalTokenCount": 28
  }
}
```

生成的文本在 `candidates[0].content.parts[].text` 中，用量在 `usageMetadata` 中，按 tokens 和分组倍率计费。

## 流式输出

把方法换成 `:streamGenerateContent` 并加上 `?alt=sse`，响应以 SSE 格式返回，每行 `data:` 是一个与上面结构相同的响应片段：

<!-- code-group -->

```bash
curl -N "{{BASE_URL}}/v1beta/models/gemini-3.8-flash:streamGenerateContent?alt=sse" \
  -H "x-goog-api-key: sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      { "role": "user", "parts": [{ "text": "写一首关于秋天的短诗" }] }
    ]
  }'
```

```powershell
$body = '{"contents":[{"role":"user","parts":[{"text":"写一首关于秋天的短诗"}]}]}'
[IO.File]::WriteAllText("$PWD\body.json", $body)
curl.exe -N "{{BASE_URL}}/v1beta/models/gemini-3.8-flash:streamGenerateContent?alt=sse" `
  -H "x-goog-api-key: sk-xxxxxxxx" `
  -H "Content-Type: application/json" `
  --data-binary "@body.json"
```

```python
for chunk in client.models.generate_content_stream(
    model="gemini-3.8-flash",
    contents="写一首关于秋天的短诗",
):
    print(chunk.text, end="", flush=True)
```

```javascript
const stream = await ai.models.generateContentStream({
  model: 'gemini-3.8-flash',
  contents: '写一首关于秋天的短诗',
})
for await (const chunk of stream) {
  process.stdout.write(chunk.text ?? '')
}
```

<!-- /code-group -->

Python 和 Node.js 示例沿用上一节创建的 `client` 和 `ai`。

## 主要参数

| 字段                  | 说明                                                                                                                   |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `contents`            | 必填。对话内容数组，每项包含 `role`（`user` 或 `model`）和 `parts`                                                     |
| `systemInstruction`   | 系统指令，也接受 `system_instruction` 写法                                                                             |
| `generationConfig`    | 生成参数，常用 `temperature`、`topP`、`topK`、`maxOutputTokens`、`stopSequences`、`responseMimeType`、`thinkingConfig` |
| `safetySettings`      | 安全过滤设置                                                                                                           |
| `tools`、`toolConfig` | 函数调用等工具配置                                                                                                     |
| `cachedContent`       | 上下文缓存的名称                                                                                                       |

字段的完整取值以 Google Gemini API 官方文档为准，模型不支持的参数可能被上游拒绝。

## 常见错误

| 现象                                             | 原因                                               | 处理方法                                              |
| ------------------------------------------------ | -------------------------------------------------- | ----------------------------------------------------- |
| 404 `Invalid URL`                                | Base URL 多写了 `/v1beta`，或调用了 `:countTokens` | Base URL 改为 `{{BASE_URL}}`；`:countTokens` 暂不支持 |
| 401 `无效的令牌`                                 | 没有携带令牌，或令牌已禁用、过期                   | 检查 `x-goog-api-key` 请求头或 `?key=` 参数           |
| 503 `分组 xxx 下模型 xxx 无可用渠道`             | URL 中的模型不在令牌分组内，或线路暂时不可用       | 用 `GET /v1beta/models` 核对模型名                    |
| 用 `/v1/chat/completions` 调用 `gemini` 分组失败 | `gemini` 分组只支持原生格式                        | 改用本页的 `/v1beta/models/...` 接口                  |

本接口的网关错误以 `{"error": {"message": "...", "type": "...", "code": "..."}}` 结构返回，错误信息末尾附有请求 ID。完整排查方法见 [错误处理](/docs/errors)，Gemini CLI 的配置见 [Gemini CLI](/docs/gemini-cli)。
