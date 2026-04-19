import { useEffect, useState } from "react";
import { Activity, AlertCircle, ArrowLeft, Bot, Boxes, Cpu, Server } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { http } from "../../../common/api/http";
import { useGatewayConnectionState } from "../../../common/config/gateway-connection-store";
import type { OverviewMetrics } from "../../../common/api/types";

const metricCards = [
  {
    key: "onlineMachines",
    label: "Online Machines",
    description: "Currently connected machines",
    icon: Server,
  },
  {
    key: "runningAgents",
    label: "Running Agents",
    description: "Active agent runtimes",
    icon: Cpu,
  },
  {
    key: "activeThreads",
    label: "Active Threads",
    description: "Threads with in-flight work",
    icon: Activity,
  },
  {
    key: "pendingApprovals",
    label: "Pending Approvals",
    description: "Approvals waiting for action",
    icon: AlertCircle,
  },
  {
    key: "environmentItems",
    label: "Environment Items",
    description: "Managed skills, MCPs, and plugins",
    icon: Boxes,
  },
] as const satisfies Array<{
  key: keyof OverviewMetrics;
  label: string;
  description: string;
  icon: typeof Server;
}>;

export function OverviewPage() {
  const navigate = useNavigate();
  const connection = useGatewayConnectionState();
  const [metrics, setMetrics] = useState<OverviewMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(connection.remoteEnabled);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!connection.remoteEnabled) {
      setMetrics(null);
      setError(null);
      setIsLoading(false);
      return;
    }

    let cancelled = false;
    setIsLoading(true);

    void http<OverviewMetrics>("/overview/metrics")
      .then((nextMetrics) => {
        if (cancelled) {
          return;
        }
        setMetrics(nextMetrics);
        setError(null);
      })
      .catch((loadError) => {
        if (cancelled) {
          return;
        }
        setMetrics(null);
        setError(
          loadError instanceof Error ? loadError.message : "Unable to load overview metrics.",
        );
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [connection.remoteEnabled]);

  return (
    <div className="size-full bg-zinc-950 text-zinc-100 flex flex-col overflow-hidden">
      <header className="lg:hidden flex items-center gap-3 px-4 py-3 bg-zinc-900 border-b border-zinc-800 flex-shrink-0">
        <button
          type="button"
          aria-label="返回线程"
          onClick={() => navigate("/")}
          className="p-2 text-zinc-400 hover:text-zinc-50 rounded-lg hover:bg-zinc-800 transition-colors"
        >
          <ArrowLeft className="size-5" />
        </button>

        <div className="flex-1">
          <div className="flex items-center gap-2">
            <div className="size-6 rounded-lg bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center">
              <Bot className="size-3.5 text-white" />
            </div>
            <span className="text-sm text-zinc-50 tracking-tight">概览</span>
          </div>
        </div>
      </header>

      <div className="flex flex-1 min-h-0">
        <div className="hidden lg:flex flex-shrink-0 flex-col bg-zinc-900 border-r border-zinc-800 w-16">
          <div className="flex items-center justify-center py-4 border-b border-zinc-800">
            <button
              type="button"
              aria-label="返回线程"
              onClick={() => navigate("/")}
              className="size-10 rounded-xl bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center transition-colors"
              title="返回线程"
            >
              <ArrowLeft className="size-5 text-zinc-400" />
            </button>
          </div>
        </div>

        <main className="flex-1 overflow-y-auto p-6">
          <div className="max-w-6xl mx-auto space-y-6">
            <div className="flex items-center gap-3">
              <div className="size-12 rounded-2xl bg-zinc-900 border border-zinc-800 flex items-center justify-center">
                <Activity className="size-5 text-blue-400" />
              </div>
              <div>
                <h1 className="text-xl text-zinc-50">Gateway Overview</h1>
                <p className="text-sm text-zinc-500">
                  Live control-plane metrics from the feature runtime.
                </p>
              </div>
            </div>

            {error ? (
              <div className="rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">
                {error}
              </div>
            ) : null}

            {isLoading ? (
              <div className="rounded-2xl border border-zinc-800 bg-zinc-900/80 px-4 py-3 text-sm text-zinc-400">
                正在加载概览指标…
              </div>
            ) : null}

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {metricCards.map((card) => {
                const Icon = card.icon;
                return (
                  <section
                    key={card.key}
                    className="rounded-3xl border border-zinc-800 bg-zinc-900/80 p-5 shadow-[0_16px_60px_rgba(0,0,0,0.35)]"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="text-xs uppercase tracking-[0.18em] text-zinc-500">
                          {card.label}
                        </p>
                        <p className="mt-2 text-3xl text-zinc-50">
                          {metrics ? metrics[card.key] : "0"}
                        </p>
                        <p className="mt-2 text-sm text-zinc-500">{card.description}</p>
                      </div>
                      <div className="size-10 rounded-2xl bg-zinc-800/90 border border-zinc-700 flex items-center justify-center">
                        <Icon className="size-4 text-zinc-300" />
                      </div>
                    </div>
                  </section>
                );
              })}
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
