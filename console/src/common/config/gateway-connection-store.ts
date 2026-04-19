import { useMemo, useSyncExternalStore } from "react";

export interface GatewayConnectionConfig {
  gatewayUrl: string;
  apiKey: string;
}

export type GatewayConnectionState = "missing" | "ready" | "authFailed";

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

const GATEWAY_URL_COOKIE = "cag_gateway_url";
const GATEWAY_API_KEY_COOKIE = "cag_gateway_api_key";

let gatewayConnectionConfig = readGatewayConnectionFromCookies();
let gatewayConnectionState: GatewayConnectionState =
  gatewayConnectionConfig === null ? "missing" : "ready";
const listeners = new Set<() => void>();

function notifyGatewayConnectionSubscribers() {
  for (const listener of listeners) {
    listener();
  }
}

function parseCookieValue(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const cookies = document.cookie ? document.cookie.split(";") : [];
  for (const cookie of cookies) {
    const [rawName, ...valueParts] = cookie.split("=");
    if (!rawName || rawName.trim() !== name) {
      continue;
    }

    try {
      return decodeURIComponent(valueParts.join("="));
    } catch {
      return null;
    }
  }

  return null;
}

function normalizeGatewayUrl(value: string | null): string | null {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return null;
  }

  let parsedUrl: URL;
  try {
    parsedUrl = new URL(trimmed);
  } catch {
    return null;
  }

  if (parsedUrl.protocol !== "http:" && parsedUrl.protocol !== "https:") {
    return null;
  }

  return parsedUrl.toString().replace(/\/$/, "");
}

function normalizeApiKey(value: string | null): string | null {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length === 0 ? null : trimmed;
}

function hashIdentity(value: string): string {
  let hash = 0x811c9dc5;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16).padStart(8, "0");
}

function writeCookie(name: string, value: string, maxAgeSeconds?: number): void {
  if (typeof document === "undefined") {
    return;
  }

  const maxAgePart = typeof maxAgeSeconds === "number" ? `; Max-Age=${maxAgeSeconds}` : "";
  const securePart =
    typeof location !== "undefined" && location.protocol === "https:" ? "; Secure" : "";
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; SameSite=Lax${securePart}${maxAgePart}`;
}

function syncGatewayConnectionFromCookies(): void {
  const nextConfig = readGatewayConnectionFromCookies();
  gatewayConnectionConfig = nextConfig;
  gatewayConnectionState = nextConfig === null ? "missing" : "ready";
  notifyGatewayConnectionSubscribers();
}

function refreshGatewayConnectionFromCookies(): void {
  if (gatewayConnectionState === "authFailed") {
    return;
  }

  const nextConfig = readGatewayConnectionFromCookies();
  gatewayConnectionConfig = nextConfig;
  gatewayConnectionState = nextConfig === null ? "missing" : "ready";
}

export function readGatewayConnectionFromCookies(): GatewayConnectionConfig | null {
  const gatewayUrl = normalizeGatewayUrl(parseCookieValue(GATEWAY_URL_COOKIE));
  const apiKey = normalizeApiKey(parseCookieValue(GATEWAY_API_KEY_COOKIE));

  if (gatewayUrl === null || apiKey === null) {
    return null;
  }

  return { gatewayUrl, apiKey };
}

export function saveGatewayConnectionToCookies(config: GatewayConnectionConfig): void {
  writeCookie(GATEWAY_URL_COOKIE, config.gatewayUrl);
  writeCookie(GATEWAY_API_KEY_COOKIE, config.apiKey);
  syncGatewayConnectionFromCookies();
}

export function clearGatewayConnectionCookies(): void {
  writeCookie(GATEWAY_URL_COOKIE, "", 0);
  writeCookie(GATEWAY_API_KEY_COOKIE, "", 0);
  syncGatewayConnectionFromCookies();
}

export function subscribeGatewayConnection(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getGatewayConnectionIdentity(): string {
  const state = getGatewayConnectionState();
  if (state !== "ready") {
    return state;
  }

  const config = getGatewayConnectionConfig();
  if (!config) {
    return "missing";
  }

  return `ready:${hashIdentity(`${config.gatewayUrl}|${config.apiKey}`)}`;
}

export function getGatewayConnectionConfig(): GatewayConnectionConfig | null {
  refreshGatewayConnectionFromCookies();
  return gatewayConnectionState === "ready" ? gatewayConnectionConfig : null;
}

export function getGatewayConnectionState(): GatewayConnectionState {
  refreshGatewayConnectionFromCookies();
  return gatewayConnectionState;
}

export function requireGatewayConnectionConfig(): GatewayConnectionConfig {
  const state = getGatewayConnectionState();
  if (state === "authFailed") {
    throw new Error("Gateway authentication failed.");
  }

  const config = getGatewayConnectionConfig();
  if (!config) {
    throw new Error("Gateway connection is not configured.");
  }

  return config;
}

export function markGatewayAuthFailed(): void {
  if (gatewayConnectionState === "authFailed") {
    return;
  }

  gatewayConnectionState = "authFailed";
  notifyGatewayConnectionSubscribers();
}

export function useGatewayConnectionState(): SettingsGatewayConnectionState {
  const gatewayState = useSyncExternalStore(
    subscribeGatewayConnection,
    getGatewayConnectionState,
    getGatewayConnectionState,
  );

  return useMemo(() => mapGatewayConnectionState(gatewayState), [gatewayState]);
}

export function resetGatewayConnectionStoreForTests(): void {
  gatewayConnectionConfig = readGatewayConnectionFromCookies();
  gatewayConnectionState = gatewayConnectionConfig === null ? "missing" : "ready";
  notifyGatewayConnectionSubscribers();
}
