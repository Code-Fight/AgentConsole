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
} from "../../../common/api/types";

export type ThreadShellDestination = "overview" | "machines" | "environment" | "settings";

export interface ThreadHubListItemViewModel {
  id: string;
  title: string;
  threadLabel: string;
  machineLabel: string;
  status: "active" | "idle" | "offline";
  statusLabel: string;
  machineRuntimeLabel: string;
}

export interface ThreadAgentViewModel {
  id: string;
  name: string;
  type: "codex";
  model: string;
  status: "active" | "idle" | "offline";
  port: number;
}

export interface ThreadSessionViewModel {
  id: string;
  title: string;
  agentName: string;
  model: string;
  status: ThreadSummary["status"];
  lastActivity: string;
  messages: Array<{
    id: string;
    role: "user" | "agent";
    content: string;
    timestamp: string;
    fileChanges?: Array<{
      path: string;
      additions: number;
      deletions: number;
    }>;
    terminalOutput?: string;
  }>;
}

export interface ThreadMachineViewModel {
  id: string;
  name: string;
  status: MachineSummary["status"];
  runtimeStatus: MachineSummary["runtimeStatus"];
  agents: ThreadAgentViewModel[];
  sessions: ThreadSessionViewModel[];
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

function machineSortLabel(machine: MachineSummary): string {
  const name = machine.name?.trim();
  if (name) {
    return name.toLowerCase();
  }
  return machine.id.toLowerCase();
}

function threadActivityEpoch(thread: ThreadSummary): number {
  const raw = thread.lastActivityAt?.trim();
  if (!raw) {
    return 0;
  }
  const epoch = Date.parse(raw);
  return Number.isNaN(epoch) ? 0 : epoch;
}

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

function formatActivityTimestamp(raw: string): string {
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw;
  }

  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())} ${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`;
}

function formatLastActivity(thread: ThreadSummary): string {
  if (thread.lastActivityAt?.trim()) {
    return formatActivityTimestamp(thread.lastActivityAt.trim());
  }
  return formatThreadStatus(thread.status);
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

export function buildThreadMachines(
  threadSummaries: ThreadSummary[],
  machineSummaries: MachineSummary[],
): ThreadMachineViewModel[] {
  const machineById = new Map<string, MachineSummary>();

  machineSummaries.forEach((machine) => machineById.set(machine.id, machine));
  threadSummaries.forEach((thread) => {
    if (!machineById.has(thread.machineId)) {
      machineById.set(thread.machineId, {
        id: thread.machineId,
        name: thread.machineId,
        status: "unknown",
        runtimeStatus: "unknown",
        agents: [],
      });
    }
  });

  return Array.from(machineById.values())
    .sort((left, right) => {
      const leftLabel = machineSortLabel(left);
      const rightLabel = machineSortLabel(right);
      if (leftLabel !== rightLabel) {
        return leftLabel.localeCompare(rightLabel);
      }
      return left.id.localeCompare(right.id);
    })
    .map((machine) => {
      const machineThreads = threadSummaries
        .filter((thread) => thread.machineId === machine.id)
        .sort((left, right) => {
          const leftActivity = threadActivityEpoch(left);
          const rightActivity = threadActivityEpoch(right);
          if (leftActivity !== rightActivity) {
            return rightActivity - leftActivity;
          }
          return left.threadId.localeCompare(right.threadId);
        });
    const agents: ThreadAgentViewModel[] = (machine.agents ?? []).map((agent) => {
      const agentThreads = machineThreads.filter((thread) => thread.agentId === agent.agentId);
      const hasActiveThread = agentThreads.some((thread) => thread.status === "active");

      return {
        id: agent.agentId,
        name: agent.displayName,
        type: agent.agentType,
        model: agent.agentType,
        status:
          agent.status === "running"
            ? hasActiveThread
              ? "active"
              : "idle"
            : "offline",
        port: 0,
      };
    });

    const sessions: ThreadSessionViewModel[] = machineThreads.map((thread) => ({
      id: thread.threadId,
      title: thread.title || thread.threadId,
      agentName: agents.find((agent) => agent.id === thread.agentId)?.name ?? "Unknown agent",
      model: agents.find((agent) => agent.id === thread.agentId)?.model ?? "unknown",
      status: thread.status,
      lastActivity: formatLastActivity(thread),
      messages: [],
    }));

    return {
      id: machine.id,
      name: machine.name || machine.id,
      status: machine.status,
      runtimeStatus: machine.runtimeStatus ?? "unknown",
      agents,
      sessions,
    };
    });
}

export function findThreadSelection(
  machines: ThreadMachineViewModel[],
  threadId: string | null,
) {
  if (!threadId) {
    return {
      selectedMachine: null,
      selectedSession: null,
    };
  }

  for (const machine of machines) {
    const session = machine.sessions.find((candidate) => candidate.id === threadId);
    if (session) {
      return {
        selectedMachine: machine,
        selectedSession: session,
      };
    }
  }

  return {
    selectedMachine: null,
    selectedSession: null,
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
  if (payload.turn.status === "failed" && payload.errorMessage) {
    return {
      id: `completed:${payload.turn.turnId}`,
      kind: "system",
      text: `Turn ${payload.turn.turnId} failed: ${payload.errorMessage}`,
      turnId: payload.turn.turnId,
    };
  }

  return {
    id: `completed:${payload.turn.turnId}`,
    kind: "system",
    text: `Turn ${payload.turn.turnId} ${payload.turn.status}`,
    turnId: payload.turn.turnId,
  };
}

export function buildDefaultApprovalAnswers(
  questions?: ApprovalQuestion[],
): Record<string, string> {
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
