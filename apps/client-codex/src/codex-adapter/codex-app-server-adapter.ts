import type { CodexAdapter, CodexThread, EnvironmentResource } from "./adapter.js";

type AppServerRunner = {
  request<T>(method: string, payload?: unknown): Promise<T>;
};

export class CodexAppServerAdapter implements CodexAdapter {
  constructor(private readonly runner: AppServerRunner) {}

  async listThreads() {
    return this.runner.request<CodexThread[]>("thread/list");
  }

  async getEnvironmentResources() {
    return this.runner.request<EnvironmentResource[]>("environment/list");
  }
}
