import { useState, useRef, useEffect } from "react";
import {
  ChevronDown,
  ChevronRight,
  Terminal,
  FileCode2,
  Send,
  User,
  Bot,
  Copy,
  Check,
  MoreHorizontal,
  Circle,
  Plus,
  Minus,
  ArrowUpRight,
} from "lucide-react";
import type {
  ConsoleSession as Session,
  ConsoleMachine as Machine,
  ConsoleMessage as Message,
  ConsoleFileChange as FileChange,
} from "../../design-host/use-console-host";

interface SessionChatProps {
  session: Session;
  machine: Machine;
  prompt: string;
  isSubmitting: boolean;
  onPromptChange: (value: string) => void;
  onSendPrompt: () => void;
}

function FileChangesBadge({ changes }: { changes: FileChange[] }) {
  const [expanded, setExpanded] = useState(false);
  const totalAdd = changes.reduce((s, f) => s + f.additions, 0);
  const totalDel = changes.reduce((s, f) => s + f.deletions, 0);

  return (
    <div className="mt-3 border border-zinc-700/60 rounded-lg overflow-hidden bg-zinc-950/60">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-zinc-800/40 transition-colors text-left"
      >
        <FileCode2 className="size-4 text-zinc-400 flex-shrink-0" />
        <span className="text-sm text-zinc-300 flex-1">{changes.length} 个文件已更改</span>
        <span className="text-xs text-emerald-400 flex items-center gap-0.5">
          <Plus className="size-3" />
          {totalAdd}
        </span>
        <span className="text-xs text-red-400 flex items-center gap-0.5 ml-1">
          <Minus className="size-3" />
          {totalDel}
        </span>
        <button className="ml-2 text-xs text-zinc-500 hover:text-zinc-300 px-2 py-0.5 rounded border border-zinc-700 hover:border-zinc-600 transition-colors">
          撤销
        </button>
        {expanded ? (
          <ChevronDown className="size-4 text-zinc-500 ml-1" />
        ) : (
          <ChevronRight className="size-4 text-zinc-500 ml-1" />
        )}
      </button>
      {expanded ? (
        <div className="border-t border-zinc-700/60 divide-y divide-zinc-800/50">
          {changes.map((file) => (
            <div
              key={file.path}
              className="flex items-center gap-3 px-4 py-2 hover:bg-zinc-800/30 transition-colors group"
            >
              <FileCode2 className="size-3.5 text-zinc-500 flex-shrink-0" />
              <span className="text-xs text-zinc-400 font-mono flex-1 truncate">{file.path}</span>
              <span className="text-xs text-emerald-400">+{file.additions}</span>
              <span className="text-xs text-red-400 ml-1">-{file.deletions}</span>
              <ArrowUpRight className="size-3.5 text-zinc-600 group-hover:text-zinc-400 transition-colors ml-1 opacity-0 group-hover:opacity-100" />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function TerminalBlock({ output }: { output: string }) {
  const [expanded, setExpanded] = useState(true);
  const lines = output.split("\n");

  return (
    <div className="mt-3 border border-zinc-700/60 rounded-lg overflow-hidden bg-zinc-950">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-4 py-2 hover:bg-zinc-800/30 transition-colors text-left border-b border-zinc-700/60"
      >
        <Terminal className="size-4 text-zinc-400 flex-shrink-0" />
        <span className="text-sm text-zinc-300 flex-1">终端输出</span>
        <span className="text-xs text-zinc-500">{lines.length} 行</span>
        {expanded ? (
          <ChevronDown className="size-4 text-zinc-500 ml-2" />
        ) : (
          <ChevronRight className="size-4 text-zinc-500 ml-2" />
        )}
      </button>
      {expanded ? (
        <div className="p-4 font-mono text-xs text-zinc-300 overflow-x-auto max-h-64 overflow-y-auto">
          {lines.map((line, i) => (
            <div
              key={i}
              className={`leading-5 ${
                line.startsWith("✓") || line.includes("PASS")
                  ? "text-emerald-400"
                  : line.startsWith("✗") || line.includes("FAIL") || line.includes("ERROR")
                    ? "text-red-400"
                    : line.startsWith("---") || line.startsWith("===")
                      ? "text-blue-400"
                      : line.includes("Warning") || line.includes("warning")
                        ? "text-amber-400"
                        : "text-zinc-300"
              }`}
            >
              {line || "\u00A0"}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const [copied, setCopied] = useState(false);
  const isUser = message.role === "user";

  const handleCopy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (isUser) {
    return (
      <div className="flex gap-3 justify-end group">
        <div className="max-w-[75%]">
          <div className="bg-blue-600/20 border border-blue-500/30 rounded-2xl rounded-tr-sm px-4 py-3">
            <p className="text-sm text-zinc-200 whitespace-pre-wrap leading-6">{message.content}</p>
          </div>
          <div className="flex items-center justify-end gap-2 mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <span className="text-xs text-zinc-600">{message.timestamp}</span>
          </div>
        </div>
        <div className="size-8 rounded-full bg-zinc-700 flex items-center justify-center flex-shrink-0 mt-1">
          <User className="size-4 text-zinc-300" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex gap-3 group">
      <div className="size-8 rounded-full bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center flex-shrink-0 mt-1">
        <Bot className="size-4 text-white" />
      </div>
      <div className="flex-1 min-w-0">
        {message.content ? (
          <div className="bg-zinc-800/50 border border-zinc-700/50 rounded-2xl rounded-tl-sm px-4 py-3">
            <p className="text-sm text-zinc-200 whitespace-pre-wrap leading-6">{message.content}</p>
          </div>
        ) : null}
        {message.fileChanges && message.fileChanges.length > 0 ? (
          <FileChangesBadge changes={message.fileChanges} />
        ) : null}
        {message.terminalOutput ? <TerminalBlock output={message.terminalOutput} /> : null}
        <div className="flex items-center gap-2 mt-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <span className="text-xs text-zinc-600">{message.timestamp}</span>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
          </button>
        </div>
      </div>
    </div>
  );
}

const MODEL_OPTIONS = [
  "claude-sonnet-4-5",
  "claude-opus-4",
  "gpt-4-turbo",
  "gemini-1.5-pro",
  "claude-haiku-3-5",
];

const PERMISSION_OPTIONS = ["本地", "完全访问权限", "只读模式"];

export default function SessionChat({
  session,
  machine,
  prompt,
  isSubmitting,
  onPromptChange,
  onSendPrompt,
}: SessionChatProps) {
  const [selectedModel, setSelectedModel] = useState(session.model);
  const [selectedPermission, setSelectedPermission] = useState("完全访问权限");
  const [showModelDrop, setShowModelDrop] = useState(false);
  const [showPermDrop, setShowPermDrop] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setSelectedModel(session.model);
  }, [session.model]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [session.messages, isSubmitting]);

  const handleSend = () => {
    if (isSubmitting) {
      return;
    }

    const trimmed = prompt.trim();
    if (!trimmed) return;
    onSendPrompt();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleTextareaInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onPromptChange(e.target.value);
    const ta = e.target;
    ta.style.height = "auto";
    ta.style.height = `${Math.min(ta.scrollHeight, 200)}px`;
  };

  const statusColor = {
    active: "text-emerald-400",
    idle: "text-zinc-500",
    systemError: "text-red-400",
    notLoaded: "text-amber-400",
    unknown: "text-zinc-500",
  }[session.status];

  const machineStatusDot = {
    online: "bg-emerald-400",
    offline: "bg-zinc-600",
    reconnecting: "bg-amber-400",
    unknown: "bg-zinc-500",
  }[machine.status];

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-4 lg:px-6 py-3">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0">
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex items-center gap-1.5">
                <div className={`size-2 rounded-full ${machineStatusDot}`} />
                <span className="text-xs text-zinc-500 font-mono hidden sm:block">{machine.name}</span>
              </div>
              <ChevronRight className="size-3.5 text-zinc-600 flex-shrink-0" />
              <span className="text-sm text-zinc-200 truncate">{session.title}</span>
            </div>
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            <div className="hidden sm:flex items-center gap-1.5">
              <div
                className={`size-1.5 rounded-full ${
                  session.status === "active" ? "bg-emerald-400 animate-pulse" : "bg-zinc-600"
                }`}
              />
              <span className={`text-xs ${statusColor}`}>
                {session.status === "active"
                  ? "进行中"
                  : session.status === "idle"
                    ? "空闲"
                    : session.status === "systemError"
                      ? "异常"
                      : session.status === "notLoaded"
                        ? "未加载"
                        : "未知"}
              </span>
            </div>
            <span className="text-xs text-zinc-600 hidden md:block font-mono">{session.agentName}</span>
            <button className="p-1.5 text-zinc-500 hover:text-zinc-300 transition-colors rounded-lg hover:bg-zinc-800">
              <MoreHorizontal className="size-4" />
            </button>
          </div>
        </div>
        <div className="hidden lg:flex items-center gap-4 mt-1.5">
          <span className="text-xs text-zinc-600 font-mono">ID: {machine.id}</span>
          <span className="text-zinc-700">·</span>
          <span className="text-xs text-zinc-600">Runtime: {machine.runtimeStatus}</span>
          <span className="text-zinc-700">·</span>
          <span className="text-xs text-zinc-600">{machine.agents.length} agents</span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-4 lg:px-8 py-6 space-y-6">
        {session.messages.map((msg) => (
          <MessageBubble key={msg.id} message={msg} />
        ))}

        {isSubmitting ? (
          <div className="flex gap-3">
            <div className="size-8 rounded-full bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center flex-shrink-0">
              <Bot className="size-4 text-white" />
            </div>
            <div className="bg-zinc-800/50 border border-zinc-700/50 rounded-2xl rounded-tl-sm px-4 py-3">
              <div className="flex gap-1.5 items-center h-5">
                <div className="size-1.5 bg-zinc-400 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                <div className="size-1.5 bg-zinc-400 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                <div className="size-1.5 bg-zinc-400 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
              </div>
            </div>
          </div>
        ) : null}
        <div ref={messagesEndRef} />
      </div>

      <div className="flex-shrink-0 border-t border-zinc-800 bg-zinc-900/80 p-4 lg:px-6 relative z-10">
        <div className="bg-zinc-800/60 border border-zinc-700/60 rounded-xl focus-within:border-zinc-600 transition-colors overflow-visible">
          <textarea
            ref={textareaRef}
            value={prompt}
            onChange={handleTextareaInput}
            onKeyDown={handleKeyDown}
            placeholder={`向 ${session.agentName} 发送指令...`}
            rows={1}
            className="w-full bg-transparent px-4 pt-3 pb-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:outline-none resize-none min-h-[44px] max-h-[200px]"
          />
          <div className="flex items-center justify-between px-3 pb-2.5 pt-1 relative">
            <div className="flex items-center gap-2 relative z-50">
              <div className="relative">
                <button
                  onClick={() => {
                    setShowModelDrop(!showModelDrop);
                    setShowPermDrop(false);
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-700/60 hover:bg-zinc-700 transition-colors text-xs text-zinc-300"
                >
                  <Circle className="size-2.5 text-violet-400 fill-violet-400" />
                  <span className="max-w-[100px] truncate">{selectedModel}</span>
                  <ChevronDown className="size-3 text-zinc-500" />
                </button>
                {showModelDrop ? (
                  <div className="absolute bottom-full mb-2 left-0 bg-zinc-800 border border-zinc-700 rounded-lg py-1 min-w-[180px] shadow-2xl">
                    {MODEL_OPTIONS.map((m) => (
                      <button
                        key={m}
                        onClick={() => {
                          setSelectedModel(m);
                          setShowModelDrop(false);
                        }}
                        className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                          selectedModel === m
                            ? "text-zinc-50 bg-zinc-700"
                            : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50"
                        }`}
                      >
                        {m}
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>

              <div className="relative hidden sm:block">
                <button
                  onClick={() => {
                    setShowPermDrop(!showPermDrop);
                    setShowModelDrop(false);
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-700/60 hover:bg-zinc-700 transition-colors text-xs text-zinc-300"
                >
                  <span className="text-emerald-400 text-[10px]">●</span>
                  <span>{selectedPermission}</span>
                  <ChevronDown className="size-3 text-zinc-500" />
                </button>
                {showPermDrop ? (
                  <div className="absolute bottom-full mb-2 left-0 bg-zinc-800 border border-zinc-700 rounded-lg py-1 min-w-[140px] shadow-2xl">
                    {PERMISSION_OPTIONS.map((p) => (
                      <button
                        key={p}
                        onClick={() => {
                          setSelectedPermission(p);
                          setShowPermDrop(false);
                        }}
                        className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                          selectedPermission === p
                            ? "text-zinc-50 bg-zinc-700"
                            : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50"
                        }`}
                      >
                        {p}
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>

            <button
              onClick={handleSend}
              disabled={!prompt.trim() || isSubmitting}
              className="size-8 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white flex items-center justify-center transition-colors flex-shrink-0"
            >
              <Send className="size-3.5" />
            </button>
          </div>
        </div>
        <p className="text-center text-xs text-zinc-700 mt-2">
          Shift+Enter 换行 · Enter 发送 · Agent 在 {machine.name} 上运行
        </p>
      </div>
    </div>
  );
}
