import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import type { ThreadWorkspaceViewModel } from "../hooks/use-thread-workspace";
import type {
  ThreadMachineViewModel,
  ThreadSessionViewModel,
  WorkspaceMessageViewModel,
} from "../model/thread-view-model";
import SessionChat from "./session-chat";

const baseSession: ThreadSessionViewModel = {
  id: "thread-1",
  title: "Thread 1",
  agentName: "Codex",
  model: "gpt-5.4",
  status: "active",
  lastActivity: "进行中",
  messages: [],
};

const baseMachine: ThreadMachineViewModel = {
  id: "machine-1",
  name: "Machine 1",
  status: "online",
  runtimeStatus: "running",
  agents: [],
  sessions: [baseSession],
};

function buildWorkspace(
  messages: WorkspaceMessageViewModel[],
  overrides: Partial<ThreadWorkspaceViewModel> = {},
): ThreadWorkspaceViewModel {
  return {
    title: "Thread 1",
    subtitle: "thread-1",
    error: null,
    machine: {
      statusLabel: "在线",
      runtimeLabel: "running",
      name: "Machine 1",
    },
    messages,
    pendingApprovals: [],
    activeTurnId: null,
    prompt: "",
    steerPrompt: "",
    isSubmitting: false,
    isExecuting: false,
    runtimeModel: "gpt-5.4",
    runtimeApprovalPolicy: "",
    runtimeSandboxMode: "workspace-write",
    runtimeModelOptions: [],
    runtimeSandboxOptions: [],
    isUpdatingRuntimeSettings: false,
    canStartTurn: true,
    canSteerTurn: false,
    canInterruptTurn: false,
    setPrompt: vi.fn(),
    setSteerPrompt: vi.fn(),
    handlePromptSubmit: vi.fn(),
    handleRuntimeModelChange: vi.fn(),
    handleRuntimePermissionChange: vi.fn(),
    handleSteerSubmit: vi.fn(),
    handleInterrupt: vi.fn(),
    handleApprovalAnswerChange: vi.fn(),
    handleApprovalDecision: vi.fn(),
    ...overrides,
  } as ThreadWorkspaceViewModel;
}

test("renders agent markdown as formatted content", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace([
        {
          id: "turn-1:1",
          kind: "agent",
          turnId: "turn-1",
          text: [
            "**重点**",
            "",
            "| 产品 | 状态 |",
            "| --- | --- |",
            "| Codex | 可用 |",
          ].join("\n"),
        },
      ])}
    />,
  );

  expect(screen.getByRole("table")).toBeInTheDocument();
  expect(screen.getByText("重点").tagName.toLowerCase()).toBe("strong");
});

test("attaches executing indicator to the active streaming agent message", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace(
        [
          {
            id: "turn-1:1",
            kind: "agent",
            turnId: "turn-1",
            text: "正在生成第一条回复",
          },
        ],
        {
          activeTurnId: "turn-1",
          isExecuting: true,
        },
      )}
    />,
  );

  const status = screen.getByText("正在执行...");
  const activeMessage = screen.getByText("正在生成第一条回复");
  const bubble = screen.getByTestId("agent-message-bubble");
  expect(
    status.compareDocumentPosition(activeMessage) & Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  expect(within(bubble).getByText("正在执行...")).toBeInTheDocument();
  expect(within(bubble).getByText("正在生成第一条回复")).toBeInTheDocument();
});

test("attaches executing indicator to a pending agent placeholder", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace(
        [
          {
            id: "pending-agent:1",
            kind: "agent",
            text: "",
            isPending: true,
          },
        ],
        {
          isExecuting: true,
        },
      )}
    />,
  );

  const bubble = screen.getByTestId("agent-message-bubble");
  expect(within(bubble).getByText("正在执行...")).toBeInTheDocument();
});


test("keeps completed progress collapsed beside the final answer", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace([
        {
          id: "turn-1:1",
          kind: "agent",
          turnId: "turn-1",
          text: "最终报告",
          progressText: "我先查资料",
        },
      ])}
    />,
  );

  expect(screen.getByText("最终报告")).toBeInTheDocument();
  expect(screen.getByText("已处理")).toBeInTheDocument();
  expect(screen.queryByText("我先查资料")).not.toBeInTheDocument();

  fireEvent.click(screen.getByText("已处理"));

  expect(screen.getByText("我先查资料")).toBeInTheDocument();
});

test("renders streaming process text without a synthetic progress heading", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace([
        {
          id: "agent:turn-1",
          kind: "agent",
          turnId: "turn-1",
          text: "完成了",
          progressText: "你要我在当前目录创建",
        },
      ])}
    />,
  );

  fireEvent.click(screen.getByText("已处理"));

  expect(screen.queryByText("过程")).not.toBeInTheDocument();
  expect(screen.getByText("你要我在当前目录创建")).toBeInTheDocument();
});

test("keeps the completed process container when the progress heading is omitted", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace([
        {
          id: "agent:turn-1",
          kind: "agent",
          turnId: "turn-1",
          text: "已创建文件",
          progressText: "我会在当前目录创建名为 `测试A` 的文件。",
        },
      ])}
    />,
  );

  expect(screen.getByText("已处理")).toBeInTheDocument();
  expect(screen.queryByText("我会在当前目录创建名为")).not.toBeInTheDocument();

  fireEvent.click(screen.getByText("已处理"));

  expect(screen.queryByText("过程")).not.toBeInTheDocument();
  const inlineCode = screen.getByText("测试A");
  expect(inlineCode).toBeInTheDocument();
  expect(inlineCode.closest("p")).toHaveTextContent("我会在当前目录创建名为");
});

test("renders timeline command output in a terminal block without splitting the answer", () => {
  render(
    <SessionChat
      session={baseSession}
      machine={baseMachine}
      workspace={buildWorkspace([
        {
          id: "agent:turn-1",
          kind: "agent",
          turnId: "turn-1",
          text: "最终报告",
          progressText: "运行测试",
          terminalOutput: "=== RUN TestTimeline\n--- PASS: TestTimeline (0.01s)\nPASS",
        },
      ])}
    />,
  );

  expect(screen.getAllByTestId("agent-message-bubble")).toHaveLength(1);
  expect(screen.getByText("最终报告")).toBeInTheDocument();
  expect(screen.getByText("终端输出")).toBeInTheDocument();
  expect(screen.getByText("=== RUN TestTimeline")).toBeInTheDocument();
  expect(screen.getByText("--- PASS: TestTimeline (0.01s)")).toBeInTheDocument();
});
