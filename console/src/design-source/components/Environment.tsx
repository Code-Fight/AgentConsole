import { useState } from "react";
import { Package, Puzzle, Blocks, Plus, Trash2, X, Check, Server, Bot } from "lucide-react";
import * as Dialog from "@radix-ui/react-dialog";
import type { Machine, SkillResource, MCPResource, PluginResource } from "../data/mockData";

interface EnvironmentProps {
  machines: Machine[];
  skills?: SkillResource[];
  mcps?: MCPResource[];
  plugins?: PluginResource[];
  onAddSkill?: (machineId: string, agentId: string, name: string, description: string) => void;
  onAddMCP?: (machineId: string, agentId: string, name: string, serverUrl: string) => void;
  onAddPlugin?: (machineId: string, agentId: string, name: string, version: string) => void;
  onDeleteSkill?: (skillId: string) => void;
  onDeleteMCP?: (mcpId: string) => void;
  onDeletePlugin?: (pluginId: string) => void;
}

type ResourceType = "skill" | "mcp" | "plugin";

function AddResourceDialog({
  type,
  machines,
  open,
  onOpenChange,
  onAdd,
}: {
  type: ResourceType;
  machines: Machine[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (machineId: string, agentId: string, data: any) => void;
}) {
  const [selectedMachineId, setSelectedMachineId] = useState("");
  const [selectedAgentId, setSelectedAgentId] = useState("");
  const [name, setName] = useState("");
  const [extraField, setExtraField] = useState("");

  const onlineMachines = machines.filter((m) => m.status === "online" && m.agents.length > 0);
  const selectedMachine = onlineMachines.find((m) => m.id === selectedMachineId);

  const handleMachineChange = (machineId: string) => {
    setSelectedMachineId(machineId);
    const machine = onlineMachines.find((m) => m.id === machineId);
    if (machine && machine.agents.length > 0) {
      setSelectedAgentId(machine.agents[0].id);
    } else {
      setSelectedAgentId("");
    }
  };

  const handleAdd = () => {
    if (!selectedMachineId || !selectedAgentId || !name.trim()) return;

    if (type === "skill") {
      onAdd(selectedMachineId, selectedAgentId, { name: name.trim(), description: extraField });
    } else if (type === "mcp") {
      onAdd(selectedMachineId, selectedAgentId, { name: name.trim(), serverUrl: extraField });
    } else if (type === "plugin") {
      onAdd(selectedMachineId, selectedAgentId, { name: name.trim(), version: extraField || "1.0.0" });
    }

    onOpenChange(false);
    setSelectedMachineId("");
    setSelectedAgentId("");
    setName("");
    setExtraField("");
  };

  const titles = {
    skill: "添加 Skill",
    mcp: "添加 MCP Server",
    plugin: "添加 Plugin",
  };

  const extraLabels = {
    skill: "描述",
    mcp: "Server URL",
    plugin: "版本",
  };

  const extraPlaceholders = {
    skill: "例如: 代码生成工具",
    mcp: "https://mcp.example.com",
    plugin: "1.0.0",
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
          <Dialog.Title className="text-lg text-zinc-100 mb-1">{titles[type]}</Dialog.Title>
          <Dialog.Description className="text-xs text-zinc-500 mb-6">
            选择目标机器和 Agent，然后添加资源
          </Dialog.Description>

          <div className="space-y-4">
            <div>
              <label className="block text-xs text-zinc-400 mb-2">目标机器</label>
              <select
                value={selectedMachineId}
                onChange={(e) => handleMachineChange(e.target.value)}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
              >
                <option value="">选择机器</option>
                {onlineMachines.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </select>
            </div>

            {selectedMachine ? (
              <div>
                <label className="block text-xs text-zinc-400 mb-2">目标 Agent</label>
                <select
                  value={selectedAgentId}
                  onChange={(e) => setSelectedAgentId(e.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
                >
                  {selectedMachine.agents.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name} ({a.model})
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            <div>
              <label className="block text-xs text-zinc-400 mb-2">名称</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={`例如: ${type === "skill" ? "code-generator" : type === "mcp" ? "github-mcp" : "my-plugin"}`}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-xs text-zinc-400 mb-2">{extraLabels[type]}</label>
              <input
                type="text"
                value={extraField}
                onChange={(e) => setExtraField(e.target.value)}
                placeholder={extraPlaceholders[type]}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div className="flex items-center gap-2 mt-6">
            <button
              onClick={handleAdd}
              disabled={!selectedMachineId || !selectedAgentId || !name.trim()}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
            >
              <Check className="size-4" />
              添加
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

export default function Environment({
  machines,
  skills = [],
  mcps = [],
  plugins = [],
  onAddSkill,
  onAddMCP,
  onAddPlugin,
  onDeleteSkill,
  onDeleteMCP,
  onDeletePlugin,
}: EnvironmentProps) {
  const [addDialogType, setAddDialogType] = useState<ResourceType | null>(null);

  const handleAddSkill = (machineId: string, agentId: string, data: any) => {
    onAddSkill?.(machineId, agentId, data.name, data.description);
  };

  const handleAddMCP = (machineId: string, agentId: string, data: any) => {
    onAddMCP?.(machineId, agentId, data.name, data.serverUrl);
  };

  const handleAddPlugin = (machineId: string, agentId: string, data: any) => {
    onAddPlugin?.(machineId, agentId, data.name, data.version);
  };

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
            <Package className="size-5 text-zinc-400" />
          </div>
          <div>
            <h1 className="text-lg text-zinc-100">环境资源</h1>
            <p className="text-xs text-zinc-500 mt-0.5">管理绑定到机器 Agent 的 Skill、MCP 和 Plugin 资源</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-5xl mx-auto space-y-6">
          <section>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Puzzle className="size-4 text-violet-400" />
                <h2 className="text-sm text-zinc-300">Skills</h2>
                <span className="text-xs text-zinc-600">({skills.length})</span>
              </div>
              <button
                onClick={() => setAddDialogType("skill")}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
              >
                <Plus className="size-3.5" />
                添加
              </button>
            </div>
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
              {skills.length > 0 ? (
                <div className="space-y-2">
                  {skills.map((skill) => (
                    <div key={skill.id} className="flex items-start justify-between gap-3 p-3 bg-zinc-800/50 rounded-lg group hover:bg-zinc-800 transition-colors">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm text-zinc-300 mb-1">{skill.name}</p>
                        <p className="text-xs text-zinc-600 mb-2">{skill.description}</p>
                        <div className="flex items-center gap-2 text-[10px] text-zinc-600">
                          <Server className="size-3" />
                          <span>{skill.machineName}</span>
                          <span>›</span>
                          <Bot className="size-3" />
                          <span>{skill.agentName}</span>
                        </div>
                      </div>
                      <button
                        onClick={() => onDeleteSkill?.(skill.id)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 text-red-400 hover:text-red-300 hover:bg-zinc-700 rounded transition-all"
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-zinc-500 text-center py-8">暂无 Skill 配置</p>
              )}
            </div>
          </section>

          <section>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Blocks className="size-4 text-blue-400" />
                <h2 className="text-sm text-zinc-300">MCP Servers</h2>
                <span className="text-xs text-zinc-600">({mcps.length})</span>
              </div>
              <button
                onClick={() => setAddDialogType("mcp")}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
              >
                <Plus className="size-3.5" />
                添加
              </button>
            </div>
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
              {mcps.length > 0 ? (
                <div className="space-y-2">
                  {mcps.map((mcp) => (
                    <div key={mcp.id} className="flex items-start justify-between gap-3 p-3 bg-zinc-800/50 rounded-lg group hover:bg-zinc-800 transition-colors">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm text-zinc-300 mb-1">{mcp.name}</p>
                        <p className="text-xs text-zinc-600 font-mono mb-2">{mcp.serverUrl}</p>
                        <div className="flex items-center gap-2 text-[10px] text-zinc-600">
                          <Server className="size-3" />
                          <span>{mcp.machineName}</span>
                          <span>›</span>
                          <Bot className="size-3" />
                          <span>{mcp.agentName}</span>
                        </div>
                      </div>
                      <button
                        onClick={() => onDeleteMCP?.(mcp.id)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 text-red-400 hover:text-red-300 hover:bg-zinc-700 rounded transition-all"
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-zinc-500 text-center py-8">暂无 MCP Server 配置</p>
              )}
            </div>
          </section>

          <section>
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <Package className="size-4 text-emerald-400" />
                <h2 className="text-sm text-zinc-300">Plugins</h2>
                <span className="text-xs text-zinc-600">({plugins.length})</span>
              </div>
              <button
                onClick={() => setAddDialogType("plugin")}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
              >
                <Plus className="size-3.5" />
                添加
              </button>
            </div>
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
              {plugins.length > 0 ? (
                <div className="space-y-2">
                  {plugins.map((plugin) => (
                    <div key={plugin.id} className="flex items-start justify-between gap-3 p-3 bg-zinc-800/50 rounded-lg group hover:bg-zinc-800 transition-colors">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <p className="text-sm text-zinc-300">{plugin.name}</p>
                          <span className="text-[10px] text-zinc-600 bg-zinc-700 px-1.5 py-0.5 rounded">
                            v{plugin.version}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-zinc-600">
                          <Server className="size-3" />
                          <span>{plugin.machineName}</span>
                          <span>›</span>
                          <Bot className="size-3" />
                          <span>{plugin.agentName}</span>
                        </div>
                      </div>
                      <button
                        onClick={() => onDeletePlugin?.(plugin.id)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 text-red-400 hover:text-red-300 hover:bg-zinc-700 rounded transition-all"
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-zinc-500 text-center py-8">暂无 Plugin 配置</p>
              )}
            </div>
          </section>
        </div>
      </div>

      {addDialogType === "skill" ? (
        <AddResourceDialog
          type="skill"
          machines={machines}
          open={addDialogType === "skill"}
          onOpenChange={(open) => !open && setAddDialogType(null)}
          onAdd={handleAddSkill}
        />
      ) : null}

      {addDialogType === "mcp" ? (
        <AddResourceDialog
          type="mcp"
          machines={machines}
          open={addDialogType === "mcp"}
          onOpenChange={(open) => !open && setAddDialogType(null)}
          onAdd={handleAddMCP}
        />
      ) : null}

      {addDialogType === "plugin" ? (
        <AddResourceDialog
          type="plugin"
          machines={machines}
          open={addDialogType === "plugin"}
          onOpenChange={(open) => !open && setAddDialogType(null)}
          onAdd={handleAddPlugin}
        />
      ) : null}
    </div>
  );
}
