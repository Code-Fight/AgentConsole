export type MachineStatus = "online" | "offline" | "reconnecting" | "unknown";
export type ThreadStatus = "notLoaded" | "idle" | "active" | "systemError";
export type TurnStatus = "completed" | "interrupted" | "failed";
export type EventCategory = "system" | "command" | "event" | "snapshot";

export interface MachineSummary {
  id: string;
  name: string;
  status: MachineStatus;
}

export interface OverviewSummary {
  totalMachines: number;
  onlineMachines: number;
}

export interface EnvironmentVariable {
  key: string;
  value: string;
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

export interface EventEnvelope<TPayload = unknown> {
  version: string;
  category: EventCategory;
  name: string;
  timestamp: string;
  payload: TPayload;
  requestId?: string;
  machineId?: string;
}
