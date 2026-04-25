import "@testing-library/jest-dom/vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import type { EventEnvelope } from "../../../common/api/types";
import {
  UNREAD_THREAD_STORAGE_KEY,
  useThreadHub,
} from "./use-thread-hub";

const connectConsoleSocketMock = vi.hoisted(() => vi.fn());

const archiveThreadMock = vi.hoisted(() => vi.fn());
const createThreadMock = vi.hoisted(() => vi.fn());
const deleteThreadMock = vi.hoisted(() => vi.fn());
const listMachinesMock = vi.hoisted(() => vi.fn());
const listThreadsMock = vi.hoisted(() => vi.fn());
const renameThreadMock = vi.hoisted(() => vi.fn());
const resumeThreadMock = vi.hoisted(() => vi.fn());

vi.mock("../../../common/api/ws", () => ({
  connectConsoleSocket: connectConsoleSocketMock,
}));

vi.mock("../api/thread-api", () => ({
  archiveThread: archiveThreadMock,
  createThread: createThreadMock,
  deleteThread: deleteThreadMock,
  listMachines: listMachinesMock,
  listThreads: listThreadsMock,
  renameThread: renameThreadMock,
  resumeThread: resumeThreadMock,
}));

type MessageHandler = (event: MessageEvent<string>) => void;

const THREADS_RESPONSE = {
  items: [
    {
      threadId: "thread-1",
      machineId: "machine-1",
      status: "idle",
      title: "Thread 1",
    },
    {
      threadId: "thread-2",
      machineId: "machine-1",
      status: "idle",
      title: "Thread 2",
    },
  ],
};

const MACHINES_RESPONSE = {
  items: [
    {
      id: "machine-1",
      name: "Machine 1",
      status: "online",
      runtimeStatus: "running",
      agents: [],
    },
  ],
};

beforeEach(() => {
  localStorage.clear();

  archiveThreadMock.mockReset();
  createThreadMock.mockReset();
  deleteThreadMock.mockReset();
  listMachinesMock.mockReset();
  listThreadsMock.mockReset();
  renameThreadMock.mockReset();
  resumeThreadMock.mockReset();
  connectConsoleSocketMock.mockReset();

  listThreadsMock.mockResolvedValue(THREADS_RESPONSE);
  listMachinesMock.mockResolvedValue(MACHINES_RESPONSE);
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

test("persists unread state in localStorage and clears it after selecting the thread", async () => {
  let onMessage: MessageHandler | null = null;
  connectConsoleSocketMock.mockImplementation((_, handler: MessageHandler) => {
    onMessage = handler;
    return () => {};
  });

  const { result, rerender } = renderHook(
    ({ selectedThreadId }: { selectedThreadId: string | null }) =>
      useThreadHub({ enabled: true, selectedThreadId }),
    {
      initialProps: { selectedThreadId: "thread-1" },
    },
  );

  await waitFor(() => {
    expect(result.current.threadSummaries).toHaveLength(2);
  });

  expect(onMessage).not.toBeNull();

  emitEnvelope(onMessage!, {
    version: "v1",
    category: "event",
    name: "turn.completed",
    timestamp: "2026-04-21T10:00:00Z",
    payload: {
      turn: {
        turnId: "turn-2",
        threadId: "thread-2",
        status: "completed",
      },
    },
  });

  await waitFor(() => {
    expect(result.current.unreadThreadIds.has("thread-2")).toBe(true);
  });

  const savedUnreadBeforeClear = JSON.parse(
    localStorage.getItem(UNREAD_THREAD_STORAGE_KEY) ?? "[]",
  ) as string[];
  expect(savedUnreadBeforeClear).toContain("thread-2");

  rerender({ selectedThreadId: "thread-2" });

  await waitFor(() => {
    expect(result.current.unreadThreadIds.has("thread-2")).toBe(false);
  });

  const savedUnreadAfterClear = JSON.parse(
    localStorage.getItem(UNREAD_THREAD_STORAGE_KEY) ?? "[]",
  ) as string[];
  expect(savedUnreadAfterClear).not.toContain("thread-2");
});

test("restores unread state from localStorage on initialization", async () => {
  localStorage.setItem(UNREAD_THREAD_STORAGE_KEY, JSON.stringify(["thread-2"]));

  connectConsoleSocketMock.mockImplementation(() => () => {});

  const { result } = renderHook(() => useThreadHub({ enabled: true, selectedThreadId: null }));

  await waitFor(() => {
    expect(result.current.threadSummaries).toHaveLength(2);
  });

  expect(result.current.unreadThreadIds.has("thread-2")).toBe(true);
});
