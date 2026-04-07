import type { ThreadSummary } from "@cag/domain";

export class ThreadStore {
  #threads = new Map<string, ThreadSummary>();

  listThreads(): ThreadSummary[] {
    return [...this.#threads.values()];
  }

  upsertThread(thread: ThreadSummary): void {
    this.#threads.set(thread.threadId, thread);
  }
}
