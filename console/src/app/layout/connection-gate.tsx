import { useLocation, useNavigate } from "react-router-dom";
import { ConnectionRequiredDialog } from "../../design-host/connection-required-dialog";
import { useGatewayConnectionState } from "../../features/settings/model/gateway-connection-store";

export function ConnectionGate() {
  const connection = useGatewayConnectionState();
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
