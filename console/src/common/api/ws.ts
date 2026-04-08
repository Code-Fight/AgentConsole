function getDefaultWsUrl(): string {
  if (typeof window === "undefined") {
    return "ws://localhost/ws";
  }

  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  return `${protocol}://${window.location.host}/ws`;
}

const WS_URL = import.meta.env.VITE_WS_URL ?? getDefaultWsUrl();

export function buildThreadSocketUrl(threadId: string): string {
  const separator = WS_URL.includes("?") ? "&" : "?";
  return `${WS_URL}${separator}threadId=${encodeURIComponent(threadId)}`;
}

export function connectConsoleSocket(
  onMessage: (event: MessageEvent<string>) => void,
): () => void {
  const socket = new WebSocket(WS_URL);
  socket.addEventListener("message", onMessage);

  return () => {
    socket.removeEventListener("message", onMessage);
    socket.close();
  };
}
