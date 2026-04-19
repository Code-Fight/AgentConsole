import { useEffect, useState } from "react";
import { http } from "../api/http";
import type { CapabilitySnapshot } from "../api/types";
import {
  getGatewayConnectionIdentity,
  subscribeGatewayConnection,
} from "./gateway-connection-store";

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
let loadedIdentity: string | null = null;
let observedIdentity = getGatewayConnectionIdentity();

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

function resetSnapshot() {
  updateSnapshot({ ...defaultCapabilities });
}

subscribeGatewayConnection(() => {
  const nextIdentity = getGatewayConnectionIdentity();
  if (nextIdentity === observedIdentity) {
    return;
  }

  observedIdentity = nextIdentity;
  loadedIdentity = null;
  loadPromise = null;
  resetSnapshot();
});

export async function refreshCapabilities(): Promise<CapabilitySnapshot> {
  const identity = getGatewayConnectionIdentity();
  if (identity === "missing" || identity === "authFailed") {
    if (loadedIdentity !== identity) {
      loadedIdentity = identity;
      resetSnapshot();
    }
    return currentSnapshot;
  }

  if (loadedIdentity !== identity) {
    loadedIdentity = identity;
    resetSnapshot();
  }

  if (loadPromise) {
    return loadPromise;
  }

  const requestIdentity = identity;
  loadPromise = http<CapabilitySnapshot>("/capabilities")
    .then((snapshot) => {
      if (requestIdentity !== getGatewayConnectionIdentity()) {
        return currentSnapshot;
      }
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

export function resetCapabilitiesForTests(): void {
  loadPromise = null;
  loadedIdentity = null;
  observedIdentity = getGatewayConnectionIdentity();
  resetSnapshot();
}
