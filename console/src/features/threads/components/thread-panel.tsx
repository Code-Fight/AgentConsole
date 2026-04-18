import { useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import {
  Activity,
  Bot,
  Check,
  ChevronDown,
  ChevronRight,
  FolderOpen,
  MessageSquare,
  Package,
  Plus,
  Server,
  Settings,
  Wifi,
  WifiOff,
  X,
} from "lucide-react";
import ThreadItem from "./thread-item";
import type {
  ThreadMachineViewModel as Machine,
  ThreadSessionViewModel as Session,
  ThreadShellDestination,
} from "../model/thread-view-model";

interface ThreadPanelProps {
  machines: Machine[];
  selectedSessionId: string | null;
  onSelectSession: (machine: Machine, session: Session) => void;
  onNavigate?: (page: ThreadShellDestination) => void;
  onRenameSession?: (sessionId: string, newTitle: string) => void;
  onDeleteSession?: (sessionId: string) => void;
  onCreateThread?: (
    machineId: string,
    agentId: string,
    title: string,
    workDir: string,
  ) => void;
}

const statusConfig = {
  online: {
    dot: "bg-emerald-400",
    text: "text-emerald-400",
    label: "在线",
    icon: Wifi,
  },
  reconnecting: {
    dot: "bg-amber-400",
    text: "text-amber-400",
    label: "重连中",
    icon: Wifi,
  },
  unknown: {
    dot: "bg-zinc-500",
    text: "text-zinc-500",
    label: "未知",
    icon: Server,
  },
  offline: {
    dot: "bg-zinc-600",
    text: "text-zinc-500",
    label: "离线",
    icon: WifiOff,
  },
};

function MachineGroup({
  machine,
  selectedSessionId,
  onSelectSession,
  onRenameSession,
  onDeleteSession,
}: {
  machine: Machine;
  selectedSessionId: string | null;
  onSelectSession: (machine: Machine, session: Session) => void;
  onRenameSession?: (sessionId: string, newTitle: string) => void;
  onDeleteSession?: (sessionId: string) => void;
}) {
  const [expanded, setExpanded] = useState(machine.status !== "offline");
  const cfg = statusConfig[machine.status];

  return (
    <div className="mb-1">
      <button
        onClick={() => setExpanded(!expanded)}
        disabled={machine.status === "offline"}
        className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg group transition-colors ${
          machine.status === "offline"
            ? "opacity-50 cursor-not-allowed"
            : "hover:bg-zinc-800/50"
        }`}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {expanded ? (
            <ChevronDown className="size-3.5 text-zinc-500 flex-shrink-0" />
          ) : (
            <ChevronRight className="size-3.5 text-zinc-500 flex-shrink-0" />
          )}
          <div className={`size-2 rounded-full flex-shrink-0 ${cfg.dot}`} />
          <div className="min-w-0 text-left">
            <div className="text-xs text-zinc-300 truncate leading-tight">{machine.name}</div>
            <div className="text-[10px] text-zinc-600 truncate leading-tight font-mono">
              ID: {machine.id}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1.5 flex-shrink-0">
          {machine.sessions.length > 0 ? (
            <span className="text-[10px] text-zinc-500 bg-zinc-800 px-1.5 py-0.5 rounded">
              {machine.sessions.filter((session) => session.status === "active").length > 0
                ? `${machine.sessions.filter((session) => session.status === "active").length} 活跃`
                : `${machine.sessions.length}`}
            </span>
          ) : null}
          {machine.agents.length > 0 ? (
            <div className="flex items-center gap-0.5">
              <Bot className="size-3 text-zinc-600" />
              <span className="text-[10px] text-zinc-600">{machine.agents.length}</span>
            </div>
          ) : null}
        </div>
      </button>

      {expanded && machine.sessions.length > 0 ? (
        <div className="ml-2 pl-3 border-l border-zinc-800/70 mt-0.5 space-y-0.5">
          {machine.sessions.map((session) => (
            <ThreadItem
              key={session.id}
              session={session}
              machine={machine}
              isSelected={selectedSessionId === session.id}
              onSelect={onSelectSession}
              onRename={onRenameSession}
              onDelete={onDeleteSession}
            />
          ))}
        </div>
      ) : null}

      {expanded && machine.sessions.length === 0 && machine.status !== "offline" ? (
        <div className="ml-2 pl-3 border-l border-zinc-800/70 mt-0.5 px-3 py-2">
          <span className="text-[10px] text-zinc-600">无活跃会话</span>
        </div>
      ) : null}
    </div>
  );
}

function NewThreadDialog({
  machines,
  open,
  onOpenChange,
  onCreate,
}: {
  machines: Machine[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (machineId: string, agentId: string, title: string, workDir: string) => void;
}) {
  const [selectedMachineId, setSelectedMachineId] = useState("");
  const [selectedAgentId, setSelectedAgentId] = useState("");
  const [title, setTitle] = useState("");
  const [workDir, setWorkDir] = useState("");

  const onlineMachines = machines.filter((machine) => machine.status === "online" && machine.agents.length > 0);
  const selectedMachine = onlineMachines.find((machine) => machine.id === selectedMachineId);

  const handleMachineChange = (machineId: string) => {
    setSelectedMachineId(machineId);
    const machine = onlineMachines.find((candidate) => candidate.id === machineId);
    if (machine && machine.agents.length > 0) {
      setSelectedAgentId(machine.agents[0].id);
    } else {
      setSelectedAgentId("");
    }
  };

  const handleCreate = () => {
    if (!selectedMachineId || !selectedAgentId || !title.trim()) {
      return;
    }

    onCreate(selectedMachineId, selectedAgentId, title.trim(), workDir.trim() || "/workspace");
    onOpenChange(false);
    setSelectedMachineId("");
    setSelectedAgentId("");
    setTitle("");
    setWorkDir("");
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
          <Dialog.Title className="text-lg text-zinc-100 mb-1">新建线程</Dialog.Title>
          <Dialog.Description className="text-xs text-zinc-500 mb-6">
            选择机器和 Agent，开始新的工作会话
          </Dialog.Description>

          <div className="space-y-4">
            <div>
              <label className="block text-xs text-zinc-400 mb-2">选择机器</label>
              <select
                value={selectedMachineId}
                onChange={(event) => handleMachineChange(event.target.value)}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
              >
                <option value="">选择机器</option>
                {onlineMachines.map((machine) => (
                  <option key={machine.id} value={machine.id}>
                    {machine.name}
                  </option>
                ))}
              </select>
            </div>

            {selectedMachine ? (
              <div>
                <label className="block text-xs text-zinc-400 mb-2">选择 Agent</label>
                <select
                  value={selectedAgentId}
                  onChange={(event) => setSelectedAgentId(event.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
                >
                  {selectedMachine.agents.map((agent) => (
                    <option key={agent.id} value={agent.id}>
                      {agent.name} ({agent.model})
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            <div>
              <label className="block text-xs text-zinc-400 mb-2">话题名称</label>
              <input
                type="text"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder="例如: 实现用户认证功能"
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-xs text-zinc-400 mb-2">工作目录</label>
              <div className="relative">
                <FolderOpen className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-zinc-600" />
                <input
                  type="text"
                  value={workDir}
                  onChange={(event) => setWorkDir(event.target.value)}
                  placeholder="/workspace"
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg pl-10 pr-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 mt-6">
            <button
              onClick={handleCreate}
              disabled={!selectedMachineId || !selectedAgentId || !title.trim()}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
            >
              <Check className="size-4" />
              创建
            </button>
            <Dialog.Close asChild>
              <button className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors">
                <X className="size-4" />
                取消
              </button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

export default function ThreadPanel({
  machines,
  selectedSessionId,
  onSelectSession,
  onNavigate,
  onRenameSession,
  onDeleteSession,
  onCreateThread,
}: ThreadPanelProps) {
  const [newThreadDialogOpen, setNewThreadDialogOpen] = useState(false);

  const totalActive = machines.reduce(
    (sum, machine) => sum + machine.sessions.filter((session) => session.status === "active").length,
    0,
  );
  const onlineMachines = machines.filter((machine) => machine.status === "online").length;

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 px-4 py-3 border-b border-zinc-800/60">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <MessageSquare className="size-3.5 text-zinc-500" />
            <span className="text-xs text-zinc-400">线程</span>
          </div>
          <button
            onClick={() => setNewThreadDialogOpen(true)}
            className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 transition-colors"
          >
            <Plus className="size-3" />
            新建
          </button>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1">
            <div className="size-1.5 rounded-full bg-emerald-400" />
            <span className="text-[10px] text-zinc-500">{onlineMachines} 台在线</span>
          </div>
          <div className="flex items-center gap-1">
            <Server className="size-3 text-zinc-600" />
            <span className="text-[10px] text-zinc-500">{totalActive} 个活跃</span>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto py-2 px-2">
        {machines.map((machine) => (
          <MachineGroup
            key={machine.id}
            machine={machine}
            selectedSessionId={selectedSessionId}
            onSelectSession={onSelectSession}
            onRenameSession={onRenameSession}
            onDeleteSession={onDeleteSession}
          />
        ))}
      </div>

      {onNavigate ? (
        <div className="flex-shrink-0 border-t border-zinc-800/60 px-2 py-2">
          <div className="space-y-0.5">
            <button
              onClick={() => onNavigate("overview")}
              className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50 transition-colors"
            >
              <Activity className="size-4 flex-shrink-0" />
              <span className="text-xs">概览</span>
            </button>
            <button
              onClick={() => onNavigate("machines")}
              className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50 transition-colors"
            >
              <Server className="size-4 flex-shrink-0" />
              <span className="text-xs">机器管理</span>
            </button>
            <button
              onClick={() => onNavigate("environment")}
              className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50 transition-colors"
            >
              <Package className="size-4 flex-shrink-0" />
              <span className="text-xs">环境资源</span>
            </button>
            <button
              onClick={() => onNavigate("settings")}
              className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50 transition-colors"
            >
              <Settings className="size-4 flex-shrink-0" />
              <span className="text-xs">设置</span>
            </button>
          </div>
        </div>
      ) : null}

      <NewThreadDialog
        machines={machines}
        open={newThreadDialogOpen}
        onOpenChange={setNewThreadDialogOpen}
        onCreate={(machineId, agentId, title, workDir) => {
          onCreateThread?.(machineId, agentId, title, workDir);
        }}
      />
    </div>
  );
}
