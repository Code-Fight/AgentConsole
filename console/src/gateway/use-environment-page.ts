import { type FormEvent, useEffect, useState } from "react";
import { http } from "../common/api/http";
import type {
  EventEnvelope,
  EnvironmentListResponse,
  EnvironmentResource,
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";
import { supportsCapability } from "./capabilities";

interface EnvironmentSections {
  skills: EnvironmentResource[];
  mcps: EnvironmentResource[];
  plugins: EnvironmentResource[];
}

export interface MCPFormState {
  machineId: string;
  resourceId: string;
  configText: string;
}

export interface SkillFormState {
  machineId: string;
  name: string;
  description: string;
}

export interface PluginFormState {
  machineId: string;
  pluginId: string;
  pluginName: string;
  marketplacePath: string;
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

export function useEnvironmentPage() {
  const [sections, setSections] = useState<EnvironmentSections>({
    skills: [],
    mcps: [],
    plugins: [],
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pendingActionKey, setPendingActionKey] = useState<string | null>(null);
  const [refreshNonce, setRefreshNonce] = useState(0);
  const [expandedResourceKey, setExpandedResourceKey] = useState<string | null>(null);
  const [mcpForm, setMcpForm] = useState<MCPFormState | null>(null);
  const [skillForm, setSkillForm] = useState<SkillFormState | null>(null);
  const [pluginForm, setPluginForm] = useState<PluginFormState | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadEnvironment() {
      try {
        const [skillsResponse, mcpsResponse, pluginsResponse] = await Promise.all([
          http<EnvironmentListResponse>("/environment/skills"),
          http<EnvironmentListResponse>("/environment/mcps"),
          http<EnvironmentListResponse>("/environment/plugins"),
        ]);

        if (!cancelled) {
          setSections({
            skills: skillsResponse.items,
            mcps: mcpsResponse.items,
            plugins: pluginsResponse.items,
          });
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setSections({
            skills: [],
            mcps: [],
            plugins: [],
          });
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
  }, [refreshNonce]);

  useEffect(
    () =>
      connectConsoleSocket(undefined, (event) => {
        let envelope: EventEnvelope | null = null;

        try {
          envelope = JSON.parse(event.data) as EventEnvelope;
        } catch {
          return;
        }

        if (envelope.name === "resource.changed") {
          setRefreshNonce((current) => current + 1);
        }
      }),
    [],
  );

  function openCreateMCPForm() {
    setSkillForm(null);
    setPluginForm(null);
    setMcpForm({
      machineId: defaultMachineId(sections),
      resourceId: "",
      configText: '{\n  "command": "npx"\n}',
    });
    setError(null);
  }

  function openCreateSkillForm() {
    setMcpForm(null);
    setPluginForm(null);
    setSkillForm({
      machineId: defaultMachineId(sections),
      name: "",
      description: "",
    });
    setError(null);
  }

  function openInstallPluginForm() {
    setMcpForm(null);
    setSkillForm(null);
    setPluginForm({
      machineId: defaultMachineId(sections),
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
      resourceId: resource.resourceId,
      configText: JSON.stringify(extractMCPConfig(resource), null, 2),
    });
    setError(null);
  }

  function toggleDetails(resource: EnvironmentResource) {
    const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}`;
    setExpandedResourceKey((current) => (current === resourceKey ? null : resourceKey));
  }

  async function handleResourceMutation(
    resource: EnvironmentResource,
    actionKey: string,
    method: "POST" | "DELETE",
    path: string,
    payload?: Record<string, unknown>,
  ) {
    setPendingActionKey(actionKey);
    setError(null);

    try {
      const requestBody = {
        machineId: resource.machineId,
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
    if (!mcpForm) {
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
    if (!skillForm) {
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
    if (!pluginForm) {
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

  return {
    sections,
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
    capabilities: {
      syncCatalog: supportsCapability("environmentSyncCatalog"),
      restartBridge: supportsCapability("environmentRestartBridge"),
      openMarketplace: supportsCapability("environmentOpenMarketplace"),
      mutateResources: supportsCapability("environmentMutateResources"),
      writeMcp: supportsCapability("environmentWriteMcp"),
      writeSkills: supportsCapability("environmentWriteSkills"),
    },
  };
}
