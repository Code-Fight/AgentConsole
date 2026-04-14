import { useEffect, useState } from "react";
import { Server, Wifi, WifiOff, Bot, Plus, Trash2, X, Check, Edit } from "lucide-react";
import * as ContextMenu from "@radix-ui/react-context-menu";
import * as Dialog from "@radix-ui/react-dialog";
import type { Machine, AgentInfo } from "../data/mockData";
import { http } from "../../common/api/http";

interface MachinesProps {
  machines: Machine[];
  onInstallAgent?: (machineId: string, agentType: string, agentName: string) => void;
  onDeleteAgent?: (machineId: string, agentId: string) => void;
  onUpdateAgentConfig?: (machineId: string, agentId: string, config: string) => void;
}

const statusConfig = {
  online: {
    dot: "bg-emerald-400",
    text: "text-emerald-400",
    label: "在线",
    icon: Wifi,
    ring: "ring-emerald-400/20",
  },
  offline: {
    dot: "bg-zinc-600",
    text: "text-zinc-500",
    label: "离线",
    icon: WifiOff,
    ring: "ring-zinc-600/20",
  },
};

const agentTypeOptions = [
  { value: "claude-code", label: "Claude Code" },
  { value: "codex", label: "Codex" },
  { value: "custom", label: "Custom" },
];

function AgentCard({
  agent,
  machineId,
  onDelete,
  onEdit,
}: {
  agent: AgentInfo;
  machineId: string;
  onDelete: (machineId: string, agentId: string) => void;
  onEdit: (machineId: string, agentId: string) => void;
}) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleDeleteClick = () => {
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    onDelete(machineId, agent.id);
    setDeleteDialogOpen(false);
  };

  const handleEdit = () => {
    onEdit(machineId, agent.id);
  };

  const agentContent = (
    <div className="flex items-center gap-2 px-3 py-2.5 bg-zinc-800/50 rounded-lg group">
      <div
        className={`size-1.5 rounded-full flex-shrink-0 ${
          agent.status === "active"
            ? "bg-emerald-400 animate-pulse"
            : agent.status === "idle"
              ? "bg-zinc-600"
              : "bg-red-400"
        }`}
      />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-zinc-300 truncate">{agent.name}</p>
        <p className="text-[10px] text-zinc-600 font-mono truncate">{agent.model}</p>
      </div>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100">
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleEdit();
          }}
          className="p-1 text-blue-400 hover:text-blue-300 hover:bg-zinc-700 rounded transition-all"
          title="编辑配置"
        >
          <Edit className="size-3" />
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            handleDeleteClick();
          }}
          className="p-1 text-red-400 hover:text-red-300 hover:bg-zinc-700 rounded transition-all"
          title="删除 Agent"
        >
          <Trash2 className="size-3" />
        </button>
      </div>
    </div>
  );

  return (
    <>
      <ContextMenu.Root>
        <ContextMenu.Trigger asChild>{agentContent}</ContextMenu.Trigger>
        <ContextMenu.Portal>
          <ContextMenu.Content
            className="min-w-[160px] bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl py-1.5 z-50"
          >
            <ContextMenu.Item
              onClick={handleEdit}
              className="flex items-center gap-2.5 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-700 hover:text-zinc-50 cursor-pointer outline-none"
            >
              <Bot className="size-3.5" />
              <span>编辑配置</span>
            </ContextMenu.Item>
            <ContextMenu.Separator className="h-px bg-zinc-700 my-1" />
            <ContextMenu.Item
              onClick={handleDeleteClick}
              className="flex items-center gap-2.5 px-3 py-2 text-xs text-red-400 hover:bg-zinc-700 hover:text-red-300 cursor-pointer outline-none"
            >
              <Trash2 className="size-3.5" />
              <span>删除 Agent</span>
            </ContextMenu.Item>
          </ContextMenu.Content>
        </ContextMenu.Portal>
      </ContextMenu.Root>

      <Dialog.Root open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-sm z-50 p-6">
            <Dialog.Title className="text-lg text-zinc-100 mb-2">确认删除 Agent</Dialog.Title>
            <Dialog.Description className="text-sm text-zinc-500 mb-6">
              确定要删除 Agent "{agent.name}" 吗？此操作无法撤销。
            </Dialog.Description>

            <div className="flex items-center gap-2">
              <button
                onClick={handleConfirmDelete}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg text-sm transition-colors"
              >
                删除
              </button>
              <Dialog.Close asChild>
                <button className="flex-1 px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-sm transition-colors">
                  取消
                </button>
              </Dialog.Close>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </>
  );
}

function InstallAgentDialog({
  machine,
  open,
  onOpenChange,
  onInstall,
}: {
  machine: Machine;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInstall: (machineId: string, agentType: string, agentName: string) => void;
}) {
  const [agentType, setAgentType] = useState<"claude-code" | "codex" | "custom">("claude-code");
  const [agentName, setAgentName] = useState("");

  const handleInstall = () => {
    if (!agentName.trim()) return;
    onInstall(machine.id, agentType, agentName.trim());
    onOpenChange(false);
    setAgentName("");
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
          <Dialog.Title className="text-lg text-zinc-100 mb-1">安装 Agent</Dialog.Title>
          <Dialog.Description className="text-xs text-zinc-500 mb-6">
            在 {machine.name} 上安装新的 Agent
          </Dialog.Description>

          <div className="space-y-4">
            <div>
              <label className="block text-xs text-zinc-400 mb-2">Agent 类型</label>
              <select
                value={agentType}
                onChange={(e) => setAgentType(e.target.value as "claude-code" | "codex" | "custom")}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
              >
                {agentTypeOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-zinc-400 mb-2">Agent 名称</label>
              <input
                type="text"
                value={agentName}
                onChange={(e) => setAgentName(e.target.value)}
                placeholder="例如: Claude Sonnet"
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div className="flex items-center gap-2 mt-6">
            <button
              onClick={handleInstall}
              disabled={!agentName.trim()}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
            >
              <Check className="size-4" />
              安装
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

function EditAgentConfigDialog({
  agent,
  machine,
  open,
  onOpenChange,
  onSave,
}: {
  agent: AgentInfo;
  machine: Machine;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (machineId: string, agentId: string, config: string) => void;
}) {
  const defaultConfigs: Record<string, string> = {
    "claude-code": `# Claude Code Agent Configuration
model = "claude-sonnet-4-5"
temperature = 0.7
max_tokens = 4096

[timeout]
default = 60000

[retry]
max_retries = 3
backoff = "exponential"`,
    codex: `# Codex Agent Configuration
model = "claude-sonnet-4-5"
api_version = "v1"
enable_caching = true

[timeout]
default = 30000`,
    custom: `# Custom Agent Configuration
model = "gpt-4-turbo"
temperature = 0.8
max_tokens = 2048`,
  };

  const [config, setConfig] = useState(defaultConfigs[agent.type] || defaultConfigs.custom);

  useEffect(() => {
    if (!open) {
      return;
    }

    let active = true;
    const loadConfig = async () => {
      const fallbackConfig = defaultConfigs[agent.type] ?? defaultConfigs.custom;
      try {
        const response = await http<{ document?: { content?: string } }>(
          `/machines/${encodeURIComponent(machine.id)}/agents/${encodeURIComponent(agent.id)}/config`,
        );
        if (!active) {
          return;
        }
        setConfig(response.document?.content ?? fallbackConfig);
      } catch {
        if (!active) {
          return;
        }
        setConfig(fallbackConfig);
      }
    };

    void loadConfig();
    return () => {
      active = false;
    };
  }, [open, machine.id, agent.id, agent.type]);

  const handleSave = () => {
    onSave(machine.id, agent.id, config);
    onOpenChange(false);
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-2xl z-50 p-6">
          <Dialog.Title className="text-lg text-zinc-100 mb-1">编辑 Agent 配置</Dialog.Title>
          <Dialog.Description className="text-xs text-zinc-500 mb-6">
            {agent.name} ({agent.type}) @ {machine.name}
          </Dialog.Description>

          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <label className="block text-sm text-zinc-400">配置文件 (TOML)</label>
              <span className="text-xs text-zinc-600">类型: {agent.type}</span>
            </div>
            <textarea
              value={config}
              onChange={(e) => setConfig(e.target.value)}
              rows={18}
              className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-300 font-mono focus:outline-none focus:border-blue-500 transition-colors resize-none"
              placeholder="输入 TOML 配置..."
            />
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleSave}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-sm transition-colors"
            >
              <Check className="size-4" />
              保存
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

export default function Machines({
  machines,
  onInstallAgent,
  onDeleteAgent,
  onUpdateAgentConfig,
}: MachinesProps) {
  const [installDialogMachine, setInstallDialogMachine] = useState<Machine | null>(null);
  const [editingAgent, setEditingAgent] = useState<{ machine: Machine; agent: AgentInfo } | null>(null);

  const handleInstall = (machineId: string, agentType: string, agentName: string) => {
    if (onInstallAgent) {
      onInstallAgent(machineId, agentType, agentName);
    }
  };

  const handleDelete = (machineId: string, agentId: string) => {
    if (onDeleteAgent) {
      onDeleteAgent(machineId, agentId);
    }
  };

  const handleEditConfig = (machineId: string, agentId: string) => {
    const machine = machines.find((m) => m.id === machineId);
    const agent = machine?.agents.find((a) => a.id === agentId);
    if (machine && agent) {
      setEditingAgent({ machine, agent });
    }
  };

  const handleSaveConfig = (machineId: string, agentId: string, config: string) => {
    if (onUpdateAgentConfig) {
      onUpdateAgentConfig(machineId, agentId, config);
    }
  };

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
            <Server className="size-5 text-zinc-400" />
          </div>
          <div>
            <h1 className="text-lg text-zinc-100">机器管理</h1>
            <p className="text-xs text-zinc-500 mt-0.5">管理接入的机器及其 Agent</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-5xl mx-auto space-y-4">
          {machines.map((machine) => {
            const cfg = statusConfig[machine.status];
            const StatusIcon = cfg.icon;

            return (
              <div
                key={machine.id}
                className="bg-zinc-900 border border-zinc-800 rounded-xl p-5 hover:border-zinc-700 transition-colors"
              >
                <div className="flex items-start justify-between gap-4 mb-4">
                  <div className="flex items-start gap-3 flex-1 min-w-0">
                    <div className={`size-10 rounded-lg bg-zinc-800 flex items-center justify-center ring-2 ${cfg.ring}`}>
                      <StatusIcon className={`size-5 ${cfg.text}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="text-sm text-zinc-100 font-medium">{machine.name}</h3>
                        <span className={`text-xs ${cfg.text} flex items-center gap-1`}>
                          <div className={`size-1.5 rounded-full ${cfg.dot}`} />
                          {cfg.label}
                        </span>
                      </div>
                      <p className="text-xs text-zinc-500 font-mono">{machine.host}</p>
                      <p className="text-xs text-zinc-600 mt-0.5">{machine.os}</p>
                    </div>
                  </div>
                </div>

                <div className="border-t border-zinc-800 pt-4">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <Bot className="size-3.5 text-zinc-500" />
                      <span className="text-xs text-zinc-400">Agents ({machine.agents.length})</span>
                    </div>
                    <button
                      onClick={() => setInstallDialogMachine(machine)}
                      disabled={machine.status === "offline"}
                      className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg transition-colors"
                    >
                      <Plus className="size-3.5" />
                      安装 Agent
                    </button>
                  </div>
                  {machine.agents.length > 0 ? (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      {machine.agents.map((agent) => (
                        <AgentCard
                          key={agent.id}
                          agent={agent}
                          machineId={machine.id}
                          onDelete={handleDelete}
                          onEdit={handleEditConfig}
                        />
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-zinc-600 italic">暂无 Agent</p>
                  )}
                </div>

                <div className="border-t border-zinc-800 pt-4 mt-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs text-zinc-400">
                      活跃线程: {machine.sessions.filter((s) => s.status === "active").length} / {machine.sessions.length}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {installDialogMachine ? (
        <InstallAgentDialog
          machine={installDialogMachine}
          open={!!installDialogMachine}
          onOpenChange={(open) => !open && setInstallDialogMachine(null)}
          onInstall={handleInstall}
        />
      ) : null}

      {editingAgent ? (
        <EditAgentConfigDialog
          agent={editingAgent.agent}
          machine={editingAgent.machine}
          open={!!editingAgent}
          onOpenChange={(open) => !open && setEditingAgent(null)}
          onSave={handleSaveConfig}
        />
      ) : null}
    </div>
  );
}
