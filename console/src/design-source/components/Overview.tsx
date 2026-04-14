import type { ReactNode } from "react";
import { Activity, Bot, CheckCircle2, Database, Server } from "lucide-react";
import type { OverviewMetrics } from "../../common/api/types";

interface OverviewProps {
  metrics: OverviewMetrics | null;
  isLoading: boolean;
  error: string | null;
}

function MetricCard(props: {
  title: string;
  value: number | string;
  detail: string;
  icon: ReactNode;
}) {
  return (
    <article className="bg-zinc-900 border border-zinc-800 rounded-2xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-xs text-zinc-500">{props.title}</p>
          <p className="text-3xl text-zinc-100 mt-1">{props.value}</p>
        </div>
        <div className="size-11 rounded-xl bg-zinc-800 flex items-center justify-center">
          {props.icon}
        </div>
      </div>
      <p className="text-xs text-zinc-600">{props.detail}</p>
    </article>
  );
}

export default function Overview(props: OverviewProps) {
  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
            <Activity className="size-5 text-zinc-400" />
          </div>
          <div>
            <h1 className="text-lg text-zinc-100">Overview</h1>
            <p className="text-xs text-zinc-500 mt-0.5">
              Gateway-backed machine, thread, approval, and environment activity snapshot
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {props.isLoading ? (
          <p className="text-sm text-zinc-500">Loading overview metrics...</p>
        ) : null}
        {props.error ? <p className="text-sm text-red-400">{props.error}</p> : null}

        {!props.isLoading && !props.error && props.metrics ? (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            <MetricCard
              title="Online Machines"
              value={props.metrics.onlineMachines}
              detail="Machines currently connected to the gateway."
              icon={<Server className="size-5 text-emerald-400" />}
            />
            <MetricCard
              title="Running Agents"
              value={props.metrics.runningAgents}
              detail="Managed agent instances currently running."
              icon={<Bot className="size-5 text-blue-400" />}
            />
            <MetricCard
              title="Active Threads"
              value={props.metrics.activeThreads}
              detail="Threads whose status is currently active."
              icon={<CheckCircle2 className="size-5 text-violet-400" />}
            />
            <MetricCard
              title="Pending Approvals"
              value={props.metrics.pendingApprovals}
              detail="Approval requests awaiting operator action."
              icon={<Activity className="size-5 text-amber-400" />}
            />
            <MetricCard
              title="Environment Items"
              value={props.metrics.environmentItems}
              detail="Tracked skills, MCP servers, and plugins."
              icon={<Database className="size-5 text-zinc-300" />}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}
