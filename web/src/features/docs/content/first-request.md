本页带你用刚创建的令牌发出第一个请求：先查询令牌可用的模型，再调用一次对话接口，最后试试流式输出。示例覆盖 curl（Bash 和 Windows PowerShell）、Python 和 Node.js，任选一种即可。

## 准备工作

开始前请确认：

1. 已 [注册账户](/docs/register) 并登录控制台。
2. 账户有余额，余额不足时请求会被拒绝，充值方法见 [计费说明](/docs/billing)。
3. 已在 [API 密钥](/keys) 页面创建令牌，并在行操作菜单中点「复制密钥」拿到以 `sk-` 开头的密钥，步骤见 [创建令牌](/docs/api-keys)。
4. 知道令牌绑定的分组。示例使用 `DeepSeek` 分组的 `deepseek-v4-pro`，请换成你的令牌分组中的模型，分组与模型的对应关系见 [分组说明](/docs/groups)。

接口地址如下，示例中的 `{{BASE_URL}}` 就是本站地址：

<!-- widget:base-urls -->

> [!WARNING]
> 令牌等同于账户的付费凭证，不要写进前端代码、提交到公开仓库或发给他人。示例中的 `sk-xxxxxxxx` 需要替换成你自己的令牌。

## 检查环境

### curl

macOS、Linux 和 Windows 10/11 都自带 curl。打开终端执行：

<!-- code-group -->

```bash
curl --version
```

```powershell
curl.exe --version
```

<!-- /code-group -->

> [!IMPORTANT]
> 在 Windows PowerShell 中，`curl` 是 `Invoke-WebRequest` 的别名，参数格式完全不同。本站所有 curl 示例在 PowerShell 中都要写成 `curl.exe`。

### Python 与 Node.js

使用 SDK 时，先安装 OpenAI 官方 SDK。{{SITE_NAME}} 兼容 OpenAI 接口，只需把 Base URL 改成本站地址：

```bash
pip install openai     # Python
npm install openai     # Node.js
```

## 第一步：查询可用模型

先用令牌请求模型列表。能正常返回，就说明令牌、地址和网络都没有问题：

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/models \
  -H "Authorization: Bearer sk-xxxxxxxx"
```

```powershell
curl.exe {{BASE_URL}}/v1/models `
  -H "Authorization: Bearer sk-xxxxxxxx"
```

<!-- /code-group -->

返回结果的结构如下：

```json
{
  "success": true,
  "object": "list",
  "data": [
    {
      "id": "deepseek-v4-pro",
      "object": "model",
      "created": 1626777600,
      "owned_by": "...",
      "supported_endpoint_types": ["openai", "openai-response", "anthropic"]
    }
  ]
}
```

- `data` 里只包含令牌所属分组可用的模型，`id` 就是调用时 `model` 字段要填的值。
- `supported_endpoint_types` 表示该模型支持的接口格式，含义见 [分组说明](/docs/groups)。
- 令牌开启了「模型限制」时，列表只包含允许的模型。

## 第二步：发送对话请求

选一个列表中的模型，调用 Chat Completions 接口：

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "messages": [
      { "role": "user", "content": "你好，请用一句话介绍你自己" }
    ]
  }'
```

```powershell
$body = @'
{
  "model": "deepseek-v4-pro",
  "messages": [
    { "role": "user", "content": "你好，请用一句话介绍你自己" }
  ]
}
'@
[IO.File]::WriteAllText("$PWD\body.json", $body)
curl.exe {{BASE_URL}}/v1/chat/completions `
  -H "Authorization: Bearer sk-xxxxxxxx" `
  -H "Content-Type: application/json" `
  --data-binary "@body.json"
```

```python
from openai import OpenAI

client = OpenAI(
    base_url="{{BASE_URL}}/v1",
    api_key="sk-xxxxxxxx",
)

resp = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=[{"role": "user", "content": "你好，请用一句话介绍你自己"}],
)
print(resp.choices[0].message.content)
```

```javascript
import OpenAI from 'openai'

const client = new OpenAI({
  baseURL: '{{BASE_URL}}/v1',
  apiKey: 'sk-xxxxxxxx',
})

const resp = await client.chat.completions.create({
  model: 'deepseek-v4-pro',
  messages: [{ role: 'user', content: '你好，请用一句话介绍你自己' }],
})
console.log(resp.choices[0].message.content)
```

<!-- /code-group -->

- PowerShell 示例先把请求体写入 `body.json` 再发送，可以避开不同 PowerShell 版本对命令行引号的处理差异。
- Node.js 示例使用了顶层 `await`，请保存为 `hello.mjs` 后用 `node hello.mjs` 运行。
- 两个 SDK 都会读取环境变量 `OPENAI_API_KEY` 和 `OPENAI_BASE_URL`。设置这两个变量后，代码里可以不写令牌，更安全。

成功时返回类似下面的结果，回复内容在 `choices[0].message.content`，本次用量在 `usage`：

```json
{
  "id": "chatcmpl-xxxxxxxx",
  "object": "chat.completion",
  "model": "deepseek-v4-pro",
  "choices": [
    {
      "index": 0,
      "message": { "role": "assistant", "content": "你好！我是一个 AI 助手……" },
      "finish_reason": "stop"
    }
  ],
  "usage": { "prompt_tokens": 16, "completion_tokens": 24, "total_tokens": 40 }
}
```

## 第三步：流式输出

在请求体中加入 `"stream": true`，回复会边生成边返回，适合聊天界面和长文本。Python 和 Node.js 示例沿用上一步创建的 `client`：

<!-- code-group -->

```bash
curl -N {{BASE_URL}}/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "stream": true,
    "messages": [{ "role": "user", "content": "写一首关于秋天的短诗" }]
  }'
```

```powershell
$body = '{"model":"deepseek-v4-pro","stream":true,"messages":[{"role":"user","content":"写一首关于秋天的短诗"}]}'
[IO.File]::WriteAllText("$PWD\body.json", $body)
curl.exe -N {{BASE_URL}}/v1/chat/completions `
  -H "Authorization: Bearer sk-xxxxxxxx" `
  -H "Content-Type: application/json" `
  --data-binary "@body.json"
```

```python
stream = client.chat.completions.create(
    model="deepseek-v4-pro",
    messages=[{"role": "user", "content": "写一首关于秋天的短诗"}],
    stream=True,
)
for chunk in stream:
    if chunk.choices and chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

```javascript
const stream = await client.chat.completions.create({
  model: 'deepseek-v4-pro',
  messages: [{ role: 'user', content: '写一首关于秋天的短诗' }],
  stream: true,
})
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content ?? '')
}
```

<!-- /code-group -->

流式响应使用 SSE 格式，每行以 `data:` 开头，最后一行是 `data: [DONE]`：

```text
data: {"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"秋"}}]}

data: {"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"风"}}]}

data: [DONE]
```

流式请求同样按实际用量计费。每次调用的模型、用量和费用都可以在控制台 [使用日志](/usage-logs) 中查看。

## 常见问题排查

| 现象                                         | 原因                                               | 处理方法                                                              |
| -------------------------------------------- | -------------------------------------------------- | --------------------------------------------------------------------- |
| 401，提示 `无效的令牌`                       | 令牌填错、复制不完整，或令牌已禁用、过期、额度用完 | 到 [API 密钥](/keys) 页面检查令牌状态，重新复制完整的 `sk-` 密钥      |
| 返回一整段 HTML 网页                         | Base URL 漏了 `/v1`，请求发到了网站页面            | OpenAI 格式的地址要以 `/v1` 结尾                                      |
| 404，提示 `Invalid URL (POST /v1/v1/...)`    | Base URL 已带 `/v1`，路径又拼了一次                | 去掉重复的 `/v1`                                                      |
| 403，提示 `该令牌无权访问模型 xxx`           | 令牌开启了「模型限制」                             | 编辑令牌，在「模型限制」中加入该模型或清空限制                        |
| 403，提示 `用户额度不足` 或 `预扣费额度失败` | 账户余额不足                                       | 到 [钱包](/wallet) 充值                                               |
| 503，提示 `分组 xxx 下模型 xxx 无可用渠道`   | 模型不在令牌所属分组，或模型名拼写有误             | 对照第一步的模型列表修改 `model`，或换用对应分组的令牌                |
| PowerShell 报错中出现 `Invoke-WebRequest`    | 执行的是 `curl` 别名，不是真正的 curl              | 改用 `curl.exe`                                                       |
| 中文回复显示为乱码                           | 终端编码不是 UTF-8                                 | PowerShell 先执行 `[Console]::OutputEncoding = [Text.Encoding]::UTF8` |

更多错误码见 [错误处理](/docs/errors)。反馈问题时，请附上响应头中的 `X-Oneapi-Request-Id`。

## 下一步

- 接入编程工具：[CC Switch 一键导入](/docs/cc-switch)、[Claude Code](/docs/claude-code)、[Codex](/docs/codex)、[Gemini CLI](/docs/gemini-cli)。
- 接入聊天客户端：[聊天客户端](/docs/chat-clients)。
- 查看完整接口说明：[接口概览](/docs/api-overview)、[Chat Completions](/docs/chat-completions)。
