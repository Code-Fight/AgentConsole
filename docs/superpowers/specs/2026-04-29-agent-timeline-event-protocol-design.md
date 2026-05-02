# Agent Timeline Event 通用协议技术方案

## 1. 背景

当前系统的实时输出主要通过 `turn.started`、`turn.delta`、`turn.completed`、`turn.failed`、`approval.required`、`approval.resolved` 这几类事件传递。这个模型可以支撑简单流式文本，但已经暴露出几个问题：

- `turn.delta.kind` 只有 `content` 和 `progress`，无法完整表达工具调用、推理摘要、文件变更、MCP 调用、子 Agent、图片生成等结构化过程。
- Console 只能靠文本和少量 `kind` 判断“最终输出”和“过程输出”，容易把正在执行状态挂到错误的回复上。
- Codex App Server 已经提供更完整的 item/notification 协议，但这些细节不应该直接泄漏到 Gateway 和 Console。
- 项目目标是 Codex-first，但架构要给其他 Agent 预留稳定抽象，因此不能把 Codex 的 `ThreadItem` 当成公共协议。

因此需要新增一套通用的 Agent 时间线事件模型，作为 client -> Gateway -> Console 的主协议。

## 2. 设计目标

本方案目标：

- 用统一模型表达一次 Agent turn 中发生的所有重要事件。
- 覆盖最终回答、过程输出、推理摘要、计划、工具调用、命令输出、文件变更、审批、错误、上下文压缩、图片和未知扩展事件。
- 让 Gateway 和 Console 只理解产品化通用模型，不理解 Codex 原生协议。
- Console 能把一次 turn 的多个事件聚合成一个稳定的回复容器，最终输出和处理过程分层展示。
- 保留旧协议一段时间，避免一次性破坏现有功能和测试。

非目标：

- 不在 Gateway 中实现 Codex 专有逻辑。
- 不把 provider 私有 chain-of-thought 当作最终 UI 内容展示。
- 不在本次改造中移除旧 `turn.delta` 兼容事件。
- 不一次性接入 Claude、LangGraph、Google ADK、AutoGen 的真实 runtime，只验证抽象能覆盖它们的事件模型。

## 3. 外部 Agent 协议调研结论

主流 Agent 框架的命名不同，但核心事件可以归为同一组概念：

- OpenAI Codex App Server：`turn/*`、`item/*`、`ThreadItem.type`、server request、approval、reasoning、tool、MCP、file change。
- Anthropic Claude / Claude Code：消息流、content block、tool use、tool result、thinking/reasoning、usage、error。
- LangGraph：updates/messages/custom/debug 等 stream mode，可表达节点更新、消息 token、工具过程和自定义事件。
- Google ADK：以 event/action 表达 agent 执行、工具调用、长任务、artifact、state change。
- Microsoft AutoGen：消息、工具调用、工具结果、handoff、停止条件和流式事件。

这些系统都能映射为：

1. run/turn 生命周期
2. item/step 生命周期
3. 文本或结构化 delta
4. 工具调用和工具结果
5. 人工审批或人工输入
6. 推理摘要或处理过程
7. 文件、图片、artifact
8. 错误、取消、中断
9. provider 原始扩展字段

所以公共协议应抽象这些稳定概念，而不是复制某一家 provider 的字段。

## 4. 核心抽象：AgentTimelineEvent

`AgentTimelineEvent` 是“Agent 运行过程中的一条标准时间线事件”。它不是最终聊天消息，而是 UI、索引和状态机的原始标准输入。

建议结构：

```ts
type AgentTimelineEvent = {
  schemaVersion: "agent-timeline.v1";
  eventId: string;
  sequence: number;
  timestamp: string;

  machineId: string;
  agentId: string;
  threadId: string;
  turnId?: string;
  itemId?: string;

  eventType:
    | "turn.started"
    | "turn.completed"
    | "turn.failed"
    | "item.started"
    | "item.delta"
    | "item.completed"
    | "item.failed"
    | "approval.requested"
    | "approval.resolved"
    | "system.event";

  itemType?:
    | "message"
    | "reasoning"
    | "plan"
    | "tool"
    | "command"
    | "file_change"
    | "web_search"
    | "browser_action"
    | "mcp_tool"
    | "subagent"
    | "artifact"
    | "image"
    | "context"
    | "mode_change"
    | "unknown";

  role?: "user" | "assistant" | "system" | "tool";
  phase?: "input" | "analysis" | "progress" | "final" | "system";
  status?: "pending" | "running" | "blocked" | "completed" | "failed" | "declined" | "cancelled";

  content?: {
    contentType: "markdown" | "text" | "json" | "terminal" | "diff" | "image" | "file";
    delta?: string;
    text?: string;
    snapshot?: unknown;
    appendMode?: "append" | "replace" | "snapshot";
  };

  tool?: {
    kind: "shell" | "web_search" | "browser" | "mcp" | "function" | "file_edit" | "subagent" | "image_generation" | "unknown";
    name?: string;
    input?: unknown;
    output?: unknown;
  };

  approval?: {
    requestId: string;
    kind: string;
    title?: string;
    reason?: string;
    questions?: Array<{ id: string; label: string; options?: string[] }>;
    decision?: "accept" | "decline" | "cancel";
  };

  error?: { message: string; code?: string };
  raw?: { provider: string; method?: string; payload?: unknown };
};
```

字段语义：

- `eventType` 表示发生了什么生命周期事件。
- `itemType` 表示事件属于哪类内容或步骤。
- `phase` 表示 UI 应如何归类：最终输出、处理中、分析、系统事件或用户输入。
- `content` 表示可渲染内容，支持增量、快照和替换。
- `tool` 表示工具调用或工具结果。
- `approval` 表示审批请求和审批结果。
- `raw` 保留 provider 原始信息，用于调试和未来扩展，但 Gateway/Console 不应依赖它做主逻辑。

## 5. Codex 映射规则

Codex 原生事件只在 `client/internal/agent/codex` 中解析并映射为 `AgentTimelineEvent`。

Codex lifecycle 映射：

- `turn/started` -> `eventType=turn.started`
- `turn/completed` -> `eventType=turn.completed` 或 `turn.failed`
- `item/started` -> `eventType=item.started`
- `item/completed` -> `eventType=item.completed`
- `item/agentMessage/delta` -> `eventType=item.delta`
- `item/reasoning/summaryTextDelta` -> `item.delta + itemType=reasoning + phase=analysis`
- `item/commandExecution/outputDelta` -> `item.delta + itemType=command + contentType=terminal`
- server request -> `approval.requested`
- `serverRequest/resolved` -> `approval.resolved`

Codex `ThreadItem.type` 映射：

- `userMessage` -> `message + role=user + phase=input`
- `hookPrompt` -> `message + role=system + phase=system`
- `agentMessage` -> `message`
- `plan` -> `plan + phase=progress`
- `reasoning` -> `reasoning + phase=analysis`
- `commandExecution` -> `command + tool.kind=shell`
- `fileChange` -> `file_change + tool.kind=file_edit`
- `mcpToolCall` -> `mcp_tool + tool.kind=mcp`
- `dynamicToolCall` -> `tool + tool.kind=function`
- `collabAgentToolCall` -> `subagent + tool.kind=subagent`
- `webSearch` -> `web_search + tool.kind=web_search`
- `imageView` -> `image`
- `imageGeneration` -> `image + tool.kind=image_generation`
- `enteredReviewMode` / `exitedReviewMode` -> `mode_change`
- `contextCompaction` -> `context`
- 未知类型 -> `unknown + raw.provider=codex`

`agentMessage.phase` 映射：

- `final_answer` -> `phase=final`
- `commentary` -> `phase=progress`
- 缺失 phase -> 兼容处理为 `phase=final`

## 6. 分层架构

### Client

Client 是 provider 适配边界：

- Codex 原生 JSON-RPC notification 和 server request 只在 `client/internal/agent/codex` 解析。
- `client/internal/agent/types` 暴露通用 `RuntimeTimelineEventSource`。
- `client/cmd/client` 把 runtime 事件发送为 `timeline.event`，同时在过渡期继续发送旧事件。

### Gateway

Gateway 是控制面和路由边界：

- 根据 `event.threadId` 做 Console 订阅过滤。
- 根据 `turn.started` / `turn.completed` / `turn.failed` 更新 active turn 状态。
- 可以保存每个 thread 最近一段 timeline events，用于刷新或重新打开线程时恢复工作区。
- 不理解 Codex 的 item 名称，只处理通用字段。

### Console

Console 是展示聚合边界：

- WebSocket 优先消费 `timeline.event`。
- 旧 `turn.delta` 通过兼容层转成 timeline patch。
- 使用 `turnId + itemId` 聚合同一轮、同一 item 的 delta。
- 每个 turn 形成稳定 Agent 容器：最终回答在主区域，过程和工具进入可折叠区域。

## 7. Console 展示规则

UI 按 `phase` 和 `itemType` 分层：

- `phase=final`：最终回答，主气泡展示，支持 Markdown/GFM。
- `phase=progress`：过程性文字，执行中展开，完成后默认折叠。
- `phase=analysis`：推理摘要或分析摘要，完成后默认折叠。
- `itemType=command`：终端输出块。
- `itemType=file_change`：文件变更块。
- `itemType=web_search/mcp_tool/tool/subagent`：工具过程块。
- `approval.requested`：审批卡片。
- `turn.started/completed`：只影响状态，不渲染为普通聊天消息。

执行中状态只由 active turn 和 pending item 决定，不再依赖 `正在分析...` 这类注入文本。

## 8. 兼容策略

本次改造采用双轨兼容：

- 新协议：`timeline.event`
- 旧协议：`turn.started`、`turn.delta`、`turn.completed`、`turn.failed`、`approval.required`、`approval.resolved`

Console 优先消费 timeline；如果没有 timeline，则继续走旧事件兼容路径。等 timeline 稳定后，再单独计划清理旧协议。

## 9. 验证策略

必须覆盖：

- 每一种 Codex `ThreadItem.type` 的 mapper 单元测试。
- `agentMessage` final/progress 的区分测试。
- reasoning summary 折叠展示测试。
- command/file/webSearch/MCP/dynamic tool/subagent 的过程展示测试。
- approval request/resolved 测试。
- turn completed 后 Gateway 和 Console 都清理 active 状态。
- Console UI 展示矩阵测试，确认每一种通用 `itemType` 都能在真实界面中以正确形态展示：
  - `message + phase=input`：用户消息右侧气泡。
  - `message + phase=final`：Agent 主气泡，Markdown/GFM 正常渲染。
  - `message + phase=progress`：过程区，执行中展开，完成后折叠。
  - `reasoning + phase=analysis`：处理过程折叠框，不进入最终回答正文。
  - `plan`：处理过程折叠框中的计划内容。
  - `command`：终端输出块，支持多行、PASS/FAIL/Error 基础高亮和横向滚动。
  - `file_change`：文件变更块，展示文件路径、增删数量和展开/收起状态。
  - `web_search`：搜索/打开网页过程块，展示 query、URL、状态。
  - `mcp_tool`：MCP 工具调用块，展示 server、tool、status、结果或错误。
  - `tool` / `dynamicToolCall`：通用函数工具块，展示工具名、参数摘要、结果摘要。
  - `subagent`：子 Agent 调用块，展示目标 Agent、状态和返回结果。
  - `image` / `artifact`：图片或产物块，展示生成状态、路径或预览占位。
  - `approval.requested` / `approval.resolved`：审批卡片展示问题、选项、决策和完成状态。
  - `context` / `mode_change` / `system.event`：紧凑系统事件，不污染主回答。
  - `unknown`：不破坏页面；有可展示文本时显示紧凑 debug row，否则仅保留 raw。
- Console UI 状态测试：
  - active turn 时显示“正在执行/处理中”。
  - turn completed 后隐藏“正在执行”，过程区变成“已处理”并默认折叠。
  - failed turn 清理 active 状态并显示错误。
  - 同一 turn 的多个 item 不拆成多个 Agent 对话框，除非它们是不同的最终回答 item。
  - 历史 events 重新加载后 UI 与实时流式过程一致。
  - 旧 `turn.delta` fallback 不与 timeline event 重复渲染。
- 用户真实多轮 case：
  - 第一轮：`hi`
  - 第二轮：`你调研一下，目前最新的Agent助理有哪些？做一份总结报告。`
  - 第二轮执行状态必须挂在第二轮回复上。
  - 最终输出必须在一个 Markdown 气泡里。
  - 第一轮回答不能被覆盖。

## 10. 风险与取舍

- Timeline 模型比旧 `turn.delta` 更完整，但实现范围跨 client、gateway、console，需要分阶段落地。
- Gateway 保存历史 timeline 会增加内存占用，因此需要按 thread 限制事件数量。
- reasoning 原始内容可能涉及 provider 私有推理，不应默认展示；只展示明确可显示的摘要。
- 旧协议保留会带来短期重复事件风险，Console 需要对 timeline 和 legacy 做去重或优先级处理。
