---
mode: plan
cwd: /Users/zfcode/Documents/DEV/CodingAgentGateway
task: 调整线程工作区执行中状态、流式输出与 turn 系统消息展示
complexity: medium
planning_method: builtin
created_at: 2026-04-25T21:44:28+08:00
---

# Plan: Thread Streaming Execution UI

🎯 任务概述
让用户发出对话后，线程工作区立即进入“正在执行”体验；执行期间通过现有 turn delta 事件流式显示 Agent 输出；执行完成或失败后隐藏执行中状态，并去掉 `Turn started` / `Turn completed` 这类控制事件文案。

服务端当前已经具备所需主能力：client 能从 Codex App Server 接收 started/delta/completed/failed 通知，gateway 能按线程广播给 console，并维护 active turn 状态。实现重点应放在 console 的 workspace 状态建模与渲染。

📋 执行计划
1. 梳理线程工作区状态边界：在 `use-thread-workspace.ts` 中区分用户提交中的 HTTP 状态、turn active 状态、流式 agent message 状态，避免只等 HTTP `POST /threads/{id}/turns` 返回后才更新 UI。
2. 调整提交路径：用户发送后先乐观插入 user message、清空输入、设置本地 pending/executing 标记；收到 `turn.started` 后绑定真实 `turnId`，收到 `turn.completed/failed` 后清除执行中状态。
3. 调整 WS 事件消费：保留 `turn.started` 用于状态变更，但不再追加 system message；`turn.completed` 成功时只清状态，不追加完成提示；`turn.failed` 可保留或改成错误提示，避免静默失败。
4. 强化流式输出：继续按 `turn.delta` 聚合到同一条 agent message；如果 delta 先于 `turn.started` 到达，仍能创建 agent message 并驱动执行中状态；如果 HTTP fallback 返回同步 deltas，也按同一聚合逻辑渲染。
5. 调整 `SessionChat` UI：在 `activeTurnId` 或本地 pending 状态存在时显示绿色旋转图标和“正在执行...”文本；正在执行区域与 agent 流式气泡并存，完成后隐藏。
6. 删除或废弃 `toTurnStartedMessage` / 成功完成 message 的使用路径；补充测试确保 UI 不再出现 `Turn started` / `Turn ... completed`。
7. 补充回归测试：覆盖提交后立即显示执行中、delta 流式合并、completed 隐藏执行中、failed 显示错误或失败态、刷新详情时 `activeTurnId` 恢复执行中。
8. 验证：运行相关 vitest 页面/模型测试、`corepack pnpm --dir console build`；如果改动触及实时 UI，建议再用 dev integration 做一次手工 WS 冒烟。

⚠️ 风险与注意事项
- HTTP `startThreadTurn` 当前等待 gateway 命令完成才返回 `turnId`；如果底层启动慢，前端必须先用本地 pending 状态展示“正在执行”，不能依赖 HTTP 返回。
- 当前服务端已有 streaming，但真实 Codex 事件顺序可能出现 delta/completed 与 HTTP 响应交错，前端聚合逻辑需要按 `turnId` 去重，避免重复 agent message。
- `turn.failed` 不应简单隐藏所有提示，否则用户会误以为执行完成但无结果；失败态需要保留明确错误信息。
- CSS 中残留旧状态 class 不等于 UI 一定使用，实施时只清理确认无引用的样式。

📎 参考
- `client/internal/agent/codex/appserver_client.go:460`
- `client/cmd/client/main.go:641`
- `client/cmd/client/main.go:672`
- `client/internal/gateway/session.go:107`
- `gateway/internal/websocket/client_hub.go:271`
- `gateway/internal/websocket/console_hub.go:192`
- `gateway/internal/api/server.go:1288`
- `gateway/internal/api/server.go:1755`
- `console/src/features/threads/hooks/use-thread-workspace.ts:296`
- `console/src/features/threads/model/thread-view-model.ts:288`
- `console/src/features/threads/components/session-chat.tsx:227`
