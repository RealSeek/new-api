[根目录](../CLAUDE.md) > **middleware**

# middleware — 认证 / 限流 / 分发

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / middleware

## 模块职责

Gin 中间件层，处理**认证、限流、渠道分发、CORS、日志、i18n、恢复**等横切关注点，在请求进入 controller/relay 前完成上下文准备与拦截。

## 关键中间件

| 分类 | 文件 | 职责 |
| --- | --- | --- |
| 认证 | `auth.go` | JWT / API Token / session 鉴权，写入 user/role/token 到 `gin.Context` |
| ⭐ 分发 | `distributor.go` | **中继前置核心**：按模型/分组选择上游 channel，写入 `channel_type`/`apiType` 等到 context，供 `relay/` 使用 |
| 限流 | `rate-limit.go`、`model-rate-limit.go`、`email-verification-rate-limit.go` | 全局/按模型/按邮件验证限流 |
| 安全 | `secure_verification.go`、`turnstile-check.go` | 二次验证、人机验证 |
| 通用 | `cors.go`、`gzip.go`、`cache.go`、`disable-cache.go`、`request-id.go`、`request_body_limit.go`、`body_cleanup.go`、`recover.go` | 跨域、压缩、缓存、请求 ID、体积限制、panic 恢复 |
| 观测 | `logger.go`、`stats.go`、`performance.go`、`audit.go` | 访问日志、统计、性能、审计 |
| i18n | `i18n.go` | 后端多语言（go-i18n，en/zh） |
| 任务适配 | `jimeng_adapter.go`、`kling_adapter.go` | 特定 task 平台的请求适配 |

## 分发（distributor）与 relay 的关系

`distributor.go` 是理解「一条中继请求如何选到具体上游」的关键：它根据请求模型、用户分组、渠道能力（`model/ability.go`）与优先级/权重选出 channel，把 `channel_type`、`apiType`、渠道密钥等写入 `gin.Context`；随后 `relay/relay_adaptor.go` 的 `GetAdaptor(apiType)` 据此拿到对应 provider adaptor。

## 认证要点

- 中继 API（`/v1/*`）：API Token（`Authorization: Bearer sk-...`）。
- 管理 API（`/api/*`）：JWT / session cookie（含 Passkey、OAuth）。

## 常见问题 (FAQ)

- **Q：请求没走到预期渠道？** A：排查 `distributor.go` 的选渠道逻辑与 `model/ability.go` 的模型→渠道映射。
- **Q：新增鉴权方式？** A：在 `auth.go` 扩展，并在 `router/` 对应路由挂载。

## 相关文件清单

- 认证：`middleware/auth.go`
- 分发：`middleware/distributor.go`
- 限流：`middleware/rate-limit.go`、`middleware/model-rate-limit.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 middleware 概览文档（认证、限流、distributor 分发）。
