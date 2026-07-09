[根目录](../CLAUDE.md) > **service**

# service — 业务逻辑层

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / service

## 模块职责

分层架构的 **Service 层**：承载核心业务逻辑，被 `controller` 与 `relay` 调用，向下依赖 `model`。计费/配额、渠道选择、请求格式互转、鉴权、文件与媒体处理、支付、系统任务等都在这里。

## 关键子域

| 域 | 文件 / 子包（节选） |
| --- | --- |
| ⭐ 计费/配额 | `quota.go`、`pre_consume_quota.go`（预扣费）、`text_quota.go`、`tool_billing.go`、`task_billing.go`、`tiered_settle.go`（分级结算）、`violation_fee.go`、`billing.go`、`billing_session.go`、`log_info_generate.go`（含 `attachQuotaSaturation` 审计） |
| 渠道选择 | `channel.go`、`channel_select.go`、`channel_affinity.go` |
| ⭐ 格式互转 | `convert.go`、`relayconvert/`（chat↔responses）、`openai_chat_responses_compat.go`、`openaicompat`、`tokenizer.go`、`token_counter.go`、`token_estimator.go`、`usage_helpr.go` |
| 鉴权(RBAC) | `authz/`（casbin `enforcer.go`、`role.go`、`permission.go`、`resolver.go`、`seed.go`） |
| Passkey | `passkey/`（`service.go`、`session.go`、`user.go`） |
| 文件/媒体 | `file_service.go`、`file_decoder.go`、`audio.go`、`image.go`、`download.go` |
| 支付 | `epay.go`、`waffo_pancake.go`、`funding_source.go` |
| 系统任务 | `system_task.go`、`task_polling.go`、`subscription_reset_task.go`、`codex_credential_refresh*.go` |
| 通知/HTTP | `webhook.go`、`notify-limit.go`、`user_notify.go`、`http_client.go`、`protected_fetch_client.go`（SSRF 防护） |

## 计费安全（务必遵守，来自根 AGENTS.md）

- 预扣费（预扣）与结算（差额）都必须防溢出：饱和的超大 quota 必须在预扣费阶段以「配额不足」失败，绝不静默回绕。
- quota 换算统一走 `common/quota_math.go`；发生 clamp 时用 `*Checked` 变体拿到 `QuotaClamp`，经 `log_info_generate.go` 的 `attachQuotaSaturation` 落到日志 `other.admin_info.quota_saturation` 并 `LogWarn`。
- 新增计费路径需走完整链路：validation → EstimateBilling/OtherRatios → quota 转换 → 预扣费 → 结算/退款，每步保持不变式。详见 [pkg/billingexpr/CLAUDE.md](../pkg/billingexpr/CLAUDE.md)。

## 关键依赖与约定

- JSON 走 `common.Marshal/Unmarshal`；数据访问经 `model/`（GORM 三库兼容）。
- 格式互转（Claude/Gemini/Responses ↔ OpenAI）是 relay 复用的基础，供 adaptor 调用。

## 常见问题 (FAQ)

- **Q：新增计费路径怎么保证安全？** A：用 `common.QuotaFrom*Checked`，捕获 `QuotaClamp` 并经 `attachQuotaSaturation` 审计，参考现有 `*_test.go`。
- **Q：想复用格式转换？** A：看 `convert.go`、`relayconvert/`、`openaicompat`。

## 相关文件清单

- 配额：`service/quota.go`、`service/pre_consume_quota.go`、`service/log_info_generate.go`
- 转换：`service/convert.go`、`service/relayconvert/`
- 鉴权：`service/authz/enforcer.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 service 概览文档（计费/配额、格式互转、RBAC 等子域）。
