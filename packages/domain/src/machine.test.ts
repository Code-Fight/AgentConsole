import { describe, expect, it } from "vitest";
import { buildMachineId, buildRuntimeId } from "./machine";

describe("machine identity helpers", () => {
  it("builds stable machine and runtime ids", () => {
    expect(buildMachineId({ hostname: "mac-mini-01", agentKind: "codex" })).toBe(
      "mac-mini-01:codex",
    );
    expect(buildRuntimeId({ machineId: "mac-mini-01:codex", runtimeKind: "codex" })).toBe(
      "mac-mini-01:codex/runtime/codex",
    );
  });
});
