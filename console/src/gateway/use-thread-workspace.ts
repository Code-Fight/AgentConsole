import { useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { buildThreadApiPath, http } from "../common/api/http";
import type {
  ApprovalDecision,
  ApprovalRequiredPayload,
  ApprovalResolvedPayload,
  MachineDetailResponse,
  MachineSummary,
  MachineUpdatedPayload,
  StartTurnResponse,
  ThreadDetailResponse,
  TurnCompletedPayload,
  TurnDeltaPayload,
  TurnStartedPayload,
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";
import { supportsCapability } from "./capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "./gateway-connection-store";
import {
  buildDefaultApprovalAnswers,
  formatMachineStatus,
  getEnvelopeThreadId,
  parseEnvelope,
  toApprovalCardViewModel,
  toTurnCompletedMessage,
  toTurnStartedMessage,
  toWorkspaceMessage,
  type WorkspaceMessageViewModel,
} from "./thread-view-model";

type ApprovalAnswerMap = Record<string, string>;
type PendingApproval = ApprovalRequiredPayload & { requestId: string };

export type ThreadWorkspaceViewModel = ReturnType<typeof useThreadWorkspace>;

interface UseThreadWorkspaceOptions {
  enabled?: boolean;
}

export function useThreadWorkspace(threadId: string, options?: UseThreadWorkspaceOptions) {
  const enabled = options?.enabled ?? true;
  const connectionIdentity = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionIdentity,
    getGatewayConnectionIdentity,
  );
  const [prompt, setPrompt] = useState("");
  const [steerPrompt, setSteerPrompt] = useState("");
  const [messages, setMessages] = useState<WorkspaceMessageViewModel[]>([]);
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>([]);
  const [approvalAnswers, setApprovalAnswers] = useState<Record<string, ApprovalAnswerMap>>({});
  const [machine, setMachine] = useState<MachineSummary | null>(null);
  const [activeTurnId, setActiveTurnId] = useState<string | null>(null);
  const [threadTitle, setThreadTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    setPrompt("");
    setSteerPrompt("");
    setMessages([]);
    setPendingApprovals([]);
    setApprovalAnswers({});
    setMachine(null);
    setActiveTurnId(null);
    setThreadTitle("");
    setError(null);
  }, [threadId, connectionIdentity]);

  const loadWorkspace = useCallback(async () => {
    if (!enabled || !threadId) {
      return;
    }

    try {
      const detail = await http<ThreadDetailResponse>(buildThreadApiPath(threadId));
      const approvals = Array.isArray(detail.pendingApprovals)
        ? detail.pendingApprovals
            .filter((approval): approval is PendingApproval => Boolean(approval?.requestId))
            .map((approval) => ({ ...approval, requestId: approval.requestId }))
        : [];

      setThreadTitle(detail.thread.title ?? "");
      setActiveTurnId((current) => current ?? detail.activeTurnId ?? null);
      setMessages((current) => (current.length === 0 ? (detail.messages ?? []) : current));
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
      setError(null);

      try {
        const machineDetail = await http<MachineDetailResponse>(
          `/machines/${encodeURIComponent(detail.thread.machineId)}`,
        );
        setMachine(machineDetail.machine);
      } catch {
        setMachine(null);
      }
    } catch (detailError) {
      setError(
        detailError instanceof Error ? detailError.message : "Unable to load thread.",
      );
    }
  }, [enabled, threadId]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void loadWorkspace();
  }, [enabled, connectionIdentity, loadWorkspace]);

  useEffect(() => {
    if (!enabled || !threadId) {
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
          },
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
        setMessages((current) => {
          const next = [...current];
          const lastMessage = next[next.length - 1];
          if (lastMessage?.kind === "agent" && lastMessage.turnId === payload.turnId) {
            next[next.length - 1] = {
              ...lastMessage,
              text: `${lastMessage.text}${payload.delta}`,
            };
            return next;
          }

          next.push(toWorkspaceMessage(payload));
          return next;
        });
        return;
      }

      if (envelope.name === "turn.started") {
        const payload = envelope.payload as TurnStartedPayload;
        setActiveTurnId(payload.turnId);
        setMessages((current) => [...current, toTurnStartedMessage(payload)]);
        return;
      }

      if (envelope.name === "turn.completed" || envelope.name === "turn.failed") {
        const payload = envelope.payload as TurnCompletedPayload;
        setActiveTurnId((current) => (current === payload.turn.turnId ? null : current));
        setMessages((current) => [...current, toTurnCompletedMessage(payload)]);
      }
    });
  }, [enabled, threadId, connectionIdentity]);

  const handleApprovalDecision = useCallback(
    async (requestId: string, decision: ApprovalDecision, answers?: ApprovalAnswerMap) => {
      if (!enabled || !supportsCapability("approvals")) {
        return;
      }

      setError(null);

      try {
        await http<void>(`/approvals/${encodeURIComponent(requestId)}/respond`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            decision,
            ...(answers ? { answers } : {}),
          }),
        });
      } catch (approvalError) {
        setError(
          approvalError instanceof Error
            ? approvalError.message
            : "Unable to respond to approval.",
        );
      }
    },
    [enabled],
  );

  const handlePromptSubmit = useCallback(async () => {
    const input = prompt.trim();
    if (!enabled || !threadId || input === "" || !supportsCapability("startTurn")) {
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await http<StartTurnResponse>(buildThreadApiPath(threadId, "turns"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ input }),
      });

      setActiveTurnId(response.turn.turnId);
      setMessages((current) => [
        ...current,
        {
          id: `user:${Date.now()}`,
          text: input,
          kind: "user",
        },
      ]);
      setPrompt("");
    } catch (submissionError) {
      setError(
        submissionError instanceof Error ? submissionError.message : "Unable to start turn.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }, [enabled, prompt, threadId]);

  const handleSteerSubmit = useCallback(async () => {
    const input = steerPrompt.trim();
    if (
      !enabled ||
      !threadId ||
      !activeTurnId ||
      input === "" ||
      !supportsCapability("steerTurn")
    ) {
      return;
    }

    setError(null);

    try {
      await http<StartTurnResponse>(buildThreadApiPath(threadId, `turns/${activeTurnId}/steer`), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ input }),
      });
      setSteerPrompt("");
    } catch (steerError) {
      setError(
        steerError instanceof Error ? steerError.message : "Unable to steer turn.",
      );
    }
  }, [activeTurnId, enabled, steerPrompt, threadId]);

  const handleInterrupt = useCallback(async () => {
    if (!enabled || !threadId || !activeTurnId || !supportsCapability("interruptTurn")) {
      return;
    }

    setError(null);

    try {
      await http<void>(buildThreadApiPath(threadId, `turns/${activeTurnId}/interrupt`), {
        method: "POST",
      });
      setActiveTurnId(null);
    } catch (interruptError) {
      setError(
        interruptError instanceof Error
          ? interruptError.message
          : "Unable to interrupt turn.",
      );
    }
  }, [activeTurnId, enabled, threadId]);

  const handleApprovalAnswerChange = useCallback(
    (requestId: string, questionId: string, value: string) => {
      setApprovalAnswers((current) => ({
        ...current,
        [requestId]: {
          ...(current[requestId] ?? {}),
          [questionId]: value,
        },
      }));
    },
    [],
  );

  const approvalCards = useMemo(
    () =>
      pendingApprovals.map((approval) =>
        toApprovalCardViewModel(approval, approvalAnswers[approval.requestId] ?? {}),
      ),
    [approvalAnswers, pendingApprovals],
  );

  return {
    title: threadTitle || "线程工作区",
    subtitle: threadId || "未选择线程",
    error,
    machine: machine
      ? {
          statusLabel: formatMachineStatus(machine.status),
          runtimeLabel: machine.runtimeStatus,
          name: machine.name || machine.id,
        }
      : null,
    messages,
    pendingApprovals: approvalCards,
    activeTurnId,
    prompt,
    steerPrompt,
    isSubmitting,
    canStartTurn: enabled && supportsCapability("startTurn"),
    canSteerTurn: enabled && supportsCapability("steerTurn") && Boolean(activeTurnId),
    canInterruptTurn: enabled && supportsCapability("interruptTurn") && Boolean(activeTurnId),
    setPrompt,
    setSteerPrompt,
    handlePromptSubmit,
    handleSteerSubmit,
    handleInterrupt,
    handleApprovalAnswerChange,
    handleApprovalDecision,
  };
}
