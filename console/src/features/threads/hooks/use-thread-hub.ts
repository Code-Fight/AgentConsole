import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import type { MachineSummary, ThreadSummary } from "../../../common/api/types";
import { connectConsoleSocket } from "../../../common/api/ws";
import { supportsCapability } from "../../../gateway/capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "../../../gateway/gateway-connection-store";
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
}

export type ThreadHubViewModel = ReturnType<typeof useThreadHub>;

export function useThreadHub(options?: UseThreadHubOptions) {
  const enabled = options?.enabled ?? true;
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

  const loadHubData = useCallback(async () => {
    try {
      const threadResponse = await listThreads();
      setThreads(threadResponse.items);
      setError(null);
    } catch {
      setThreads([]);
      setMachines({});
      setError("Unable to load live threads.");
      return;
    }

    try {
      const machineResponse = await listMachines();
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

      if (envelope.name !== "thread.updated" && envelope.name !== "machine.updated") {
        return;
      }

      void loadHubData();
    });
  }, [enabled, connectionIdentity, loadHubData]);

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
