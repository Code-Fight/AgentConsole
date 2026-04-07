export type ThreadSummary = {
  threadId: string;
  machineId: string;
  title: string;
  status: "ready" | "running" | "waiting_input" | "archived";
};

export class ThreadStore {
  #threads = new Map<string, ThreadSummary>();

  listThreads(): ThreadSummary[] {
    return [...this.#threads.values()];
  }

  upsertThread(thread: ThreadSummary): void {
    this.#threads.set(thread.threadId, thread);
  }
}
