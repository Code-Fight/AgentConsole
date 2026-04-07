export function readClientConfig() {
  return {
    machineId: process.env.MACHINE_ID ?? "mac-mini-01:codex",
    gatewayUrl: process.env.GATEWAY_URL ?? "ws://localhost:3000/client",
  };
}
