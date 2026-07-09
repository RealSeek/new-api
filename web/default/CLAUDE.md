[根目录](../../CLAUDE.md) > [web](../) > **default**

# web/default — 默认前端（前端主战场）

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / web / default
>
> ⭐ 这是二开团队的**前端主战场**。前端改动应尽量集中在本目录，以降低与上游每日自动同步的冲突面（详见根 [CLAUDE.md 二开维护指南](../../CLAUDE.md)）。
>
> **权威规范**：本目录的 `AGENTS.md` 是前端开发规范的权威来源，本文档仅作结构导航与快速上手，具体规则以 `AGENTS.md` 与 `package.json` 为准。

## 模块职责

new-api 的现代默认前端，提供管理后台（渠道、用户、令牌、日志、系统设置）、Playground、定价页、仪表盘、订阅/钱包等完整 Web 界面。基于文件式路由 + 功能模块（feature）化组织。

## 技术栈

| 类别 | 技术 |
| --- | --- |
| 包管理 | **Bun**（优先于 npm/yarn/pnpm） |
| 框架 | React 19 + TypeScript |
| 构建 | Rsbuild（`rsbuild.config.ts`） |
| 路由 | TanStack Router（**文件式路由**，`src/routes/`） |
| 数据请求 | TanStack Query（`useQuery`/`useMutation`）+ axios |
| 状态管理 | Zustand（`src/stores/`） |
| 表格/虚拟化 | TanStack Table + TanStack Virtual |
| 表单/校验 | React Hook Form + Zod |
| UI/样式 | Base UI（`@base-ui/react`）+ Tailwind CSS v4 + Hugeicons + `clsx`/`cva` |
| 图表 | VisActor VChart + Recharts |
| i18n | i18next + react-i18next + browser-languagedetector |
| 工具链 | `tsgo`（typecheck）、`oxlint`（lint）、`oxfmt`（format）、`knip`（未用依赖检测） |

## 入口与启动

- HTML 入口：`index.html` → `src/main.tsx`
- 路由树：`src/routeTree.gen.ts`（由 TanStack Router 插件**自动生成，勿手改**）
- 根路由：`src/routes/__root.tsx`

### 开发命令（在 `web/default/` 下执行）

```bash
bun install            # 安装依赖
bun run dev            # 开发服务器（rsbuild dev）
bun run build          # 生产构建（rsbuild build）
bun run build:check    # tsgo -b 类型检查 + 构建
bun run typecheck      # 仅类型检查（tsgo -b）
bun run lint           # oxlint 检查
bun run lint:fix       # oxlint 自动修复
bun run format         # 格式化（保护版权头）
bun run i18n:sync      # 同步 i18n 翻译键（scripts/sync-i18n.mjs）
bun run knip           # 检测未使用的依赖/导出
```

> 规范要求：改动 TS/TSX 后必须 `typecheck` 至无错；提交前必须对改动文件 `lint` 修复所有 error。

## 目录结构

```
web/default/
├── index.html                 # HTML 入口
├── package.json               # 依赖与脚本（Bun）
├── rsbuild.config.ts          # 构建配置
├── AGENTS.md                  # ⭐ 前端开发规范（权威）
├── components.json            # shadcn/Base UI 组件配置
├── .oxlintrc.json / .oxfmtrc.json   # lint / format 配置
├── scripts/                   # i18n 同步、版权头、格式化脚本
└── src/
    ├── main.tsx               # 应用入口
    ├── routeTree.gen.ts       # 自动生成的路由树（勿手改）
    ├── routes/                # ⭐ 文件式路由（页面）
    ├── features/              # ⭐ 功能模块（业务主体）
    ├── components/            # 通用组件
    │   ├── ui/                #   Base UI 基础组件（button/dialog/table…约 60+）
    │   ├── ai-elements/       #   AI 对话/Playground 相关组件
    │   ├── data-table/        #   通用数据表格（core/layout/toolbar/static）
    │   └── layout/            #   布局（header/sidebar/footer…）
    ├── lib/                   # 通用工具与类型（api、format、utils…）
    ├── hooks/                 # 通用自定义 Hooks
    ├── stores/                # Zustand store（auth/notification/system-config）
    ├── i18n/                  # 国际化（config、languages、locales/*.json）
    └── styles/                # 全局样式（index/theme/theme-presets .css）
```

## 功能模块（`src/features/`）

按功能域组织，每个 feature 内含 `components/`、`lib/`、`hooks/`，以及按需的 `api.ts`、`types.ts`、`constants.ts`、`index.tsx` 入口：

`about` · `auth`（含 sign-in/sign-up/otp/passkey/secure-verification/forgot-password）· `channels`（渠道管理，组件最丰富）· `chat` · `dashboard` · `home` · `keys` · `models` · `playground` · `pricing` · `profile` · `rankings` · `redemption-codes` · `subscriptions` · `system-info` · `system-settings`（auth/billing/content/models/operations/security/site 分区）· `usage-logs` · `users` · `wallet`

## 对外接口（前端 → 后端）

- 统一 axios 实例：`src/lib/api.ts`（含 `baseURL`、`withCredentials: true`、认证/错误拦截）。
- 数据获取用 TanStack Query，`queryKey` 用数组分层；变更后 `invalidateQueries`。
- 服务端错误统一走 `src/lib/handle-server-error.ts`，配合 `sonner` toast 展示。

## 状态管理（`src/stores/`）

- `auth-store.ts` — 登录态与用户信息（选择器订阅：`useAuthStore((s) => s.auth.user)`）。
- `notification-store.ts` — 通知。
- `system-config-store.ts` — 系统配置。
- 需持久化的状态在 store 内读写 `localStorage`。

## 国际化（i18n）

- 配置：`src/i18n/config.ts`；语言映射：`src/i18n/languages.ts`。
- 支持语言：`en`（base）、`zhCN`、`zhTW`、`fr`、`ru`、`ja`、`vi`（`fallbackLng: 'en'`）。
- 翻译文件：`src/i18n/locales/{en,zh,zh-TW,fr,ru,ja,vi}.json` —— **扁平 JSON，key 即英文源串**。
- 用法：组件内 `const { t } = useTranslation()`，调用 `t('English key')`；非 React 环境用 `import { t } from 'i18next'`。
- 常量中的消息/label 存 i18n key，展示时必须 `t(...)`，禁止直接把常量当最终文案。
- 新增文案后运行 `bun run i18n:sync` 同步各语言键。

## 如何新增页面 / 组件

**新增页面（路由）**：

1. 在 `src/routes/` 下按文件式路由约定新增 `*.tsx`（受保护页面放 `_authenticated/` 下；用 `createFileRoute` 定义；搜索参数用 Zod `validateSearch`；认证/重定向放 `beforeLoad`）。
2. 复杂业务放到对应 `src/features/<feature>/`，路由文件仅做装配与懒加载（`React.lazy`）。
3. `routeTree.gen.ts` 由插件自动更新，**勿手改**。

**新增功能模块**：

1. 建 `src/features/<feature>/`，内含 `index.tsx` 入口 + 按需 `components/`、`lib/`、`hooks/`、`api.ts`、`types.ts`、`constants.ts`。
2. 组件文件 PascalCase，工具/类型文件 kebab-case；类型 `export type` 且 PascalCase。
3. 单文件超约 200 行考虑拆子组件或抽 Hooks。

**新增基础 UI 组件**：优先复用 `src/components/ui/`（Base UI 封装，约 60+ 个）；样式以 Tailwind 工具类为主，动态类名用 `cn()`。

## 数据模型（前端类型）

- 各 feature 的 `types.ts` 定义本域类型；通用类型放 `src/lib/`。
- 表单 schema 用 Zod 定义于 feature 的 `lib/`，`z.infer` 导出表单类型。

## 测试与质量

- 单元测试用 Vitest（`*.test.ts`），组件用 React Testing Library；测行为而非实现细节。
- 提交前执行 `typecheck` + `lint`（error 必须清零）+ `format`。
- 测试须保护真实用户行为/稳定契约/回归路径，禁止为覆盖率堆 smoke/sleep/随机输入测试。

## 常见问题 (FAQ)

- **Q：路由树报错或页面 404？** A：确认 `src/routes/` 文件命名符合 TanStack 约定；`routeTree.gen.ts` 由插件生成，不要手动编辑。
- **Q：新增文案没生效/其他语言缺失？** A：文案写成 `t('English key')`，再 `bun run i18n:sync`。
- **Q：为什么用 tsgo/oxlint 而非 tsc/eslint？** A：本项目采用原生 TS 预览与 oxc 工具链提速，脚本以 `package.json` 为准。
- **Q：改这里会不会被上游同步覆盖？** A：`web/default/` 是冲突面较小的前端主战场，但仍是上游目录；改动越集中、越倾向新增文件越安全。

## 相关文件清单

- 规范：`web/default/AGENTS.md`
- 入口：`src/main.tsx`、`src/routes/__root.tsx`
- 请求：`src/lib/api.ts`、`src/lib/handle-server-error.ts`
- 状态：`src/stores/*.ts`
- i18n：`src/i18n/config.ts`、`src/i18n/locales/*.json`
- 构建：`rsbuild.config.ts`、`package.json`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化前端主战场文档（技术栈、目录、i18n、状态管理、开发命令、新增页面/组件指引）。
