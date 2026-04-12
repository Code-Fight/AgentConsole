import { NavLink } from "react-router-dom";

export interface ThreadHubItem {
  id: string;
  title: string;
  machineLabel: string;
  status: "active" | "idle" | "offline";
}

interface ThreadHubPanelProps {
  threads?: ThreadHubItem[];
  activeThreadId?: string;
  onNavigate?: () => void;
}

export const THREAD_HUB_DEMO_ITEMS: ThreadHubItem[] = [
  { id: "thread-alpha", title: "Gateway rollout checks", machineLabel: "machine-01", status: "active" },
  { id: "thread-beta", title: "Session sandbox diagnostics", machineLabel: "machine-02", status: "idle" },
  { id: "thread-gamma", title: "Policy review sync", machineLabel: "machine-03", status: "offline" },
];

function toStatusLabel(status: ThreadHubItem["status"]): string {
  if (status === "active") {
    return "Active";
  }
  if (status === "idle") {
    return "Idle";
  }
  return "Offline";
}

export function ThreadHubPanel(props: ThreadHubPanelProps) {
  const threads = props.threads ?? THREAD_HUB_DEMO_ITEMS;

  return (
    <div className="hub-panel">
      <div className="hub-panel-header">
        <div className="hub-panel-title-row">
          <div className="hub-panel-brand">
            <div className="thread-shell-brand-mark">CA</div>
            <div>
              <p className="thread-shell-kicker">Thread Hub</p>
              <h2>Sessions</h2>
            </div>
          </div>
        </div>
        <div className="hub-panel-summary">
          <span>{threads.filter((thread) => thread.status === "active").length} active</span>
          <span>{threads.length} total</span>
        </div>
      </div>

      <div className="hub-panel-scroll">
        <div className="hub-machine-group">
          <div className="hub-thread-list">
            {threads.map((thread) => {
              const isActive = props.activeThreadId === thread.id;
              return (
                <NavLink
                  key={thread.id}
                  to={`/threads/${thread.id}`}
                  className="hub-thread-link"
                  onClick={props.onNavigate}
                  aria-current={isActive ? "page" : undefined}
                >
                  <span className="session-status-dot is-online" aria-hidden />
                  <span>
                    <strong>{thread.title}</strong>
                    <small>{thread.machineLabel}</small>
                  </span>
                  <span className="hub-thread-chip">{toStatusLabel(thread.status)}</span>
                </NavLink>
              );
            })}
          </div>
        </div>
      </div>

      <div className="hub-panel-footer">
        <NavLink to="/machines" aria-label="Machines" className="hub-footer-link" onClick={props.onNavigate}>
          Machines
        </NavLink>
        <NavLink to="/environment" aria-label="Environment" className="hub-footer-link" onClick={props.onNavigate}>
          Environment
        </NavLink>
        <NavLink to="/settings" aria-label="Settings" className="hub-footer-link" onClick={props.onNavigate}>
          Settings
        </NavLink>
      </div>
    </div>
  );
}
