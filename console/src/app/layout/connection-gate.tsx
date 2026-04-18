import { useLocation, useNavigate } from "react-router-dom";
import { ConnectionRequiredDialog } from "../../design-host/connection-required-dialog";
import { useConsoleConnectionState } from "../../design-host/use-console-host";

export function ConnectionGate() {
  const connection = useConsoleConnectionState();
  const navigate = useNavigate();
  const location = useLocation();

  const isSettingsRoute =
    location.pathname === "/settings" || location.pathname.startsWith("/settings/");
  const open = connection.status !== "ready" && !isSettingsRoute;

  return (
    <ConnectionRequiredDialog
      open={open}
      message={connection.message}
      onOpenSettings={() => navigate("/settings")}
    />
  );
}
