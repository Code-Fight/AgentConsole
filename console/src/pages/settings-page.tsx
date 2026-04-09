import { useEffect, useMemo, useState } from "react";
import { http } from "../common/api/http";
import type {
  AgentConfigDocument,
  AgentDescriptor,
  AgentListResponse,
  AgentType,
  MachineAgentConfigAssignment,
  MachineListResponse,
  MachineSummary,
} from "../common/api/types";

function emptyDocument(agentType: AgentType): AgentConfigDocument {
  return {
    agentType,
    format: "toml",
    content: "",
  };
}

function validateTOML(content: string): boolean {
  const trimmed = content.trim();
  if (trimmed === "") {
    throw new Error("empty");
  }
  if (/\[\s*$/.test(trimmed)) {
    throw new Error("unterminated array or table");
  }
  return true;
}

export function SettingsPage() {
  const [agents, setAgents] = useState<AgentDescriptor[]>([]);
  const [machines, setMachines] = useState<MachineSummary[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<AgentType | null>(null);
  const [selectedMachineId, setSelectedMachineId] = useState<string | null>(null);
  const [globalDocument, setGlobalDocument] = useState<AgentConfigDocument | null>(null);
  const [machineOverride, setMachineOverride] = useState<AgentConfigDocument | null>(null);
  const [usesGlobalDefault, setUsesGlobalDefault] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const machineLabelById = useMemo(() => {
    const entries = new Map<string, string>();
    for (const machine of machines) {
      entries.set(machine.id, machine.name || machine.id);
    }
    return entries;
  }, [machines]);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const [agentsResponse, machinesResponse] = await Promise.all([
          http<AgentListResponse>("/settings/agents"),
          http<MachineListResponse>("/machines")
        ]);
        if (cancelled) {
          return;
        }

        setAgents(agentsResponse.items);
        setMachines(machinesResponse.items);
        setSelectedAgent((current) => current ?? agentsResponse.items[0]?.agentType ?? null);
        setSelectedMachineId((current) => current ?? machinesResponse.items[0]?.id ?? null);
        setError(null);
      } catch {
        if (!cancelled) {
          setError("Unable to load settings.");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!selectedAgent) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const globalResponse = await http<{ document: AgentConfigDocument | null }>(
          `/settings/agents/${encodeURIComponent(selectedAgent)}/global`
        );
        if (!cancelled) {
          setGlobalDocument(globalResponse.document ?? emptyDocument(selectedAgent));
        }
      } catch {
        if (!cancelled) {
          setError("Unable to load global settings.");
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [selectedAgent]);

  useEffect(() => {
    if (!selectedAgent || !selectedMachineId) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const assignment = await http<MachineAgentConfigAssignment>(
          `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}`
        );
        if (cancelled) {
          return;
        }
        setMachineOverride(assignment.machineOverride ?? emptyDocument(selectedAgent));
        setUsesGlobalDefault(assignment.usesGlobalDefault);
        setError(null);
      } catch {
        if (!cancelled) {
          setError("Unable to load machine settings.");
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [selectedAgent, selectedMachineId]);

  async function saveGlobalDefault() {
    if (!selectedAgent || !globalDocument) {
      return;
    }

    try {
      validateTOML(globalDocument.content);
    } catch {
      setError("Invalid TOML content.");
      return;
    }

    setError(null);
    const response = await http<{ document: AgentConfigDocument }>(
      `/settings/agents/${encodeURIComponent(selectedAgent)}/global`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ content: globalDocument.content })
      }
    );
    setGlobalDocument(response.document);
    setStatusMessage("Global default saved.");
  }

  async function saveMachineOverride() {
    if (!selectedAgent || !selectedMachineId || !machineOverride) {
      return;
    }

    try {
      validateTOML(machineOverride.content);
    } catch {
      setError("Invalid TOML content.");
      return;
    }

    setError(null);
    const response = await http<{ document: AgentConfigDocument }>(
      `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ content: machineOverride.content })
      }
    );
    setMachineOverride(response.document);
    setUsesGlobalDefault(false);
    setStatusMessage("Machine override saved.");
  }

  async function deleteMachineOverride() {
    if (!selectedAgent || !selectedMachineId) {
      return;
    }

    setError(null);
    await http<void>(
      `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}`,
      {
        method: "DELETE"
      }
    );
    setMachineOverride(emptyDocument(selectedAgent));
    setUsesGlobalDefault(true);
    setStatusMessage("Machine override deleted.");
  }

  async function applyToMachine() {
    if (!selectedAgent || !selectedMachineId) {
      return;
    }

    setError(null);
    const response = await http<{ machineId: string; agentType: string; source: string; filePath?: string }>(
      `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}/apply`,
      {
        method: "POST"
      }
    );
    setStatusMessage(`Applied ${response.source} config to ${machineLabelById.get(selectedMachineId) ?? selectedMachineId}.`);
  }

  return (
    <section className="page">
      <header className="page-header">
        <h1>Settings</h1>
        <p>Manage agent configuration defaults and per-machine overrides.</p>
      </header>

      {isLoading ? <p>Loading settings…</p> : null}
      {error ? <p>{error}</p> : null}
      {statusMessage ? <p>{statusMessage}</p> : null}

      {!isLoading ? (
        <div className="settings-layout">
          <section className="settings-sidebar">
            <label className="config-form">
              <span>Agent</span>
              <select
                aria-label="Agent"
                value={selectedAgent ?? ""}
                onChange={(event) => setSelectedAgent(event.target.value as AgentType)}
              >
                {agents.map((agent) => (
                  <option key={agent.agentType} value={agent.agentType}>
                    {agent.displayName}
                  </option>
                ))}
              </select>
            </label>

            <div className="settings-machine-list">
              <h2>Machines</h2>
              {machines.map((machine) => (
                <button
                  key={machine.id}
                  type="button"
                  className={selectedMachineId === machine.id ? "machine-selected" : ""}
                  onClick={() => setSelectedMachineId(machine.id)}
                >
                  {machine.name || machine.id}
                </button>
              ))}
            </div>
          </section>

          <section className="config-form">
            <h2>Global Default</h2>
            <label>
              <span>Global Default TOML</span>
              <textarea
                aria-label="Global Default TOML"
                rows={16}
                value={globalDocument?.content ?? ""}
                onChange={(event) =>
                  setGlobalDocument((current) => ({
                    ...(current ?? emptyDocument(selectedAgent ?? "codex")),
                    content: event.target.value
                  }))
                }
              />
            </label>
            <button type="button" onClick={() => void saveGlobalDefault()}>
              Save Global Default
            </button>
          </section>

          <section className="config-form">
            <h2>Machine Override</h2>
            {usesGlobalDefault ? <p>Using Global Default</p> : null}
            <label>
              <span>Machine Override TOML</span>
              <textarea
                aria-label="Machine Override TOML"
                rows={16}
                value={machineOverride?.content ?? ""}
                onChange={(event) =>
                  setMachineOverride((current) => ({
                    ...(current ?? emptyDocument(selectedAgent ?? "codex")),
                    content: event.target.value
                  }))
                }
              />
            </label>
            <div className="settings-actions">
              <button type="button" onClick={() => void saveMachineOverride()}>
                Save Machine Override
              </button>
              <button type="button" onClick={() => void deleteMachineOverride()}>
                Delete Machine Override
              </button>
              <button type="button" onClick={() => void applyToMachine()}>
                Apply To Machine
              </button>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
}
