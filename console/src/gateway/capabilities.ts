export const consoleCapabilities = {
  threadHub: true,
  threadWorkspace: true,
  approvals: true,
  startTurn: true,
  steerTurn: true,
  interruptTurn: true,
  dashboardMetrics: false,
  agentLifecycle: false,
} as const;

export type ConsoleCapability = keyof typeof consoleCapabilities;

export function supportsCapability(capability: ConsoleCapability): boolean {
  return consoleCapabilities[capability];
}
