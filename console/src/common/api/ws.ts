function getDefaultWsUrl(): string {
  if (typeof window === "undefined") {
    return "ws://localhost/ws";
  }

  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  return `${protocol}://${window.location.host}/ws`;
}

const WS_URL = import.meta.env.VITE_WS_URL ?? getDefaultWsUrl();

export function buildConsoleSocketUrl(threadId?: string): string {
  if (!threadId) {
    return WS_URL;
  }

  const separator = WS_URL.includes("?") ? "&" : "?";
  return `${WS_URL}${separator}threadId=${encodeURIComponent(threadId)}`;
}

export function buildThreadSocketUrl(threadId: string): string {
  return buildConsoleSocketUrl(threadId);
}

export function connectConsoleSocket(
  threadId: string | undefined,
  onMessage: (event: MessageEvent<string>) => void,
): () => void {
  const socket = new WebSocket(buildConsoleSocketUrl(threadId));
  socket.addEventListener("message", onMessage);

  return () => {
    socket.removeEventListener("message", onMessage);
    socket.close();
  };
}
