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
- 技术栈固定为：
  - `gateway`: Go
  - `client`: Go
  - `console`: React + TypeScript + Vite
  - `gateway <-> client`: WebSocket + JSON
  - `client <-> Codex`: Codex App Server

## 3. 官方能力映射

以下能力边界来自 OpenAI 官方 Codex 文档：

- Codex App Server 面向深度产品集成，覆盖认证、会话历史、审批、流式 agent 事件。
- Codex App Server 提供 `thread/start`、`thread/resume`、`thread/read`、`thread/list`、`thread/archive` 等线程级原语。
- Codex App Server 提供 `turn/start`、`turn/steer`、`turn/interrupt` 等执行轮次原语。
- 线程运行态直接参考 Codex 的 `status.type`，包括 `notLoaded`、`idle`、`systemError`、`active`。
- 审批能力直接参考 Codex 的 server request / item 机制，而不是单独发明一套审批模型。
- Skill 是一个包含 `SKILL.md` 的目录，可显式或隐式触发；可以通过 `~/.codex/config.toml` 中的 `[[skills.config]]` 做启停。
- MCP server 配置保存在 `~/.codex/config.toml` 或项目级 `.codex/config.toml`，CLI 和 IDE 共用配置。
- Plugin 是 Codex 的可安装分发单元，可打包 skill、app integration 和 MCP server；禁用 plugin 同样通过 `config.toml` 完成，并可能需要重启 Codex。

设计上的关键推断：

- 由于 V1 需要做实时 thread 对话、审批、事件流和工作台式 H5，机器侧应通过本地 `Codex App Server` 做深度集成，而不是把 CLI/TUI 直接暴露给 Gateway。
- `client` 中应该先定义统一 Agent 抽象，再由 `codex` 作为其中一个实现挂接进去。
- `gateway`、`console` 和 `common` 都不应理解 Codex 的内部实现细节，Codex 原生语义只在 `client/internal/agent/codex` 中处理。

说明：上面几条是基于官方能力边界做的工程推断，不是官方强制架构。

## 4. 总体架构

系统由四个核心部分组成：

1. `Console`
2. `Gateway`
3. `Machine Client`
4. `Codex App Server`

主数据流：

1. `console` 通过 HTTP 调用控制面接口，通过 WebSocket 订阅实时状态。
2. `gateway` 维护机器注册、资源索引、请求路由和实时事件分发。
3. `client` 与 `gateway` 建立 WebSocket 长连接，负责本机 Codex 的实际执行和快照上报。
4. `client/internal/agent/codex` 将统一命令翻译为 Codex App Server 的原生命令与事件流。

职责划分：

- `gateway` 是控制面真相源，负责：
  - 机器注册
  - 路由与寻址
  - 面向 `console` 的产品化 API
  - WebSocket 广播与订阅
  - 轻量 thread 元数据索引
- `client` 是执行面真相源，负责：
  - 本机 Codex 生命周期控制
  - thread 执行与恢复
  - skill/mcp/plugin 的本地配置变更
  - 本机快照上报
- `console` 是展示与交互层，负责：
  - 机器、thread、环境资源视图
  - thread 工作区
  - approval 操作
- `Codex App Server` 是 `client` 的下游，不直接暴露给 `gateway` 或 `console`。

## 5. 仓库结构

V1 推荐使用一个 Go monorepo，加一个前端目录：

```text
go.work
gateway/
  go.mod
  cmd/gateway/
  internal/
client/
  go.mod
  cmd/client/
  internal/
common/
  go.mod
  domain/
  protocol/
  transport/
  version/
console/
  package.json
  vite.config.ts
  src/
```

其中：

- `gateway/`：Go 服务，负责控制面和注册中心。
- `client/`：Go 服务，负责本机执行面。
- `common/`：Go 通用包，给 `gateway` 和 `client` 复用。
- `console/`：React + TypeScript + Vite 的 H5 前端。

## 6. common 包边界

`common/` 只拆成 4 个稳定包：

### 6.1 `common/domain`

负责共享领域模型，不放传输细节。

例如：

- `Machine`
- `Runtime`
- `Thread`
- `Turn`
- `ApprovalRequest`
- `Skill`
- `McpServer`
- `Plugin`
- `MachineStatus`
- `ThreadStatus`
- `TurnStatus`
- `ApprovalStatus`

### 6.2 `common/protocol`

负责 `gateway <-> client` 的 WebSocket + JSON 消息结构。

例如：

- `Envelope`
- `SystemMessage`
- `CommandMessage`
- `EventMessage`
- `SnapshotMessage`
- `ErrorMessage`

### 6.3 `common/transport`

负责通用传输辅助，不放业务状态。

例如：

- JSON 编解码
- message validation
- requestId correlation
- protocol error helpers

### 6.4 `common/version`

负责协议版本和兼容性标识。

例如：

- `ProtocolVersion`
- `ClientMinVersion`
- `GatewayMinVersion`

设计约束：

- `domain` 不放 WebSocket message envelope。
- `protocol` 不放 Codex 适配逻辑。
- `transport` 不放 thread 状态机。
- `common` 不放 `gateway` 注册中心逻辑。
- `common` 不放 `client` 的 Codex App Server 调用逻辑。

原则：

- `domain` 解决“是什么”
- `protocol` 解决“怎么表达”
- `transport` 解决“怎么传”
- `version` 解决“怎么兼容”

## 7. Gateway / Client / Console 职责划分

### 7.1 Gateway

`gateway` 作为控制面和注册中心，负责：

- 维护机器注册、在线状态、连接会话、路由信息
- 接收 `console` 的 HTTP/WebSocket 请求
- 把请求转成统一命令，下发到目标 `client`
- 汇总并转发 `client` 回来的事件、快照和 thread 状态
- 不直接对接 Codex App Server

### 7.2 Client

`client` 部署在每台目标机器上，负责：

- 维护到 `gateway` 的长连接
- 管理本机 Codex App Server 生命周期
- 把 `gateway` 的统一命令翻译成对 Codex 的本机操作
- 负责本机 `thread / skill / mcp / plugin` 的真实执行和快照采集
- 作为本机运行态的真相源

### 7.3 Console

`console` 只对接 `gateway`，负责：

- 展示机器列表、thread 列表、环境资源、实时对话工作区
- 不直接连接任何机器上的 Codex
- 不理解 Codex 原生协议，只理解 Gateway 提供的产品化接口

原则：

- `gateway` 负责“控制与协调”
- `client` 负责“执行与观测”
- `console` 负责“展示与交互”

## 8. Client 内部结构

`client/` 建议结构：

```text
client/
  go.mod
  cmd/client/
    main.go
  internal/
    config/
    gateway/
    runtime/
    snapshot/
    agent/
      types/
      manager/
      registry/
      codex/
```

职责拆分：

- `config/`
  - 本机 client 配置
- `gateway/`
  - 到 gateway 的 WebSocket 连接
  - command 收发、event/snapshot 上报
- `runtime/`
  - 本机运行时生命周期管理
- `snapshot/`
  - 机器、runtime、thread、environment 快照组装
- `agent/types`
  - 统一 Agent 抽象
- `agent/manager`
  - 接收来自 `gateway` 的命令，统一分发到具体 Agent 实现
- `agent/registry`
  - 注册本机支持的 Agent 实现，一期只注册 `codex`
- `agent/codex`
  - Codex App Server 适配实现

`client` 中应该先定义统一 Agent 接口，至少覆盖：

- `runtime start/stop/status`
- `thread create/resume/read/list/archive`
- `turn start/steer/interrupt`
- `approval respond`
- `environment list/update`

原则：

- `client` 不应先长成 `codex-first` 的结构
- `codex` 只是统一 Agent 抽象的一个实现

## 9. Gateway 内部结构

`gateway/` 建议结构：

```text
gateway/
  go.mod
  cmd/gateway/
    main.go
  internal/
    config/
    api/
    websocket/
    registry/
    routing/
    session/
    runtimeindex/
```

职责拆分：

- `config/`
  - 读取配置、端口、超时、日志级别
- `api/`
  - 面向 `console` 的 HTTP 接口
- `websocket/`
  - 管理 `console` 和 `client` 的 WebSocket 连接
- `registry/`
  - 机器注册中心
  - 在线状态、最后心跳、连接映射
- `routing/`
  - 把某个 thread/command 路由到正确的机器
- `session/`
  - 管理 console 订阅和 client 会话
- `runtimeindex/`
  - 维护来自 client 的快照索引

设计约束：

- `gateway/internal/*` 绝不直接 import Codex 相关包
- Codex 原生语义只允许存在于 `client/internal/agent/codex`

## 10. Console 前端结构

`console/` 建议结构：

```text
console/
  package.json
  vite.config.ts
  src/
    app/
    pages/
    features/
    entities/
    common/
```

职责拆分：

- `app/`
  - 应用入口
  - 路由
  - 全局 providers
  - HTTP / WebSocket 客户端初始化
- `pages/`
  - 页面级容器
- `features/`
  - 用户动作和交互逻辑
- `entities/`
  - 领域对象的前端展示模型
- `common/`
  - UI 基础组件
  - hooks
  - utils
  - API 类型镜像

推荐页面：

- `Overview`
- `Machines`
- `Threads`
- `Thread Workspace`
- `Environment`

布局：

- Web：左侧导航，中间主上下文，右侧检查器
- Mobile：单主面板 + bottom sheet + bottom tab

原则：

- `console` 不直接理解 Codex 原生事件
- `console` 只理解 Gateway 暴露出来的产品对象和产品化事件

## 11. Northbound API：Console 到 Gateway

建议拆成两层：

- HTTP API
- WebSocket Subscription

### 11.1 HTTP API

用于控制和初始列表拉取：

- `GET /machines`
- `GET /machines/{id}`
- `POST /machines/{id}/runtime/start`
- `POST /machines/{id}/runtime/stop`
- `GET /threads`
- `GET /threads/{id}`
- `POST /threads`
- `POST /threads/{id}/archive`
- `DELETE /threads/{id}`
- `GET /environment/skills`
- `GET /environment/mcps`
- `GET /environment/plugins`
- `POST /environment/skills/{id}/enable`
- `POST /environment/skills/{id}/disable`
- `POST /environment/mcps`
- `DELETE /environment/mcps/{id}`
- `POST /environment/plugins/{id}/install`
- `POST /environment/plugins/{id}/enable`
- `POST /environment/plugins/{id}/disable`
- `DELETE /environment/plugins/{id}`

### 11.2 WebSocket Subscription

用于实时订阅：

- `machine.updated`
- `runtime.updated`
- `thread.updated`
- `turn.started`
- `turn.delta`
- `turn.completed`
- `turn.failed`
- `approval.required`
- `approval.resolved`
- `resource.changed`

规范：

- `console` 通过 HTTP 发起控制动作
- `console` 通过 WebSocket 接收异步执行结果和流式事件
- `console` 看到的对象都是 Gateway 聚合后的产品对象，而不是 client 或 Codex 原生对象

## 12. Southbound 协议：Gateway 到 Client

`gateway <-> client` 采用 `WebSocket + JSON`，协议统一成 4 类消息：

- `system`
- `command`
- `event`
- `snapshot`

统一信封：

```json
{
  "version": "v1",
  "category": "command",
  "name": "thread.start",
  "requestId": "req_123",
  "machineId": "machine_01",
  "timestamp": "2026-04-07T10:00:00Z",
  "payload": {}
}
```

### 12.1 system

用于连接握手和保活：

- `client.register`
- `client.heartbeat`
- `gateway.ack`
- `gateway.ping`
- `client.pong`

### 12.2 command

`gateway -> client`

一期先覆盖：

- `runtime.start`
- `runtime.stop`
- `thread.create`
- `thread.resume`
- `thread.archive`
- `thread.delete`
- `turn.start`
- `turn.steer`
- `turn.interrupt`
- `approval.respond`
- `skill.enable`
- `skill.disable`
- `mcp.upsert`
- `mcp.remove`
- `plugin.install`
- `plugin.enable`
- `plugin.disable`
- `plugin.uninstall`

### 12.3 event

`client -> gateway`

一期先覆盖：

- `command.accepted`
- `command.rejected`
- `command.completed`
- `runtime.updated`
- `thread.updated`
- `turn.started`
- `turn.delta`
- `turn.completed`
- `turn.failed`
- `approval.required`
- `approval.resolved`
- `resource.changed`

### 12.4 snapshot

`client -> gateway`

一期建议覆盖：

- `machine.snapshot`
- `runtime.snapshot`
- `thread.snapshot`
- `environment.snapshot`

协议约束：

1. `gateway` 只发 `system/command`
2. `client` 只发 `system/event/snapshot`
3. 所有 `command` 必须带 `requestId`
4. 所有由命令触发的 `event` 必须回带 `requestId`
5. `snapshot` 永远是完整真相，不是增量 patch

## 13. Thread / Turn / Approval 模型

这一部分直接参考 Codex App Server 的原生设计，只做最薄封装。

### 13.1 Thread

直接对齐 Codex 的线程语义：

- `thread/start`
- `thread/resume`
- `thread/read`
- `thread/list`
- `thread/archive`

线程运行态直接参考 Codex 的 `status.type`：

- `notLoaded`
- `idle`
- `systemError`
- `active`

### 13.2 Turn

直接对齐 Codex 的执行轮次语义：

- `turn/start`
- `turn/steer`
- `turn/interrupt`

`turn/start` 返回一个进行中的 turn。

最终结果按 Codex 保持：

- `completed`
- `interrupted`
- `failed`

### 13.3 Approval

不单独发明审批实体，而是参考 Codex 的 server request / item 机制。

前端展示时，可以投影成统一的 `ApprovalRequest` 视图，但协议层必须保持与 Codex 的关系：

- 必须带 `threadId`
- 必须带 `turnId`
- 通常还要带 `itemId` 或 `requestId`

一期至少覆盖：

- 命令执行审批
- 文件变更审批
- MCP / app tool 调用审批

领域关系：

- 一个 `thread` 同时只能有一个活跃 `turn`
- 一台机器可以同时执行多个不同 `thread` 的 `turn`
- `approval` 永远隶属于某个 `turn`
- `interrupt` 打断的是 `turn`
- `archive/delete` 作用的是 `thread`

## 14. Skill / MCP / Plugin 运维模型

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

- `gateway` 只下发意图级命令。
- `client` 负责具体的本地配置写入和安装行为。
- 如果变更需要重启 Codex，`client` 在快照中返回 `restartRequired = true`。
- `Environment` 页共享一套外观，但右侧 Inspector 展示资源类型专属字段。

## 15. 错误处理与恢复

一期保持“轻状态控制面”原则。

错误分层：

- `gateway error`
  - 路由失败
  - client 离线
  - 请求参数错误
  - websocket 会话失效
- `client error`
  - 本机执行失败
  - Codex App Server 不可达
  - 本地配置写入失败
- `codex error`
  - thread/turn 执行失败
  - approval 超时
  - 上下文或运行时问题
- `environment error`
  - plugin 安装失败
  - mcp 鉴权失败
  - skill/plugin 状态异常

恢复策略：

- `gateway` 重启后：
  - `client` 自动重连
  - 重新发送 `client.register`
  - 重新上报 `machine/runtime/thread/environment snapshot`
  - `gateway` 用 snapshot 重建控制面视图
- `client` 断连后：
  - `gateway` 把机器标记为 `offline / reconnecting`
  - 相关 thread 视图标记为 `state unknown`
  - 等待机器恢复后用 snapshot 纠正
- Codex App Server 异常：
  - `client` 负责探测和重连
  - `client` 向 `gateway` 发送 `runtime.updated` 和 `runtime.snapshot`
  - `gateway` 不自行猜测运行态

约束：

- `snapshot` 才是恢复真相
- `gateway` 不伪造机器状态
- northbound 错误要产品化
- southbound 错误要协议化

## 16. 一期非目标与未来扩展

V1 明确不做：

- 多租户、多组织和复杂 RBAC
- 第二种 Agent 的实际落地适配
- Gateway 内发布自定义 skill/plugin
- 完整消息历史持久化、审计中心、断点级恢复
- 原生 App
- 多 runtime 编排

二期预留：

- 新增第二种 Code Agent 适配器
- 多租户与权限体系
- 完整持久化与审计
- 多 runtime 编排
- Gateway 级 skill/plugin 发布能力
- 原生 App 客户端

稳定边界应保持在：

- `console <-> gateway` 的 northbound API
- `gateway <-> client` 的 southbound 协议
- `common/domain`
- `client/internal/agent/*` 的统一 Agent 抽象

## 17. 实施建议

建议按以下顺序进入实施规划：

1. 先搭建 `go.work`、`common/`、`gateway/`、`client/` 和 `console/` 的新结构。
2. 定义 `common/domain` 与 `common/protocol`。
3. 实现 `gateway <-> client` 的 WebSocket + JSON 基础链路。
4. 先打通 `client/internal/agent` 统一抽象，再实现 `agent/codex`。
5. 优先做 Codex 最小闭环：
   - 机器注册
   - thread 列表
   - 新建 thread
   - thread 对话
   - interrupt/steer
6. 然后补 `Environment` 资源管理：
   - skills
   - MCP
   - plugins
7. 最后补 `console` 的页面、移动端适配与错误态。

## 18. 风险与注意事项

- Codex 原生能力和配置形态是官方事实，但 Gateway 的统一协议是本项目自定义设计，后续要严控兼容性。
- `skill / mcp / plugin` 虽然都属于环境增强能力，但它们的来源与生命周期不同，领域模型不能强行做成完全同构。
- 机器侧只保留一个 Codex runtime 的前提下，多 thread 并发的真实上限需要在实现阶段实测并可配置。
- Gateway 采用轻状态意味着观测与恢复强依赖 Client 的快照质量。
- `client` 的统一 Agent 抽象要足够稳定，否则后续接第二种 Agent 会重新污染主干结构。

## 19. 参考资料

官方文档：

- OpenAI Codex App Server: https://developers.openai.com/codex/app-server
- OpenAI Codex SDK: https://developers.openai.com/codex/sdk
- OpenAI Codex Skills: https://developers.openai.com/codex/skills
- OpenAI Codex MCP: https://developers.openai.com/codex/mcp
- OpenAI Codex Plugins: https://developers.openai.com/codex/plugins

本次设计直接依据的官方事实：

- App Server 适合深度集成，覆盖 conversation history、approvals、streamed agent events：
  - https://developers.openai.com/codex/app-server
- App Server `thread/start`、`thread/resume`、`thread/read`、`thread/list`、`thread/archive`：
  - https://developers.openai.com/codex/app-server
- App Server `turn/start`、`turn/steer`、`turn/interrupt`：
  - https://developers.openai.com/codex/app-server
- 线程状态和 `waitingOnApproval`：
  - https://developers.openai.com/codex/app-server
- Skill 目录结构、显式/隐式触发、`[[skills.config]]` 启停：
  - https://developers.openai.com/codex/skills
- MCP 配置位于 `~/.codex/config.toml` 或项目级 `.codex/config.toml`：
  - https://developers.openai.com/codex/mcp
- Plugin 打包 skill/app/MCP，安装和禁用需要配置与重启：
  - https://developers.openai.com/codex/plugins
