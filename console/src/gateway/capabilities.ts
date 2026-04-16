import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type { CapabilitySnapshot } from "../common/api/types";

const defaultCapabilities: CapabilitySnapshot = {
  threadHub: false,
  threadWorkspace: false,
  approvals: false,
  startTurn: false,
  steerTurn: false,
  interruptTurn: false,
  machineInstallAgent: false,
  machineRemoveAgent: false,
  environmentSyncCatalog: false,
  environmentRestartBridge: false,
  environmentOpenMarketplace: false,
  environmentMutateResources: false,
  environmentWriteMcp: false,
  environmentWriteSkills: false,
  settingsEditGatewayEndpoint: false,
  settingsEditConsoleProfile: false,
  settingsEditSafetyPolicy: false,
  settingsGlobalDefault: false,
  settingsMachineOverride: false,
  settingsApplyMachine: false,
  dashboardMetrics: false,
  agentLifecycle: false,
};

export type ConsoleCapability = keyof CapabilitySnapshot;

type CapabilityListener = (snapshot: CapabilitySnapshot) => void;

let currentSnapshot: CapabilitySnapshot = { ...defaultCapabilities };
let loadPromise: Promise<CapabilitySnapshot> | null = null;
const listeners = new Set<CapabilityListener>();

function normalizeSnapshot(snapshot: Partial<CapabilitySnapshot>): CapabilitySnapshot {
  const merged = { ...defaultCapabilities, ...snapshot } as CapabilitySnapshot;
  for (const key of Object.keys(defaultCapabilities) as ConsoleCapability[]) {
    merged[key] = Boolean(merged[key]);
  }
  return merged;
}

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
      const next = normalizeSnapshot(snapshot ?? {});
      updateSnapshot(next);
      return next;
    })
    .catch(() => currentSnapshot)
    .finally(() => {
      loadPromise = null;
    });
  return loadPromise;
}

export function useCapabilities(enabled = true): CapabilitySnapshot {
  const [snapshot, setSnapshot] = useState<CapabilitySnapshot>(currentSnapshot);

  useEffect(() => {
    const listener: CapabilityListener = (next) => setSnapshot(next);
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }, []);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void refreshCapabilities();
  }, [enabled]);

  return snapshot;
}

export function supportsCapability(capability: ConsoleCapability): boolean {
  return Boolean(currentSnapshot[capability]);
}
