export type MachineStatus = "online" | "offline" | "reconnecting" | "unknown";
export type MachineRuntimeStatus = "running" | "stopped" | "unknown";
export type ThreadStatus = "notLoaded" | "idle" | "active" | "unknown" | "systemError";
export type TurnStatus = "completed" | "interrupted" | "failed";
export type EventCategory = "system" | "command" | "event" | "snapshot";
export type EnvironmentKind = "skill" | "mcp" | "plugin";
export type ApprovalDecision = "accept" | "decline" | "cancel";
export type AgentType = "codex";
export type AgentConfigFormat = "toml";
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
  runtimeStatus: MachineRuntimeStatus;
}

export interface MachineListResponse {
  items: MachineSummary[];
}

export interface MachineUpdatedPayload {
  machine: MachineSummary;
}

export interface MachineDetailResponse {
  machine: MachineSummary;
}

export interface AgentDescriptor {
  agentType: AgentType;
  displayName: string;
}

export interface AgentListResponse {
  items: AgentDescriptor[];
}

export interface AgentConfigDocument {
  agentType: AgentType;
  format: AgentConfigFormat;
  content: string;
  updatedAt?: string;
  updatedBy?: string;
  version?: number;
}

export interface MachineAgentConfigAssignment {
  machineId: string;
  agentType: AgentType;
  globalDefault?: AgentConfigDocument | null;
  machineOverride?: AgentConfigDocument | null;
  usesGlobalDefault: boolean;
}

export interface EnvironmentResource {
  resourceId: string;
  machineId: string;
  kind: EnvironmentKind;
  displayName: string;
  status: EnvironmentResourceStatus;
  restartRequired: boolean;
  lastObservedAt: string;
  details?: Record<string, unknown>;
}

export interface EnvironmentListResponse {
  items: EnvironmentResource[];
}

export interface ResourceChangedPayload {
  machineId: string;
  kind?: EnvironmentKind;
  resourceId?: string;
  resource?: EnvironmentResource;
  action?: "snapshot" | "updated" | "removed";
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

export interface CreateThreadResponse {
  thread: ThreadSummary;
}

export interface ThreadDeleteResponse {
  threadId: string;
  deleted: boolean;
  archived: boolean;
}

export interface ThreadUpdatedPayload {
  machineId: string;
  threadId?: string;
  thread?: ThreadSummary;
}

export interface ApprovalQuestion {
  id: string;
  header?: string;
  text?: string;
  options?: string[];
}

export interface ThreadDetailResponse {
  thread: ThreadSummary;
  activeTurnId?: string | null;
  pendingApprovals: ApprovalRequiredPayload[];
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

export interface TurnStartedPayload {
  threadId: string;
  turnId: string;
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
  questions?: ApprovalQuestion[];
}

export interface ApprovalResolvedPayload {
  requestId: string;
  threadId?: string;
  turnId?: string;
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
