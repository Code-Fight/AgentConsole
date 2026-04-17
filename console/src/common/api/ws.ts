import {
  getGatewayConnectionConfig,
  markGatewayAuthFailed,
} from "../../gateway/gateway-connection-store";

function toWebSocketOrigin(gatewayUrl: string): string {
  const parsedUrl = new URL(gatewayUrl);
  const wsProtocol = parsedUrl.protocol === "https:" ? "wss:" : "ws:";
  return `${wsProtocol}//${parsedUrl.host}`;
}

export function buildConsoleSocketUrl(threadId?: string): string | null {
  const config = getGatewayConnectionConfig();
  if (!config) {
    return null;
  }

  const wsBaseUrl = `${toWebSocketOrigin(config.gatewayUrl)}/ws`;
  const params = new URLSearchParams();
  if (threadId) {
    params.set("threadId", threadId);
  }
  params.set("apiKey", config.apiKey);
  const query = params.toString();
  return query.length > 0 ? `${wsBaseUrl}?${query}` : wsBaseUrl;
}

export function buildThreadSocketUrl(threadId: string): string | null {
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

  const handleClose = (event: CloseEvent) => {
    if (event.code === 1008 || event.code === 4001) {
      markGatewayAuthFailed();
      closed = true;
      clearReconnectTimer();
      socket = null;
      return;
    }

    socket = null;
    scheduleReconnect();
  };

  const handleError = () => {
    if (socket !== null) {
      // Some runtimes can emit error without close; force closure and reconnect.
      const activeSocket = socket;
      socket = null;
      activeSocket.close();
    }

    scheduleReconnect();
  };

  const connect = () => {
    if (closed) {
      return;
    }

    const socketUrl = buildConsoleSocketUrl(threadId);
    if (socketUrl === null) {
      closed = true;
      clearReconnectTimer();
      return;
    }

    const nextSocket = new WebSocket(socketUrl);
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
