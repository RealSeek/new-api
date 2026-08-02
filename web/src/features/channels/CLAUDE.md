[根目录](../../../../../CLAUDE.md) > [web/default](../../../CLAUDE.md) > features > **channels**

# features/channels — 渠道管理（前端主战场核心，最复杂 feature）⭐

> 更新时间：2026-07-09 23:51:43 ｜ 面包屑：根目录 / web/default / features/channels
>
> ⭐ 这是 `web/default` 里**组件最多、逻辑最重**的功能模块：上游渠道（provider 接入点）的列表、搜索、增删改、连通性测试、拉取模型、多密钥管理、标签批量操作、上游模型更新等。前端接「对接其他网关」的绝大多数改动都会落在这里。
>
> **权威规范**：前端整体约定见 [web/default/CLAUDE.md](../../../CLAUDE.md) 与 `web/default/AGENTS.md`；本文件仅作本 feature 的结构导航与上手指引。

## 模块职责

管理后端 `channel` 资源（对应 `constant/channel.go` 的 `ChannelType`）。一个「渠道」= 一个上游 provider 接入配置（类型、Key、BaseURL、支持模型、分组、优先级、参数/头覆盖、多密钥等）。本 feature 负责其完整 CRUD 与运维操作的 UI。

## 目录结构

```
web/default/src/features/channels/
├── index.tsx                     # 入口 <Channels/>：装配 Provider + Table + 按钮 + Dialogs
├── api.ts                        # ⭐ 后端 API 封装（/api/channel*，axios）
├── constants.ts                  # ⭐ 常量：CHANNEL_TYPES/状态/多密钥/字段提示/错误消息（多为 i18n key）
├── types.ts                      # 本域 TS 类型（Channel、请求/响应参数）
├── hooks/
│   ├── use-channel-mutate-form.ts    # ⭐ 创建/更新提交 mutation（敏感字段权限门控）
│   └── use-channel-upstream-updates.ts # 上游模型更新检测状态
├── lib/
│   ├── index.ts                  # 桶导出（channelsQueryKeys、transform*、util 等）
│   ├── channel-form.ts           # ⭐ Zod 表单 schema + 表单↔API 载荷转换
│   ├── channel-form-errors.ts    # 高级设置区错误归集（驱动折叠区红点）
│   ├── channel-type-config.ts    # ⭐ 各渠道类型的图标/默认 BaseURL/提示/校验配置
│   ├── channel-utils.ts          # 标签聚合、类型图标/标签、连接信息解析等展示辅助
│   ├── channel-actions.ts        # channelsQueryKeys + 行级操作（启用/禁用/删除/测试…）封装
│   ├── multi-key-utils.ts        # 多密钥解析/统计辅助
│   ├── model-mapping-validation.ts # 模型映射 JSON 校验
│   ├── advanced-custom.ts        # Advanced Custom(58) 路由配置解析/校验
│   ├── upstream-update-utils.ts  # 上游模型更新工具
│   ├── status-code-risk-guard.ts # 状态码映射风险识别
│   └── ollama-utils.ts           # Ollama 相关辅助
└── components/
    ├── channels-provider.tsx     # ⭐ Context：弹窗类型/当前行/标签模式/批量模式/上游状态
    ├── channels-table.tsx        # ⭐ 列表：TanStack Table + URL 状态 + 卡片/表格视图
    ├── channels-columns.tsx      # useChannelsColumns 列定义
    ├── channels-dialogs.tsx      # 挂载所有对话框（按 open 类型渲染）
    ├── channels-primary-buttons.tsx # 顶部主操作按钮（新增/批量等）
    ├── channel-card.tsx          # 移动端卡片视图
    ├── data-table-bulk-actions.tsx / data-table-row-actions.tsx / data-table-tag-row-actions.tsx
    ├── model-mapping-editor.tsx / numeric-spinner-input.tsx
    ├── drawers/
    │   ├── channel-mutate-drawer.tsx # ⭐ 新增/编辑侧抽屉（Sheet），装配各 section
    │   └── sections/             #   basic/auth/models/advanced/api-access + loading-state
    └── dialogs/                  # ⭐ 对话框群（见下表）
```

## 数据流

```
<Channels> (index.tsx)
  └─ <ChannelsProvider>                         // Context：open 弹窗、currentRow、标签/批量模式、upstream 状态
       ├─ <ChannelsPrimaryButtons/>             // 触发 setOpen('create-channel') 等
       ├─ <ChannelsTable/>                       // 列表主体
       │    ├─ useQuery(getChannels/searchChannels)   // 拉数据
       │    ├─ useTableUrlState(...)                   // filter/sort/pagination 同步到 URL
       │    ├─ useChannelsColumns()                    // 列定义（含行操作）
       │    └─ 卡片/表格双视图 + 列宽/列可见性存 localStorage
       └─ <ChannelsDialogs/>                     // 按 open 类型渲染对应抽屉/对话框
```

1. **列表加载**：`ChannelsTable` 用 `useQuery` 调 `api.ts` 的 `getChannels`（或 `searchChannels`）。分页/排序/筛选/关键词经 `useTableUrlState` 双向绑定到路由 search 参数；状态筛选还回退读 `localStorage`。标签模式下用 `aggregateChannelsByTag` 把同标签渠道聚合成父行。
2. **表格渲染**：`useChannelsColumns` 定义列；行操作（启用/禁用/测试/复制/删除/多密钥…）经 `data-table-row-actions.tsx` 触发，通过 Provider 的 `setOpen` + `setCurrentRow` 打开对应弹窗。
3. **增删改抽屉**：`channel-mutate-drawer.tsx` 是一个 `Sheet` 侧抽屉，用 React Hook Form + `zodResolver(channelFormSchema)` 承载表单，按 section 拆分（基础/认证/模型/高级/API 访问）。提交走 `use-channel-mutate-form.ts`（`createChannel` / `updateChannel` mutation），成功后 `invalidateQueries(channelsQueryKeys.all)` 刷新列表。
4. **测试 / 取模型等对话框**：由 `channels-dialogs.tsx` 依据 Provider 的 `open` 值渲染，各自调 `api.ts` 对应函数（`testChannel`、`fetchUpstreamModels`、`manageMultiKeys`…）。

### API 层要点（`api.ts`）

- 统一走 `@/lib/api`（axios 实例）；渠道操作用 `channelActionConfig()` 包装（`skipBusinessError` + `skipErrorHandler`，让 mutation 层自行处理错误/toast）。
- 覆盖：基础 CRUD（`getChannels`/`searchChannels`/`getChannel`/`createChannel`/`updateChannel`/`deleteChannel` + 批量）、运维（`testChannel`/`updateChannelBalance`/`fetchUpstreamModels`/`copyChannel`/`fixChannelAbilities`）、多密钥（`manageMultiKeys` 及 enable/disable/delete 系列）、标签（`enableTagChannels`/`editTagChannels`…）、Codex（`refreshCodexCredential`/`getCodexUsage`…）、Ollama、`getPrefillGroups`（模型分组快选）。
- 端点前缀统一 `/api/channel`。

## 表单体系（重点）

- **`lib/channel-form.ts`** — `channelFormSchema`（Zod）定义所有字段，`.superRefine` 按渠道类型追加校验（如 type 3/8/36/45 必填 base_url；type 41 Vertex JSON key 校验；type 57 Codex 凭证 JSON 校验；Advanced Custom(58) 路由校验）。转换函数：
  - `transformChannelToFormDefaults(channel)` — 后端 Channel → 表单默认值（解析 `setting`/`settings` JSON 出各开关；**Key 永不回填**）。
  - `transformFormDataToCreatePayload` / `transformFormDataToUpdatePayload` — 表单 → API 载荷，`buildSettingJSON`/`buildSettingsJSON` 按类型把开关写回 `setting`/`settings` JSON；更新时对可空字段发显式空串以便 GORM 清空。
  - 辅助：`parseModels`/`formatModels`/`parseGroups`/`formatGroups`/`validateJSON`/`validateModelMapping`。
- **`hooks/use-channel-mutate-form.ts`** — 提交 mutation：编辑时 Key 为空则不发；**敏感字段**（`SENSITIVE_UPDATE_FIELDS`：type/key/base_url/param_override/header_override/setting/settings/other 等）需 `SENSITIVE_WRITE` 权限，无权限则从载荷剔除；多密钥编辑支持 `key_mode`（append/replace）。
- **`lib/channel-form-errors.ts`** — `isAdvancedSettingsField`/`hasAdvancedSettingsErrors` 判断错误是否落在「高级设置」字段集合，用于给折叠区标红点。
- **`components/drawers/sections/`** — 抽屉分区（`index.ts` 桶导出）：`channel-basic-section`（名称/类型/状态）、`channel-auth-section`（Key/BaseURL/组织）、`channel-models-section`（模型/映射/分组）、`channel-advanced-section`（优先级/权重/参数头覆盖/各开关）、`channel-api-access-section`（API 访问相关）、`channel-editor-loading-state`（加载态）。

## 关键库（`lib/` 与 `constants.ts`）

- **`lib/channel-type-config.ts`** — `CHANNEL_TYPE_CONFIGS: Record<number, ChannelTypeConfig>`，逐类型给 `icon`/`defaultBaseUrl`/`hints`（baseUrl/key/models）/`validation`（keyFormat/keyMinLength）/`requiresOrganization`/`requiresRegion` 等。
- **`constants.ts`** — `CHANNEL_TYPES`（**镜像 `constant/channel.go` 的 ChannelType**）、`CHANNEL_TYPE_OPTIONS`（含展示顺序）、`CHANNEL_STATUS*`、`MULTI_KEY_*`、`ADD_MODE_OPTIONS`、`MODEL_FETCHABLE_TYPES`（可从上游拉模型的类型集合）、`TYPE_TO_KEY_PROMPT`（各类型 Key 格式提示）、`CHANNEL_TYPE_WARNINGS`、`FIELD_PLACEHOLDERS`/`FIELD_DESCRIPTIONS`、`ERROR_MESSAGES`/`SUCCESS_MESSAGES`。**注意：这些 label/message 值都是英文 i18n key，展示时必须 `t(...)`。**
- **`types.ts`** — `Channel` 及各请求/响应参数类型（`GetChannelsParams`、`AddChannelRequest`、`MultiKeyStatusResponse`…）。

## 对话框群（`components/dialogs/`）

| 文件 | 一句话职责 |
| --- | --- |
| `channel-test-dialog.tsx` | 渠道连通性测试（选模型/端点类型/流式后调 `testChannel`） |
| `fetch-models-dialog.tsx` | 从上游拉取可用模型列表并回填 |
| `missing-models-confirmation-dialog.tsx` | 提交前对缺失模型的确认 |
| `multi-key-manage-dialog.tsx` | 多密钥管理（启用/禁用/删除/统计） |
| `multi-key-statistics-card.tsx` | 多密钥状态统计卡片 |
| `multi-key-table-row-actions.tsx` | 多密钥表内行操作 |
| `model-mapping` 相关（`model-mapping-editor.tsx`） | 模型映射 JSON 可视化编辑 |
| `param-override-editor-dialog.tsx` | 请求参数覆盖 JSON 编辑 |
| `advanced-custom-editor-dialog.tsx` | Advanced Custom(58) 路由配置编辑 |
| `status-code-risk-dialog.tsx` | 状态码映射风险提示确认 |
| `upstream-update-dialog.tsx` | 上游模型更新（检测/同步） |
| `balance-query-dialog.tsx` | 渠道余额查询 |
| `codex-usage-dialog.tsx` | Codex 渠道用量查看/重置 |
| `copy-channel-dialog.tsx` | 复制/克隆渠道 |
| `edit-tag-dialog.tsx` / `tag-batch-edit-dialog.tsx` | 编辑单个标签 / 标签批量编辑 |
| `ollama-models-dialog.tsx` | Ollama 模型管理 |

## 如何新增一个渠道配置项

**新增一个渠道类型（对接新 provider 的前端侧）**：

1. 在 `constants.ts` 的 `CHANNEL_TYPES` 补 `{id}: 'ProviderName'`（id 与 `constant/channel.go` 的 `ChannelType` 一致），并加入 `CHANNEL_TYPE_DISPLAY_ORDER`。
2. 在 `lib/channel-type-config.ts` 的 `CHANNEL_TYPE_CONFIGS` 补该 id 的 `icon`/`defaultBaseUrl`/`hints`/`validation`；如需特殊 Key 格式提示，补 `TYPE_TO_KEY_PROMPT`；如可拉模型，加入 `MODEL_FETCHABLE_TYPES`。
3. 若该类型需要特殊必填或校验，在 `channelFormSchema.superRefine` 内按 `data.type` 追加规则。

**新增一个表单字段**：

1. `channelFormSchema` 加字段 + `CHANNEL_FORM_DEFAULT_VALUES` 加默认值。
2. `transformChannelToFormDefaults` 解析回填；`buildSettingJSON` / `buildSettingsJSON`（或顶层字段）写回 API 载荷。
3. 把字段放进对应 `drawers/sections/*`；若属高级设置，加入 `channel-form-errors.ts` 的 `ADVANCED_SETTINGS_FIELDS`。

## 如何新增一个对话框

1. 在 `components/dialogs/` 新建 `xxx-dialog.tsx`（用 `@/components/ui` 的 `Dialog`/`Sheet`，文案 `t('English key')`）。
2. 在 `channels-provider.tsx` 的 `DialogType` 联合类型加上 `'xxx'`。
3. 在 `channels-dialogs.tsx` 挂载：当 `open === 'xxx'` 时渲染；触发处调 `setOpen('xxx')`（需要行数据时先 `setCurrentRow(row)`）。
4. 需要后端交互时在 `api.ts` 加封装函数，用 `useQuery`/`useMutation` 调用，变更后 `invalidateQueries(channelsQueryKeys.all)`。

## 国际化（i18n）

- 所有用户可见文案用 `const { t } = useTranslation()` + `t('English key')`（key 即英文源串）。
- `constants.ts` 中的 label/message/placeholder/description **只存英文 key**，组件渲染时再 `t(...)`，不要把常量当最终文案。
- 新增文案后在 `web/default/` 下运行 `bun run i18n:sync` 同步各语言 locale JSON（`en/zh/zh-TW/fr/ru/ja/vi`）。

## 常见问题 (FAQ)

- **Q：编辑渠道时 Key 输入框为空？** A：出于安全，Key 从不回填（`transformChannelToFormDefaults` 里 `key: ''`）；留空提交表示不改 Key。
- **Q：某些字段改了没保存成功？** A：type/key/base_url、param/header override、setting(s)/other 属敏感字段，需 `CHANNEL_SENSITIVE_WRITE` 权限；无权限会被 mutation 剔除。
- **Q：新增渠道类型前端下拉没有？** A：`constants.ts` 的 `CHANNEL_TYPES` 与 `CHANNEL_TYPE_DISPLAY_ORDER` 都要补；后端 `constant/channel.go` 也要有同号 `ChannelType`。
- **Q：改这里会被上游同步覆盖吗？** A：`web/default/` 是冲突面较小的前端主战场，但本 feature 上游也在演进；改动尽量集中、优先新增文件（如新对话框/新 section）。

## 相关文件清单

- 入口/装配：`index.tsx`、`components/channels-provider.tsx`、`components/channels-dialogs.tsx`
- 列表：`components/channels-table.tsx`、`components/channels-columns.tsx`
- 表单：`components/drawers/channel-mutate-drawer.tsx`、`lib/channel-form.ts`、`hooks/use-channel-mutate-form.ts`
- 配置/常量：`lib/channel-type-config.ts`、`constants.ts`、`types.ts`
- 请求：`api.ts` ↔ 后端 `controller/`（`/api/channel*`）、`constant/channel.go`（ChannelType 同步）

## 变更记录 (Changelog)

- **2026-07-09 23:51:43**：初始化渠道管理 feature 详尽文档（目录结构、数据流、表单体系、关键库、对话框群、新增配置项/对话框步骤、i18n 用法）。
