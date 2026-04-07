type FrameSender = (frame: unknown) => void;

export class GatewaySession {
  constructor(
    private readonly input: {
      machineId: string;
      send: FrameSender;
      now: () => Date;
    },
  ) {}

  register(): void {
    this.input.send({
      type: "client.register",
      machineId: this.input.machineId,
      connectedAt: this.input.now().toISOString(),
    });
  }

  heartbeat(): void {
    this.input.send({
      type: "client.heartbeat",
      machineId: this.input.machineId,
      sentAt: this.input.now().toISOString(),
    });
  }
}
