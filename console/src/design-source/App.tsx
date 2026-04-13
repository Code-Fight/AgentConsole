import { useState } from "react";
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
import { machines } from "./data/mockData";
import type {
  Machine,
  Session,
  SkillResource,
  MCPResource,
  PluginResource,
} from "./data/mockData";

type AppPage = "threads" | "machines" | "environment" | "settings";

export default function App() {
  const [activePage, setActivePage] = useState<AppPage>("threads");
  const [machinesData, setMachinesData] = useState<Machine[]>(machines);
  const [selectedMachine, setSelectedMachine] = useState<Machine | null>(machines[0]);
  const [selectedSession, setSelectedSession] = useState<Session | null>(machines[0].sessions[0]);
  const [mobilePanelOpen, setMobilePanelOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const [skills, setSkills] = useState<SkillResource[]>([]);
  const [mcps, setMCPs] = useState<MCPResource[]>([]);
  const [plugins, setPlugins] = useState<PluginResource[]>([]);

  const handleSelectSession = (machine: Machine, session: Session) => {
    setSelectedMachine(machine);
    setSelectedSession(session);
    setActivePage("threads");
    setMobilePanelOpen(false);
  };

  const handleNavigate = (page: "machines" | "environment" | "settings") => {
    setActivePage(page);
    setMobilePanelOpen(false);
  };

  const handleBackToThreads = () => {
    setActivePage("threads");
  };

  const handleRenameSession = (sessionId: string, newTitle: string) => {
    setMachinesData((prevMachines) =>
      prevMachines.map((machine) => ({
        ...machine,
        sessions: machine.sessions.map((session) =>
          session.id === sessionId ? { ...session, title: newTitle } : session,
        ),
      })),
    );

    if (selectedSession?.id === sessionId) {
      setSelectedSession((prev) => (prev ? { ...prev, title: newTitle } : null));
    }
  };

  const handleDeleteSession = (sessionId: string) => {
    setMachinesData((prevMachines) =>
      prevMachines.map((machine) => ({
        ...machine,
        sessions: machine.sessions.filter((session) => session.id !== sessionId),
      })),
    );

    if (selectedSession?.id === sessionId) {
      setSelectedSession(null);
    }
  };

  const handleInstallAgent = (machineId: string, agentType: string, agentName: string) => {
    setMachinesData((prevMachines) =>
      prevMachines.map((machine) => {
        if (machine.id === machineId) {
          const modelMap = {
            "claude-code": "claude-sonnet-4-5",
            codex: "claude-sonnet-4-5",
            custom: "gpt-4-turbo",
          };
          const basePort = 18000;
          const existingPorts = machine.agents.map((a) => a.port);
          let newPort = basePort;
          while (existingPorts.includes(newPort)) {
            newPort++;
          }

          const newAgent = {
            id: `agent-${Date.now()}`,
            name: agentName,
            type: agentType as "claude-code" | "codex" | "custom",
            model: modelMap[agentType as keyof typeof modelMap],
            status: "idle" as const,
            port: newPort,
          };
          return {
            ...machine,
            agents: [...machine.agents, newAgent],
          };
        }
        return machine;
      }),
    );
  };

  const handleUpdateAgentConfig = (machineId: string, agentId: string, config: string) => {
    console.log(`Updated config for agent ${agentId} on machine ${machineId}:`, config);
  };

  const handleDeleteAgent = (machineId: string, agentId: string) => {
    setMachinesData((prevMachines) =>
      prevMachines.map((machine) => {
        if (machine.id === machineId) {
          return {
            ...machine,
            agents: machine.agents.filter((agent) => agent.id !== agentId),
          };
        }
        return machine;
      }),
    );
  };

  const handleAddSkill = (
    machineId: string,
    agentId: string,
    name: string,
    description: string,
  ) => {
    const machine = machinesData.find((m) => m.id === machineId);
    const agent = machine?.agents.find((a) => a.id === agentId);
    if (!machine || !agent) return;

    const newSkill: SkillResource = {
      id: `skill-${Date.now()}`,
      name,
      machineId,
      machineName: machine.name,
      agentId,
      agentName: agent.name,
      description,
    };
    setSkills((prev) => [...prev, newSkill]);
  };

  const handleAddMCP = (
    machineId: string,
    agentId: string,
    name: string,
    serverUrl: string,
  ) => {
    const machine = machinesData.find((m) => m.id === machineId);
    const agent = machine?.agents.find((a) => a.id === agentId);
    if (!machine || !agent) return;

    const newMCP: MCPResource = {
      id: `mcp-${Date.now()}`,
      name,
      machineId,
      machineName: machine.name,
      agentId,
      agentName: agent.name,
      serverUrl,
    };
    setMCPs((prev) => [...prev, newMCP]);
  };

  const handleAddPlugin = (
    machineId: string,
    agentId: string,
    name: string,
    version: string,
  ) => {
    const machine = machinesData.find((m) => m.id === machineId);
    const agent = machine?.agents.find((a) => a.id === agentId);
    if (!machine || !agent) return;

    const newPlugin: PluginResource = {
      id: `plugin-${Date.now()}`,
      name,
      machineId,
      machineName: machine.name,
      agentId,
      agentName: agent.name,
      version,
    };
    setPlugins((prev) => [...prev, newPlugin]);
  };

  const handleDeleteSkill = (skillId: string) => {
    setSkills((prev) => prev.filter((s) => s.id !== skillId));
  };

  const handleDeleteMCP = (mcpId: string) => {
    setMCPs((prev) => prev.filter((m) => m.id !== mcpId));
  };

  const handleDeletePlugin = (pluginId: string) => {
    setPlugins((prev) => prev.filter((p) => p.id !== pluginId));
  };

  const handleCreateThread = (
    machineId: string,
    agentId: string,
    title: string,
    workDir: string,
  ) => {
    const machine = machinesData.find((m) => m.id === machineId);
    const agent = machine?.agents.find((a) => a.id === agentId);
    if (!machine || !agent) return;

    const newSession: Session = {
      id: `sess-${Date.now()}`,
      title,
      agentName: agent.name,
      model: agent.model,
      status: "idle",
      lastActivity: "刚刚",
      messages: [],
    };

    setMachinesData((prevMachines) =>
      prevMachines.map((m) => {
        if (m.id === machineId) {
          return {
            ...m,
            sessions: [...m.sessions, newSession],
          };
        }
        return m;
      }),
    );

    console.log("workDir", workDir);
    setSelectedMachine(machine);
    setSelectedSession(newSession);
    setActivePage("threads");
  };

  const renderMainContent = () => {
    switch (activePage) {
      case "threads":
        if (selectedSession && selectedMachine) {
          return (
            <SessionChat
              key={selectedSession.id}
              session={selectedSession}
              machine={selectedMachine}
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
            machines={machinesData}
            onInstallAgent={handleInstallAgent}
            onDeleteAgent={handleDeleteAgent}
            onUpdateAgentConfig={handleUpdateAgentConfig}
          />
        );
      case "environment":
        return (
          <Environment
            machines={machinesData}
            skills={skills}
            mcps={mcps}
            plugins={plugins}
            onAddSkill={handleAddSkill}
            onAddMCP={handleAddMCP}
            onAddPlugin={handleAddPlugin}
            onDeleteSkill={handleDeleteSkill}
            onDeleteMCP={handleDeleteMCP}
            onDeletePlugin={handleDeletePlugin}
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
            onClick={handleBackToThreads}
            className="p-2 text-zinc-400 hover:text-zinc-50 rounded-lg hover:bg-zinc-800 transition-colors"
          >
            <ArrowLeft className="size-5" />
          </button>
        ) : (
          <button
            onClick={() => setMobilePanelOpen(!mobilePanelOpen)}
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
              machines={machinesData}
              selectedSessionId={selectedSession?.id ?? null}
              onSelectSession={handleSelectSession}
              onNavigate={handleNavigate}
              onRenameSession={handleRenameSession}
              onDeleteSession={handleDeleteSession}
              onCreateThread={handleCreateThread}
            />
          </div>
          <div
            className="lg:hidden absolute inset-0 top-[57px] bg-black/50 z-40"
            onClick={() => setMobilePanelOpen(false)}
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
                  onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
                  className="p-1 text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
                >
                  <ChevronLeft className="size-4" />
                </button>
              </div>
              <div className="flex-1 overflow-hidden">
                <MachinePanel
                  machines={machinesData}
                  selectedSessionId={selectedSession?.id ?? null}
                  onSelectSession={handleSelectSession}
                  onNavigate={handleNavigate}
                  onRenameSession={handleRenameSession}
                  onDeleteSession={handleDeleteSession}
                  onCreateThread={handleCreateThread}
                />
              </div>
            </div>
          </div>
        ) : null}

        {showThreadPanel && sidebarCollapsed ? (
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

        {isManagementPage ? (
          <div className="flex-shrink-0 flex flex-col bg-zinc-900 border-r border-zinc-800 w-16">
            <div className="flex items-center justify-center py-4 border-b border-zinc-800">
              <button
                onClick={handleBackToThreads}
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
