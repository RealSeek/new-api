[根目录](../CLAUDE.md) > **oauth**

# oauth — 第三方登录 Provider 抽象与注册表

> 更新时间：2026-07-09 23:51:43 ｜ 面包屑：根目录 / oauth
>
> `package oauth`，定义 OAuth 登录的统一 `Provider` 接口 + 全局注册表，内置若干固定 provider，并支持从数据库动态加载「自定义（通用）OAuth」provider。

## 模块职责

把「GitHub / Discord / LinuxDo / OIDC / 自定义」等不同 OAuth 服务抽象成同一 `Provider` 接口，供登录控制器（`controller/`）统一调用：换取 token → 拉取用户信息 → 绑定/查找本地 `model.User`。

## 文件清单与职责

| 文件 | 职责 |
| --- | --- |
| `provider.go` | ⭐ `Provider` 接口定义（所有 provider 必须实现）。 |
| `registry.go` | ⭐ 全局注册表：`Register` / `RegisterCustom` / `GetProvider` / `GetAllProviders` / `LoadCustomProviders` 等。 |
| `types.go` | 通用数据类型：`OAuthToken`、`OAuthUser`、`OAuthError`（i18n 可翻译错误）、`AccessDeniedError`。 |
| `github.go` | GitHub provider（`init()` 内 `Register("github", …)`，用数字 ID 作主标识）。 |
| `discord.go` | Discord provider（`Register("discord", …)`）。 |
| `linuxdo.go` | LinuxDo provider（`Register("linuxdo", …)`）。 |
| `oidc.go` | OIDC provider（`Register("oidc", …)`，标准 OpenID Connect）。 |
| `generic.go` | ⭐ `GenericOAuthProvider`：数据库配置驱动的**通用/自定义** provider（gjson 字段抽取 + 访问策略引擎 + 多种 AuthStyle）。 |

## 核心接口：`Provider`（`provider.go`）

| 方法 | 职责 |
| --- | --- |
| `GetName()` | 展示名（"GitHub"、"Discord"…）。 |
| `IsEnabled()` | 该 provider 是否启用（读 `common.*OAuthEnabled` 或配置）。 |
| `ExchangeToken(ctx, code, c)` | 用授权码换 `*OAuthToken`（部分 provider 需 `gin.Context` 拼 redirect_uri）。 |
| `GetUserInfo(ctx, token)` | 用 token 拉取 `*OAuthUser`。 |
| `IsUserIDTaken(providerUserID)` | provider 用户 ID 是否已被占用。 |
| `FillUserByProviderID(user, id)` | 按 provider 用户 ID 回填本地 `model.User`。 |
| `SetProviderUserID(user, id)` | 在 `model.User` 上写入 provider 用户 ID。 |
| `GetProviderPrefix()` | 自动生成用户名前缀（如 `github_`）。 |

## 注册机制（`registry.go`）

- 内置固定 provider 在各自文件的 `init()` 里调 `Register(slug, provider)`，进程启动即可用。**当前固定注册的有 4 个：`github` / `discord` / `linuxdo` / `oidc`。**
- 自定义 provider 走 `GenericOAuthProvider`：由 `LoadCustomProviders()` 从数据库（`model.GetAllCustomOAuthProviders()`）读取配置，逐个 `RegisterCustom(slug, provider)`；`customProviderSlugs` 标记哪些可被卸载/重载（`ReloadCustomProviders` / `RegisterOrUpdateCustomProvider` / `UnregisterCustomProvider`）。
- 注册表用 `sync.RWMutex` 保护并发读写。

> 说明：本包的**固定内置 provider 是 4 个**（github/discord/linuxdo/oidc），外加 1 套 `GenericOAuthProvider` 通用机制承载任意「自定义」provider（数量由数据库配置决定）。项目其它登录方式（如邮箱、Telegram、微信、WebAuthn/Passkey）不在本包内。

## 通用自定义 provider 亮点（`generic.go`）

- **AuthStyle**：`AutoDetect` / `InParams`（表单参数）/ `InHeader`（Basic Auth）三种客户端凭据传递方式。
- **字段抽取**：用 `gjson` 按配置的 `UserIdField` / `UsernameField` / `DisplayNameField` / `EmailField` 从用户信息 JSON 中取值（支持 JSONPath 式路径）。
- **访问策略引擎**：`AccessPolicy`（JSON 配置）支持 `and`/`or` 逻辑、嵌套分组与 `eq/ne/gt/gte/lt/lte/in/not_in/contains/not_contains/exists/not_exists` 等算子，不满足则抛 `AccessDeniedError`（支持 `{{provider}}`/`{{field}}`/`{{current.x}}` 等占位符渲染拒绝文案）。

## 如何新增一个 OAuth 登录 provider

**方案 A（推荐，无需改代码）——自定义/通用 provider**：在管理后台新增一条自定义 OAuth 配置（TokenEndpoint / UserInfoEndpoint / 字段映射 / AuthStyle / 访问策略），由 `GenericOAuthProvider` 承载。适合绝大多数标准 OAuth2 服务。

**方案 B——固定内置 provider（需改代码，会动上游文件，注意二开冲突）**：

1. 新建 `oauth/{name}.go`，定义 `type {Name}Provider struct{}` 并实现 `Provider` 全部 8 个方法。
2. 文件内 `func init() { Register("{name}", &{Name}Provider{}) }`。
3. 加启用开关（`common` 变量或 `setting/system_setting`）供 `IsEnabled()` 读取。
4. 在 `model.User` 上加对应 ID 字段 + `FillUserBy{Name}Id()` + `Is{Name}IdAlreadyTaken()`。
5. 在 `controller/` 加登录回调处理、`router/` 注册回调端点；错误用 `NewOAuthError(i18n.Key, ...)` 以支持多语言。

## 常见问题 (FAQ)

- **Q：新增标准 OAuth 一定要写 Go 代码吗？** A：不用。优先用后台「自定义 OAuth」（`GenericOAuthProvider`），零代码接入。
- **Q：GitHub 用户改了用户名会掉登录吗？** A：不会。GitHub provider 用**数字 ID** 作主标识，`login` 仅存于 `Extra.legacy_id` 供旧账号迁移。
- **Q：自定义 provider 改了配置为何没生效？** A：需 `ReloadCustomProviders()` / `RegisterOrUpdateCustomProvider(...)` 重新注册（后台保存时会触发）。

## 相关文件清单

- 接口 / 注册表：`provider.go`、`registry.go`
- 通用自定义：`generic.go` ↔ `model.CustomOAuthProvider`、`controller/custom_oauth.go`
- 数据类型：`types.go`
- 登录入口：`controller/`（OAuth 回调）、`router/`（回调路由）

## 变更记录 (Changelog)

- **2026-07-09 23:51:43**：初始化 oauth 文档（Provider 接口、注册表机制、4 个固定内置 provider + 通用自定义 provider、新增 provider 的两种方案）。
