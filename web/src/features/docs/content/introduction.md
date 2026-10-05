{{SITE_NAME}} 是一个 AI 模型接口聚合平台。注册账户并创建一个令牌，就能用 OpenAI、Anthropic、Gemini 兼容的接口调用 GPT、Claude、Gemini、DeepSeek 等模型，也可以直接接入 Claude Code、Codex、Gemini CLI 等编程工具，或生成图片和视频。

## 一个令牌，多种协议

同一个令牌可以调用多种格式的接口，具体能用哪些格式取决于令牌所属的分组：

| 接口格式             | Base URL          | 主要接口                                     | 适用场景                                        |
| -------------------- | ----------------- | -------------------------------------------- | ----------------------------------------------- |
| OpenAI 兼容          | `{{BASE_URL}}/v1` | `/v1/chat/completions`、`/v1/responses`      | OpenAI SDK、Codex、Cherry Studio 等大多数客户端 |
| Anthropic 兼容       | `{{BASE_URL}}`    | `/v1/messages`                               | Anthropic SDK、Claude Code                      |
| Gemini 原生          | `{{BASE_URL}}`    | `/v1beta/models/{model}:generateContent`     | Google Gen AI SDK、Gemini CLI                   |
| 图片生成             | `{{BASE_URL}}/v1` | `/v1/images/generations`、`/v1/images/edits` | 生图工具和自定义调用                            |
| 视频生成（video-v1） | `{{BASE_URL}}/v1` | `/v1/videos`                                 | 创建、查询、下载视频任务                        |

各格式的 Base URL 可以在下方直接复制：

<!-- widget:base-urls -->

> [!IMPORTANT]
> OpenAI 格式的地址需要以 `/v1` 结尾；Anthropic 和 Gemini 格式直接填写站点地址，客户端会自动拼接后面的路径。地址填错是最常见的接入问题。

## 快速开始

1. 注册：在 [注册页](/sign-up) 用邮箱注册并完成验证，见 [注册账户](/docs/register)。
2. 充值：在 [钱包](/wallet) 页面充值余额，见 [计费说明](/docs/billing)。
3. 创建令牌：在 [API 密钥](/keys) 页面点击「创建 API 密钥」，按用途选择分组，见 [创建令牌](/docs/api-keys)。
4. 配置客户端：把 Base URL 和令牌填入客户端，见 [CC Switch](/docs/cc-switch)、[Claude Code](/docs/claude-code)、[Codex](/docs/codex)、[Gemini CLI](/docs/gemini-cli) 或 [聊天客户端](/docs/chat-clients)。

> [!TIP]
> 配置客户端前，可以先按 [发起第一个请求](/docs/first-request) 用 curl 测试令牌是否可用。

## 按场景选择

| 场景                     | 推荐分组                                                       | 接口格式     | 参考文档                         |
| ------------------------ | -------------------------------------------------------------- | ------------ | -------------------------------- |
| Claude Code、Claude 模型 | `claude-kiro`、`claude-kiro正价`、`claude-max`                 | Anthropic    | [Claude Code](/docs/claude-code) |
| Codex、GPT 模型          | `codex-plus`、`codex-pro`、`codex-官key`                       | Responses    | [Codex](/docs/codex)             |
| Gemini CLI、Gemini 模型  | `gemini`                                                       | Gemini 原生  | [Gemini CLI](/docs/gemini-cli)   |
| 日常对话、聊天客户端     | `DeepSeek`、`kimi`、`glm`、`qwen` 等国产模型分组，或 `codex-*` | OpenAI       | [聊天客户端](/docs/chat-clients) |
| 角色扮演、酒馆类聊天     | `酒馆`（按次计费，禁止用于 AI 编程）                           | OpenAI       | [聊天客户端](/docs/chat-clients) |
| 图片生成                 | `生图`                                                         | 图片接口     | [图片生成](/docs/images)         |
| 视频生成                 | `满血视频`、`特价视频`                                         | video-v1     | [视频生成](/docs/videos)         |
| 网页版对话与创作         | 登录后按引导配置                                               | 无需手动配置 | [AI 工作台](/docs/workbench)     |

各分组的倍率、模型和可用率见 [分组说明](/docs/groups)，模型价格见 [模型广场](/pricing)。

## 基本概念

| 概念            | 说明                                                              |
| --------------- | ----------------------------------------------------------------- |
| 令牌（API Key） | 以 `sk-` 开头的密钥，调用接口时放在请求头中；每个令牌对应一个分组 |
| 分组            | 决定可用的模型、上游线路和价格倍率                                |
| 倍率            | 分组的价格系数，实际费用 = 模型基础价格 × 分组倍率                |
| 余额            | 账户的可用额度，以美元显示，调用时按实际用量扣减                  |

> [!TIP]
> 遇到问题时，请记下响应头中的请求 ID `X-Oneapi-Request-Id`，反馈时一并提供。更多内容见 [常见问题](/docs/faq) 和 [错误处理](/docs/errors)。

网关基于 new-api，通用配置可参考 [new-api 官方文档](https://docs.newapi.pro)。本站线路与工作台的使用方法以本文档为准。
