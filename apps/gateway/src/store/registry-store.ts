export type MachineSummary = {
  machineId: string;
  hostname: string;
  agentKind: "codex";
  status: "online" | "offline" | "unknown";
};

export class RegistryStore {
  #machines = new Map<string, MachineSummary>();

  listMachines(): MachineSummary[] {
    return [...this.#machines.values()];
  }

  upsertMachine(machine: MachineSummary): void {
    this.#machines.set(machine.machineId, machine);
  }
}
