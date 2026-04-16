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
import { supportsCapability } from "./capabilities";

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

interface UseSettingsPageOptions {
  enabled?: boolean;
}

export function useSettingsPage(options?: UseSettingsPageOptions) {
  const enabled = options?.enabled ?? true;
  const [agents, setAgents] = useState<AgentDescriptor[]>([]);
  const [machines, setMachines] = useState<MachineSummary[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<AgentType | null>(null);
  const [selectedMachineId, setSelectedMachineId] = useState<string | null>(null);
  const [globalDocument, setGlobalDocument] = useState<AgentConfigDocument | null>(null);
  const [machineOverride, setMachineOverride] = useState<AgentConfigDocument | null>(null);
  const [usesGlobalDefault, setUsesGlobalDefault] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(enabled);

  const machineLabelById = useMemo(() => {
    const entries = new Map<string, string>();
    for (const machine of machines) {
      entries.set(machine.id, machine.name || machine.id);
    }
    return entries;
  }, [machines]);

  useEffect(() => {
    if (!enabled) {
      setAgents([]);
      setMachines([]);
      setSelectedAgent(null);
      setSelectedMachineId(null);
      setGlobalDocument(null);
      setMachineOverride(null);
      setUsesGlobalDefault(true);
      setError(null);
      setStatusMessage(null);
      setIsLoading(false);
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const [agentsResponse, machinesResponse] = await Promise.all([
          http<AgentListResponse>("/settings/agents"),
          http<MachineListResponse>("/machines"),
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
  }, [enabled]);

  useEffect(() => {
    if (!enabled || !selectedAgent) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const globalResponse = await http<{ document: AgentConfigDocument | null }>(
          `/settings/agents/${encodeURIComponent(selectedAgent)}/global`,
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
  }, [enabled, selectedAgent]);

  useEffect(() => {
    if (!enabled || !selectedAgent || !selectedMachineId) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const assignment = await http<MachineAgentConfigAssignment>(
          `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}`,
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
  }, [enabled, selectedAgent, selectedMachineId]);

  async function saveGlobalDefault() {
    if (
      !enabled ||
      !selectedAgent ||
      !globalDocument ||
      !supportsCapability("settingsGlobalDefault")
    ) {
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
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: globalDocument.content }),
      },
    );
    setGlobalDocument(response.document);
    setStatusMessage("Global default saved.");
  }

  async function saveMachineOverride() {
    if (
      !enabled ||
      !selectedAgent ||
      !selectedMachineId ||
      !machineOverride ||
      !supportsCapability("settingsMachineOverride")
    ) {
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
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: machineOverride.content }),
      },
    );
    setMachineOverride(response.document);
    setUsesGlobalDefault(false);
    setStatusMessage("Machine override saved.");
  }

  async function deleteMachineOverride() {
    if (
      !enabled ||
      !selectedAgent ||
      !selectedMachineId ||
      !supportsCapability("settingsMachineOverride")
    ) {
      return;
    }

    setError(null);
    await http<void>(
      `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}`,
      {
        method: "DELETE",
      },
    );
    setMachineOverride(emptyDocument(selectedAgent));
    setUsesGlobalDefault(true);
    setStatusMessage("Machine override deleted.");
  }

  async function applyToMachine() {
    if (
      !enabled ||
      !selectedAgent ||
      !selectedMachineId ||
      !supportsCapability("settingsApplyMachine")
    ) {
      return;
    }

    setError(null);
    const response = await http<{ machineId: string; agentType: string; source: string; filePath?: string }>(
      `/settings/machines/${encodeURIComponent(selectedMachineId)}/agents/${encodeURIComponent(selectedAgent)}/apply`,
      {
        method: "POST",
      },
    );
    setStatusMessage(
      `Applied ${response.source === "machine" ? "machine override" : "global default"} to ${machineLabelById.get(selectedMachineId) ?? selectedMachineId}.`,
    );
  }

  return {
    agents,
    machines,
    selectedAgent,
    selectedMachineId,
    globalDocument,
    machineOverride,
    usesGlobalDefault,
    error,
    statusMessage,
    isLoading,
    setSelectedAgent,
    setSelectedMachineId,
    setGlobalDocument,
    setMachineOverride,
    saveGlobalDefault,
    saveMachineOverride,
    deleteMachineOverride,
    applyToMachine,
    capabilities: {
      editGatewayEndpoint: enabled && supportsCapability("settingsEditGatewayEndpoint"),
      editConsoleProfile: enabled && supportsCapability("settingsEditConsoleProfile"),
      editSafetyPolicy: enabled && supportsCapability("settingsEditSafetyPolicy"),
      globalDefault: enabled && supportsCapability("settingsGlobalDefault"),
      machineOverride: enabled && supportsCapability("settingsMachineOverride"),
      applyMachine: enabled && supportsCapability("settingsApplyMachine"),
    },
  };
}
