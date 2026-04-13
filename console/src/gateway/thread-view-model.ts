import type {
  ApprovalQuestion,
  ApprovalRequiredPayload,
  ApprovalResolvedPayload,
  EventEnvelope,
  MachineSummary,
  ThreadStatus,
  ThreadSummary,
  TurnCompletedPayload,
  TurnDeltaPayload,
  TurnStartedPayload,
} from "../common/api/types";

export interface ThreadHubListItemViewModel {
  id: string;
  title: string;
  threadLabel: string;
  machineLabel: string;
  status: "active" | "idle" | "offline";
  statusLabel: string;
  machineRuntimeLabel: string;
}

export interface WorkspaceMessageViewModel {
  id: string;
  kind: "user" | "agent" | "system";
  text: string;
  turnId?: string;
}

export interface WorkspaceApprovalQuestionViewModel {
  id: string;
  label: string;
  value: string;
  options?: string[];
}

export interface WorkspaceApprovalCardViewModel {
  requestId: string;
  title: string;
  kind: string;
  questions: WorkspaceApprovalQuestionViewModel[];
}

export function parseEnvelope(raw: string): EventEnvelope | null {
  try {
    return JSON.parse(raw) as EventEnvelope;
  } catch {
    return null;
  }
}

export function formatThreadStatus(status: ThreadStatus): string {
  switch (status) {
    case "active":
      return "进行中";
    case "idle":
      return "空闲";
    case "systemError":
      return "异常";
    case "notLoaded":
      return "未加载";
    default:
      return "未知";
  }
}

export function formatMachineStatus(status?: MachineSummary["status"]): string {
  switch (status) {
    case "online":
      return "在线";
    case "offline":
      return "离线";
    case "reconnecting":
      return "重连中";
    default:
      return "未知";
  }
}

export function toThreadHubStatus(status: ThreadStatus): ThreadHubListItemViewModel["status"] {
  if (status === "active") {
    return "active";
  }
  if (status === "idle") {
    return "idle";
  }
  return "offline";
}

export function toThreadHubItem(
  thread: ThreadSummary,
  machines: Record<string, MachineSummary>,
): ThreadHubListItemViewModel {
  const machine = machines[thread.machineId];

  return {
    id: thread.threadId,
    title: thread.title || thread.threadId,
    threadLabel: thread.threadId,
    machineLabel: machine?.name || thread.machineId,
    status: toThreadHubStatus(thread.status),
    statusLabel: formatThreadStatus(thread.status),
    machineRuntimeLabel: `${formatMachineStatus(machine?.status)} / ${machine?.runtimeStatus ?? "unknown"}`,
  };
}

export function toWorkspaceMessage(delta: TurnDeltaPayload): WorkspaceMessageViewModel {
  return {
    id: `${delta.turnId}:${delta.sequence}`,
    kind: "agent",
    text: delta.delta,
    turnId: delta.turnId,
  };
}

export function toTurnStartedMessage(payload: TurnStartedPayload): WorkspaceMessageViewModel {
  return {
    id: `started:${payload.turnId}`,
    kind: "system",
    text: `Turn started: ${payload.turnId}`,
    turnId: payload.turnId,
  };
}

export function toTurnCompletedMessage(payload: TurnCompletedPayload): WorkspaceMessageViewModel {
  return {
    id: `completed:${payload.turn.turnId}`,
    kind: "system",
    text: `Turn ${payload.turn.turnId} ${payload.turn.status}`,
    turnId: payload.turn.turnId,
  };
}

export function buildDefaultApprovalAnswers(questions?: ApprovalQuestion[]): Record<string, string> {
  if (!questions?.length) {
    return {};
  }

  return Object.fromEntries(
    questions.map((question) => [question.id, question.options?.[0] ?? ""]),
  );
}

export function toApprovalCardViewModel(
  approval: ApprovalRequiredPayload,
  answers: Record<string, string>,
): WorkspaceApprovalCardViewModel {
  return {
    requestId: approval.requestId,
    title: approval.command || approval.reason || approval.kind,
    kind: approval.kind,
    questions: (approval.questions ?? []).map((question) => ({
      id: question.id,
      label: question.text || question.header || question.id,
      value: answers[question.id] ?? "",
      options: question.options,
    })),
  };
}

export function getEnvelopeThreadId(envelope: EventEnvelope): string | null {
  if (envelope.name === "turn.delta") {
    return (envelope.payload as TurnDeltaPayload).threadId;
  }

  if (envelope.name === "turn.started") {
    return (envelope.payload as TurnStartedPayload).threadId;
  }

  if (envelope.name === "turn.completed" || envelope.name === "turn.failed") {
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
