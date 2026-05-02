# Agent Timeline Event 通用协议实施计划

> **给后续执行 Agent 的要求：** 按任务逐步执行。推荐使用 `superpowers:subagent-driven-development`，也可以使用 `superpowers:executing-plans`。每个任务使用 checkbox 追踪进度。

**目标：** 用 provider-neutral 的 `AgentTimelineEvent` 替换当前过于简化的 `turn.delta` 主链路，从 client 到 Gateway 再到 Console 完整保留 turn 生命周期、消息 delta、推理摘要、工具调用、审批、文件/产物、错误和完成状态。

**架构：** 在 `common/domain` 增加稳定的 `AgentTimelineEvent` 产品模型，并通过 `timeline.event` 传输。Codex 原生协议映射只放在 `client/internal/agent/codex`；Gateway 只路由和索引标准化事件；Console 基于 timeline 聚合 UI，不直接依赖 Codex item 名称，也不再依赖临时 `content/progress` 文本判断。过渡期保留旧的 `turn.started`、`turn.delta`、`turn.completed`、`turn.failed`、`approval.required`、`approval.resolved`。

**技术栈：** Go shared domain/protocol，Go client/gateway WebSocket transport，React + TypeScript Console，Vitest/React Testing Library，Go unit tests，现有 dev integration 脚本。

## 实施记录

2026-04-29 已完成主链路落地：

- `common` 增加 `AgentTimelineEvent` 通用定义、timeline payload、thread message 兼容字段和 `agentTimelineEvents` capability。
- `client` 增加 `RuntimeTimelineEventSource`，Codex adapter 将 turn、item、delta、reasoning、tool、approval、error 映射为 `timeline.event`，并保留旧 turn/approval 事件兼容。
- `gateway` 支持 timeline 事件路由、active turn 状态维护、审批归一化、thread 级历史事件索引和 `/threads/:id` events 返回。
- `console` 优先消费 timeline 事件，按 turn 聚合为一个 Agent 容器，最终输出 Markdown 渲染，过程/工具/终端输出进入可折叠处理区，旧事件仍回退兼容。
- 已补充 Go、Vitest、页面级 UI 和 Playwright smoke 覆盖。真实 Docker dev-integration 联调未在本轮启动，避免改动当前本地 14173 运行环境；需要时可单独执行。

---

## 阶段一：公共协议模型

### 任务 1：增加共享 Timeline 领域类型

**文件：**
- 新建：`common/domain/agent_timeline.go`
- 修改：`common/domain/thread.go`
- 测试：`common/domain/agent_timeline_test.go`

- [ ] 定义 `AgentTimelineEvent`、`AgentTimelineEventType`、`AgentTimelineItemType`、`AgentTimelinePhase`、`AgentTimelineStatus`、`AgentTimelineContent`、`AgentTimelineTool`、`AgentTimelineApproval`、`AgentTimelineError`、`AgentTimelineRaw`。
- [ ] 增加基础校验 helper：`eventId`、`sequence`、`threadId`、`eventType` 必填；turn 和 item 事件必须有 `turnId`。
- [ ] 为历史 `ThreadMessage` 增加兼容用的可选结构化字段：
  - `Phase string json:"phase,omitempty"`
  - `ContentType string json:"contentType,omitempty"`
  - `ProgressText string json:"progressText,omitempty"`
- [ ] 单测覆盖最终消息、推理摘要、命令工具、文件变更、审批、未知 raw 事件的 JSON 序列化和校验。

### 任务 2：增加 Timeline 协议 Payload

**文件：**
- 修改：`common/protocol/messages.go`
- 测试：`common/protocol/messages_test.go`

- [ ] 增加 `TimelineEventPayload struct { Event domain.AgentTimelineEvent json:"event" }`。
- [ ] 如 Gateway 历史重建测试需要，再增加 `TimelineSnapshotPayload struct { ThreadID string json:"threadId"; Events []domain.AgentTimelineEvent json:"events" }`。
- [ ] 保持现有 turn/approval payload 不变，用于兼容。
- [ ] 测试 `Envelope.Validate` 能继续接受 `Name: "timeline.event"` 的 event envelope。

---

## 阶段二：Client Runtime 抽象

### 任务 3：升级 Runtime 事件接口

**文件：**
- 修改：`client/internal/agent/types/interfaces.go`
- 修改：`client/cmd/client/main.go`
- 测试：`client/cmd/client/main_test.go`

- [ ] 增加 `RuntimeTimelineEventSource interface { SetTimelineEventHandler(func(domain.AgentTimelineEvent)) }`。
- [ ] 临时保留 `RuntimeTurnEventSource`，让旧 fake runtime 和兼容 runtime 继续工作。
- [ ] 增加 `clientSession.TimelineEvent(requestID string, payload protocol.TimelineEventPayload) error`。
- [ ] 优先绑定 `RuntimeTimelineEventSource`；如果 runtime 不支持，则把旧 `RuntimeTurnEventSource` 桥接为 timeline event。
- [ ] 过渡期同时发送 `timeline.event` 和旧 turn 事件。
- [ ] 单测：runtime timeline final message 会同时发出 `timeline.event` 和兼容 `turn.delta`。
- [ ] 单测：`turn.completed` 清理 active 状态，并且只触发一次 thread snapshot refresh。

### 任务 4：标准化 Codex 原生 Notification

**文件：**
- 修改：`client/internal/agent/codex/appserver_client.go`
- 新建：`client/internal/agent/codex/timeline_mapper.go`
- 测试：`client/internal/agent/codex/appserver_client_test.go`
- 测试：`client/internal/agent/codex/timeline_mapper_test.go`

- [ ] 为 timeline event 维护独立的 per-turn sequence，不和旧 delta sequence 混用。
- [ ] 映射 Codex lifecycle：
  - `turn/started` -> `turn.started`
  - `turn/completed` -> `turn.completed` 或 `turn.failed`
  - 非 retry 的 `error` -> 带上下文的 `item.failed` 或 `turn.failed`
- [ ] 映射 Codex item lifecycle：
  - `item/started` -> `item.started`
  - `item/completed` -> `item.completed`
- [ ] 映射文本 delta：
  - `item/agentMessage/delta` + `phase=final_answer` -> `item.delta`、`itemType=message`、`phase=final`、`contentType=markdown`
  - `phase=commentary` -> `item.delta`、`itemType=message`、`phase=progress`
  - 缺失 phase -> 兼容为 `phase=final`
- [ ] 映射 reasoning：
  - `item/reasoning/summaryTextDelta` -> `item.delta`、`itemType=reasoning`、`phase=analysis`、`contentType=markdown`
  - `item/reasoning/textDelta` 默认只进入 raw/debug，不进主 UI，除非后续明确允许展示。
- [ ] 映射所有 Codex `ThreadItem.type`：
  - `userMessage` -> `message`、`role=user`、`phase=input`
  - `hookPrompt` -> `message`、`role=system`、`phase=system`
  - `agentMessage` -> `message`
  - `plan` -> `plan`、`phase=progress`
  - `reasoning` -> `reasoning`、`phase=analysis`
  - `commandExecution` -> `command`、`tool.kind=shell`
  - `fileChange` -> `file_change`、`tool.kind=file_edit`
  - `mcpToolCall` -> `mcp_tool`、`tool.kind=mcp`
  - `dynamicToolCall` -> `tool`、`tool.kind=function`
  - `collabAgentToolCall` -> `subagent`、`tool.kind=subagent`
  - `webSearch` -> `web_search`、`tool.kind=web_search`
  - `imageView` -> `image`
  - `imageGeneration` -> `image`、`tool.kind=image_generation`
  - `enteredReviewMode` / `exitedReviewMode` -> `mode_change`
  - `contextCompaction` -> `context`
  - 未知类型 -> `unknown`，并保留 `raw.provider=codex`
- [ ] 移除 Codex mapper 中注入的 `正在分析...`，执行状态由 lifecycle 驱动。
- [ ] 单测覆盖每一种 Codex item type。
- [ ] 单测覆盖两轮对话：第一轮 `hi`，第二轮调研 prompt；所有 delta 都绑定正确 `turnId`，第一轮最终回答不被覆盖。

### 任务 5：标准化审批事件

**文件：**
- 修改：`client/internal/agent/codex/approvals.go`
- 修改：`client/internal/agent/codex/appserver_client.go`
- 修改：`client/cmd/client/main.go`
- 测试：`client/internal/agent/codex/appserver_client_test.go`
- 测试：`client/cmd/client/main_test.go`

- [ ] 对 command、file change、permissions、dynamic tool user input、MCP elicitation、legacy approval methods 发出 `approval.requested` timeline event。
- [ ] 在收到 `serverRequest/resolved` 或本地响应成功后，发出 `approval.resolved` timeline event。
- [ ] 保留旧 `approval.required` 和 `approval.resolved` envelope，保证现有 Console 兼容。
- [ ] 对用户输入审批填充 `approval.questions`，对命令/文件变更填充 `tool.input`。

---

## 阶段三：Gateway 路由与索引

### 任务 6：ClientHub 和 ConsoleHub 支持 Timeline 事件

**文件：**
- 修改：`gateway/internal/websocket/client_hub.go`
- 修改：`gateway/internal/websocket/console_hub.go`
- 测试：`gateway/internal/websocket/client_hub_test.go`
- 测试：`gateway/internal/websocket/console_hub_test.go`

- [ ] 通过 `payload.event.threadId` 将 `timeline.event` 纳入 thread-filtered event routing。
- [ ] 根据 `eventType=turn.started|turn.completed|turn.failed` 更新 active turn 状态。
- [ ] 对 `timeline.event.approval.requestId` 使用和旧审批 envelope 相同的 requestId 归一化逻辑。
- [ ] 保证旧 `turn.completed` 和新 `timeline.event(turn.completed)` 同时出现时不会错误重复清理 active 状态。
- [ ] 单测：订阅 thread A 的 Console 只收到 thread A 的 timeline event。
- [ ] 单测：只有 timeline completion 事件时，active 状态也能清理。

### 任务 7：Runtime Index 支持 Timeline 状态

**文件：**
- 修改：`gateway/internal/registry/store.go`
- 修改：`gateway/internal/api/server.go`
- 测试：`gateway/internal/registry/store_test.go`
- 测试：`gateway/internal/api/server_test.go`

- [ ] 按 thread 保存最近一段 timeline events，用于刷新或重新打开工作区时重建 UI。
- [ ] 对每个 thread 设置 event 数量上限，保持 Gateway 轻状态。
- [ ] 在线程详情响应中增加 `events?: AgentTimelineEvent[]`，同时保留 `messages` 兼容字段。
- [ ] pending approvals 同时来自 timeline approval event 和旧 registry。
- [ ] 单测：`GET /threads/:id` 同时返回历史 `events` 和现有 `messages`。

---

## 阶段四：Console 类型与工作区模型

### 任务 8：增加前端 Timeline 类型

**文件：**
- 修改：`console/src/common/api/types.ts`
- 新建：`console/src/features/threads/model/timeline-model.ts`
- 测试：`console/src/features/threads/model/timeline-model.test.ts`

- [ ] 在 TypeScript 中镜像共享 timeline event 类型。
- [ ] 增加解析 helper：
  - `isTimelineEventEnvelope`
  - `getTimelineThreadId`
  - `timelineEventToWorkspacePatch`
- [ ] 兼容期保留旧 `TurnDeltaPayload` 类型。
- [ ] 测试 final markdown、progress commentary、reasoning summary、command output、file change、MCP tool、web search、approval、unknown event 的映射。

### 任务 9：用 Timeline 重建 Workspace 状态

**文件：**
- 修改：`console/src/features/threads/hooks/use-thread-workspace.ts`
- 修改：`console/src/features/threads/model/thread-view-model.ts`
- 测试：`console/src/features/threads/hooks/use-thread-workspace.test.tsx`
- 测试：`console/src/features/threads/model/thread-view-model.test.ts`

- [ ] Workspace message 以 `turnId + itemId` 作为聚合 key。
- [ ] 同一个 item 的所有 final message delta 合并成一个 agent bubble。
- [ ] progress/reasoning/tool event 合并到同一 turn bubble 的可折叠过程模型中。
- [ ] 执行中状态由 optimistic submit 和 `turn.started` 立即触发，不依赖合成文本。
- [ ] 收到 `turn.completed|failed` 后清理 `activeTurnId`，并把过程区标记为完成。
- [ ] 保留旧 `turn.delta` handler，把它转换为 timeline-style patch。
- [ ] 单测用户真实例子：
  - 发送 `hi`
  - 收到最终 `hi`
  - 发送 `你调研一下，目前最新的Agent助理有哪些？做一份总结报告。`
  - 执行 indicator 挂到第二轮 turn
  - 最终输出保持在一个 Markdown 气泡
  - 第一轮回答不变
- [ ] 单测 late `turn.completed` 能清理线程 active 状态。

### 任务 10：渲染 Timeline 感知的 Chat UI

**文件：**
- 修改：`console/src/features/threads/components/session-chat.tsx`
- 测试：`console/src/features/threads/components/session-chat.test.tsx`
- 测试：`console/src/features/threads/pages/thread-workspace-page.test.tsx`

- [ ] `phase=final` 内容作为主气泡 Markdown/GFM 渲染。
- [ ] 过程内容渲染为可折叠块：
  - 执行中 label：`处理中`
  - 完成后 label：`已处理`
  - 内含 reasoning summary、progress、tools、command output、web search、file changes、approvals
- [ ] 不把 `turn.started`、`turn.completed` 或合成 lifecycle message 渲染成普通聊天 pill。
- [ ] command output 使用终端样式，file changes 使用文件变更块。
- [ ] unknown timeline item 只有在有可展示文本时才渲染为紧凑 debug row。
- [ ] 单测/视觉测试覆盖 Markdown 表格、列表、链接、inline code、code block、过程折叠。

### 任务 11：更新线程列表 Active 状态

**文件：**
- 修改：`console/src/features/threads/hooks/use-thread-hub.ts`
- 修改：`console/src/features/machines/hooks/use-machines-page.ts`
- 测试：`console/src/features/threads/hooks/use-thread-hub.test.tsx`
- 测试：`console/src/features/machines/pages/machines-page.test.tsx`

- [ ] `timeline.event(turn.started)` 视为 active。
- [ ] `timeline.event(turn.completed|turn.failed)` 视为 idle。
- [ ] 保留旧 lifecycle handling。
- [ ] 单测：仅使用 timeline events 时，turn 完成后线程列表不再显示 active。

---

## 阶段五：兼容、联调与验证

### 任务 12：兼容发布策略

**文件：**
- 修改：`common/version/version.go`
- 修改：`console/src/common/config/capabilities.ts`
- 测试：`common/version/version_test.go`
- 测试：`console/src/common/config/capabilities.test.ts`

- [ ] 增加 capability flag，例如 `agentTimelineEvents`。
- [ ] Gateway 能同时广播旧事件和 timeline event。
- [ ] Console 优先消费 timeline event，没有 timeline 时回退到 legacy。
- [ ] 文档标记旧 `turn.delta.kind=content|progress` 为 deprecated，但仍在过渡期支持。

### 任务 13：端到端 Case 覆盖

**文件：**
- 修改或新增现有测试目录中的测试：
  - `client/internal/agent/codex/appserver_client_test.go`
  - `gateway/internal/websocket/client_hub_test.go`
  - `gateway/internal/api/server_test.go`
  - `console/src/features/threads/pages/thread-workspace-page.test.tsx`
  - `console/src/features/threads/hooks/use-thread-workspace.test.tsx`

- [ ] Case：最终 Markdown 回答流式输出后只形成一个气泡。
- [ ] Case：progress/reasoning 执行中展开，完成后折叠。
- [ ] Case：command execution output 归入过程区。
- [ ] Case：file change 归入过程区。
- [ ] Case：web search / MCP / dynamic tool / subagent 归入过程区。
- [ ] Case：approval request 和 resolution 显示在同一 turn 过程里。
- [ ] Case：failed turn 清理 active 状态并显示错误。
- [ ] Case：unknown provider item 保留 `raw`，且不破坏 UI。
- [ ] Case：两轮真实例子中，执行 indicator 始终挂在正确 turn。

### 任务 14：完整验证

**命令：**
- `go test ./common/... ./gateway/... ./client/...`
- `cd console && corepack pnpm test`
- `cd console && corepack pnpm build`
- `cd console && corepack pnpm e2e`
- `./testing/environments/dev-integration/run.sh up`

- [ ] 运行 Go 单元测试。
- [ ] 运行 Console 单元测试。
- [ ] 运行 Console build。
- [ ] 运行可用的 e2e。
- [ ] 增加或更新 Console UI 测试夹具，让 `session-chat` / thread workspace 能直接注入一组完整 timeline events。
- [ ] 用 UI 测试夹具覆盖消息展示矩阵：
  - `userMessage` / `message + phase=input`：用户消息右侧气泡。
  - `agentMessage + phase=final`：Agent 主气泡，Markdown/GFM 正常渲染，包括标题、列表、表格、链接、inline code、code block。
  - `agentMessage + phase=progress`：显示在过程区，执行中展开，完成后折叠。
  - `reasoning` / `reasoning summary`：显示在“处理中/已处理”折叠框里，不进入最终回答正文。
  - `plan`：显示为过程内容，不渲染成独立聊天消息。
  - `commandExecution`：显示终端输出块，多行输出、PASS/FAIL/Error 文本样式和横向滚动正常。
  - `fileChange`：显示文件变更块，路径、增删数量、展开/收起状态正常。
  - `webSearch`：显示搜索过程块，query、URL、开始/完成状态正常。
  - `mcpToolCall`：显示 MCP 工具块，server、tool、status、result/error 正常。
  - `dynamicToolCall`：显示通用工具块，工具名、参数摘要、结果摘要正常。
  - `collabAgentToolCall`：显示子 Agent 调用块，目标 Agent、状态、返回结果正常。
  - `imageView` / `imageGeneration`：显示图片/产物块，生成状态、路径或预览占位正常。
  - `approval.requested` / `approval.resolved`：审批卡片展示问题、选项、决策和完成态正常。
  - `contextCompaction`：显示紧凑系统事件，不污染主回答。
  - `enteredReviewMode` / `exitedReviewMode`：显示紧凑模式变化事件。
  - `unknown`：页面不崩溃；有文本时显示紧凑 debug row，无文本时不产生空气泡。
- [ ] 增加 UI 状态验收：
  - active turn 立即显示“正在执行/处理中”。
  - 第一个真实内容到达后不保留无意义的占位文本。
  - turn completed 后隐藏“正在执行”，过程区变成“已处理”并默认折叠。
  - failed turn 清理 active 状态并显示错误。
  - 同一个 turn 的多个 progress/tool/reasoning item 都挂在同一个 Agent 容器下。
  - 同一 item 的流式 delta 不拆成多个对话框。
  - 从 `GET /threads/:id` 历史 events 恢复出来的 UI 与实时流式 UI 一致。
  - 旧 `turn.delta` fallback 不和 `timeline.event` 重复渲染。
- [ ] 使用 in-app browser 真实验证两轮 case：
  - 第一轮 prompt：`hi`
  - 第二轮 prompt：`你调研一下，目前最新的Agent助理有哪些？做一份总结报告。`
  - 验证立即显示执行中、最终回答流式进入一个气泡、完成后过程折叠、线程列表状态回到空闲。

---

## 注意事项

- 不展示 provider 私有 chain-of-thought。Reasoning 只展示明确可显示的 summary/progress。
- 不把 Codex 专有名称泄漏到 Gateway 或 Console，除非放在 `raw.provider` 这类调试字段里。
- 本次改造不删除旧事件支持。旧协议清理应在 timeline 路径稳定后另开任务。
- 当前 worktree 已经是 dirty。实施前必须检查目标文件里的现有改动，并在其基础上继续，不要回滚用户或已有生成改动。
