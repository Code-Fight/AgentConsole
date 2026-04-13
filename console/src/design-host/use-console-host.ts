import { useCallback, useEffect, useMemo, useState } from "react";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  CreateThreadResponse,
  MachineListResponse,
  MachineSummary,
  ThreadDeleteResponse,
  ThreadDetailResponse,
  ThreadListResponse,
  ThreadSummary,
} from "../common/api/types";
import { useCapabilities } from "../gateway/capabilities";
import { formatThreadStatus } from "../gateway/thread-view-model";
import { useConsolePreferences } from "../gateway/use-console-preferences";
import { useThreadWorkspace } from "../gateway/use-thread-workspace";

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

export interface ConsoleSkillResource {
  id: string;
  name: string;
  machineId: string;
  machineName: string;
  agentId: string;
  agentName: string;
  description: string;
}

export interface ConsoleMCPResource {
  id: string;
  name: string;
  machineId: string;
  machineName: string;
  agentId: string;
  agentName: string;
  serverUrl: string;
}

export interface ConsolePluginResource {
  id: string;
  name: string;
  machineId: string;
  machineName: string;
  agentId: string;
  agentName: string;
  version: string;
}

export interface ConsoleHostViewModel {
  activePage: AppPage;
  machines: ConsoleMachine[];
  selectedSession: ConsoleSession | null;
  selectedMachine: ConsoleMachine | null;
  skills: ConsoleSkillResource[];
  mcps: ConsoleMCPResource[];
  plugins: ConsolePluginResource[];
  prompt: string;
  isSubmitting: boolean;
  mobilePanelOpen: boolean;
  sidebarCollapsed: boolean;
  onPromptChange: (value: string) => void;
  onSendPrompt: () => void;
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
  onAddSkill?: (machineId: string, agentId: string, name: string, description: string) => void;
  onAddMCP?: (machineId: string, agentId: string, name: string, serverUrl: string) => void;
  onAddPlugin?: (machineId: string, agentId: string, name: string, version: string) => void;
  onDeleteSkill?: (skillId: string) => void;
  onDeleteMCP?: (mcpId: string) => void;
  onDeletePlugin?: (pluginId: string) => void;
}

interface UseConsoleHostOptions {
  activePage: AppPage;
  threadId: string | null;
  navigate: (path: string) => void;
}

function formatTimestamp(): string {
  return new Date().toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function useConsoleHost({
  activePage,
  threadId,
  navigate,
}: UseConsoleHostOptions): ConsoleHostViewModel {
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [machines, setMachines] = useState<MachineSummary[]>([]);
  const [mobilePanelOpen, setMobilePanelOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [restoreAttempted, setRestoreAttempted] = useState(false);
  const [restoredThreadId, setRestoredThreadId] = useState<string | null>(null);
  const [lastVerifiedThreadId, setLastVerifiedThreadId] = useState<string | null>(null);

  useCapabilities();
  const {
    preferences,
    isLoading: preferencesLoading,
    error: preferencesError,
    updatePreferences,
  } = useConsolePreferences();

  const loadHubData = useCallback(async () => {
    const [threadResult, machineResult] = await Promise.allSettled([
      http<ThreadListResponse>("/threads"),
      http<MachineListResponse>("/machines"),
    ]);

    if (threadResult.status === "fulfilled") {
      setThreads(threadResult.value.items);
    }

    if (machineResult.status === "fulfilled") {
      setMachines(machineResult.value.items);
    }
  }, []);

  useEffect(() => {
    void loadHubData();
  }, [loadHubData]);

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

  const workspaceMessages = useMemo<ConsoleMessage[]>(() => {
    if (!threadId) {
      return [];
    }
    return workspace.messages
      .filter(
        (message): message is typeof message & { kind: "user" | "agent" } =>
          message.kind === "user" || message.kind === "agent",
      )
      .map((message) => ({
        id: message.id,
        role: message.kind,
        content: message.text,
        timestamp: formatTimestamp(),
      }));
  }, [threadId, workspace.messages]);

  const machineById = useMemo(() => {
    const map = new Map<string, MachineSummary>();
    machines.forEach((machine) => map.set(machine.id, machine));
    threads.forEach((thread) => {
      if (!map.has(thread.machineId)) {
        map.set(thread.machineId, {
          id: thread.machineId,
          name: thread.machineId,
          status: "unknown",
          runtimeStatus: "unknown",
        });
      }
    });
    return map;
  }, [machines, threads]);

  const consoleMachines = useMemo(() => {
    return Array.from(machineById.values()).map((machine) => {
      const machineThreads = threads.filter((thread) => thread.machineId === machine.id);
      const hasActiveThread = machineThreads.some((thread) => thread.status === "active");
      const agent: ConsoleAgentInfo = {
        id: `${machine.id}-codex`,
        name: "Codex",
        type: "codex",
        model: "codex",
        status: hasActiveThread ? "active" : "idle",
        port: 0,
      };
      const sessions = machineThreads.map((thread) => ({
        id: thread.threadId,
        title: thread.title || thread.threadId,
        agentName: agent.name,
        model: agent.model,
        status: thread.status,
        lastActivity: formatThreadStatus(thread.status),
        messages: thread.threadId === threadId ? workspaceMessages : [],
      }));

      return {
        id: machine.id,
        name: machine.name || machine.id,
        status: machine.status,
        runtimeStatus: machine.runtimeStatus ?? "unknown",
        agents: [agent],
        sessions,
      };
    });
  }, [machineById, threads, threadId, workspaceMessages]);

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
    async (machineId: string, _agentId: string, title: string) => {
      const nextTitle = title.trim();
      if (!machineId || nextTitle === "") {
        return;
      }

      try {
        const response = await http<CreateThreadResponse>("/threads", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ machineId, title: nextTitle }),
        });
        if (response.thread?.threadId) {
          navigate(`/threads/${response.thread.threadId}`);
        }
      } catch {
        return;
      }

      await loadHubData();
    },
    [loadHubData, navigate],
  );

  const handleDeleteSession = useCallback(
    async (sessionId: string) => {
      if (!sessionId) {
        return;
      }

      try {
        await http<ThreadDeleteResponse>(`/threads/${encodeURIComponent(sessionId)}`, {
          method: "DELETE",
        });
      } catch {
        return;
      }

      await loadHubData();
    },
    [loadHubData],
  );

  return {
    activePage,
    machines: consoleMachines,
    selectedSession,
    selectedMachine,
    skills: [],
    mcps: [],
    plugins: [],
    prompt: workspace.prompt,
    isSubmitting: workspace.isSubmitting,
    mobilePanelOpen,
    sidebarCollapsed,
    onPromptChange: (value) => workspace.setPrompt(value),
    onSendPrompt: workspace.handlePromptSubmit,
    onSelectSession: handleSelectSession,
    onNavigate: handleNavigate,
    onBackToThreads: handleBackToThreads,
    onToggleMobilePanel: () => setMobilePanelOpen((current) => !current),
    onCloseMobilePanel: () => setMobilePanelOpen(false),
    onToggleSidebar: () => setSidebarCollapsed((current) => !current),
    onExpandSidebar: () => setSidebarCollapsed(false),
    onDeleteSession: handleDeleteSession,
    onCreateThread: handleCreateThread,
  };
}
