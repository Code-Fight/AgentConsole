import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import type { AgentTimelineEvent, EventEnvelope } from "../../../common/api/types";
import { useThreadWorkspace } from "./use-thread-workspace";

const connectConsoleSocketMock = vi.hoisted(() => vi.fn());
const supportsCapabilityMock = vi.hoisted(() => vi.fn());

const getMachineDetailMock = vi.hoisted(() => vi.fn());
const getThreadDetailMock = vi.hoisted(() => vi.fn());
const getThreadRuntimeSettingsMock = vi.hoisted(() => vi.fn());
const interruptThreadTurnMock = vi.hoisted(() => vi.fn());
const respondToApprovalMock = vi.hoisted(() => vi.fn());
const resumeThreadMock = vi.hoisted(() => vi.fn());
const startThreadTurnMock = vi.hoisted(() => vi.fn());
const steerThreadTurnMock = vi.hoisted(() => vi.fn());
const updateThreadRuntimeSettingsMock = vi.hoisted(() => vi.fn());

vi.mock("../../../common/api/ws", () => ({
  connectConsoleSocket: connectConsoleSocketMock,
}));

vi.mock("../../../common/config/capabilities", () => ({
  supportsCapability: supportsCapabilityMock,
}));

vi.mock("../api/thread-api", () => ({
  getMachineDetail: getMachineDetailMock,
  getThreadDetail: getThreadDetailMock,
  getThreadRuntimeSettings: getThreadRuntimeSettingsMock,
  interruptThreadTurn: interruptThreadTurnMock,
  respondToApproval: respondToApprovalMock,
  resumeThread: resumeThreadMock,
  startThreadTurn: startThreadTurnMock,
  steerThreadTurn: steerThreadTurnMock,
  updateThreadRuntimeSettings: updateThreadRuntimeSettingsMock,
}));

type MessageHandler = (event: MessageEvent<string>) => void;

const THREAD_DETAIL_RESPONSE = {
  thread: {
    threadId: "thread-1",
    machineId: "machine-1",
    status: "idle",
    title: "Thread 1",
  },
  activeTurnId: null,
  pendingApprovals: [],
  messages: [],
};

const MACHINE_DETAIL_RESPONSE = {
  machine: {
    id: "machine-1",
    name: "Machine 1",
    status: "online",
    runtimeStatus: "running",
    agents: [],
  },
};

const RUNTIME_SETTINGS_RESPONSE = {
  settings: {
    threadId: "thread-1",
    preferences: {
      model: "gpt-5.4",
      sandboxMode: "workspace-write",
    },
    options: {
      models: [{ id: "gpt-5.4", displayName: "GPT-5.4", isDefault: true }],
      approvalPolicies: [],
      sandboxModes: ["workspace-write"],
    },
  },
};

beforeEach(() => {
  connectConsoleSocketMock.mockReset();
  supportsCapabilityMock.mockReset();
  getMachineDetailMock.mockReset();
  getThreadDetailMock.mockReset();
  getThreadRuntimeSettingsMock.mockReset();
  interruptThreadTurnMock.mockReset();
  respondToApprovalMock.mockReset();
  resumeThreadMock.mockReset();
  startThreadTurnMock.mockReset();
  steerThreadTurnMock.mockReset();
  updateThreadRuntimeSettingsMock.mockReset();

  getThreadDetailMock.mockResolvedValue(THREAD_DETAIL_RESPONSE);
  getMachineDetailMock.mockResolvedValue(MACHINE_DETAIL_RESPONSE);
  getThreadRuntimeSettingsMock.mockResolvedValue(RUNTIME_SETTINGS_RESPONSE);
  supportsCapabilityMock.mockImplementation((capability: string) => capability === "startTurn");
});

function emitEnvelope(handler: MessageHandler, envelope: EventEnvelope) {
  act(() => {
    handler(
      new MessageEvent<string>("message", {
        data: JSON.stringify(envelope),
      }),
    );
  });
}

function timelineEvent(overrides: Partial<AgentTimelineEvent>): AgentTimelineEvent {
  return {
    schemaVersion: "agent-timeline.v1",
    eventId: overrides.eventId ?? `event-${overrides.sequence ?? 1}`,
    sequence: overrides.sequence ?? 1,
    threadId: "thread-1",
    turnId: overrides.turnId ?? "turn-1",
    eventType: "item.delta",
    ...overrides,
  };
}

function emitTimeline(handler: MessageHandler, event: AgentTimelineEvent) {
  emitEnvelope(handler, {
    version: "v1",
    category: "event",
    name: "timeline.event",
    timestamp: "2026-04-25T10:00:00Z",
    payload: { event },
  });
}

test("streams turn deltas incrementally into one agent message", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });
  expect(connectConsoleSocketMock).toHaveBeenCalledWith("thread-1", expect.any(Function));
  expect(onMessage).not.toBeNull();

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:00Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 1,
      delta: "hello",
    },
  });

  await waitFor(() => {
    expect(result.current.isExecuting).toBe(true);
    expect(result.current.messages).toEqual([
      {
        id: "turn-1:1",
        kind: "agent",
        text: "hello",
        turnId: "turn-1",
      },
    ]);
  });

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:01Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 2,
      delta: " world",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "turn-1:1",
        kind: "agent",
        text: "hello world",
        turnId: "turn-1",
      },
    ]);
  });

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.completed",
    timestamp: "2026-04-25T10:00:02Z",
    payload: {
      turn: {
        threadId: "thread-1",
        turnId: "turn-1",
        status: "completed",
      },
    },
  });

  await waitFor(() => {
    expect(result.current.isExecuting).toBe(false);
    expect(result.current.messages).toHaveLength(1);
  });
});

test("removes transient analyzing progress when turn content arrives", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });
  expect(onMessage).not.toBeNull();

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:00Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 1,
      delta: "\n\n正在分析...",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "turn-1:1",
        kind: "agent",
        text: "\n\n正在分析...",
        turnId: "turn-1",
      },
    ]);
  });

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:01Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 2,
      delta: "hello",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "turn-1:1",
        kind: "agent",
        text: "hello",
        turnId: "turn-1",
      },
    ]);
  });
});

test("keeps progress deltas separate from final content", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });
  expect(onMessage).not.toBeNull();

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:00Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 1,
      delta: "我先查资料",
      kind: "progress",
    },
  });
  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:01Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 2,
      delta: "最终报告",
      kind: "content",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "turn-1:1",
        kind: "agent",
        text: "最终报告",
        progressText: "我先查资料",
        turnId: "turn-1",
      },
    ]);
  });
});

test("keeps pending execution visible when an older turn completion arrives", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });
  startThreadTurnMock.mockReturnValue(new Promise(() => {}));

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });

  act(() => {
    result.current.setPrompt("第二轮问题");
  });
  act(() => {
    void result.current.handlePromptSubmit();
  });

  await waitFor(() => {
    expect(result.current.isExecuting).toBe(true);
    expect(result.current.messages).toEqual([
      {
        id: expect.stringMatching(/^user:/),
        kind: "user",
        text: "第二轮问题",
      },
      {
        id: expect.stringMatching(/^pending-agent:/),
        kind: "agent",
        text: "",
        isPending: true,
      },
    ]);
  });

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.completed",
    timestamp: "2026-04-25T10:00:02Z",
    payload: {
      turn: {
        threadId: "thread-1",
        turnId: "previous-turn",
        status: "completed",
      },
    },
  });

  expect(result.current.isExecuting).toBe(true);
});

test("binds the pending agent placeholder to the streaming turn", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });
  startThreadTurnMock.mockReturnValue(new Promise(() => {}));

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });

  act(() => {
    result.current.setPrompt("第二轮问题");
  });
  act(() => {
    void result.current.handlePromptSubmit();
  });

  await waitFor(() => {
    expect(result.current.messages.at(-1)?.isPending).toBe(true);
  });

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:03Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-2",
      sequence: 1,
      delta: "开始处理",
      kind: "progress",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: expect.stringMatching(/^user:/),
        kind: "user",
        text: "第二轮问题",
      },
      {
        id: expect.stringMatching(/^pending-agent:/),
        kind: "agent",
        text: "",
        progressText: "开始处理",
        turnId: "turn-2",
      },
    ]);
  });
});

test("rebuilds workspace messages from timeline history when available", async () => {
  getThreadDetailMock.mockResolvedValue({
    ...THREAD_DETAIL_RESPONSE,
    messages: [
      {
        id: "legacy-message",
        kind: "agent",
        text: "legacy",
        turnId: "turn-legacy",
      },
    ],
    events: [
      timelineEvent({
        sequence: 1,
        turnId: "turn-history",
        itemId: "reasoning-1",
        itemType: "reasoning",
        phase: "analysis",
        content: { contentType: "markdown", delta: "先分析", appendMode: "append" },
      }),
      timelineEvent({
        sequence: 2,
        turnId: "turn-history",
        itemId: "message-1",
        itemType: "message",
        role: "assistant",
        phase: "final",
        content: { contentType: "markdown", delta: "最终回答", appendMode: "append" },
      }),
    ],
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "agent:turn-history",
        kind: "agent",
        text: "最终回答",
        progressText: "**分析**\n\n先分析",
        turnId: "turn-history",
      },
    ]);
  });
});

test("keeps real two-turn timeline output separated and shows the second pending state immediately", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });
  startThreadTurnMock
    .mockResolvedValueOnce({ turn: { threadId: "thread-1", turnId: "turn-hi" } })
    .mockReturnValueOnce(new Promise(() => {}));

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });

  act(() => {
    result.current.setPrompt("hi");
  });
  act(() => {
    void result.current.handlePromptSubmit();
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: expect.stringMatching(/^user:/),
        kind: "user",
        text: "hi",
      },
      {
        id: expect.stringMatching(/^pending-agent:/),
        kind: "agent",
        text: "",
        turnId: "turn-hi",
      },
    ]);
  });

  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 1,
      eventId: "turn-hi-delta",
      turnId: "turn-hi",
      itemId: "message-hi",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "hi", appendMode: "append" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 2,
      eventId: "turn-hi-completed",
      turnId: "turn-hi",
      eventType: "turn.completed",
      status: "completed",
    }),
  );

  await waitFor(() => {
    expect(result.current.isExecuting).toBe(false);
    expect(result.current.messages[1]).toMatchObject({
      kind: "agent",
      text: "hi",
      turnId: "turn-hi",
    });
  });

  act(() => {
    result.current.setPrompt("你调研一下，目前最新的Agent助理有哪些？做一份总结报告。");
  });
  act(() => {
    void result.current.handlePromptSubmit();
  });

  await waitFor(() => {
    expect(result.current.isExecuting).toBe(true);
    expect(result.current.messages.at(-1)).toEqual({
      id: expect.stringMatching(/^pending-agent:/),
      kind: "agent",
      text: "",
      isPending: true,
    });
  });

  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 1,
      eventId: "turn-research-started",
      turnId: "turn-research",
      eventType: "turn.started",
      status: "running",
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 2,
      eventId: "turn-research-progress",
      turnId: "turn-research",
      itemId: "reasoning-research",
      itemType: "reasoning",
      phase: "analysis",
      content: { contentType: "markdown", delta: "我先确认最新信息", appendMode: "append" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 3,
      eventId: "turn-research-final-1",
      turnId: "turn-research",
      itemId: "message-research",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "**总结报告**", appendMode: "append" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 4,
      eventId: "turn-research-final-2",
      turnId: "turn-research",
      itemId: "message-research",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "\n\n| Agent | 状态 |", appendMode: "append" },
    }),
  );

  await waitFor(() => {
    expect(result.current.messages).toHaveLength(4);
    expect(result.current.messages[1]).toMatchObject({
      kind: "agent",
      text: "hi",
      turnId: "turn-hi",
    });
    expect(result.current.messages[3]).toMatchObject({
      id: "agent:turn-research",
      kind: "agent",
      text: "**总结报告**\n\n| Agent | 状态 |",
      progressText: "**分析**\n\n我先确认最新信息",
      turnId: "turn-research",
    });
  });
});

test("ignores legacy turn events once timeline events for the same turn are present", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });

  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 1,
      turnId: "turn-1",
      itemId: "message-1",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "timeline", appendMode: "append" },
    }),
  );

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.delta",
    timestamp: "2026-04-25T10:00:01Z",
    payload: {
      threadId: "thread-1",
      turnId: "turn-1",
      sequence: 2,
      delta: " legacy",
      kind: "content",
    },
  });

  await waitFor(() => {
    expect(result.current.messages).toEqual([
      {
        id: "agent:turn-1",
        kind: "agent",
        text: "timeline",
        turnId: "turn-1",
      },
    ]);
  });
});

test("deduplicates a real streamed process snapshot in workspace messages", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result } = renderHook(() => useThreadWorkspace("thread-1"));

  await waitFor(() => {
    expect(result.current.title).toBe("Thread 1");
  });
  expect(onMessage).not.toBeNull();

  const progressSentence = "我会在当前目录创建名为 `测试B` 的文件，并申请写权限执行创建。";

  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 1,
      eventId: "turn-file-started",
      turnId: "turn-file",
      eventType: "turn.started",
      status: "running",
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 2,
      eventId: "progress-delta",
      turnId: "turn-file",
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", delta: progressSentence, appendMode: "append" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 3,
      eventId: "progress-completed",
      turnId: "turn-file",
      eventType: "item.completed",
      itemId: "progress-message",
      itemType: "message",
      role: "assistant",
      phase: "progress",
      content: { contentType: "markdown", text: progressSentence, appendMode: "snapshot" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 4,
      eventId: "approval-requested",
      turnId: "turn-file",
      eventType: "approval.requested",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      approval: { requestId: "approval-1", kind: "command", title: "/bin/sh -lc 'touch 测试B'" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 5,
      eventId: "approval-resolved",
      turnId: "turn-file",
      eventType: "approval.resolved",
      itemId: "command-1",
      itemType: "command",
      phase: "progress",
      approval: { requestId: "approval-1", kind: "command", decision: "accept" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 6,
      eventId: "final-message",
      turnId: "turn-file",
      itemId: "final-message",
      itemType: "message",
      role: "assistant",
      phase: "final",
      content: { contentType: "markdown", delta: "已创建文件 `测试B`。", appendMode: "append" },
    }),
  );
  emitTimeline(
    onMessage!,
    timelineEvent({
      sequence: 7,
      eventId: "turn-file-completed",
      turnId: "turn-file",
      eventType: "turn.completed",
      status: "completed",
    }),
  );

  await waitFor(() => {
    const progressText = result.current.messages[0]?.progressText ?? "";
    expect(result.current.messages).toHaveLength(1);
    expect(progressText.split(progressSentence)).toHaveLength(2);
    expect(progressText).toContain("**审批** /bin/sh -lc 'touch 测试B'");
    expect(progressText).toContain("**审批** 已处理：accept");
    expect(result.current.messages[0]).toMatchObject({
      id: "agent:turn-file",
      kind: "agent",
      text: "已创建文件 `测试B`。",
      turnId: "turn-file",
    });
  });
});
