import { useEffect, useMemo, useState } from "react";
import { Bot, Key, Save, Server } from "lucide-react";
import type { ConsolePreferences } from "../../common/api/types";
import { useConsoleConnectionState } from "../../design-host/use-console-host";
import {
  readGatewayConnectionFromCookies,
  saveGatewayConnectionToCookies,
} from "../../gateway/gateway-connection-store";
import { useConsolePreferences } from "../../gateway/use-console-preferences";
import { useSettingsPage } from "../../gateway/use-settings-page";

const emptyConsolePreferences: ConsolePreferences = {
  consoleUrl: "",
  apiKey: "",
  profile: "",
  safetyPolicy: "",
  lastThreadId: "",
};

export default function Settings() {
  const connection = useConsoleConnectionState();
  const vm = useSettingsPage({ enabled: connection.remoteEnabled });
  const {
    preferences,
    isLoading: preferencesLoading,
    isSaving: preferencesSaving,
    loadError: preferencesLoadError,
    saveError: preferencesSaveError,
    hasAttempted: preferencesAttempted,
    savePreferences,
  } = useConsolePreferences({ enabled: connection.remoteEnabled });
  const [draftGatewayUrl, setDraftGatewayUrl] = useState("");
  const [draftApiKey, setDraftApiKey] = useState("");
  const [draftPreferences, setDraftPreferences] = useState<ConsolePreferences>(
    emptyConsolePreferences,
  );
  const [hasDraftPreferences, setHasDraftPreferences] = useState(false);
  const [gatewayErrorMessage, setGatewayErrorMessage] = useState<string | null>(null);
  const [connectionMessage, setConnectionMessage] = useState<string | null>(null);
  const [consoleStatusMessage, setConsoleStatusMessage] = useState<string | null>(null);

  useEffect(() => {
    const localConnection = readGatewayConnectionFromCookies();
    setDraftGatewayUrl(localConnection?.gatewayUrl ?? "");
    setDraftApiKey(localConnection?.apiKey ?? "");
  }, []);

  useEffect(() => {
    if (!connection.remoteEnabled) {
      setDraftPreferences(emptyConsolePreferences);
      setHasDraftPreferences(true);
      return;
    }

    if (preferences) {
      setDraftPreferences(preferences);
      setHasDraftPreferences(true);
      return;
    }

    if (preferencesAttempted) {
      setDraftPreferences(emptyConsolePreferences);
      setHasDraftPreferences(true);
    }
  }, [connection.remoteEnabled, preferences, preferencesAttempted]);

  const combinedError =
    gatewayErrorMessage ??
    vm.error ??
    preferencesLoadError ??
    preferencesSaveError ??
    (!connection.remoteEnabled ? connection.message : null);
  const combinedStatusMessage = connectionMessage ?? consoleStatusMessage ?? vm.statusMessage;
  const combinedLoading =
    vm.isLoading || (connection.remoteEnabled && (preferencesLoading || !hasDraftPreferences));

  const selectedAgentLabel = useMemo(
    () =>
      vm.agents.find((agent) => agent.agentType === vm.selectedAgent)?.displayName ??
      vm.selectedAgent ??
      "Codex",
    [vm.agents, vm.selectedAgent],
  );

  const selectedMachineLabel = useMemo(
    () =>
      vm.machines.find((machine) => machine.id === vm.selectedMachineId)?.name ??
      vm.selectedMachineId ??
      "未选择机器",
    [vm.machines, vm.selectedMachineId],
  );

  const handleSaveGatewayConnection = () => {
    const gatewayUrl = draftGatewayUrl.trim();
    const apiKey = draftApiKey.trim();

    let parsedUrl: URL | null = null;
    try {
      parsedUrl = new URL(gatewayUrl);
    } catch {
      parsedUrl = null;
    }
    const hasValidUrl =
      parsedUrl !== null &&
      (parsedUrl.protocol === "http:" || parsedUrl.protocol === "https:");
    const hasApiKey = apiKey.length > 0;
    if (!hasValidUrl || !hasApiKey) {
      setGatewayErrorMessage("Please enter a valid Gateway URL and API key.");
      setConnectionMessage(null);
      setConsoleStatusMessage(null);
      return;
    }

    setGatewayErrorMessage(null);
    saveGatewayConnectionToCookies({
      gatewayUrl,
      apiKey,
    });
    setConsoleStatusMessage(null);
    setConnectionMessage("Gateway connection saved.");
  };

  const handleConsolePreferenceChange = (patch: Partial<ConsolePreferences>) => {
    setGatewayErrorMessage(null);
    setConnectionMessage(null);
    setConsoleStatusMessage(null);
    setDraftPreferences((current) => ({ ...current, ...patch }));
  };

  const handleSaveConsolePreferences = async () => {
    if (!connection.remoteEnabled) {
      return;
    }

    const response = await savePreferences(draftPreferences);
    if (response) {
      setConsoleStatusMessage("Console settings saved.");
    }
  };

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
            <Bot className="size-5 text-zinc-400" />
          </div>
          <div>
            <h1 aria-label="Settings" className="text-lg text-zinc-100">
              设置
            </h1>
            <p className="text-xs text-zinc-500 mt-0.5">配置 Console 偏好和 Agent 默认参数</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-5xl mx-auto space-y-6">
          {combinedError ? (
            <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-sm text-red-300">
              {combinedError}
            </div>
          ) : null}
          {combinedStatusMessage ? (
            <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl px-4 py-3 text-sm text-emerald-300">
              {combinedStatusMessage}
            </div>
          ) : null}
          {combinedLoading ? (
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-zinc-400">
              正在加载设置…
            </div>
          ) : null}

          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-zinc-800 rounded-lg">
                <Key className="size-5 text-blue-400" />
              </div>
              <div>
                <h2 className="text-base text-zinc-100">API Configuration</h2>
                <p className="text-xs text-zinc-500">管理本地 Gateway 连接配置</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="gateway-url">
                  Gateway URL
                </label>
                <input
                  id="gateway-url"
                  aria-label="Gateway URL"
                  type="text"
                  value={draftGatewayUrl}
                  onChange={(event) => {
                    setGatewayErrorMessage(null);
                    setConnectionMessage(null);
                    setDraftGatewayUrl(event.target.value);
                  }}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="gateway-api-key">
                  API Key
                </label>
                <input
                  id="gateway-api-key"
                  aria-label="Gateway API Key"
                  type="password"
                  value={draftApiKey}
                  onChange={(event) => {
                    setGatewayErrorMessage(null);
                    setConnectionMessage(null);
                    setDraftApiKey(event.target.value);
                  }}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>

              <div className="flex items-center justify-end pt-2">
                <button
                  type="button"
                  aria-label="Save Gateway Connection"
                  onClick={handleSaveGatewayConnection}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
                >
                  <Save className="size-4" />
                  保存 Gateway 连接
                </button>
              </div>
            </div>
          </div>

          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-zinc-800 rounded-lg">
                <Key className="size-5 text-cyan-400" />
              </div>
              <div>
                <h2 className="text-base text-zinc-100">Console Preferences</h2>
                <p className="text-xs text-zinc-500">远程 Console 偏好配置</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="console-url">
                  Console URL
                </label>
                <input
                  id="console-url"
                  aria-label="Console URL"
                  type="text"
                  value={draftPreferences.consoleUrl}
                  onChange={(event) =>
                    handleConsolePreferenceChange({ consoleUrl: event.target.value })
                  }
                  disabled={!connection.remoteEnabled}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-cyan-500 transition-colors disabled:bg-zinc-900/60 disabled:text-zinc-500"
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="console-preferences-api-key">
                  API Key
                </label>
                <input
                  id="console-preferences-api-key"
                  aria-label="API Key"
                  type="password"
                  value={draftPreferences.apiKey}
                  onChange={(event) =>
                    handleConsolePreferenceChange({ apiKey: event.target.value })
                  }
                  disabled={!connection.remoteEnabled}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-cyan-500 transition-colors disabled:bg-zinc-900/60 disabled:text-zinc-500"
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-sm text-zinc-400 mb-2" htmlFor="console-profile">
                    Console Profile
                  </label>
                  <input
                    id="console-profile"
                    aria-label="Console Profile"
                    type="text"
                    value={draftPreferences.profile}
                    onChange={(event) =>
                      handleConsolePreferenceChange({ profile: event.target.value })
                    }
                    disabled={!connection.remoteEnabled}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-cyan-500 transition-colors disabled:bg-zinc-900/60 disabled:text-zinc-500"
                  />
                </div>

                <div>
                  <label className="block text-sm text-zinc-400 mb-2" htmlFor="console-safety-policy">
                    Safety Policy
                  </label>
                  <input
                    id="console-safety-policy"
                    aria-label="Safety Policy"
                    type="text"
                    value={draftPreferences.safetyPolicy}
                    onChange={(event) =>
                      handleConsolePreferenceChange({ safetyPolicy: event.target.value })
                    }
                    disabled={!connection.remoteEnabled}
                    className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-cyan-500 transition-colors disabled:bg-zinc-900/60 disabled:text-zinc-500"
                  />
                </div>
              </div>

              <div className="flex items-center justify-end pt-2">
                <button
                  type="button"
                  aria-label="Save Console Settings"
                  disabled={!connection.remoteEnabled || preferencesSaving}
                  onClick={() => void handleSaveConsolePreferences()}
                  className="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
                >
                  <Save className="size-4" />
                  保存 Console 配置
                </button>
              </div>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[260px,1fr]">
            <div className="space-y-6">
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-2 bg-zinc-800 rounded-lg">
                    <Bot className="size-5 text-violet-400" />
                  </div>
                  <div>
                    <h2 className="text-base text-zinc-100">Agent 默认配置</h2>
                    <p className="text-xs text-zinc-500">选择 Agent 类型和目标机器</p>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm text-zinc-400 mb-2">Agent 类型</label>
                    <div className="flex flex-wrap gap-2">
                      {vm.agents.map((agent) => (
                        <button
                          key={agent.agentType}
                          type="button"
                          className={`px-4 py-2 rounded-lg text-sm transition-colors ${
                            vm.selectedAgent === agent.agentType
                              ? "bg-blue-600 text-white"
                              : "bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200"
                          }`}
                          onClick={() => vm.setSelectedAgent(agent.agentType)}
                        >
                          {agent.displayName}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm text-zinc-400 mb-2">目标机器</label>
                    <div className="space-y-2">
                      {vm.machines.map((machine) => (
                        <button
                          key={machine.id}
                          type="button"
                          className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                            vm.selectedMachineId === machine.id
                              ? "bg-blue-600 text-white"
                              : "bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200"
                          }`}
                          onClick={() => vm.setSelectedMachineId(machine.id)}
                        >
                          {machine.name || machine.id}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="bg-zinc-800/70 border border-zinc-700 rounded-lg p-3 text-xs text-zinc-400 space-y-1">
                    <div className="flex items-center gap-2">
                      <Bot className="size-3.5 text-zinc-500" />
                      <span>当前 Agent: {selectedAgentLabel}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Server className="size-3.5 text-zinc-500" />
                      <span>目标机器: {selectedMachineLabel}</span>
                    </div>
                    {vm.usesGlobalDefault ? (
                      <p className="text-emerald-300">Using global default</p>
                    ) : (
                      <p className="text-blue-300">Using machine override</p>
                    )}
                  </div>
                </div>
              </div>

              <div className="bg-zinc-900/50 border border-zinc-800/50 rounded-xl p-4">
                <h3 className="text-sm text-zinc-400 mb-3">配置说明</h3>
                <div className="space-y-2 text-xs text-zinc-600">
                  <div className="flex gap-2">
                    <span className="text-blue-400">•</span>
                    <span>Global default 会作为默认配置保存到 Gateway。</span>
                  </div>
                  <div className="flex gap-2">
                    <span className="text-blue-400">•</span>
                    <span>Machine override 会覆盖当前目标机器的默认值。</span>
                  </div>
                  <div className="flex gap-2">
                    <span className="text-blue-400">•</span>
                    <span>Apply To Machine 会把当前生效配置写入目标机器。</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="space-y-6">
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-2 bg-zinc-800 rounded-lg">
                    <Bot className="size-5 text-violet-400" />
                  </div>
                  <div>
                    <h2 className="text-base text-zinc-100">Global Default</h2>
                    <p className="text-xs text-zinc-500">为所选 Agent 维护全局 TOML 默认配置</p>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm text-zinc-400 mb-2" htmlFor="global-default-toml">
                      Global Default TOML
                    </label>
                    <textarea
                      id="global-default-toml"
                      aria-label="Global Default TOML"
                      rows={14}
                      value={vm.globalDocument?.content ?? ""}
                      onChange={(event) =>
                        vm.setGlobalDocument((current) => ({
                          ...(current ?? {
                            agentType: vm.selectedAgent ?? "codex",
                            format: "toml",
                            content: "",
                          }),
                          content: event.target.value,
                        }))
                      }
                      className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-300 font-mono focus:outline-none focus:border-blue-500 transition-colors"
                    />
                  </div>

                  <div className="flex items-center justify-end">
                    <button
                      type="button"
                      aria-label="Save Global Default"
                      disabled={!vm.capabilities.globalDefault}
                      onClick={() => void vm.saveGlobalDefault()}
                      className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
                    >
                      <Save className="size-4" />
                      保存 Global Default
                    </button>
                  </div>
                </div>
              </div>

              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="p-2 bg-zinc-800 rounded-lg">
                    <Server className="size-5 text-blue-400" />
                  </div>
                  <div>
                    <h2 className="text-base text-zinc-100">Machine Override</h2>
                    <p className="text-xs text-zinc-500">为目标机器维护覆盖配置并按需下发</p>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm text-zinc-400 mb-2" htmlFor="machine-override-toml">
                      Machine Override TOML
                    </label>
                    <textarea
                      id="machine-override-toml"
                      aria-label="Machine Override TOML"
                      rows={14}
                      value={vm.machineOverride?.content ?? ""}
                      onChange={(event) =>
                        vm.setMachineOverride((current) => ({
                          ...(current ?? {
                            agentType: vm.selectedAgent ?? "codex",
                            format: "toml",
                            content: "",
                          }),
                          content: event.target.value,
                        }))
                      }
                      className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-300 font-mono focus:outline-none focus:border-blue-500 transition-colors"
                    />
                  </div>

                  <div className="flex flex-wrap items-center justify-end gap-2">
                    <button
                      type="button"
                      aria-label="Save Machine Override"
                      disabled={!vm.capabilities.machineOverride}
                      onClick={() => void vm.saveMachineOverride()}
                      className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
                    >
                      <Save className="size-4" />
                      保存 Override
                    </button>
                    <button
                      type="button"
                      aria-label="Delete Machine Override"
                      disabled={!vm.capabilities.machineOverride}
                      onClick={() => void vm.deleteMachineOverride()}
                      className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 disabled:bg-zinc-800/60 disabled:text-zinc-600 text-zinc-300 rounded-lg text-sm transition-colors"
                    >
                      删除 Override
                    </button>
                    <button
                      type="button"
                      aria-label="Apply To Machine"
                      disabled={!vm.capabilities.applyMachine}
                      onClick={() => void vm.applyToMachine()}
                      className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg text-sm transition-colors"
                    >
                      Apply To Machine
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
