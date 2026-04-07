import type { CodexAdapter } from "../codex-adapter/adapter.js";
import { buildSnapshot } from "../snapshot/snapshot-builder.js";
import { GatewaySession } from "./gateway-session.js";

type FrameSender = (frame: unknown) => void;

type ClientCommand = "register" | "heartbeat" | "snapshot.push";

export class CommandDispatcher {
  constructor(
    private readonly input: {
      machineId: string;
      session: GatewaySession;
      adapter: CodexAdapter;
      send: FrameSender;
      now: () => Date;
    },
  ) {}

  async dispatch(command: ClientCommand): Promise<void> {
    if (command === "register") {
      this.input.session.register();
      return;
    }

    if (command === "heartbeat") {
      this.input.session.heartbeat();
      return;
    }

    const snapshot = await buildSnapshot(this.input.adapter);
    this.input.send({
      type: "client.snapshot",
      machineId: this.input.machineId,
      sentAt: this.input.now().toISOString(),
      snapshot,
    });
  }
}
