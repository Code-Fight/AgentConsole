export interface AgentInfo {
  id: string;
  name: string;
  type: string;
  model: string;
  status: "active" | "idle" | "offline" | "error";
}

export interface SessionInfo {
  id: string;
  title: string;
  status: "active" | "idle" | "offline" | "error";
}

export interface Machine {
  id: string;
  name: string;
  status: "online" | "offline";
  host: string;
  os: string;
  agents: AgentInfo[];
  sessions: SessionInfo[];
}
