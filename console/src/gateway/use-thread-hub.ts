import { useCallback, useEffect, useState } from "react";
import { http } from "../common/api/http";
import type {
  CreateThreadResponse,
  MachineListResponse,
  MachineSummary,
  ThreadDeleteResponse,
  ThreadListResponse,
  ThreadSummary,
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";
import { supportsCapability } from "./capabilities";
import { parseEnvelope, toThreadHubItem } from "./thread-view-model";

interface UseThreadHubOptions {
  enabled?: boolean;
}

export type ThreadHubViewModel = ReturnType<typeof useThreadHub>;

export function useThreadHub(options?: UseThreadHubOptions) {
  const enabled = options?.enabled ?? true;
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [machines, setMachines] = useState<Record<string, MachineSummary>>({});
  const [error, setError] = useState<string | null>(null);
  const [machineId, setMachineId] = useState("");
  const [title, setTitle] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const loadHubData = useCallback(async () => {
    try {
      const threadResponse = await http<ThreadListResponse>("/threads");
      setThreads(threadResponse.items);
      setError(null);
    } catch {
      setThreads([]);
      setMachines({});
      setError("Unable to load live threads.");
      return;
    }

    try {
      const machineResponse = await http<MachineListResponse>("/machines");
      setMachines(
        Object.fromEntries(machineResponse.items.map((machine) => [machine.id, machine])),
      );
      setError(null);
    } catch {
      setMachines({});
      setError("Unable to load machines.");
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
      setThreads([]);
      setMachines({});
      setError(null);
      return;
    }
    void loadHubData();
  }, [enabled, loadHubData]);

  useEffect(
    () => {
      if (!enabled) {
        return undefined;
      }

      return connectConsoleSocket(undefined, (event) => {
        const envelope = parseEnvelope(event.data);
        if (!envelope) {
          return;
        }

        if (envelope.name !== "thread.updated" && envelope.name !== "machine.updated") {
          return;
        }

        void loadHubData();
      });
    },
    [enabled, loadHubData],
  );

  const handleCreateThread = useCallback(
    async (machineOverride?: string, titleOverride?: string) => {
      const nextMachineId = (machineOverride ?? machineId).trim();
      const nextTitle = (titleOverride ?? title).trim();
      if (!enabled || !supportsCapability("threadHub") || nextMachineId === "" || nextTitle === "") {
        return null;
      }

      setIsSubmitting(true);
      setError(null);

      try {
        const response = await http<CreateThreadResponse>("/threads", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            machineId: nextMachineId,
            title: nextTitle,
          }),
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
        await http<void>(`/threads/${encodeURIComponent(threadId)}`, {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ title: nextTitle }),
        });
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
        await http<void>(`/threads/${encodeURIComponent(threadId)}/archive`, {
          method: "POST",
        });
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
        await http<void>(`/threads/${encodeURIComponent(threadId)}/resume`, {
          method: "POST",
        });
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
        await http<ThreadDeleteResponse>(`/threads/${encodeURIComponent(threadId)}`, {
          method: "DELETE",
        });
        await loadHubData();
      } catch {
        setError("Unable to delete thread.");
      }
    },
    [enabled, loadHubData],
  );

  return {
    error,
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
