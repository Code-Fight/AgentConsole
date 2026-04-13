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
      const [threadResponse, machineResponse] = await Promise.all([
        http<ThreadListResponse>("/threads"),
        http<MachineListResponse>("/machines"),
      ]);
      setThreads(threadResponse.items);
      setMachines(
        Object.fromEntries(machineResponse.items.map((machine) => [machine.id, machine])),
      );
      setError(null);
    } catch {
      setThreads([]);
      setMachines({});
      setError("Unable to load live threads.");
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
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

  const handleCreateThread = useCallback(async () => {
    const nextMachineId = machineId.trim();
    const nextTitle = title.trim();
    if (!supportsCapability("threadHub") || nextMachineId === "" || nextTitle === "") {
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await http<CreateThreadResponse>("/threads", {
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
    } catch {
      setError("Unable to create thread.");
    } finally {
      setIsSubmitting(false);
    }
  }, [loadHubData, machineId, title]);

  const handleArchive = useCallback(
    async (threadId: string) => {
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
    [loadHubData],
  );

  const handleResume = useCallback(
    async (threadId: string) => {
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
    [loadHubData],
  );

  const handleDelete = useCallback(
    async (threadId: string) => {
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
    [loadHubData],
  );

  return {
    error,
    threads: threads.map((thread) => toThreadHubItem(thread, machines)),
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
    handleArchive,
    handleResume,
    handleDelete,
  };
}
