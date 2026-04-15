# Machine Identity And Name 设计文档

## 1. 背景

当前系统里 `machineId` 既承担路由标识，又被直接展示在 Console 中。

这导致两个问题：

- 机器管理页展示的值不友好，常见为默认值 `machine-01`
- `machineId` 的来源依赖客户端启动参数，不适合作为“安装级稳定身份”

本次设计要把“机器唯一标识”和“机器展示名”分开处理，同时尽量控制改动范围，保留现有 API 字段名和路由模型。

## 2. 目标

- 保留现有字段名 `machineId`
- 调整 `machineId` 的语义：它表示客户端安装级稳定机器标识，而不是可配置展示名
- `machineId` 由客户端首次启动时生成并持久化，后续重启复用
- Console 和 Gateway 对外展示统一使用 `machine.name`
- `machine.name` 优先来自 `MACHINE_NAME`，未配置时默认取当前机器 hostname

## 3. 非目标

- 不把所有 API 字段名从 `machineId` 改成别的名字
- 不引入 Gateway 侧的第二套机器主键
- 不增加人工分配机器令牌或额外运维步骤
- 不做历史数据迁移工具；旧实例在新版本 client 重连后自然完成身份与名称更新

## 4. 核心决策

### 4.1 `machineId` 继续存在，但含义调整

- 协议、API、索引、路由中的字段名继续使用 `machineId`
- 语义改为“客户端安装级稳定 ID”
- Gateway 不再为机器单独分配另一套内部 ID

### 4.2 `machineId` 由 client 首次生成并持久化

- client 启动时先读取本地身份文件
- 若文件存在，则复用其中的 `machineId`
- 若文件不存在，则生成新的 `machineId`
- 生成格式采用 `hostname + "_" + uuid`

示例：

```text
mbp-zfcode_550e8400-e29b-41d4-a716-446655440000
```

生成规则：

- `hostname` 取首次生成时的 hostname，并做安全归一化
- `uuid` 使用标准随机 UUID
- `hostname` 部分只参与首次生成，后续 hostname 变化不影响既有 `machineId`

### 4.3 `machine.name` 作为友好展示名

每次 client 启动时计算运行时展示名：

1. 优先读取环境变量 `MACHINE_NAME`
2. 若未配置，则读取当前系统 hostname
3. 若 hostname 读取失败，则回退到 `machineId`

该值通过机器快照或注册消息同步到 Gateway，并体现在 Console 的机器展示中。

## 5. 客户端设计

### 5.1 配置来源

client 配置调整为：

- 删除“通过环境变量 `MACHINE_ID` 指定机器身份”的默认路径
- 保留内部配置字段 `MachineID`，但它来自本地身份文件而不是环境变量
- 新增 `MachineName` 运行时配置值

具体行为：

- `MachineID`：从 identity store 中读取；首次缺失时生成并写回
- `MachineName`：`MACHINE_NAME` > 当前 hostname > `MachineID`

### 5.2 identity store

client 新增本地身份文件，用于持久化：

- `machineId`
- 可选元数据，例如首次生成时间、首次 hostname

建议位置：

```text
~/.code-agent-gateway/machine.json
```

如果现有目录结构更适合复用，也可以放入现有 client 本地状态目录，但要求：

- 单机唯一
- 重启可复用
- 不依赖容器临时文件系统以外的短生命周期路径

### 5.3 注册与快照

client 连接 Gateway 后：

- 注册消息中的 `machineId` 使用持久化后的稳定值
- 机器快照中的 `machine.id` 使用相同值
- 机器快照中的 `machine.name` 使用运行时解析出的友好名称

因此：

- 路由、线程归属、环境归属仍统一使用 `machineId`
- 展示统一使用 `machine.name`

## 6. Gateway 设计

### 6.1 注册模型

Gateway 不再承担“为机器生成另一套内部 ID”的职责。

Gateway 接收 client 上报的 `machineId` 后：

- 把它当作唯一机器标识
- 用它绑定 websocket 连接
- 用它归属线程、环境资源和审批请求

这意味着 Gateway 现有按 `machineId` 路由的逻辑可以大部分保留，仅需调整对展示名的处理方式。

### 6.2 机器名称更新

Gateway 应接受 client 上报的最新 `machine.name` 并覆盖当前机器展示名。

这样支持：

- 用户新增或修改 `MACHINE_NAME`
- 机器 hostname 发生变化
- client 升级后首次补齐友好名称

### 6.3 断线重连

因为 `machineId` 来源于本地持久化文件，同一台机器在重启或网络重连后会继续使用同一个 `machineId`。

因此以下关系可以保持稳定：

- websocket 路由归属
- 线程所属机器
- 环境资源所属机器
- 审批请求所属机器
- 机器级配置和设置页关联

## 7. Console 设计

Console 继续使用已有机器模型：

- `machine.id` 作为系统标识展示和调试信息
- `machine.name` 作为默认可读名称

展示规则统一为：

1. 优先显示 `machine.name`
2. 若 `machine.name` 为空，则回退到 `machine.id`

适用范围包括：

- 机器管理页
- 线程列表中的机器标签
- 工作区中的机器信息
- 设置页机器选择器

## 8. 兼容性与迁移

### 8.1 协议兼容

- 保留 `machineId` 字段名，不引入新的对外字段名
- 现有前后端接口和 websocket envelope 结构可以保持稳定
- 主要变化是 client 的 `machineId` 来源，以及 `machine.name` 的更新逻辑

### 8.2 旧 client 行为

旧版本 client 仍可能继续上报 `machineId = machine-01`、`machine.name = machine-01`。

在新版本 client 发布后：

- 首次启动会生成并持久化新的稳定 `machineId`
- 同时上报更友好的 `machine.name`

这会让该机器以新身份重新出现在 Gateway 中。该行为是可接受的，因为旧实现本身没有可靠的安装级身份保证。

说明：

- 本次不做“把历史 `machine-01` 自动映射为新安装身份”的兼容桥接
- 若后续需要平滑迁移旧机器身份，可单独设计一次性迁移策略

## 9. 异常与边界处理

- 若 hostname 读取失败：
  - `machine.name` 回退为 `machineId`
  - `machineId` 中的 hostname 前缀使用固定回退值，例如 `machine`
- 若 identity store 无法写入：
  - client 启动失败，并明确报错
  - 不允许以临时随机 `machineId` 继续运行，否则会破坏稳定身份语义
- 若 identity store 内容损坏：
  - client 报错并拒绝启动
  - 由人工删除损坏文件后重新生成

## 10. 测试策略

### 10.1 client

- 未发现 identity store 时会生成 `hostname_uuid` 格式的 `machineId`
- 已存在 identity store 时复用既有 `machineId`
- 未配置 `MACHINE_NAME` 时取当前 hostname
- 配置 `MACHINE_NAME` 时优先使用环境变量
- hostname 读取失败时正确回退

### 10.2 gateway

- 注册时使用 client 上报的 `machineId` 作为连接归属键
- 同一 `machineId` 重连后覆盖旧连接并保持机器归属稳定
- 收到新的 `machine.name` 后机器列表和详情正确更新

### 10.3 console

- 机器管理页优先显示 `machine.name`
- 各页面在 `name` 缺失时正确回退到 `id`
- 线程、设置、工作区中机器标签保持一致展示策略

## 11. 实施范围

预计会涉及以下区域：

- client 配置读取与本地状态初始化
- client 注册和机器快照构造
- gateway 机器注册与快照处理测试
- console 机器展示相关测试
- dev/test 环境文档和启动变量说明

## 12. 最终结论

本次方案保留现有 `machineId` 字段名和系统路由模型，但把其语义收敛为“客户端安装级稳定机器身份”，由 client 首次生成并持久化。

与此同时，界面展示从 `machineId` 切换到更友好的 `machine.name`，其值由 `MACHINE_NAME` 或当前 hostname 提供。

这样可以在不重构整体协议和索引结构的前提下，解决当前机器标识不稳定和展示不友好的问题。
