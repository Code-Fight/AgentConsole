import { FormEvent, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  ApprovalDecision,
  ApprovalQuestion,
  ApprovalRequiredPayload,
  ApprovalResolvedPayload,
  EventEnvelope,
  MachineDetailResponse,
  MachineSummary,
  MachineUpdatedPayload,
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

type ApprovalAnswerMap = Record<string, string>;

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
  const [steerPrompt, setSteerPrompt] = useState("");
  const [messages, setMessages] = useState<WorkspaceMessage[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>([]);
  const [approvalAnswers, setApprovalAnswers] = useState<Record<string, ApprovalAnswerMap>>({});
  const [machine, setMachine] = useState<MachineSummary | null>(null);
  const [activeTurnId, setActiveTurnId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setMessages([]);
    setPendingApprovals([]);
    setApprovalAnswers({});
    setMachine(null);
    setActiveTurnId(null);
    setSteerPrompt("");
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
        setActiveTurnId((current) => current ?? detail.activeTurnId ?? null);
        setPendingApprovals((current) => {
          const next = current.filter(
            (item) => !approvals.some((approval) => approval.requestId === item.requestId),
          );
          next.push(...approvals);
          return next;
        });
        setApprovalAnswers((current) => {
          const next = { ...current };
          for (const approval of approvals) {
            next[approval.requestId] = {
              ...(next[approval.requestId] ?? {}),
              ...buildDefaultApprovalAnswers(approval.questions),
            };
          }
          return next;
        });
        try {
          const machineDetail = await http<MachineDetailResponse>(
            `/machines/${encodeURIComponent(detail.thread.machineId)}`,
          );
          if (!cancelled) {
            setMachine(machineDetail.machine);
          }
        } catch {
          if (!cancelled) {
            setMachine(null);
          }
        }
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

    return connectConsoleSocket(undefined, (event) => {
      const envelope = parseEnvelope(event.data);
      if (!envelope) {
        return;
      }

      if (envelope.name === "machine.updated") {
        const payload = envelope.payload as MachineUpdatedPayload;
        setMachine((current) => (current?.id === payload.machine.id ? payload.machine : current));
        return;
      }

      if (getEnvelopeThreadId(envelope) !== threadId) {
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
        setApprovalAnswers((current) => ({
          ...current,
          [payload.requestId]: {
            ...(current[payload.requestId] ?? {}),
            ...buildDefaultApprovalAnswers(payload.questions),
          }
        }));
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
        setApprovalAnswers((current) => {
          const next = { ...current };
          delete next[payload.requestId];
          return next;
        });
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
        setActiveTurnId(payload.turnId);
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
        setActiveTurnId((current) =>
          current === payload.turn.turnId ? null : current,
        );
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

  async function handleApprovalDecision(
    requestId: string,
    decision: ApprovalDecision,
    answers?: ApprovalAnswerMap,
  ) {
    setError(null);

    try {
      await http<void>(`/approvals/${encodeURIComponent(requestId)}/respond`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          decision,
          ...(answers ? { answers } : {})
        })
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
      const response = await http<StartTurnResponse>(buildThreadApiPath(threadId, "turns"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ input })
      });

      setActiveTurnId(response.turn.turnId);
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

  async function handleSteerSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const input = steerPrompt.trim();
    if (!threadId || !activeTurnId || input === "") {
      return;
    }

    setError(null);

    try {
      await http<StartTurnResponse>(
        buildThreadApiPath(threadId, `turns/${activeTurnId}/steer`),
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ input })
        },
      );
      setSteerPrompt("");
    } catch (steerError) {
      setError(
        steerError instanceof Error
          ? steerError.message
          : "Unable to steer turn.",
      );
    }
  }

  async function handleInterrupt() {
    if (!threadId || !activeTurnId) {
      return;
    }

    setError(null);

    try {
      await http<void>(buildThreadApiPath(threadId, `turns/${activeTurnId}/interrupt`), {
        method: "POST"
      });
      setActiveTurnId(null);
    } catch (interruptError) {
      setError(
        interruptError instanceof Error
          ? interruptError.message
          : "Unable to interrupt turn.",
      );
    }
  }

  function handleApprovalAnswerChange(requestId: string, questionId: string, value: string) {
    setApprovalAnswers((current) => ({
      ...current,
      [requestId]: {
        ...(current[requestId] ?? {}),
        [questionId]: value
      }
    }));
  }

  return (
    <section>
      <h1>Thread Workspace</h1>
      <p>{threadId || "No thread selected"}</p>
      {machine ? <p>{machine.status} / {machine.runtimeStatus}</p> : null}
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
      {activeTurnId ? (
        <section>
          <p>Active turn: {activeTurnId}</p>
          <form onSubmit={handleSteerSubmit}>
            <label htmlFor="steer-prompt">Steer prompt</label>
            <textarea
              id="steer-prompt"
              aria-label="Steer prompt"
              value={steerPrompt}
              onChange={(event) => setSteerPrompt(event.target.value)}
            />
            <button type="submit">Send steer</button>
          </form>
          <button type="button" onClick={() => void handleInterrupt()}>
            Interrupt turn
          </button>
        </section>
      ) : null}
      {pendingApprovals.length > 0 ? (
        <section>
          <h2>Approval required</h2>
          <ul>
            {pendingApprovals.map((approval) => (
              <li key={approval.requestId}>
                <div>{approval.command || approval.reason || approval.kind}</div>
                {approval.kind === "tool_user_input" && approval.questions?.length ? (
                  <div>
                    {approval.questions.map((question) => (
                      <ApprovalQuestionField
                        key={question.id}
                        question={question}
                        value={approvalAnswers[approval.requestId]?.[question.id] ?? ""}
                        onChange={(value) =>
                          handleApprovalAnswerChange(approval.requestId, question.id, value)
                        }
                      />
                    ))}
                  </div>
                ) : null}
                <button
                  type="button"
                  onClick={() => void handleApprovalDecision(
                    approval.requestId,
                    "accept",
                    approval.kind === "tool_user_input" && approval.questions?.length
                      ? approvalAnswers[approval.requestId] ?? buildDefaultApprovalAnswers(approval.questions)
                      : undefined,
                  )}
                >
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

function buildDefaultApprovalAnswers(questions?: ApprovalQuestion[]): ApprovalAnswerMap {
  if (!questions?.length) {
    return {};
  }

  return Object.fromEntries(
    questions
      .filter((question) => question.id)
      .map((question) => [question.id, question.options?.[0] ?? ""]),
  );
}

function ApprovalQuestionField(props: {
  question: ApprovalQuestion;
  value: string;
  onChange: (value: string) => void;
}) {
  const { question, value, onChange } = props;
  const label = question.text || question.header || question.id;

  return (
    <div>
      {question.header && question.header !== label ? <p>{question.header}</p> : null}
      <label>
        {label}
        {question.options?.length ? (
          <select value={value} onChange={(event) => onChange(event.target.value)}>
            {question.options.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        ) : (
          <input type="text" value={value} onChange={(event) => onChange(event.target.value)} />
        )}
      </label>
    </div>
  );
}
