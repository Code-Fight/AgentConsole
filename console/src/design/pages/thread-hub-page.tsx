import { FormEvent } from "react";
import { Link } from "react-router-dom";

export interface ThreadHubPageThreadItem {
  id: string;
  title: string;
  threadLabel: string;
  machineLabel: string;
  statusLabel: string;
  machineRuntimeLabel: string;
}

interface ThreadHubPageProps {
  error?: string | null;
  threads?: ThreadHubPageThreadItem[];
  machineSuggestions?: Array<{ id: string; label: string }>;
  machineCount?: number;
  machineId?: string;
  title?: string;
  isSubmitting?: boolean;
  onMachineIdChange?: (value: string) => void;
  onTitleChange?: (value: string) => void;
  onCreateThread?: () => void;
  onResume?: (threadId: string) => void;
  onArchive?: (threadId: string) => void;
  onDelete?: (threadId: string) => void;
}

export function ThreadHubPage(props: ThreadHubPageProps) {
  const threads = props.threads ?? [];

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    props.onCreateThread?.();
  }

  return (
    <section className="page threads-page">
      <header className="page-header">
        <span className="page-kicker">Thread Hub</span>
        <h1>Thread Hub</h1>
        <p>Unified thread lifecycle, live machine routing, and workspace entry points through the Gateway adapter layer.</p>
      </header>

      {props.error ? <p>{props.error}</p> : null}

      <div className="threads-grid">
        <form className="threads-create-card" onSubmit={handleSubmit}>
          <div className="threads-card-heading">
            <div>
              <span className="page-kicker">Create</span>
              <h2>Create Thread</h2>
            </div>
            <span className="session-panel-count">{props.machineCount ?? 0}</span>
          </div>
          <label htmlFor="hub-machine-id">Machine ID</label>
          <input
            id="hub-machine-id"
            aria-label="Machine ID"
            list="hub-machine-suggestions"
            value={props.machineId ?? ""}
            onChange={(event) => props.onMachineIdChange?.(event.target.value)}
          />
          <datalist id="hub-machine-suggestions">
            {(props.machineSuggestions ?? []).map((machine) => (
              <option key={machine.id} value={machine.id}>
                {machine.label}
              </option>
            ))}
          </datalist>
          <label htmlFor="hub-thread-title">Title</label>
          <input
            id="hub-thread-title"
            aria-label="Title"
            value={props.title ?? ""}
            onChange={(event) => props.onTitleChange?.(event.target.value)}
          />
          <button type="submit" aria-label="Create thread" disabled={props.isSubmitting}>
            Create thread
          </button>
        </form>

        <section className="threads-list-card">
          <div className="threads-card-heading">
            <div>
              <span className="page-kicker">Threads</span>
              <h2>Active List</h2>
            </div>
            <span className="session-panel-count">{threads.length}</span>
          </div>

          {threads.length === 0 ? <p>当前没有可用线程。</p> : null}

          {threads.length > 0 ? (
            <ul className="threads-list">
              {threads.map((thread) => (
                <li key={thread.id} className="threads-list-item">
                  <div className="threads-list-main">
                    <Link to={`/threads/${thread.id}`}>{thread.title}</Link>
                    <small>{thread.threadLabel}</small>
                  </div>
                  <div className="threads-list-meta">
                    <span className="hub-thread-chip">{thread.statusLabel}</span>
                    <span className="meta-pill">{thread.machineRuntimeLabel}</span>
                  </div>
                  <div className="threads-list-actions">
                    <button
                      type="button"
                      aria-label={`Resume ${thread.title}`}
                      onClick={() => props.onResume?.(thread.id)}
                    >
                      恢复
                    </button>
                    <button
                      type="button"
                      aria-label={`Archive ${thread.title}`}
                      onClick={() => props.onArchive?.(thread.id)}
                    >
                      归档
                    </button>
                    <button
                      type="button"
                      aria-label={`Delete ${thread.title}`}
                      onClick={() => props.onDelete?.(thread.id)}
                    >
                      删除
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          ) : null}
        </section>
      </div>
    </section>
  );
}
