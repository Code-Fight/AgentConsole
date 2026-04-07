import type { CodexAdapter } from "../codex-adapter/adapter.js";

export async function buildSnapshot(adapter: CodexAdapter) {
  return {
    threads: await adapter.listThreads(),
    resources: await adapter.getEnvironmentResources(),
  };
}
