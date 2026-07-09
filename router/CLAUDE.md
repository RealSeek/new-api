[根目录](../CLAUDE.md) > **router**

# router — HTTP 路由划分与对外 API 契约

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / router

## 模块职责

分层架构最外层，负责 **HTTP 路由注册**：把 URL 路径 + 方法映射到 `controller` handler，并挂载对应 `middleware`（认证、限流、分发、CORS 等）。

## 路由划分

| 文件 | 职责 |
| --- | --- |
| `main.go` | 路由总装配（SetRouter），挂载全局中间件 |
| `relay-router.go` | ⭐ **对外 OpenAI 兼容中继 API**（`/v1/*` 等），走 token 鉴权 + distributor |
| `api-router.go` | 管理/控制台 API（`/api/*`），JWT/session 鉴权 |
| `dashboard.go` | 仪表盘数据 API |
| `channel-router.go` | 渠道管理路由 |
| `authz-router.go` | RBAC 授权相关路由 |
| `video-router.go` | 视频任务/代理路由 |
| `web-router.go` | 前端静态资源与主题（`web/default`、`web/classic`）分发 |

## 对外 API 契约与认证

- **中继 API（`/v1/*`）**：OpenAI 兼容协议（chat/completions、responses、embeddings、images、audio、rerank，及 Claude `/v1/messages`、Gemini 原生路径等）。使用 **API Token**（`Authorization: Bearer sk-...`）鉴权，经 `middleware` 的 token 校验 + `distributor` 选渠道。
- **管理 API（`/api/*`）**：面向前端后台，使用 **JWT / session cookie**（含 WebAuthn/Passkey、OAuth 登录态）。
- 具体认证逻辑见 [middleware/CLAUDE.md](../middleware/CLAUDE.md)。

## 对接其他网关的意义

若要让其他网关消费 new-api 的统一 API，或模仿其对外契约，`relay-router.go` + `relay/` 是权威参照：它定义了 new-api 暴露哪些 OpenAI 兼容端点、如何鉴权、如何进入中继链路。

## 常见问题 (FAQ)

- **Q：新增对外中继端点？** A：在 `relay-router.go` 注册路径，handler 走 `controller/relay.go` → `relay/`。
- **Q：新增后台管理端点？** A：在 `api-router.go` 注册，注意挂载正确的鉴权中间件。

## 相关文件清单

- 装配：`router/main.go`
- 中继：`router/relay-router.go`
- 管理：`router/api-router.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 router 概览文档（路由划分、对外契约与认证）。
