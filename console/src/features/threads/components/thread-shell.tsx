import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { Bot, ChevronLeft, ChevronRight, Menu, MessageSquare, X } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";
import { useCapabilities } from "../../../common/config/capabilities";
import {
  getGatewayConnectionIdentity,
  getGatewayConnectionState,
  subscribeGatewayConnection,
} from "../../../common/config/gateway-connection-store";
import { useConsolePreferences } from "../../settings/hooks/use-console-preferences";
import { getThreadDetail } from "../api/thread-api";
import { useThreadHub } from "../hooks/use-thread-hub";
import { useThreadWorkspace } from "../hooks/use-thread-workspace";
import { ThreadHubProvider } from "../model/thread-hub-context";
import {
  buildThreadMachines,
  findThreadSelection,
  type ThreadMachineViewModel,
  type ThreadSessionViewModel,
  type ThreadShellDestination,
} from "../model/thread-view-model";
import SessionChat from "./session-chat";
import ThreadPanel from "./thread-panel";

interface ThreadShellProps {
  threadId?: string | null;
}

function resolveRoute(page: ThreadShellDestination) {
  if (page === "overview") {
    return "/overview";
  }
  if (page === "machines") {
    return "/machines";
  }
  if (page === "environment") {
    return "/environment";
  }
  return "/settings";
}

export function ThreadShell({ threadId = null }: ThreadShellProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const connectionState = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionState,
    getGatewayConnectionState,
  );
  const connectionIdentity = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionIdentity,
    getGatewayConnectionIdentity,
  );
  const remoteEnabled = connectionState === "ready";
  const [mobilePanelOpen, setMobilePanelOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [restoreAttempted, setRestoreAttempted] = useState(false);
  const [restoredThreadId, setRestoredThreadId] = useState<string | null>(null);
  const [lastVerifiedThreadId, setLastVerifiedThreadId] = useState<string | null>(null);
  const restoredThreadIdRef = useRef<string | null>(null);

  useCapabilities(remoteEnabled);
  const hub = useThreadHub({ enabled: remoteEnabled });
  const workspace = useThreadWorkspace(threadId ?? "", { enabled: remoteEnabled });
  const {
    preferences,
    isLoading: preferencesLoading,
    hasAttempted: preferencesAttempted,
    hasLoadedSuccessfully: preferencesLoadedSuccessfully,
    updatePreferences,
  } = useConsolePreferences({ enabled: remoteEnabled });
  const lastPreferenceThreadId = preferences?.lastThreadId ?? "";
  const restoredThreadIdFromNavigation = useMemo(() => {
    const state = location.state;
    if (!state || typeof state !== "object" || !("restoredThreadId" in state)) {
      return null;
    }

    const value = (state as Record<string, unknown>).restoredThreadId;
    return typeof value === "string" && value.trim() ? value.trim() : null;
  }, [location.state]);

  useEffect(() => {
    setRestoreAttempted(false);
    setRestoredThreadId(null);
    setLastVerifiedThreadId(null);
    restoredThreadIdRef.current = null;
  }, [connectionIdentity]);

  useEffect(() => {
    setMobilePanelOpen(false);
  }, [threadId]);

  useEffect(() => {
    if (
      !remoteEnabled ||
      threadId ||
      restoreAttempted ||
      preferencesLoading ||
      !preferencesAttempted
    ) {
      return;
    }

    if (!preferencesLoadedSuccessfully) {
      setRestoreAttempted(true);
      return;
    }

    const lastThreadId = lastPreferenceThreadId.trim();
    if (!lastThreadId) {
      setRestoreAttempted(true);
      return;
    }

    setRestoreAttempted(true);
    setRestoredThreadId(lastThreadId);
    restoredThreadIdRef.current = lastThreadId;
    navigate(`/threads/${lastThreadId}`, {
      state: { restoredThreadId: lastThreadId },
    });
  }, [
    navigate,
    preferencesAttempted,
    preferencesLoadedSuccessfully,
    preferencesLoading,
    lastPreferenceThreadId,
    remoteEnabled,
    restoreAttempted,
    threadId,
  ]);

  useEffect(() => {
    if (
      !remoteEnabled ||
      !threadId ||
      lastVerifiedThreadId === threadId ||
      preferencesLoading ||
      !preferencesAttempted ||
      !preferencesLoadedSuccessfully
    ) {
      return;
    }

    let active = true;

    const verifyThread = async () => {
      try {
        await getThreadDetail(threadId);
        if (!active) {
          return;
        }
        if (lastPreferenceThreadId === threadId) {
          restoredThreadIdRef.current = null;
          setLastVerifiedThreadId(threadId);
          return;
        }
        const updated = await updatePreferences({ lastThreadId: threadId });
        if (!active) {
          return;
        }
        if (updated) {
          restoredThreadIdRef.current = null;
          setLastVerifiedThreadId(threadId);
        }
      } catch {
        if (!active) {
          return;
        }
        if (
          restoredThreadIdFromNavigation === threadId ||
          (restoredThreadIdRef.current ?? restoredThreadId) === threadId
        ) {
          setLastVerifiedThreadId(threadId);
          restoredThreadIdRef.current = null;
          setRestoredThreadId(null);
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
    lastVerifiedThreadId,
    navigate,
    preferencesAttempted,
    preferencesLoadedSuccessfully,
    preferencesLoading,
    lastPreferenceThreadId,
    updatePreferences,
    remoteEnabled,
    restoredThreadId,
    restoredThreadIdFromNavigation,
    threadId,
  ]);

  const machines = useMemo(
    () => buildThreadMachines(hub.threadSummaries, hub.machineSummaries),
    [hub.machineSummaries, hub.threadSummaries],
  );
  const selection = useMemo(() => findThreadSelection(machines, threadId), [machines, threadId]);

  const handleSelectSession = useCallback(
    (_machine: ThreadMachineViewModel, session: ThreadSessionViewModel) => {
      setMobilePanelOpen(false);
      navigate(`/threads/${session.id}`);
    },
    [navigate],
  );

  const handleNavigate = useCallback(
    (page: ThreadShellDestination) => {
      setMobilePanelOpen(false);
      navigate(resolveRoute(page));
    },
    [navigate],
  );

  const handleCreateThread = useCallback(
    async (machineId: string, agentId: string, title: string) => {
      if (!remoteEnabled) {
        return;
      }

      const created = await hub.handleCreateThread(machineId, agentId, title);
      if (created?.threadId) {
        navigate(`/threads/${created.threadId}`);
      }
    },
    [hub, navigate, remoteEnabled],
  );

  const renderMainContent = () => {
    if (selection.selectedSession && selection.selectedMachine) {
      return (
        <SessionChat
          key={selection.selectedSession.id}
          session={selection.selectedSession}
          machine={selection.selectedMachine}
          workspace={workspace}
        />
      );
    }

    return (
      <div className="flex flex-col items-center justify-center h-full text-center px-6">
        <div className="size-16 rounded-2xl bg-zinc-800/80 flex items-center justify-center mb-4">
          <MessageSquare className="size-8 text-zinc-500" />
        </div>
        <h2 className="text-zinc-300 mb-2">选择一个线程</h2>
        <p className="text-sm text-zinc-600 max-w-sm">
          从左侧线程栏选择一个线程，开始与 Coding Agent 进行交互。
        </p>
      </div>
    );
  };

  return (
    <ThreadHubProvider value={hub}>
      <div className="size-full bg-zinc-950 text-zinc-100 flex flex-col overflow-hidden">
        <header className="lg:hidden flex items-center gap-3 px-4 py-3 bg-zinc-900 border-b border-zinc-800 flex-shrink-0">
          <button
            onClick={() => setMobilePanelOpen((current) => !current)}
            className="p-2 text-zinc-400 hover:text-zinc-50 rounded-lg hover:bg-zinc-800 transition-colors"
          >
            {mobilePanelOpen ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>

          <div className="flex-1">
            <div className="flex items-center gap-2">
              <div className="size-6 rounded-lg bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center">
                <Bot className="size-3.5 text-white" />
              </div>
              <span className="text-sm text-zinc-50 tracking-tight">Agent Console</span>
            </div>
            {selection.selectedSession ? (
              <p className="text-[10px] text-zinc-500 truncate mt-0.5">
                {selection.selectedMachine?.name} › {selection.selectedSession.title}
              </p>
            ) : null}
          </div>
        </header>

        {mobilePanelOpen ? (
          <>
            <div className="lg:hidden absolute top-[57px] left-0 bottom-0 w-[300px] bg-zinc-900 border-r border-zinc-800 z-50 shadow-2xl flex flex-col">
              <ThreadPanel
                machines={machines}
                selectedSessionId={selection.selectedSession?.id ?? null}
                onSelectSession={handleSelectSession}
                onNavigate={handleNavigate}
                onRenameSession={(sessionId, newTitle) => void hub.handleRename(sessionId, newTitle)}
                onDeleteSession={(sessionId) => void hub.handleDelete(sessionId)}
                onCreateThread={(machineId, agentId, title) =>
                  void handleCreateThread(machineId, agentId, title)
                }
              />
            </div>
            <div
              className="lg:hidden absolute inset-0 top-[57px] bg-black/50 z-40"
              onClick={() => setMobilePanelOpen(false)}
            />
          </>
        ) : null}

        <div className="hidden lg:flex flex-1 overflow-hidden">
          <div
            className="flex-shrink-0 bg-zinc-900 border-r border-zinc-800 transition-all duration-300"
            style={{
              width: sidebarCollapsed ? "0px" : "280px",
              overflow: "hidden",
            }}
          >
            <div className="w-[280px] h-full flex flex-col">
              <div className="flex items-center justify-between px-4 py-3.5 border-b border-zinc-800 flex-shrink-0">
                <div className="flex items-center gap-2">
                  <div className="size-6 rounded-lg bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center flex-shrink-0">
                    <Bot className="size-3.5 text-white" />
                  </div>
                  <span className="text-sm text-zinc-200 tracking-tight">Agent Console</span>
                </div>
                <button
                  onClick={() => setSidebarCollapsed(true)}
                  className="p-1 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
                >
                  <ChevronLeft className="size-4" />
                </button>
              </div>
              <div className="flex-1 overflow-hidden">
                <ThreadPanel
                  machines={machines}
                  selectedSessionId={selection.selectedSession?.id ?? null}
                  onSelectSession={handleSelectSession}
                  onNavigate={handleNavigate}
                  onRenameSession={(sessionId, newTitle) => void hub.handleRename(sessionId, newTitle)}
                  onDeleteSession={(sessionId) => void hub.handleDelete(sessionId)}
                  onCreateThread={(machineId, agentId, title) =>
                    void handleCreateThread(machineId, agentId, title)
                  }
                />
              </div>
            </div>
          </div>

          {sidebarCollapsed ? (
            <div className="flex-shrink-0 flex items-start pt-4 border-r border-zinc-800 bg-zinc-900">
              <button
                onClick={() => setSidebarCollapsed(false)}
                className="mx-1 p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
                title="展开线程面板"
              >
                <ChevronRight className="size-4" />
              </button>
            </div>
          ) : null}

          <main className="flex-1 overflow-hidden">{renderMainContent()}</main>
        </div>

        <main className="lg:hidden flex-1 overflow-hidden">{renderMainContent()}</main>
      </div>
    </ThreadHubProvider>
  );
}
