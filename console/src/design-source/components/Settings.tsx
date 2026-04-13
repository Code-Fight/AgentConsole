import { useState } from "react";
import { Key, Bot, Save } from "lucide-react";

interface AgentDefaultConfig {
  "claude-code": string;
  codex: string;
  custom: string;
}

export default function Settings() {
  const [apiUrl, setApiUrl] = useState("https://gateway.example.com");
  const [apiKey, setApiKey] = useState("sk_live_xxxxxxxxxxxxxxxxxxxxx");
  const [selectedAgentType, setSelectedAgentType] = useState<"claude-code" | "codex" | "custom">("claude-code");

  const [agentConfigs, setAgentConfigs] = useState<AgentDefaultConfig>({
    "claude-code": `# Claude Code Agent Default Configuration
model = "claude-sonnet-4-5"
temperature = 0.7
max_tokens = 4096

[timeout]
default = 60000

[retry]
max_retries = 3
backoff = "exponential"`,
    codex: `# Codex Agent Default Configuration
model = "claude-sonnet-4-5"
api_version = "v1"
enable_caching = true

[timeout]
default = 30000`,
    custom: `# Custom Agent Default Configuration
model = "gpt-4-turbo"
temperature = 0.8
max_tokens = 2048`,
  });

  const handleConfigChange = (value: string) => {
    setAgentConfigs((prev) => ({
      ...prev,
      [selectedAgentType]: value,
    }));
  };

  const handleSave = () => {
    console.log("Saving configurations...");
  };

  const agentTypeOptions = [
    { value: "claude-code" as const, label: "Claude Code" },
    { value: "codex" as const, label: "Codex" },
    { value: "custom" as const, label: "Custom" },
  ];

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="size-10 rounded-xl bg-zinc-800/60 flex items-center justify-center">
              <Bot className="size-5 text-zinc-400" />
            </div>
            <div>
              <h1 className="text-lg text-zinc-100">设置</h1>
              <p className="text-xs text-zinc-500 mt-0.5">配置 API 和 Agent 默认参数</p>
            </div>
          </div>
          <button
            onClick={handleSave}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-sm transition-colors"
          >
            <Save className="size-4" />
            保存配置
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl mx-auto space-y-6">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-zinc-800 rounded-lg">
                <Key className="size-5 text-blue-400" />
              </div>
              <div>
                <h2 className="text-base text-zinc-100">API Configuration</h2>
                <p className="text-xs text-zinc-500">管理 API 密钥和端点</p>
              </div>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-zinc-400 mb-2">Gateway URL</label>
                <input
                  type="text"
                  value={apiUrl}
                  onChange={(e) => setApiUrl(e.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>
              <div>
                <label className="block text-sm text-zinc-400 mb-2">API Key</label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-2.5 text-sm text-zinc-300 focus:outline-none focus:border-blue-500 transition-colors"
                />
              </div>
            </div>
          </div>

          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 bg-zinc-800 rounded-lg">
                <Bot className="size-5 text-violet-400" />
              </div>
              <div>
                <h2 className="text-base text-zinc-100">Agent 默认配置</h2>
                <p className="text-xs text-zinc-500">为不同类型的 Agent 设置默认启动配置</p>
              </div>
            </div>

            <div className="mb-4">
              <label className="block text-sm text-zinc-400 mb-2">Agent 类型</label>
              <div className="flex gap-2">
                {agentTypeOptions.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => setSelectedAgentType(option.value)}
                    className={`px-4 py-2 rounded-lg text-sm transition-colors ${
                      selectedAgentType === option.value
                        ? "bg-blue-600 text-white"
                        : "bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-200"
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm text-zinc-400">配置文件 (TOML)</label>
                <span className="text-xs text-zinc-600">
                  当前编辑: {agentTypeOptions.find((o) => o.value === selectedAgentType)?.label}
                </span>
              </div>
              <textarea
                value={agentConfigs[selectedAgentType]}
                onChange={(e) => handleConfigChange(e.target.value)}
                rows={16}
                className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-300 font-mono focus:outline-none focus:border-blue-500 transition-colors resize-none"
                placeholder="输入 TOML 配置..."
              />
              <p className="text-xs text-zinc-600 mt-2">
                💡 在机器管理界面安装 Agent 时，会自动使用此配置作为默认参数
              </p>
            </div>
          </div>

          <div className="bg-zinc-900/50 border border-zinc-800/50 rounded-xl p-4">
            <h3 className="text-sm text-zinc-400 mb-3">配置说明</h3>
            <div className="space-y-2 text-xs text-zinc-600">
              <div className="flex gap-2">
                <span className="text-blue-400">•</span>
                <span>
                  <strong className="text-zinc-500">Claude Code</strong>: 用于 Claude Code Agent 的默认配置
                </span>
              </div>
              <div className="flex gap-2">
                <span className="text-blue-400">•</span>
                <span>
                  <strong className="text-zinc-500">Codex</strong>: 用于 Codex Agent 的默认配置
                </span>
              </div>
              <div className="flex gap-2">
                <span className="text-blue-400">•</span>
                <span>
                  <strong className="text-zinc-500">Custom</strong>: 用于自定义 Agent 的默认配置
                </span>
              </div>
              <div className="flex gap-2 mt-3 pt-3 border-t border-zinc-800">
                <span className="text-amber-400">⚠</span>
                <span>修改配置后需要点击右上角的 "保存配置" 按钮才会生效</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
