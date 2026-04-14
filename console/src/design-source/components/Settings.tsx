import { SettingsPageView } from "../../design";
import { useSettingsPage } from "../../gateway/use-settings-page";

export default function Settings() {
  const vm = useSettingsPage();

  return (
    <SettingsPageView
      agents={vm.agents}
      machines={vm.machines}
      selectedAgent={vm.selectedAgent}
      selectedMachineId={vm.selectedMachineId}
      globalDocument={vm.globalDocument}
      machineOverride={vm.machineOverride}
      usesGlobalDefault={vm.usesGlobalDefault}
      error={vm.error}
      statusMessage={vm.statusMessage}
      isLoading={vm.isLoading}
      capabilities={vm.capabilities}
      onSelectedAgentChange={vm.setSelectedAgent}
      onSelectedMachineIdChange={vm.setSelectedMachineId}
      onGlobalDocumentChange={vm.setGlobalDocument}
      onMachineOverrideChange={vm.setMachineOverride}
      onSaveGlobalDefault={() => void vm.saveGlobalDefault()}
      onSaveMachineOverride={() => void vm.saveMachineOverride()}
      onDeleteMachineOverride={() => void vm.deleteMachineOverride()}
      onApplyToMachine={() => void vm.applyToMachine()}
    />
  );
}
