# CodingAgentGateway

## 仓库概览

`CodingAgentGateway` 是一个以 `Codex` 为第一目标运行时的单租户控制面仓库。当前代码和方案文档共同指向同一个核心结构：

- `gateway` 是控制面和注册中心，对外暴露 HTTP API 与 WebSocket。
- `client` 是每台机器上的执行面常驻进程，负责连接 Gateway、管理本机 Agent 运行时，并与本机 `Codex App Server` 集成。
- `console` 是前端管理台，只连接 `gateway`，不直接连接 `client` 或 `Codex`。
- `common` 是 `gateway` 与 `client` 共享的稳定领域模型和协议层。

当前阶段可以概括为：

- 架构上仍是 `Gateway + Machine Client + Console` 的三段式系统。
- 产品上以 `Thread-first` 工作流为中心，主路径是线程列表、线程工作区、审批、`steer`、`interrupt`。
- 前端上采用 `design-source first` 策略，保持设计导出代码尽量原样落地，再由宿主层和 Gateway 适配层接入真实能力。
- 机器标识已从“可随启动参数变化的展示值”演进到“持久化 `machineId` + 友好 `machine.name`”模型。

## 三个主模块怎么理解

### Gateway

`gateway/` 是控制面真相源，负责：

- 机器注册、连接会话、路由和索引
- 对 `console` 暴露产品化 northbound API
- 接收 `client` 上报的快照、事件和实时状态
- 维护线程、环境资源、设置、能力快照等控制面视图

不要把 `gateway` 理解成 Codex 代理本身。它不直接操作 Codex，而是协调和路由。

### Console

`console/` 是前端管理台，负责：

- 线程导航和线程工作区
- 机器、环境、设置等页面
- 调用 Gateway HTTP API
- 订阅 Gateway WebSocket 事件

`console` 只理解 Gateway 的产品化模型，不应直接理解 Codex App Server 的原生协议细节。

### Client

`client/` 是每台机器上的执行面代理，负责：

- 与 `gateway` 建立长连接
- 管理本机 Agent 运行时和本机状态
- 把 Gateway 的统一命令翻译为本机 Agent 操作
- 通过 `client/internal/agent/codex` 对接 `Codex App Server`
- 上报机器、线程、环境资源、审批、Turn 事件等快照和流

如果需要处理 Codex 专有语义，优先放进 `client/internal/agent/codex`，不要把这层细节泄漏到 `gateway`、`console` 或 `common`。

## 目录地图

下面按当前仓库已存在的目录说明职责。

```text
client/                         每台机器上的执行面 Go 服务
  cmd/client/                   client 进程入口
  internal/
    agent/                      统一 Agent 抽象与运行时管理
      codex/                    Codex 适配层；唯一应理解 Codex App Server 细节的位置
      manager/                  运行时统一管理入口，封装 thread/environment/config 等操作
      registry/                 运行时注册表
      types/                    运行时接口定义和跨实现参数结构
    config/                     client 启动配置、路径、机器身份与机器展示名解析
    gateway/                    client -> gateway 会话与 southbound envelope 发送
    snapshot/                   本机运行时快照组装

common/                         gateway 与 client 共享的稳定 Go 包
  domain/                       共享领域模型，如 machine/thread/environment/capability/config
  protocol/                     gateway <-> client 的消息 envelope 与 payload
  transport/                    JSON 编解码等通用传输辅助
  version/                      协议版本常量

console/                        React + TypeScript + Vite 前端
  src/
    app/                        应用级 providers、router 等装配层
    common/                     前端公共能力
      api/                      HTTP / WebSocket 客户端与类型定义
    design-source/              上游设计导入层；近似只读，尽量保持接近设计导出
      components/               设计导入的页面/模块组件
      data/                     设计导入层附带的数据或静态结构占位
      styles/                   设计导入层样式
    design-host/                最薄宿主层，负责挂载 design-source 并处理兼容与桥接
    design/                     当前仓库内的设计壳层、页面视图和样式组合
      components/               设计层复用组件
      pages/                    设计层页面视图
      shell/                    设计壳层
      styles/                   设计层样式
    gateway/                    Gateway 适配层，把 API/WS 数据转成前端 view-model
    pages/                      真实路由页面与页面级测试入口
  tests/                        Playwright 等前端端到端/烟雾测试

docs/                           方案、计划、设计沉淀
  superpowers/
    specs/                      设计文档，描述目标架构、边界和关键决策
    plans/                      实施计划，描述按任务拆分的落地路径

gateway/                        控制面 Go 服务
  cmd/gateway/                  gateway 进程入口
  internal/
    api/                        面向 console 的 HTTP API 与 northbound 编排
    config/                     gateway 启动配置
    registry/                   机器注册与在线会话存储
    routing/                    thread/machine 路由解析
    runtimeindex/               机器、线程、环境资源、指标等控制面索引
    settings/                   settings 和 console preferences 存储
    websocket/                  client / console 两侧 WebSocket hub

scripts/                        常用测试脚本，当前主要服务 settings 相关验证

testenv/                        联调和端到端测试环境
  dev-integration/              本地联调环境，拉起 gateway/client/console
  settings-e2e/                 settings 配置下发链路的专用 E2E 环境
```

补充说明：

- 根目录 `go.work` 负责把 `common/`、`gateway/`、`client/` 组织成一个 Go workspace。
- `console/README.md` 和 `console/README.zh-CN.md` 记录了前端的 `design-source first` 约束；修改前端前应先读。
- `docs/superpowers/specs/` 是理解项目意图的第一入口，代码结构是这些决策的具体实现。

## 基于代码和方案文档的当前理解

### 1. 这是一个控制面/执行面分离的系统

项目的主边界不是“前后端”，而是：

- `console` 负责展示与交互
- `gateway` 负责控制与协调
- `client` 负责本机执行与观测
- `Codex App Server` 是 `client` 的下游依赖，而不是系统内的公共接口

任何跨层设计都应先问一句：这件事是展示层、控制面，还是执行面职责。

### 2. 仓库仍然是 Codex-first，但结构上为未来多 Agent 预留了口子

设计文档从一开始就强调“V1 聚焦 Codex，但接口层要为未来其他 Agent 预留稳定抽象”。当前代码已经体现为：

- `client/internal/agent/types` 中有统一运行时接口
- `manager` / `registry` 依赖抽象而不是直接依赖 Codex
- `common/domain` 和 `common/protocol` 共享的是产品化模型，不是 Codex 原生类型

因此：

- 新增 Codex 相关能力时，优先扩展统一接口，再在 `codex/` 中落地
- 非共享的 Codex 细节不要“顺手”加到 `common/`

### 3. Console 的当前主路线是 Thread-first

从 `2026-04-11-thread-first-console-design.md` 和当前 `console` 代码可以看出，Console 不是传统 Dashboard-first 的后台，而是围绕线程工作区组织：

- 默认心智是“回到正在处理的线程”
- 审批、Turn 状态、`steer`、`interrupt` 尽量并入同一消息流
- `Machines`、`Environment`、`Settings` 是支撑页，不是主舞台

新增前端能力时，优先考虑它是否服务线程主路径，而不是是否丰富了管理页。

### 4. Console 的代码组织遵循 design-source first

当前前端最重要的边界不是页面路由，而是三层分工：

- `src/design-source/` 保持接近上游设计导出
- `src/design-host/` 负责最薄的宿主兼容与挂载
- `src/gateway/` 负责真实 Gateway 接入、状态拼装和 view-model 映射

这意味着：

- 不要把业务逻辑、fetch、WebSocket、能力判断塞回 `src/design-source/`
- 不要为了“代码风格统一”而重写设计导入层
- 设计变更优先通过替换 `design-source` + 最小 host 修复的方式落地

### 5. 机器身份和机器展示名已经被明确拆开

从 `2026-04-16-machine-identity-and-name-design.md` 和 `client/internal/config` 当前结构可知：

- `machineId` 是安装级稳定身份，用于路由和归属
- `machine.name` 是展示名，优先来自 `MACHINE_NAME`，否则取 hostname
- Console 默认展示 `machine.name`，而不是把 `machineId` 当作人类可读名称

因此：

- 涉及路由、线程归属、环境归属时看 `machineId`
- 涉及 UI 展示时优先使用 `machine.name`

### 6. 环境资源和设置都应走 Gateway，而不是绕过控制面

从 settings、environment、capability rollout 的设计与代码可以看出：

- `skill / mcp / plugin` 被统一建模为环境资源
- 全局默认配置、机器覆盖配置、配置下发都以 `gateway` 为控制面入口
- `console` 不应直接操作本地文件或宿主机状态

如果某项能力还没接通，应显式表现为未接入或禁用态，而不是在前端伪造成功路径。

## 开发边界和约束

### Console 相关

- 修改前先读 [console/README.md](console/README.md)。
- `console/src/design-source/` 视为近似只读层。
- 新的接口接入、能力判断、状态编排应优先放在 `console/src/gateway/`。
- 页面级行为改动通常落在 `console/src/pages/`，而不是把状态逻辑写回设计导入层。

### Gateway 相关

- `gateway/internal/api/` 负责 northbound API 语义，不要在这里引入 Codex 专有细节。
- `gateway/internal/runtimeindex/` 和 `routing/` 负责控制面索引和路由归属，适合承接机器、线程、环境资源视图。
- `gateway/internal/settings/` 是配置和 console preferences 的存储边界。

### Client 相关

- `client/internal/agent/codex/` 是 Codex 集成唯一主战场。
- `client/internal/agent/manager/` 负责统一 runtime 操作，不要在上层直接拼装 Codex 行为。
- 机器身份、路径、启动配置放在 `client/internal/config/`，不要分散到入口和 runtime 里。

### Shared model 相关

- 共享的产品化领域模型放 `common/domain/`。
- `gateway <-> client` 的消息结构放 `common/protocol/`。
- 只有在 `gateway` 和 `client` 都需要稳定依赖时，才把东西提到 `common/`。

## 推荐阅读顺序

第一次接手这个仓库，建议按下面顺序建立上下文：

1. [AGENTS.md](AGENTS.md)
2. [docs/superpowers/specs/2026-04-07-code-agent-gateway-v1-design.md](docs/superpowers/specs/2026-04-07-code-agent-gateway-v1-design.md)
3. [docs/superpowers/specs/2026-04-11-thread-first-console-design.md](docs/superpowers/specs/2026-04-11-thread-first-console-design.md)
4. [docs/superpowers/specs/2026-04-12-console-design-source-replacement-design.md](docs/superpowers/specs/2026-04-12-console-design-source-replacement-design.md)
5. [docs/superpowers/specs/2026-04-10-agent-config-settings-design.md](docs/superpowers/specs/2026-04-10-agent-config-settings-design.md)
6. [docs/superpowers/specs/2026-04-16-machine-identity-and-name-design.md](docs/superpowers/specs/2026-04-16-machine-identity-and-name-design.md)
7. [console/README.md](console/README.md)

## 一句话总结

把这个仓库理解成“一个以 Codex 为当前首个运行时的控制面系统”：`gateway` 统一对外，`client` 统一对内，`console` 围绕线程工作区组织，`common` 只沉淀稳定共享模型，而前端设计层与业务接入层必须严格分离。
