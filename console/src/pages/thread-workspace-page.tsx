import { FormEvent, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  ApprovalDecision,
  ApprovalRequiredPayload,
  ApprovalResolvedPayload,
  EventEnvelope,
  ThreadDetailResponse,
  StartTurnResponse,
  TurnStartedPayload,
  TurnCompletedPayload,
  TurnDeltaPayload
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

interface WorkspaceMessage {
  id: string;
  text: string;
}

interface PendingApproval extends ApprovalRequiredPayload {
  requestId: string;
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

  if (envelope.name === "turn.started") {
    return (envelope.payload as TurnStartedPayload).threadId;
  }

  if (envelope.name === "turn.completed") {
    return (envelope.payload as TurnCompletedPayload).turn.threadId;
  }

  if (envelope.name === "approval.required") {
    return (envelope.payload as ApprovalRequiredPayload).threadId ?? null;
  }

  if (envelope.name === "approval.resolved") {
    return (envelope.payload as ApprovalResolvedPayload).threadId ?? null;
  }

  return null;
}

export function ThreadWorkspacePage() {
  const { threadId = "" } = useParams();
  const [prompt, setPrompt] = useState("");
  const [messages, setMessages] = useState<WorkspaceMessage[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setMessages([]);
    setPendingApprovals([]);
    setError(null);
  }, [threadId]);

  useEffect(() => {
    if (!threadId) {
      return undefined;
    }

    let cancelled = false;

    void (async () => {
      try {
        const detail = await http<ThreadDetailResponse>(buildThreadApiPath(threadId));
        if (cancelled) {
          return;
        }

        const approvals = Array.isArray(detail?.pendingApprovals)
          ? detail.pendingApprovals
              .filter((approval): approval is PendingApproval => Boolean(approval?.requestId))
              .map((approval) => ({ ...approval, requestId: approval.requestId }))
          : [];
        setPendingApprovals((current) => {
          const next = current.filter(
            (item) => !approvals.some((approval) => approval.requestId === item.requestId),
          );
          next.push(...approvals);
          return next;
        });
      } catch (detailError) {
        if (cancelled) {
          return;
        }
        setError(
          detailError instanceof Error
            ? detailError.message
            : "Unable to load thread.",
        );
      }
    })();

    return () => {
      cancelled = true;
    };
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

      if (envelope.name === "approval.required") {
        const payload = envelope.payload as ApprovalRequiredPayload;
        if (!payload.requestId) {
          return;
        }

        setPendingApprovals((current) => {
          const next = current.filter((item) => item.requestId !== payload.requestId);
          next.push({ ...payload, requestId: payload.requestId });
          return next;
        });
        return;
      }

      if (envelope.name === "approval.resolved") {
        const payload = envelope.payload as ApprovalResolvedPayload;
        if (!payload.requestId) {
          return;
        }

        setPendingApprovals((current) =>
          current.filter((item) => item.requestId !== payload.requestId),
        );
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

      if (envelope.name === "turn.started") {
        const payload = envelope.payload as TurnStartedPayload;
        setMessages((current) => [
          ...current,
          {
            id: `started:${payload.turnId}`,
            text: `Turn started: ${payload.turnId}`
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

  async function handleApprovalDecision(requestId: string, decision: ApprovalDecision) {
    setError(null);

    try {
      await http<void>(`/approvals/${encodeURIComponent(requestId)}/respond`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ decision })
      });
    } catch (approvalError) {
      setError(
        approvalError instanceof Error
          ? approvalError.message
          : "Unable to respond to approval.",
      );
    }
  }

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
      {pendingApprovals.length > 0 ? (
        <section>
          <h2>Approval required</h2>
          <ul>
            {pendingApprovals.map((approval) => (
              <li key={approval.requestId}>
                <div>{approval.command || approval.reason || approval.kind}</div>
                <button type="button" onClick={() => void handleApprovalDecision(approval.requestId, "accept")}>
                  Accept
                </button>
                <button type="button" onClick={() => void handleApprovalDecision(approval.requestId, "decline")}>
                  Decline
                </button>
                <button type="button" onClick={() => void handleApprovalDecision(approval.requestId, "cancel")}>
                  Cancel
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      <div>Realtime messages</div>
      <ul>
        {messages.map((message) => (
          <li key={message.id}>{message.text}</li>
        ))}
      </ul>
    </section>
  );
}
