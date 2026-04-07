import { z } from "zod";

export const resourceKindSchema = z.enum(["skill", "mcp", "plugin"]);
export const resourceSnapshotSchema = z.object({
  resourceId: z.string(),
  kind: resourceKindSchema,
  machineId: z.string(),
  displayName: z.string(),
  scope: z.enum(["system", "user", "repo", "project-config", "plugin-bundled"]),
  status: z.enum(["enabled", "disabled", "auth_required", "error", "unknown"]),
  source: z.enum(["builtin", "curated", "local-path", "config-entry", "plugin-bundle"]),
  restartRequired: z.boolean(),
  lastObservedAt: z.string().datetime(),
});
