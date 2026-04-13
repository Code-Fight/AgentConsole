import {
  Menu,
  X,
  Bot,
  MessageSquare,
  ChevronLeft,
  ChevronRight,
  ArrowLeft,
} from "lucide-react";
import Settings from "./components/Settings";
import Machines from "./components/Machines";
import Environment from "./components/Environment";
import SessionChat from "./components/SessionChat";
import MachinePanel from "./components/MachinePanel";
import type { ConsoleHostViewModel } from "../design-host/use-console-host";
import type { Machine as ManagementMachine } from "./data/mockData";

type AppProps = ConsoleHostViewModel;

export default function App({
  activePage,
  machines,
  selectedSession,
  selectedMachine,
  skills,
  mcps,
  plugins,
  prompt,
  isSubmitting,
  mobilePanelOpen,
  sidebarCollapsed,
  onPromptChange,
  onSendPrompt,
  onSelectSession,
  onNavigate,
  onBackToThreads,
  onToggleMobilePanel,
  onCloseMobilePanel,
  onToggleSidebar,
  onExpandSidebar,
  onRenameSession,
  onDeleteSession,
  onCreateThread,
  onInstallAgent,
  onDeleteAgent,
  onUpdateAgentConfig,
  onAddSkill,
  onAddMCP,
  onAddPlugin,
  onDeleteSkill,
  onDeleteMCP,
  onDeletePlugin,
}: AppProps) {
  const managementMachines: ManagementMachine[] = machines.map((machine) => {
    const statusLabel =
      machine.status === "reconnecting"
        ? "重连中"
        : machine.status === "unknown"
          ? "未知"
          : "";
    const name = statusLabel
      ? `${machine.name || machine.id} (${statusLabel})`
      : machine.name || machine.id;
    const status: ManagementMachine["status"] =
      machine.status === "online" || machine.status === "offline" ? machine.status : "offline";

    return {
      ...machine,
      name,
      status,
      host: "未提供",
      os: "未提供",
    };
  });

  const renderMainContent = () => {
    switch (activePage) {
      case "threads":
        if (selectedSession && selectedMachine) {
          return (
            <SessionChat
              key={selectedSession.id}
              session={selectedSession}
              machine={selectedMachine}
              prompt={prompt}
              isSubmitting={isSubmitting}
              onPromptChange={onPromptChange}
              onSendPrompt={onSendPrompt}
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
      case "machines":
        return (
          <Machines
            machines={managementMachines}
            onInstallAgent={onInstallAgent}
            onDeleteAgent={onDeleteAgent}
            onUpdateAgentConfig={onUpdateAgentConfig}
          />
        );
      case "environment":
        return (
          <Environment
            machines={managementMachines}
            skills={skills}
            mcps={mcps}
            plugins={plugins}
            onAddSkill={onAddSkill}
            onAddMCP={onAddMCP}
            onAddPlugin={onAddPlugin}
            onDeleteSkill={onDeleteSkill}
            onDeleteMCP={onDeleteMCP}
            onDeletePlugin={onDeletePlugin}
          />
        );
      case "settings":
        return <Settings />;
      default:
        return null;
    }
  };

  const showThreadPanel = activePage === "threads";
  const isManagementPage = activePage !== "threads";

  return (
    <div className="size-full bg-zinc-950 text-zinc-100 flex flex-col overflow-hidden">
      <header className="lg:hidden flex items-center gap-3 px-4 py-3 bg-zinc-900 border-b border-zinc-800 flex-shrink-0">
        {isManagementPage ? (
          <button
            onClick={onBackToThreads}
            className="p-2 text-zinc-400 hover:text-zinc-50 rounded-lg hover:bg-zinc-800 transition-colors"
          >
            <ArrowLeft className="size-5" />
          </button>
        ) : (
          <button
            onClick={onToggleMobilePanel}
            className="p-2 text-zinc-400 hover:text-zinc-50 rounded-lg hover:bg-zinc-800 transition-colors"
          >
            {mobilePanelOpen ? <X className="size-5" /> : <Menu className="size-5" />}
          </button>
        )}

        <div className="flex-1">
          <div className="flex items-center gap-2">
            <div className="size-6 rounded-lg bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center">
              <Bot className="size-3.5 text-white" />
            </div>
            <span className="text-sm text-zinc-50 tracking-tight">
              {isManagementPage
                ? activePage === "machines"
                  ? "机器管理"
                  : activePage === "environment"
                    ? "环境资源"
                    : "设置"
                : "Agent Console"}
            </span>
          </div>
          {activePage === "threads" && selectedSession ? (
            <p className="text-[10px] text-zinc-500 truncate mt-0.5">
              {selectedMachine?.name} › {selectedSession.title}
            </p>
          ) : null}
        </div>
      </header>

      {mobilePanelOpen && activePage === "threads" ? (
        <>
          <div className="lg:hidden absolute top-[57px] left-0 bottom-0 w-[300px] bg-zinc-900 border-r border-zinc-800 z-50 shadow-2xl flex flex-col">
            <MachinePanel
              machines={machines}
              selectedSessionId={selectedSession?.id ?? null}
              onSelectSession={onSelectSession}
              onNavigate={onNavigate}
              onRenameSession={onRenameSession}
              onDeleteSession={onDeleteSession}
              onCreateThread={onCreateThread}
            />
          </div>
          <div
            className="lg:hidden absolute inset-0 top-[57px] bg-black/50 z-40"
            onClick={onCloseMobilePanel}
          />
        </>
      ) : null}

      <div className="hidden lg:flex flex-1 overflow-hidden">
        {showThreadPanel ? (
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
                  onClick={onToggleSidebar}
                  className="p-1 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
                >
                  <ChevronLeft className="size-4" />
                </button>
              </div>
              <div className="flex-1 overflow-hidden">
                <MachinePanel
                  machines={machines}
                  selectedSessionId={selectedSession?.id ?? null}
                  onSelectSession={onSelectSession}
                  onNavigate={onNavigate}
                  onRenameSession={onRenameSession}
                  onDeleteSession={onDeleteSession}
                  onCreateThread={onCreateThread}
                />
              </div>
            </div>
          </div>
        ) : null}

        {showThreadPanel && sidebarCollapsed ? (
          <div className="flex-shrink-0 flex items-start pt-4 border-r border-zinc-800 bg-zinc-900">
            <button
              onClick={onExpandSidebar}
              className="mx-1 p-1.5 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
              title="展开线程面板"
            >
              <ChevronRight className="size-4" />
            </button>
          </div>
        ) : null}

        {isManagementPage ? (
          <div className="flex-shrink-0 flex flex-col bg-zinc-900 border-r border-zinc-800 w-16">
            <div className="flex items-center justify-center py-4 border-b border-zinc-800">
              <button
                onClick={onBackToThreads}
                className="size-10 rounded-xl bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center transition-colors"
                title="返回线程"
              >
                <ArrowLeft className="size-5 text-zinc-400" />
              </button>
            </div>
          </div>
        ) : null}

        <main className="flex-1 overflow-hidden">{renderMainContent()}</main>
      </div>

      <main className="lg:hidden flex-1 overflow-hidden">{renderMainContent()}</main>
    </div>
  );
}
