# Dev Integration Multi-Client 设计文档

## 1. 背景与目标

`testenv/dev-integration` 当前只会拉起一个 `client` 容器，并默认通过 `MACHINE_NAME=Dev Integration Client` 在 Gateway 和 Console 中展示为单台机器。

这对基础联调足够，但无法覆盖以下场景：

- 多 client 同时在线时的 Gateway 注册与 Console 展示
- `MACHINE_NAME` 显式指定与 hostname 回退并存的行为
- 多个 client 本地状态隔离，避免 `machineId` 互相覆盖

本次功能目标是在一次 `dev-integration` 启动中同时拉起 3 个 client，并让三台机器都稳定注册到 Gateway：

1. 默认友好名 client：展示名固定为 `Dev Integration Client`
2. hostname 回退 client：不传 `MACHINE_NAME`，依赖 client 现有逻辑回退到 hostname
3. 显式友好名 client：展示名固定为 `Not Agent`

## 2. 设计边界

本次改动只覆盖 `testenv/dev-integration` 联调环境，不修改 `client`、`gateway`、`console` 的运行时业务逻辑。

允许修改的范围：

- `testenv/dev-integration/docker-compose.yml`
- `testenv/dev-integration/run.sh`
- `testenv/dev-integration/README.md`

明确非目标：

- 不新增 Gateway 或 client 的协议字段
- 不改动 `MachineName` 的解析优先级
- 不修改 Console 的机器展示逻辑
- 不把多 client 能力推广到 `testenv/settings-e2e` 之外的其他环境，除非现有配置已直接依赖 `dev-integration` 的单 client 结构

## 3. 方案概览

### 3.1 Compose 层

`docker-compose.yml` 中不再只有一个 `client` service，而是显式声明三个独立 client service：

- `client-default`
- `client-hostname`
- `client-not-agent`

三个 service 共用相同的镜像构建方式、Gateway 地址和 runtime 默认值，但在名称相关配置上存在差异：

- `client-default`
  - 传入 `MACHINE_NAME=Dev Integration Client`
- `client-hostname`
  - 不传 `MACHINE_NAME`
  - 显式设置 docker `hostname: hostname-fallback-client`
- `client-not-agent`
  - 传入 `MACHINE_NAME=Not Agent`

### 3.2 本地状态隔离

每个 client 都必须挂载独立 volume 到 `/home/appuser/.code-agent-gateway`，避免共享同一份 `machine.json`：

- `client-default-state`
- `client-hostname-state`
- `client-not-agent-state`

这样三个容器首次启动时会各自生成不同的 `machineId`，后续重启时也能保持稳定。

### 3.3 运行脚本

`run.sh` 从“等待一台机器注册”改为“等待三台目标机器全部在线”。

脚本内维护一组预期机器名：

- `Dev Integration Client`
- `hostname-fallback-client`
- `Not Agent`

对于 `client-hostname`，脚本不从环境变量推断名字，而是直接按 compose 中固定的 docker hostname 等待。这样既保持“没有传 `MACHINE_NAME`，由 client 代码回退 hostname”的语义，又让联调行为稳定且可预测。

启动成功后，脚本打印三台机器各自的：

- Machine Name
- Machine ID

`restart` 的输出也同步改成多 client 汇总结果。

### 3.4 文档

README 更新为多 client 说明，明确：

- 启动后会看到三台机器
- 三台机器分别对应的命名来源
- `client-hostname` 的名字来自 hostname 回退，不来自 `MACHINE_NAME`
- 如需自定义显式名称，可以修改 compose 或通过环境变量覆盖显式命名的两个 client

## 4. 数据流与行为

### 4.1 注册链路

三台 client 的注册链路保持不变：

1. 容器启动 client 进程
2. client 读取本地状态目录中的 `machine.json`，不存在则生成新的 `machineId`
3. client 解析 `machineName`
4. client 通过 `client.register` 把 `machineId` 和 `machine.name` 发给 Gateway
5. Gateway 将机器写入 registry
6. Console 通过 `/machines` 和 WebSocket 看到三台机器

### 4.2 名称解析

三台机器的名称来源分别是：

- `client-default`：环境变量 `MACHINE_NAME`
- `client-hostname`：`os.Hostname()`
- `client-not-agent`：环境变量 `MACHINE_NAME`

这意味着本次实现复用现有 client 配置逻辑，不引入新的“测试环境专用命名规则”。

## 5. 错误处理与可观测性

`run.sh` 的等待逻辑需要从单值等待改成集合等待，并保持超时失败信息清晰：

- 若 Gateway/Console 未就绪，继续沿用现有 HTTP wait 行为
- 若三台机器中有任意一台未按时上线，超时报错应指出缺失的机器名集合
- 查询 `/machines` 时若返回数据不完整，脚本继续轮询直到超时

输出策略：

- `up` 成功后，逐行输出三台机器的 name/id 映射
- `restart` 成功后，输出相同摘要
- `logs`、`ps`、`down` 保持原命令接口不变

## 6. 测试与验证

本功能优先通过配置和脚本级验证落地。

### 6.1 自动化验证

至少覆盖以下检查：

- `docker-compose.yml` 中存在三个 client service
- `client-hostname` 不包含 `MACHINE_NAME`
- 三个 client 使用独立 state volume
- `run.sh` 等待三台机器而不是单台机器
- `run.sh` 能按机器名查回三条 machine 记录

若仓库内已有适合的 shell/配置测试入口，则优先补测试；若没有，则至少通过 focused manual verification 记录行为证据。

### 6.2 手工验证

在 `dev-integration` 环境中执行：

1. `./testenv/dev-integration/run.sh up`
2. `curl http://localhost:18080/machines`
3. 确认返回中存在三台在线机器：
   - `Dev Integration Client`
   - `hostname-fallback-client`
   - `Not Agent`
4. 打开 Console，确认 Machines/Threads 相关机器展示可见三台机器
5. `./testenv/dev-integration/run.sh down`

## 7. 方案取舍

本方案刻意没有采用“让第二台 client 使用随机容器 hostname”的做法。随机 hostname 虽然更接近“完全不配任何名称”的纯粹状态，但会导致：

- 每次启动名字变化
- `run.sh` 无法按稳定名字等待
- 联调和文档说明变得脆弱

因此这里选择固定 docker hostname，同时不传 `MACHINE_NAME`。这样保持了目标行为的本质验证点：第二台 client 的名字确实来自 hostname 回退，而不是来自显式配置。

## 8. 实施摘要

实现时按以下顺序推进：

1. 先写失败测试或最小化验证脚本断言，锁定多 client compose 结构与 `run.sh` 预期行为
2. 修改 compose，拆出三个 client service 与三个 state volume
3. 修改 `run.sh`，支持等待和打印三台机器
4. 更新 README
5. 运行 focused verification，记录三台机器同时上线的证据
