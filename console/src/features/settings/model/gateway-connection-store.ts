import { useMemo, useSyncExternalStore } from "react";
import {
  clearGatewayConnectionCookies as clearLegacyGatewayConnectionCookies,
  getGatewayConnectionConfig as getLegacyGatewayConnectionConfig,
  getGatewayConnectionIdentity as getLegacyGatewayConnectionIdentity,
  getGatewayConnectionState as getLegacyGatewayConnectionState,
  readGatewayConnectionFromCookies as readLegacyGatewayConnectionFromCookies,
  saveGatewayConnectionToCookies as saveLegacyGatewayConnectionToCookies,
  subscribeGatewayConnection as subscribeLegacyGatewayConnection,
  type GatewayConnectionConfig,
  type GatewayConnectionState,
} from "../../../gateway/gateway-connection-store";

export type SettingsGatewayConnectionStatus = "unconfigured" | "ready" | "authFailed";

export interface SettingsGatewayConnectionState {
  status: SettingsGatewayConnectionStatus;
  message: string;
  remoteEnabled: boolean;
}

function mapGatewayConnectionState(
  state: GatewayConnectionState,
): SettingsGatewayConnectionState {
  if (state === "ready") {
    return {
      status: "ready",
      message: "",
      remoteEnabled: true,
    };
  }

  if (state === "authFailed") {
    return {
      status: "authFailed",
      message: "Gateway 鉴权失败，请检查 API Key 后重试。",
      remoteEnabled: false,
    };
  }

  return {
    status: "unconfigured",
    message: "请先在设置页填写 Gateway URL 与 API Key。",
    remoteEnabled: false,
  };
}

export function readGatewayConnectionFromCookies(): GatewayConnectionConfig | null {
  return readLegacyGatewayConnectionFromCookies();
}

export function saveGatewayConnectionToCookies(config: GatewayConnectionConfig): void {
  saveLegacyGatewayConnectionToCookies(config);
}

export function clearGatewayConnectionCookies(): void {
  clearLegacyGatewayConnectionCookies();
}

export function subscribeGatewayConnection(listener: () => void): () => void {
  return subscribeLegacyGatewayConnection(listener);
}

export function getGatewayConnectionIdentity(): string {
  return getLegacyGatewayConnectionIdentity();
}

export function getGatewayConnectionConfig(): GatewayConnectionConfig | null {
  return getLegacyGatewayConnectionConfig();
}

export function useGatewayConnectionState(): SettingsGatewayConnectionState {
  const gatewayState = useSyncExternalStore(
    subscribeGatewayConnection,
    getLegacyGatewayConnectionState,
    getLegacyGatewayConnectionState,
  );

  return useMemo(() => mapGatewayConnectionState(gatewayState), [gatewayState]);
}

export function resetGatewayConnectionStoreForTests(): void {
  const config = readLegacyGatewayConnectionFromCookies();
  if (config) {
    saveLegacyGatewayConnectionToCookies(config);
    return;
  }

  clearLegacyGatewayConnectionCookies();
}
