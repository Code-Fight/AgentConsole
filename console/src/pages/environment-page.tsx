import { type FormEvent, type ReactNode, useEffect, useState } from "react";
import { http } from "../common/api/http";
import type {
  EventEnvelope,
  EnvironmentListResponse,
  EnvironmentResource
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

interface EnvironmentSections {
  skills: EnvironmentResource[];
  mcps: EnvironmentResource[];
  plugins: EnvironmentResource[];
}

interface MCPFormState {
  machineId: string;
  resourceId: string;
  configText: string;
}

type ResourceMutation = {
  label: string;
  method: "POST" | "DELETE";
  path: string;
};

function formatResourceStatus(resource: EnvironmentResource): string {
  if (resource.status === "auth_required") {
    return "Auth required";
  }

  return resource.status.charAt(0).toUpperCase() + resource.status.slice(1);
}

function defaultMachineId(sections: EnvironmentSections): string {
  const allResources = [...sections.skills, ...sections.mcps, ...sections.plugins];
  for (const resource of allResources) {
    if (resource.machineId) {
      return resource.machineId;
    }
  }
  return "";
}

function hasDetails(resource: EnvironmentResource): boolean {
  return Boolean(resource.details && Object.keys(resource.details).length > 0);
}

function buildResourceMutations(resource: EnvironmentResource): ResourceMutation[] {
  if (resource.kind === "skill") {
    const action = resource.status === "enabled" ? "disable" : "enable";
    return [{
      label: resource.status === "enabled" ? "Disable" : "Enable",
      method: "POST",
      path: `/environment/skills/${encodeURIComponent(resource.resourceId)}/${action}`
    }];
  }

  if (resource.kind === "mcp") {
    const action = resource.status === "enabled" ? "disable" : "enable";
    return [
      {
        label: resource.status === "enabled" ? "Disable" : "Enable",
        method: "POST",
        path: `/environment/mcps/${encodeURIComponent(resource.resourceId)}/${action}`
      },
      {
        label: "Delete",
        method: "DELETE",
        path: `/environment/mcps/${encodeURIComponent(resource.resourceId)}`
      }
    ];
  }

  if (resource.kind === "plugin") {
    if (resource.status === "unknown") {
      return [{
        label: "Install",
        method: "POST",
        path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}/install`
      }];
    }

    const action = resource.status === "enabled" ? "disable" : "enable";
    return [
      {
        label: resource.status === "enabled" ? "Disable" : "Enable",
        method: "POST",
        path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}/${action}`
      },
      {
        label: "Uninstall",
        method: "DELETE",
        path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}`
      }
    ];
  }

  return [];
}

function extractMCPConfig(resource: EnvironmentResource): Record<string, unknown> {
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

function formatDetailLabel(key: string): string {
  return key
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/_/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function renderDetailValue(value: unknown) {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <span>Empty</span>;
    }
    return (
      <div className="detail-pill-list">
        {value.map((item, index) => (
          <span key={`${String(item)}-${index}`} className="meta-pill">
            {typeof item === "string" ? item : JSON.stringify(item)}
          </span>
        ))}
      </div>
    );
  }

  if (value && typeof value === "object") {
    return <pre className="detail-code">{JSON.stringify(value, null, 2)}</pre>;
  }

  if (typeof value === "boolean") {
    return <span>{value ? "Yes" : "No"}</span>;
  }

  if (value === null || value === undefined || value === "") {
    return <span>Empty</span>;
  }

  return <span>{String(value)}</span>;
}

function renderDetailsPanel(resource: EnvironmentResource) {
  if (!hasDetails(resource)) {
    return <p className="detail-empty">No details reported.</p>;
  }

  const details = resource.details ?? {};
  const entries = Object.entries(details);

  return (
    <div className="resource-details">
      {entries.map(([key, value]) => (
        <div key={key} className="resource-detail-row">
          <strong>{formatDetailLabel(key)}</strong>
          {renderDetailValue(value)}
        </div>
      ))}
    </div>
  );
}

function renderEnvironmentSection(
  title: string,
  items: EnvironmentResource[],
  emptyLabel: string,
  pendingActionKey: string | null,
  expandedResourceKey: string | null,
  onToggleDetails: (resource: EnvironmentResource) => void,
  onResourceMutation: (resource: EnvironmentResource, mutation: ResourceMutation) => void,
  onEditMCP: (resource: EnvironmentResource) => void,
  headerAction?: ReactNode,
) {
  return (
    <section className="environment-section" aria-label={title}>
      <div className="section-heading">
        <div className="section-heading-main">
          <h2>{title}</h2>
          <span>{items.length}</span>
        </div>
        {headerAction}
      </div>
      {items.length === 0 ? <p>{emptyLabel}</p> : null}
      {items.length > 0 ? (
        <div className="resource-list">
          {items.map((item) => {
            const resourceKey = `${item.kind}:${item.machineId}:${item.resourceId}`;
            const mutations = buildResourceMutations(item);
            const expanded = expandedResourceKey === resourceKey;

            return (
              <article key={resourceKey} className="resource-card">
                <div className="resource-card-main">
                  <h3>{item.displayName || item.resourceId}</h3>
                  <p>{item.machineId}</p>
                  {expanded ? renderDetailsPanel(item) : null}
                </div>
                <div className="resource-card-meta">
                  <span className={`status-badge status-${item.status}`}>
                    {formatResourceStatus(item)}
                  </span>
                  {item.restartRequired ? <span className="meta-pill">Restart required</span> : null}
                  {hasDetails(item) ? (
                    <button type="button" onClick={() => onToggleDetails(item)}>
                      {expanded ? "Hide details" : "View details"}
                    </button>
                  ) : null}
                  {item.kind === "mcp" ? (
                    <button type="button" onClick={() => onEditMCP(item)}>
                      Edit
                    </button>
                  ) : null}
                  {mutations.map((mutation) => {
                    const actionKey = `${resourceKey}:${mutation.label}`;
                    return (
                      <button
                        key={mutation.label}
                        type="button"
                        disabled={pendingActionKey === actionKey}
                        onClick={() => onResourceMutation(item, mutation)}
                      >
                        {mutation.label}
                      </button>
                    );
                  })}
                </div>
              </article>
            );
          })}
        </div>
      ) : null}
    </section>
  );
}

export function EnvironmentPage() {
  const [sections, setSections] = useState<EnvironmentSections>({
    skills: [],
    mcps: [],
    plugins: []
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
          http<EnvironmentListResponse>("/environment/plugins")
        ]);

        if (!cancelled) {
          setSections({
            skills: skillsResponse.items,
            mcps: mcpsResponse.items,
            plugins: pluginsResponse.items
          });
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setSections({
            skills: [],
            mcps: [],
            plugins: []
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

  useEffect(() => connectConsoleSocket(undefined, (event) => {
    let envelope: EventEnvelope | null = null;

    try {
      envelope = JSON.parse(event.data) as EventEnvelope;
    } catch {
      return;
    }

    if (envelope.name === "resource.changed") {
      setRefreshNonce((current) => current + 1);
    }
  }), []);

  function openCreateMCPForm() {
    setMcpForm({
      machineId: defaultMachineId(sections),
      resourceId: "",
      configText: "{\n  \"command\": \"npx\"\n}"
    });
    setError(null);
  }

  function openEditMCPForm(resource: EnvironmentResource) {
    setMcpForm({
      machineId: resource.machineId,
      resourceId: resource.resourceId,
      configText: JSON.stringify(extractMCPConfig(resource), null, 2)
    });
    setError(null);
  }

  function toggleDetails(resource: EnvironmentResource) {
    const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}`;
    setExpandedResourceKey((current) => current === resourceKey ? null : resourceKey);
  }

  async function handleResourceMutation(resource: EnvironmentResource, mutation: ResourceMutation) {
    const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}:${mutation.label}`;
    setPendingActionKey(resourceKey);
    setError(null);

    try {
      await http<void>(mutation.path, {
        method: mutation.method,
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ machineId: resource.machineId })
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
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          machineId: mcpForm.machineId.trim(),
          resourceId: mcpForm.resourceId.trim(),
          config
        })
      });
      setMcpForm(null);
      setRefreshNonce((current) => current + 1);
    } catch {
      setError("Unable to save MCP configuration.");
    } finally {
      setPendingActionKey(null);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <h1>Environment</h1>
        <p>Skills, MCP servers, and plugins reported by connected machines.</p>
      </header>

      {mcpForm ? (
        <form className="config-form" onSubmit={(event) => void handleMCPSubmit(event)}>
          <div className="config-form-heading">
            <h2>{mcpForm.resourceId ? "Edit MCP" : "Add MCP"}</h2>
            <button type="button" onClick={() => setMcpForm(null)}>
              Cancel
            </button>
          </div>
          <label>
            <span>Machine ID</span>
            <input
              value={mcpForm.machineId}
              onChange={(event) => setMcpForm((current) => current ? { ...current, machineId: event.target.value } : current)}
            />
          </label>
          <label>
            <span>Server ID</span>
            <input
              value={mcpForm.resourceId}
              onChange={(event) => setMcpForm((current) => current ? { ...current, resourceId: event.target.value } : current)}
            />
          </label>
          <label>
            <span>Config JSON</span>
            <textarea
              rows={10}
              value={mcpForm.configText}
              onChange={(event) => setMcpForm((current) => current ? { ...current, configText: event.target.value } : current)}
            />
          </label>
          <button type="submit" disabled={pendingActionKey === "mcp-form"}>
            Save MCP
          </button>
        </form>
      ) : null}

      {isLoading ? <p>Loading environment…</p> : null}
      {error ? <p>{error}</p> : null}

      {!isLoading && !error ? (
        <div className="environment-grid">
          {renderEnvironmentSection("Skills", sections.skills, "No skills reported.", pendingActionKey, expandedResourceKey, toggleDetails, handleResourceMutation, openEditMCPForm)}
          {renderEnvironmentSection(
            "MCPs",
            sections.mcps,
            "No MCP servers reported.",
            pendingActionKey,
            expandedResourceKey,
            toggleDetails,
            handleResourceMutation,
            openEditMCPForm,
            <button type="button" onClick={openCreateMCPForm}>Add MCP</button>
          )}
          {renderEnvironmentSection("Plugins", sections.plugins, "No plugins reported.", pendingActionKey, expandedResourceKey, toggleDetails, handleResourceMutation, openEditMCPForm)}
        </div>
      ) : null}
    </section>
  );
}
