import { BrowserRouter, useLocation, useNavigate } from "react-router-dom";
import { ConsoleHostRouter } from "./console-host-router";
import { ConnectionRequiredDialog } from "./connection-required-dialog";
import { useConsoleConnectionState } from "./use-console-host";

function ConnectionGate() {
  const connection = useConsoleConnectionState();
  const navigate = useNavigate();
  const location = useLocation();

  const open = connection.status !== "ready" && location.pathname !== "/settings";

  return (
    <ConnectionRequiredDialog
      open={open}
      message={connection.message}
      onOpenSettings={() => navigate("/settings")}
    />
  );
}

export function DesignSourceAppRoot() {
  return (
    <div className="dark fixed inset-0 overflow-hidden relative">
      <BrowserRouter>
        <ConsoleHostRouter />
        <ConnectionGate />
      </BrowserRouter>
    </div>
  );
}
