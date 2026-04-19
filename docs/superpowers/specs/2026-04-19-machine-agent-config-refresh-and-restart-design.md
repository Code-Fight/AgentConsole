# Machine Agent 配置实时读取与重启生效设计文档

## 1. 背景

当前机器管理页（`/machines`）支持对单个 Agent 打开“编辑配置”弹窗并保存配置。

现有链路虽然已经具备：

- 前端打开弹窗后请求 `GET /machines/{machineId}/agents/{agentId}/config`
- 前端保存时请求 `PUT /machines/{machineId}/agents/{agentId}/config`
- gateway 将请求转发为 `machine.agent.config.read` / `machine.agent.config.write`
- client 在本机 agent 实例目录下读写 `~/.codex/config.toml`

但仍存在两个产品缺口：

1. 编辑前缺少“读取成功门禁”，读取失败时会回退本地默认模板，可能误导用户。
2. 保存后未触发 Agent 重启，配置生效时机不确定。

## 2. 目标

围绕机器管理页的“单 Agent 配置编辑”建立一致流程：

1. 点击编辑时，必须先从目标机器的目标 Agent 读取最新配置，并通过 gateway 返回到页面。
2. 点击保存时，先下发新配置覆盖该 Agent 的默认配置，再重启该 Agent。
3. 页面明确反馈三类结果：
- 写入成功且重启成功
- 写入成功但重启失败
- 写入失败

## 3. 非目标

本期不包含：

- settings 页面（global default / machine override）语义调整
- 多 Agent 类型差异化配置编辑器
- 配置版本历史、diff、回滚
- 批量 Agent 配置下发

## 4. 方案选择

采用方案 A：前端串行调用两个 northbound API。

1. `PUT /machines/{machineId}/agents/{agentId}/config`
2. `POST /machines/{machineId}/agents/{agentId}/restart`

选择理由：

- 与现有机器管理页接口风格一致，改动面最小。
- 与 `runtime.start/stop`、`machine.agent.install/delete` 一样保持动作可组合。
- 失败语义清晰，便于页面给出“已保存但重启失败”提示。

## 5. 端到端流程

## 5.1 编辑弹窗打开流程（读取最新配置）

1. 用户点击 Agent 的“编辑配置”。
2. 前端进入 `loading` 状态并请求 `GET /machines/{machineId}/agents/{agentId}/config`。
3. gateway 转发 `machine.agent.config.read` 到目标 machine。
4. client 从该 agent 实例配置文件读取内容返回。
5. 前端收到响应后填充文本域，解除禁用。

交互约束：

- 读取成功前，保存按钮不可点击。
- 读取失败时显示错误，不回退到伪造默认模板；用户可“重试加载”。

## 5.2 保存流程（写入 + 重启）

1. 用户点击保存。
2. 前端先调用 `PUT /machines/{machineId}/agents/{agentId}/config`。
3. gateway 校验 TOML 后转发 `machine.agent.config.write`。
4. client 写入 agent 配置文件。
5. 写入成功后，前端再调用 `POST /machines/{machineId}/agents/{agentId}/restart`。
6. gateway 转发 `machine.agent.restart`。
7. client 对目标 agent 执行重启（停止后再启动）。
8. 重启成功后，前端提示成功并关闭弹窗，刷新机器页数据。

失败分支：

- 写入失败：终止流程，显示“保存失败”。
- 写入成功、重启失败：提示“配置已保存，但重启失败”，弹窗保持打开或提供“重试重启”。

## 6. API 与协议设计

## 6.1 Northbound API（gateway 对 console）

保留：

- `GET /machines/{machineId}/agents/{agentId}/config`
- `PUT /machines/{machineId}/agents/{agentId}/config`

新增：

- `POST /machines/{machineId}/agents/{agentId}/restart`

响应建议：

```json
{
  "machineId": "machine-01",
  "agentId": "agent-01",
  "status": "restarted"
}
```

错误约定：

- `400` 参数错误
- `404` machine 或 agent 不存在
- `502` 转发到 client 失败或 client 执行失败

## 6.2 Southbound command（gateway -> client）

新增 command：`machine.agent.restart`

payload：

```json
{
  "agentId": "agent-01"
}
```

result：

```json
{
  "agentId": "agent-01"
}
```

## 6.3 Client 执行语义

在 `Supervisor` 增加 `RestartAgent(agentID)`：

1. 校验 agentId 合法且已安装。
2. 若运行中：`stopAgent(agentID)` -> `startAgent(record)`。
3. 若已停止：直接 `startAgent(record)`。
4. 任一步骤失败返回错误。

执行后：

- 返回 `command.completed`
- 刷新 machine snapshot（agent status/running state）

## 7. 前端状态机（机器页弹窗）

新增最小状态：

- `isLoadingConfig`
- `isSavingConfig`
- `isRestarting`
- `loadError`
- `saveError`
- `restartError`

行为约束：

1. `isLoadingConfig=true` 时禁用文本域和保存按钮。
2. 保存点击后禁止重复提交。
3. 仅当写入成功才触发重启。
4. 重启失败时不清空编辑内容。

## 8. 能力开关与兼容性

本期不新增 capability 字段，沿用现有机器页可达性与网关权限模型。

若后续需要更细粒度控制，再补充：

- `machineRestartAgent`
- `machineEditAgentConfig`

## 9. 测试要求

## 9.1 Console

补充 `machines-page` 测试场景：

1. 打开编辑弹窗时发起 GET 并填充内容。
2. GET 失败时显示错误且保存不可用。
3. 保存成功后按顺序调用：PUT -> POST restart。
4. PUT 成功、restart 失败时显示部分成功提示。

## 9.2 Gateway API

补充 `server_test`：

1. `POST /machines/{machineId}/agents/{agentId}/restart` 转发 `machine.agent.restart`。
2. 参数缺失、sender 缺失、client 返回错误的 HTTP 状态码。

## 9.3 Client

补充 `main_test` 与 `supervisor_test`：

1. `machine.agent.restart` 调用 supervisor restart。
2. restart 成功后返回 `command.completed` 并刷新 snapshot。
3. restart 失败返回 `command.rejected`。
4. stopped agent 的 restart 会启动成功。

## 10. 验收标准

满足以下条件视为完成：

1. 机器页编辑弹窗在配置读取成功前不可保存。
2. 保存流程严格执行“先写入，再重启”。
3. 配置写入成功但重启失败时，页面有明确提示，且不误报“全部成功”。
4. gateway/client/console 三层新增测试全部通过。

## 11. 风险与后续

风险：

1. 若重启期间有进行中的 turn，可能被中断，需在 UI 提示“重启会影响当前运行”。
2. 若 client 与 gateway 连接抖动，可能出现写入成功但前端未收到 restart 结果。

后续可选增强：

1. 合并为编排接口（单 API 完成写入+重启）。
2. 增加重启前确认弹窗与运行中线程提示。
3. 为配置编辑增加 server-side optimistic lock/version。
