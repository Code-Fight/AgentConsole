import { useEffect, useState } from "react";
import { http } from "../common/api/http";
import type { EventEnvelope, MachineListResponse, MachineSummary } from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

interface OverviewStat {
  label: string;
  value: number;
}

function buildOverviewStats(machines: MachineSummary[]): OverviewStat[] {
  return [
    {
      label: "Total machines",
      value: machines.length
    },
    {
      label: "Online",
      value: machines.filter((machine) => machine.status === "online").length
    },
    {
      label: "Offline",
      value: machines.filter((machine) => machine.status === "offline").length
    },
    {
      label: "Reconnecting",
      value: machines.filter((machine) => machine.status === "reconnecting").length
    }
  ];
}

export function OverviewPage() {
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
          setError("Unable to load machine overview.");
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

  const stats = buildOverviewStats(machines);

  return (
    <section className="page">
      <header className="page-header">
        <h1>Overview</h1>
        <p>Current machine availability across the gateway.</p>
      </header>

      {isLoading ? <p>Loading overview…</p> : null}
      {error ? <p>{error}</p> : null}

      {!isLoading && !error ? (
        <div className="stats-grid">
          {stats.map((stat) => (
            <article key={stat.label} className="stat-card">
              <span className="stat-value">{stat.value}</span>
              <span className="stat-label">{stat.label}</span>
            </article>
          ))}
        </div>
      ) : null}

      {!isLoading && !error && machines.length === 0 ? (
        <p>No machines have reported in yet.</p>
      ) : null}
    </section>
  );
}
