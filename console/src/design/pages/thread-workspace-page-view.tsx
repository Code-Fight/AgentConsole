import { useParams } from "react-router-dom";
import { SessionChatView } from "../components/session-chat-view";

export function ThreadWorkspacePageView() {
  const { threadId = "thread-alpha" } = useParams();

  return (
    <SessionChatView
      title={`Thread ${threadId}`}
      subtitle="Design workspace import. Live turn streaming and approvals will be wired by the Gateway adapter layer."
    />
  );
}
