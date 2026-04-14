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
      capabilities={vm.capabilities}
      setMcpForm={vm.setMcpForm}
      onCloseMcpForm={vm.setMcpFormClosed}
      onOpenCreateMcpForm={vm.openCreateMCPForm}
      onOpenEditMcpForm={vm.openEditMCPForm}
      onToggleDetails={vm.toggleDetails}
      onResourceMutation={vm.handleResourceMutation}
      onMcpSubmit={(event) => void vm.handleMCPSubmit(event)}
    />
  );
}
