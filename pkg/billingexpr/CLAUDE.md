[根目录](../../CLAUDE.md) > [pkg](../) > **billingexpr**

# pkg/billingexpr — 表达式计费系统与配额安全

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / pkg / billingexpr
>
> ⚠️ **动手前必读**：`pkg/billingexpr/expr.md`（设计哲学、表达式语言、完整架构、token 归一化、quota 换算、表达式版本化）。所有表达式计费改动都必须遵循该文档。

## 模块职责

实现**分级/动态（表达式）计费**：把可配置的计费表达式编译、执行、结算，得到最终 quota。用于按 token/时长/分辨率等维度做阶梯或动态定价。

## 关键文件

| 文件 | 职责 |
| --- | --- |
| `compile.go` | 编译计费表达式 |
| `run.go` | 执行表达式，代入 token/参数得数值 |
| `settle.go` | 结算（含 clamp 检查，见 `settle_clamp_test.go`） |
| `round.go` | `billingexpr.QuotaRound` 委托 `common.QuotaRound` |
| `types.go` | 表达式相关类型 |

## 配额安全不变式（务必遵守，来自根 AGENTS.md）

- **绝不产生负计费**：任何用户可控的计费乘数（图片 `n`、视频 `seconds`/`duration`、分辨率/质量倍率、批量数）必须在进入 quota 计算前**校验上界**，越界在请求校验处返回 400。复用既有常量（`dto.MaxImageN`、`relaycommon.MaxTaskDurationSeconds`、`relay/helper/valid_request.go` 的 `maxTokensLimit`）。
- **禁止裸 `int(...)` 强转** quota/token：统一用 `common/quota_math.go` 的 `QuotaFromFloat`（截断）、`QuotaRound`（四舍五入远离零）、`QuotaFromDecimal`。饱和上界为 int32（quota 列是 32 位整数）。
- **饱和审计**：用 `*Checked` 变体（`QuotaFromFloatChecked` 等）拿 `*common.QuotaClamp`，经 `service/log_info_generate.go` 的 `attachQuotaSaturation` 落到日志 `other.admin_info.quota_saturation` 并 `LogWarn`（`admin_info` 对非管理员视图自动脱敏）。
- 乘数 map 走 `types.PriceData.AddOtherRatio`（拒绝非正/NaN/+Inf），勿直接写 `PriceData.OtherRatios`。
- 预扣费与结算都要防溢出；对 `*uint` 字段仅 `>= 0` 不够，**必须设上界**。
- 回归测试与其保护的边界同处：`common/quota_math_test.go`、`relay/helper/openai_image_request_test.go`、`relay/common/relay_utils_test.go`、本目录 `billingexpr_test.go` / `settle_clamp_test.go`。

## 关键依赖

- 换算/舍入：`common/quota_math.go`
- 价格数据：`types/price_data.go`（`AddOtherRatio` 守卫）
- 审计落库：`service/log_info_generate.go`

## 常见问题 (FAQ)

- **Q：新增一种动态计费维度？** A：先读 `expr.md`；乘数先设上界，换算走 `quota_math` 的 `*Checked` 变体，走完 validation→换算→预扣→结算全链路。
- **Q：为什么日志里有 `quota_saturation`？** A：发生 clamp 的审计标记，说明某次计费触发了饱和上界，需排查异常输入。

## 相关文件清单

- 必读设计文档：`pkg/billingexpr/expr.md`
- 换算助手：`common/quota_math.go`
- 审计：`service/log_info_generate.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 billingexpr 概览文档（表达式计费 + 配额安全不变式）。
