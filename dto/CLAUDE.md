[根目录](../CLAUDE.md) > **dto**

# dto — 请求/响应数据结构

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / dto

## 模块职责

定义客户端与上游之间的**数据传输对象（DTO）**：各协议的请求/响应结构，供 `relay/`、`controller/`、`service/` 解析与再序列化。

## 关键文件

| 协议/域 | 文件 |
| --- | --- |
| OpenAI | `openai_request.go`、`openai_response.go`、`openai_image.go`、`openai_video.go`、`openai_compaction.go`、`openai_responses_compaction_request.go` |
| Claude / Gemini | `claude.go`、`gemini.go` |
| 其他端点 | `embedding.go`、`rerank.go`、`audio.go`、`realtime.go`、`video.go`、`task.go`、`suno.go`、`midjourney.go` |
| 通用/配置 | `request_common.go`、`values.go`、`error.go`、`pricing.go`、`ratio_sync.go`、`notify.go`、`sensitive.go`、`channel_settings.go`、`user_settings.go`、`playground.go` |

## 核心约定：指针 + omitempty 保零值（务必遵守）

对**从客户端 JSON 解析、再 re-marshal 给上游**的请求结构：

- 可选标量字段 **必须用指针类型 + `omitempty`**（`*int`、`*uint`、`*float64`、`*bool`）。
- 语义：客户端缺省 → `nil`，marshal 时省略；客户端显式传 `0`/`0.0`/`false` → 非 `nil`，必须原样发上游。
- **禁止**非指针标量配 `omitempty`（零值会被静默丢弃，破坏「显式零值」语义）。
- 回归测试见 `dto/openai_request_zero_value_test.go`、`dto/gemini_generation_config_test.go`。

## 计费相关的边界常量

- 计费乘数上界常量在 dto 层定义（如 `dto.MaxImageN` 限制图片生成数量），供 validator 复用，避免溢出成负计费。新增带乘数的请求字段时，从第一天起就在校验器里设上界（配合 `relay/helper/valid_request.go`）。

## 关键依赖与约定

- JSON 收发走 `common.Marshal/Unmarshal`；`json.RawMessage`/`json.Number` 可作类型引用，但 marshal/unmarshal 必须走 `common.*`。

## 常见问题 (FAQ)

- **Q：客户端传 `temperature: 0` 被丢了？** A：字段用了非指针 + `omitempty`，改为 `*float64` + `omitempty`。
- **Q：新增可选参数怎么定义？** A：优先 `*T` + `omitempty`，并在对应 validator 设上界（若是计费乘数）。

## 相关文件清单

- OpenAI 请求：`dto/openai_request.go`
- 零值回归：`dto/openai_request_zero_value_test.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 dto 概览文档（指针 + omitempty 保零值规则）。
