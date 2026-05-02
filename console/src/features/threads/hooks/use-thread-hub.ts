import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import type {
  EventEnvelope,
  MachineSummary,
  TimelineEventPayload,
  ThreadSummary,
  TurnCompletedPayload,
} from "../../../common/api/types";
import { supportsCapability } from "../../../common/config/capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "../../../common/config/gateway-connection-store";
import { connectConsoleSocket } from "../../../common/api/ws";
import {
  archiveThread,
  createThread,
  deleteThread,
  listMachines,
  listThreads,
  renameThread,
  resumeThread,
} from "../api/thread-api";
import { parseEnvelope, toThreadHubItem } from "../model/thread-view-model";

interface UseThreadHubOptions {
  enabled?: boolean;
  selectedThreadId?: string | null;
}

export type ThreadHubViewModel = ReturnType<typeof useThreadHub>;
export const UNREAD_THREAD_STORAGE_KEY = "thread-hub.unread-thread-ids";

function readUnreadThreadIds(): Set<string> {
  if (typeof window === "undefined") {
    return new Set();
  }

  try {
    const raw = window.localStorage.getItem(UNREAD_THREAD_STORAGE_KEY);
    if (!raw) {
      return new Set();
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return new Set();
    }
    return new Set(
      parsed
        .map((value) => (typeof value === "string" ? value.trim() : ""))
        .filter((value) => value.length > 0),
    );
  } catch {
    return new Set();
  }
}

function persistUnreadThreadIds(unreadThreadIds: Set<string>) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(
      UNREAD_THREAD_STORAGE_KEY,
      JSON.stringify(Array.from(unreadThreadIds)),
    );
  } catch {
    // localStorage may be unavailable in strict privacy mode.
  }
}

function resolveCompletedThreadId(envelope: EventEnvelope): string | null {
  if (envelope.name === "timeline.event") {
    const event = (envelope.payload as TimelineEventPayload).event;
    if (event?.eventType === "turn.completed" || event?.eventType === "turn.failed") {
      return event.threadId ?? null;
    }
    return null;
  }

  if (envelope.name !== "turn.completed" && envelope.name !== "turn.failed") {
    return null;
  }

  const payload = envelope.payload as TurnCompletedPayload;
  return payload.turn?.threadId ?? null;
}

function isThreadHubRefreshEvent(envelope: EventEnvelope): boolean {
  if (
    envelope.name === "thread.updated" ||
    envelope.name === "machine.updated" ||
    envelope.name === "turn.started" ||
    envelope.name === "turn.completed" ||
    envelope.name === "turn.failed"
  ) {
    return true;
  }

  if (envelope.name !== "timeline.event") {
    return false;
  }

  const event = (envelope.payload as TimelineEventPayload).event;
  return (
    event?.eventType === "turn.started" ||
    event?.eventType === "turn.completed" ||
    event?.eventType === "turn.failed"
  );
}

export function useThreadHub(options?: UseThreadHubOptions) {
  const enabled = options?.enabled ?? true;
  const selectedThreadId = options?.selectedThreadId ?? null;
  const connectionIdentity = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionIdentity,
    getGatewayConnectionIdentity,
  );
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [machines, setMachines] = useState<Record<string, MachineSummary>>({});
  const [error, setError] = useState<string | null>(null);
  const [machineId, setMachineId] = useState("");
  const [title, setTitle] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [unreadThreadIds, setUnreadThreadIds] = useState<Set<string>>(readUnreadThreadIds);

  const loadHubData = useCallback(async () => {
    const [threadResult, machineResult] = await Promise.allSettled([listThreads(), listMachines()]);

    if (threadResult.status === "fulfilled") {
      setThreads(threadResult.value.items);
    } else {
      setThreads([]);
      setMachines({});
      setError("Unable to load live threads.");
      return;
    }

    if (machineResult.status === "fulfilled") {
      setMachines(
        Object.fromEntries(machineResult.value.items.map((machine) => [machine.id, machine])),
      );
      setError(null);
    } else {
      setMachines({});
      setError("Unable to load live machines.");
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
      setThreads([]);
      setMachines({});
      setError(null);
      return;
    }

    setThreads([]);
    setMachines({});
    setError(null);
    void loadHubData();
  }, [enabled, connectionIdentity, loadHubData]);

  useEffect(() => {
    if (!enabled) {
      return undefined;
    }

    return connectConsoleSocket(undefined, (event) => {
      const envelope = parseEnvelope(event.data);
      if (!envelope) {
        return;
      }

      const completedThreadId = resolveCompletedThreadId(envelope);
      if (completedThreadId) {
        const selectedId = selectedThreadId?.trim() ?? "";
        if (completedThreadId !== selectedId) {
          setUnreadThreadIds((current) => {
            if (current.has(completedThreadId)) {
              return current;
            }
            const next = new Set(current);
            next.add(completedThreadId);
            persistUnreadThreadIds(next);
            return next;
          });
        }
      }

      if (!isThreadHubRefreshEvent(envelope)) {
        return;
      }

      void loadHubData();
    });
  }, [enabled, connectionIdentity, loadHubData, selectedThreadId]);

  useEffect(() => {
    const currentThreadId = selectedThreadId?.trim();
    if (!currentThreadId) {
      return;
    }

    setUnreadThreadIds((current) => {
      if (!current.has(currentThreadId)) {
        return current;
      }
      const next = new Set(current);
      next.delete(currentThreadId);
      persistUnreadThreadIds(next);
      return next;
    });
  }, [selectedThreadId]);

  const handleCreateThread = useCallback(
    async (machineOverride?: string, agentOverride?: string, titleOverride?: string) => {
      const nextMachineId = (machineOverride ?? machineId).trim();
      const nextAgentId = (agentOverride ?? "").trim();
      const nextTitle = (titleOverride ?? title).trim();

      if (!enabled || !supportsCapability("threadHub") || nextMachineId === "" || nextTitle === "") {
        return null;
      }

      setIsSubmitting(true);
      setError(null);

      try {
        const response = await createThread({
          machineId: nextMachineId,
          ...(nextAgentId ? { agentId: nextAgentId } : {}),
          title: nextTitle,
        });
        setTitle("");
        await loadHubData();
        return response.thread ?? null;
      } catch {
        setError("Unable to create thread.");
        return null;
      } finally {
        setIsSubmitting(false);
      }
    },
    [enabled, loadHubData, machineId, title],
  );

  const handleRename = useCallback(
    async (threadId: string, newTitle: string) => {
      const nextTitle = newTitle.trim();
      if (!enabled || !threadId || nextTitle === "") {
        return;
      }

      setError(null);

      try {
        await renameThread(threadId, nextTitle);
        await loadHubData();
      } catch {
        setError("Unable to rename thread.");
      }
    },
    [enabled, loadHubData],
  );

  const handleArchive = useCallback(
    async (threadId: string) => {
      if (!enabled) {
        return;
      }

      setError(null);

      try {
        await archiveThread(threadId);
        await loadHubData();
      } catch {
        setError("Unable to archive thread.");
      }
    },
    [enabled, loadHubData],
  );

  const handleResume = useCallback(
    async (threadId: string) => {
      if (!enabled) {
        return;
      }

      setError(null);

      try {
        await resumeThread(threadId);
        await loadHubData();
      } catch {
        setError("Unable to resume thread.");
      }
    },
    [enabled, loadHubData],
  );

  const handleDelete = useCallback(
    async (threadId: string) => {
      if (!enabled) {
        return;
      }

      setError(null);

      try {
        await deleteThread(threadId);
        await loadHubData();
      } catch {
        setError("Unable to delete thread.");
      }
    },
    [enabled, loadHubData],
  );

  return {
    error,
    unreadThreadIds,
    threadSummaries: threads,
    threads: threads.map((thread) => toThreadHubItem(thread, machines)),
    machineSummaries: Object.values(machines),
    machineSuggestions: Object.values(machines).map((machine) => ({
      id: machine.id,
      label: machine.name || machine.id,
    })),
    machineCount: Object.keys(machines).length,
    machineId,
    title,
    isSubmitting,
    setMachineId,
    setTitle,
    handleCreateThread,
    handleRename,
    handleArchive,
    handleResume,
    handleDelete,
    reload: enabled ? loadHubData : async () => {},
  };
}
