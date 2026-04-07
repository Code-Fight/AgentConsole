export type EnvironmentKind = "skill" | "mcp" | "plugin";

export type EnvironmentResource = {
  resourceId: string;
  machineId: string;
  kind: EnvironmentKind;
  displayName: string;
  status: "enabled" | "disabled" | "auth_required" | "error" | "unknown";
};
