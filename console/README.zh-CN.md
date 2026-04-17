# Console

`console/` 现在采用的是 `upstream-source first` 策略。

当前 Console 的上游界面代码不再来自 Figma 导出同步流程，而是来自这个 Git 仓库：

- `git@github.com:Code-Fight/Agentconsolewebsite.git`

`src/design-source/` 仍然保留为上游镜像层，但它现在镜像的是上游 Git 仓库中的前端代码，而不是设计导出源码。当前仓库继续在镜像层之外承载 Gateway 接入、路由、宿主控制，以及后续用于稳定接缝的本地适配层。

## 唯一上游来源

- 上游 UI 仓库：`git@github.com:Code-Fight/Agentconsolewebsite.git`
- 机器可读同步契约：`console/upstream-sync.manifest.json`
- 详细策略与接缝清单：`docs/superpowers/specs/2026-04-18-console-upstream-git-sync-design.md`

旧的 Figma 导出假设、`design-source-sync` 工作流，以及任何 `scale` 方案，都已经不再适用于当前 Console，后续同步时不能再使用。

## 架构分层

当前策略定义四层边界：

1. `src/design-source/`
   上游镜像层。这里是上游 UI 仓库在当前仓库中的镜像，预期允许按整包替换。
2. `src/design-bridge/`
   本地受保护接缝层。这里用于放稳定的本地桥接逻辑，用来把上游组件改造成可持续接入 Gateway 的视图接口。
3. `src/design-host/`
   宿主控制层。入口挂载、全局引导、路由宿主控制等问题放在这里。
4. `src/gateway/`
   Gateway 接入层。HTTP、WebSocket、capability policy、transport helper、view-model 组装都放在这里。

`src/common/api/` 和 `src/pages/` 仍然属于本地实现区域，不属于上游镜像层。

## 当前状态

当前运行时仍然通过 `src/design-host/` 挂载 active Console，并复用本地 Gateway hooks。本次策略迭代只是正式引入 `src/design-bridge/` 作为长期受保护的接缝层，当前运行时代码尚未全部迁移到该层。后续新增适配时，应当朝这个方向迁移，而不是继续把业务逻辑塞回 `src/design-source/`。

## 允许写入的区域

同步上游代码时，只允许覆盖 `console/upstream-sync.manifest.json` 中声明的镜像目标。

本地适配应写在这些目录中：

- `src/design-bridge/`
- `src/design-host/`
- `src/gateway/`
- `src/common/api/`
- `src/pages/`
- 测试

## 受保护目录

同步上游 UI 代码时，禁止覆盖以下路径：

- `console/src/design-bridge/`
- `console/src/design-host/`
- `console/src/gateway/`
- `console/src/common/api/`
- `console/src/pages/`
- `console/tests/`
- `testing/`
- `testenv/`

## AI 同步契约

任何 AI Agent 在更新上游 UI 代码时，都必须按以下流程执行：

1. 先读本 README 和 `console/upstream-sync.manifest.json`
2. 把 `git@github.com:Code-Fight/Agentconsolewebsite.git` 拉到临时目录
3. checkout 目标分支或 commit
4. 严格按照 `console/upstream-sync.manifest.json` 中的路径映射同步镜像层
5. 不允许擅自扩大覆盖范围
6. 同步后必须检查 manifest 中列出的关键接缝组件：
   - `App`
   - `SetupWizard`
   - `Settings`
   - `MachinePanel`
   - `ThreadItem`
   - `SessionChat`
   - `Machines`
   - `Environment`
7. 如果上游结构变化影响本地运行时接入：
   - 优先修 `src/design-bridge/`
   - 其次修 `src/design-host/`
   - 只有在本地 view-model 契约真的变化时，才更新 `src/gateway/`
8. 不要为了“先接通再说”把 Gateway 逻辑、mock 清理、连接流程或路由控制直接补进 `src/design-source/`

## 禁止事项

在同步和集成过程中，不要做这些事情：

- 不要继续使用旧的 Figma 导出同步流程
- 不要继续依赖 `design-source-sync` skill 处理当前 Console
- 不要再使用任何 `scale` 方案
- 不要把 Gateway HTTP / WebSocket 逻辑写进 `src/design-source/`
- 不要把上游 mock 运行时状态继续保留在 active runtime 主路径中
- 不要在镜像替换时覆盖本地保护区

## 上游同步流程

标准同步链路应当是：

`拉取上游仓库 -> 只替换镜像目标 -> 检查关键接缝组件 -> 如有必要只修 bridge/host 接缝 -> 重新验证`

这就是当前要区分“上游镜像层”和“本地保护层”的原因。

## 验证

每次同步上游代码后，至少执行：

```bash
cd console
corepack pnpm test
corepack pnpm build
```

如果变更影响到关键页面或交互流程，再执行：

```bash
cd console
corepack pnpm e2e
cd ..
./testing/environments/settings-e2e/run.sh
```

## Manifest 说明

`console/upstream-sync.manifest.json` 是 AI Agent 的机器可读同步契约，里面记录：

- 上游仓库地址
- 上游路径到本地镜像目标的映射
- 本地受保护目录
- 关键接缝组件
- 必跑验证命令

AI Agent 在执行同步时，应把 manifest 视为本 README 的执行版配套文件。
