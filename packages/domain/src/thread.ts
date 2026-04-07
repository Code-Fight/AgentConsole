export type ThreadStatus = "created" | "ready" | "running" | "waiting_input" | "archived";

export type ThreadSummary = {
  threadId: string;
  machineId: string;
  title: string;
  status: ThreadStatus;
};
