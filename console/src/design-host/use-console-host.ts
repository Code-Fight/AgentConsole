import { useCallback, useEffect, useMemo, useState } from "react";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  CreateThreadResponse,
  MachineSummary,
  ThreadDetailResponse,
  ThreadSummary,
} from "../common/api/types";
import { useCapabilities } from "../gateway/capabilities";
import { formatThreadStatus } from "../gateway/thread-view-model";
import { useConsolePreferences } from "../gateway/use-console-preferences";
import { useThreadHub } from "../gateway/use-thread-hub";
import { useThreadWorkspace, type ThreadWorkspaceViewModel } from "../gateway/use-thread-workspace";

export type AppPage = "threads" | "machines" | "environment" | "settings";

export interface ConsoleFileChange {
  path: string;
  additions: number;
  deletions: number;
}

export interface ConsoleMessage {
  id: string;
  role: "user" | "agent";
  content: string;
  timestamp: string;
  fileChanges?: ConsoleFileChange[];
  terminalOutput?: string;
}

export type ConsoleThreadStatus = ThreadSummary["status"];

export interface ConsoleSession {
  id: string;
  title: string;
  agentName: string;
  model: string;
  status: ConsoleThreadStatus;
  lastActivity: string;
  messages: ConsoleMessage[];
}

export interface ConsoleAgentInfo {
  id: string;
  name: string;
  type: "claude-code" | "codex" | "custom";
  model: string;
  status: "active" | "idle" | "offline";
  port: number;
}

export type ConsoleMachineStatus = MachineSummary["status"];

export interface ConsoleMachine {
  id: string;
  name: string;
  status: ConsoleMachineStatus;
  runtimeStatus: MachineSummary["runtimeStatus"];
  agents: ConsoleAgentInfo[];
  sessions: ConsoleSession[];
}

export interface ConsoleHostViewModel {
  activePage: AppPage;
  machines: ConsoleMachine[];
  selectedSession: ConsoleSession | null;
  selectedMachine: ConsoleMachine | null;
  workspace: ThreadWorkspaceViewModel;
  mobilePanelOpen: boolean;
  sidebarCollapsed: boolean;
  onSelectSession: (machine: ConsoleMachine, session: ConsoleSession) => void;
  onNavigate: (page: AppPage) => void;
  onBackToThreads: () => void;
  onToggleMobilePanel: () => void;
  onCloseMobilePanel: () => void;
  onToggleSidebar: () => void;
  onExpandSidebar: () => void;
  onDeleteSession: (sessionId: string) => void;
  onCreateThread: (
    machineId: string,
    agentId: string,
    title: string,
    workDir: string,
  ) => void;
  onRenameSession?: (sessionId: string, newTitle: string) => void;
  onInstallAgent?: (machineId: string, agentType: string, agentName: string) => void;
  onDeleteAgent?: (machineId: string, agentId: string) => void;
  onUpdateAgentConfig?: (machineId: string, agentId: string, config: string) => void;
}

interface UseConsoleHostOptions {
  activePage: AppPage;
  threadId: string | null;
  navigate: (path: string) => void;
}

export function useConsoleHost({
  activePage,
  threadId,
  navigate,
}: UseConsoleHostOptions): ConsoleHostViewModel {
  const [mobilePanelOpen, setMobilePanelOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [restoreAttempted, setRestoreAttempted] = useState(false);
  const [restoredThreadId, setRestoredThreadId] = useState<string | null>(null);
  const [lastVerifiedThreadId, setLastVerifiedThreadId] = useState<string | null>(null);

  useCapabilities();
  const hub = useThreadHub({ enabled: activePage !== "settings" });
  const {
    preferences,
    isLoading: preferencesLoading,
    error: preferencesError,
    updatePreferences,
  } = useConsolePreferences();

  useEffect(() => {
    if (activePage !== "threads") {
      setMobilePanelOpen(false);
    }
  }, [activePage]);

  useEffect(() => {
    if (threadId || restoreAttempted || preferencesLoading) {
      return;
    }

    if (preferencesError) {
      setRestoreAttempted(true);
      return;
    }

    const lastThreadId = preferences?.lastThreadId?.trim();
    if (!lastThreadId) {
      setRestoreAttempted(true);
      return;
    }

    setRestoreAttempted(true);
    setRestoredThreadId(lastThreadId);
    navigate(`/threads/${lastThreadId}`);
  }, [
    threadId,
    restoreAttempted,
    preferencesLoading,
    preferencesError,
    preferences,
    navigate,
  ]);

  const workspace = useThreadWorkspace(threadId ?? "");

  useEffect(() => {
    if (!threadId || lastVerifiedThreadId === threadId || preferencesLoading || preferencesError) {
      return;
    }

    let active = true;
    const verifyThread = async () => {
      try {
        await http<ThreadDetailResponse>(buildThreadApiPath(threadId));
        if (!active) {
          return;
        }
        setLastVerifiedThreadId(threadId);
        if (preferences?.lastThreadId !== threadId) {
          await updatePreferences({ lastThreadId: threadId });
        }
      } catch {
        if (!active) {
          return;
        }
        if (restoredThreadId === threadId) {
          await updatePreferences({ lastThreadId: "" });
          navigate("/");
        }
      }
    };

    void verifyThread();
    return () => {
      active = false;
    };
  }, [
    threadId,
    lastVerifiedThreadId,
    preferencesLoading,
    preferencesError,
    preferences,
    updatePreferences,
    restoredThreadId,
    navigate,
  ]);

  const machineById = useMemo(() => {
    const map = new Map<string, MachineSummary>();
    hub.machineSummaries.forEach((machine) => map.set(machine.id, machine));
    hub.threadSummaries.forEach((thread) => {
      if (!map.has(thread.machineId)) {
        map.set(thread.machineId, {
          id: thread.machineId,
          name: thread.machineId,
          status: "unknown",
          runtimeStatus: "unknown",
          agents: [],
        });
      }
    });
    return map;
  }, [hub.machineSummaries, hub.threadSummaries]);

  const consoleMachines = useMemo(() => {
    return Array.from(machineById.values()).map((machine) => {
      const machineThreads = hub.threadSummaries.filter(
        (thread) => thread.machineId === machine.id,
      );
      const agents: ConsoleAgentInfo[] = (machine.agents ?? []).map((agent) => {
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
      const sessions = machineThreads.map((thread) => ({
        id: thread.threadId,
        title: thread.title || thread.threadId,
        agentName:
          agents.find((agent) => agent.id === thread.agentId)?.name ?? "Unknown agent",
        model:
          agents.find((agent) => agent.id === thread.agentId)?.model ?? "unknown",
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
  }, [hub.threadSummaries, machineById]);

  const selectedSession = useMemo(() => {
    if (!threadId) {
      return null;
    }
    for (const machine of consoleMachines) {
      const session = machine.sessions.find((candidate) => candidate.id === threadId);
      if (session) {
        return session;
      }
    }
    return null;
  }, [consoleMachines, threadId]);

  const selectedMachine = useMemo(() => {
    if (!threadId) {
      return null;
    }
    return (
      consoleMachines.find((machine) =>
        machine.sessions.some((session) => session.id === threadId),
      ) ?? null
    );
  }, [consoleMachines, threadId]);

  const handleSelectSession = useCallback(
    (_machine: ConsoleMachine, session: ConsoleSession) => {
      setMobilePanelOpen(false);
      navigate(`/threads/${session.id}`);
    },
    [navigate],
  );

  const handleNavigate = useCallback(
    (page: AppPage) => {
      setMobilePanelOpen(false);
      if (page === "threads") {
        navigate("/");
      } else {
        navigate(`/${page}`);
      }
    },
    [navigate],
  );

  const handleBackToThreads = useCallback(() => {
    setMobilePanelOpen(false);
    navigate("/");
  }, [navigate]);

  const handleCreateThread = useCallback(
    async (machineId: string, agentId: string, title: string) => {
      const nextTitle = title.trim();
      if (!machineId || !agentId || nextTitle === "") {
        return;
      }

      const created = await http<CreateThreadResponse>("/threads", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          machineId,
          agentId,
          title: nextTitle,
        }),
      });
      await hub.reload();
      if (created?.thread?.threadId) {
        navigate(`/threads/${created.thread.threadId}`);
      }
    },
    [hub, navigate],
  );

  const handleDeleteSession = useCallback(
    async (sessionId: string) => {
      if (!sessionId) {
        return;
      }

      await hub.handleDelete(sessionId);
    },
    [hub],
  );

  const handleRenameSession = useCallback(
    async (sessionId: string, newTitle: string) => {
      await hub.handleRename(sessionId, newTitle);
    },
    [hub],
  );

  const handleInstallAgent = useCallback(
    async (machineId: string, agentType: string, agentName: string) => {
      if (!machineId || !agentType || !agentName.trim()) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          agentType,
          displayName: agentName.trim(),
        }),
      });
      await hub.reload();
    },
    [hub],
  );

  const handleDeleteAgent = useCallback(
    async (machineId: string, agentId: string) => {
      if (!machineId || !agentId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}`, {
        method: "DELETE",
      });
      await hub.reload();
    },
    [hub],
  );

  const handleUpdateAgentConfig = useCallback(
    async (machineId: string, agentId: string, config: string) => {
      if (!machineId || !agentId) {
        return;
      }

      await http(`/machines/${encodeURIComponent(machineId)}/agents/${encodeURIComponent(agentId)}/config`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: config }),
      });
      await hub.reload();
    },
    [hub],
  );

  return {
    activePage,
    machines: consoleMachines,
    selectedSession,
    selectedMachine,
    workspace,
    mobilePanelOpen,
    sidebarCollapsed,
    onSelectSession: handleSelectSession,
    onNavigate: handleNavigate,
    onBackToThreads: handleBackToThreads,
    onToggleMobilePanel: () => setMobilePanelOpen((current) => !current),
    onCloseMobilePanel: () => setMobilePanelOpen(false),
    onToggleSidebar: () => setSidebarCollapsed((current) => !current),
    onExpandSidebar: () => setSidebarCollapsed(false),
    onDeleteSession: handleDeleteSession,
    onCreateThread: handleCreateThread,
    onRenameSession: handleRenameSession,
    onInstallAgent: handleInstallAgent,
    onDeleteAgent: handleDeleteAgent,
    onUpdateAgentConfig: handleUpdateAgentConfig,
  };
}
