# Console

`console/` 已完成 feature-oriented frontend migration。当前运行时只保留三层结构：

```text
console/src/
  app/
  features/
  common/
```

## Runtime Architecture

- `src/app/`
  入口、router、providers、顶层 layout。`app/` 只负责装配，不持有 `threads / machines / environment / settings` 的业务真相源。
- `src/features/`
  以产品域组织页面、组件、hooks、API 适配和 feature 内状态。
- `src/common/`
  只放跨 feature 复用能力，例如：
  - `common/api/http.ts`
  - `common/api/ws.ts`
  - `common/config/capabilities.ts`
  - `common/ui/console.css`

当前关键接缝如下：

- Gateway 连接状态由 `src/common/config/gateway-connection-store.ts` 统一维护。
- `src/features/settings/model/gateway-connection-store.ts` 只是兼容 re-export，不再拥有独立状态真相源。
- `common/api/http.ts`、`common/api/ws.ts`、跨 feature runtime 代码都直接依赖该 common store。
- `common/config/capabilities.ts` 负责共享 capability 缓存和 capability gating，并直接订阅该 common store。
- `src/common/ui/console.css` 已自包含 Tailwind/theme/style 基线，不再依赖任何已删除目录中的样式文件。

## Routes

- `/` -> threads hub
- `/threads/:threadId` -> thread workspace
- `/machines` -> machines feature
- `/environment` -> environment feature
- `/settings/*` -> settings feature
- `/overview` -> overview feature
- catch-all -> 重定向到 `/`

## Removed Layers

以下目录已从运行时和保留测试中彻底移除，不得重新引入：

- `src/design-source/`
- `src/design-host/`
- `src/design-bridge/`
- `src/gateway/`
- `src/design/`
- `src/pages/`

同样，旧的聚合入口和 shim 也不再存在：

- `src/app/router.tsx`
- `src/app/providers.tsx`

## Development Rules

- 新的页面行为、view-model、mutation、feature 级状态，放进所属 `features/<domain>/`。
- 只有真正跨 feature 复用的 transport、config、UI primitive、工具函数，才放进 `common/`。
- `app/` 不得重新演化成新的“大一统 host”。
- 不要为了目录统一去重写现有 feature 组件；优先保持现有视觉和行为基线稳定。

## Verification

完成 Console 改动后，至少运行：

```bash
cd console && corepack pnpm test
cd console && corepack pnpm build
cd console && corepack pnpm e2e
./testing/environments/settings-e2e/run.sh
./testing/environments/console-visual-regression/run.sh
./testing/environments/dev-integration/run.sh up
```
