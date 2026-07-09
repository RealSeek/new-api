[根目录](../CLAUDE.md) > **setting**

# setting — 配置管理

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / setting

## 模块职责

集中管理系统运行时配置：倍率（ratio）、模型（model）、运营（operation）、系统（system）、性能（performance）等。配置多以 `model/option.go` 持久化、内存缓存，运行时可动态更新。

## 子包结构

| 子包 | 职责（节选） |
| --- | --- |
| ⭐ `ratio_setting/` | 计费倍率核心：`model_ratio.go`、`group_ratio.go`、`cache_ratio.go`、`expose_ratio.go`、`compact_suffix.go` |
| `model_setting/` | 各模型族设置：`claude.go`、`gemini.go`、`grok.go`、`qwen.go`、`global.go` |
| `operation_setting/` | 运营：`general_setting.go`、`quota_setting.go`、`payment_setting.go`、`token_setting.go`、`monitor_setting.go`、`checkin_setting.go`、`channel_affinity_setting.go`、`status_code_ranges.go` |
| `system_setting/` | 系统：`oidc.go`、`passkey.go`、`discord.go`、`legal.go`、`theme.go`、`fetch_setting.go` |
| `billing_setting/` | `tiered_billing.go`（分级计费开关/参数） |
| `performance_setting/`、`perf_metrics_setting/` | 性能与指标 |
| `config/`、`console_setting/`、`reasoning/` | 全局 config、控制台设置、推理后缀 |
| 顶层 | `rate_limit.go`、`sensitive.go`、`chat.go`、`midjourney.go`、`auto_group.go`、`payment_*.go`、`user_usable_group.go` |

## 与计费的关系

`ratio_setting/` 是计费的关键输入：模型倍率、分组倍率、缓存倍率等决定最终 quota 计算。改动倍率语义须结合 [pkg/billingexpr/CLAUDE.md](../pkg/billingexpr/CLAUDE.md) 与 `service/quota.go` 一起看。

## 关键依赖与约定

- 持久化经 `model/option.go`；JSON 走 `common.Marshal/Unmarshal`。
- 配置读写注意并发安全（多为全局变量 + 锁 / 原子）。

## 常见问题 (FAQ)

- **Q：新增一项系统配置？** A：在对应子包加字段与读写，接入 `model/option.go` 持久化，前端在 `web/default` 的 system-settings 对应分区加 UI。
- **Q：模型倍率在哪配？** A：`ratio_setting/model_ratio.go`。

## 相关文件清单

- 倍率：`setting/ratio_setting/model_ratio.go`、`setting/ratio_setting/group_ratio.go`
- 分级计费：`setting/billing_setting/tiered_billing.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 setting 概览文档（ratio/model/operation/system/performance）。
