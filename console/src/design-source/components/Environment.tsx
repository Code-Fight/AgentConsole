import { EnvironmentPageView } from "../../design";
import { useEnvironmentPage } from "../../gateway/use-environment-page";

export default function Environment() {
  const vm = useEnvironmentPage();

  return (
    <EnvironmentPageView
      sections={vm.sections}
      isLoading={vm.isLoading}
      error={vm.error}
      pendingActionKey={vm.pendingActionKey}
      expandedResourceKey={vm.expandedResourceKey}
      mcpForm={vm.mcpForm}
      skillForm={vm.skillForm}
      pluginForm={vm.pluginForm}
      capabilities={vm.capabilities}
      setMcpForm={vm.setMcpForm}
      setSkillForm={vm.setSkillForm}
      setPluginForm={vm.setPluginForm}
      onCloseMcpForm={vm.setMcpFormClosed}
      onCloseSkillForm={vm.setSkillFormClosed}
      onClosePluginForm={vm.setPluginFormClosed}
      onOpenCreateMcpForm={vm.openCreateMCPForm}
      onOpenCreateSkillForm={vm.openCreateSkillForm}
      onOpenInstallPluginForm={vm.openInstallPluginForm}
      onOpenEditMcpForm={vm.openEditMCPForm}
      onToggleDetails={vm.toggleDetails}
      onResourceMutation={vm.handleResourceMutation}
      onMcpSubmit={(event) => void vm.handleMCPSubmit(event)}
      onSkillSubmit={(event) => void vm.handleSkillSubmit(event)}
      onPluginInstallSubmit={(event) => void vm.handlePluginInstallSubmit(event)}
    />
  );
}
