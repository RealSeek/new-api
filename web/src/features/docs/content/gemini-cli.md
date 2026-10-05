Gemini CLI 是 Google 推出的命令行 AI 助手，通过 Gemini 原生接口（`/v1beta/models/{model}:generateContent`）调用模型。设置令牌和接口地址两个变量，就能让 Gemini CLI 使用 {{SITE_NAME}} 的 `gemini` 分组。

> [!TIP]
> 不想手动改配置，可以用 [CC Switch](/docs/cc-switch) 从控制台一键导入。

## 准备令牌

在 [API 密钥](/keys) 页面创建令牌，分组选择 `gemini`，步骤见 [创建令牌](/docs/api-keys)。

| 分组     | 倍率 | 说明                  | 模型示例                                                                     |
| -------- | ---- | --------------------- | ---------------------------------------------------------------------------- |
| `gemini` | 0.3  | Gemini 反重力逆向号池 | `gemini-3.1-pro`、`gemini-3.8-flash`、`gemini-4-preview`、`gemini-2.5-flash` |

模型和倍率会随线路调整，以 [模型广场](/pricing) 和 [分组说明](/docs/groups) 为准。

> [!NOTE]
> `codex-*`、`kimi` 分组也支持 Gemini 原生格式，但模型要改成该分组内的模型（例如 `gpt-5.5`、`kimi-k3`）。Gemini CLI 针对 Gemini 模型设计，推荐使用 `gemini` 分组。

## 安装

Gemini CLI 需要 Node.js 20 或更高版本（以官方文档为准）：

```bash
node -v
npm install -g @google/gemini-cli
gemini --version
```

也可以不安装、直接运行 `npx @google/gemini-cli`，macOS 还可以用 `brew install gemini-cli`，以官方文档为准。

## 配置

Gemini CLI 需要以下变量：

| 变量                     | 说明                                                                                 |
| ------------------------ | ------------------------------------------------------------------------------------ |
| `GEMINI_API_KEY`         | 令牌 `sk-xxxxxxxx`，以 `x-goog-api-key` 请求头发送                                   |
| `GOOGLE_GEMINI_BASE_URL` | 接口地址，填 `{{BASE_URL}}`，不要加 `/v1beta`。客户端会自动拼接 `/v1beta/models/...` |
| `GEMINI_MODEL`           | 默认模型，可选，建议设置为 `gemini` 分组内的模型                                     |

下面两种方式选一种即可。

### 方式一：.env 文件（推荐）

Gemini CLI 启动时会自动加载 `.env` 文件。用户级文件放在以下位置，项目目录下的 `.gemini/.env` 只对该项目生效：

| 系统         | 路径                         |
| ------------ | ---------------------------- |
| macOS、Linux | `~/.gemini/.env`             |
| Windows      | `%USERPROFILE%\.gemini\.env` |

文件内容：

```text
GEMINI_API_KEY=sk-xxxxxxxx
GOOGLE_GEMINI_BASE_URL={{BASE_URL}}
GEMINI_MODEL=gemini-3.1-pro
```

也可以用命令创建。下面的命令会覆盖已有的 `.env`，原有内容请手动合并：

<!-- code-group -->

```bash
mkdir -p ~/.gemini
cat > ~/.gemini/.env <<'EOF'
GEMINI_API_KEY=sk-xxxxxxxx
GOOGLE_GEMINI_BASE_URL={{BASE_URL}}
GEMINI_MODEL=gemini-3.1-pro
EOF
```

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.gemini" | Out-Null
@"
GEMINI_API_KEY=sk-xxxxxxxx
GOOGLE_GEMINI_BASE_URL={{BASE_URL}}
GEMINI_MODEL=gemini-3.1-pro
"@ | Set-Content -Encoding ascii "$env:USERPROFILE\.gemini\.env"
```

<!-- /code-group -->

`.env` 文件的查找顺序以官方文档为准。

### 方式二：环境变量

只在当前终端窗口生效，适合临时测试：

<!-- code-group -->

```bash
export GEMINI_API_KEY="sk-xxxxxxxx"
export GOOGLE_GEMINI_BASE_URL="{{BASE_URL}}"
export GEMINI_MODEL="gemini-3.1-pro"
gemini
```

```powershell
$env:GEMINI_API_KEY = "sk-xxxxxxxx"
$env:GOOGLE_GEMINI_BASE_URL = "{{BASE_URL}}"
$env:GEMINI_MODEL = "gemini-3.1-pro"
gemini
```

<!-- /code-group -->

长期使用时，写入 shell 配置文件（bash 用户把 `~/.zshrc` 换成 `~/.bashrc`）或 Windows 用户环境变量，写入后重新打开终端：

<!-- code-group -->

```bash
cat >> ~/.zshrc <<'EOF'
export GEMINI_API_KEY="sk-xxxxxxxx"
export GOOGLE_GEMINI_BASE_URL="{{BASE_URL}}"
EOF
source ~/.zshrc
```

```powershell
[Environment]::SetEnvironmentVariable("GEMINI_API_KEY", "sk-xxxxxxxx", "User")
[Environment]::SetEnvironmentVariable("GOOGLE_GEMINI_BASE_URL", "{{BASE_URL}}", "User")
```

<!-- /code-group -->

## 选择 API Key 认证

Gemini CLI 支持 Google 账号登录、Gemini API Key、Vertex AI 等认证方式。接入 {{SITE_NAME}} 时需要选择 Gemini API Key：

1. 配置好上面的变量后，在终端运行 `gemini`。
2. 首次启动出现认证方式选择时，选择使用 Gemini API Key（选项文字以客户端显示为准）。
3. 之前选过 Google 账号登录的，在会话中输入 `/auth` 重新选择。

所选的认证方式会保存在 `~/.gemini/settings.json`（Windows 为 `%USERPROFILE%\.gemini\settings.json`）中，字段名随版本变化，以官方文档为准。

> [!WARNING]
> 选择 Google 账号登录或 Vertex AI 时，Gemini CLI 不会使用 {{SITE_NAME}} 的令牌和地址，调用也不会出现在 [使用日志](/usage-logs) 中。

## 选择模型

| 方式       | 用法                                       |
| ---------- | ------------------------------------------ |
| 默认模型   | 设置 `GEMINI_MODEL`                        |
| 启动时指定 | `gemini -m gemini-3.1-pro`                 |
| 会话中切换 | 部分版本支持 `/model` 命令，以官方文档为准 |

模型名必须是 `gemini` 分组内的模型，并且完全一致（区分大小写），以 [模型广场](/pricing) 为准。不指定模型时，Gemini CLI 会使用自带的默认模型，该模型不在分组中时会报错。

## 验证

1. 运行 `gemini`，发送一句测试问题，例如“请回复 OK”。也可以用非交互模式：`gemini -p "Reply with OK"`。
2. 打开 [使用日志](/usage-logs)，确认这次调用的令牌、分组和模型与预期一致。

Gemini CLI 报错时，可以先绕过客户端，直接请求 Gemini 原生接口，确认令牌和模型可用：

<!-- code-group -->

```bash
curl "{{BASE_URL}}/v1beta/models/gemini-3.1-pro:generateContent" \
  -H "x-goog-api-key: sk-xxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"contents":[{"parts":[{"text":"Reply with OK"}]}]}'
```

```powershell
$body = @{ contents = @(@{ parts = @(@{ text = "Reply with OK" }) }) } | ConvertTo-Json -Depth 6
Invoke-RestMethod -Uri "{{BASE_URL}}/v1beta/models/gemini-3.1-pro:generateContent" -Method Post `
  -Headers @{ "x-goog-api-key" = "sk-xxxxxxxx" } `
  -ContentType "application/json" -Body $body
```

<!-- /code-group -->

## 常见问题

### 仍然要求登录 Google 账号

当前没有选择 Gemini API Key 认证。在会话中输入 `/auth` 改选 Gemini API Key，并确认 `GEMINI_API_KEY` 已经生效（写在 `.env` 中，或已重新打开终端）。

### 报 API key not valid，使用日志里也没有记录

请求没有发到 {{SITE_NAME}}，而是直接发到了 Google 官方接口。请检查：

- `GOOGLE_GEMINI_BASE_URL` 是否设置成功。修改环境变量后需要重新打开终端。
- 是否还设置了 `GOOGLE_API_KEY`、`GOOGLE_GENAI_USE_VERTEXAI`、`GOOGLE_CLOUD_PROJECT` 等变量。它们可能让 Gemini CLI 改用其他认证方式，不需要时可以删除（以官方文档为准）。

### 401：无效的令牌

令牌填错、已删除、已禁用、已过期，或额度已用尽。到 [API 密钥](/keys) 检查状态，重新复制完整的 `sk-` 令牌。

### 返回 404（Invalid URL）

`GOOGLE_GEMINI_BASE_URL` 多写了 `/v1beta` 等路径时，请求会变成 `/v1beta/v1beta/models/...` 并返回 404。改回 `{{BASE_URL}}` 即可。

### countTokens 返回 404

{{SITE_NAME}} 暂未开放 `:countTokens` 接口，该请求会返回 404。客户端统计 token 数等功能可能因此报错，通常不影响正常对话。

### 提示分组下不存在模型或无可用渠道

错误信息类似“分组 xxx 下不存在模型 xxx”。常见原因是没有设置 `GEMINI_MODEL`，客户端用了自带的默认模型。用 `-m` 或 `GEMINI_MODEL` 指定 `gemini` 分组内的模型。模型名确认无误时，可能是线路暂时不可用，稍后重试。

### 403：额度不足

“用户额度不足”表示账户余额不足，到 [钱包](/wallet) 充值。`token quota is not enough` 表示令牌剩余额度不够本次请求，到 [API 密钥](/keys) 调高额度或开启无限配额。

### 偶发 429 或 5xx

通常是上游限流或线路波动，稍后重试即可。持续出现时，请记下响应头中的 `X-Oneapi-Request-Id` 并反馈，排查方法见 [错误处理](/docs/errors)。
