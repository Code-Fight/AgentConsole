import { Outlet } from "react-router-dom";
import { ConnectionGate } from "./connection-gate";

export function AppShell() {
  return (
    <div className="dark fixed inset-0 overflow-hidden relative">
      <Outlet />
      <ConnectionGate />
    </div>
  );
}
