import { describe, expect, it } from "vitest";
import { GatewaySession } from "./gateway-session.js";

describe("GatewaySession", () => {
  it("emits register and heartbeat frames", async () => {
    const frames: unknown[] = [];

    const session = new GatewaySession({
      machineId: "mac-mini-01:codex",
      send: (frame) => {
        frames.push(frame);
      },
      now: () => new Date("2026-04-07T09:00:00.000Z"),
    });

    session.register();
    session.heartbeat();

    expect(frames).toEqual([
      expect.objectContaining({ type: "client.register" }),
      expect.objectContaining({ type: "client.heartbeat" }),
    ]);
  });
});
