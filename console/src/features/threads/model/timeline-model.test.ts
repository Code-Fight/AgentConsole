import { expect, test } from "vitest";
import type { AgentTimelineEvent } from "../../../common/api/types";
import {
  mergeTimelineEventIntoMessages,
  timelineEventToApprovalRequired,
} from "./timeline-model";
import type { WorkspaceMessageViewModel } from "./thread-view-model";

function event(overrides: Partial<AgentTimelineEvent>): AgentTimelineEvent {
  return {
    schemaVersion: "agent-timeline.v1",
    eventId: overrides.eventId ?? `event-${overrides.sequence ?? 1}`,
    sequence: overrides.sequence ?? 1,
    threadId: "thread-1",
    turnId: "turn-1",
    eventType: "item.delta",
    ...overrides,
  };
}

test("merges final markdown deltas from one turn into a single agent message", () => {
  const messages = [
    event({
      sequence: 1,
      itemId: "message-1",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "**报告**", appendMode: "append" },
    }),
    event({
      sequence: 2,
      itemId: "message-1",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "\n\n| 产品 | 状态 |\n| --- | --- |", appendMode: "append" },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  expect(messages).toEqual([
    {
      id: "agent:turn-1",
      kind: "agent",
      text: "**报告**\n\n| 产品 | 状态 |\n| --- | --- |",
      turnId: "turn-1",
    },
  ]);
});

test("keeps progress, reasoning, tools, and command output on the same turn message", () => {
  const messages = [
    event({
      sequence: 1,
      itemId: "reasoning-1",
      itemType: "reasoning",
      phase: "analysis",
      content: { contentType: "markdown", delta: "我先分析范围", appendMode: "append" },
    }),
    event({
      sequence: 2,
      itemId: "tool-1",
      itemType: "web_search",
      phase: "progress",
      tool: { kind: "web_search", name: "search" },
      content: { contentType: "text", text: "搜索 OpenAI Agent", appendMode: "append" },
    }),
    event({
      sequence: 3,
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      tool: { kind: "shell", name: "go test" },
      content: { contentType: "terminal", delta: "PASS\n", appendMode: "append" },
    }),
    event({
      sequence: 4,
      itemId: "message-1",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "最终报告", appendMode: "append" },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  expect(messages).toHaveLength(1);
  expect(messages[0]).toMatchObject({
    id: "agent:turn-1",
    kind: "agent",
    text: "最终报告",
    terminalOutput: "PASS\n",
    turnId: "turn-1",
  });
  expect(messages[0]?.progressText).toContain("**分析**");
  expect(messages[0]?.progressText).toContain("我先分析范围");
  expect(messages[0]?.progressText).toContain("**网页搜索**");
  expect(messages[0]?.progressText).toContain("搜索 OpenAI Agent");
  expect(messages[0]?.progressText).toContain("**终端输出**");
});

test("keeps a streaming process message without a synthetic progress label", () => {
  const messages = [
    event({
      sequence: 1,
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", delta: "你", appendMode: "append" },
    }),
    event({
      sequence: 2,
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", delta: "要", appendMode: "append" },
    }),
    event({
      sequence: 3,
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", delta: "我在当前目录创建", appendMode: "append" },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  expect(messages).toHaveLength(1);
  expect(messages[0]?.progressText).toBe("你要我在当前目录创建");
});

test("keeps completed process message snapshots without a synthetic progress label", () => {
  const messages = [
    event({
      sequence: 1,
      eventType: "item.completed",
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: {
        contentType: "markdown",
        text: "我会在当前目录创建名为 `测试A` 的文件。",
        appendMode: "snapshot",
      },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  expect(messages).toHaveLength(1);
  expect(messages[0]?.progressText).toBe("我会在当前目录创建名为 `测试A` 的文件。");
});

test("does not duplicate a completed process snapshot after streaming the same sentence", () => {
  const progressSentence = "我会在当前目录创建名为 `测试B` 的文件，并申请写权限执行创建。";
  const messages = [
    event({
      sequence: 1,
      eventId: "progress-delta",
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", delta: progressSentence, appendMode: "append" },
    }),
    event({
      sequence: 2,
      eventId: "progress-completed",
      eventType: "item.completed",
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", text: progressSentence, appendMode: "snapshot" },
    }),
    event({
      sequence: 3,
      eventId: "command-started",
      eventType: "item.started",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      tool: { kind: "shell", name: "/bin/sh -lc 'touch 测试B'" },
    }),
    event({
      sequence: 4,
      eventId: "approval-requested",
      eventType: "approval.requested",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      approval: { requestId: "approval-1", kind: "command", title: "/bin/sh -lc 'touch 测试B'" },
    }),
    event({
      sequence: 5,
      eventId: "approval-resolved",
      eventType: "approval.resolved",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      approval: { requestId: "approval-1", kind: "command", decision: "accept" },
    }),
    event({
      sequence: 6,
      eventId: "command-completed",
      eventType: "item.completed",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
    }),
    event({
      sequence: 7,
      eventId: "final-message",
      itemId: "final-message",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "已创建文件 `测试B`。", appendMode: "append" },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  const progressText = messages[0]?.progressText ?? "";
  expect(messages).toHaveLength(1);
  expect(progressText.split(progressSentence)).toHaveLength(2);
  expect(progressText).toContain("**终端输出** 开始。");
  expect(progressText).toContain("**审批** /bin/sh -lc 'touch 测试B'");
  expect(progressText).toContain("**审批** 已处理：accept");
  expect(progressText).toContain("**终端输出** 完成。");
  expect(messages[0]?.text).toBe("已创建文件 `测试B`。");
});

test("maps timeline approval requests to pending approval payloads", () => {
  expect(
    timelineEventToApprovalRequired(
      event({
        eventType: "approval.requested",
        itemType: "command",
        itemId: "approval-item",
        approval: {
          requestId: "approval-1",
          kind: "command",
          title: "go test ./...",
          reason: "需要运行测试",
          questions: [{ id: "mode", label: "选择模式", options: ["accept", "decline"] }],
        },
      }),
    ),
  ).toEqual({
    requestId: "approval-1",
    threadId: "thread-1",
    turnId: "turn-1",
    itemId: "approval-item",
    kind: "command",
    reason: "需要运行测试",
    command: "go test ./...",
    questions: [{ id: "mode", text: "选择模式", options: ["accept", "decline"] }],
  });
});

test("keeps unknown displayable provider events as compact process rows", () => {
  const messages = [
    event({
      itemType: "unknown",
      phase: "progress",
      raw: { provider: "other-agent", method: "custom/event" },
      content: { contentType: "text", text: "自定义事件", appendMode: "append" },
    }),
  ].reduce(mergeTimelineEventIntoMessages, [] as WorkspaceMessageViewModel[]);

  expect(messages).toEqual([
    {
      id: "agent:turn-1",
      kind: "agent",
      text: "",
      progressText: "**unknown**\n\n自定义事件",
      turnId: "turn-1",
    },
  ]);
});
