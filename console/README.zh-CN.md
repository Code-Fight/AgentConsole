# Console

`console/` 现在采用的是 `design source first` 策略。

当前阶段的目标不是“参考设计稿重写一版前端”，而是把生成出来的设计源码尽量原样落进项目里，这样后续设计稿再次迭代时，可以快速重新引入，而不需要每次都手工合并 UI。

## 架构分层

当前 Console 分成三层：

1. `src/design-source/`
   上游生成的设计源码镜像。这里应当视为近似只读，尽量保持和最新设计导出一致，不要把 Gateway 业务逻辑塞进去。
2. `src/design-host/`
   最薄的一层宿主兼容层。只负责挂载上游设计源码，以及处理运行时兼容、入口包装、样式接入、构建修补等问题。
3. `src/gateway/`
   后续 Gateway 对接层。HTTP、WebSocket、能力策略、view-model 映射都应该放在这里，而不是写回 `design-source`。

`src/app/` 仍然用于应用装配，例如 router 或 providers，但当前 1:1 设计接入阶段，真正渲染入口是 `src/design-host/`。

## 目录职责

- 上游设计源码：`src/design-source/`
- 宿主兼容层：`src/design-host/`
- Gateway 接入层：`src/gateway/`

## 基本规则

- 不要在 `src/design-source/` 里直接写业务逻辑。
- 不要为了“代码风格统一”就把设计源码重写成我们自己的组件体系。
- 如果设计更新导致构建出问题，优先修 `design-host`、构建配置或样式入口，而不是改坏上游设计源码。
- 如果某个设计控件后端暂时不支持，也不要在 `design-source` 里伪造本地成功逻辑。

## 设计源码更新流程

当上游设计再次更新时，按下面流程处理：

1. 获取最新生成的设计源码。
2. 只覆盖 `src/design-source/` 下对应文件。
3. 保持设计源码尽量原样，不把 Gateway 逻辑混进去。
4. 在 `src/design-host/` 和构建层补最小兼容修复。
5. 如果设计接口或文案变化影响到外围逻辑，再更新：
   - `src/design-host/`
   - `src/gateway/`
   - 测试
6. 跑完整验证。

验证命令：

```bash
cd console
corepack pnpm test
corepack pnpm build
corepack pnpm e2e
cd ..
./testenv/settings-e2e/run.sh
```

如果要看联调效果：

```bash
./testenv/dev-integration/run.sh restart
```

访问：

- `http://localhost:14173`
- `http://localhost:18080`

## 项目内 Skill

本项目已经沉淀了一个用于同步设计源码的 skill：

- 路径：[`../skills/design-source-sync/SKILL.md`](/Users/zfcode/Documents/DEV/CodingAgentGateway/skills/design-source-sync/SKILL.md)
- 名称：`design-source-sync`

它的职责是指导你如何：

- 重新拉取最新设计源码
- 只替换 `src/design-source/`
- 保留宿主兼容层
- 避免把 Gateway 逻辑写进设计源码
- 跑完整验证

## 如何使用这个 Skill

如果你后续要再次同步设计，可以直接这样告诉 Codex：

```text
使用 design-source-sync skill，把最新 design source 同步到 console
```

或者：

```text
Use the design-source-sync skill and refresh the latest design source into console/src/design-source
```

使用这个 skill 时，预期流程应该是：

1. 拉取最新设计源码
2. 替换 `src/design-source/`
3. 保留并检查 `design-host`
4. 跑测试、构建、e2e、settings harness

## 当前阶段说明

当前仓库已经进入 `gateway-backed active console` 阶段：

- 运行入口仍然通过 `src/design-host/` 挂载上游设计源码
- Thread Hub、Workspace、Environment、Settings、Managed Agents 和 Overview Metrics 都已经接入 Gateway
- `src/design-source/` 继续保持展示优先，路由、状态和协议接入仍由 `src/design-host/` 与 `src/gateway/` 承担
- 仍然保留为显式断开态的控件，应视为后续产品缺口，而不是本地 mock 行为
