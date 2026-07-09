[根目录](../CLAUDE.md) > **common**

# common — 共享工具

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / common

## 模块职责

全项目复用的基础工具：JSON 包装、配额换算、加密、Redis/缓存、限流、环境变量、SSRF 防护、邮件、TOTP、协程池等。被几乎所有其他模块依赖。

## ⭐ 硬性约定：JSON 包装（`common/json.go`）

所有 JSON marshal/unmarshal **必须**走以下包装，**禁止**在业务代码直接 `import`/调用 `encoding/json`：

- `common.Marshal(v any) ([]byte, error)`
- `common.Unmarshal(data []byte, v any) error`
- `common.UnmarshalJsonStr(data string, v any) error`
- `common.DecodeJson(reader io.Reader, v any) error`
- `common.GetJsonType(data json.RawMessage) string`

> `json.RawMessage`、`json.Number` 等**类型**仍可引用，但实际编解码走 `common.*`。

## ⭐ 配额换算（`common/quota_math.go`）

计费/配额换算的唯一入口，禁止裸 `int(...)` 强转：

- `QuotaFromFloat`（截断）、`QuotaRound`（四舍五入远离零）、`QuotaFromDecimal`（decimal 乘积）
- `*Checked` 变体返回 `*common.QuotaClamp` 用于饱和审计（int32 上界）
- 详见 [pkg/billingexpr/CLAUDE.md](../pkg/billingexpr/CLAUDE.md)。

## 其他关键文件

| 域 | 文件 |
| --- | --- |
| 安全 | `crypto.go`、`hash.go`、`totp.go`、`ssrf_protection.go`、`url_validator.go`、`session_cookie.go`、`verification.go` |
| 存储/缓存 | `redis.go`、`disk_cache.go`、`database.go`、`body_storage.go` |
| 限流/并发 | `rate-limit.go`、`limiter/limiter.go`、`gopool.go`、`go-channel.go` |
| 运行时 | `env.go`、`init.go`、`constants.go`、`gin.go`、`system_monitor*.go`、`pprof.go`、`pyro.go` |
| 邮件 | `email.go`、`email-outlook-auth.go`、`email_ntlm_auth.go` |
| 其他 | `api_type.go`、`endpoint_type.go`、`endpoint_defaults.go`、`str.go`、`utils.go`、`ip.go`、`page_info.go`、`copy.go`、`validate.go`、`topup-ratio.go` |

## 关键依赖与约定

- 本包应尽量零业务依赖，供上层复用；改动影响面大，需谨慎（也更易与上游同步冲突，尽量新增文件）。

## 常见问题 (FAQ)

- **Q：能不能直接用 `json.Marshal`？** A：不能，业务代码一律走 `common.Marshal`（项目硬性规则）。
- **Q：quota 怎么转 int？** A：用 `common/quota_math.go` 的 helper，别裸强转。

## 相关文件清单

- JSON 包装：`common/json.go`
- 配额换算：`common/quota_math.go`
- SSRF 防护：`common/ssrf_protection.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 common 概览文档（JSON 包装、quota 换算等硬性约定）。
