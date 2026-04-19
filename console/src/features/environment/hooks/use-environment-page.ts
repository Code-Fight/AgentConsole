import { type FormEvent, useEffect, useState, useSyncExternalStore } from "react";
import { http } from "../../../common/api/http";
import type {
  EnvironmentListResponse,
  EnvironmentResource,
  EventEnvelope,
  MachineListResponse,
  MachineSummary,
} from "../../../common/api/types";
import { connectConsoleSocket } from "../../../common/api/ws";
import { supportsCapability, useCapabilities } from "../../../gateway/capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "../../../gateway/gateway-connection-store";

interface EnvironmentSections {
  skills: EnvironmentResource[];
  mcps: EnvironmentResource[];
  plugins: EnvironmentResource[];
}

export interface MCPFormState {
  machineId: string;
  agentId: string;
  resourceId: string;
  configText: string;
}

export interface SkillFormState {
  machineId: string;
  agentId: string;
  name: string;
  description: string;
}

export interface PluginFormState {
  machineId: string;
  agentId: string;
  pluginId: string;
  pluginName: string;
  marketplacePath: string;
}

interface UseEnvironmentPageOptions {
  enabled?: boolean;
}

export function defaultMachineId(sections: EnvironmentSections): string {
  const allResources = [...sections.skills, ...sections.mcps, ...sections.plugins];
  for (const resource of allResources) {
    if (resource.machineId) {
      return resource.machineId;
    }
  }
  return "";
}

function defaultAgentId(machines: MachineSummary[], machineID: string): string {
  if (!machineID) {
    return "";
  }
  const machine = machines.find((item) => item.id === machineID);
  return machine?.agents?.[0]?.agentId?.trim() ?? "";
}

export function extractMCPConfig(resource: EnvironmentResource): Record<string, unknown> {
  const details = resource.details ?? {};
  const config = details.config;
  if (config && typeof config === "object" && !Array.isArray(config)) {
    return config as Record<string, unknown>;
  }

  const fallback: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(details)) {
    if (["config", "status", "error", "enabled", "needsAuth"].includes(key)) {
      continue;
    }
    fallback[key] = value;
  }
  return fallback;
}

export function useEnvironmentPage(options?: UseEnvironmentPageOptions) {
  const enabled = options?.enabled ?? true;
  useCapabilities(enabled);
  const connectionIdentity = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionIdentity,
    getGatewayConnectionIdentity,
  );
  const [sections, setSections] = useState<EnvironmentSections>({
    skills: [],
    mcps: [],
    plugins: [],
  });
  const [machines, setMachines] = useState<MachineSummary[]>([]);
  const [isLoading, setIsLoading] = useState(enabled);
  const [error, setError] = useState<string | null>(null);
  const [pendingActionKey, setPendingActionKey] = useState<string | null>(null);
  const [refreshNonce, setRefreshNonce] = useState(0);
  const [expandedResourceKey, setExpandedResourceKey] = useState<string | null>(null);
  const [mcpForm, setMcpForm] = useState<MCPFormState | null>(null);
  const [skillForm, setSkillForm] = useState<SkillFormState | null>(null);
  const [pluginForm, setPluginForm] = useState<PluginFormState | null>(null);

  useEffect(() => {
    setPendingActionKey(null);
    setExpandedResourceKey(null);
    setMcpForm(null);
    setSkillForm(null);
    setPluginForm(null);
    setError(null);
  }, [connectionIdentity]);

  useEffect(() => {
    if (!enabled) {
      setSections({ skills: [], mcps: [], plugins: [] });
      setMachines([]);
      setIsLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;
    setIsLoading(true);

    async function loadEnvironment() {
      try {
        const [skillsResponse, mcpsResponse, pluginsResponse, machinesResponse] = await Promise.all([
          http<EnvironmentListResponse>("/environment/skills"),
          http<EnvironmentListResponse>("/environment/mcps"),
          http<EnvironmentListResponse>("/environment/plugins"),
          http<MachineListResponse>("/machines").catch(() => ({ items: [] })),
        ]);

        if (!cancelled) {
          setSections({
            skills: skillsResponse.items,
            mcps: mcpsResponse.items,
            plugins: pluginsResponse.items,
          });
          setMachines(machinesResponse.items ?? []);
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setSections({
            skills: [],
            mcps: [],
            plugins: [],
          });
          setMachines([]);
          setError("Unable to load environment resources.");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadEnvironment();

    return () => {
      cancelled = true;
    };
  }, [enabled, connectionIdentity, refreshNonce]);

  useEffect(() => {
    if (!enabled) {
      return undefined;
    }

    return connectConsoleSocket(undefined, (event) => {
      let envelope: EventEnvelope | null = null;

      try {
        envelope = JSON.parse(event.data) as EventEnvelope;
      } catch {
        return;
      }

      if (envelope.name === "resource.changed" || envelope.name === "machine.updated") {
        setRefreshNonce((current) => current + 1);
      }
    });
  }, [enabled, connectionIdentity]);

  function openCreateMCPForm() {
    setSkillForm(null);
    setPluginForm(null);
    const machineId = defaultMachineId(sections);
    setMcpForm({
      machineId,
      agentId: defaultAgentId(machines, machineId),
      resourceId: "",
      configText: '{\n  "command": "npx"\n}',
    });
    setError(null);
  }

  function openCreateSkillForm() {
    setMcpForm(null);
    setPluginForm(null);
    const machineId = defaultMachineId(sections);
    setSkillForm({
      machineId,
      agentId: defaultAgentId(machines, machineId),
      name: "",
      description: "",
    });
    setError(null);
  }

  function openInstallPluginForm() {
    setMcpForm(null);
    setSkillForm(null);
    const machineId = defaultMachineId(sections);
    setPluginForm({
      machineId,
      agentId: defaultAgentId(machines, machineId),
      pluginId: "",
      pluginName: "",
      marketplacePath: "",
    });
    setError(null);
  }

  function openEditMCPForm(resource: EnvironmentResource) {
    setSkillForm(null);
    setPluginForm(null);
    setMcpForm({
      machineId: resource.machineId,
      agentId: resolveResourceAgentID(resource),
      resourceId: resource.resourceId,
      configText: JSON.stringify(extractMCPConfig(resource), null, 2),
    });
    setError(null);
  }

  function toggleDetails(resource: EnvironmentResource) {
    const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}`;
    setExpandedResourceKey((current) => (current === resourceKey ? null : resourceKey));
  }

  function resolveResourceAgentID(resource: EnvironmentResource): string {
    if (resource.agentId?.trim()) {
      return resource.agentId.trim();
    }

    const machine = machines.find((item) => item.id === resource.machineId);
    return machine?.agents?.[0]?.agentId?.trim() ?? "";
  }

  async function handleResourceMutation(
    resource: EnvironmentResource,
    actionKey: string,
    method: "POST" | "DELETE",
    path: string,
    payload?: Record<string, unknown>,
  ) {
    if (!enabled) {
      return;
    }
    setPendingActionKey(actionKey);
    setError(null);

    try {
      const requestBody = {
        machineId: resource.machineId,
        agentId: resolveResourceAgentID(resource),
        ...(payload ?? {}),
      };
      await http<void>(path, {
        method,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to update environment resources.");
    } finally {
      setPendingActionKey(null);
    }
  }

  async function handleMCPSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!enabled || !mcpForm) {
      return;
    }

    let config: Record<string, unknown>;
    try {
      const parsed = JSON.parse(mcpForm.configText);
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        throw new Error("Config JSON must be an object.");
      }
      config = parsed as Record<string, unknown>;
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "Invalid MCP config JSON.");
      return;
    }

    setError(null);
    setPendingActionKey("mcp-form");
    try {
      await http<void>("/environment/mcps", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          machineId: mcpForm.machineId.trim(),
          agentId: mcpForm.agentId.trim(),
          resourceId: mcpForm.resourceId.trim(),
          config,
        }),
      });
      setMcpForm(null);
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to save MCP configuration.");
    } finally {
      setPendingActionKey(null);
    }
  }

  async function handleSkillSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!enabled || !skillForm) {
      return;
    }
    if (!skillForm.name.trim()) {
      setError("Skill name is required.");
      return;
    }

    setError(null);
    setPendingActionKey("skill-form");
    try {
      await http<void>("/environment/skills", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          machineId: skillForm.machineId.trim(),
          agentId: skillForm.agentId.trim(),
          name: skillForm.name.trim(),
          description: skillForm.description.trim(),
        }),
      });
      setSkillForm(null);
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to create skill scaffold.");
    } finally {
      setPendingActionKey(null);
    }
  }

  async function handlePluginInstallSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!enabled || !pluginForm) {
      return;
    }
    if (!pluginForm.pluginName.trim()) {
      setError("Plugin name is required.");
      return;
    }
    if (!pluginForm.marketplacePath.trim()) {
      setError("Marketplace path is required.");
      return;
    }

    const pluginName = pluginForm.pluginName.trim();
    const pluginId = pluginForm.pluginId.trim() || pluginName;

    setError(null);
    setPendingActionKey("plugin-form");
    try {
      await http<void>("/environment/plugins/install", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          machineId: pluginForm.machineId.trim(),
          agentId: pluginForm.agentId.trim(),
          pluginId,
          pluginName,
          marketplacePath: pluginForm.marketplacePath.trim(),
        }),
      });
      setPluginForm(null);
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to install plugin.");
    } finally {
      setPendingActionKey(null);
    }
  }

  async function handleSyncCatalog() {
    if (!enabled) {
      return;
    }
    setError(null);
    setPendingActionKey("sync-catalog");
    try {
      await http<void>("/environment/sync", {
        method: "POST",
      });
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to sync environment catalog.");
    } finally {
      setPendingActionKey(null);
    }
  }

  async function handleRestartBridge() {
    if (!enabled) {
      return;
    }
    setError(null);
    setPendingActionKey("restart-bridge");
    try {
      await http<void>("/environment/mcps/restart-bridge", {
        method: "POST",
      });
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to restart MCP bridge.");
    } finally {
      setPendingActionKey(null);
    }
  }

  return {
    sections,
    machines,
    isLoading,
    error,
    pendingActionKey,
    expandedResourceKey,
    mcpForm,
    skillForm,
    pluginForm,
    setMcpForm,
    setSkillForm,
    setPluginForm,
    setMcpFormClosed: () => setMcpForm(null),
    setSkillFormClosed: () => setSkillForm(null),
    setPluginFormClosed: () => setPluginForm(null),
    openCreateMCPForm,
    openCreateSkillForm,
    openInstallPluginForm,
    openEditMCPForm,
    toggleDetails,
    handleResourceMutation,
    handleMCPSubmit,
    handleSkillSubmit,
    handlePluginInstallSubmit,
    handleSyncCatalog,
    handleRestartBridge,
    capabilities: {
      syncCatalog: enabled && supportsCapability("environmentSyncCatalog"),
      restartBridge: enabled && supportsCapability("environmentRestartBridge"),
      openMarketplace: enabled && supportsCapability("environmentOpenMarketplace"),
      mutateResources: enabled && supportsCapability("environmentMutateResources"),
      writeMcp: enabled && supportsCapability("environmentWriteMcp"),
      writeSkills: enabled && supportsCapability("environmentWriteSkills"),
    },
  };
}
