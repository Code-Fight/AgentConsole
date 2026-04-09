import { useEffect, useState } from "react";
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

function formatResourceStatus(resource: EnvironmentResource): string {
  if (resource.status === "auth_required") {
    return "Auth required";
  }

  return resource.status.charAt(0).toUpperCase() + resource.status.slice(1);
}

function renderEnvironmentSection(
  title: string,
  items: EnvironmentResource[],
  emptyLabel: string,
  pendingResourceKey: string | null,
  onResourceAction: (resource: EnvironmentResource) => void,
) {
  return (
    <section className="environment-section" aria-label={title}>
      <div className="section-heading">
        <h2>{title}</h2>
        <span>{items.length}</span>
      </div>
      {items.length === 0 ? <p>{emptyLabel}</p> : null}
      {items.length > 0 ? (
        <div className="resource-list">
          {items.map((item) => (
            <article key={`${item.kind}:${item.machineId}:${item.resourceId}`} className="resource-card">
              <div className="resource-card-main">
                <h3>{item.displayName || item.resourceId}</h3>
                <p>{item.machineId}</p>
              </div>
              <div className="resource-card-meta">
                <span className={`status-badge status-${item.status}`}>
                  {formatResourceStatus(item)}
                </span>
                {item.restartRequired ? <span className="meta-pill">Restart required</span> : null}
                {renderResourceAction(item, pendingResourceKey, onResourceAction)}
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function resourceActionLabel(resource: EnvironmentResource): string | null {
  if (resource.kind === "skill") {
    return resource.status === "enabled" ? "Disable" : "Enable";
  }

  if (resource.kind === "plugin") {
    return "Uninstall";
  }

  return null;
}

function renderResourceAction(
  resource: EnvironmentResource,
  pendingResourceKey: string | null,
  onResourceAction: (resource: EnvironmentResource) => void,
) {
  const label = resourceActionLabel(resource);
  if (!label) {
    return null;
  }

  const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}`;

  return (
    <button
      type="button"
      disabled={pendingResourceKey === resourceKey}
      onClick={() => onResourceAction(resource)}
    >
      {label}
    </button>
  );
}

function buildResourceMutationPath(resource: EnvironmentResource): { method: "POST" | "DELETE"; path: string } | null {
  if (resource.kind === "skill") {
    const action = resource.status === "enabled" ? "disable" : "enable";
    return {
      method: "POST",
      path: `/environment/skills/${encodeURIComponent(resource.resourceId)}/${action}`
    };
  }

  if (resource.kind === "plugin") {
    return {
      method: "DELETE",
      path: `/environment/plugins/${encodeURIComponent(resource.resourceId)}`
    };
  }

  return null;
}

export function EnvironmentPage() {
  const [sections, setSections] = useState<EnvironmentSections>({
    skills: [],
    mcps: [],
    plugins: []
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pendingResourceKey, setPendingResourceKey] = useState<string | null>(null);
  const [refreshNonce, setRefreshNonce] = useState(0);

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

  async function handleResourceAction(resource: EnvironmentResource) {
    const mutation = buildResourceMutationPath(resource);
    if (!mutation) {
      return;
    }

    const resourceKey = `${resource.kind}:${resource.machineId}:${resource.resourceId}`;
    setPendingResourceKey(resourceKey);
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
      setPendingResourceKey(null);
    }
  }

  return (
    <section className="page">
      <header className="page-header">
        <h1>Environment</h1>
        <p>Skills, MCP servers, and plugins reported by connected machines.</p>
      </header>

      {isLoading ? <p>Loading environment…</p> : null}
      {error ? <p>{error}</p> : null}

      {!isLoading && !error ? (
        <div className="environment-grid">
          {renderEnvironmentSection("Skills", sections.skills, "No skills reported.", pendingResourceKey, handleResourceAction)}
          {renderEnvironmentSection("MCPs", sections.mcps, "No MCP servers reported.", pendingResourceKey, handleResourceAction)}
          {renderEnvironmentSection("Plugins", sections.plugins, "No plugins reported.", pendingResourceKey, handleResourceAction)}
        </div>
      ) : null}
    </section>
  );
}
