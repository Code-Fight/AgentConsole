const DEFAULT_WS_URL = "ws://localhost:8080/ws";
const WS_URL = import.meta.env.VITE_WS_URL ?? DEFAULT_WS_URL;

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
