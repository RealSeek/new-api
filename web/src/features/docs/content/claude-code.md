Claude Code 是 Anthropic 推出的命令行编程助手，通过 Anthropic Messages 接口（`POST /v1/messages`）调用模型。把接口地址和令牌指向 {{SITE_NAME}}，就能在终端里使用 Claude 以及其他支持 Anthropic 格式的模型。

> [!TIP]
> 不想手动改配置，可以用 [CC Switch](/docs/cc-switch) 从控制台一键导入。

## 准备令牌

在 [API 密钥](/keys) 页面创建令牌，分组选择支持 Anthropic 格式的分组，步骤见 [创建令牌](/docs/api-keys)。

| 分组                                           | 倍率           | 模型示例                                                            |
| ---------------------------------------------- | -------------- | ------------------------------------------------------------------- |
| `claude-kiro`、`claude-kiro正价`、`claude-max` | 0.1、0.35、1   | `claude-sonnet-4-6`、`claude-opus-5-5`、`claude-haiku-4-5-20251001` |
| `DeepSeek`                                     | 0.3            | `deepseek-v4-pro`、`deepseek-v4.1-flash`                            |
| `glm`                                          | 0.2            | `glm-5.3`                                                           |
| `kimi`                                         | 0.25           | `kimi-k3`                                                           |
| `minimax`                                      | 0.3            | `MiniMax-M3`                                                        |
| `grok`                                         | 0.22           | `grok-4.7`                                                          |
| `codex-plus`、`codex-pro`、`codex-官key`       | 0.13、0.2、0.4 | `gpt-5.5`                                                           |

三个 Claude 分组的区别：`claude-kiro` 是低价线路（低价 kiro，风味 Claude），`claude-kiro正价` 是正价 kiro，`claude-max` 是外接 max 线路。倍率和模型会随线路调整，以 [分组说明](/docs/groups) 的实时列表和 [模型广场](/pricing) 为准。使用非 Claude 分组时，需要把模型改成该分组内的模型，见下文“选择模型”。

## 安装

Claude Code 需要 Node.js 18 或更高版本。先确认版本，再用 npm 全局安装（macOS、Linux、Windows 通用）：

```bash
node -v
npm install -g @anthropic-ai/claude-code
claude --version
```

也可以使用官方的原生安装脚本，命令以官方文档为准：

<!-- code-group -->

```bash
curl -fsSL https://claude.ai/install.sh | bash
```

```powershell
irm https://claude.ai/install.ps1 | iex
```

<!-- /code-group -->

> [!NOTE]
> 不要使用 `sudo npm install -g`。遇到权限错误时，按 npm 文档调整全局安装目录，或改用原生安装脚本。

## 配置

二选一即可：

| 方式                          | 适用场景                                    |
| ----------------------------- | ------------------------------------------- |
| `settings.json` 的 `env` 字段 | 推荐，只对 Claude Code 生效，不影响其他程序 |
| 系统或 shell 环境变量         | 临时测试，或在脚本、CI 中注入               |

### 方式一：settings.json（推荐）

编辑用户级配置文件，不存在就新建：

| 系统         | 路径                                  |
| ------------ | ------------------------------------- |
| macOS、Linux | `~/.claude/settings.json`             |
| Windows      | `%USERPROFILE%\.claude\settings.json` |

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "{{BASE_URL}}",
    "ANTHROPIC_AUTH_TOKEN": "sk-xxxxxxxx",
    "ANTHROPIC_MODEL": "claude-sonnet-4-6",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "API_TIMEOUT_MS": "3000000"
  }
}
```

- `env` 里的值都写成带英文双引号的字符串。
- 文件里已有其他配置时，只把 `env` 字段合并进去，不要整份覆盖。
- 保存后重新启动 `claude` 生效。

### 方式二：环境变量

只在当前终端窗口生效，适合临时测试：

<!-- code-group -->

```bash
export ANTHROPIC_BASE_URL="{{BASE_URL}}"
export ANTHROPIC_AUTH_TOKEN="sk-xxxxxxxx"
export ANTHROPIC_MODEL="claude-sonnet-4-6"
claude
```

```powershell
$env:ANTHROPIC_BASE_URL = "{{BASE_URL}}"
$env:ANTHROPIC_AUTH_TOKEN = "sk-xxxxxxxx"
$env:ANTHROPIC_MODEL = "claude-sonnet-4-6"
claude
```

<!-- /code-group -->

长期使用时，macOS、Linux 写入 shell 配置文件（zsh 为 `~/.zshrc`，bash 为 `~/.bashrc`），Windows 写入用户环境变量：

<!-- code-group -->

```bash
cat >> ~/.zshrc <<'EOF'
export ANTHROPIC_BASE_URL="{{BASE_URL}}"
export ANTHROPIC_AUTH_TOKEN="sk-xxxxxxxx"
EOF
source ~/.zshrc
```

```powershell
[Environment]::SetEnvironmentVariable("ANTHROPIC_BASE_URL", "{{BASE_URL}}", "User")
[Environment]::SetEnvironmentVariable("ANTHROPIC_AUTH_TOKEN", "sk-xxxxxxxx", "User")
# 或者使用 setx（同样写入用户环境变量）
setx ANTHROPIC_BASE_URL "{{BASE_URL}}"
setx ANTHROPIC_AUTH_TOKEN "sk-xxxxxxxx"
```

<!-- /code-group -->

写入用户环境变量后，要重新打开终端才会生效。

### 变量说明

| 变量                                                                                              | 说明                                                                             |
| ------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ANTHROPIC_BASE_URL`                                                                              | 接口地址，填 `{{BASE_URL}}`，不要加 `/v1`。Claude Code 会自动拼接 `/v1/messages` |
| `ANTHROPIC_AUTH_TOKEN`                                                                            | 令牌，以 `Authorization: Bearer sk-...` 请求头发送                               |
| `ANTHROPIC_API_KEY`                                                                               | 令牌的另一种写法，以 `x-api-key` 请求头发送                                      |
| `ANTHROPIC_MODEL`                                                                                 | 默认使用的模型                                                                   |
| `ANTHROPIC_DEFAULT_OPUS_MODEL`、`ANTHROPIC_DEFAULT_SONNET_MODEL`、`ANTHROPIC_DEFAULT_HAIKU_MODEL` | 把 Opus、Sonnet、Haiku 三个档位分别映射到具体模型                                |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`                                                        | 设为 `1` 时关闭自动更新、遥测、错误上报等非必要流量，可选                        |
| `API_TIMEOUT_MS`                                                                                  | 单次请求的超时时间（毫秒），长任务可以适当调大，可选                             |

> [!IMPORTANT]
> `ANTHROPIC_AUTH_TOKEN` 和 `ANTHROPIC_API_KEY` 只设置一个即可，{{SITE_NAME}} 两种请求头都支持。同时设置时 Claude Code 可能提示认证冲突。使用 `ANTHROPIC_API_KEY` 时，首次启动可能会询问是否使用该密钥，选择同意即可（以官方文档为准）。

## 选择模型

| 方式       | 用法                                                                                            |
| ---------- | ----------------------------------------------------------------------------------------------- |
| 默认模型   | 设置 `ANTHROPIC_MODEL`                                                                          |
| 启动时指定 | `claude --model claude-opus-5-5`                                                                |
| 会话中切换 | 输入 `/model`，选择档位或输入模型名称                                                           |
| 档位映射   | 设置三个 `ANTHROPIC_DEFAULT_*_MODEL` 变量，`/model` 中的 Opus、Sonnet、Haiku 档位会使用对应模型 |

Claude Code 的部分后台任务会调用 Haiku 档位的模型。使用 Claude 分组时，可以按下面的方式映射：

| 变量                             | 示例值                      |
| -------------------------------- | --------------------------- |
| `ANTHROPIC_DEFAULT_OPUS_MODEL`   | `claude-opus-5-5`           |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | `claude-sonnet-4-6`         |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL`  | `claude-haiku-4-5-20251001` |

### 使用非 Claude 分组

令牌绑定 `DeepSeek`、`glm`、`kimi`、`minimax`、`grok` 或 `codex-*` 分组时，默认模型和三个档位都要改成该分组内的模型。否则 Claude Code 会请求分组中不存在的 Claude 模型并报错。以 `DeepSeek` 分组为例：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "{{BASE_URL}}",
    "ANTHROPIC_AUTH_TOKEN": "sk-xxxxxxxx",
    "ANTHROPIC_MODEL": "deepseek-v4-pro",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "deepseek-v4-pro",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-pro",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "deepseek-v4.1-flash"
  }
}
```

其他分组同理，模型名以 [模型广场](/pricing) 为准，需要完全一致（区分大小写）。

## 验证

1. 在项目目录运行 `claude`，输入 `/status`，确认显示的接口地址是 `{{BASE_URL}}`（显示内容以官方文档为准）。
2. 发送一句测试问题，例如“请回复 OK”，能正常回复即配置成功。
3. 打开 [使用日志](/usage-logs)，确认这次调用的令牌、分组和模型与预期一致。

Claude Code 报错时，可以先绕过客户端，直接请求接口确认令牌和模型可用：

<!-- code-group -->

```bash
curl {{BASE_URL}}/v1/messages \
  -H "x-api-key: sk-xxxxxxxx" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-sonnet-4-6","max_tokens":64,"messages":[{"role":"user","content":"Reply with OK"}]}'
```

```powershell
$body = @{
  model = "claude-sonnet-4-6"
  max_tokens = 64
  messages = @(@{ role = "user"; content = "Reply with OK" })
} | ConvertTo-Json -Depth 5
Invoke-RestMethod -Uri "{{BASE_URL}}/v1/messages" -Method Post `
  -Headers @{ "x-api-key" = "sk-xxxxxxxx"; "anthropic-version" = "2023-06-01" } `
  -ContentType "application/json" -Body $body
```

<!-- /code-group -->

## 常见问题

### 401：无效的令牌

令牌填错、已删除、已禁用、已过期，或额度已用尽。到 [API 密钥](/keys) 检查状态，重新复制完整的 `sk-` 令牌。同时检查是否在多处设置了令牌（`settings.json`、shell 配置文件、系统环境变量），旧值可能覆盖了新值。

### 提示分组下不存在模型或无可用渠道

错误信息类似“分组 xxx 下不存在模型 xxx”。说明请求的模型不在令牌的分组里，常见于非 Claude 分组没有改模型，或 Haiku 档位仍指向 Claude 模型。按上文“使用非 Claude 分组”设置全部模型变量。模型名确认无误时，可能是线路暂时不可用，稍后重试或换用其他分组。

### 403：额度不足或无权访问

- “用户额度不足”：账户余额不足，到 [钱包](/wallet) 充值。
- `token quota is not enough`：令牌的剩余额度不够本次请求，到 [API 密钥](/keys) 调高额度或开启无限配额。
- “该令牌无权访问模型 xxx”：令牌设置了模型限制，到 [API 密钥](/keys) 编辑令牌的模型限制。
- “您的 IP 不在令牌允许访问的列表中”：令牌设置了 IP 白名单，需要加入当前 IP。

### 启动后要求登录 Anthropic 账号

配置了 `ANTHROPIC_AUTH_TOKEN` 或 `ANTHROPIC_API_KEY` 后，Claude Code 一般不再要求登录。如果首次启动仍停在登录引导，可以在 `~/.claude.json`（Windows 为 `%USERPROFILE%\.claude.json`）中加入 `"hasCompletedOnboarding": true` 后重启。该字段属于客户端内部配置，以官方文档为准。

### 地址填成了 /v1

`ANTHROPIC_BASE_URL` 加了 `/v1` 时，请求会变成 `/v1/v1/messages`，返回 404（`Invalid URL`）。改回 `{{BASE_URL}}` 即可。

### count_tokens 返回 404

{{SITE_NAME}} 暂未开放 `POST /v1/messages/count_tokens`，该请求会返回 404。正常对话走 `/v1/messages`，一般不受影响。

### 长任务超时

调大 `API_TIMEOUT_MS`，或把大任务拆成几步执行。偶发的上游错误可以稍后重试。

反馈问题时，请提供响应头中的 `X-Oneapi-Request-Id`，排查方法见 [错误处理](/docs/errors)。
