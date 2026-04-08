import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { http } from "../common/api/http";
import type { ThreadListResponse, ThreadSummary } from "../common/api/types";

export function ThreadsPage() {
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadThreads() {
      try {
        const response = await http<ThreadListResponse>("/threads");
        if (!cancelled) {
          setThreads(response.items);
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setThreads([]);
          setError("Unable to load live threads.");
        }
      }
    }

    void loadThreads();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section>
      <h1>Threads</h1>
      {error ? <p>{error}</p> : null}
      {threads.length === 0 ? <p>No threads available.</p> : null}
      <ul>
        {threads.map((thread) => (
          <li key={thread.threadId}>
            <Link to={`/threads/${thread.threadId}`}>{thread.title || thread.threadId}</Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
