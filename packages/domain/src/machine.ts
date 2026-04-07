export function buildMachineId(input: { hostname: string; agentKind: string }): string {
  return `${input.hostname}:${input.agentKind}`;
}

export function buildRuntimeId(input: { machineId: string; runtimeKind: string }): string {
  return `${input.machineId}/runtime/${input.runtimeKind}`;
}
