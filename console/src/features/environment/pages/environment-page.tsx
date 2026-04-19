import { ArrowLeft, Bot } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useGatewayConnectionState } from "../../../common/config/gateway-connection-store";
import { EnvironmentScreen } from "../components/environment-screen";

export function EnvironmentPage() {
  const navigate = useNavigate();
  const connection = useGatewayConnectionState();

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
            <span className="text-sm text-zinc-50 tracking-tight">环境资源</span>
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

        <main className="flex-1 overflow-hidden">
          <EnvironmentScreen enabled={connection.remoteEnabled} />
        </main>
      </div>
    </div>
  );
}
