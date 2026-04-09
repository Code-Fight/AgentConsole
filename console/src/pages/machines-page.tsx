import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type { EventEnvelope, MachineListResponse, MachineSummary } from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

function formatMachineStatus(status: MachineSummary["status"]): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

export function MachinesPage() {
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

  useEffect(() => connectConsoleSocket(undefined, (event) => {
    let envelope: EventEnvelope | null = null;

    try {
      envelope = JSON.parse(event.data) as EventEnvelope;
    } catch {
      return;
    }

    if (envelope.name === "machine.updated") {
      setRefreshNonce((current) => current + 1);
    }
  }), []);

  return (
    <section className="page">
      <header className="page-header">
        <h1>Machines</h1>
        <p>Connected runtimes and their current connection state.</p>
      </header>

      {isLoading ? <p>Loading machines…</p> : null}
      {error ? <p>{error}</p> : null}
      {!isLoading && !error && machines.length === 0 ? (
        <p>No machines available.</p>
      ) : null}

      {!isLoading && !error && machines.length > 0 ? (
        <div className="resource-list">
          {machines.map((machine) => (
            <article key={machine.id} className="resource-card">
              <div className="resource-card-main">
                <h2>{machine.name || machine.id}</h2>
                <p>{machine.id}</p>
              </div>
              <span className={`status-badge status-${machine.status}`}>
                {formatMachineStatus(machine.status)}
              </span>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}
