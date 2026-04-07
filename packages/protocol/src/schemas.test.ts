import { describe, expect, it } from "vitest";
import { commandEnvelopeSchema, resourceSnapshotSchema } from "./index";

describe("protocol schemas", () => {
  it("accepts a start thread command and a plugin snapshot", () => {
    expect(
      commandEnvelopeSchema.parse({
        commandId: "cmd-1",
        type: "command.startThread",
        payload: { machineId: "mac-mini-01:codex", title: "Investigate flaky test" },
      }),
    ).toMatchObject({ type: "command.startThread" });

    expect(
      resourceSnapshotSchema.parse({
        resourceId: "plugin:figma",
        kind: "plugin",
        machineId: "mac-mini-01:codex",
        displayName: "Figma Plugin",
        scope: "user",
        status: "enabled",
        source: "plugin-bundle",
        restartRequired: false,
        lastObservedAt: "2026-04-07T09:00:00.000Z",
      }),
    ).toMatchObject({ kind: "plugin" });
  });
});
