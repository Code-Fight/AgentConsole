import { useLocation, useNavigate } from "react-router-dom";
import { useGatewayConnectionState } from "../../common/config/gateway-connection-store";
import { ConnectionRequiredDialog } from "./connection-required-dialog";

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
