# Code Agent Gateway V1 设计文档

## 1. 背景与目标

本项目要实现一个面向 Code Agent 的统一 Gateway。第一期聚焦 `Codex`，但接口层必须为未来接入其他 Code Agent 预留稳定抽象。

V1 目标：

- 提供统一的 `Gateway + 注册中心`，接收前端请求并路由到目标机器上的 Code Agent Client。
- 支持机器侧拉起、停止、删除、管理 Codex 会话，并管理 `thread`。
- 支持前端与某个指定 `thread` 进行实时对话。
- 支持对 `skill / mcp / plugin` 做统一运维管理。
- 提供一个 H5 管理台，同时适配移动端和 Web 端。

V1 非目标：

- 多租户、多组织、复杂 RBAC。
- 多 Agent 同时落地适配。除 Codex 外，仅保留统一接口，不实现第二种 Agent。
- Gateway 内创建或发布自定义 skill/plugin。
- 完整消息历史持久化、审计中心、强恢复。
- 原生 App。
- 单机多 Codex runtime 编排。

## 2. 一期边界

已确认的一期边界如下：

- 单租户内部工具。
- 注册中心并入 Gateway。
- 前端和机器侧 Client 都只连 Gateway。
- 每台机器一个常驻 Codex 实例。
- 单机允许多个 `thread` 并发，但同一个 `thread` 同时只允许一个活跃执行轮次。
- `skill / mcp / plugin` 纳入统一资源模型，并支持运维控制，不做发布平台。
- Gateway 采用轻状态控制面。重启后靠机器重连与快照恢复视图。

## 3. 官方能力映射

以下能力边界来自 OpenAI 官方 Codex 文档：

- Codex SDK 支持 `startThread()`、`resumeThread(threadId)`，适合自动化与程序化 thread 控制。
- Codex App Server 面向深度产品集成，明确覆盖认证、会话历史、审批、流式 agent 事件。
- Codex App Server 提供 `thread/start`、`thread/resume`、`thread/list`、`thread/archive`、`turn/start`、`turn/steer` 等原语。
- Skill 是一个包含 `SKILL.md` 的目录，可显式或隐式触发；可以通过 `~/.codex/config.toml` 中的 `[[skills.config]]` 做启停。
- MCP server 配置保存在 `~/.codex/config.toml` 或项目级 `.codex/config.toml`，CLI 和 IDE 共用配置。
- Plugin 是 Codex 的可安装分发单元，可打包 skill、app integration 和 MCP server；禁用 plugin 同样通过 `config.toml` 完成，并可能需要重启 Codex。

设计上的关键推断：

- 由于 V1 需要做实时 thread 对话、审批、事件流和工作台式 H5，机器侧应通过本地 `Codex Adapter` 对接 Codex App Server 风格能力，而不是把 CLI/TUI 直接暴露给 Gateway。
- SDK 能力仍保留在适配层内部，可作为自动化或补充路径，但不直接暴露给前端。

说明：上面两条是基于官方能力边界做的工程推断，不是官方强制架构。

## 4. 总体架构

系统由四个核心部分组成：

1. `H5 Console`
2. `Gateway`
3. `Machine Client`
4. `Local Codex Adapter / Runtime`

主数据流：

1. H5 通过 HTTP 调用控制面接口，通过 WebSocket 订阅实时状态。
2. Gateway 维护机器注册、资源索引、请求路由和实时事件分发。
3. Machine Client 与 Gateway 建立长连接，负责本机 Codex 的实际执行和快照上报。
4. Local Codex Adapter 将 Gateway 的统一命令翻译为 Codex 原生命令与事件流。

职责划分：

- Gateway 是控制面真相源，负责：
  - 机器注册
  - 路由与寻址
  - 前端产品化 API
  - WebSocket 广播与订阅
  - 轻量 thread 元数据索引
- Machine Client 是执行面真相源，负责：
  - 本机 Codex 生命周期控制
  - thread 执行与恢复
  - skill/mcp/plugin 的本地配置变更
  - 本机快照上报

## 5. 资源模型

### 5.1 控制面资源

- `Machine`
  - 机器节点
  - 在线状态
  - 当前连接 session
  - 关联 runtime
- `Runtime`
  - 当前机器上的 Codex 运行实例
  - 状态、能力、并发配置
- `Thread`
  - 会话容器
  - 归属 machine/runtime
  - 状态、名称、预览、最近更新时间
- `Turn`
  - thread 中的一次执行轮次
  - 负责承载一次输入到结果完成的完整执行链路

### 5.2 环境资源

统一资源外壳字段：

- `resourceId`
- `kind`: `skill | mcp | plugin`
- `machineId`
- `displayName`
- `scope`: `system | user | repo | project-config | plugin-bundled`
- `status`: `enabled | disabled | auth_required | error | unknown`
- `source`: `builtin | curated | local-path | config-entry | plugin-bundle`
- `restartRequired`
- `lastObservedAt`

差异字段：

- `Skill`
  - `path`
  - `description`
  - `triggerMode`
  - `ownerPluginId?`
- `McpServer`
  - `transport`
  - `command/url`
  - `enabledTools`
  - `authMode`
  - `ownerPluginId?`
- `Plugin`
  - `pluginKey`
  - `version`
  - `bundledSkills[]`
  - `bundledMcps[]`
  - `apps[]`

## 6. Thread 与 Turn 模型

定义：

- `Thread`：一个完整会话容器，包含会话身份、历史、名称、归档状态。
- `Turn`：该 thread 中的一次具体执行轮次，一般对应“用户输入 -> Codex 执行 -> 流式结果/工具调用/审批 -> 完成或失败”。

状态约束：

- 同一个 `thread` 同时只能有一个活跃 `turn`。
- 同一台机器可以并发处理多个不同 `thread` 的活跃 `turn`。
- 并发发生在多个 thread 之间，不发生在同一 thread 内部。

建议状态机：

- `Thread`
  - `created`
  - `ready`
  - `running`
  - `waiting_input`
  - `completed`
  - `archived`
  - `deleted`
- `Turn`
  - `queued`
  - `dispatching`
  - `streaming`
  - `awaiting_approval`
  - `awaiting_input`
  - `completed`
  - `failed`
  - `interrupted`
  - `canceled`

调度规则：

- Machine Client 维护本机 `activeTurnSlots + queue`。
- 超过并发上限的 turn 进入队列。
- `interrupt` 作用于当前 turn，不直接销毁 thread。
- `steer` 作用于当前正在执行的 turn。

## 7. 协议面设计

### 7.1 H5 到 Gateway

两类公共入口：

- HTTP 控制面
- WebSocket 实时面

建议 HTTP 资源：

- `GET /machines`
- `POST /machines/{id}/agent/start`
- `POST /machines/{id}/agent/stop`
- `POST /machines/{id}/agent/restart`
- `GET /threads`
- `POST /threads`
- `POST /threads/{id}/archive`
- `DELETE /threads/{id}`
- `GET /skills`
- `GET /mcps`
- `GET /plugins`
- `POST /skills/{id}/enable`
- `POST /skills/{id}/disable`
- `POST /mcps`
- `PUT /mcps/{id}`
- `DELETE /mcps/{id}`
- `POST /plugins/{id}/install`
- `POST /plugins/{id}/enable`
- `POST /plugins/{id}/disable`
- `POST /plugins/{id}/uninstall`

建议 WebSocket 事件：

- `subscribe.machine`
- `subscribe.thread`
- `thread.startTurn`
- `thread.steerTurn`
- `thread.interruptTurn`

规范：

- HTTP 请求只返回命令是否被接收或拒绝。
- 执行进度与结果通过 WebSocket 回推。
- 前端不直接使用 Codex 原生协议。

### 7.2 Gateway 到 Machine Client

Client 长连接生命周期：

- `client.register`
- `client.heartbeat`
- `client.snapshot.report`

统一命令：

- `command.startThread`
- `command.resumeThread`
- `command.archiveThread`
- `command.deleteThread`
- `command.startTurn`
- `command.steerTurn`
- `command.interruptTurn`
- `command.toggleSkill`
- `command.upsertMcp`
- `command.removeMcp`
- `command.installPlugin`
- `command.togglePlugin`
- `command.uninstallPlugin`

统一事件：

- `event.commandAccepted`
- `event.commandRejected`
- `event.commandCompleted`
- `event.thread.started`
- `event.thread.updated`
- `event.thread.archived`
- `event.thread.deleted`
- `event.turn.started`
- `event.turn.delta`
- `event.turn.completed`
- `event.turn.failed`
- `event.item.agentMessageDelta`
- `event.item.toolProgress`
- `event.item.diffUpdated`
- `event.resource.changed`

### 7.3 Codex 适配层映射

Machine Client 内部，统一协议映射到 Codex 原生能力：

- `command.startThread` -> `thread/start`
- `command.resumeThread` -> `thread/resume`
- `command.startTurn` -> `turn/start`
- `command.steerTurn` -> `turn/steer`
- `thread list/read/archive` -> 对应 `thread/list`、`thread/read`、`thread/archive`
- skill/mcp/plugin 管理 -> 本地配置文件与本地安装动作

设计要求：

- 前端永远只认 Gateway 协议。
- Gateway 协议字段在未来接第二种 Agent 时保持稳定。
- Codex 特有差异留在适配层内部解决。

## 8. H5 信息架构

推荐使用一个统一 H5 外壳，兼顾控制台与 thread 工作台。

推荐页面：

- `Overview`
  - 在线机器
  - runtime 健康状态
  - 活跃 turn
  - 最近错误
- `Machines`
  - 机器列表
  - 连接状态
  - 启停 Codex
  - 机器详情
- `Threads`
  - thread 列表
  - 按 machine 筛选
  - 打开 thread 工作区
- `Thread Workspace`
  - 消息流
  - turn 状态
  - 输入框
  - `interrupt/steer` 操作
  - 审批交互
- `Environment`
  - `Skill` 标签页
  - `MCP` 标签页
  - `Plugin` 标签页

Web 端布局：

- 左侧：导航与对象切换
- 中间：当前上下文、thread 时间线、主工作区
- 右侧：动作区、元数据、检查器

移动端布局：

- 单主面板
- bottom sheet 用于 machine/thread 切换
- bottom tab 负责一级导航

## 9. Skill / MCP / Plugin 运维模型

V1 允许的管理动作：

- `Skill`
  - 列表
  - 查看
  - 启用
  - 禁用
- `MCP`
  - 列表
  - 新增
  - 更新
  - 删除
  - 启用
  - 禁用
- `Plugin`
  - 列表
  - 安装
  - 卸载
  - 启用
  - 禁用
  - 查看打包内容

设计原则：

- Gateway 只下发意图级命令。
- Machine Client 负责具体的本地配置写入和安装行为。
- 如果变更需要重启 Codex，Client 在快照中返回 `restartRequired = true`。
- H5 的 `Environment` 页共享一套外观，但右侧 Inspector 展示资源类型专属字段。

## 10. 错误处理与恢复

错误分层：

- `Gateway 错误`
  - 路由失败
  - 机器不存在
  - Client 离线
  - 参数错误
- `Client 错误`
  - 本地 Codex 进程不可用
  - 本地配置写入失败
  - 插件安装失败
- `Codex 错误`
  - 上下文超限
  - 网络/额度/沙箱问题
  - 上游执行失败
- `环境资源错误`
  - MCP 鉴权失败
  - plugin 依赖 app 未授权
  - skill 路径失效

恢复策略：

- Gateway 重启后：
  - Machine Client 自动重连
  - 重新注册
  - 重新上报快照
  - H5 重新建立订阅并重新拉取当前对象
- Machine Client 断连后：
  - Gateway 将机器状态标记为 `offline / reconnecting / unknown`
  - 相关活跃 turn 标记为 `interrupted` 或 `status unknown`
  - 等待新快照纠正状态

约束：

- V1 的权威运行态来自机器快照，不来自 Gateway 持久层。
- 事件流允许丢失后重建，不能假设事件永不丢失。

## 11. 审批链路

V1 前端至少要能处理两类审批：

- 文件变更审批
- 网络访问审批

审批路径：

1. Codex 在 turn 执行中发出审批请求。
2. Machine Client 将审批事件转发给 Gateway。
3. Gateway 将事件推送到订阅该 thread 的 H5。
4. H5 用户做出审批决策。
5. Gateway 将审批结果回传给对应 Machine Client。
6. Machine Client 将结果继续提交给本地 Codex。

这里保持单租户最小能力，不引入复杂组织级策略引擎。

## 12. 未来扩展预留

本设计为未来二期预留以下扩展点：

- 新增第二种 Code Agent 适配器
- 多租户与权限体系
- 完整持久化与审计
- 多 runtime 编排
- Gateway 级 skill/plugin 发布能力
- 原生 App 客户端

稳定边界应保持在：

- H5 北向产品 API
- Gateway 南向 `command/event` 协议
- 资源领域模型

## 13. 实施建议

推荐按以下顺序进入实施规划：

1. 先定义 `Gateway <-> Machine Client` 协议和领域模型。
2. 再定义 `H5 <-> Gateway` 的控制 API 与 WebSocket 订阅模型。
3. 优先做 Codex 适配器最小闭环：
   - 机器注册
   - thread 列表
   - 新建 thread
   - thread 对话
   - interrupt/steer
4. 然后补 `Environment` 资源管理：
   - skills
   - MCP
   - plugins
5. 最后补 H5 的移动端适配与错误态。

## 14. 风险与注意事项

- Codex 原生能力和配置形态是官方事实，但 Gateway 的统一协议是本项目自定义设计，后续要严控兼容性。
- `skill / mcp / plugin` 虽然都属于环境增强能力，但它们的来源与生命周期不同，落库和接口不能强行做成完全同构。
- 机器侧只保留一个 Codex runtime 的前提下，多 thread 并发的真实上限需要在实现阶段实测并可配置。
- Gateway 采用轻状态意味着观测与恢复强依赖机器侧 Client 的快照质量。

## 15. 参考资料

官方文档：

- OpenAI Codex App Server: https://developers.openai.com/codex/app-server
- OpenAI Codex SDK: https://developers.openai.com/codex/sdk
- OpenAI Codex Skills: https://developers.openai.com/codex/skills
- OpenAI Codex MCP: https://developers.openai.com/codex/mcp
- OpenAI Codex Plugins: https://developers.openai.com/codex/plugins

本次设计直接依据的官方事实：

- SDK `startThread()` / `resumeThread(threadId)`：
  - https://developers.openai.com/codex/sdk
- App Server 适合深度集成，覆盖 conversation history、approvals、streamed agent events：
  - https://developers.openai.com/codex/app-server
- App Server `thread/start`、`thread/resume`、`thread/list`、`turn/start`、`turn/steer`：
  - https://developers.openai.com/codex/app-server
- Skill 目录结构、显式/隐式触发、`[[skills.config]]` 启停：
  - https://developers.openai.com/codex/skills
- MCP 配置位于 `~/.codex/config.toml` 或项目级 `.codex/config.toml`：
  - https://developers.openai.com/codex/mcp
- Plugin 打包 skill/app/MCP，安装和禁用需要配置与重启：
  - https://developers.openai.com/codex/plugins
