import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type {
  EnvironmentListResponse,
  EnvironmentResource
} from "../common/api/types";

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
              </div>
            </article>
          ))}
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
  }, []);

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
          {renderEnvironmentSection("Skills", sections.skills, "No skills reported.")}
          {renderEnvironmentSection("MCPs", sections.mcps, "No MCP servers reported.")}
          {renderEnvironmentSection("Plugins", sections.plugins, "No plugins reported.")}
        </div>
      ) : null}
    </section>
  );
}
