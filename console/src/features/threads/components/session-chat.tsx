import { useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  ArrowUpRight,
  Bot,
  Check,
  ChevronDown,
  ChevronRight,
  Circle,
  Copy,
  FileCode2,
  Loader2,
  Minus,
  MoreHorizontal,
  Plus,
  Send,
  Terminal,
  User,
} from "lucide-react";
import type { ApprovalDecision } from "../../../common/api/types";
import type { ThreadWorkspaceViewModel } from "../hooks/use-thread-workspace";
import type {
  ThreadMachineViewModel as Machine,
  ThreadSessionViewModel as Session,
  WorkspaceApprovalCardViewModel,
  WorkspaceMessageViewModel,
} from "../model/thread-view-model";

interface SessionChatProps {
  session: Session;
  machine: Machine;
  workspace: ThreadWorkspaceViewModel;
}

function FileChangesBadge({
  changes,
}: {
  changes: Array<{
    path: string;
    additions: number;
    deletions: number;
  }>;
}) {
  const [expanded, setExpanded] = useState(false);
  const totalAdd = changes.reduce((sum, file) => sum + file.additions, 0);
  const totalDel = changes.reduce((sum, file) => sum + file.deletions, 0);

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
          {lines.map((line, index) => (
            <div
              key={index}
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

type ChatMessage = WorkspaceMessageViewModel & {
  timestamp?: string;
  fileChanges?: Array<{
    path: string;
    additions: number;
    deletions: number;
  }>;
  terminalOutput?: string;
};

function ExecutingIndicator({ className = "" }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2 text-emerald-400 text-sm ${className}`}>
      <Loader2 className="size-4 animate-spin" />
      <span>正在执行...</span>
    </div>
  );
}

function ProgressBlock({ text, isExecuting }: { text: string; isExecuting?: boolean }) {
  const [expanded, setExpanded] = useState(Boolean(isExecuting));
  const cleanedText = text.trim();
  const lineCount = cleanedText ? cleanedText.split("\n").filter((line) => line.trim() !== "").length : 0;

  useEffect(() => {
    setExpanded(Boolean(isExecuting));
  }, [isExecuting]);

  if (!cleanedText) {
    return null;
  }

  return (
    <div className="mb-2 overflow-hidden rounded-lg border border-zinc-800/80 bg-zinc-900/40">
      <button
        type="button"
        onClick={() => setExpanded((current) => !current)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left transition-colors hover:bg-zinc-800/40"
      >
        {expanded ? (
          <ChevronDown className="size-4 flex-shrink-0 text-zinc-500" />
        ) : (
          <ChevronRight className="size-4 flex-shrink-0 text-zinc-500" />
        )}
        <span className="text-xs text-zinc-400">{isExecuting ? "处理中" : "已处理"}</span>
        {lineCount > 0 ? <span className="text-xs text-zinc-600">{lineCount} 行</span> : null}
      </button>
      {expanded ? (
        <div className="border-t border-zinc-800/80 px-3 py-2">
          <AgentMarkdown text={cleanedText} />
        </div>
      ) : null}
    </div>
  );
}

function AgentMarkdown({ text }: { text: string }) {
  return (
    <div className="text-sm text-zinc-200 leading-6 break-words">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          p: ({ children }) => <p className="my-2 first:mt-0 last:mb-0">{children}</p>,
          strong: ({ children }) => <strong className="font-semibold text-zinc-100">{children}</strong>,
          em: ({ children }) => <em className="italic text-zinc-200">{children}</em>,
          h1: ({ children }) => <h1 className="mt-4 mb-2 text-xl font-semibold text-zinc-50">{children}</h1>,
          h2: ({ children }) => <h2 className="mt-4 mb-2 text-lg font-semibold text-zinc-50">{children}</h2>,
          h3: ({ children }) => <h3 className="mt-3 mb-2 text-base font-semibold text-zinc-50">{children}</h3>,
          ul: ({ children }) => <ul className="my-2 ml-5 list-disc space-y-1">{children}</ul>,
          ol: ({ children }) => <ol className="my-2 ml-5 list-decimal space-y-1">{children}</ol>,
          li: ({ children }) => <li className="pl-1">{children}</li>,
          blockquote: ({ children }) => (
            <blockquote className="my-3 border-l-2 border-zinc-600 pl-3 text-zinc-300">
              {children}
            </blockquote>
          ),
          code: ({ children }) => (
            <code className="rounded bg-zinc-950/70 px-1.5 py-0.5 font-mono text-[0.85em] text-zinc-100">
              {children}
            </code>
          ),
          pre: ({ children }) => (
            <pre className="my-3 overflow-x-auto rounded-lg border border-zinc-700/60 bg-zinc-950 p-3 text-xs leading-5">
              {children}
            </pre>
          ),
          table: ({ children }) => (
            <div className="my-3 overflow-x-auto rounded-lg border border-zinc-700/70">
              <table className="w-full border-collapse text-left text-xs">{children}</table>
            </div>
          ),
          thead: ({ children }) => <thead className="bg-zinc-900/80 text-zinc-200">{children}</thead>,
          tbody: ({ children }) => <tbody className="divide-y divide-zinc-800">{children}</tbody>,
          th: ({ children }) => <th className="border-b border-zinc-700 px-3 py-2 font-semibold">{children}</th>,
          td: ({ children }) => <td className="px-3 py-2 align-top text-zinc-300">{children}</td>,
          a: ({ children, href }) => (
            <a
              href={href}
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline decoration-blue-400/40 underline-offset-2 hover:text-blue-200"
            >
              {children}
            </a>
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  );
}

function MessageBubble({
  message,
  isExecuting,
}: {
  message: ChatMessage;
  isExecuting?: boolean;
}) {
  const [copied, setCopied] = useState(false);
  const isUser = message.kind === "user";

  if (message.kind === "system") {
    return (
      <div className="flex justify-center">
        <div className="px-3 py-1.5 rounded-full bg-zinc-900/80 border border-zinc-800 text-xs text-zinc-400">
          {message.text}
        </div>
      </div>
    );
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(message.text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (isUser) {
    return (
      <div className="flex gap-3 justify-end group">
        <div className="max-w-[75%]">
          <div className="bg-blue-600/20 border border-blue-500/30 rounded-2xl rounded-tr-sm px-4 py-3">
            <p className="text-sm text-zinc-200 whitespace-pre-wrap leading-6">{message.text}</p>
          </div>
          <div className="flex items-center justify-end gap-2 mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
            {message.timestamp ? (
              <span className="text-xs text-zinc-600">{message.timestamp}</span>
            ) : null}
          </div>
        </div>
        <div className="size-8 rounded-full bg-zinc-700 flex items-center justify-center flex-shrink-0 mt-1">
          <User className="size-4 text-zinc-300" />
        </div>
      </div>
    );
  }

  const hasAgentBody = Boolean(isExecuting || message.progressText || message.text);

  return (
    <div className="flex gap-3 group">
      <div className="size-8 rounded-full bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center flex-shrink-0 mt-1">
        <Bot className="size-4 text-white" />
      </div>
      <div className="flex-1 min-w-0">
        {hasAgentBody ? (
          <div
            data-testid="agent-message-bubble"
            className="bg-zinc-800/50 border border-zinc-700/50 rounded-2xl rounded-tl-sm px-4 py-3"
          >
            {isExecuting ? (
              <ExecutingIndicator className={message.progressText || message.text ? "mb-3" : ""} />
            ) : null}
            {message.progressText ? (
              <ProgressBlock text={message.progressText} isExecuting={isExecuting} />
            ) : null}
            {message.text ? <AgentMarkdown text={message.text} /> : null}
          </div>
        ) : null}
        {message.fileChanges && message.fileChanges.length > 0 ? (
          <FileChangesBadge changes={message.fileChanges} />
        ) : null}
        {message.terminalOutput ? <TerminalBlock output={message.terminalOutput} /> : null}
        <div className="flex items-center gap-2 mt-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
          {message.timestamp ? (
            <span className="text-xs text-zinc-600">{message.timestamp}</span>
          ) : null}
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

const sandboxModeLabels: Record<string, string> = {
  "workspace-write": "本地",
  "danger-full-access": "完全访问权限",
  "read-only": "只读模式",
};

function formatSandboxMode(mode: string) {
  return sandboxModeLabels[mode] ?? mode;
}

export default function SessionChat({ session, machine, workspace }: SessionChatProps) {
  const [showModelDrop, setShowModelDrop] = useState(false);
  const [showPermDrop, setShowPermDrop] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const messages = useMemo<ChatMessage[]>(() => {
    if (workspace.messages.length > 0) {
      return workspace.messages;
    }
    return session.messages.map((message) => ({
      id: message.id,
      kind: message.role,
      text: message.content,
      timestamp: message.timestamp,
      fileChanges: message.fileChanges,
      terminalOutput: message.terminalOutput,
    }));
  }, [session.messages, workspace.messages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.({ behavior: "smooth" });
  }, [messages, workspace.pendingApprovals, workspace.isExecuting]);

  const handleSend = () => {
    if (workspace.isSubmitting) {
      return;
    }

    const trimmed = workspace.prompt.trim();
    if (!trimmed) {
      return;
    }
    workspace.handlePromptSubmit();
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
  };

  const handleTextareaInput = (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    workspace.setPrompt(event.target.value);
    const textarea = event.target;
    textarea.style.height = "auto";
    textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
  };

  const statusColor =
    session.status === "active"
      ? "text-emerald-400"
      : session.status === "systemError"
        ? "text-red-400"
        : "text-zinc-500";

  const machineStatusDot = {
    online: "bg-emerald-400",
    offline: "bg-zinc-600",
    reconnecting: "bg-amber-400",
    unknown: "bg-zinc-500",
  }[machine.status];

  const displayTitle = workspace.title || session.title || "线程工作区";
  const displayMachineName = workspace.machine?.name ?? machine.name;
  const canSend = workspace.canStartTurn && !workspace.isSubmitting;
  const selectedModel = workspace.runtimeModel || session.model;
  const selectedPermission = workspace.runtimeSandboxMode || "workspace-write";
  const modelOptions = workspace.runtimeModelOptions.length
    ? workspace.runtimeModelOptions
    : selectedModel
      ? [{ id: selectedModel, displayName: selectedModel, isDefault: true }]
      : [];
  const permissionOptions = workspace.runtimeSandboxOptions.length
    ? workspace.runtimeSandboxOptions
    : ["workspace-write", "danger-full-access", "read-only"];
  const executingMessageId =
    workspace.isExecuting
      ? messages.find(
          (message) =>
            message.kind === "agent" &&
            ((workspace.activeTurnId && message.turnId === workspace.activeTurnId) ||
              message.isPending),
        )?.id
      : undefined;
  const showExecutingPlaceholder = workspace.isExecuting && !executingMessageId;

  const handleApprovalDecision = (
    requestId: string,
    decision: ApprovalDecision,
    answers?: Record<string, string>,
  ) => {
    workspace.handleApprovalDecision(requestId, decision, answers);
  };

  return (
    <div className="flex flex-col h-full bg-zinc-950">
      <div className="flex-shrink-0 border-b border-zinc-800 bg-zinc-900/80 px-4 lg:px-6 py-3">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0">
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex items-center gap-1.5">
                <div className={`size-2 rounded-full ${machineStatusDot}`} />
                <span className="text-xs text-zinc-500 font-mono hidden sm:block">
                  {displayMachineName}
                </span>
              </div>
              <ChevronRight className="size-3.5 text-zinc-600 flex-shrink-0" />
              <span className="text-sm text-zinc-200 truncate">{displayTitle}</span>
            </div>
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            <div className="hidden sm:flex items-center gap-1.5">
              <div
                className={`size-1.5 rounded-full ${
                  session.status === "active" ? "bg-emerald-400 animate-pulse" : "bg-zinc-600"
                }`}
              />
              <span className={`text-xs ${statusColor}`}>{session.lastActivity}</span>
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
        {messages.map((message) => (
          <MessageBubble
            key={message.id}
            message={message}
            isExecuting={message.id === executingMessageId}
          />
        ))}

        {workspace.pendingApprovals.map((approval: WorkspaceApprovalCardViewModel) => (
          <div
            key={approval.requestId}
            className="border border-zinc-800/80 bg-zinc-900/60 rounded-2xl p-4"
          >
            <div className="flex items-center justify-between mb-3">
              <div>
                <p className="text-xs text-zinc-500">审批</p>
                <p className="text-sm text-zinc-200">待处理审批</p>
              </div>
              <span className="text-xs text-emerald-400">Live</span>
            </div>
            <div className="mb-3">
              <p className="text-sm text-zinc-200">{approval.title}</p>
              <p className="text-xs text-zinc-500">{approval.kind}</p>
            </div>
            {approval.questions.length > 0 ? (
              <div className="space-y-3">
                {approval.questions.map((question) => (
                  <div key={question.id} className="space-y-1">
                    <label
                      htmlFor={`${approval.requestId}-${question.id}`}
                      className="text-xs text-zinc-400"
                    >
                      {question.label}
                    </label>
                    {question.options?.length ? (
                      <select
                        id={`${approval.requestId}-${question.id}`}
                        aria-label={question.label}
                        value={question.value}
                        onChange={(event) =>
                          workspace.handleApprovalAnswerChange(
                            approval.requestId,
                            question.id,
                            event.target.value,
                          )
                        }
                        className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:outline-none focus:border-blue-500"
                      >
                        {question.options.map((option) => (
                          <option key={option} value={option}>
                            {option}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <textarea
                        id={`${approval.requestId}-${question.id}`}
                        aria-label={question.label}
                        value={question.value}
                        onChange={(event) =>
                          workspace.handleApprovalAnswerChange(
                            approval.requestId,
                            question.id,
                            event.target.value,
                          )
                        }
                        rows={2}
                        className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:outline-none focus:border-blue-500 resize-none"
                      />
                    )}
                  </div>
                ))}
              </div>
            ) : null}
            <div className="flex items-center gap-2 mt-4">
              <button
                type="button"
                aria-label="Accept"
                onClick={() =>
                  handleApprovalDecision(
                    approval.requestId,
                    "accept",
                    approval.questions.length > 0
                      ? Object.fromEntries(
                          approval.questions.map((question) => [question.id, question.value]),
                        )
                      : undefined,
                  )
                }
                className="flex-1 px-3 py-2 rounded-lg bg-emerald-600 text-white text-xs hover:bg-emerald-500 transition-colors"
              >
                接受
              </button>
              <button
                type="button"
                aria-label="Decline"
                onClick={() => handleApprovalDecision(approval.requestId, "decline")}
                className="flex-1 px-3 py-2 rounded-lg bg-zinc-800 text-zinc-200 text-xs hover:bg-zinc-700 transition-colors"
              >
                拒绝
              </button>
              <button
                type="button"
                aria-label="Cancel"
                onClick={() => handleApprovalDecision(approval.requestId, "cancel")}
                className="flex-1 px-3 py-2 rounded-lg bg-zinc-800 text-zinc-200 text-xs hover:bg-zinc-700 transition-colors"
              >
                取消
              </button>
            </div>
          </div>
        ))}

        {showExecutingPlaceholder ? (
          <div className="flex gap-3">
            <div className="size-8 rounded-full bg-gradient-to-br from-violet-600 to-blue-600 flex items-center justify-center flex-shrink-0">
              <Bot className="size-4 text-white" />
            </div>
            <ExecutingIndicator />
          </div>
        ) : null}

        <div ref={messagesEndRef} />
      </div>

      <div className="flex-shrink-0 border-t border-zinc-800 bg-zinc-900/80 p-4 lg:px-6 relative z-10">
        <div className="bg-zinc-800/60 border border-zinc-700/60 rounded-xl focus-within:border-zinc-600 transition-colors overflow-visible">
          <textarea
            ref={textareaRef}
            value={workspace.prompt}
            onChange={handleTextareaInput}
            onKeyDown={handleKeyDown}
            placeholder={`向 ${session.agentName} 发送指令...`}
            rows={1}
            aria-label="Prompt"
            className="w-full bg-transparent px-4 pt-3 pb-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:outline-none resize-none min-h-[44px] max-h-[200px]"
          />
          <div className="flex items-center justify-between px-3 pb-2.5 pt-1 relative">
            <div className="flex items-center gap-2 relative z-50">
              <div className="relative">
                <button
                  disabled={workspace.isUpdatingRuntimeSettings || modelOptions.length === 0}
                  onClick={() => {
                    setShowModelDrop(!showModelDrop);
                    setShowPermDrop(false);
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-700/60 hover:bg-zinc-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-xs text-zinc-300"
                >
                  <Circle className="size-2.5 text-violet-400 fill-violet-400" />
                  <span className="max-w-[100px] truncate">{selectedModel || "未配置"}</span>
                  <ChevronDown className="size-3 text-zinc-500" />
                </button>
                {showModelDrop ? (
                  <div className="absolute bottom-full mb-2 left-0 bg-zinc-800 border border-zinc-700 rounded-lg py-1 min-w-[180px] shadow-2xl">
                    {modelOptions.map((model) => (
                      <button
                        key={model.id}
                        onClick={async () => {
                          await workspace.handleRuntimeModelChange(model.id);
                          setShowModelDrop(false);
                        }}
                        className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                          selectedModel === model.id
                            ? "text-zinc-50 bg-zinc-700"
                            : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50"
                        }`}
                      >
                        {model.displayName || model.id}
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>

              <div className="relative hidden sm:block">
                <button
                  disabled={workspace.isUpdatingRuntimeSettings || permissionOptions.length === 0}
                  onClick={() => {
                    setShowPermDrop(!showPermDrop);
                    setShowModelDrop(false);
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-700/60 hover:bg-zinc-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-xs text-zinc-300"
                >
                  <span className="text-emerald-400 text-[10px]">●</span>
                  <span>{formatSandboxMode(selectedPermission)}</span>
                  <ChevronDown className="size-3 text-zinc-500" />
                </button>
                {showPermDrop ? (
                  <div className="absolute bottom-full mb-2 left-0 bg-zinc-800 border border-zinc-700 rounded-lg py-1 min-w-[140px] shadow-2xl">
                    {permissionOptions.map((permission) => (
                      <button
                        key={permission}
                        onClick={async () => {
                          await workspace.handleRuntimePermissionChange(permission);
                          setShowPermDrop(false);
                        }}
                        className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                          selectedPermission === permission
                            ? "text-zinc-50 bg-zinc-700"
                            : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50"
                        }`}
                      >
                        {formatSandboxMode(permission)}
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>

            <button
              onClick={handleSend}
              aria-label="Send prompt"
              disabled={!workspace.prompt.trim() || !canSend}
              className="size-8 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-700 disabled:text-zinc-500 text-white flex items-center justify-center transition-colors flex-shrink-0"
            >
              <Send className="size-3.5" />
            </button>
          </div>
        </div>
        <p className="text-center text-xs text-zinc-700 mt-2">
          Shift+Enter 换行 · Enter 发送 · Agent 在 {displayMachineName} 上运行
        </p>
      </div>
    </div>
  );
}
