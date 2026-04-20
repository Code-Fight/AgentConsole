import { buildThreadApiPath, http } from "../../../common/api/http";
import type {
  ApprovalDecision,
  CreateThreadResponse,
  MachineDetailResponse,
  MachineListResponse,
  StartTurnResponse,
  ThreadDeleteResponse,
  ThreadDetailResponse,
  ThreadResumeResponse,
  ThreadRuntimeSettingsResponse,
  ThreadRuntimeUpdateRequest,
  ThreadListResponse,
} from "../../../common/api/types";

interface CreateThreadInput {
  machineId: string;
  title: string;
  agentId?: string;
}

export function listThreads() {
  return http<ThreadListResponse>("/threads");
}

export function listMachines() {
  return http<MachineListResponse>("/machines");
}

export function createThread(input: CreateThreadInput) {
  return http<CreateThreadResponse>("/threads", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });
}

export function renameThread(threadId: string, title: string) {
  return http<void>(buildThreadApiPath(threadId), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title }),
  });
}

export function archiveThread(threadId: string) {
  return http<void>(buildThreadApiPath(threadId, "archive"), {
    method: "POST",
  });
}

export function resumeThread(threadId: string) {
  return http<ThreadResumeResponse>(buildThreadApiPath(threadId, "resume"), {
    method: "POST",
  });
}

export function deleteThread(threadId: string) {
  return http<ThreadDeleteResponse>(buildThreadApiPath(threadId), {
    method: "DELETE",
  });
}

export function getThreadDetail(threadId: string) {
  return http<ThreadDetailResponse>(buildThreadApiPath(threadId));
}

export function getThreadRuntimeSettings(threadId: string) {
  return http<ThreadRuntimeSettingsResponse>(buildThreadApiPath(threadId, "runtime"));
}

export function updateThreadRuntimeSettings(
  threadId: string,
  patch: ThreadRuntimeUpdateRequest,
) {
  return http<ThreadRuntimeSettingsResponse>(buildThreadApiPath(threadId, "runtime"), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(patch),
  });
}

export function getMachineDetail(machineId: string) {
  return http<MachineDetailResponse>(`/machines/${encodeURIComponent(machineId)}`);
}

export function respondToApproval(
  requestId: string,
  decision: ApprovalDecision,
  answers?: Record<string, string>,
) {
  return http<void>(`/approvals/${encodeURIComponent(requestId)}/respond`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      decision,
      ...(answers ? { answers } : {}),
    }),
  });
}

export function startThreadTurn(threadId: string, input: string) {
  return http<StartTurnResponse>(buildThreadApiPath(threadId, "turns"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ input }),
  });
}

export function steerThreadTurn(threadId: string, turnId: string, input: string) {
  return http<StartTurnResponse>(buildThreadApiPath(threadId, `turns/${turnId}/steer`), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ input }),
  });
}

export function interruptThreadTurn(threadId: string, turnId: string) {
  return http<void>(buildThreadApiPath(threadId, `turns/${turnId}/interrupt`), {
    method: "POST",
  });
}
