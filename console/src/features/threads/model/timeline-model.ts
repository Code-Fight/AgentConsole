import type {
  AgentTimelineEvent,
  ApprovalRequiredPayload,
  EventEnvelope,
  TimelineEventPayload,
} from "../../../common/api/types";
import type { WorkspaceMessageViewModel } from "./thread-view-model";

export function isTimelineEventEnvelope(
  envelope: EventEnvelope,
): envelope is EventEnvelope<TimelineEventPayload> {
  return envelope.name === "timeline.event" && Boolean((envelope.payload as TimelineEventPayload)?.event);
}

export function getTimelineThreadId(envelope: EventEnvelope): string | null {
  if (!isTimelineEventEnvelope(envelope)) {
    return null;
  }
  return envelope.payload.event.threadId || null;
}

export function timelineEventToApprovalRequired(event: AgentTimelineEvent): ApprovalRequiredPayload | null {
  if (event.eventType !== "approval.requested" || !event.approval?.requestId) {
    return null;
  }
  return {
    requestId: event.approval.requestId,
    threadId: event.threadId,
    turnId: event.turnId,
    itemId: event.itemId,
    kind: event.approval.kind,
    reason: event.approval.reason,
    command: event.approval.title,
    questions: event.approval.questions?.map((question) => ({
      id: question.id,
      text: question.label,
      options: question.options,
    })),
  };
}

export function mergeTimelineEventIntoMessages(
  current: WorkspaceMessageViewModel[],
  event: AgentTimelineEvent,
): WorkspaceMessageViewModel[] {
  if (!event.turnId && event.eventType !== "system.event") {
    return current;
  }

  if (event.role === "user" || event.phase === "input") {
    const text = timelineDisplayText(event);
    if (!text) {
      return current;
    }
    return upsertMessage(current, {
      id: `user:${event.turnId ?? event.eventId}:${event.itemId ?? event.sequence}`,
      kind: "user",
      text,
      turnId: event.turnId,
    });
  }

  if (event.eventType === "turn.failed") {
    return upsertMessage(current, {
      id: `failed:${event.turnId}`,
      kind: "system",
      text: event.error?.message ? `执行失败：${event.error.message}` : "执行失败",
      turnId: event.turnId,
    });
  }

  if (event.eventType === "system.event" || event.phase === "system") {
    const text = timelineSystemText(event);
    if (!text) {
      return current;
    }
    return upsertMessage(current, {
      id: `system:${event.eventId}`,
      kind: "system",
      text,
      turnId: event.turnId,
    });
  }

  if (!event.turnId) {
    return current;
  }

  const messageID = `agent:${event.turnId}`;
  const existing = current.find(
    (message) =>
      message.id === messageID ||
      (message.turnId === event.turnId && message.kind === "agent"),
  );
  const lifecycleOnly =
    event.eventType === "turn.started" ||
    event.eventType === "turn.completed" ||
    event.eventType === "item.started" ||
    event.eventType === "item.completed";

  if (!existing && lifecycleOnly && !timelineHasDisplayableProcessContent(event)) {
    return current;
  }

  const base: WorkspaceMessageViewModel =
    existing ?? {
      id: messageID,
      kind: "agent",
      text: "",
      turnId: event.turnId,
    };

  const next = mergeAgentTimelineEvent(base, event);
  if (next === base && existing) {
    return current;
  }
  if (!existing) {
    return [...current, next];
  }
  return current.map((message) => (message === existing ? next : message));
}

export function timelineEventMarksTurnCompleted(event: AgentTimelineEvent): boolean {
  return event.eventType === "turn.completed" || event.eventType === "turn.failed";
}

function mergeAgentTimelineEvent(
  message: WorkspaceMessageViewModel,
  event: AgentTimelineEvent,
): WorkspaceMessageViewModel {
  if (event.eventType === "turn.started") {
    return clearPendingFlag({ ...message, turnId: event.turnId });
  }

  if (event.eventType === "turn.completed") {
    return withOptionalProgressText(clearPendingFlag(message), cleanProgressText(message.progressText));
  }

  if (event.phase === "final" && event.itemType === "message") {
    const text = timelineDisplayText(event);
    if (!text) {
      return message;
    }
    return withOptionalProgressText(
      {
        ...clearPendingFlag(message),
        text: mergeText(message.text, text, event.content?.appendMode),
      },
      cleanProgressText(message.progressText),
    );
  }

  const processText = timelineProcessText(event);
  if (!processText) {
    return message;
  }

  if (event.itemType === "command" && event.content?.contentType === "terminal") {
    return {
      ...clearPendingFlag(message),
      terminalOutput: mergeText(message.terminalOutput ?? "", timelineDisplayText(event), event.content?.appendMode),
      progressText: appendProgress(message.progressText, processText, event),
    };
  }

  return {
    ...clearPendingFlag(message),
    progressText: appendProgress(message.progressText, processText, event),
  };
}

function clearPendingFlag(message: WorkspaceMessageViewModel): WorkspaceMessageViewModel {
  const { isPending: _isPending, ...next } = message;
  return next;
}

function withOptionalProgressText(
  message: WorkspaceMessageViewModel,
  progressText: string | undefined,
): WorkspaceMessageViewModel {
  if (progressText) {
    return { ...message, progressText };
  }
  const { progressText: _progressText, ...next } = message;
  return next;
}

function upsertMessage(
  current: WorkspaceMessageViewModel[],
  next: WorkspaceMessageViewModel,
): WorkspaceMessageViewModel[] {
  const existing = current.find((message) => message.id === next.id);
  if (!existing) {
    return [...current, next];
  }
  return current.map((message) => (message.id === next.id ? { ...message, ...next } : message));
}

function timelineDisplayText(event: AgentTimelineEvent): string {
  const content = event.content;
  if (!content) {
    return "";
  }
  if (typeof content.delta === "string") {
    return content.delta;
  }
  if (typeof content.text === "string") {
    return content.text;
  }
  if (typeof content.snapshot === "string") {
    return content.snapshot;
  }
  if (content.snapshot !== undefined && content.snapshot !== null) {
    return JSON.stringify(content.snapshot, null, 2);
  }
  return "";
}

function timelineProcessText(event: AgentTimelineEvent): string {
  const text = timelineDisplayText(event);
  const label = timelineItemLabel(event);

  if (event.eventType === "item.started") {
    if (isQuietLifecycleItem(event)) {
      return "";
    }
    return `**${label}** 开始。`;
  }
  if (event.eventType === "item.completed") {
    if (isQuietLifecycleItem(event) && !text) {
      return "";
    }
    if (!text) {
      return `**${label}** 完成。`;
    }
  }
  if (event.eventType === "item.failed") {
    return `**${label}** 失败：${event.error?.message ?? "未知错误"}`;
  }
  if (event.eventType === "approval.resolved") {
    return `**审批** 已处理：${event.approval?.decision ?? "completed"}`;
  }
  if (event.eventType === "approval.requested") {
    return `**审批** ${event.approval?.title ?? event.approval?.kind ?? "待处理"}`;
  }
  if (text) {
    if (event.itemType === "message" && event.phase === "progress") {
      return text;
    }
    return `**${label}**\n\n${text}`;
  }
  if (event.tool?.name || event.tool?.kind) {
    return `**${label}** ${event.tool.name ?? event.tool.kind}`;
  }
  return "";
}

function timelineHasDisplayableProcessContent(event: AgentTimelineEvent): boolean {
  return Boolean(timelineProcessText(event));
}

function timelineSystemText(event: AgentTimelineEvent): string {
  if (event.itemType === "context") {
    return "上下文已压缩";
  }
  if (event.itemType === "mode_change") {
    return timelineDisplayText(event) || "模式已变更";
  }
  return timelineDisplayText(event);
}

function timelineItemLabel(event: AgentTimelineEvent): string {
  switch (event.itemType) {
    case "reasoning":
      return "分析";
    case "plan":
      return "计划";
    case "command":
      return "终端输出";
    case "file_change":
      return "文件变更";
    case "web_search":
      return "网页搜索";
    case "mcp_tool":
      return "MCP 工具";
    case "tool":
      return "工具调用";
    case "subagent":
      return "子 Agent";
    case "image":
      return "图片";
    case "artifact":
      return "产物";
    case "context":
      return "上下文";
    case "mode_change":
      return "模式";
    case "message":
      return event.phase === "analysis" ? "分析" : "过程";
    default:
      return event.itemType || "事件";
  }
}

function mergeText(current: string, next: string, appendMode?: string): string {
  if (appendMode === "replace" || appendMode === "snapshot") {
    return next;
  }
  return `${current}${next}`;
}

function appendProgress(
  current: string | undefined,
  next: string,
  event?: AgentTimelineEvent,
): string {
  if (event?.itemType === "message" && event.phase === "progress") {
    const text = timelineDisplayText(event);
    if (
      event.eventType === "item.delta" &&
      event.content?.appendMode === "append" &&
      text
    ) {
      return appendPlainProgressDelta(current, text);
    }
    if (text) {
      return mergePlainProgressSnapshot(current, text);
    }
  }

  if (
    event?.eventType === "item.delta" &&
    event.content?.appendMode === "append" &&
    timelineDisplayText(event)
  ) {
    return appendProgressDeltaToSection(
      current,
      timelineItemLabel(event),
      timelineDisplayText(event),
    );
  }
  if (!current?.trim()) {
    return next;
  }
  return `${current.trimEnd()}\n\n${next}`;
}

function appendPlainProgressDelta(current: string | undefined, delta: string): string {
  const existing = current?.trimEnd() ?? "";
  return `${existing}${delta}`;
}

function mergePlainProgressSnapshot(current: string | undefined, snapshot: string): string {
  const existing = current?.trimEnd() ?? "";
  const next = snapshot.trim();
  if (!existing) {
    return snapshot;
  }
  if (!next) {
    return existing;
  }
  if (existing === next || existing.includes(next)) {
    return existing;
  }
  if (next.startsWith(existing)) {
    return snapshot;
  }

  const sections = existing.split("\n\n");
  for (let index = 0; index < sections.length; index += 1) {
    const section = sections[index] ?? "";
    if (!section) {
      continue;
    }
    if (section === next || section.includes(next)) {
      return existing;
    }
    if (next.startsWith(section)) {
      sections[index] = snapshot;
      return sections.join("\n\n");
    }
  }

  return `${existing}\n\n${snapshot}`;
}

function appendProgressDeltaToSection(
  current: string | undefined,
  label: string,
  delta: string,
): string {
  const header = `**${label}**\n\n`;
  const existing = current?.trimEnd() ?? "";
  if (!existing) {
    return `${header}${delta}`;
  }

  const prefixedHeader = `\n\n${header}`;
  let headerStart = -1;
  if (existing.startsWith(header)) {
    headerStart = 0;
  } else {
    const sectionStart = existing.lastIndexOf(prefixedHeader);
    if (sectionStart >= 0) {
      headerStart = sectionStart + 2;
    }
  }
  if (headerStart >= 0 && existing.slice(headerStart).startsWith(header)) {
    const bodyStart = headerStart + header.length;
    const body = existing.slice(bodyStart);
    if (!body.includes("\n\n**")) {
      return `${existing}${delta}`;
    }
  }

  return `${existing}\n\n${header}${delta}`;
}

function isQuietLifecycleItem(event: AgentTimelineEvent): boolean {
  return event.itemType === "message" || event.itemType === "reasoning" || event.itemType === "plan";
}

function cleanProgressText(text?: string): string | undefined {
  const cleaned = (text ?? "").replace(/\n\n正在分析\.\.\./g, "").trim();
  return cleaned ? cleaned : undefined;
}
