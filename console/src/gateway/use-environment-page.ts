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
    setMcpForm({
      machineId: defaultMachineId(sections),
      resourceId: "",
      configText: '{\n  "command": "npx"\n}',
    });
    setError(null);
  }

  function openEditMCPForm(resource: EnvironmentResource) {
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
  ) {
    setPendingActionKey(actionKey);
    setError(null);

    try {
      await http<void>(path, {
        method,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ machineId: resource.machineId }),
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

  return {
    sections,
    isLoading,
    error,
    pendingActionKey,
    expandedResourceKey,
    mcpForm,
    setMcpForm,
    setMcpFormClosed: () => setMcpForm(null),
    openCreateMCPForm,
    openEditMCPForm,
    toggleDetails,
    handleResourceMutation,
    handleMCPSubmit,
    capabilities: {
      syncCatalog: supportsCapability("environmentSyncCatalog"),
      restartBridge: supportsCapability("environmentRestartBridge"),
      openMarketplace: supportsCapability("environmentOpenMarketplace"),
      mutateResources: supportsCapability("environmentMutateResources"),
      writeMcp: supportsCapability("environmentWriteMcp"),
    },
  };
}
