import type { CodexAdapter } from "./adapter.js";

export class FakeCodexAdapter implements CodexAdapter {
  async listThreads() {
    return [];
  }

  async getEnvironmentResources() {
    return [];
  }
}
