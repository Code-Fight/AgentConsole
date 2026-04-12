import { Link } from "react-router-dom";
import { THREAD_HUB_DEMO_ITEMS } from "../components/thread-hub-panel";

export function ThreadHubPage() {
  return (
    <section className="page threads-page">
      <header className="page-header">
        <span className="page-kicker">Thread Hub</span>
        <h1>Design Import Preview</h1>
        <p>This page is sourced from the isolated design layer. Gateway data wiring comes in later tasks.</p>
      </header>

      <div className="threads-grid">
        <section className="threads-create-card">
          <div className="threads-card-heading">
            <div>
              <span className="page-kicker">Design Source</span>
              <h2>Create Thread</h2>
            </div>
          </div>
          <label htmlFor="hub-machine-id">Machine ID</label>
          <input id="hub-machine-id" value="machine-01" readOnly />
          <label htmlFor="hub-thread-title">Title</label>
          <input id="hub-thread-title" value="New Gateway Session" readOnly />
          <button type="button" disabled>
            Create thread
          </button>
        </section>

        <section className="threads-list-card">
          <div className="threads-card-heading">
            <div>
              <span className="page-kicker">Threads</span>
              <h2>Active List</h2>
            </div>
            <span className="session-panel-count">{THREAD_HUB_DEMO_ITEMS.length}</span>
          </div>

          <ul className="threads-list">
            {THREAD_HUB_DEMO_ITEMS.map((thread) => (
              <li key={thread.id} className="threads-list-item">
                <div className="threads-list-main">
                  <Link to={`/threads/${thread.id}`}>{thread.title}</Link>
                  <small>{thread.machineLabel}</small>
                </div>
                <div className="threads-list-meta">
                  <span className="hub-thread-chip">{thread.status}</span>
                </div>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </section>
  );
}
