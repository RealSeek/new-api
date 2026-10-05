Cherry Studio、Lobe Chat、SillyTavern 等聊天客户端大多支持 OpenAI 兼容接口，填入 {{SITE_NAME}} 的接口地址和令牌就能使用。部分客户端还可以从控制台一键导入，不用手动填写。

## 选择分组

令牌的分组决定能用哪些模型，也决定客户端要用哪种接口格式。在 [API 密钥](/keys) 页面按用途创建令牌，步骤见 [创建令牌](/docs/api-keys)。

| 用途                                   | 推荐分组                                                                | 客户端使用的格式 |
| -------------------------------------- | ----------------------------------------------------------------------- | ---------------- |
| 日常对话、写作、翻译                   | `DeepSeek`、`kimi`、`qwen`、`glm`、`豆包`、`minimax`、`hunyuan`、`mimo` | OpenAI           |
| GPT 模型                               | `codex-plus`、`codex-pro`、`codex-官key`                                | OpenAI           |
| Grok 模型                              | `grok`                                                                  | OpenAI           |
| Claude 模型                            | `claude-kiro`、`claude-kiro正价`、`claude-max`                          | 仅 Anthropic     |
| Gemini 模型                            | `gemini`                                                                | 仅 Gemini 原生   |
| 角色扮演（SillyTavern 等酒馆类客户端） | `酒馆`                                                                  | OpenAI，按次计费 |

倍率和模型以 [分组说明](/docs/groups) 的实时列表和 [模型广场](/pricing) 为准。

> [!WARNING]
> `酒馆` 分组按次计费，每次请求按固定价格扣费，适合角色扮演类聊天。该分组禁止用于 Vibe Code（AI 编程），不要在 Claude Code、Codex 等编程工具中使用。分组内很多模型名以 `[c]` 开头，填写时要带上前缀。

> [!IMPORTANT]
> Claude 分组只支持 Anthropic 格式，`gemini` 分组只支持 Gemini 原生格式。在客户端中添加服务商时要选择对应的服务商类型，用 OpenAI 类型调用这些分组不保证可用。

## 接口地址

| 接口格式    | 接口地址                                    | 客户端中的服务商类型        |
| ----------- | ------------------------------------------- | --------------------------- |
| OpenAI 兼容 | `{{BASE_URL}}/v1` 或 `{{BASE_URL}}`，见下表 | OpenAI、OpenAI 兼容、自定义 |
| Anthropic   | `{{BASE_URL}}`                              | Anthropic、Claude           |
| Gemini      | `{{BASE_URL}}`                              | Gemini、Google              |

### OpenAI 地址要不要加 /v1

不同客户端拼接路径的方式不同。简单的判断方法是：把地址输入框默认值或示例中的官方域名（如 `https://api.openai.com`）换成 `{{BASE_URL}}`，后面的路径保持不变。

| 地址输入框的默认值或示例                       | 填写                               | 常见客户端                         |
| ---------------------------------------------- | ---------------------------------- | ---------------------------------- |
| `https://api.openai.com`（客户端自动补 `/v1`） | `{{BASE_URL}}`                     | Cherry Studio、NextChat            |
| `https://api.openai.com/v1`                    | `{{BASE_URL}}/v1`                  | Lobe Chat、SillyTavern、OpenAI SDK |
| 要求填写完整请求地址                           | `{{BASE_URL}}/v1/chat/completions` | 少数插件和脚本                     |

填错时的表现：

- 报 404，错误信息中出现 `/v1/v1/`：多写了 `/v1`，去掉即可。
- 返回网页内容，或提示无法解析响应：少了 `/v1`，补上即可。

表中常见客户端的填法基于常用版本，以客户端官方文档为准。

## 从控制台一键导入

控制台的「聊天」菜单列出了管理员配置的客户端，目前包括 Cherry Studio、Lobe Chat 和 CC Switch（以控制台显示为准）。有两个入口：

| 入口                                       | 使用的令牌                   | 说明                                 |
| ------------------------------------------ | ---------------------------- | ------------------------------------ |
| [API 密钥](/keys) → 令牌行的「⋯」→「聊天」 | 当前这一行的令牌             | 推荐，可以明确导入的是哪个分组的令牌 |
| 左侧菜单「聊天」                           | 令牌列表中第一个已启用的令牌 | 网页版客户端会在控制台页面内打开     |

带外链图标的是桌面客户端，点击后浏览器会询问是否打开对应应用，选择允许即可。CC Switch 请使用令牌菜单中单独的「CC Switch」项，见 [CC Switch](/docs/cc-switch)。

> [!CAUTION]
> 一键导入的链接中包含完整的令牌，不要复制或分享给他人。

### Cherry Studio

[Cherry Studio](https://cherry-ai.com) 是一款开源的桌面 AI 客户端，支持 Windows、macOS、Linux。

1. 安装并启动 Cherry Studio。
2. 在 [API 密钥](/keys) 中找到要使用的令牌，点击「⋯」→「聊天」→「Cherry Studio」。
3. 浏览器询问是否打开 Cherry Studio 时，选择允许。
4. Cherry Studio 弹出添加服务商的确认窗口，确认后会自动填入接口地址 `{{BASE_URL}}` 和令牌。
5. 在该服务商中添加要使用的模型（见下文“添加模型”），之后就可以在对话中选择。

导入的地址不带 `/v1`，由 Cherry Studio 自动补全。导入的服务商 ID 为 `new-api`，通常对应 Cherry Studio 内置的 New API 服务商，确认窗口和服务商名称以客户端显示为准。

### Lobe Chat

点击「聊天」中的 Lobe Chat（名称可能显示为「Lobe Chat 官方示例」），会打开 Lobe Chat 网页版，并通过链接把 OpenAI 的接口地址（`{{BASE_URL}}/v1`）和令牌写入它的设置。打开后选择 OpenAI 服务商下、令牌分组内的模型即可。实际打开的网址由管理员配置，以控制台为准。

## 手动配置 Cherry Studio

1. 打开 Cherry Studio 的「设置」→「模型服务」，点击「添加」。
2. 填写名称（例如 {{SITE_NAME}}），提供商类型按下表选择。
3. 「API 密钥」填 `sk-xxxxxxxx`，「API 地址」填 `{{BASE_URL}}`。
4. 添加模型，然后点击 API 密钥旁边的「检查」测试连接，并打开服务商开关。

| 要使用的分组                                                  | 提供商类型 | API 地址       |
| ------------------------------------------------------------- | ---------- | -------------- |
| OpenAI 格式的分组（`DeepSeek`、`kimi`、`qwen`、`codex-*` 等） | OpenAI     | `{{BASE_URL}}` |
| Claude 分组                                                   | Anthropic  | `{{BASE_URL}}` |
| `gemini` 分组                                                 | Gemini     | `{{BASE_URL}}` |

Cherry Studio 的 API 地址以 `/` 结尾时不会自动补 `/v1`，以 `#` 结尾时会直接使用填写的完整地址。按钮名称和这些规则以客户端官方文档为准。

### 添加模型

在服务商的模型区域点击「管理」，可以从接口获取模型列表（调用 `GET /v1/models`），勾选需要的模型即可。列表只包含令牌所属分组的模型。也可以点击「添加」手动填写模型 ID，名称必须与 [模型广场](/pricing) 中的完全一致，`[c]` 前缀也要带上。

## 其他 OpenAI 兼容客户端

大多数客户端都有“OpenAI 兼容”或“自定义”服务商，按下表填写即可：

| 客户端中的字段                           | 填写内容                                              |
| ---------------------------------------- | ----------------------------------------------------- |
| 服务商类型                               | OpenAI、OpenAI 兼容或自定义（Custom）                 |
| API 地址（Base URL、Endpoint、API Host） | `{{BASE_URL}}/v1` 或 `{{BASE_URL}}`，判断方法见上文   |
| API Key                                  | `sk-xxxxxxxx`                                         |
| 模型                                     | 令牌分组内的模型名，例如 `deepseek-v4-pro`、`kimi-k3` |

客户端的“获取模型列表”功能会调用 `GET /v1/models`，返回结果只包含令牌所属分组的模型。网页版客户端可以直接在浏览器中调用 {{SITE_NAME}} 的接口，接口已允许跨域请求。

## Anthropic、Gemini 格式的客户端

使用 Claude 分组或 `gemini` 分组时，在客户端中选择对应类型的服务商：

| 服务商类型          | 接口地址       | 令牌的发送方式                                | 模型示例            |
| ------------------- | -------------- | --------------------------------------------- | ------------------- |
| Anthropic（Claude） | `{{BASE_URL}}` | `x-api-key` 或 `Authorization: Bearer` 请求头 | `claude-sonnet-4-6` |
| Gemini（Google）    | `{{BASE_URL}}` | `x-goog-api-key` 请求头或 `?key=` 参数        | `gemini-3.1-pro`    |

客户端会自动拼接 `/v1/messages` 或 `/v1beta/models/...`，地址中不要带这些路径。示例地址本身带有路径时，同样只替换域名部分。获取模型列表时，Anthropic 客户端调用 `GET /v1/models`（带 `x-api-key` 和 `anthropic-version` 请求头），Gemini 客户端调用 `GET /v1beta/models`，都只返回令牌分组内的模型。接口细节见 [Anthropic Messages](/docs/anthropic-messages) 和 [Gemini 原生接口](/docs/gemini-native)。

## 酒馆类客户端（SillyTavern 等）

角色扮演类客户端建议使用 `酒馆` 分组的令牌。以 SillyTavern 为例：

1. 打开 API 连接设置，API 类型选择 Chat Completion。
2. 聊天补全来源选择 Custom (OpenAI-compatible)。
3. 自定义端点（Base URL）填 `{{BASE_URL}}/v1`，API Key 填 `酒馆` 分组的令牌 `sk-xxxxxxxx`。
4. 点击连接，从模型列表中选择以 `[c]` 开头的模型，或手动输入模型 ID。

界面名称随版本变化，以 SillyTavern 官方文档为准。`酒馆` 分组按次计费，客户端每发起一次请求就扣费一次，重新生成、续写等操作也会各算一次。

## 生图

图片生成使用 `生图` 分组的令牌，模型按次计费。除了在支持生图的客户端中调用 [图片生成接口](/docs/images)，也可以使用配套的生图站点 https://image.realseek.wiki/ 。站点已经预先配置好接口地址，粘贴令牌（建议使用 `生图` 分组）即可使用。

## 验证

1. 在客户端中选择令牌分组内的模型，发送一句测试问题，例如“请回复 OK”。
2. 打开 [使用日志](/usage-logs)，确认这次调用的令牌、分组和模型与预期一致。

客户端报错时，可以先确认令牌能列出模型：

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/models \
  -H "Authorization: Bearer sk-xxxxxxxx"
```

```powershell
Invoke-RestMethod -Uri "{{BASE_URL}}/v1/models" `
  -Headers @{ Authorization = "Bearer sk-xxxxxxxx" }
```

<!-- /code-group -->

返回的 `data` 列表就是这个令牌可以使用的模型，其中的 `id` 就是客户端中要填写的模型名。

## 常见问题

### 一键导入后客户端没有反应

- 确认已安装对应的桌面客户端，并且至少启动过一次。
- 浏览器可能拦截了外部应用链接，在弹窗中选择允许，或换一个浏览器重试。
- 仍然不行时，按上文手动配置。

### 提示没有可用的令牌或无效的聊天链接

- 提示 `No enabled API keys found` 或「当前没有可用的启用令牌」：先在 [API 密钥](/keys) 创建或启用一个令牌。
- 提示无效的聊天链接：管理员配置的链接有误，请联系管理员。

### 401：无效的令牌

令牌填错、已删除、已禁用、已过期，或额度已用尽。到 [API 密钥](/keys) 检查状态，重新复制完整的 `sk-` 令牌，注意不要带空格。

### 404 或无法解析响应

接口地址多写或漏写了 `/v1`，按上文“OpenAI 地址要不要加 /v1”调整。

### 模型列表为空，或提示分组下不存在模型

- 令牌所属分组里没有这个模型。到 [模型广场](/pricing) 确认模型所属分组，换用对应分组的令牌。
- 模型名要完全一致（区分大小写），`[c]` 前缀不能省略。
- 使用 Claude 或 `gemini` 分组时，确认服务商类型选的是 Anthropic 或 Gemini。

### 403：额度不足

“用户额度不足”表示账户余额不足，到 [钱包](/wallet) 充值。`token quota is not enough` 表示令牌剩余额度不够本次请求，到 [API 密钥](/keys) 调高额度或开启无限配额。

### 酒馆分组扣费比预期多

`酒馆` 分组按次计费，与 token 数量无关。重新生成、续写，以及客户端在后台自动发起的请求都会单独计费，每次请求的扣费可以在 [使用日志](/usage-logs) 中查看。

反馈问题时，请提供响应头中的 `X-Oneapi-Request-Id`，排查方法见 [错误处理](/docs/errors)。
