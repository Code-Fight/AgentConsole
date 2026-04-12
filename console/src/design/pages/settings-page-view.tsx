const SETTINGS_SECTIONS = [
  { title: "Gateway Endpoint", description: "http://localhost:5181", status: "configured" },
  { title: "Console Profile", description: "design-import-preview", status: "preview" },
  { title: "Safety Policy", description: "Awaiting adapter wiring", status: "pending" },
];

export function SettingsPageView() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Settings</h1>
        <p>Imported settings surface. Save actions remain disabled until Gateway adapters are connected.</p>
      </header>

      <div className="settings-section-list">
        {SETTINGS_SECTIONS.map((section) => (
          <article key={section.title} className="settings-section-card">
            <h2>{section.title}</h2>
            <p>{section.description}</p>
            <div className="settings-actions">
              <span className="hub-thread-chip">{section.status}</span>
              <button type="button" disabled>
                Edit
              </button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
