import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { http } from "../../../common/api/http";
import type {
  EventEnvelope,
  MachineListResponse,
  MachineSummary,
  ThreadListResponse,
  ThreadSummary,
} from "../../../common/api/types";
import { connectConsoleSocket } from "../../../common/api/ws";
import { useCapabilities } from "../../../common/config/capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
  useGatewayConnectionState,
} from "../../../common/config/gateway-connection-store";

export interface MachinesPageAgentViewModel {
  id: string;
  name: string;
  type: "codex";
  model: string;
  status: "active" | "idle" | "offline";
  port: number;
}

export interface MachinesPageSessionViewModel {
  id: string;
  title: string;
  agentName: string;
  model: string;
  status: ThreadSummary["status"];
  lastActivity: string;
  messages: [];
}

export interface MachinesPageMachineViewModel {
  id: string;
  name: string;
  status: MachineSummary["status"];
  runtimeStatus: MachineSummary["runtimeStatus"];
  agents: MachinesPageAgentViewModel[];
  sessions: MachinesPageSessionViewModel[];
}

function formatThreadStatus(status: ThreadSummary["status"]): string {
  switch (status) {
    case "active":
      return "进行中";
    case "idle":
      return "空闲";
    case "systemError":
      return "异常";
    case "notLoaded":
      return "未加载";
    default:
      return "未知";
  }
}

function buildMachinesPageModel(
  threadSummaries: ThreadSummary[],
  machineSummaries: MachineSummary[],
): MachinesPageMachineViewModel[] {
  const machineById = new Map<string, MachineSummary>();

  machineSummaries.forEach((machine) => machineById.set(machine.id, machine));
  threadSummaries.forEach((thread) => {
    if (!machineById.has(thread.machineId)) {
      machineById.set(thread.machineId, {
        id: thread.machineId,
        name: thread.machineId,
        status: "unknown",
        runtimeStatus: "unknown",
        agents: [],
      });
    }
  });

  return Array.from(machineById.values()).map((machine) => {
    const machineThreads = threadSummaries.filter((thread) => thread.machineId === machine.id);
    const agents: MachinesPageAgentViewModel[] = (machine.agents ?? []).map((agent) => {
      const agentThreads = machineThreads.filter((thread) => thread.agentId === agent.agentId);
      const hasActiveThread = agentThreads.some((thread) => thread.status === "active");

      return {
        id: agent.agentId,
        name: agent.displayName,
        type: agent.agentType,
        model: agent.agentType,
        status:
          agent.status === "running"
            ? hasActiveThread
              ? "active"
              : "idle"
            : "offline",
        port: 0,
      };
    });

    const sessions: MachinesPageSessionViewModel[] = machineThreads.map((thread) => ({
      id: thread.threadId,
      title: thread.title || thread.threadId,
      agentName: agents.find((agent) => agent.id === thread.agentId)?.name ?? "Unknown agent",
      model: agents.find((agent) => agent.id === thread.agentId)?.model ?? "unknown",
      status: thread.status,
      lastActivity: formatThreadStatus(thread.status),
      messages: [],
    }));

    return {
      id: machine.id,
      name: machine.name || machine.id,
      status: machine.status,
      runtimeStatus: machine.runtimeStatus ?? "unknown",
      agents,
      sessions,
    };
  });
}

export function useMachinesPage() {
  const connection = useGatewayConnectionState();
  const remoteEnabled = connection.remoteEnabled;
  const connectionIdentity = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionIdentity,
    getGatewayConnectionIdentity,
  );
  const capabilities = useCapabilities(remoteEnabled);
  const [threadSummaries, setThreadSummaries] = useState<ThreadSummary[]>([]);
  const [machineSummaries, setMachineSummaries] = useState<MachineSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  const loadMachinesPageData = useCallback(async () => {
    try {
      const threadResponse = await http<ThreadListResponse>("/threads");
      setThreadSummaries(threadResponse.items);
      setError(null);
    } catch {
      setThreadSummaries([]);
      setMachineSummaries([]);
      setError("Unable to load live threads.");
      return;
    }

    try {
      const machineResponse = await http<MachineListResponse>("/machines");
      setMachineSummaries(machineResponse.items);
      setError(null);
    } catch {
      setMachineSummaries([]);
      setError("Unable to load machines.");
    }
  }, []);

  useEffect(() => {
    if (!remoteEnabled) {
      setThreadSummaries([]);
      setMachineSummaries([]);
      setError(null);
      return;
    }

    setThreadSummaries([]);
    setMachineSummaries([]);
    setError(null);
    void loadMachinesPageData();
  }, [connectionIdentity, loadMachinesPageData, remoteEnabled]);

  useEffect(() => {
    if (!remoteEnabled) {
      return undefined;
    }

    return connectConsoleSocket(undefined, (event) => {
      let envelope: EventEnvelope | null = null;

      try {
        envelope = JSON.parse(event.data) as EventEnvelope;
      } catch {
        return;
      }

      if (envelope.name !== "thread.updated" && envelope.name !== "machine.updated") {
        return;
      }

      void loadMachinesPageData();
    });
  }, [connectionIdentity, loadMachinesPageData, remoteEnabled]);

  const machines = useMemo(
    () => buildMachinesPageModel(threadSummaries, machineSummaries),
    [machineSummaries, threadSummaries],
  );

  const handleInstallAgent = useCallback(
    async (machineId: string, agentType: string, agentName: string) => {
      const nextAgentName = agentName.trim();
      if (
        !remoteEnabled ||
        !capabilities.machineInstallAgent ||
        !machineId ||
        !agentType ||
        nextAgentName === ""
      ) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          agentType,
          displayName: nextAgentName,
        }),
      });
      await loadMachinesPageData();
    },
    [capabilities.machineInstallAgent, loadMachinesPageData, remoteEnabled],
  );

  const handleDeleteAgent = useCallback(
    async (machineId: string, agentId: string) => {
      if (!remoteEnabled || !capabilities.machineRemoveAgent || !machineId || !agentId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}`, {
        method: "DELETE",
      });
      await loadMachinesPageData();
    },
    [capabilities.machineRemoveAgent, loadMachinesPageData, remoteEnabled],
  );

  const handleUpdateAgentConfig = useCallback(
    async (machineId: string, agentId: string, config: string) => {
      if (!remoteEnabled || !machineId || !agentId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}/config`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: config }),
      });
      await loadMachinesPageData();
    },
    [loadMachinesPageData, remoteEnabled],
  );

  const handleStartRuntime = useCallback(
    async (machineId: string) => {
      if (!remoteEnabled || !machineId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/runtime/start`, {
        method: "POST",
      });
      await loadMachinesPageData();
    },
    [loadMachinesPageData, remoteEnabled],
  );

  const handleStopRuntime = useCallback(
    async (machineId: string) => {
      if (!remoteEnabled || !machineId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/runtime/stop`, {
        method: "POST",
      });
      await loadMachinesPageData();
    },
    [loadMachinesPageData, remoteEnabled],
  );

  return {
    machines,
    error,
    onInstallAgent: handleInstallAgent,
    onDeleteAgent: handleDeleteAgent,
    onUpdateAgentConfig: handleUpdateAgentConfig,
    onStartRuntime: handleStartRuntime,
    onStopRuntime: handleStopRuntime,
  };
}
