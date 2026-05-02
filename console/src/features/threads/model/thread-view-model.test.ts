import { expect, test } from "vitest";
import type { MachineSummary, ThreadSummary } from "../../../common/api/types";
import { buildThreadMachines, formatThreadStatus, toThreadHubItem } from "./thread-view-model";

test("formats non-active non-error thread statuses as idle", () => {
  expect(formatThreadStatus("active")).toBe("进行中");
  expect(formatThreadStatus("systemError")).toBe("异常");
  expect(formatThreadStatus("idle")).toBe("空闲");
  expect(formatThreadStatus("notLoaded")).toBe("空闲");
  expect(formatThreadStatus("unknown")).toBe("空闲");
});

test("treats not loaded threads as idle in thread hub and session view models", () => {
  const thread: ThreadSummary = {
    threadId: "thread-not-loaded",
    machineId: "machine-01",
    agentId: "agent-01",
    status: "notLoaded",
    title: "Not loaded",
  };
  const machine: MachineSummary = {
    id: "machine-01",
    name: "Machine 01",
    status: "online",
    runtimeStatus: "running",
    agents: [
      {
        agentId: "agent-01",
        agentType: "codex",
        displayName: "Codex",
        status: "running",
      },
    ],
  };

  expect(toThreadHubItem(thread, { "machine-01": machine })).toMatchObject({
    status: "idle",
    statusLabel: "空闲",
  });
  expect(buildThreadMachines([thread], [machine])[0]?.sessions[0]).toMatchObject({
    status: "idle",
    lastActivity: "空闲",
  });
});

test("buildThreadMachines sorts machines by name and sessions by last activity descending", () => {
  const machines: MachineSummary[] = [
    {
      id: "machine-z",
      name: "Zulu",
      status: "online",
      runtimeStatus: "running",
      agents: [],
    },
    {
      id: "machine-a",
      name: "Alpha",
      status: "online",
      runtimeStatus: "running",
      agents: [],
    },
  ];

  const threads: ThreadSummary[] = [
    {
      threadId: "thread-older",
      machineId: "machine-a",
      status: "idle",
      title: "Older",
      lastActivityAt: "2026-04-20T10:00:00Z",
    },
    {
      threadId: "thread-latest",
      machineId: "machine-a",
      status: "active",
      title: "Latest",
      lastActivityAt: "2026-04-20T11:00:00Z",
    },
    {
      threadId: "thread-b",
      machineId: "machine-z",
      status: "idle",
      title: "No Activity B",
    },
    {
      threadId: "thread-a",
      machineId: "machine-z",
      status: "idle",
      title: "No Activity A",
    },
  ];

  const result = buildThreadMachines(threads, machines);

  expect(result.map((machine) => machine.id)).toEqual(["machine-a", "machine-z"]);
  expect(result[0]?.sessions.map((session) => session.id)).toEqual(["thread-latest", "thread-older"]);
  expect(result[1]?.sessions.map((session) => session.id)).toEqual(["thread-a", "thread-b"]);
  expect(result[0]?.sessions[0]?.lastActivity).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
  expect(result[0]?.sessions[0]?.lastActivity.includes("T")).toBe(false);
  expect(result[0]?.sessions[0]?.lastActivity.endsWith("Z")).toBe(false);
});
