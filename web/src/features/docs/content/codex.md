Codex 是 OpenAI 推出的命令行编程助手（Codex CLI），通过 Responses 接口（`POST /v1/responses`）调用模型。在 `~/.codex/config.toml` 中添加一个指向 {{SITE_NAME}} 的自定义服务商，就能在终端里使用 codex 分组的 GPT 模型。

> [!TIP]
> 不想手动改配置，可以用 [CC Switch](/docs/cc-switch) 从控制台一键导入。

> [!NOTE]
> Codex 更新较快，配置项的名称和默认值可能随版本变化。本文以常见版本为例，以 [官方文档](https://github.com/openai/codex) 为准。

## 准备令牌

在 [API 密钥](/keys) 页面创建令牌，分组选择 codex 分组之一，步骤见 [创建令牌](/docs/api-keys)。

| 分组          | 倍率 | 说明                                       |
| ------------- | ---- | ------------------------------------------ |
| `codex-plus`  | 0.13 | plus 号池                                  |
| `codex-pro`   | 0.2  | pro 号池                                   |
| `codex-官key` | 0.4  | 稳定高速分组，遇到缓存和降智问题请直接反馈 |

常用模型有 `gpt-5.5`、`gpt-5.6-sol`、`gpt-6-sol`、`gpt-6.1-sol`、`gpt-6-astra`、`codex-auto-review`。模型和倍率会随线路调整，以 [模型广场](/pricing) 和 [分组说明](/docs/groups) 为准。

codex 分组支持以下接口：

| 接口                                                              | 用途                                            |
| ----------------------------------------------------------------- | ----------------------------------------------- |
| `POST /v1/responses`                                              | Codex 的主要接口，对应 `wire_api = "responses"` |
| `POST /v1/responses/compact`                                      | 上下文压缩接口，Codex 压缩长对话时可能会调用    |
| `POST /v1/alpha/search`                                           | Codex 的独立网页搜索接口                        |
| `POST /v1/chat/completions`、`POST /v1/messages`、Gemini 原生接口 | 同一个令牌也可以给其他客户端使用                |

## 安装

需要先安装 Node.js（版本要求以官方文档为准），然后全局安装 Codex：

```bash
npm install -g @openai/codex
codex --version
```

- 升级到最新版：`npm install -g @openai/codex@latest`。
- macOS 也可以用 Homebrew 安装（`brew install --cask codex`，以官方文档为准）。
- Windows 可以直接安装，原生支持情况以官方文档为准，也可以在 WSL 中安装使用。

## 配置

Codex 的配置保存在以下目录，配置文件是其中的 `config.toml`，不存在就新建：

| 系统         | 目录                    |
| ------------ | ----------------------- |
| macOS、Linux | `~/.codex/`             |
| Windows      | `%USERPROFILE%\.codex\` |

设置了 `CODEX_HOME` 环境变量时，以该变量指向的目录为准。下面两种方式选一种即可。

### 方式一：config.toml + 环境变量（推荐）

在 `config.toml` 中添加自定义服务商，令牌从环境变量读取：

```toml
model = "gpt-5.5"
model_provider = "onlycode"
model_reasoning_effort = "medium"

[model_providers.onlycode]
name = "{{SITE_NAME}}"
base_url = "{{BASE_URL}}/v1"
env_key = "ONLYCODE_API_KEY"
wire_api = "responses"
```

| 配置项           | 说明                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------ |
| `model_provider` | 当前使用的服务商 ID，要与 `[model_providers.onlycode]` 中的 `onlycode` 一致。ID 可以自定义 |
| `base_url`       | 必须以 `/v1` 结尾，Codex 会自动拼接 `/responses`                                           |
| `env_key`        | 存放令牌的环境变量名，令牌以 `Authorization: Bearer` 请求头发送                            |
| `wire_api`       | 填 `responses`，请求走 `/v1/responses`                                                     |

> [!IMPORTANT]
> `model`、`model_provider` 等顶层配置必须写在所有 `[...]` 表头之前。写在 `[model_providers.onlycode]` 下面会被当成服务商的字段，导致配置不生效或报错。

然后设置环境变量，下面的写法只对当前终端生效：

<!-- code-group -->

```bash
export ONLYCODE_API_KEY="sk-xxxxxxxx"
codex
```

```powershell
$env:ONLYCODE_API_KEY = "sk-xxxxxxxx"
codex
```

<!-- /code-group -->

长期使用时，写入 shell 配置文件（bash 用户把 `~/.zshrc` 换成 `~/.bashrc`）或 Windows 用户环境变量，写入后重新打开终端：

<!-- code-group -->

```bash
echo 'export ONLYCODE_API_KEY="sk-xxxxxxxx"' >> ~/.zshrc
source ~/.zshrc
```

```powershell
[Environment]::SetEnvironmentVariable("ONLYCODE_API_KEY", "sk-xxxxxxxx", "User")
```

<!-- /code-group -->

### 方式二：auth.json + requires_openai_auth

这也是 CC Switch 写入的方式，不需要设置环境变量。`config.toml` 中的服务商改为：

```toml
model = "gpt-5.5"
model_provider = "onlycode"
model_reasoning_effort = "medium"

[model_providers.onlycode]
name = "{{SITE_NAME}}"
base_url = "{{BASE_URL}}/v1"
wire_api = "responses"
requires_openai_auth = true
```

在同一目录的 `auth.json` 中写入令牌：

```json
{
  "OPENAI_API_KEY": "sk-xxxxxxxx"
}
```

`requires_openai_auth = true` 表示该服务商使用 Codex 自己保存的认证信息，也就是 `auth.json` 中的 `OPENAI_API_KEY`。`auth.json` 里如果已有 ChatGPT 登录信息，覆盖前请先备份。该字段的行为以官方文档为准。

## 选择模型与推理强度

| 方式         | 用法                                     |
| ------------ | ---------------------------------------- |
| 默认模型     | `config.toml` 顶层的 `model`             |
| 启动时指定   | `codex -m gpt-6-sol`                     |
| 会话中切换   | 输入 `/model`，选择模型和推理强度        |
| 临时覆盖配置 | `codex -c model_reasoning_effort="high"` |

`model_reasoning_effort` 控制推理强度，常用 `low`、`medium`、`high`，部分版本和模型还支持其他取值，以官方文档为准。强度越高，通常消耗的 token 越多，响应也越慢，日常编码用 `medium` 即可。

模型名必须是令牌分组内的模型，并且完全一致（区分大小写），以 [模型广场](/pricing) 为准。

> [!NOTE]
> `DeepSeek`、`glm`、`grok`、`kimi`、`minimax` 分组也支持 Responses 接口，但没有 openai-response-compact 端点类型。Codex 调用 `/v1/responses/compact` 压缩上下文时，这些分组的请求可能失败。Codex 推荐使用 codex 分组。

## 验证

1. 重新打开终端，在项目目录运行 `codex`。
2. 输入 `/status`，确认模型和服务商是刚才配置的值（显示内容以官方文档为准）。
3. 发送一句测试问题，例如“请回复 OK”。也可以用非交互模式：`codex exec "Reply with OK"`。
4. 打开 [使用日志](/usage-logs)，确认这次调用的令牌、分组和模型与预期一致。

Codex 报错时，可以先绕过客户端，直接请求 Responses 接口，确认令牌和模型可用：

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/responses \
  -H "Authorization: Bearer sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.5","input":"Reply with OK"}'
```

```powershell
$body = @{ model = "gpt-5.5"; input = "Reply with OK" } | ConvertTo-Json
Invoke-RestMethod -Uri "{{BASE_URL}}/v1/responses" -Method Post `
  -Headers @{ Authorization = "Bearer sk-xxxxxxxx" } `
  -ContentType "application/json" -Body $body
```

<!-- /code-group -->

## 常见问题

### Selected model is at capacity. Please try a different model

通常是 gpt 官方算力不足导致的，只能等待官方恢复，稍后再试。

### 提示缺少环境变量或 401 无效的令牌

- `env_key` 中的变量名要与实际设置的环境变量完全一致。
- 用 `setx` 或 `SetEnvironmentVariable` 写入后，要重新打开终端。在 VS Code 等编辑器的内置终端中使用时，需要重启编辑器。
- 使用方式二时，检查 `auth.json` 是否为合法 JSON，令牌是否完整（以 `sk-` 开头）。
- 令牌被删除、禁用、过期或额度用尽时也会返回 401，到 [API 密钥](/keys) 检查状态。

### 返回 404 或无法解析响应

`base_url` 必须是 `{{BASE_URL}}/v1`。漏掉 `/v1` 时，请求会打到网页而不是接口，客户端通常报无法解析响应；多写一个 `/v1` 会返回 404（`Invalid URL (POST /v1/v1/responses)`）。`wire_api` 请保持 `responses`。

### 提示分组下不存在模型或无可用渠道

模型不在令牌的分组里，或模型名拼写有误。检查 `model` 和 `/model` 中选择的模型，确认令牌绑定的是 codex 分组。模型名确认无误时，可能是线路暂时不可用，稍后重试或换用其他 codex 分组。

### 403：额度不足

“用户额度不足”表示账户余额不足，到 [钱包](/wallet) 充值。`token quota is not enough` 表示令牌剩余额度不够本次请求，到 [API 密钥](/keys) 调高额度或开启无限配额。

### 配置没有生效

- 确认修改的是正确目录下的 `config.toml`（Windows 为 `%USERPROFILE%\.codex\config.toml`，或 `CODEX_HOME` 指向的目录）。
- 确认 `model`、`model_provider` 写在所有 `[...]` 表头之前。
- 使用 CC Switch 时，切换服务商会改写这些文件，手动修改可能被覆盖。

### 遇到缓存或降智问题

使用 `codex-官key` 分组时，如果遇到缓存或回答质量明显下降，请直接反馈。反馈时附上 [使用日志](/usage-logs) 中的时间和模型，以及响应头中的 `X-Oneapi-Request-Id`，排查方法见 [错误处理](/docs/errors)。
