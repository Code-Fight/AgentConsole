const MACHINES = [
  { id: "machine-01", status: "online", runtime: "running" },
  { id: "machine-02", status: "reconnecting", runtime: "warming" },
  { id: "machine-03", status: "offline", runtime: "unknown" },
];

export function MachinesPageView() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Machines</h1>
        <p>Design-source management surface (static in Task 2).</p>
      </header>

      <div className="machines-grid">
        {MACHINES.map((machine) => (
          <article key={machine.id} className="machine-card">
            <div className="machine-card-header">
              <strong>{machine.id}</strong>
              <span className="hub-thread-chip">{machine.status}</span>
            </div>
            <p>Runtime: {machine.runtime}</p>
            <div className="approval-actions">
              <button type="button" disabled>
                Install agent
              </button>
              <button type="button" disabled>
                Remove agent
              </button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
