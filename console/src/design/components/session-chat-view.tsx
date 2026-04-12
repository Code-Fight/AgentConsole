import { FormEvent, useState } from "react";

export interface SessionMessage {
  id: string;
  kind: "user" | "agent" | "system";
  text: string;
}

interface SessionChatViewProps {
  title: string;
  subtitle: string;
  messages?: SessionMessage[];
  onSend?: (prompt: string) => void;
}

const DEFAULT_MESSAGES: SessionMessage[] = [
  { id: "m-1", kind: "system", text: "Waiting for live Gateway events." },
  { id: "m-2", kind: "user", text: "Start a health check on machine-01." },
  { id: "m-3", kind: "agent", text: "Queued. I will stream turn output here once connected." },
];

function classNameForMessage(kind: SessionMessage["kind"]): string {
  if (kind === "system") {
    return "chat-stream-item chat-stream-info";
  }
  if (kind === "agent") {
    return "chat-stream-item chat-stream-success";
  }
  return "chat-stream-item";
}

export function SessionChatView(props: SessionChatViewProps) {
  const messages = props.messages ?? DEFAULT_MESSAGES;
  const [draft, setDraft] = useState("");

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextPrompt = draft.trim();
    if (nextPrompt) {
      props.onSend?.(nextPrompt);
    }
    setDraft("");
  }

  return (
    <div className="thread-workspace-page page">
      <header className="page-header">
        <span className="page-kicker">Thread Workspace</span>
        <h1>{props.title}</h1>
        <p>{props.subtitle}</p>
      </header>

      <div className="thread-workspace-layout">
        <section className="workspace-conversation-card">
          <div className="chat-stream-header">
            <h2>Session Chat</h2>
            <span className="session-panel-count">{messages.length}</span>
          </div>
          <ul className="chat-stream workspace-stream-list">
            {messages.map((message) => (
              <li key={message.id} className={classNameForMessage(message.kind)}>
                {message.text}
              </li>
            ))}
          </ul>

          <form className="chat-compose" onSubmit={handleSubmit}>
            <label htmlFor="session-chat-input">Prompt</label>
            <textarea
              id="session-chat-input"
              aria-label="Prompt"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              placeholder="Message input is design-only in Task 2."
            />
            <button type="submit">Send</button>
          </form>
        </section>

        <aside className="workspace-control-stack">
          <section className="workspace-status-card">
            <h2>Turn Controls</h2>
            <p>Gateway actions are intentionally not connected in this import step.</p>
            <div className="approval-actions">
              <button type="button" disabled>
                Interrupt turn
              </button>
              <button type="button" disabled>
                Steer turn
              </button>
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
