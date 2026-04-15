import { type ReactNode, useEffect, useMemo, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import {
  Blocks,
  Bot,
  Check,
  ExternalLink,
  Package,
  Pencil,
  Plus,
  Puzzle,
  RefreshCw,
  Server,
  Trash2,
  X,
} from "lucide-react";
import type { EnvironmentResource } from "../../common/api/types";
import { useEnvironmentPage } from "../../gateway/use-environment-page";
import type { Machine } from "../data/mockData";

interface EnvironmentProps {
  machines: Machine[];
}

type ResourceType = "skill" | "mcp" | "plugin";

interface ResourceMutation {
  label: string;
  method: "POST" | "DELETE";
  path: string;
  payload?: Record<string, unknown>;
}

function formatStatus(resource: EnvironmentResource): string {
  switch (resource.status) {
    case "enabled":
      return "Enabled";
    case "disabled":
      return "Disabled";
    case "auth_required":
      return "Auth required";
    case "error":
      return "Error";
    case "unknown":
      return "Not installed";
    default:
      return resource.status;
  }
}

function buildMutations(resource: EnvironmentResource): ResourceMutation[] {
  if (resource.kind === "skill") {
    const action = resource.status === "enabled" ? "disable" : "enable";
    return [
      {
        label: resource.status === "enabled" ? "Disable" : "Enable",
        method: "POST",
        path: `/environment/skills/${encodeURIComponent(resource.resourceId)}/${action}`,
      },
    ];
  }

  if (resource.kind === "mcp") {
    const action = resource.status === "enabled" ? "disable" : "enable";
    return [
      {
        label: resource.status === "enabled" ? "Disable" : "Enable",
        method: "POST",
        path: `/environment/mcps/${encodeURIComponent(resource.resourceId)}/${action}`,
      },
      {
        label: "Delete",
        method: "DELETE",
        path: `/environment/mcps/${encodeURIComponent(resource.resourceId)}`,
      },
    ];
  }

  if (resource.kind === "plugin") {
    if (resource.status === "unknown") {
      const details = resource.details ?? {};
      const pluginName =
        typeof details.pluginName === "string" && details.pluginName.trim()
          ? details.pluginName
          : resource.displayName || resource.resourceId;
      const marketplacePath =
        typeof details.marketplacePath === "string" ? details.marketplacePath : "";
      return [
        {
          label: "Install",
          method: "POST",
          path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}/install`,
          payload: {
            pluginId: resource.resourceId,
            pluginName,
            marketplacePath,
          },
        },
      ];
    }

    const action = resource.status === "enabled" ? "disable" : "enable";
    return [
      {
        label: resource.status === "enabled" ? "Disable" : "Enable",
        method: "POST",
        path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}/${action}`,
      },
      {
        label: "Uninstall",
        method: "DELETE",
        path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}`,
      },
    ];
  }

  return [];
}

function buildResourceKey(resource: EnvironmentResource) {
  return `${resource.kind}:${resource.machineId}:${resource.resourceId}`;
}

function statusBadgeClasses(resource: EnvironmentResource) {
  switch (resource.status) {
    case "enabled":
      return "bg-emerald-500/15 text-emerald-300 border border-emerald-500/20";
    case "disabled":
      return "bg-zinc-700 text-zinc-300 border border-zinc-600";
    case "auth_required":
      return "bg-amber-500/15 text-amber-300 border border-amber-500/20";
    case "error":
      return "bg-red-500/15 text-red-300 border border-red-500/20";
    default:
      return "bg-blue-500/15 text-blue-300 border border-blue-500/20";
  }
}

function resourceDescription(resource: EnvironmentResource) {
  if (resource.kind === "skill") {
    const description = resource.details?.description;
    if (typeof description === "string" && description.trim()) {
      return description;
    }
    return "Skill scaffold managed through Gateway.";
  }

  if (resource.kind === "mcp") {
    const command = resource.details?.config;
    if (command && typeof command === "object") {
      return "MCP server configuration synced from Gateway.";
    }
    return "MCP server registered on the selected machine.";
  }

  const marketplacePath = resource.details?.marketplacePath;
  if (typeof marketplacePath === "string" && marketplacePath.trim()) {
    return marketplacePath;
  }
  return "Plugin record managed through Gateway.";
}

function formatDetailLabel(key: string) {
  return key
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/_/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function renderDetailValue(value: unknown): ReactNode {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <span className="text-xs text-zinc-500">Empty</span>;
    }

    return (
      <div className="flex flex-wrap gap-2">
        {value.map((item, index) => (
          <span
            key={`${String(item)}-${index}`}
            className="text-[10px] text-zinc-300 bg-zinc-950 border border-zinc-700 px-2 py-1 rounded"
          >
            {typeof item === "string" ? item : JSON.stringify(item)}
          </span>
        ))}
      </div>
    );
  }

  if (value && typeof value === "object") {
    return (
      <pre className="bg-zinc-950 border border-zinc-700 rounded-lg p-3 text-[11px] text-zinc-300 overflow-x-auto">
        {JSON.stringify(value, null, 2)}
      </pre>
    );
  }

  if (value === null || value === undefined || value === "") {
    return <span className="text-xs text-zinc-500">Empty</span>;
  }

  return <span className="text-xs text-zinc-300">{String(value)}</span>;
}

function resourceVersion(resource: EnvironmentResource) {
  const version = resource.details?.version;
  if (typeof version === "string" && version.trim()) {
    return version;
  }
  return null;
}

export default function Environment({ machines }: EnvironmentProps) {
  const vm = useEnvironmentPage();
  const dialogContainer =
    typeof document === "undefined" ? undefined : document.querySelector("main") ?? undefined;
  const onlineMachines = useMemo(
    () => machines.filter((machine) => machine.status === "online" && machine.agents.length > 0),
    [machines],
  );
  const machineById = useMemo(
    () => new Map(machines.map((machine) => [machine.id, machine])),
    [machines],
  );
  const [skillAgentId, setSkillAgentId] = useState("");
  const [mcpAgentId, setMcpAgentId] = useState("");
  const [pluginAgentId, setPluginAgentId] = useState("");

  useEffect(() => {
    if (!vm.skillForm) {
      setSkillAgentId("");
      return;
    }
    const machine = machineById.get(vm.skillForm.machineId);
    setSkillAgentId(machine?.agents[0]?.id ?? "");
  }, [vm.skillForm, machineById]);

  useEffect(() => {
    if (!vm.mcpForm) {
      setMcpAgentId("");
      return;
    }
    const machine = machineById.get(vm.mcpForm.machineId);
    setMcpAgentId(machine?.agents[0]?.id ?? "");
  }, [vm.mcpForm, machineById]);

  useEffect(() => {
    if (!vm.pluginForm) {
      setPluginAgentId("");
      return;
    }
    const machine = machineById.get(vm.pluginForm.machineId);
    setPluginAgentId(machine?.agents[0]?.id ?? "");
  }, [vm.pluginForm, machineById]);

  const handleMachineSelection = (
    type: ResourceType,
    machineId: string,
  ) => {
    const machine = machineById.get(machineId);
    const defaultAgentId = machine?.agents[0]?.id ?? "";

    if (type === "skill") {
      setSkillAgentId(defaultAgentId);
      vm.setSkillForm((current) =>
        current ? { ...current, machineId } : current,
      );
      return;
    }

    if (type === "mcp") {
      setMcpAgentId(defaultAgentId);
      vm.setMcpForm((current) =>
        current ? { ...current, machineId } : current,
      );
      return;
    }

    setPluginAgentId(defaultAgentId);
    vm.setPluginForm((current) =>
      current ? { ...current, machineId } : current,
    );
  };

  const renderMachineAgent = (resource: EnvironmentResource) => {
    const machine = machineById.get(resource.machineId);
    const machineLabel = machine?.name || resource.machineId;
    const agent =
      machine?.agents.find((item) => item.id === resource.agentId) ?? machine?.agents[0] ?? null;

    return (
      <div className="flex items-center gap-2 text-[10px] text-zinc-600">
        <Server className="size-3" />
        <span>{machineLabel}</span>
        <span>›</span>
        <Bot className="size-3" />
        <span>{agent?.name ?? resource.agentId ?? "默认 Agent"}</span>
      </div>
    );
  };

  const renderResourceSection = (
    type: ResourceType,
    title: string,
    icon: React.ReactNode,
    addActionLabel: string,
    items: EnvironmentResource[],
  ) => (
    <section>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          {icon}
          <h2 className="text-sm text-zinc-300">{title}</h2>
          <span className="text-xs text-zinc-600">({items.length})</span>
        </div>
        <button
          type="button"
          aria-label={addActionLabel}
          onClick={() => {
            if (type === "skill") {
              vm.openCreateSkillForm();
            } else if (type === "mcp") {
              vm.openCreateMCPForm();
            } else {
              vm.openInstallPluginForm();
            }
          }}
          disabled={
            type === "skill"
              ? !vm.capabilities.writeSkills
              : !vm.capabilities.writeMcp && type === "mcp"
          }
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 disabled:bg-zinc-800/60 disabled:text-zinc-600 rounded-lg transition-colors"
        >
          <Plus className="size-3.5" />
          添加
        </button>
      </div>
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
        {items.length > 0 ? (
          <div className="space-y-2">
            {items.map((resource) => {
              const resourceKey = buildResourceKey(resource);
              const mutations = buildMutations(resource);
              const expanded = vm.expandedResourceKey === resourceKey;
              const version = resourceVersion(resource);
              const canDeleteSkill = type === "skill" && vm.capabilities.writeSkills;
              const hasDetails = Boolean(
                resource.details && Object.keys(resource.details).length > 0,
              );

              return (
                <article
                  key={resourceKey}
                  className="p-3 bg-zinc-800/50 rounded-lg group hover:bg-zinc-800 transition-colors"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0 space-y-2">
                      <div className="flex items-center gap-2">
                        <p className="text-sm text-zinc-300">
                          {resource.displayName || resource.resourceId}
                        </p>
                        {version ? (
                          <span className="text-[10px] text-zinc-600 bg-zinc-700 px-1.5 py-0.5 rounded">
                            v{version}
                          </span>
                        ) : null}
                        <span
                          className={`text-[10px] px-1.5 py-0.5 rounded ${statusBadgeClasses(resource)}`}
                        >
                          {formatStatus(resource)}
                        </span>
                        {resource.restartRequired ? (
                          <span className="text-[10px] text-amber-300 bg-amber-500/10 border border-amber-500/20 px-1.5 py-0.5 rounded">
                            Restart required
                          </span>
                        ) : null}
                      </div>
                      <p className="text-xs text-zinc-600">{resourceDescription(resource)}</p>
                      {renderMachineAgent(resource)}
                      {expanded && hasDetails ? (
                        <div className="bg-zinc-950 border border-zinc-700 rounded-lg p-3 space-y-3">
                          {Object.entries(resource.details ?? {}).map(([key, value]) => (
                            <div key={key} className="space-y-1">
                              <p className="text-[10px] uppercase tracking-wide text-zinc-500">
                                {formatDetailLabel(key)}
                              </p>
                              {renderDetailValue(value)}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap justify-end gap-2">
                      {hasDetails ? (
                        <button
                          type="button"
                          aria-label={expanded ? "Hide details" : "View details"}
                          onClick={() => vm.toggleDetails(resource)}
                          className="px-2.5 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-900 hover:bg-zinc-700 rounded-md transition-colors"
                        >
                          {expanded ? "收起详情" : "查看详情"}
                        </button>
                      ) : null}
                      {type === "mcp" ? (
                        <button
                          type="button"
                          aria-label="Edit"
                          disabled={!vm.capabilities.writeMcp}
                          onClick={() => vm.openEditMCPForm(resource)}
                          className="px-2.5 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 bg-zinc-900 hover:bg-zinc-700 disabled:bg-zinc-900/60 disabled:text-zinc-600 rounded-md transition-colors"
                        >
                          <span className="inline-flex items-center gap-1">
                            <Pencil className="size-3" />
                            编辑
                          </span>
                        </button>
                      ) : null}
                      {canDeleteSkill ? (
                        <button
                          type="button"
                          aria-label="Delete skill"
                          disabled={vm.pendingActionKey === `${resourceKey}:Delete skill`}
                          onClick={() =>
                            vm.handleResourceMutation(
                              resource,
                              `${resourceKey}:Delete skill`,
                              "DELETE",
                              `/environment/skills/${encodeURIComponent(resource.resourceId)}`,
                            )
                          }
                          className="px-2.5 py-1.5 text-xs text-red-300 hover:text-red-200 bg-red-500/10 hover:bg-red-500/15 disabled:bg-red-500/5 disabled:text-red-500 rounded-md transition-colors"
                        >
                          删除
                        </button>
                      ) : null}
                      {mutations.map((mutation) => {
                        const actionKey = `${resourceKey}:${mutation.label}`;
                        const destructive =
                          mutation.label === "Delete" || mutation.label === "Uninstall";

                        return (
                          <button
                            key={actionKey}
                            type="button"
                            aria-label={mutation.label}
                            disabled={
                              !vm.capabilities.mutateResources ||
                              vm.pendingActionKey === actionKey
                            }
                            onClick={() =>
                              vm.handleResourceMutation(
                                resource,
                                actionKey,
                                mutation.method,
                                mutation.path,
                                mutation.payload,
                              )
                            }
                            className={`px-2.5 py-1.5 text-xs rounded-md transition-colors ${
                              destructive
                                ? "text-red-300 hover:text-red-200 bg-red-500/10 hover:bg-red-500/15 disabled:bg-red-500/5 disabled:text-red-500"
                                : "text-zinc-400 hover:text-zinc-200 bg-zinc-900 hover:bg-zinc-700 disabled:bg-zinc-900/60 disabled:text-zinc-600"
                            }`}
                          >
                            {mutation.label}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        ) : (
          <p className="text-xs text-zinc-500 text-center py-8">暂无配置</p>
        )}
      </div>
    </section>
  );

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
              <Package className="size-5 text-zinc-400" />
            </div>
            <div>
              <h1 aria-label="Environment" className="text-lg text-zinc-100">
                环境资源
              </h1>
              <p className="text-xs text-zinc-500 mt-0.5">
                管理绑定到机器 Agent 的 Skill、MCP 和 Plugin 资源
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              aria-label="Sync catalog"
              disabled={!vm.capabilities.syncCatalog || vm.pendingActionKey === "sync-catalog"}
              onClick={() => void vm.handleSyncCatalog()}
              className="flex items-center gap-2 px-3 py-2 bg-zinc-800 hover:bg-zinc-700 disabled:bg-zinc-800/60 disabled:text-zinc-600 text-zinc-300 rounded-lg text-sm transition-colors"
            >
              <RefreshCw className="size-4" />
              同步目录
            </button>
            <button
              type="button"
              aria-label="Restart bridge"
              disabled={!vm.capabilities.restartBridge || vm.pendingActionKey === "restart-bridge"}
              onClick={() => void vm.handleRestartBridge()}
              className="flex items-center gap-2 px-3 py-2 bg-zinc-800 hover:bg-zinc-700 disabled:bg-zinc-800/60 disabled:text-zinc-600 text-zinc-300 rounded-lg text-sm transition-colors"
            >
              <RefreshCw className="size-4" />
              重启 MCP Bridge
            </button>
            <button
              type="button"
              aria-label="Open marketplace"
              disabled={!vm.capabilities.openMarketplace}
              onClick={vm.openInstallPluginForm}
              className="flex items-center gap-2 px-3 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
            >
              <ExternalLink className="size-4" />
              打开 Marketplace
            </button>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-5xl mx-auto space-y-6">
          {vm.error ? (
            <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-sm text-red-300">
              {vm.error}
            </div>
          ) : null}
          {vm.isLoading ? (
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-zinc-400">
              正在加载环境资源…
            </div>
          ) : null}

          {renderResourceSection(
            "skill",
            "Skills",
            <Puzzle className="size-4 text-violet-400" />,
            "Add skill",
            vm.sections.skills,
          )}
          {renderResourceSection(
            "mcp",
            "MCP Servers",
            <Blocks className="size-4 text-blue-400" />,
            "Add MCP",
            vm.sections.mcps,
          )}
          {renderResourceSection(
            "plugin",
            "Plugins",
            <Package className="size-4 text-emerald-400" />,
            "Add plugin record",
            vm.sections.plugins,
          )}
        </div>
      </div>

      <Dialog.Root modal={false} open={Boolean(vm.skillForm)} onOpenChange={(open) => !open && vm.setSkillFormClosed()}>
        <Dialog.Portal container={dialogContainer}>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
            <Dialog.Title className="text-lg text-zinc-100 mb-1">添加 Skill</Dialog.Title>
            <Dialog.Description className="text-xs text-zinc-500 mb-6">
              选择目标机器和 Agent，然后创建 Skill scaffold
            </Dialog.Description>

            {vm.skillForm ? (
              <form className="space-y-4" onSubmit={(event) => void vm.handleSkillSubmit(event)}>
                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="skill-machine">
                    目标机器
                  </label>
                  <input
                    id="skill-machine"
                    aria-label="Machine ID"
                    value={vm.skillForm.machineId}
                    onChange={(event) => handleMachineSelection("skill", event.target.value)}
                    placeholder={onlineMachines[0]?.id ?? "machine-01"}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                {vm.skillForm.machineId ? (
                  <div>
                    <label className="block text-xs text-zinc-400 mb-2" htmlFor="skill-agent">
                      目标 Agent
                    </label>
                    <select
                      id="skill-agent"
                      value={skillAgentId}
                      onChange={(event) => setSkillAgentId(event.target.value)}
                      className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
                    >
                      {(machineById.get(vm.skillForm.machineId)?.agents ?? []).map((agent) => (
                        <option key={agent.id} value={agent.id}>
                          {agent.name} ({agent.model})
                        </option>
                      ))}
                    </select>
                  </div>
                ) : null}

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="skill-name">
                    名称
                  </label>
                  <input
                    id="skill-name"
                    aria-label="Skill name"
                    type="text"
                    value={vm.skillForm.name}
                    onChange={(event) =>
                      vm.setSkillForm((current) =>
                        current ? { ...current, name: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="skill-description">
                    描述
                  </label>
                  <input
                    id="skill-description"
                    aria-label="Description"
                    type="text"
                    value={vm.skillForm.description}
                    onChange={(event) =>
                      vm.setSkillForm((current) =>
                        current ? { ...current, description: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div className="flex items-center gap-2 mt-6">
                  <button
                    type="submit"
                    aria-label="Create skill"
                    disabled={!vm.capabilities.writeSkills || vm.pendingActionKey === "skill-form"}
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
              </form>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <Dialog.Root modal={false} open={Boolean(vm.mcpForm)} onOpenChange={(open) => !open && vm.setMcpFormClosed()}>
        <Dialog.Portal container={dialogContainer}>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
            <Dialog.Title className="text-lg text-zinc-100 mb-1">
              {vm.mcpForm?.resourceId ? "编辑 MCP Server" : "添加 MCP Server"}
            </Dialog.Title>
            <Dialog.Description className="text-xs text-zinc-500 mb-6">
              为目标机器配置 MCP Server
            </Dialog.Description>

            {vm.mcpForm ? (
              <form className="space-y-4" onSubmit={(event) => void vm.handleMCPSubmit(event)}>
                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="mcp-machine">
                    目标机器
                  </label>
                  <input
                    id="mcp-machine"
                    aria-label="Machine ID"
                    value={vm.mcpForm.machineId}
                    onChange={(event) => handleMachineSelection("mcp", event.target.value)}
                    placeholder={onlineMachines[0]?.id ?? "machine-01"}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                {vm.mcpForm.machineId ? (
                  <div>
                    <label className="block text-xs text-zinc-400 mb-2" htmlFor="mcp-agent">
                      目标 Agent
                    </label>
                    <select
                      id="mcp-agent"
                      value={mcpAgentId}
                      onChange={(event) => setMcpAgentId(event.target.value)}
                      className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
                    >
                      {(machineById.get(vm.mcpForm.machineId)?.agents ?? []).map((agent) => (
                        <option key={agent.id} value={agent.id}>
                          {agent.name} ({agent.model})
                        </option>
                      ))}
                    </select>
                  </div>
                ) : null}

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="mcp-id">
                    Server ID
                  </label>
                  <input
                    id="mcp-id"
                    aria-label="Server ID"
                    type="text"
                    value={vm.mcpForm.resourceId}
                    onChange={(event) =>
                      vm.setMcpForm((current) =>
                        current ? { ...current, resourceId: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="mcp-config">
                    Config JSON
                  </label>
                  <textarea
                    id="mcp-config"
                    aria-label="Config JSON"
                    value={vm.mcpForm.configText}
                    onChange={(event) =>
                      vm.setMcpForm((current) =>
                        current ? { ...current, configText: event.target.value } : current,
                      )
                    }
                    rows={8}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 font-mono placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div className="flex items-center gap-2 mt-6">
                  <button
                    type="submit"
                    aria-label="Save MCP"
                    disabled={!vm.capabilities.writeMcp || vm.pendingActionKey === "mcp-form"}
                    className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
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
              </form>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <Dialog.Root modal={false} open={Boolean(vm.pluginForm)} onOpenChange={(open) => !open && vm.setPluginFormClosed()}>
        <Dialog.Portal container={dialogContainer}>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-md z-50 p-6">
            <Dialog.Title className="text-lg text-zinc-100 mb-1">添加 Plugin</Dialog.Title>
            <Dialog.Description className="text-xs text-zinc-500 mb-6">
              在目标机器上创建或安装 Plugin 记录
            </Dialog.Description>

            {vm.pluginForm ? (
              <form className="space-y-4" onSubmit={(event) => void vm.handlePluginInstallSubmit(event)}>
                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="plugin-machine">
                    目标机器
                  </label>
                  <input
                    id="plugin-machine"
                    aria-label="Machine ID"
                    value={vm.pluginForm.machineId}
                    onChange={(event) => handleMachineSelection("plugin", event.target.value)}
                    placeholder={onlineMachines[0]?.id ?? "machine-01"}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                {vm.pluginForm.machineId ? (
                  <div>
                    <label className="block text-xs text-zinc-400 mb-2" htmlFor="plugin-agent">
                      目标 Agent
                    </label>
                    <select
                      id="plugin-agent"
                      value={pluginAgentId}
                      onChange={(event) => setPluginAgentId(event.target.value)}
                      className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 focus:outline-none focus:border-blue-500"
                    >
                      {(machineById.get(vm.pluginForm.machineId)?.agents ?? []).map((agent) => (
                        <option key={agent.id} value={agent.id}>
                          {agent.name} ({agent.model})
                        </option>
                      ))}
                    </select>
                  </div>
                ) : null}

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="plugin-id">
                    Plugin ID
                  </label>
                  <input
                    id="plugin-id"
                    aria-label="Plugin ID"
                    type="text"
                    value={vm.pluginForm.pluginId}
                    onChange={(event) =>
                      vm.setPluginForm((current) =>
                        current ? { ...current, pluginId: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="plugin-name">
                    Plugin name
                  </label>
                  <input
                    id="plugin-name"
                    aria-label="Plugin name"
                    type="text"
                    value={vm.pluginForm.pluginName}
                    onChange={(event) =>
                      vm.setPluginForm((current) =>
                        current ? { ...current, pluginName: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs text-zinc-400 mb-2" htmlFor="plugin-marketplace">
                    Marketplace path
                  </label>
                  <input
                    id="plugin-marketplace"
                    aria-label="Marketplace path"
                    type="text"
                    value={vm.pluginForm.marketplacePath}
                    onChange={(event) =>
                      vm.setPluginForm((current) =>
                        current ? { ...current, marketplacePath: event.target.value } : current,
                      )
                    }
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 placeholder:text-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div className="flex items-center gap-2 mt-6">
                  <button
                    type="submit"
                    aria-label="Install plugin"
                    disabled={vm.pendingActionKey === "plugin-form"}
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
              </form>
            ) : null}
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  );
}
