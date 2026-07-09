[根目录](../../CLAUDE.md) > [web](../) > **classic**

# web/classic — 经典前端（旧版）

> 更新时间：2026-07-09 23:28:40 ｜ 面包屑：根目录 / web / classic

## 模块职责

new-api 的**经典（旧版）前端**，功能与 `web/default` 大体对应，但技术栈更旧。二开团队的前端主战场是 `web/default`；`web/classic` 一般仅作兼容维护或参考，非必要不主改。

## 技术栈

| 类别 | 技术 |
| --- | --- |
| 框架 | React 18 |
| 构建 | Vite |
| UI | Semi Design（`@douyinfe/semi-ui`） |
| 语言 | JavaScript / JSX（非 TS） |
| i18n | i18next（`src/i18n/`） |

## 入口与启动

- 入口：`src/index.jsx` → `src/App.jsx`
- 脚本以本目录 `package.json` 为准（Vite dev/build）。

## 目录结构（概览）

```
web/classic/src/
├── index.jsx / App.jsx     # 入口与根组件
├── pages/                  # 页面（Setting、TopUp 等）
├── components/             # 组件（layout、playground、table…）
├── hooks/                  # 业务 Hooks（channels、dashboard、playground…）
├── helpers/                # 工具（api、quota、token、statusCodeRules…）
├── constants/              # 常量（channel、billing、console…）
├── context/                # 全局状态（Status/User reducer）
├── services/               # secureVerification 等
└── i18n/                   # 国际化（i18n.js、language.js）
```

## 与 default 前端的关系

- 两套前端并存于 `web/` 下，通过后端 `web-router` 按主题选择。
- `.agents/skills/classic-to-default-sync/` 存在「classic → default 同步」技能，说明部分能力以 default 为主、classic 跟随。

## 常见问题 (FAQ)

- **Q：新功能加在 classic 还是 default？** A：优先 `web/default`（主战场）。classic 仅在需要兼容旧主题时改动。
- **Q：classic 用什么组件库？** A：Semi Design（与 default 的 Base UI 不同，勿混用）。

## 相关文件清单

- 入口：`src/index.jsx`、`src/App.jsx`
- 请求：`src/helpers/api.js`
- i18n：`src/i18n/i18n.js`

## 变更记录 (Changelog)

- **2026-07-09 23:28:40**：初始化经典前端概览文档。
