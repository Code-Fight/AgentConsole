import { useParams } from "react-router-dom";
import { ThreadWorkspacePageView } from "../design";
import { useThreadWorkspace } from "../gateway/use-thread-workspace";

export function ThreadWorkspacePage() {
  const { threadId = "" } = useParams();
  const vm = useThreadWorkspace(threadId);

  return (
    <ThreadWorkspacePageView
      title={vm.title}
      subtitle={vm.subtitle}
      error={vm.error}
      machine={vm.machine}
      messages={vm.messages}
      pendingApprovals={vm.pendingApprovals}
      activeTurnId={vm.activeTurnId}
      prompt={vm.prompt}
      steerPrompt={vm.steerPrompt}
      isSubmitting={vm.isSubmitting}
      canStartTurn={vm.canStartTurn}
      canSteerTurn={vm.canSteerTurn}
      canInterruptTurn={vm.canInterruptTurn}
      onPromptChange={vm.setPrompt}
      onSteerPromptChange={vm.setSteerPrompt}
      onPromptSubmit={() => void vm.handlePromptSubmit()}
      onSteerSubmit={() => void vm.handleSteerSubmit()}
      onInterrupt={() => void vm.handleInterrupt()}
      onApprovalAnswerChange={vm.handleApprovalAnswerChange}
      onApprovalDecision={(requestId, decision, answers) =>
        void vm.handleApprovalDecision(requestId, decision, answers)
      }
    />
  );
}
