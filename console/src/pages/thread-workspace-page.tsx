import { FormEvent, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  EventEnvelope,
  StartTurnResponse,
  TurnCompletedPayload,
  TurnDeltaPayload
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

interface WorkspaceMessage {
  id: string;
  text: string;
}

function parseEnvelope(raw: string): EventEnvelope | null {
  try {
    return JSON.parse(raw) as EventEnvelope;
  } catch {
    return null;
  }
}

function getEnvelopeThreadId(envelope: EventEnvelope): string | null {
  if (envelope.name === "turn.delta") {
    return (envelope.payload as TurnDeltaPayload).threadId;
  }

  if (envelope.name === "turn.completed") {
    return (envelope.payload as TurnCompletedPayload).turn.threadId;
  }

  return null;
}

export function ThreadWorkspacePage() {
  const { threadId = "" } = useParams();
  const [prompt, setPrompt] = useState("");
  const [messages, setMessages] = useState<WorkspaceMessage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setMessages([]);
    setError(null);
  }, [threadId]);

  useEffect(() => {
    if (!threadId) {
      return undefined;
    }

    return connectConsoleSocket(threadId, (event) => {
      const envelope = parseEnvelope(event.data);
      if (!envelope || getEnvelopeThreadId(envelope) !== threadId) {
        return;
      }

      if (envelope.name === "turn.delta") {
        const payload = envelope.payload as TurnDeltaPayload;
        setMessages((current) => [
          ...current,
          {
            id: `${payload.turnId}:${payload.sequence}`,
            text: payload.delta
          }
        ]);
        return;
      }

      if (envelope.name === "turn.completed") {
        const payload = envelope.payload as TurnCompletedPayload;
        setMessages((current) => [
          ...current,
          {
            id: `completed:${payload.turn.turnId}`,
            text: `Turn ${payload.turn.turnId} ${payload.turn.status}`
          }
        ]);
      }
    });
  }, [threadId]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const input = prompt.trim();
    if (!threadId || input === "") {
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await http<StartTurnResponse>(buildThreadApiPath(threadId, "turns"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ input })
      });

      setPrompt("");
    } catch (submissionError) {
      setError(
        submissionError instanceof Error
          ? submissionError.message
          : "Unable to start turn.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section>
      <h1>Thread Workspace</h1>
      <p>{threadId || "No thread selected"}</p>
      {error ? <p>{error}</p> : null}
      <form onSubmit={handleSubmit}>
        <label htmlFor="thread-prompt">Prompt</label>
        <textarea
          id="thread-prompt"
          aria-label="Prompt"
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
        />
        <button type="submit" disabled={isSubmitting}>
          Send prompt
        </button>
      </form>
      <div>Realtime messages</div>
      <ul>
        {messages.map((message) => (
          <li key={message.id}>{message.text}</li>
        ))}
      </ul>
    </section>
  );
}
