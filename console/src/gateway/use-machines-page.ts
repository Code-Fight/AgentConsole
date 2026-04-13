import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type { EventEnvelope, MachineListResponse, MachineSummary } from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";
import { supportsCapability } from "./capabilities";

export function useMachinesPage() {
  const [machines, setMachines] = useState<MachineSummary[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshNonce, setRefreshNonce] = useState(0);

  useEffect(() => {
    let cancelled = false;

    async function loadMachines() {
      try {
        const response = await http<MachineListResponse>("/machines");
        if (!cancelled) {
          setMachines(response.items);
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setMachines([]);
          setError("Unable to load machines.");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadMachines();

    return () => {
      cancelled = true;
    };
  }, [refreshNonce]);

  useEffect(
    () =>
      connectConsoleSocket(undefined, (event) => {
        let envelope: EventEnvelope | null = null;

        try {
          envelope = JSON.parse(event.data) as EventEnvelope;
        } catch {
          return;
        }

        if (envelope.name === "machine.updated") {
          setRefreshNonce((current) => current + 1);
        }
      }),
    [],
  );

  return {
    machines,
    isLoading,
    error,
    capabilities: {
      installAgent: supportsCapability("machineInstallAgent"),
      removeAgent: supportsCapability("machineRemoveAgent"),
    },
  };
}
