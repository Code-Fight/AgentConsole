import { useParams } from "react-router-dom";
import { ThreadShell } from "../components/thread-shell";

export function ThreadWorkspacePage() {
  const { threadId = "" } = useParams();

  return <ThreadShell threadId={threadId} />;
}
