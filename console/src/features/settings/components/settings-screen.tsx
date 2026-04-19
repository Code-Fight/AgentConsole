import { useEffect, useState } from "react";
import { Bot, Key, LoaderCircle, Save } from "lucide-react";
import {
  readGatewayConnectionFromCookies,
  saveGatewayConnectionToCookies,
} from "../model/gateway-connection-store";

export function SettingsScreen() {
  const [draftGatewayUrl, setDraftGatewayUrl] = useState("");
  const [draftApiKey, setDraftApiKey] = useState("");
  const [lastSuccessfulTestSignature, setLastSuccessfulTestSignature] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  useEffect(() => {
    const localConnection = readGatewayConnectionFromCookies();
    setDraftGatewayUrl(localConnection?.gatewayUrl ?? "");
    setDraftApiKey(localConnection?.apiKey ?? "");
  }, []);

  const currentGatewayUrl = draftGatewayUrl.trim();
  const currentApiKey = draftApiKey.trim();
  const currentConnectionSignature = `${currentGatewayUrl}\n${currentApiKey}`;
  const canSaveGatewayConnection =
    lastSuccessfulTestSignature !== null &&
    lastSuccessfulTestSignature === currentConnectionSignature;

  const handleSaveGatewayConnection = () => {
    if (!canSaveGatewayConnection) {
      setErrorMessage("Please test the Gateway connection successfully before saving.");
      setStatusMessage(null);
      return;
    }

    const gatewayUrl = currentGatewayUrl;
    const apiKey = currentApiKey;

    let parsedUrl: URL | null = null;
    try {
      parsedUrl = new URL(gatewayUrl);
    } catch {
      parsedUrl = null;
    }

    const hasValidUrl =
      parsedUrl !== null &&
      (parsedUrl.protocol === "http:" || parsedUrl.protocol === "https:");
    const hasApiKey = apiKey.length > 0;

    if (!hasValidUrl || !hasApiKey) {
      setErrorMessage("Please enter a valid Gateway URL and API key.");
      setStatusMessage(null);
      return;
    }

    saveGatewayConnectionToCookies({ gatewayUrl, apiKey });
    setErrorMessage(null);
    setStatusMessage("Gateway connection saved.");
  };

  const handleTestGatewayConnection = async () => {
    const gatewayUrl = currentGatewayUrl;
    const apiKey = currentApiKey;

    let parsedUrl: URL | null = null;
    try {
      parsedUrl = new URL(gatewayUrl);
    } catch {
      parsedUrl = null;
    }

    const hasValidUrl =
      parsedUrl !== null &&
      (parsedUrl.protocol === "http:" || parsedUrl.protocol === "https:");
    const hasApiKey = apiKey.length > 0;

    if (!hasValidUrl || !hasApiKey) {
      setLastSuccessfulTestSignature(null);
      setErrorMessage("Please enter a valid Gateway URL and API key.");
      setStatusMessage(null);
      return;
    }

    const headers = new Headers();
    headers.set("Accept", "application/json");
    headers.set("Authorization", `Bearer ${apiKey}`);

    setIsTestingConnection(true);
    setErrorMessage(null);
    setStatusMessage(null);

    try {
      const response = await fetch(`${gatewayUrl}/machines`, {
        headers,
      });

      if (response.status === 401) {
        setLastSuccessfulTestSignature(null);
        setErrorMessage("Gateway authentication failed.");
        return;
      }

      if (!response.ok) {
        setLastSuccessfulTestSignature(null);
        setErrorMessage(`Gateway connection test failed with status ${response.status}.`);
        return;
      }

      setLastSuccessfulTestSignature(`${gatewayUrl}\n${apiKey}`);
      setStatusMessage("Gateway connection test succeeded.");
    } catch {
      setLastSuccessfulTestSignature(null);
      setErrorMessage("Gateway connection test failed.");
    } finally {
      setIsTestingConnection(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
            <Bot className="size-5 text-zinc-400" />
          </div>
          <div>
            <h1 aria-label="Settings" className="text-lg text-zinc-100">
              设置
            </h1>
            <p className="text-xs text-zinc-500 mt-0.5">管理 Gateway 连接配置</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl mx-auto space-y-6">
          {errorMessage ? (
            <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-sm text-red-300">
              {errorMessage}
            </div>
          ) : null}
          {statusMessage ? (
            <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl px-4 py-3 text-sm text-emerald-300">
              {statusMessage}
            </div>
          ) : null}

          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-zinc-800 rounded-lg">
                <Key className="size-5 text-blue-400" />
              </div>
              <div>
                <h2 className="text-base text-zinc-100">API Configuration</h2>
                <p className="text-xs text-zinc-500">管理 Gateway 连接配置</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="gateway-url">
                  Gateway URL
                </label>
                <input
                  id="gateway-url"
                  aria-label="Gateway URL"
                  type="text"
                  value={draftGatewayUrl}
                  onChange={(event) => {
                    setErrorMessage(null);
                    setStatusMessage(null);
                    setLastSuccessfulTestSignature(null);
                    setDraftGatewayUrl(event.target.value);
                  }}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-2" htmlFor="gateway-api-key">
                  API Key
                </label>
                <input
                  id="gateway-api-key"
                  aria-label="Gateway API Key"
                  type="password"
                  value={draftApiKey}
                  onChange={(event) => {
                    setErrorMessage(null);
                    setStatusMessage(null);
                    setLastSuccessfulTestSignature(null);
                    setDraftApiKey(event.target.value);
                  }}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-2">
                <button
                  type="button"
                  aria-label="Test Gateway Connection"
                  disabled={isTestingConnection}
                  onClick={() => void handleTestGatewayConnection()}
                  className="flex items-center gap-2 px-4 py-2 bg-zinc-800 hover:bg-zinc-700 disabled:bg-zinc-800/60 disabled:text-zinc-500 text-zinc-200 rounded-lg text-sm transition-colors"
                >
                  {isTestingConnection ? <LoaderCircle className="size-4 animate-spin" /> : null}
                  {isTestingConnection ? "测试中..." : "测试连接"}
                </button>
                <button
                  type="button"
                  aria-label="Save Gateway Connection"
                  disabled={!canSaveGatewayConnection}
                  onClick={handleSaveGatewayConnection}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-600/50 disabled:text-zinc-200/70 disabled:cursor-not-allowed text-white rounded-lg text-sm transition-colors"
                >
                  <Save className="size-4" />
                  保存 Gateway 连接
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
