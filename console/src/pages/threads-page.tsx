import { ThreadHubPage } from "../design";
import { useThreadHubContext } from "../gateway/thread-hub-context";
import { useThreadHub } from "../gateway/use-thread-hub";

export function ThreadsPage() {
  const sharedVm = useThreadHubContext();
  const fallbackVm = useThreadHub({ enabled: sharedVm === null });
  const vm = sharedVm ?? fallbackVm;

  return (
    <ThreadHubPage
      error={vm.error}
      threads={vm.threads}
      machineSuggestions={vm.machineSuggestions}
      machineCount={vm.machineCount}
      machineId={vm.machineId}
      title={vm.title}
      isSubmitting={vm.isSubmitting}
      onMachineIdChange={vm.setMachineId}
      onTitleChange={vm.setTitle}
      onCreateThread={() => void vm.handleCreateThread()}
      onResume={(threadId) => void vm.handleResume(threadId)}
      onArchive={(threadId) => void vm.handleArchive(threadId)}
      onDelete={(threadId) => void vm.handleDelete(threadId)}
    />
  );
}
