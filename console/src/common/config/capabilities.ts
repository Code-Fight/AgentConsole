import { useEffect, useState } from "react";
import { http } from "../api/http";
import type { CapabilitySnapshot } from "../api/types";

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

export function useCapabilities(enabled = true): CapabilitySnapshot {
  const [capabilities, setCapabilities] = useState<CapabilitySnapshot>(defaultCapabilities);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let cancelled = false;
    void http<CapabilitySnapshot>("/capabilities")
      .then((snapshot) => {
        if (cancelled) {
          return;
        }
        setCapabilities({ ...defaultCapabilities, ...snapshot });
      })
      .catch(() => {
        if (cancelled) {
          return;
        }
        setCapabilities(defaultCapabilities);
      });

    return () => {
      cancelled = true;
    };
  }, [enabled]);

  return capabilities;
}
