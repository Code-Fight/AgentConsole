export type MachineStatus = "online" | "offline" | "reconnecting" | "unknown";
export type ThreadStatus = "notLoaded" | "idle" | "active" | "unknown" | "systemError";
export type TurnStatus = "completed" | "interrupted" | "failed";
export type EventCategory = "system" | "command" | "event" | "snapshot";
export type EnvironmentKind = "skill" | "mcp" | "plugin";
export type ApprovalDecision = "accept" | "decline" | "cancel";
export type EnvironmentResourceStatus =
  | "unknown"
  | "enabled"
  | "disabled"
  | "auth_required"
  | "error";

export interface MachineSummary {
  id: string;
  name: string;
  status: MachineStatus;
}

export interface MachineListResponse {
  items: MachineSummary[];
}

export interface EnvironmentResource {
  resourceId: string;
  machineId: string;
  kind: EnvironmentKind;
  displayName: string;
  status: EnvironmentResourceStatus;
  restartRequired: boolean;
  lastObservedAt: string;
}

export interface EnvironmentListResponse {
  items: EnvironmentResource[];
}

export interface ThreadSummary {
  threadId: string;
  machineId: string;
  status: ThreadStatus;
  title: string;
}

export interface TurnSummary {
  turnId: string;
  threadId: string;
  status: TurnStatus;
}

export interface ThreadListResponse {
  items: ThreadSummary[];
}

export interface StartTurnResponse {
  turn: {
    turnId: string;
    threadId: string;
  };
}

export interface TurnDeltaPayload {
  threadId: string;
  turnId: string;
  sequence: number;
  delta: string;
}

export interface TurnCompletedPayload {
  turn: TurnSummary;
}

export interface ApprovalRequiredPayload {
  requestId: string;
  threadId?: string;
  turnId?: string;
  itemId?: string;
  kind: string;
  reason?: string;
  command?: string;
}

export interface ApprovalResolvedPayload {
  requestId: string;
  threadId?: string;
  decision: ApprovalDecision;
}

export interface EventEnvelope<TPayload = unknown> {
  version: string;
  category: EventCategory;
  name: string;
  timestamp: string;
  payload: TPayload;
  requestId?: string;
  machineId?: string;
}
