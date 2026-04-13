import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type { CapabilitySnapshot } from "../common/api/types";

const defaultCapabilities: CapabilitySnapshot = {
  threadHub: true,
  threadWorkspace: true,
  approvals: true,
  startTurn: true,
  steerTurn: true,
  interruptTurn: true,
  machineInstallAgent: false,
  machineRemoveAgent: false,
  environmentSyncCatalog: false,
  environmentRestartBridge: false,
  environmentOpenMarketplace: false,
  environmentMutateResources: true,
  environmentWriteMcp: true,
  settingsEditGatewayEndpoint: false,
  settingsEditConsoleProfile: false,
  settingsEditSafetyPolicy: false,
  settingsGlobalDefault: true,
  settingsMachineOverride: true,
  settingsApplyMachine: true,
  dashboardMetrics: false,
  agentLifecycle: false,
};

export type ConsoleCapability = keyof CapabilitySnapshot;

type CapabilityListener = (snapshot: CapabilitySnapshot) => void;

let currentSnapshot: CapabilitySnapshot = { ...defaultCapabilities };
let loadPromise: Promise<CapabilitySnapshot> | null = null;
const listeners = new Set<CapabilityListener>();

function updateSnapshot(snapshot: CapabilitySnapshot) {
  currentSnapshot = snapshot;
  listeners.forEach((listener) => listener(snapshot));
}

export async function refreshCapabilities(): Promise<CapabilitySnapshot> {
  if (loadPromise) {
    return loadPromise;
  }
  loadPromise = http<CapabilitySnapshot>("/capabilities")
    .then((snapshot) => {
      updateSnapshot(snapshot);
      return snapshot;
    })
    .catch(() => currentSnapshot)
    .finally(() => {
      loadPromise = null;
    });
  return loadPromise;
}

export function useCapabilities(): CapabilitySnapshot {
  const [snapshot, setSnapshot] = useState<CapabilitySnapshot>(currentSnapshot);

  useEffect(() => {
    const listener: CapabilityListener = (next) => setSnapshot(next);
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }, []);

  useEffect(() => {
    void refreshCapabilities();
  }, []);

  return snapshot;
}

export function supportsCapability(capability: ConsoleCapability): boolean {
  return currentSnapshot[capability];
}
