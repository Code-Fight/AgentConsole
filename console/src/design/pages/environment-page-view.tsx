const RESOURCES = [
  { title: "Skills", detail: "12 loaded", action: "Sync catalog" },
  { title: "MCP", detail: "3 endpoints", action: "Restart bridge" },
  { title: "Plugins", detail: "5 enabled", action: "Open marketplace" },
];

export function EnvironmentPageView() {
  return (
    <section className="page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Environment</h1>
        <p>Design-source environment page imported into the isolated `design/` layer.</p>
      </header>

      <div className="environment-grid">
        {RESOURCES.map((resource) => (
          <article key={resource.title} className="resource-card-main">
            <h2>{resource.title}</h2>
            <p>{resource.detail}</p>
            <button type="button" disabled>
              {resource.action}
            </button>
          </article>
        ))}
      </div>
    </section>
  );
}
