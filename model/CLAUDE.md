[根目录](../CLAUDE.md) > **model**

# model — 数据模型与渠道（GORM 跨三库）

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / model

## 模块职责

分层架构的 **Model 层**：定义 GORM 数据模型与所有数据库访问逻辑。必须**同时兼容 SQLite、MySQL ≥ 5.7.8、PostgreSQL ≥ 9.6**。

## 关键模型（按域）

| 域 | 文件（节选） |
| --- | --- |
| ⭐ 渠道 | `channel.go`、`channel_cache.go`、`channel_satisfy.go`、`ability.go`（模型→渠道能力路由）、`vendor_meta.go` |
| 用户/令牌 | `user.go`、`user_cache.go`、`user_oauth_binding.go`、`token.go`、`token_cache.go`、`passkey.go`、`twofa.go` |
| 计费/配额 | `redemption.go`、`topup.go`、`subscription.go`、`pricing*.go`、`usedata*.go`、`usedata_rankings.go` |
| 日志/任务 | `log.go`、`task.go`、`system_task.go`、`midjourney.go`、`perf_metric.go` |
| 权限(RBAC) | `authz_role.go`、`casbin_rule.go` |
| 系统 | `option.go`（系统配置持久化）、`setup.go`、`main.go`（DB 初始化/迁移/跨库工具）、`system_instance.go` |

## 渠道模型（对接重点）

`channel.go` 存储上游渠道配置：类型（`ChannelType`）、密钥、支持的模型列表、分组（group）、优先级/权重、状态、代理、多密钥模式等。`ability.go` 维护「模型 → 可用渠道」的路由表，供 `middleware/distributor` 选渠道。

## 跨库兼容要点（务必遵守，来自根 AGENTS.md）

- 优先 GORM 方法（`Create`/`Find`/`Where`/`Updates`），少写 raw SQL。
- 主键交给 GORM 生成，勿用 `AUTO_INCREMENT`/`SERIAL`。
- 保留字列 `group`/`key` 用 `model/main.go` 的 `commonGroupCol`/`commonKeyCol`；布尔用 `commonTrueVal`/`commonFalseVal`。
- 主库/日志库分支：`common.UsingMainDatabase(...)` / `common.UsingLogDatabase(...)`。
- 迁移用 `ALTER TABLE ... ADD COLUMN`（SQLite 不支持 `ALTER COLUMN`），参考 `main.go` 现有模式。
- **避免 `gorm:"default:true"`**：MySQL/PG 布尔默认值归一化不同，会导致 `AutoMigrate` 每次重启反复 `ALTER TABLE`；默认值改在请求/模型规范化、hook、构造函数或 service 中设置。

## 关键依赖与约定

- JSON 走 `common.Marshal/Unmarshal`。
- 缓存：部分模型有 `*_cache.go`（内存 + Redis），注意读写一致性。

## 常见问题 (FAQ)

- **Q：新增字段导致重启反复迁移？** A：多半是布尔 `default` 标签，改用代码层默认值。
- **Q：raw SQL 在某库报错？** A：检查列引号（PG `"col"` vs MySQL/SQLite `` `col` ``）与保留字，用 `commonGroupCol`/`commonKeyCol`。

## 相关文件清单

- DB 初始化/迁移/跨库工具：`model/main.go`
- 渠道：`model/channel.go`、`model/ability.go`
- 用户/令牌：`model/user.go`、`model/token.go`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化 model 概览文档（渠道模型、GORM 三库兼容要点）。
