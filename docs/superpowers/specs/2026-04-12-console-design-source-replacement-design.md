# Console Design-Source Replacement 设计文档

## 1. 背景

当前 `console/` 已经具备一套可运行的 Gateway 前端，但它的页面结构、交互层级和视觉语言不再是最新设计稿。

现在已经有一份新的 design source，可直接产出完整的 React 前端原型代码。该 design source 覆盖了新的主信息架构与主要页面，包括：

- Thread Hub
- Thread Workspace / Session Chat
- Machines
- Environment
- Settings
- 以及部分管理型和指标型页面原型

本次工作的目标不是继续在旧 Console 上做局部修补，而是把 `console/` 直接替换为最新 design source 对应的前端体系，并把现有 Gateway 能力真实接入进去。

同时必须考虑后续 design source 还会继续变化，因此这次替换不能做成一次性人工搬运，而要把“后续快速同步设计变更”的机制一起设计进去。

## 2. 目标

本次设计目标如下：

1. 以最新 design source 作为 Console 的唯一界面基准
2. 旧 Console 的页面结构和主要视觉实现不再保留为主路径
3. 已有 Gateway 能力直接接入新界面
4. Gateway 尚未具备的能力在界面中保留结构，但以明确的未接入或禁用态表达
5. 建立一套可重复的 design source 更新机制，方便后续设计继续演进
6. 在 Console README 中记录结构分层、能力映射和设计更新流程，降低下一次改版的理解成本

## 3. 范围

### 3.1 本次纳入范围

- 用 design source 对应的 React 代码体系替换 `console/` 主界面
- 重排 Console 信息架构，使其遵循 design source 的页面组织方式
- 将现有 Gateway HTTP / WebSocket 能力接入新界面
- 梳理新界面中每个功能点与 Gateway 能力的对应关系
- 标记当前缺失能力，并设计后续补齐方案
- 在 README 中补充 design-driven 前端架构和更新说明

### 3.2 本次不纳入范围

- 为了迁就旧 Console 结构而保留旧导航、旧布局、旧壳层
- 为了让界面“看起来完整”而伪造后端不存在的数据
- 在未做明确能力设计前，临时拼出新的 northbound / southbound 协议

## 4. 核心原则

### 4.1 设计稿优先

Console 的页面组织、视觉结构、交互优先级以最新 design source 为准，而不是以当前 `console/` 的历史实现为准。

### 4.2 真实能力优先

只要 Gateway 已具备能力，就必须真实接入，不允许继续保留 mock 流程。

### 4.3 缺失能力显式表达

对当前 Gateway 尚不具备的功能，不删除其界面结构，但必须明确标成：

- 未接入
- 禁用
- 暂不可用

而不是继续使用本地 state 假装已经可用。

### 4.4 设计变更可重复落地

本次替换必须保证未来 design source 改动时，可以通过稳定流程快速同步，而不是重新人工重写一遍 Console。

## 5. 信息架构

Console 以后按 design source 的主结构组织：

1. 默认进入 Thread Hub / Thread Workspace 主路径
2. Machines、Environment、Settings 作为管理页存在
3. 旧 `Overview` 不再作为默认首页；是否保留仅作为低频管理页，由 design source 实际页面结构决定
4. 整体导航与页面层级服从 design source，而不是保留旧 Console 的历史路由习惯

## 6. 分层架构

本次替换后的 Console 采用三层结构。

### 6.1 Design Source Layer

职责：

- 承载由 design source 导出的页面、组件、样式和资源
- 尽量少改，尽量保持接近上游设计源码
- 作为后续 design 更新时的主要覆盖对象

要求：

- 不直接写 Gateway fetch / websocket / mutation 细节
- 不直接实现能力存在性判断
- 不继续混入本地 mock 业务逻辑

建议命名：

- `console/src/design/`

### 6.2 Gateway Adapter Layer

职责：

- 负责把 Gateway 的 HTTP / WebSocket 数据转成 design 页面可消费的 view-model
- 负责把 design 页面动作转成真实的 Gateway 请求
- 负责事件归一化、状态拼装、错误态转换

要求：

- 所有真实后端接入逻辑集中在此层
- 页面不直接依赖底层协议细节
- 适配层要能承接未来 design source 更新，而不要求重写业务接入

建议命名：

- `console/src/gateway/`
- 或 `console/src/integrations/gateway/`

### 6.3 Capability Policy Layer

职责：

- 声明 design source 中每个功能点当前的能力状态
- 统一定义：
  - 已接入
  - 未接入但展示占位
  - 暂时隐藏

要求：

- 页面不在各处零散写 `if backend not ready`
- 所有设计与真实能力之间的差异统一登记和维护

可作为：

- 独立的 policy 配置
- adapter 层中的集中 capability map
- README 中的一部分维护规范

## 7. 主要页面与 Gateway 能力映射

### 7.1 Thread Hub / Thread Workspace

这是第一优先级真实接入页。

当前 Gateway 已具备的能力：

- 线程列表
- 线程详情
- 机器状态
- 实时事件流
- Turn 启动
- `steer`
- `interrupt`
- 审批请求与审批响应

结论：

- Thread Hub 左侧线程目录可真实接入
- Workspace 中的消息流、审批卡片、活跃 Turn 控制可真实接入
- design source 中这一块不应继续保留 mock 数据

### 7.2 Machines

当前已具备：

- 机器列表
- 机器连接状态
- runtime 状态

当前可能缺失：

- Agent 安装
- Agent 删除
- 更细粒度的 Agent 配置编辑和生命周期控制

结论：

- Machines 页可做部分真实接入
- Gateway 尚不支持的 Agent 管理动作以禁用态或未接入态保留

### 7.3 Environment

当前已具备：

- `skill / mcp / plugin` 资源列表
- 部分资源管理动作

风险点：

- design source 中若存在纯前端新增、编辑、安装、删除流程，需要逐项核对 Gateway 当前 API 能力

结论：

- Environment 页可以部分真实接入
- 每个动作都要做 capability mapping，不能默认按 design source 的本地 state 流程实现

### 7.4 Settings

当前已具备：

- Agent 全局默认配置
- 机器覆盖配置
- 配置下发

结论：

- Settings 页应直接真实接入
- 新 UI 只替换交互和视觉，不改变当前 Gateway 作为配置来源的事实
- 不允许再通过本地环境挂载或宿主机真实配置去绕过 Gateway 配置链路

### 7.5 Dashboard / 指标视图

当前 Gateway 不具备成熟的统计与聚合指标能力，例如：

- 请求量
- 成功率
- 平均响应时间
- 活跃 Agent 指标面板

结论：

- 若 design source 中保留这类页面或卡片，第一版应明确标为未接入实时指标
- 第二阶段单独设计统计接口与数据来源，而不是在本次 Console 替换中临时伪造

## 8. 缺失能力分级

为了后续规划，将缺失能力分成三类：

### 8.1 P1 已有能力直接接入

- Thread Hub
- Thread Workspace
- 审批流
- Turn 控制
- 机器状态
- Agent 配置中心

### 8.2 P2 有部分能力但需接口梳理

- Environment 中的资源管理动作
- Machines 中的部分管理动作
- design source 中存在但当前 northbound API 语义不够直接的页面交互

### 8.3 P3 当前纯缺口能力

- 统计类 Dashboard
- 完整 Agent 生命周期管理
- 任何 design source 中仅由 mock 数据支撑的高级操作

## 9. 更新机制

为应对后续 design source 继续变化，Console 必须采用固定更新流程。

### 9.1 标准更新流程

1. 获取最新 design source 导出代码
2. 覆盖 `design source layer` 中的上游页面、组件、样式、资源
3. 审查新增、变更、删除的功能点
4. 更新 capability policy
5. 只在 gateway adapter layer 中补数据映射和事件接入
6. 跑 adapter tests、page integration tests、smoke / e2e tests

### 9.2 禁止事项

- 直接在 design source layer 中散落写 Gateway 请求逻辑
- 用 mock 数据掩盖后端缺口
- 每次设计变更都重新手工改全站业务逻辑

## 10. README 要求

Console README 必须新增专门章节，至少覆盖以下内容：

1. 当前 Console 是以 design source 驱动的前端
2. `design source layer / gateway adapter layer / capability policy layer` 的职责划分
3. 后续 design source 更新时的标准流程
4. 已接入能力与未接入能力的表达原则
5. 哪些目录可以视为“上游设计代码”，哪些目录是本项目自有适配逻辑

README 的目标不是介绍 UI，而是帮助下一次修改设计的人快速理解：

- 这套 Console 为什么这样分层
- 从哪里替换设计代码
- 从哪里接 Gateway
- 如何处理 design 与 capability 的差异

## 11. 测试策略

### 11.1 Adapter 单测

验证：

- Gateway 响应是否正确映射成页面 view-model
- WebSocket 事件是否被正确归一化
- 能力缺口是否被正确标成禁用态或未接入态

### 11.2 页面集成测试

验证：

- 线程切换
- 消息流刷新
- 审批操作
- Turn 启动 / steer / interrupt
- 机器与环境资源页的关键路径

### 11.3 Smoke / E2E

验证：

- Console 能启动
- Console 能连接 Gateway
- 主线程工作台可打开并完成核心交互路径

## 12. 交付顺序

本项目拆成两个连续子项目：

### 子项目 A：Console Full Replace

目标：

- 用 design source 替换现有 Console 主界面
- 把现有 Gateway 能力接入新界面
- 对未具备能力做禁用态 / 未接入态表达
- 在 README 中补全设计更新机制

### 子项目 B：Gateway Capability Gap

目标：

- 基于 design source 中已出现但未接入的功能，逐项补 Gateway 能力
- 产出单独的 northbound API / runtime / client 设计和实施计划

## 13. 风险

### 13.1 上游设计代码频繁变动

缓解方式：

- 保持 design source layer 尽量少改
- 适配逻辑集中在 adapter 层
- 保持 capability policy 独立

### 13.2 设计原型混有大量 mock 逻辑

缓解方式：

- 所有业务 mock 必须在接入时清理
- 缺失能力统一转成未接入态，而不是继续保留假逻辑

### 13.3 一次性替换范围过大

缓解方式：

- 先备份当前分支状态
- 以子项目 A 为上线边界
- 子项目 B 单独立项补能力

## 14. 结论

本次 Console 改版采用 design-source-driven 的整仓替换方案：

- 界面和信息架构以最新 design source 为唯一基准
- 已具备的 Gateway 能力全部真实接入
- 尚未具备的能力显式保留为未接入态
- 通过 `design source layer + gateway adapter layer + capability policy layer` 保障后续设计变更可快速同步
- 同时把更新机制和分层原则写入 Console README，作为后续改版的长期背景文档
