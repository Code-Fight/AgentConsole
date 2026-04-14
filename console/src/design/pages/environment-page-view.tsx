import { type FormEvent, type ReactNode } from "react";
import type { EnvironmentResource } from "../../common/api/types";
import type { MCPFormState, PluginFormState, SkillFormState } from "../../gateway/use-environment-page";

interface EnvironmentSections {
  skills: EnvironmentResource[];
  mcps: EnvironmentResource[];
  plugins: EnvironmentResource[];
}

interface EnvironmentPageViewProps {
  sections: EnvironmentSections;
  isLoading: boolean;
  error: string | null;
  pendingActionKey: string | null;
  expandedResourceKey: string | null;
  mcpForm: MCPFormState | null;
  skillForm: SkillFormState | null;
  pluginForm: PluginFormState | null;
  capabilities: {
    syncCatalog: boolean;
    restartBridge: boolean;
    openMarketplace: boolean;
    mutateResources: boolean;
    writeMcp: boolean;
    writeSkills: boolean;
  };
  setMcpForm: (updater: MCPFormState | null | ((current: MCPFormState | null) => MCPFormState | null)) => void;
  setSkillForm: (
    updater: SkillFormState | null | ((current: SkillFormState | null) => SkillFormState | null),
  ) => void;
  setPluginForm: (
    updater: PluginFormState | null | ((current: PluginFormState | null) => PluginFormState | null),
  ) => void;
  onCloseMcpForm: () => void;
  onCloseSkillForm: () => void;
  onClosePluginForm: () => void;
  onOpenCreateMcpForm: () => void;
  onOpenCreateSkillForm: () => void;
  onOpenInstallPluginForm: () => void;
  onOpenEditMcpForm: (resource: EnvironmentResource) => void;
  onToggleDetails: (resource: EnvironmentResource) => void;
  onResourceMutation: (
    resource: EnvironmentResource,
    actionKey: string,
    method: "POST" | "DELETE",
    path: string,
    payload?: Record<string, unknown>,
  ) => void;
  onMcpSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onSkillSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onPluginInstallSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

interface ResourceMutation {
  label: string;
  method: "POST" | "DELETE";
  path: string;
  payload?: Record<string, unknown>;
}

function formatResourceStatus(resource: EnvironmentResource): string {
  if (resource.status === "auth_required") {
    return "Auth required";
  }

  switch (resource.status) {
    case "enabled":
      return "Enabled";
    case "disabled":
      return "Disabled";
    case "unknown":
      return "Not installed";
    case "error":
      return "Error";
    default:
      return resource.status;
  }
}

function hasDetails(resource: EnvironmentResource): boolean {
  return Boolean(resource.details && Object.keys(resource.details).length > 0);
}

function buildResourceMutations(resource: EnvironmentResource): ResourceMutation[] {
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
    return <p className="detail-empty">No detail payload reported.</p>;
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

function CapabilityCard(props: { title: string; detail: string; action: string; connected: boolean }) {
  return (
    <article className="resource-card-main">
      <h2>{props.title}</h2>
      <p>{props.detail}</p>
      <div className="settings-actions">
        <button type="button" disabled={!props.connected} aria-label={props.action}>
          {props.action}
        </button>
        {!props.connected ? <span className="meta-pill">Not connected</span> : null}
      </div>
    </article>
  );
}

function renderEnvironmentSection(
  title: string,
  items: EnvironmentResource[],
  emptyLabel: string,
  props: EnvironmentPageViewProps,
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
            const expanded = props.expandedResourceKey === resourceKey;

            return (
              <article key={resourceKey} className="resource-card">
                <div className="resource-card-main">
                  <h3>{item.displayName || item.resourceId}</h3>
                  <p>{item.machineId}</p>
                  {expanded ? renderDetailsPanel(item) : null}
                </div>
                <div className="resource-card-meta">
                  <span className={`status-badge status-${item.status}`}>{formatResourceStatus(item)}</span>
                  {item.restartRequired ? <span className="meta-pill">Restart required</span> : null}
                  {hasDetails(item) ? (
                    <button
                      type="button"
                      aria-label={expanded ? "Hide details" : "View details"}
                      onClick={() => props.onToggleDetails(item)}
                    >
                      {expanded ? "Hide details" : "View details"}
                    </button>
                  ) : null}
                  {item.kind === "mcp" ? (
                    <button
                      type="button"
                      aria-label="Edit"
                      disabled={!props.capabilities.writeMcp}
                      onClick={() => props.onOpenEditMcpForm(item)}
                    >
                      Edit
                    </button>
                  ) : null}
                  {item.kind === "skill" ? (
                    <button
                      type="button"
                      aria-label="Delete skill"
                      disabled={
                        !props.capabilities.writeSkills ||
                        props.pendingActionKey === `${resourceKey}:Delete skill`
                      }
                      onClick={() =>
                        props.onResourceMutation(
                          item,
                          `${resourceKey}:Delete skill`,
                          "DELETE",
                          `/environment/skills/${encodeURIComponent(item.resourceId)}`,
                        )
                      }
                    >
                      Delete skill
                    </button>
                  ) : null}
                  {mutations.map((mutation) => {
                    const actionKey = `${resourceKey}:${mutation.label}`;
                    return (
                      <button
                        key={mutation.label}
                        type="button"
                        aria-label={mutation.label}
                        disabled={!props.capabilities.mutateResources || props.pendingActionKey === actionKey}
                        onClick={() =>
                          props.onResourceMutation(
                            item,
                            actionKey,
                            mutation.method,
                            mutation.path,
                            mutation.payload,
                          )
                        }
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

export function EnvironmentPageView(props: EnvironmentPageViewProps) {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Environment</h1>
        <p>Gateway-backed environment controls on the design surface, with unsupported actions left explicitly disconnected.</p>
      </header>

      <div className="environment-grid">
        <CapabilityCard
          title="Skills"
          detail="Gateway can mutate reported skill state, but catalog sync is not connected in this console."
          action="Sync catalog"
          connected={props.capabilities.syncCatalog}
        />
        <CapabilityCard
          title="MCP"
          detail="MCP resources are editable, while bridge lifecycle controls remain outside the connected surface."
          action="Restart bridge"
          connected={props.capabilities.restartBridge}
        />
        <CapabilityCard
          title="Plugins"
          detail="Plugin install and uninstall stay Gateway-backed; marketplace browsing is not wired here."
          action="Open marketplace"
          connected={props.capabilities.openMarketplace}
        />
      </div>

      {props.mcpForm ? (
        <form className="config-form" onSubmit={props.onMcpSubmit}>
          <div className="config-form-heading">
            <div>
              <span className="page-kicker">MCP editor</span>
              <h2>{props.mcpForm.resourceId ? "Edit MCP" : "Add MCP"}</h2>
            </div>
            <button type="button" aria-label="Cancel" onClick={props.onCloseMcpForm}>
              Cancel
            </button>
          </div>
          <label>
            <span>Machine ID</span>
            <input
              aria-label="Machine ID"
              value={props.mcpForm.machineId}
              onChange={(event) =>
                props.setMcpForm((current) =>
                  current ? { ...current, machineId: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Server ID</span>
            <input
              aria-label="Server ID"
              value={props.mcpForm.resourceId}
              onChange={(event) =>
                props.setMcpForm((current) =>
                  current ? { ...current, resourceId: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Config JSON</span>
            <textarea
              aria-label="Config JSON"
              rows={10}
              value={props.mcpForm.configText}
              onChange={(event) =>
                props.setMcpForm((current) =>
                  current ? { ...current, configText: event.target.value } : current,
                )
              }
            />
          </label>
          <button
            type="submit"
            aria-label="Save MCP"
            disabled={!props.capabilities.writeMcp || props.pendingActionKey === "mcp-form"}
          >
            Save MCP
          </button>
        </form>
      ) : null}

      {props.skillForm ? (
        <form className="config-form" onSubmit={props.onSkillSubmit}>
          <div className="config-form-heading">
            <div>
              <span className="page-kicker">Skill scaffold</span>
              <h2>Add skill</h2>
            </div>
            <button type="button" aria-label="Cancel" onClick={props.onCloseSkillForm}>
              Cancel
            </button>
          </div>
          <label>
            <span>Machine ID</span>
            <input
              aria-label="Machine ID"
              value={props.skillForm.machineId}
              onChange={(event) =>
                props.setSkillForm((current) =>
                  current ? { ...current, machineId: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Skill name</span>
            <input
              aria-label="Skill name"
              value={props.skillForm.name}
              onChange={(event) =>
                props.setSkillForm((current) =>
                  current ? { ...current, name: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Description</span>
            <textarea
              aria-label="Description"
              rows={5}
              value={props.skillForm.description}
              onChange={(event) =>
                props.setSkillForm((current) =>
                  current ? { ...current, description: event.target.value } : current,
                )
              }
            />
          </label>
          <button
            type="submit"
            aria-label="Create skill"
            disabled={!props.capabilities.writeSkills || props.pendingActionKey === "skill-form"}
          >
            Create skill
          </button>
        </form>
      ) : null}

      {props.pluginForm ? (
        <form className="config-form" onSubmit={props.onPluginInstallSubmit}>
          <div className="config-form-heading">
            <div>
              <span className="page-kicker">Plugin install</span>
              <h2>Add plugin record</h2>
            </div>
            <button type="button" aria-label="Cancel" onClick={props.onClosePluginForm}>
              Cancel
            </button>
          </div>
          <label>
            <span>Machine ID</span>
            <input
              aria-label="Machine ID"
              value={props.pluginForm.machineId}
              onChange={(event) =>
                props.setPluginForm((current) =>
                  current ? { ...current, machineId: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Plugin name</span>
            <input
              aria-label="Plugin name"
              value={props.pluginForm.pluginName}
              onChange={(event) =>
                props.setPluginForm((current) =>
                  current ? { ...current, pluginName: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Plugin ID (optional)</span>
            <input
              aria-label="Plugin ID"
              value={props.pluginForm.pluginId}
              onChange={(event) =>
                props.setPluginForm((current) =>
                  current ? { ...current, pluginId: event.target.value } : current,
                )
              }
            />
          </label>
          <label>
            <span>Marketplace path</span>
            <input
              aria-label="Marketplace path"
              value={props.pluginForm.marketplacePath}
              onChange={(event) =>
                props.setPluginForm((current) =>
                  current ? { ...current, marketplacePath: event.target.value } : current,
                )
              }
            />
          </label>
          <button
            type="submit"
            aria-label="Install plugin"
            disabled={!props.capabilities.mutateResources || props.pendingActionKey === "plugin-form"}
          >
            Install plugin
          </button>
        </form>
      ) : null}

      {props.isLoading ? <p>Loading environment…</p> : null}
      {props.error ? <p>{props.error}</p> : null}

      {!props.isLoading && !props.error ? (
        <div className="environment-grid">
          {renderEnvironmentSection(
            "Skills",
            props.sections.skills,
            "No skills reported.",
            props,
            <div className="settings-actions">
              <button
                type="button"
                aria-label="Add skill"
                disabled={!props.capabilities.writeSkills}
                onClick={props.onOpenCreateSkillForm}
              >
                Add skill
              </button>
            </div>,
          )}
          {renderEnvironmentSection(
            "MCPs",
            props.sections.mcps,
            "No MCP servers reported.",
            props,
            <button
              type="button"
              aria-label="Add MCP"
              disabled={!props.capabilities.writeMcp}
              onClick={props.onOpenCreateMcpForm}
            >
              Add MCP
            </button>,
          )}
          {renderEnvironmentSection(
            "Plugins",
            props.sections.plugins,
            "No plugins reported.",
            props,
            <div className="settings-actions">
              <button
                type="button"
                aria-label="Add plugin record"
                disabled={!props.capabilities.mutateResources}
                onClick={props.onOpenInstallPluginForm}
              >
                Add plugin record
              </button>
            </div>,
          )}
        </div>
      ) : null}
    </section>
  );
}
