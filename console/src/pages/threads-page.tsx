import { FormEvent, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { http } from "../common/api/http";
import type {
  CreateThreadResponse,
  EventEnvelope,
  MachineListResponse,
  MachineSummary,
  ThreadDeleteResponse,
  ThreadListResponse,
  ThreadSummary
} from "../common/api/types";
import { connectConsoleSocket } from "../common/api/ws";

function parseEnvelope(raw: string): EventEnvelope | null {
  try {
    return JSON.parse(raw) as EventEnvelope;
  } catch {
    return null;
  }
}

export function ThreadsPage() {
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [machines, setMachines] = useState<Record<string, MachineSummary>>({});
  const [error, setError] = useState<string | null>(null);
  const [machineId, setMachineId] = useState("");
  const [title, setTitle] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadThreads() {
      try {
        const [threadResponse, machineResponse] = await Promise.all([
          http<ThreadListResponse>("/threads"),
          http<MachineListResponse>("/machines")
        ]);
        if (!cancelled) {
          setThreads(threadResponse.items);
          setMachines(
            Object.fromEntries(machineResponse.items.map((machine) => [machine.id, machine])),
          );
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setThreads([]);
          setMachines({});
          setError("Unable to load live threads.");
        }
      }
    }

    void loadThreads();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => connectConsoleSocket(undefined, (event) => {
    const envelope = parseEnvelope(event.data);
    if (!envelope) {
      return;
    }

    if (envelope.name !== "thread.updated" && envelope.name !== "machine.updated") {
      return;
    }

    void (async () => {
      try {
        const [threadResponse, machineResponse] = await Promise.all([
          http<ThreadListResponse>("/threads"),
          http<MachineListResponse>("/machines")
        ]);
        setThreads(threadResponse.items);
        setMachines(
          Object.fromEntries(machineResponse.items.map((machine) => [machine.id, machine])),
        );
        setError(null);
      } catch {
        setThreads([]);
        setMachines({});
        setError("Unable to load live threads.");
      }
    })();
  }), []);

  async function refreshThreads() {
    const [threadResponse, machineResponse] = await Promise.all([
      http<ThreadListResponse>("/threads"),
      http<MachineListResponse>("/machines")
    ]);
    setThreads(threadResponse.items);
    setMachines(
      Object.fromEntries(machineResponse.items.map((machine) => [machine.id, machine])),
    );
    setError(null);
  }

  async function handleCreateThread(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const nextMachineID = machineId.trim();
    const nextTitle = title.trim();
    if (nextMachineID === "" || nextTitle === "") {
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await http<CreateThreadResponse>("/threads", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          machineId: nextMachineID,
          title: nextTitle
        })
      });
      setTitle("");
      await refreshThreads();
    } catch {
      setError("Unable to create thread.");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleArchive(threadId: string) {
    setError(null);

    try {
      await http<void>(`/threads/${encodeURIComponent(threadId)}/archive`, {
        method: "POST"
      });
      await refreshThreads();
    } catch {
      setError("Unable to archive thread.");
    }
  }

  async function handleResume(threadId: string) {
    setError(null);

    try {
      await http<void>(`/threads/${encodeURIComponent(threadId)}/resume`, {
        method: "POST"
      });
      await refreshThreads();
    } catch {
      setError("Unable to resume thread.");
    }
  }

  async function handleDelete(threadId: string) {
    setError(null);

    try {
      await http<ThreadDeleteResponse>(`/threads/${encodeURIComponent(threadId)}`, {
        method: "DELETE"
      });
      await refreshThreads();
    } catch {
      setError("Unable to delete thread.");
    }
  }

  return (
    <section>
      <h1>Threads</h1>
      {error ? <p>{error}</p> : null}
      <form onSubmit={handleCreateThread}>
        <label htmlFor="thread-machine-id">Machine ID</label>
        <input
          id="thread-machine-id"
          aria-label="Machine ID"
          value={machineId}
          onChange={(event) => setMachineId(event.target.value)}
        />
        <label htmlFor="thread-title">Title</label>
        <input
          id="thread-title"
          aria-label="Title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />
        <button type="submit" disabled={isSubmitting}>
          Create thread
        </button>
      </form>
      {threads.length === 0 ? <p>No threads available.</p> : null}
      <ul>
        {threads.map((thread) => (
          <li key={thread.threadId}>
            <Link to={`/threads/${thread.threadId}`}>{thread.title || thread.threadId}</Link>
            <span>{thread.status}</span>
            <span>
              {machines[thread.machineId]?.status ?? "unknown"} / {machines[thread.machineId]?.runtimeStatus ?? "unknown"}
            </span>
            <button type="button" onClick={() => void handleResume(thread.threadId)}>
              Resume {thread.title || thread.threadId}
            </button>
            <button type="button" onClick={() => void handleArchive(thread.threadId)}>
              Archive {thread.title || thread.threadId}
            </button>
            <button type="button" onClick={() => void handleDelete(thread.threadId)}>
              Delete {thread.title || thread.threadId}
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}
