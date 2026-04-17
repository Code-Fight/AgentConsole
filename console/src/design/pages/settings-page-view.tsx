import type {
  AgentConfigDocument,
  AgentDescriptor,
  AgentType,
  ConsolePreferences,
  MachineSummary,
} from "../../common/api/types";

interface SettingsPageViewProps {
  agents: AgentDescriptor[];
  machines: MachineSummary[];
  selectedAgent: AgentType | null;
  selectedMachineId: string | null;
  globalDocument: AgentConfigDocument | null;
  machineOverride: AgentConfigDocument | null;
  usesGlobalDefault: boolean;
  error: string | null;
  statusMessage: string | null;
  isLoading: boolean;
  consolePreferences?: ConsolePreferences;
  isConsoleSaving?: boolean;
  capabilities: {
    editGatewayEndpoint: boolean;
    editConsoleProfile: boolean;
    editSafetyPolicy: boolean;
    globalDefault: boolean;
    machineOverride: boolean;
    applyMachine: boolean;
  };
  onSelectedAgentChange: (agent: AgentType) => void;
  onSelectedMachineIdChange: (machineId: string) => void;
  onGlobalDocumentChange: (updater: AgentConfigDocument | null | ((current: AgentConfigDocument | null) => AgentConfigDocument | null)) => void;
  onMachineOverrideChange: (updater: AgentConfigDocument | null | ((current: AgentConfigDocument | null) => AgentConfigDocument | null)) => void;
  onSaveGlobalDefault: () => void;
  onSaveMachineOverride: () => void;
  onDeleteMachineOverride: () => void;
  onApplyToMachine: () => void;
  onConsolePreferencesChange?: (patch: Partial<ConsolePreferences>) => void;
  onSaveConsolePreferences?: () => void;
  onFocusConsolePreferenceField?: (field: "profile" | "safetyPolicy") => void;
}

function CapabilityCard(props: {
  title: string;
  description: string;
  action: string;
  connected: boolean;
  onAction?: () => void;
}) {
  return (
    <article className="settings-section-card">
      <h2>{props.title}</h2>
      <p>{props.description}</p>
      <div className="settings-actions">
        <span className="hub-thread-chip">{props.connected ? "connected" : "not connected"}</span>
        <button
          type="button"
          disabled={!props.connected}
          aria-label={props.action}
          onClick={props.onAction}
        >
          {props.action}
        </button>
      </div>
    </article>
  );
}

export function SettingsPageView(props: SettingsPageViewProps) {
  const consolePreferences = props.consolePreferences ?? {
    profile: "",
    safetyPolicy: "",
    lastThreadId: "",
  };
  const onConsolePreferencesChange = props.onConsolePreferencesChange ?? (() => {});
  const onSaveConsolePreferences = props.onSaveConsolePreferences ?? (() => {});
  const isConsoleSaving = props.isConsoleSaving ?? false;
  const onFocusConsolePreferenceField = props.onFocusConsolePreferenceField ?? (() => {});

  return (
    <section className="page settings-page">
      <header className="page-header">
        <span className="page-kicker">Management</span>
        <h1>Settings</h1>
        <p>Gateway-backed agent configuration remains active here; unrelated console controls stay explicitly not connected.</p>
      </header>

      {props.error ? <p>{props.error}</p> : null}
      {props.statusMessage ? <p className="status-message-banner">{props.statusMessage}</p> : null}

      <div className="settings-section-list">
        <CapabilityCard
          title="Console Profile"
          description="Jump to the persisted Console Profile preference."
          action="Edit console profile"
          connected={props.capabilities.editConsoleProfile}
          onAction={() => onFocusConsolePreferenceField("profile")}
        />
        <CapabilityCard
          title="Safety Policy"
          description="Jump to the persisted Safety Policy preference."
          action="Edit safety policy"
          connected={props.capabilities.editSafetyPolicy}
          onAction={() => onFocusConsolePreferenceField("safetyPolicy")}
        />
      </div>

      {props.isLoading ? <p>Loading settings…</p> : null}

      {!props.isLoading ? (
        <div className="settings-layout">
          <section className="config-form">
            <div className="config-form-heading">
              <div>
                <span className="page-kicker">Console</span>
                <h2>Console settings</h2>
              </div>
            </div>
            <p className="form-hint">Saved console preferences persist across reloads and apply to this host.</p>
            <label>
              <span>Console Profile</span>
              <input
                id="console-profile"
                aria-label="Console Profile"
                type="text"
                value={consolePreferences.profile}
                onChange={(event) =>
                  onConsolePreferencesChange({ profile: event.target.value })
                }
              />
            </label>
            <label>
              <span>Safety Policy</span>
              <input
                id="console-safety-policy"
                aria-label="Safety Policy"
                type="text"
                value={consolePreferences.safetyPolicy}
                onChange={(event) =>
                  onConsolePreferencesChange({ safetyPolicy: event.target.value })
                }
              />
            </label>
            <button
              type="button"
              aria-label="Save Console Settings"
              disabled={isConsoleSaving}
              onClick={onSaveConsolePreferences}
            >
              Save Console Settings
            </button>
          </section>

          <section className="settings-sidebar">
            <div className="config-form settings-intro-card">
              <div className="config-form-heading">
                <div>
                  <span className="page-kicker">Gateway delivery</span>
                  <h2>Apply policy</h2>
                </div>
              </div>
              <p className="form-hint">Global defaults remain Gateway-backed. Machine overrides take precedence and can be pushed directly to a target runtime.</p>
              <div className="settings-actions">
                <span className="meta-pill">Global default</span>
                <span className="meta-pill">Machine override</span>
              </div>
            </div>

            <label className="config-form">
              <span className="page-kicker">Agent</span>
              <span>Target agent</span>
              <select
                aria-label="Agent"
                value={props.selectedAgent ?? ""}
                onChange={(event) => props.onSelectedAgentChange(event.target.value as AgentType)}
              >
                {props.agents.map((agent) => (
                  <option key={agent.agentType} value={agent.agentType}>
                    {agent.displayName}
                  </option>
                ))}
              </select>
            </label>

            <div className="settings-machine-list">
              <span className="page-kicker">Machines</span>
              <h2>Target machine</h2>
              {props.machines.map((machine) => (
                <button
                  key={machine.id}
                  type="button"
                  className={props.selectedMachineId === machine.id ? "machine-selected" : ""}
                  onClick={() => props.onSelectedMachineIdChange(machine.id)}
                >
                  {machine.name || machine.id}
                </button>
              ))}
            </div>
          </section>

          <section className="config-form">
            <div className="config-form-heading">
              <div>
                <span className="page-kicker">Default</span>
                <h2>Global default config</h2>
              </div>
              {props.globalDocument?.version ? <span className="session-panel-count">v{props.globalDocument.version}</span> : null}
            </div>
            <p className="form-hint">This TOML is stored by Gateway and reused for machines without an explicit override.</p>
            <label>
              <span>Global Default TOML</span>
              <textarea
                aria-label="Global Default TOML"
                rows={16}
                value={props.globalDocument?.content ?? ""}
                onChange={(event) =>
                  props.onGlobalDocumentChange((current) => ({
                    ...(current ?? { agentType: props.selectedAgent ?? "codex", format: "toml", content: "" }),
                    content: event.target.value,
                  }))
                }
              />
            </label>
            <p className="config-editor-note">This management page keeps the existing Gateway contract intact while presenting the new design surface.</p>
            <button
              type="button"
              aria-label="Save Global Default"
              disabled={!props.capabilities.globalDefault}
              onClick={props.onSaveGlobalDefault}
            >
              Save Global Default
            </button>
          </section>

          <section className="config-form">
            <div className="config-form-heading">
              <div>
                <span className="page-kicker">Override</span>
                <h2>Machine override config</h2>
              </div>
              {props.machineOverride?.version ? <span className="session-panel-count">v{props.machineOverride.version}</span> : null}
            </div>
            {props.usesGlobalDefault ? <p>Using global default</p> : null}
            <label>
              <span>Machine Override TOML</span>
              <textarea
                aria-label="Machine Override TOML"
                rows={16}
                value={props.machineOverride?.content ?? ""}
                onChange={(event) =>
                  props.onMachineOverrideChange((current) => ({
                    ...(current ?? { agentType: props.selectedAgent ?? "codex", format: "toml", content: "" }),
                    content: event.target.value,
                  }))
                }
              />
            </label>
            <p className="config-editor-note">Gateway does not merge fields. If this document is empty, the selected machine falls back to the global default.</p>
            <div className="settings-actions">
              <button
                type="button"
                aria-label="Save Machine Override"
                disabled={!props.capabilities.machineOverride}
                onClick={props.onSaveMachineOverride}
              >
                Save Machine Override
              </button>
              <button
                type="button"
                aria-label="Delete Machine Override"
                disabled={!props.capabilities.machineOverride}
                onClick={props.onDeleteMachineOverride}
              >
                Delete Machine Override
              </button>
              <button
                type="button"
                aria-label="Apply To Machine"
                disabled={!props.capabilities.applyMachine}
                onClick={props.onApplyToMachine}
              >
                Apply To Machine
              </button>
            </div>
          </section>
        </div>
      ) : null}
    </section>
  );
}
