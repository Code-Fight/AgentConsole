import { Outlet } from "react-router-dom";
import { ConnectionGate } from "./connection-gate";

export function AppShell() {
  return (
    <div className="dark relative size-full min-h-0 overflow-hidden bg-zinc-950 text-zinc-100">
      <Outlet />
      <ConnectionGate />
    </div>
  );
}
