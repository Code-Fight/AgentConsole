import { FormEvent } from "react";
import type { ApprovalDecision } from "../../common/api/types";
import type {
  WorkspaceApprovalCardViewModel,
  WorkspaceMessageViewModel,
} from "../../gateway/thread-view-model";

interface ThreadWorkspacePageViewProps {
  title: string;
  subtitle: string;
  error?: string | null;
  machine?: {
    statusLabel: string;
    runtimeLabel: string;
    name: string;
  } | null;
  messages: WorkspaceMessageViewModel[];
  pendingApprovals: WorkspaceApprovalCardViewModel[];
  activeTurnId?: string | null;
  prompt: string;
  steerPrompt: string;
  isSubmitting?: boolean;
  canStartTurn: boolean;
  canSteerTurn: boolean;
  canInterruptTurn: boolean;
  onPromptChange: (value: string) => void;
  onSteerPromptChange: (value: string) => void;
  onPromptSubmit: () => void;
  onSteerSubmit: () => void;
  onInterrupt: () => void;
  onApprovalAnswerChange: (requestId: string, questionId: string, value: string) => void;
  onApprovalDecision: (
    requestId: string,
    decision: ApprovalDecision,
    answers?: Record<string, string>,
  ) => void;
}

export function ThreadWorkspacePageView(props: ThreadWorkspacePageViewProps) {
  function handlePromptSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    props.onPromptSubmit();
  }

  function handleSteerSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    props.onSteerSubmit();
  }

  return (
    <section className="thread-workspace-page page">
      <header className="page-header">
        <span className="page-kicker">Thread Workspace</span>
        <h1>{props.title}</h1>
        <p>{props.subtitle}</p>
        {props.machine ? (
          <div className="thread-workspace-meta">
            <span className="meta-pill">{props.machine.statusLabel}</span>
            <span className="meta-pill">runtime {props.machine.runtimeLabel}</span>
            <span className="meta-pill">{props.machine.name}</span>
          </div>
        ) : null}
      </header>

      {props.error ? <p>{props.error}</p> : null}

      <div className="thread-workspace-layout">
        <section className="workspace-conversation-card">
          <div className="chat-stream-header">
            <h2>Session Chat</h2>
            <span className="session-panel-count">
              {props.messages.length + props.pendingApprovals.length}
            </span>
          </div>

          {props.messages.length === 0 && props.pendingApprovals.length === 0 ? (
            <div className="thread-workspace-empty">
              <strong>Waiting for live Gateway events.</strong>
              <p>Turn deltas, approvals, and completion status will stream into this timeline once the thread is active.</p>
            </div>
          ) : (
            <ul className="chat-stream workspace-stream-list">
              {props.messages.map((message) => (
                <li key={message.id} className={`chat-stream-item chat-stream-${message.kind}`}>
                  {message.text}
                </li>
              ))}

              {props.pendingApprovals.map((approval) => (
                <li key={approval.requestId} className="chat-stream-item chat-stream-info">
                  <div className="thread-workspace-approval-card">
                    <div className="thread-workspace-approval-head">
                      <div>
                        <span className="page-kicker">审批</span>
                        <h2>待处理审批</h2>
                      </div>
                      <span className="session-panel-count">Live</span>
                    </div>
                    <div className="approval-item-copy">
                      <strong>{approval.title}</strong>
                      <small>{approval.kind}</small>
                    </div>
                    {approval.questions.length > 0 ? (
                      <div>
                        {approval.questions.map((question) => (
                          <div key={question.id} className="approval-question-field">
                            <label htmlFor={`${approval.requestId}-${question.id}`}>{question.label}</label>
                            {question.options?.length ? (
                              <select
                                id={`${approval.requestId}-${question.id}`}
                                aria-label={question.label}
                                value={question.value}
                                onChange={(event) =>
                                  props.onApprovalAnswerChange(
                                    approval.requestId,
                                    question.id,
                                    event.target.value,
                                  )
                                }
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
                                  props.onApprovalAnswerChange(
                                    approval.requestId,
                                    question.id,
                                    event.target.value,
                                  )
                                }
                              />
                            )}
                          </div>
                        ))}
                      </div>
                    ) : null}
                    <div className="approval-actions">
                      <button
                        type="button"
                        aria-label="Accept"
                        onClick={() =>
                          props.onApprovalDecision(
                            approval.requestId,
                            "accept",
                            approval.questions.length > 0
                              ? Object.fromEntries(
                                  approval.questions.map((question) => [question.id, question.value]),
                                )
                              : undefined,
                          )
                        }
                      >
                        接受
                      </button>
                      <button
                        type="button"
                        aria-label="Decline"
                        onClick={() => props.onApprovalDecision(approval.requestId, "decline")}
                      >
                        拒绝
                      </button>
                      <button
                        type="button"
                        aria-label="Cancel"
                        onClick={() => props.onApprovalDecision(approval.requestId, "cancel")}
                      >
                        取消
                      </button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}

          <form className="chat-compose" onSubmit={handlePromptSubmit}>
            <label htmlFor="session-chat-input">Prompt</label>
            <textarea
              id="session-chat-input"
              aria-label="Prompt"
              value={props.prompt}
              onChange={(event) => props.onPromptChange(event.target.value)}
              placeholder="Send a message to the current thread."
            />
            <button type="submit" aria-label="Send prompt" disabled={!props.canStartTurn || props.isSubmitting}>
              Send
            </button>
          </form>
        </section>

        <aside className="workspace-control-stack">
          <section className="workspace-status-card">
            <h2>Turn Controls</h2>
            {props.activeTurnId ? <p>{`当前 Turn：${props.activeTurnId}`}</p> : <p>当前没有活动 Turn</p>}
            {props.activeTurnId ? (
              <>
                <form className="chat-compose" onSubmit={handleSteerSubmit}>
                  <label htmlFor="thread-steer-input">Steer prompt</label>
                  <textarea
                    id="thread-steer-input"
                    aria-label="Steer prompt"
                    value={props.steerPrompt}
                    onChange={(event) => props.onSteerPromptChange(event.target.value)}
                    placeholder="Guide the active turn."
                  />
                  <button type="submit" aria-label="Send steer" disabled={!props.canSteerTurn}>
                    Steer turn
                  </button>
                </form>
                <div className="approval-actions">
                  <button
                    type="button"
                    aria-label="Interrupt turn"
                    disabled={!props.canInterruptTurn}
                    onClick={props.onInterrupt}
                  >
                    Interrupt turn
                  </button>
                </div>
              </>
            ) : (
              <p>Start a new turn to unlock steer and interrupt controls.</p>
            )}
          </section>
        </aside>
      </div>
    </section>
  );
}
