# Console Feature-Oriented Frontend Architecture 设计文档

> 实施状态（2026-04-19）：迁移已完成。当前 Console 运行时只保留 `app/`、`features/`、`common/`，`design-source/`、`design-host/`、`design-bridge/`、`gateway/`、`design/`、`pages/` 已全部删除。

## 1. 背景

`console/` 在迁移开始前，运行时和文档存在明显偏差。

文档和 README 仍将当前 Console 描述为由 `design-source`、`design-bridge`、`design-host`、`gateway` 组成的多层结构，并默认：

- `design-source` 需要保持接近上游镜像
- 上游 UI 后续仍可能整包替换
- 本地运行时逻辑需要围绕这些层做保护和接缝管理

迁移前的代码现状已经不是一个稳定的分层架构：

- 当前主入口仍走 `console/src/main.tsx -> design-host -> design-source/App`
- `design-bridge/` 基本尚未成为真实运行时接缝层
- 多个 `design-source` 组件已经直接依赖 `gateway` hook、`common/api` 甚至 host 状态
- 仓库里同时又存在一套更常规的 `app/router + pages + design views + gateway hooks` 结构
- `design-host/use-console-host.ts` 已经承担了大量全局和业务聚合职责，成为实际上的“大一统运行时”

这导致当前 Console 同时承受三种结构心智：

1. 面向上游同步的镜像分层
2. 面向当前运行时的 host 聚合分层
3. 面向页面功能的常规前端分层

这种混合结构会让后续演进越来越难。

本次设计明确放弃“`design-source` 可整包替换 / 保留上游同步保护区”的目标，并已将 Console 收敛为一个普通、稳定、以产品能力为中心组织的前端工程。

## 2. 目标

本次设计目标如下：

1. 放弃 `design-source` 作为长期镜像层的架构前提
2. 将 Console 收敛为标准的前端应用结构，而不是围绕来源层组织
3. 以 `threads / machines / environment / settings` 四个产品域作为主组织轴
4. 将当前散落在 `design-host`、`design-source`、`gateway`、`design`、`pages` 中的业务逻辑按功能重新归位
5. 拆除 `use-console-host.ts` 这种大一统聚合器，恢复 feature 自治
6. 在迁移过程中保证：
   - 用户可见功能不退化
   - 页面路由和 Gateway contract 不漂移
   - 页面视觉呈现与迁移前保持一致
7. 建立一套正式的质量门禁，把功能回归和视觉回归都纳入阶段验收

## 3. 非目标

本次设计不追求以下事项：

- 不在本次重构中主动重设计 Console 的视觉风格
- 不在本次重构中调整 northbound API 语义
- 不在本次重构中引入新的产品页面或产品流程
- 不为了目录统一而重写所有现有 UI 组件
- 不把 Console 升级为完整 FSD 教条式结构
- 不保留对上游 Git 仓库整包同步的兼容诉求

## 4. 决策总结

### 4.1 放弃来源层架构

Console 不再围绕以下目录作为主结构组织：

- `design-source/`
- `design-bridge/`
- `design-host/`
- `gateway/`

这些目录已经被删除。它们不再是兼容层，也不允许作为未来新增能力的落点。

### 4.2 采用 Feature-Oriented Frontend Architecture

当前 Console 采用“应用装配层 + 业务 feature 层 + 公共能力层”的结构：

```text
console/src/
  app/
  features/
  common/
```

其中 `features/` 是唯一主结构，直接围绕产品域组织。

### 4.3 `common` 取代 `shared`

本次结构命名统一使用 `common` 概念，不使用 `shared`。

原因：

- 与仓库当前语言习惯更一致
- 更符合“跨 feature 复用能力”的本地表达
- 便于在本仓库语境中和其他前端目录命名保持一致

### 4.4 以当前 Console 视觉输出作为唯一基线

本次重构新增视觉回归环境，并明确：

- 基线不是设计稿
- 基线不是未来想要的样式
- 基线就是“迁移开始前当前 Console 的实际呈现”

后续每迁完一个 feature，都必须证明：

- 功能行为一致
- 页面视觉基线一致

## 5. 当前问题拆解

### 5.1 运行时真相源不清晰

当前 Console 的运行时真相源分散在：

- `design-host/use-console-host.ts`
- `gateway/use-thread-hub.ts`
- `gateway/use-thread-workspace.ts`
- `gateway/use-settings-page.ts`
- `gateway/use-environment-page.ts`
- `gateway/use-machines-page.ts`
- 部分 `design-source` 组件内部的本地状态和请求逻辑

这导致：

- 全局和局部状态边界模糊
- 路由、数据装配、mutation、UI 状态混在一起
- 单个文件过大，变更范围难以控制

### 5.2 组件职责漂移

当前多个组件不再是“只负责呈现”的组件，而是混合了：

- Gateway 请求
- 连接状态判断
- capability gating
- 表单数据拼装
- 业务状态恢复

这使得目录名和真实职责不匹配。

### 5.3 同时维护两套前端组织方式

当前仓库一方面保留 `design-source` 主路径，另一方面又保留：

- `app/router`
- `pages/`
- `design/`

这相当于并存两套半成品结构，后续维护成本会持续上升。

### 5.4 质量保障只覆盖功能，不覆盖视觉一致性

当前测试和环境主要覆盖：

- 单元/组件行为
- build
- smoke e2e
- settings 专项链路

但缺少一个面向“页面样式一致性”的正式门禁。对于这种以结构重组为主的迁移，这个缺口是不可接受的。

## 6. 目标架构

### 6.1 目标目录结构

```text
console/src/
  app/
    entry/
      main.tsx
    router/
      index.tsx
    providers/
      index.tsx
    layout/
      app-shell.tsx
      connection-gate.tsx

  features/
    threads/
      pages/
      components/
      hooks/
      api/
      model/
    machines/
      pages/
      components/
      hooks/
      api/
      model/
    environment/
      pages/
      components/
      hooks/
      api/
      model/
    settings/
      pages/
      components/
      hooks/
      api/
      model/

  common/
    api/
    ui/
    lib/
    config/
    types/
```

### 6.2 `app/` 职责

`app/` 只负责应用装配，不持有业务真相源。

具体包括：

- React 入口
- Router 创建
- 全局 Provider
- 顶层 layout / shell
- 连接门禁显示

禁止事项：

- 不在 `app/` 中拼接 feature 级 view-model
- 不在 `app/` 中实现页面级 mutation
- 不在 `app/` 中承载线程、环境、设置、机器等 feature 内部状态

### 6.3 `features/` 职责

`features/` 是 Console 的主结构。

每个 feature 目录都要对自己的：

- 页面装配
- 组件
- hook
- API 适配
- 领域状态与 view-model

负责。

这意味着：

- 页面相关逻辑优先留在所属 feature 内部
- 不再使用一个跨 feature 的“统一 Gateway 页面 hook 目录”作为长期结构

### 6.4 `common/` 职责

`common/` 只放真正跨 feature 的通用能力，例如：

- `common/api/http.ts`
- `common/api/ws.ts`
- 通用 UI primitives
- 通用格式化和工具方法
- 通用类型定义

禁止事项：

- 不把 feature 专属 view-model 放进 `common`
- 不把某个页面临时放不下的逻辑丢进 `common`
- 不把 `common` 当作新的“超大杂物间”

### 6.5 各 feature 边界

#### `features/threads`

负责：

- thread hub
- thread workspace
- turn 启动、steer、interrupt
- approval flow
- last thread restore
- thread 路由切换时的页面状态恢复

#### `features/settings`

负责：

- Gateway connection 配置
- console preferences
- global default config
- machine override config
- apply config to machine

说明：

- 连接门禁是否显示属于 `app/layout`
- 设置页内部的字段和保存逻辑属于 `features/settings`

#### `features/environment`

负责：

- skills / mcps / plugins 列表
- 资源详情展开
- 表单打开和编辑状态
- sync catalog / restart bridge / install / enable / disable / delete
- capability gating 映射

#### `features/machines`

负责：

- 机器列表
- runtime 状态
- agent 安装和删除
- agent 配置读取和编辑
- 机器级 runtime 操作

## 7. 现有代码到目标结构的映射原则

### 7.1 保留并下沉到 `common`

以下内容原则上可以保留并下沉到 `common/`：

- 当前 `console/src/common/api/http.ts`
- 当前 `console/src/common/api/ws.ts`
- 当前通用 transport/helper

### 7.2 拆入各 feature

以下内容需要拆入各自 feature，而不是继续停留在 `gateway/`：

- `use-thread-hub.ts`
- `use-thread-workspace.ts`
- `use-settings-page.ts`
- `use-console-preferences.ts`
- `use-environment-page.ts`
- `use-machines-page.ts`
- `thread-view-model.ts`

### 7.3 清理大聚合器

`console/src/design-host/use-console-host.ts` 不保留为长期结构。

它需要被拆分为：

- `app/layout` 所需的最小全局状态
- `features/threads` 的线程相关状态与动作
- 少量真正跨 feature 的通用逻辑

### 7.4 组件逻辑回归 feature

当前 `design-source` 和部分 `design` 组件里混入的：

- API 请求
- capability 判断
- mutation 组装
- 本地业务状态

都需要抽离到对应 feature 的 `hooks/api/model`。

## 8. 实施计划

### 8.1 Phase 0: 基线冻结

目标：

- 冻结当前功能和视觉基线

工作内容：

- 识别关键行为用例
- 清理与目录结构强耦合的测试
- 补齐行为测试
- 新增视觉回归环境
- 为关键页面生成第一版基线截图

产出：

- 行为测试基线
- 视觉截图基线
- 阶段性验收清单

### 8.2 Phase 1: 新骨架落地

目标：

- 建立新的 `app/ + features/ + common/` 主结构

工作内容：

- 建立目录骨架
- 迁移通用 API 基础设施
- 建立新的 app 入口、router、providers、layout
- 在不改变用户行为的前提下切换顶层装配结构

产出：

- 新主结构可用
- 当前页面仍可正常运行

### 8.3 Phase 2: 迁移 `threads`

目标：

- 让线程主路径脱离 `design-host/use-console-host.ts`

工作内容：

- 迁移 hub / workspace / turn / approvals / restore
- 建立 `features/threads` 独立页面、组件、hooks、api、model
- 拆除线程相关的 host 聚合逻辑

产出：

- `threads` 成为独立 feature
- 主路径运行时真相源明确

### 8.4 Phase 3: 迁移 `settings`

目标：

- 将设置相关逻辑从旧层结构迁入 `features/settings`

工作内容：

- 迁移 Gateway connection
- 迁移 console preferences
- 迁移 global default / machine override / apply machine
- 保持未连接时 `/settings` 可访问

产出：

- `settings` feature 独立运行

### 8.5 Phase 4: 迁移 `environment`

目标：

- 将环境资源管理逻辑迁入 `features/environment`

工作内容：

- 迁移资源列表和详情
- 迁移 MCP / skill / plugin 表单
- 迁移 capability gating 和 mutation 路径

产出：

- `environment` feature 独立运行

### 8.6 Phase 5: 迁移 `machines`

目标：

- 将机器和 agent 管理逻辑迁入 `features/machines`

工作内容：

- 迁移机器列表
- 迁移 agent 安装/删除
- 迁移 agent config 读取/保存
- 把组件内部直接请求逻辑收回 feature hook/api

产出：

- `machines` feature 独立运行

### 8.7 Phase 6: 删除遗留层并收尾

目标：

- 删除旧架构目录，使新结构成为唯一真相源

工作内容：

- 删除 `design-source/`
- 删除 `design-host/`
- 删除 `design-bridge/`
- 删除旧 `gateway/` 聚合目录
- 删除已被吸收的旧 `design/`、`pages/`
- 清理重复类型和 adapter
- 更新 README 和目录说明

产出：

- 单一结构落地完成
- `common/config/gateway-connection-store.ts` 已成为跨 feature 的 Gateway 连接状态真相源
- `common/api/http.ts` / `common/api/ws.ts` 已直接依赖该 common store
- 非 `features/settings/**` 的 active runtime 与跨 feature 测试均已直接依赖该 common store，而不是经过 settings shim
- `common/config/capabilities.ts` 已成为共享 capability 真相源
- `/overview` 已迁入 feature runtime，catch-all 已回收为 `/` 重定向

## 9. 质量保障设计

本次重构必须通过三条正式质量链路。

### 9.1 行为回归链路

每阶段必须运行：

- `cd console && corepack pnpm test`

行为测试原则：

- 测用户行为，不测目录结构
- 测页面交互结果，不测 import 来源

最低覆盖场景包括：

- 未配置 Gateway 时显示连接门禁
- `/settings` 在未连接时仍可访问
- 线程列表加载
- thread workspace 加载
- prompt submit / steer / interrupt
- approval decision
- last thread restore
- settings 保存
- environment 资源增删改
- machines agent config 读写

### 9.2 功能联调链路

每阶段必须运行：

- `cd console && corepack pnpm build`
- `cd console && corepack pnpm e2e`

若阶段涉及设置和配置链路，还需要运行：

- `./testing/environments/settings-e2e/run.sh`

若阶段涉及跨服务真实集成验证，还应运行：

- `./testing/environments/dev-integration/run.sh`

### 9.3 视觉回归链路

新增环境：

```text
testing/environments/console-visual-regression/
```

职责：

- 提供稳定可复现的视觉基线环境
- 不依赖实时联调波动
- 固定 mock Gateway 返回、路由、时间、时区、语言、viewport、字体

建议截图场景至少包括：

- 连接缺失弹窗
- 线程列表页
- 线程工作区页
- 待审批工作区页
- Settings 页
- Environment 页
- Environment 表单打开态
- Machines 页
- Agent config 编辑弹窗

截图基线原则：

- 基线来源是“迁移开始前当前 Console 的实际页面效果”
- 不是设计稿
- 不是理想未来状态
- 每迁完一个 feature，都必须通过视觉 diff

### 9.4 视觉环境与功能环境的分工

视觉回归环境不替代功能联调环境。

分工如下：

- 功能联调环境验证真实功能链路是否可用
- 视觉回归环境验证页面呈现是否与基线一致

两者必须并行存在，不能用其中之一替代另一条质量链路。

## 10. 阶段验收标准

每个 Phase 结束都必须同时满足：

1. 当前主入口可正常运行
2. `pnpm test` 通过
3. `pnpm build` 通过
4. `pnpm e2e` 通过
5. 若适用，专项联调环境通过
6. visual regression 通过
7. 本阶段涉及的页面截图与基线一致
8. 若确有预期视觉变化，必须显式更新基线并说明原因
9. 没有新增跨层反向依赖
10. 没有继续扩大类似 `use-console-host.ts` 这样的聚合器

## 11. 风险与控制策略

### 11.1 大爆炸迁移风险

风险：

- 同时重构目录、状态和页面，容易造成回归集中爆发

控制策略：

- 严格按 feature 分阶段迁移
- 一次只完成一个主主题

### 11.2 样式回归与功能回归混淆

风险：

- 重构期间视觉变化可能被误判为功能变化，或反之

控制策略：

- 把视觉回归环境独立成正式链路
- 所有非预期样式变化都必须被拦截

### 11.3 临时兼容层长期化

风险：

- 为了迁移便利引入的 adapter 可能被永久保留

控制策略：

- 所有兼容层都必须绑定到后续阶段删除
- Phase 6 以删除旧层为硬性目标

### 11.4 `common` 失控膨胀

风险：

- 重构后 `common` 重新变成全局杂物间

控制策略：

- 只允许跨 feature 复用能力进入 `common`
- feature 专属状态和 view-model 必须留在 feature 内

## 12. 结论

当前 Console 的主结构已经不再围绕“上游来源层”组织，而是围绕产品域组织。

本次设计的最终结果是：

- `app/` 负责应用装配
- `features/` 负责产品能力
- `common/` 负责公共能力
- 旧 `design-source/design-host/design-bridge/gateway/design/pages` 层已全部退出运行时
- 当前 Console 的视觉输出被冻结为迁移基线
- 功能回归和视觉回归都成为正式门禁

至此，Console 已经从“混合态四层遗留架构”收敛为一个成熟、稳定、可持续演进的 feature-oriented 前端项目。
