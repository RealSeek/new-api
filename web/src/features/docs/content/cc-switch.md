CC Switch 是一款开源的桌面工具，用来集中管理 Claude Code、Codex、Gemini CLI 的服务商配置，并可以一键切换。{{SITE_NAME}} 控制台支持把令牌直接导入 CC Switch，不用手动编辑配置文件。

## CC Switch 是什么

| 项目       | 说明                                                            |
| ---------- | --------------------------------------------------------------- |
| 开源地址   | [farion1231/cc-switch](https://github.com/farion1231/cc-switch) |
| 支持的工具 | Claude Code、Codex、Gemini CLI                                  |
| 作用       | 保存多个服务商（地址 + 密钥 + 模型），一键切换当前生效的配置    |
| 平台       | Windows、macOS、Linux 桌面端                                    |

CC Switch 本身不提供模型，它只负责把 {{SITE_NAME}} 的地址和令牌写进各个 CLI 的配置。CLI 工具仍需自行安装，见 [Claude Code](/docs/claude-code)、[Codex](/docs/codex)、[Gemini CLI](/docs/gemini-cli)。

## 安装

1. 打开 [CC Switch 的 Releases 页面](https://github.com/farion1231/cc-switch/releases)。
2. 按系统下载安装包（Windows、macOS、Linux 各有对应格式），安装后启动一次。
3. 控制台的一键导入通过 `ccswitch://` 链接唤起 CC Switch，请使用支持链接导入的较新版本。

> [!NOTE]
> 安装包格式、包管理器安装方式（如 Homebrew）和界面文字会随版本变化，以客户端官方文档为准。

## 准备令牌

一键导入前，先在 [API 密钥](/keys) 页面准备好对应分组的令牌，创建步骤见 [创建令牌](/docs/api-keys)。

| 要导入的应用          | 推荐分组                                                                                                                    |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Claude（Claude Code） | `claude-kiro`、`claude-kiro正价`、`claude-max`，或支持 anthropic 的 `DeepSeek`、`glm`、`kimi`、`minimax`、`grok`、`codex-*` |
| Codex                 | `codex-plus`、`codex-pro`、`codex-官key`                                                                                    |
| Gemini（Gemini CLI）  | `gemini`                                                                                                                    |

> [!IMPORTANT]
> 每个令牌只对应一个分组。Claude、Codex、Gemini 通常需要不同分组，建议分别创建令牌，例如分别命名为 `claude-code`、`codex`、`gemini-cli`。

## 一键导入（推荐）

1. 登录控制台，打开左侧菜单中的「API 密钥」（[/keys](/keys)）。
2. 在令牌列表中找到要导入的令牌，点击该行「操作」列里的「⋯」按钮（打开菜单）。
3. 在菜单中点击「CC Switch」，弹出「填入 CC Switch」对话框。
4. 按下表填写：

| 字段                               | 说明                                                                                                                       |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| 应用                               | 选择 Claude、Codex 或 Gemini。切换应用时，名称和模型会重置                                                                 |
| 名称                               | 服务商在 CC Switch 中显示的名称，默认是 My Claude、My Codex、My Gemini，可以改成 `{{SITE_NAME}}-Claude` 这类便于区分的名字 |
| 主模型                             | 必填。点击后从下拉列表中选择，可以输入关键字筛选                                                                           |
| Haiku 模型、Sonnet 模型、Opus 模型 | 只在选择 Claude 时显示，选填。用于把 Claude Code 的 Haiku、Sonnet、Opus 三个档位分别指定为某个模型                         |

5. 点击「打开 CC Switch」。浏览器询问是否打开 CC Switch 时，选择允许。
6. 按 CC Switch 的提示确认导入，新服务商会出现在对应应用的列表中。

> [!WARNING]
> 主模型的下拉列表包含你账户下所有可用分组的模型，并不只是这个令牌所属的分组。请选择令牌分组内的模型，否则调用时会提示模型不存在或无可用渠道。可以在 [模型广场](/pricing) 按分组筛选确认。

### 导入的内容

| 应用   | 接口地址          | 密钥     | 模型                                        |
| ------ | ----------------- | -------- | ------------------------------------------- |
| Claude | `{{BASE_URL}}`    | 当前令牌 | 主模型，以及填写的 Haiku、Sonnet、Opus 模型 |
| Codex  | `{{BASE_URL}}/v1` | 当前令牌 | 主模型                                      |
| Gemini | `{{BASE_URL}}`    | 当前令牌 | 主模型                                      |

点击按钮后，控制台会打开一个 `ccswitch://` 链接，解码后的格式如下：

```text
ccswitch://v1/import?resource=provider&app=claude&name=My Claude&endpoint={{BASE_URL}}&apiKey=sk-xxxxxxxx&model=claude-sonnet-4-6&homepage={{BASE_URL}}&enabled=true
```

`app` 的取值为 `claude`、`codex` 或 `gemini`。选择 Claude 时，链接还会带上填写的 `haikuModel`、`sonnetModel`、`opusModel`。

> [!CAUTION]
> 链接里包含完整的令牌，不要截图或转发给他人。

令牌菜单的「聊天」子菜单和左侧「聊天」菜单里也可能出现 CC Switch，但那里打开的链接不包含令牌和模型。导入请使用令牌菜单中单独的「CC Switch」项。

## 手动添加

一键导入无法唤起 CC Switch，或者想自己控制配置内容时，可以在 CC Switch 中手动添加服务商。

1. 打开 CC Switch，切换到要配置的应用（Claude、Codex 或 Gemini）。
2. 点击添加服务商，选择自定义配置。
3. 按下表填写后保存。

| 应用   | 接口地址          | API Key       | 模型示例            |
| ------ | ----------------- | ------------- | ------------------- |
| Claude | `{{BASE_URL}}`    | `sk-xxxxxxxx` | `claude-sonnet-4-6` |
| Codex  | `{{BASE_URL}}/v1` | `sk-xxxxxxxx` | `gpt-5.5`           |
| Gemini | `{{BASE_URL}}`    | `sk-xxxxxxxx` | `gemini-3.1-pro`    |

Codex 需要使用 Responses 接口（`wire_api = "responses"`）。需要直接编辑配置内容时，可以参考 [Claude Code](/docs/claude-code)、[Codex](/docs/codex)、[Gemini CLI](/docs/gemini-cli) 页面中的完整示例。

> [!NOTE]
> 按钮名称和表单字段会随 CC Switch 版本变化，以客户端官方文档为准。

## 切换服务商

1. 在 CC Switch 中选择应用。
2. 在服务商列表中启用 {{SITE_NAME}} 对应的条目，CC Switch 会改写该应用的配置文件。
3. 重新打开终端，或重启正在运行的 CLI 会话，让新配置生效。

CC Switch 通常写入以下位置，排查问题时可以打开查看：

| 应用        | 配置文件                                       |
| ----------- | ---------------------------------------------- |
| Claude Code | `~/.claude/settings.json` 中的 `env` 字段      |
| Codex       | `~/.codex/config.toml` 和 `~/.codex/auth.json` |
| Gemini CLI  | `~/.gemini/.env`                               |

Windows 下 `~` 对应 `%USERPROFILE%`。具体写入方式以客户端官方文档为准。

## 验证

1. 在终端启动对应的 CLI（`claude`、`codex` 或 `gemini`），发送一句简单的问题，例如“请回复 OK”。
2. 打开 [使用日志](/usage-logs)，确认出现这次调用，令牌名称、分组和模型与预期一致。
3. 在 Claude Code 中可以输入 `/status` 查看当前配置（显示内容以客户端官方文档为准）。

## 常见问题

### 点击「打开 CC Switch」没有反应

- 确认已安装 CC Switch，并且至少启动过一次。
- 浏览器可能拦截了外部应用链接，请在弹窗中选择允许，或换一个浏览器重试。
- 仍然无法唤起时，按上文“手动添加”配置。

### 提示「请选择主模型」

主模型是必填项。点击主模型输入框，从下拉列表中选择一个模型，再点击「打开 CC Switch」。

### 下拉列表显示「未找到模型」

模型列表可能还在加载，稍等片刻或关闭后重新打开对话框。仍然为空时，请到 [模型广场](/pricing) 确认账户可用的分组和模型。

### 调用时提示无效的令牌

令牌可能已被删除、禁用、过期或额度用尽。到 [API 密钥](/keys) 检查该令牌的状态，必要时重新导入。

### 提示分组下不存在模型或无可用渠道

错误信息类似“分组 xxx 下不存在模型 xxx”或“分组 xxx 下模型 xxx 无可用渠道”。说明所选模型不在令牌的分组里，或该分组的线路暂时不可用。请改选令牌分组内的模型，或换用对应分组的令牌，见 [分组说明](/docs/groups)。

### 接口地址填错

Codex 的接口地址必须以 `/v1` 结尾，Claude 和 Gemini 不要加 `/v1`。地址填错时，客户端通常会报 404（错误信息类似 `Invalid URL (POST /v1/v1/messages)`），或无法解析响应。一键导入会自动处理，手动添加时请对照上文表格检查。

### 切换后仍在使用旧配置

- 重启终端和 CLI，已经打开的会话不会自动读取新配置。
- 检查系统环境变量或 shell 配置文件中是否手动设置过 `ANTHROPIC_BASE_URL`、`OPENAI_BASE_URL`、`GEMINI_API_KEY` 等变量。它们可能与 CC Switch 写入的配置冲突，保留一处即可。

遇到其他问题时，反馈时请附上响应头中的 `X-Oneapi-Request-Id`，排查方法见 [错误处理](/docs/errors)。
