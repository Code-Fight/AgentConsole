import type { MachineSummary } from "../../common/api/types";

interface MachinesPageViewProps {
  machines: MachineSummary[];
  isLoading: boolean;
  error: string | null;
  capabilities: {
    installAgent: boolean;
    removeAgent: boolean;
  };
}

function UnsupportedAction(props: { label: string }) {
  return (
    <div className="approval-actions">
      <button type="button" disabled aria-label={props.label}>
        {props.label}
      </button>
      <span className="meta-pill">Not connected</span>
    </div>
  );
}

export function MachinesPageView(props: MachinesPageViewProps) {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Machines</h1>
        <p>Connected Gateway runtimes, routed through the design surface with explicit capability policy.</p>
      </header>

      {props.isLoading ? <p>Loading machines…</p> : null}
      {props.error ? <p>{props.error}</p> : null}
      {!props.isLoading && !props.error && props.machines.length === 0 ? (
        <p>No machines reported.</p>
      ) : null}

      {!props.isLoading && !props.error ? (
        <div className="machines-grid">
          {props.machines.map((machine) => (
            <article key={machine.id} className="machine-card">
              <div className="machine-card-header">
                <strong>{machine.name || machine.id}</strong>
                <span className="hub-thread-chip">{machine.status}</span>
              </div>
              <p>{machine.id}</p>
              <p>Runtime: {machine.runtimeStatus}</p>
              {!props.capabilities.installAgent ? <UnsupportedAction label="Install agent" /> : null}
              {!props.capabilities.removeAgent ? <UnsupportedAction label="Remove agent" /> : null}
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}
