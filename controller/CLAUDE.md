[根目录](../CLAUDE.md) > **controller**

# controller — 请求处理层（对外 API 入口）

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / controller

## 模块职责

分层架构中的 **Controller 层**，位于 `router` 与 `service`/`model` 之间。接收 Gin 请求、做参数解析与鉴权上下文读取、调用 service/model 完成业务、返回响应。是**对外 API 端点的具体实现入口**。

## 关键文件（按域分组）

| 域 | 文件（节选） |
| --- | --- |
| 中继入口 | `relay.go`（连接 `relay/` 中继调度）、`playground.go` |
| 渠道管理 | `channel.go`、`channel-billing.go`、`channel-test.go`、`channel_authz.go`、`channel_upstream_update.go`、`model_sync.go`、`missing_models.go` |
| 用户/令牌/组 | `user.go`、`token.go`、`group.go`、`prefill_group.go` |
| 认证/安全 | `oauth.go`、`custom_oauth.go`、`passkey.go`、`twofa.go`、`wechat.go`、`telegram.go`、`secure_verification.go`、`authz.go` |
| 计费/充值/订阅 | `billing.go`、`pricing.go`、`ratio_config.go`、`ratio_sync.go`、`topup*.go`、`subscription*.go`、`redemption.go`、`checkin.go` |
| 日志/用量/统计 | `log.go`、`usedata.go`、`rankings.go`、`perf_metrics.go`、`performance.go` |
| 模型元数据 | `model.go`、`model_meta.go`、`vendor_meta.go`、`deployment.go` |
| 任务/多媒体 | `task.go`、`task_video.go`、`midjourney.go`、`image.go`、`video_proxy*.go`、`swag_video.go`、`system_task*.go` |
| 系统 | `option.go`、`setup.go`、`system_info.go`、`misc.go`、`console_migrate.go` |

## 对外接口

- 端点绑定在 `router/`（见 [router/CLAUDE.md](../router/CLAUDE.md)）；controller 函数是 Gin `HandlerFunc`。
- 鉴权信息（user id、role、channel、token）由 `middleware/` 写入 `gin.Context`，controller 直接读取。

## 关键依赖与约定

- JSON 收发一律走 `common.Marshal/Unmarshal`（禁止直接 `encoding/json`）。
- 业务逻辑应下沉到 `service/`，controller 保持薄；复杂计费/配额见 `service/quota.go`、`pkg/billingexpr`。
- 数据访问经 `model/`（GORM，三库兼容）。

## 常见问题 (FAQ)

- **Q：新增一个管理端点从哪下手？** A：在对应 `controller/*.go` 写 handler，再到 `router/api-router.go` 注册路由与中间件。
- **Q：中继（/v1/*）相关逻辑在哪？** A：入口 `controller/relay.go`，核心在 `relay/`。

## 相关文件清单

- 中继入口：`controller/relay.go`
- 渠道：`controller/channel.go`
- 用户/令牌：`controller/user.go`、`controller/token.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 controller 概览文档。
