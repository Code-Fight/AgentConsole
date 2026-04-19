import {
  markGatewayAuthFailed,
  requireGatewayConnectionConfig,
} from "../config/gateway-connection-store";

export function buildThreadApiPath(threadId: string, resource?: string): string {
  const encodedThreadId = encodeURIComponent(threadId);
  if (!resource) {
    return `/threads/${encodedThreadId}`;
  }

  const normalizedResource = resource.replace(/^\/+/, "");
  return `/threads/${encodedThreadId}/${normalizedResource}`;
}

export async function http<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const config = requireGatewayConnectionConfig();

  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }
  headers.set("Authorization", `Bearer ${config.apiKey}`);

  const response = await fetch(`${config.gatewayUrl}${path}`, {
    ...init,
    headers,
  });

  if (response.status === 401) {
    markGatewayAuthFailed();
    throw new Error("Gateway authentication failed.");
  }

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
