export type CodexThread = {
  threadId: string;
  title: string;
  status: string;
};

export type EnvironmentResource = {
  resourceId: string;
  kind: "skill" | "mcp" | "plugin";
  displayName: string;
  status: string;
};

export interface CodexAdapter {
  listThreads(): Promise<CodexThread[]>;
  getEnvironmentResources(): Promise<EnvironmentResource[]>;
}
