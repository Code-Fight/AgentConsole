import { useEffect, useState } from "react";
import { SettingsPageView } from "../../design";
import type { ConsolePreferences } from "../../common/api/types";
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
  const vm = useSettingsPage();
  const {
    preferences,
    isLoading: preferencesLoading,
    isSaving: preferencesSaving,
    error: preferencesError,
    savePreferences,
  } = useConsolePreferences();
  const [draftPreferences, setDraftPreferences] = useState<ConsolePreferences>(
    emptyConsolePreferences,
  );
  const [hasDraftPreferences, setHasDraftPreferences] = useState(false);
  const [consoleStatusMessage, setConsoleStatusMessage] = useState<string | null>(null);

  useEffect(() => {
    if (preferences) {
      setDraftPreferences(preferences);
      setHasDraftPreferences(true);
      return;
    }
    if (!preferencesLoading && !preferencesError) {
      setDraftPreferences(emptyConsolePreferences);
      setHasDraftPreferences(true);
    }
  }, [preferences, preferencesLoading, preferencesError]);

  const combinedError = vm.error ?? preferencesError;
  const combinedStatusMessage = consoleStatusMessage ?? vm.statusMessage;
  const combinedLoading = vm.isLoading || preferencesLoading || !hasDraftPreferences;

  const handleConsolePreferenceChange = (patch: Partial<ConsolePreferences>) => {
    setConsoleStatusMessage(null);
    setDraftPreferences((current) => ({ ...current, ...patch }));
  };

  const handleSaveConsolePreferences = async () => {
    const response = await savePreferences(draftPreferences);
    if (response) {
      setConsoleStatusMessage("Console settings saved.");
    }
  };

  return (
    <SettingsPageView
      agents={vm.agents}
      machines={vm.machines}
      selectedAgent={vm.selectedAgent}
      selectedMachineId={vm.selectedMachineId}
      globalDocument={vm.globalDocument}
      machineOverride={vm.machineOverride}
      usesGlobalDefault={vm.usesGlobalDefault}
      error={combinedError}
      statusMessage={combinedStatusMessage}
      isLoading={combinedLoading}
      consolePreferences={draftPreferences}
      isConsoleSaving={preferencesSaving}
      capabilities={vm.capabilities}
      onSelectedAgentChange={vm.setSelectedAgent}
      onSelectedMachineIdChange={vm.setSelectedMachineId}
      onGlobalDocumentChange={vm.setGlobalDocument}
      onMachineOverrideChange={vm.setMachineOverride}
      onSaveGlobalDefault={() => void vm.saveGlobalDefault()}
      onSaveMachineOverride={() => void vm.saveMachineOverride()}
      onDeleteMachineOverride={() => void vm.deleteMachineOverride()}
      onApplyToMachine={() => void vm.applyToMachine()}
      onConsolePreferencesChange={handleConsolePreferenceChange}
      onSaveConsolePreferences={() => void handleSaveConsolePreferences()}
    />
  );
}
