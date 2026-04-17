# Console Upstream Git Sync Strategy 设计文档

## 1. 背景

`console/` 当前已经有一套可运行的 Gateway 前端，但它的“上游界面来源”发生了变化。

此前仓库中的文档默认认为：

- 上游 UI 来自设计导出代码
- `console/src/design-source/` 是设计导出镜像
- 后续更新通过“重新同步设计源码”的方式完成

现在新的现实约束是：

- 上游前端样式和页面代码的唯一来源变成了 Git 仓库：`git@github.com:Code-Fight/Agentconsolewebsite.git`
- 该仓库会持续生成并更新一整套前端页面代码
- 当前仓库仍然需要把 Gateway HTTP / WebSocket / capability / host 路由逻辑接入这套上游 UI
- 后续同步必须允许“整包更新上游 UI”，同时尽量不破坏本地已经做好的 Gateway 适配

本次迭代的目标不是立刻完成所有运行时重构，而是先把“上游同步策略、接缝边界、AI Agent 执行契约”明确下来，并固化进仓库文档和同步清单中。

## 2. 目标

本次策略设计的目标如下：

1. 明确 `Agentconsolewebsite` 是 `console/` 的唯一上游界面来源
2. 保留“上游整包同步”的机制，不要求每次手工摘页面
3. 避免上游整包同步覆盖掉本地 Gateway 适配逻辑
4. 明确哪些目录允许被同步覆盖，哪些目录是本地保护区
5. 为后续 AI Agent 提供清晰、稳定、可重复执行的同步说明
6. 把详细接缝设计和 Gateway 复用清单记录到 `docs/`

## 3. 非目标

本次不直接完成以下事项：

- 不在本次文档迭代中完成所有运行时桥接代码迁移
- 不在本次文档迭代中重写当前 `console` 入口
- 不恢复旧的 Figma 导出同步流程
- 不继续使用任何 `scale` 方案

## 4. 核心原则

### 4.1 上游整包可替换

上游代码必须允许按固定路径整包替换，而不是依赖人工摘抄局部组件。

### 4.2 本地适配不可被覆盖

Gateway 集成、host 入口控制、路由、能力判断、真实 mutation、连接状态控制等逻辑必须留在本地保护区，不能随着上游整包同步一起被覆盖。

### 4.3 接缝集中管理

上游 UI 与 Gateway 逻辑之间的接缝必须集中在稳定的本地层中维护，不能散落回上游镜像层。

### 4.4 README 面向 AI 可执行

README 不是只给人类看的说明，而是 AI Agent 的执行契约。它必须清楚写明来源、允许覆盖的目录、禁止行为、同步步骤和验证要求。

## 5. 目录分层

本次策略定义四层边界。

### 5.1 `console/src/design-source/`

职责：

- 作为上游仓库 `Agentconsolewebsite` 的镜像层
- 尽量保持与上游目录结构一致
- 允许整包替换

允许同步的典型来源：

- `src/app/App.tsx`
- `src/app/components/**`
- `src/app/data/**`
- `src/app/utils/**`
- `src/styles/**`

禁止事项：

- 不要直接在这里写 Gateway fetch / websocket / capability 逻辑
- 不要把 host 连接控制写回这里
- 不要把“为了接通后端”而新增的本地状态长期留在这里

### 5.2 `console/src/design-bridge/`

职责：

- 作为本地稳定接缝层
- 把上游组件改造成稳定的视图接口
- 承接上游结构变化对本地适配的影响

要求：

- 所有“上游组件需要真实 props、真实动作、真实状态”的地方，优先在这里消化
- 后续同步上游代码时，应优先修这里，而不是改坏 `design-source`

说明：

- 本次迭代会先把这个目录作为正式保护区和后续迁移目标写进文档
- 当前运行时代码尚未完全迁移到该层，迁移会分阶段进行

### 5.3 `console/src/design-host/`

职责：

- 控制入口、路由、全局挂载、连接引导显示时机等宿主问题
- 只负责“何时显示某个上游 UI”以及“把哪个 bridge/view 挂进去”

### 5.4 `console/src/gateway/`

职责：

- 持有所有 Gateway HTTP / WebSocket / capability / view-model 适配
- 继续作为当前 Console 的真实数据来源

## 6. 详细接缝清单

### 6.1 `App`

上游现状：

- `Agentconsolewebsite/src/app/App.tsx` 自己维护页面切换、本地线程、本地机器、本地环境资源、本地连接状态

本地策略：

- 上游 `App` 只应保留为布局和视觉结构来源
- 不应继续作为运行时真相源
- 本地运行时状态应由 `design-host` 和 `gateway` 提供

### 6.2 `SetupWizard`

上游现状：

- 自己维护 `consoleUrl`、`apiKey`
- 自己做模拟连接测试
- 自己调用 `localStorage`

本地策略：

- 保留其视觉结构、字段布局、测试中/成功/失败状态样式
- 去掉本地存储和模拟测试
- 改为由宿主提供：
  - 当前输入值
  - 当前测试状态
  - 错误文案
  - 保存动作
  - 测试动作

### 6.3 `Settings`

上游现状：

- 自己维护 API 配置和 Agent 默认配置编辑状态

本地策略：

- 保留页面结构和样式
- 以下真实数据继续来自本地 Gateway 适配：
  - 全局默认配置
  - 机器覆盖配置
  - 配置下发
  - Console preferences
  - 连接状态展示

### 6.4 `MachinePanel` / `ThreadItem`

上游现状：

- 线程面板和线程项基于本地 mock 数据工作

本地策略：

- 保留左侧线程栏样式和交互结构
- 真实数据和动作来自：
  - `useThreadHub`
  - 路由导航
  - 本地 host 选择状态

### 6.5 `SessionChat`

上游现状：

- 工作区样式来自上游
- 线程消息和执行流仍是本地 mock

本地策略：

- 保留消息流、输入区、工作区布局样式
- 真实消息、turn 状态、审批卡片、`steer`、`interrupt` 全部继续来自 `useThreadWorkspace`

### 6.6 `Machines`

本地策略：

- 保留页面样式结构
- 真实机器列表、runtime 状态、安装/删除 Agent、编辑配置等动作继续来自 Gateway 适配

### 6.7 `Environment`

本地策略：

- 保留页面样式结构
- skills / mcps / plugins 数据和动作继续来自 Gateway 适配

### 6.8 上游存在但当前未接入主路径的组件

以下组件会随着整包同步进入镜像层，但当前不直接作为 active Console 主路径的一部分：

- `Agents`
- `Dashboard`
- `Logs`
- `Requests`

策略：

- 先镜像保留
- 不把它们误认为当前必须接通的产品主路径
- 若后续产品需要，再单独设计其 Gateway 接入

## 7. Gateway 复用清单

当前可以直接保留的本地适配包括：

- `console/src/common/api/http.ts`
- `console/src/common/api/ws.ts`
- `console/src/gateway/capabilities.ts`
- `console/src/gateway/gateway-connection-store.ts`
- `console/src/gateway/use-thread-hub.ts`
- `console/src/gateway/use-thread-workspace.ts`
- `console/src/gateway/use-settings-page.ts`
- `console/src/gateway/use-console-preferences.ts`
- `console/src/gateway/use-machines-page.ts`
- `console/src/gateway/use-environment-page.ts`

说明：

- 上述逻辑不应因上游样式仓库替换而被推倒
- 真正会变化的是“如何把这些 view-model 和动作接进新的上游组件”
- 这部分变化应集中在 `design-bridge` 与 `design-host`

## 8. 上游同步来源与路径映射

上游仓库：

- `git@github.com:Code-Fight/Agentconsolewebsite.git`

建议同步映射：

- `src/app/App.tsx` -> `console/src/design-source/App.tsx`
- `src/app/components/**` -> `console/src/design-source/components/**`
- `src/app/data/**` -> `console/src/design-source/data/**`
- `src/app/utils/**` -> `console/src/design-source/utils/**`
- `src/styles/**` -> `console/src/design-source/styles/**`

这套映射将同时写入机器可读的 `console/upstream-sync.manifest.json`。

## 9. 保护区与禁止覆盖目录

同步上游代码时，不允许覆盖以下本地保护区：

- `console/src/design-bridge/`
- `console/src/design-host/`
- `console/src/gateway/`
- `console/src/common/api/`
- `console/src/pages/`
- `console/tests/`
- `testing/`
- `testenv/`

## 10. AI Agent 标准同步流程

### 10.1 读取同步契约

先读：

- `console/README.md`
- `console/upstream-sync.manifest.json`

### 10.2 拉取上游代码

- 把 `Agentconsolewebsite` 拉到临时目录
- checkout 目标分支或 commit

### 10.3 只同步镜像层

- 严格按照 manifest 中的映射，把上游文件同步到 `console/src/design-source/`
- 不要手动扩大覆盖范围

### 10.4 检查关键接缝

同步后必须检查以下关键组件是否仍然存在或发生明显结构变化：

- `App`
- `SetupWizard`
- `Settings`
- `MachinePanel`
- `ThreadItem`
- `SessionChat`
- `Machines`
- `Environment`

### 10.5 只在本地层修接缝

如果上游变化影响了集成：

- 优先修 `design-bridge`
- 其次修 `design-host`
- 必要时更新 `gateway` view-model 映射
- 不要为了快速接通把本地逻辑直接补回 `design-source`

## 11. 验证清单

每次同步后至少执行：

```bash
cd console
corepack pnpm test
corepack pnpm build
```

如涉及联调或关键页面回归，再执行：

```bash
cd console
corepack pnpm e2e
cd ..
./testing/environments/settings-e2e/run.sh
```

## 12. README 更新要求

README 必须反映以下事实：

1. 上游来源改为 Git 仓库，不再是 Figma 导出说明
2. 不再使用 `scale` 方案
3. `design-source` 是镜像层，不是本地适配层
4. `design-bridge` 是未来稳定接缝层
5. AI Agent 必须按 manifest 和 README 的同步契约执行

## 13. 一句话总结

未来的同步原则应当是：

`整包更新上游镜像层 -> 检查接缝 -> 只修本地桥接/宿主/网关适配层`

而不是：

`整包覆盖 -> 到处修运行时代码`
