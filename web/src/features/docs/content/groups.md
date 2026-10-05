分组决定了令牌（API Key）可以调用哪些模型、走哪条上游线路，以及按什么倍率计费。每个令牌对应一个分组，按用途选对分组是用好 {{SITE_NAME}} 的第一步。

## 实时分组列表

下表实时展示所有可选分组的倍率、说明、包含的模型和最近 24 小时可用率。倍率和模型会随线路调整，请以此表为准。

<!-- widget:groups -->

> [!TIP]
> 倍率越低价格越低，但各分组的线路来源和稳定性不同。选择时建议同时参考分组说明和 24 小时可用率。

## 倍率与价格

分组倍率是价格系数，同一个模型在不同分组的价格不同：

```text
实际费用 = 模型基础价格 × 分组倍率
```

以一次基础价格为 1 美元的调用为例（数字仅用于说明计算方式）：

| 分组倍率 | 实际费用 |
| -------- | -------- |
| 0.1      | 0.1 美元 |
| 0.3      | 0.3 美元 |
| 1        | 1 美元   |

- 不需要自己计算：在 [模型广场](/pricing) 打开模型详情，「按分组定价」会列出该模型在各分组的价格。
- 每次调用使用的「分组倍率」会记录在 [使用日志](/usage-logs) 中；个别账户可能设有「专属倍率」，以日志显示为准。
- 按量、按次、按秒和按任务等计费方式的说明见 [计费说明](/docs/billing)。

## 按用途选择分组

| 用途                         | 推荐分组                                                                | 说明                                          |
| ---------------------------- | ----------------------------------------------------------------------- | --------------------------------------------- |
| Claude Code、Claude 系列模型 | `claude-kiro`、`claude-kiro正价`、`claude-max`                          | 只支持 Anthropic 格式                         |
| Codex、GPT 系列模型          | `codex-plus`、`codex-pro`、`codex-官key`                                | 支持 Responses、Chat Completions 等多种格式   |
| Gemini CLI、Gemini 系列模型  | `gemini`                                                                | 只支持 Gemini 原生格式                        |
| 国产模型                     | `DeepSeek`、`glm`、`kimi`、`minimax`、`qwen`、`豆包`、`hunyuan`、`mimo` | 都支持 OpenAI 格式，部分还支持 Anthropic 格式 |
| Grok 系列模型                | `grok`                                                                  | SuperGrokHeavy 号池                           |
| 图片生成                     | `生图`                                                                  | 支持 GPT、Gemini、Grok 生图模型               |
| 视频生成                     | `满血视频`、`特价视频`                                                  | 能力差异见 [视频生成](/docs/videos)           |
| 角色扮演、酒馆类聊天         | `酒馆`                                                                  | 按次计费，禁止用于 Vibe Code（AI 编程）       |

### 同类分组的区别

- Claude：`claude-kiro` 是低价线路（说明为"低价 kiro，风味 Claude"），`claude-kiro正价` 是正价 kiro 线路，`claude-max` 是外接 max 线路。
- Codex：`codex-plus`、`codex-pro` 分别来自 plus、pro 号池；`codex-官key` 是稳定高速分组，遇到缓存或降智问题可以直接反馈。
- 视频：`满血视频` 的模型能力对齐官方，支持首尾帧、参考图、参考视频和音频；`特价视频` 能力不全，主打便宜。

## 支持的调用格式

每个分组都标有"端点类型"，表示该分组的模型可以用哪些接口格式调用。

| 分组                                           | 端点类型                                                                                 |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `codex-plus`、`codex-pro`、`codex-官key`       | openai、openai-response、openai-response-compact、anthropic、gemini、openai-alpha-search |
| `claude-kiro`、`claude-kiro正价`、`claude-max` | anthropic                                                                                |
| `gemini`                                       | gemini                                                                                   |
| `DeepSeek`、`glm`、`grok`、`minimax`           | openai、openai-response、anthropic                                                       |
| `kimi`                                         | openai、openai-response、anthropic、gemini 等                                            |
| `qwen`、`豆包`、`hunyuan`、`mimo`、`酒馆`      | openai                                                                                   |
| `生图`                                         | image-generation                                                                         |
| `满血视频`、`特价视频`                         | openai-video                                                                             |

端点类型与接口的对应关系：

| 端点类型                | 接口                                          | 文档                                           |
| ----------------------- | --------------------------------------------- | ---------------------------------------------- |
| openai                  | `POST /v1/chat/completions`                   | [Chat Completions](/docs/chat-completions)     |
| openai-response         | `POST /v1/responses`                          | [Responses](/docs/responses)                   |
| openai-response-compact | `POST /v1/responses/compact`                  | [Responses](/docs/responses)                   |
| anthropic               | `POST /v1/messages`                           | [Anthropic Messages](/docs/anthropic-messages) |
| gemini                  | `POST /v1beta/models/{model}:generateContent` | [Gemini 原生接口](/docs/gemini-native)         |
| image-generation        | `POST /v1/images/generations`                 | [图片生成](/docs/images)                       |
| openai-video            | `POST /v1/videos`                             | [视频生成](/docs/videos)                       |

> [!IMPORTANT]
> 请用分组支持的格式调用。例如 Claude 相关分组只支持 anthropic，应通过 `/v1/messages` 调用（Claude Code 默认使用这种格式），用其他格式调用不保证可用。

## 分组与令牌

- 分组在创建令牌时选择，每个令牌对应一个分组，步骤见 [创建令牌](/docs/api-keys)。
- 需要同时使用多个分组时，为每个分组分别创建令牌，在不同客户端中填入对应的令牌。
- 用令牌调用 `GET /v1/models`，可以查看该令牌当前能用的模型，见 [模型列表](/docs/models)。
- 请求按令牌所属分组的倍率计费，计算方式见 [计费说明](/docs/billing)。

### 自动分组

创建令牌时，如果「分组」下拉框中有 `auto`（自动分组）选项，可以让令牌按顺序自动选择分组：

- 选择 `auto` 后会出现「自动分组顺序」，用来选择并排列这个令牌依次尝试的分组。
- 请求时按顺序使用第一个有该模型可用线路的分组，费用按实际使用分组的倍率计算。
- 开启「跨分组重试」后，当前分组的线路调用失败时，会按顺序改用下一个分组重试。
- 出错时，错误信息中的分组可能显示为 `auto` 或 `auto(实际尝试的分组)`。

`默认` 分组的说明为"默认分组（自动分组）"，倍率为 1。为了让价格和线路可控，建议按用途明确选择分组，或在「自动分组顺序」中只保留需要的分组。

## 常见问题

### 提示当前分组下没有可用渠道

通常是请求的模型不属于令牌所选的分组，或模型名拼写有误。请对照上方实时列表或 [模型广场](/pricing) 核对，模型名需要完全一致（区分大小写）。模型名确认无误时，可能是该分组的线路暂时不可用，可以稍后重试，或改用其他分组的令牌。完整的排查步骤见 [错误处理](/docs/errors)。

### 想换一个分组怎么办

新建一个所需分组的令牌，再把客户端中的令牌替换掉即可。各分组价格不同，切换前请在 [模型广场](/pricing) 确认价格。
