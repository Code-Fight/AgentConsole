import { readClientConfig } from "./config.js";
import { FakeCodexAdapter } from "./codex-adapter/fake-codex-adapter.js";
import { CommandDispatcher } from "./gateway/command-dispatcher.js";
import { GatewaySession } from "./gateway/gateway-session.js";

const config = readClientConfig();

const send = (frame: unknown) => {
  console.log(JSON.stringify(frame));
};

const session = new GatewaySession({
  machineId: config.machineId,
  send,
  now: () => new Date(),
});

const adapter = new FakeCodexAdapter();
const dispatcher = new CommandDispatcher({
  machineId: config.machineId,
  session,
  adapter,
  send,
  now: () => new Date(),
});

console.log(`connecting to ${config.gatewayUrl}`);
await dispatcher.dispatch("register");
await dispatcher.dispatch("heartbeat");
await dispatcher.dispatch("snapshot.push");
