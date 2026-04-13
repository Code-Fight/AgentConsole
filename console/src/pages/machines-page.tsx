import { MachinesPageView } from "../design";
import { useMachinesPage } from "../gateway/use-machines-page";

export function MachinesPage() {
  const vm = useMachinesPage();

  return (
    <MachinesPageView
      machines={vm.machines}
      isLoading={vm.isLoading}
      error={vm.error}
      capabilities={vm.capabilities}
    />
  );
}
