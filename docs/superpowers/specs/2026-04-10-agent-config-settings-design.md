# Agent Config Settings 设计文档

## 1. 目标

在现有 `console -> gateway -> client -> codex` 链路上增加一套统一的配置下发能力：

- `gateway` 保存 `global default`
- `gateway` 保存可选的 `machine override`
- `console` 在设置页编辑和查看这两份配置
- `console` 通过 `gateway` 把配置下发到指定机器
- `client` 通过统一 Agent 抽象把配置写入具体 Agent 的本地配置文件

本期先支持 `codex`，但 `console`、`gateway`、`client` 的结构都要为未来的其他 Agent 适配留出口。

## 2. 范围

本期包含：

- `Settings` 页面
- `gateway` 配置存储抽象与本地文件实现
- `gateway` settings northbound API
- `gateway -> client` 复用现有 southbound command envelope
- `client` 统一配置接口抽象
- `codex` 配置写入 `~/.codex/config.toml`
- 配置功能的单元测试、系统测试、端到端测试
- 覆盖率和场景完成度门槛

本期不包含：

- 多 Agent 的真实落地实现
- 配置 diff / merge 编辑器
- 字段级合并策略
- 配置版本回滚中心
- 数据库或 Redis 落地

## 3. 核心原则

- `global default` 是一份完整配置文档
- `machine override` 是一份完整配置文档
- 没有 `machine override` 时，机器直接使用 `global default`
- 不做字段级 merge
- 文档格式使用 `TOML`
- `gateway` 存储抽象可替换
- southbound 尽量复用现有 command 协议，不引入第二套链路

## 4. 数据模型

### 4.1 AgentType

当前枚举：

- `codex`

后续预留：

- `claude_code`
- 其他 Agent

### 4.2 AgentConfigDocument

- `agentType`
- `format`
- `content`
- `updatedAt`
- `updatedBy`
- `version`

约束：

- `format` 本期固定为 `toml`
- `content` 保存完整 TOML 文本

### 4.3 MachineAgentConfigAssignment

- `machineId`
- `agentType`
- `globalDefault`
- `machineOverride`
- `usesGlobalDefault`

其中：

- `machineOverride == nil` 时，`usesGlobalDefault = true`
- `machineOverride != nil` 时，`usesGlobalDefault = false`

## 5. Gateway 设计

### 5.1 存储抽象

新增 `gateway/internal/settings`：

- `Store`
  - `ListAgentTypes()`
  - `GetGlobal(agentType)`
  - `PutGlobal(agentType, document)`
  - `GetMachine(machineID, agentType)`
  - `PutMachine(machineID, agentType, document)`
  - `DeleteMachine(machineID, agentType)`

本期实现：

- `file_store.go`
- 本地 JSON 文件持久化

后续：

- 可以替换为数据库、Redis 或混合实现

### 5.2 Northbound API

- `GET /settings/agents`
- `GET /settings/agents/{agentType}/global`
- `PUT /settings/agents/{agentType}/global`
- `GET /settings/machines/{machineId}/agents/{agentType}`
- `PUT /settings/machines/{machineId}/agents/{agentType}`
- `DELETE /settings/machines/{machineId}/agents/{agentType}`
- `POST /settings/machines/{machineId}/agents/{agentType}/apply`

返回视图统一包含：

- `agentType`
- `globalDefault`
- `machineOverride`
- `usesGlobalDefault`

### 5.3 Apply 语义

`apply` 时：

- 如果机器存在 override，则下发 override
- 否则下发 global default
- 如果两者都不存在，则返回产品化错误

## 6. Southbound 设计

继续复用既有 `command.completed` / `command.rejected` 模型。

新增最小 southbound command：

- `agent.config.apply`

payload：

- `agentType`
- `document`
- `source`
- `version`

不新增独立 southbound channel。

## 7. Client 设计

### 7.1 统一接口

在 `client/internal/agent/types` 中新增：

- `RuntimeConfigManager`
  - `ApplyConfig(document AgentConfigDocument) (ApplyConfigResult, error)`

### 7.2 Codex 实现

`codex` adapter 负责：

- 校验 TOML 文本合法性
- 定位用户根目录下的 `~/.codex/config.toml`
- 自动创建 `.codex` 目录
- 原子写入配置文件

本期不要求 client 从机器读取当前本地配置并回传给 console。

## 8. Console 设计

新增 `Settings` 页面，结构如下：

- 顶部：Agent 选择器
  - 当前只显示 `Codex`
- 左侧：机器列表
- 中间：Global Default TOML 编辑区
- 右侧：Machine Override TOML 编辑区

行为：

- 可以保存 global default
- 可以保存 machine override
- 可以删除 machine override
- 可以对选中机器执行 `Apply`
- 机器没有 override 时明确显示“当前使用 Global Default”
- 编辑器使用 TOML 文本域，不用 JSON 编辑器

## 9. 测试设计

测试必须覆盖三层：

### 9.1 单元测试

范围：

- `gateway/internal/settings`
- `gateway/internal/api` settings handler
- `client/internal/agent/types`
- `client/internal/agent/manager`
- `client/internal/agent/codex` 配置应用与 TOML 校验
- `console/src/pages/settings-page*`

门槛：

- Go settings 相关包：`line coverage >= 85%`
- 核心配置包：
  - `gateway/internal/settings`
  - `client/internal/agent/codex` 配置相关代码
  `line coverage >= 90%`
- Frontend settings 页面相关文件：
  - `statements >= 85%`
  - `functions >= 85%`
  - `branches >= 80%`

### 9.2 系统测试

范围：

- 进程内启动 `gateway`
- 使用真实 client command/session 路径
- 使用临时 HOME 目录
- 使用真实 codex config writer 写入临时 `~/.codex/config.toml`

门槛：

- 配置场景矩阵 `100% pass`
- 不少于 `10` 个系统测试场景

### 9.3 端到端测试

范围：

- 从 `console` 页面开始
- 经 `gateway`
- 到 `client`
- 最终检查机器上的 `~/.codex/config.toml`

执行方式：

- 用 Docker 测试环境拉起 `console + gateway + client`
- client HOME 挂载到宿主临时目录
- Playwright 驱动浏览器完成配置流程

门槛：

- 关键旅程 `100% pass`
- 不少于 `4` 条 E2E 场景
- 至少 `1` 条负向场景

## 10. 完成定义

以下条件同时成立才算完成：

- `Settings` 页面支持 `Codex`
- `global default` 和 `machine override` 可保存
- `Apply` 可把 TOML 下发到目标机器
- `client` 可把 TOML 写入 `~/.codex/config.toml`
- 三层测试均已实现
- 三层测试达到定量门槛
- Docker E2E 可一键启动、执行、清理

