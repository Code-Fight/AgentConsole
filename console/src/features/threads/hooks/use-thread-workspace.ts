import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import type {
  ApprovalDecision,
  ApprovalRequiredPayload,
  ApprovalResolvedPayload,
  AgentTimelineEvent,
  MachineUpdatedPayload,
  ThreadHistoryMessage,
  ThreadRuntimeSettings,
  TurnCompletedPayload,
  TurnDeltaPayload,
  TurnStartedPayload,
} from "../../../common/api/types";
import { supportsCapability } from "../../../common/config/capabilities";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "../../../common/config/gateway-connection-store";
import { connectConsoleSocket } from "../../../common/api/ws";
import {
  getMachineDetail,
  getThreadDetail,
  getThreadRuntimeSettings,
  interruptThreadTurn,
  resumeThread,
  respondToApproval,
  startThreadTurn,
  steerThreadTurn,
  updateThreadRuntimeSettings,
} from "../api/thread-api";
import {
  buildDefaultApprovalAnswers,
  formatMachineStatus,
  getEnvelopeThreadId,
  parseEnvelope,
  toApprovalCardViewModel,
  toTurnFailedMessage,
  toWorkspaceMessage,
  type WorkspaceMessageViewModel,
} from "../model/thread-view-model";
import {
  isTimelineEventEnvelope,
  mergeTimelineEventIntoMessages,
  timelineEventMarksTurnCompleted,
  timelineEventToApprovalRequired,
} from "../model/timeline-model";

type ApprovalAnswerMap = Record<string, string>;
type PendingApproval = ApprovalRequiredPayload & { requestId: string };

export type ThreadWorkspaceViewModel = ReturnType<typeof useThreadWorkspace>;

const transientAnalyzingDelta = "\n\n正在分析...";

function mergeTurnDeltaText(currentText: string, delta: string): string {
  if (delta === transientAnalyzingDelta && currentText.includes(transientAnalyzingDelta)) {
    return currentText;
  }

  if (delta === transientAnalyzingDelta) {
    return `${currentText}${delta}`;
  }

  return `${currentText.replace(transientAnalyzingDelta, "")}${delta}`;
}

function cleanTransientProgressText(text?: string): string | undefined {
  const cleaned = (text ?? "").replace(transientAnalyzingDelta, "");
  return cleaned.length > 0 ? cleaned : undefined;
}

function mergeProgressDeltaText(currentText: string | undefined, delta: string): string | undefined {
  const current = currentText ?? "";
  if (delta === transientAnalyzingDelta) {
    if (current.includes(transientAnalyzingDelta)) {
      return current;
    }
    return `${current}${delta}`;
  }

  const merged = `${current.replace(transientAnalyzingDelta, "")}${delta}`;
  return merged.length > 0 ? merged : undefined;
}

function isProgressDelta(payload: TurnDeltaPayload): boolean {
  return payload.kind === "progress";
}

function withoutPendingFlag(message: WorkspaceMessageViewModel): WorkspaceMessageViewModel {
  const { isPending: _isPending, ...next } = message;
  return next;
}

function withOptionalProgressText(
  message: WorkspaceMessageViewModel,
  progressText: string | undefined,
): WorkspaceMessageViewModel {
  if (progressText) {
    return { ...message, progressText };
  }
  const { progressText: _progressText, ...next } = message;
  return next;
}

function toWorkspaceHistoryMessages(messages: ThreadHistoryMessage[]): WorkspaceMessageViewModel[] {
  return messages.map((message) => ({
    id: message.id,
    kind: message.kind,
    text: message.text,
    turnId: message.turnId,
    progressText: message.progressText,
  }));
}

function buildTimelineWorkspaceMessages(events: AgentTimelineEvent[]): WorkspaceMessageViewModel[] {
  return [...events]
    .sort((left, right) => left.sequence - right.sequence)
    .reduce<WorkspaceMessageViewModel[]>(mergeTimelineEventIntoMessages, []);
}

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
  const [machine, setMachine] = useState<{
    id: string;
    name: string;
    status: string;
    runtimeStatus: string;
  } | null>(null);
  const [activeTurnId, setActiveTurnId] = useState<string | null>(null);
  const [threadTitle, setThreadTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTurnPending, setIsTurnPending] = useState(false);
  const [runtimeModel, setRuntimeModel] = useState("");
  const [runtimeApprovalPolicy, setRuntimeApprovalPolicy] = useState("");
  const [runtimeSandboxMode, setRuntimeSandboxMode] = useState("");
  const [runtimeModelOptions, setRuntimeModelOptions] = useState<
    Array<{ id: string; displayName: string; isDefault: boolean }>
  >([]);
  const [runtimeSandboxOptions, setRuntimeSandboxOptions] = useState<string[]>([]);
  const [isUpdatingRuntimeSettings, setIsUpdatingRuntimeSettings] = useState(false);
  const completedTurnIdsRef = useRef<Set<string>>(new Set());
  const timelineTurnIdsRef = useRef<Set<string>>(new Set());
  const pendingAgentMessageIdRef = useRef<string | null>(null);

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
    setIsSubmitting(false);
    setIsTurnPending(false);
    setRuntimeModel("");
    setRuntimeApprovalPolicy("");
    setRuntimeSandboxMode("");
    setRuntimeModelOptions([]);
    setRuntimeSandboxOptions([]);
    setIsUpdatingRuntimeSettings(false);
    completedTurnIdsRef.current.clear();
    timelineTurnIdsRef.current.clear();
    pendingAgentMessageIdRef.current = null;
  }, [threadId, connectionIdentity]);

  const appendTurnDelta = useCallback((payload: TurnDeltaPayload) => {
    completedTurnIdsRef.current.delete(payload.turnId);
    setIsTurnPending(false);
    setActiveTurnId((current) => current ?? payload.turnId);
    setMessages((current) => {
      const next = [...current];
      const lastMessage = next[next.length - 1];
      const pendingAgentMessageId = pendingAgentMessageIdRef.current;
      if (
        lastMessage?.kind === "agent" &&
        (lastMessage.turnId === payload.turnId || lastMessage.id === pendingAgentMessageId)
      ) {
        if (lastMessage.id === pendingAgentMessageId) {
          pendingAgentMessageIdRef.current = null;
        }
        if (isProgressDelta(payload)) {
          next[next.length - 1] = withOptionalProgressText(
            withoutPendingFlag({
              ...lastMessage,
              turnId: payload.turnId,
            }),
            mergeProgressDeltaText(lastMessage.progressText, payload.delta),
          );
        } else {
          next[next.length - 1] = withOptionalProgressText(
            withoutPendingFlag({
              ...lastMessage,
              turnId: payload.turnId,
              text: mergeTurnDeltaText(lastMessage.text, payload.delta),
            }),
            cleanTransientProgressText(lastMessage.progressText),
          );
        }
        return next;
      }

      next.push(toWorkspaceMessage(payload));
      return next;
    });
  }, []);

  const bindPendingAgentMessageToTurn = useCallback((turnId: string) => {
    const pendingAgentMessageId = pendingAgentMessageIdRef.current;
    if (!pendingAgentMessageId) {
      return;
    }

    pendingAgentMessageIdRef.current = null;
    setMessages((current) =>
      current.map((message) =>
        message.id === pendingAgentMessageId
          ? withoutPendingFlag({ ...message, id: `agent:${turnId}`, turnId })
          : message,
      ),
    );
  }, []);

  const applyRuntimeSettings = useCallback((settings: ThreadRuntimeSettings) => {
    const preferences = settings.preferences ?? {};
    const options = settings.options ?? {
      models: [],
      approvalPolicies: [],
      sandboxModes: [],
    };

    const nextModel = (preferences.model ?? "").trim();
    const nextApprovalPolicy = (preferences.approvalPolicy ?? "").trim();
    const nextSandboxMode = (preferences.sandboxMode ?? "").trim();

    const modelOptions = (Array.isArray(options.models) ? options.models : [])
      .map((item) => ({
        id: (item?.id ?? "").trim(),
        displayName: (item?.displayName ?? item?.id ?? "").trim(),
        isDefault: Boolean(item?.isDefault),
      }))
      .filter((item) => item.id !== "");

    if (nextModel && !modelOptions.some((item) => item.id === nextModel)) {
      modelOptions.push({
        id: nextModel,
        displayName: nextModel,
        isDefault: false,
      });
    }

    const sandboxModes = (Array.isArray(options.sandboxModes) ? options.sandboxModes : [])
      .map((item) => item.trim())
      .filter((item) => item !== "");

    if (nextSandboxMode && !sandboxModes.includes(nextSandboxMode)) {
      sandboxModes.push(nextSandboxMode);
    }

    setRuntimeModel(nextModel);
    setRuntimeApprovalPolicy(nextApprovalPolicy);
    setRuntimeSandboxMode(nextSandboxMode);
    setRuntimeModelOptions(modelOptions);
    setRuntimeSandboxOptions(sandboxModes);
  }, []);

  const loadWorkspace = useCallback(async () => {
    if (!enabled || !threadId) {
      return;
    }

    let resolvedMachineID = "";
    try {
      const detail = await getThreadDetail(threadId);
      let resolvedTitle = detail.thread.title ?? "";
      let threadMessages =
        detail.events?.length
          ? buildTimelineWorkspaceMessages(detail.events)
          : toWorkspaceHistoryMessages(detail.messages ?? []);
      let historyLoadError: string | null = null;
      if (threadMessages.length === 0 && detail.thread.status !== "active") {
        try {
          const resumed = await resumeThread(threadId);
          const resumedMessages = resumed.thread.messages ?? [];
          if (resumedMessages.length > 0) {
            threadMessages = toWorkspaceHistoryMessages(resumedMessages);
            if (resolvedTitle.trim() === "") {
              resolvedTitle = resumed.thread.title ?? resolvedTitle;
            }
          }
        } catch (resumeError) {
          historyLoadError =
            resumeError instanceof Error
              ? resumeError.message
              : "Unable to load thread history.";
        }
      }
      const approvals = Array.isArray(detail.pendingApprovals)
        ? detail.pendingApprovals
            .filter((approval): approval is PendingApproval => Boolean(approval?.requestId))
            .map((approval) => ({ ...approval, requestId: approval.requestId }))
        : [];

      setThreadTitle(resolvedTitle);
      setActiveTurnId((current) => current ?? detail.activeTurnId ?? null);
      setMessages((current) => (current.length === 0 ? threadMessages : current));
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
      if (historyLoadError !== null && threadMessages.length === 0) {
        setError(historyLoadError);
      } else {
        setError(null);
      }
      resolvedMachineID = detail.thread.machineId;

      try {
        const machineDetail = await getMachineDetail(detail.thread.machineId);
        setMachine(machineDetail.machine);
      } catch {
        setMachine(null);
      }
    } catch (detailError) {
      setError(detailError instanceof Error ? detailError.message : "Unable to load thread.");
      setMachine(null);
    }

    try {
      const runtimeResponse = await getThreadRuntimeSettings(threadId);
      applyRuntimeSettings(runtimeResponse.settings);
    } catch {
      setRuntimeModel("");
      setRuntimeApprovalPolicy("");
      setRuntimeSandboxMode("");
      setRuntimeModelOptions([]);
      setRuntimeSandboxOptions([]);
      if (resolvedMachineID === "") {
        setMachine(null);
      }
    }
  }, [applyRuntimeSettings, enabled, threadId]);

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

    return connectConsoleSocket(threadId, (event) => {
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

      if (isTimelineEventEnvelope(envelope)) {
        const timelineEvent = envelope.payload.event;
        const timelineTurnId = timelineEvent.turnId?.trim() ?? "";
        if (timelineTurnId) {
          timelineTurnIdsRef.current.add(timelineTurnId);
        }

        if (timelineEvent.eventType === "turn.started" && timelineTurnId) {
          completedTurnIdsRef.current.delete(timelineTurnId);
          setIsTurnPending(false);
          setActiveTurnId(timelineTurnId);
          bindPendingAgentMessageToTurn(timelineTurnId);
        } else if (timelineTurnId && !timelineEventMarksTurnCompleted(timelineEvent)) {
          completedTurnIdsRef.current.delete(timelineTurnId);
          setIsTurnPending(false);
          setActiveTurnId((current) => current ?? timelineTurnId);
          bindPendingAgentMessageToTurn(timelineTurnId);
        }

        if (timelineEventMarksTurnCompleted(timelineEvent) && timelineTurnId) {
          completedTurnIdsRef.current.add(timelineTurnId);
          setIsTurnPending(false);
          setActiveTurnId((current) => (current === timelineTurnId ? null : current));
          const pendingAgentMessageId = pendingAgentMessageIdRef.current;
          if (pendingAgentMessageId) {
            pendingAgentMessageIdRef.current = null;
            setMessages((current) =>
              current.filter((message) => message.id !== pendingAgentMessageId),
            );
          }
        }

        const approval = timelineEventToApprovalRequired(timelineEvent);
        if (approval) {
          setPendingApprovals((current) => {
            const next = current.filter((item) => item.requestId !== approval.requestId);
            next.push({ ...approval, requestId: approval.requestId });
            return next;
          });
          setApprovalAnswers((current) => ({
            ...current,
            [approval.requestId]: {
              ...(current[approval.requestId] ?? {}),
              ...buildDefaultApprovalAnswers(approval.questions),
            },
          }));
        } else if (
          timelineEvent.eventType === "approval.resolved" &&
          timelineEvent.approval?.requestId
        ) {
          const requestId = timelineEvent.approval.requestId;
          setPendingApprovals((current) => current.filter((item) => item.requestId !== requestId));
          setApprovalAnswers((current) => {
            const next = { ...current };
            delete next[requestId];
            return next;
          });
        }

        setMessages((current) => mergeTimelineEventIntoMessages(current, timelineEvent));
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
        if (timelineTurnIdsRef.current.has(payload.turnId)) {
          return;
        }
        appendTurnDelta(payload);
        return;
      }

      if (envelope.name === "turn.started") {
        const payload = envelope.payload as TurnStartedPayload;
        if (timelineTurnIdsRef.current.has(payload.turnId)) {
          return;
        }
        completedTurnIdsRef.current.delete(payload.turnId);
        setIsTurnPending(false);
        setActiveTurnId(payload.turnId);
        const pendingAgentMessageId = pendingAgentMessageIdRef.current;
        if (pendingAgentMessageId) {
          pendingAgentMessageIdRef.current = null;
          setMessages((current) =>
            current.map((message) =>
              message.id === pendingAgentMessageId
                ? withoutPendingFlag({ ...message, turnId: payload.turnId })
                : message,
            ),
          );
        }
        return;
      }

      if (envelope.name === "turn.completed" || envelope.name === "turn.failed") {
        const payload = envelope.payload as TurnCompletedPayload;
        if (timelineTurnIdsRef.current.has(payload.turn.turnId)) {
          return;
        }
        completedTurnIdsRef.current.add(payload.turn.turnId);
        setActiveTurnId((current) => (current === payload.turn.turnId ? null : current));
        setMessages((current) =>
          current.map((message) =>
            message.kind === "agent" && message.turnId === payload.turn.turnId
              ? withOptionalProgressText(
                  withoutPendingFlag(message),
                  cleanTransientProgressText(message.progressText),
                )
              : message,
          ),
        );
        if (envelope.name === "turn.failed") {
          setMessages((current) => [...current, toTurnFailedMessage(payload)]);
        }
      }
    });
  }, [appendTurnDelta, bindPendingAgentMessageToTurn, enabled, threadId, connectionIdentity]);

  const handleApprovalDecision = useCallback(
    async (requestId: string, decision: ApprovalDecision, answers?: ApprovalAnswerMap) => {
      if (!enabled || !supportsCapability("approvals")) {
        return;
      }

      setError(null);

      try {
        await respondToApproval(requestId, decision, answers);
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
    if (
      !enabled ||
      !threadId ||
      input === "" ||
      Boolean(activeTurnId) ||
      !supportsCapability("startTurn")
    ) {
      return;
    }

    setIsSubmitting(true);
    setIsTurnPending(true);
    setError(null);
    const messageSeed = `${Date.now()}`;
    const pendingAgentMessageId = `pending-agent:${messageSeed}`;
    pendingAgentMessageIdRef.current = pendingAgentMessageId;
    setMessages((current) => [
      ...current,
      {
        id: `user:${messageSeed}`,
        text: input,
        kind: "user",
      },
      {
        id: pendingAgentMessageId,
        text: "",
        kind: "agent",
        isPending: true,
      },
    ]);
    setPrompt("");

    try {
      const response = await startThreadTurn(threadId, input);
      setIsTurnPending(false);
      if (!completedTurnIdsRef.current.has(response.turn.turnId)) {
        setActiveTurnId((current) => current ?? response.turn.turnId);
        if (pendingAgentMessageIdRef.current === pendingAgentMessageId) {
          pendingAgentMessageIdRef.current = null;
          setMessages((current) =>
            current.map((message) =>
              message.id === pendingAgentMessageId
                ? withoutPendingFlag({ ...message, turnId: response.turn.turnId })
                : message,
            ),
          );
        }
      }
    } catch (submissionError) {
      setIsTurnPending(false);
      if (pendingAgentMessageIdRef.current === pendingAgentMessageId) {
        pendingAgentMessageIdRef.current = null;
        setMessages((current) => current.filter((message) => message.id !== pendingAgentMessageId));
      }
      setError(
        submissionError instanceof Error ? submissionError.message : "Unable to start turn.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }, [activeTurnId, enabled, prompt, threadId]);

  const handleRuntimeModelChange = useCallback(
    async (nextModel: string) => {
      const model = nextModel.trim();
      if (!enabled || !threadId || model === "") {
        return;
      }

      setIsUpdatingRuntimeSettings(true);
      setError(null);
      try {
        const response = await updateThreadRuntimeSettings(threadId, { model });
        applyRuntimeSettings(response.settings);
      } catch (runtimeError) {
        setError(
          runtimeError instanceof Error
            ? runtimeError.message
            : "Unable to update thread runtime model.",
        );
      } finally {
        setIsUpdatingRuntimeSettings(false);
      }
    },
    [applyRuntimeSettings, enabled, threadId],
  );

  const handleRuntimePermissionChange = useCallback(
    async (nextSandboxMode: string) => {
      const sandboxMode = nextSandboxMode.trim();
      if (!enabled || !threadId || sandboxMode === "") {
        return;
      }

      setIsUpdatingRuntimeSettings(true);
      setError(null);
      try {
        const response = await updateThreadRuntimeSettings(threadId, { sandboxMode });
        applyRuntimeSettings(response.settings);
      } catch (runtimeError) {
        setError(
          runtimeError instanceof Error
            ? runtimeError.message
            : "Unable to update thread runtime permission.",
        );
      } finally {
        setIsUpdatingRuntimeSettings(false);
      }
    },
    [applyRuntimeSettings, enabled, threadId],
  );

  const handleSteerSubmit = useCallback(async () => {
    const input = steerPrompt.trim();
    if (!enabled || !threadId || !activeTurnId || input === "" || !supportsCapability("steerTurn")) {
      return;
    }

    setError(null);

    try {
      await steerThreadTurn(threadId, activeTurnId, input);
      setSteerPrompt("");
    } catch (steerError) {
      setError(steerError instanceof Error ? steerError.message : "Unable to steer turn.");
    }
  }, [activeTurnId, enabled, steerPrompt, threadId]);

  const handleInterrupt = useCallback(async () => {
    if (!enabled || !threadId || !activeTurnId || !supportsCapability("interruptTurn")) {
      return;
    }

    setError(null);

    try {
      await interruptThreadTurn(threadId, activeTurnId);
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
    title: threadTitle,
    subtitle: threadId || "未选择线程",
    error,
    machine: machine
      ? {
          statusLabel: formatMachineStatus(machine.status as "online" | "offline" | "reconnecting" | "unknown"),
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
    isExecuting: isTurnPending || Boolean(activeTurnId),
    runtimeModel,
    runtimeApprovalPolicy,
    runtimeSandboxMode,
    runtimeModelOptions,
    runtimeSandboxOptions,
    isUpdatingRuntimeSettings,
    canStartTurn: enabled && supportsCapability("startTurn") && !activeTurnId && !isTurnPending,
    canSteerTurn: enabled && supportsCapability("steerTurn") && Boolean(activeTurnId),
    canInterruptTurn: enabled && supportsCapability("interruptTurn") && Boolean(activeTurnId),
    setPrompt,
    setSteerPrompt,
    handlePromptSubmit,
    handleRuntimeModelChange,
    handleRuntimePermissionChange,
    handleSteerSubmit,
    handleInterrupt,
    handleApprovalAnswerChange,
    handleApprovalDecision,
  };
}
