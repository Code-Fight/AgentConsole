export type MachineStatus = "online" | "offline" | "unknown";

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
