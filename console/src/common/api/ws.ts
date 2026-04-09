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
  let closed = false;
  let socket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let reconnectDelay = 1_000;

  const clearReconnectTimer = () => {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  };

  const scheduleReconnect = () => {
    if (closed || reconnectTimer !== null) {
      return;
    }

    const delay = reconnectDelay;
    reconnectDelay = Math.min(reconnectDelay * 2, 5_000);
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connect();
    }, delay);
  };

  const handleOpen = () => {
    reconnectDelay = 1_000;
  };

  const handleClose = () => {
    socket = null;
    scheduleReconnect();
  };

  const handleError = () => {
    scheduleReconnect();
  };

  const connect = () => {
    if (closed) {
      return;
    }

    const nextSocket = new WebSocket(buildConsoleSocketUrl(threadId));
    nextSocket.addEventListener("open", handleOpen);
    nextSocket.addEventListener("message", onMessage);
    nextSocket.addEventListener("close", handleClose);
    nextSocket.addEventListener("error", handleError);
    socket = nextSocket;
  };

  connect();

  return () => {
    closed = true;
    clearReconnectTimer();
    if (socket !== null) {
      socket.removeEventListener("open", handleOpen);
      socket.removeEventListener("message", onMessage);
      socket.removeEventListener("close", handleClose);
      socket.removeEventListener("error", handleError);
      socket.close();
      socket = null;
    }
  };
}
