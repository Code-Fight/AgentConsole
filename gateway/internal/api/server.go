package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

type CommandSender interface {
	SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error)
}

type approvalRequestResolver interface {
	ResolveApprovalMachine(requestID string) (string, bool)
}

type approvalRequestCleaner interface {
	ClearApprovalRequest(requestID string)
}

type threadDetailResponse struct {
	Thread           domain.Thread                      `json:"thread"`
	ActiveTurnID     string                             `json:"activeTurnId,omitempty"`
	PendingApprovals []protocol.ApprovalRequiredPayload `json:"pendingApprovals"`
}

type threadDeleteResponse struct {
	ThreadID string `json:"threadId"`
	Deleted  bool   `json:"deleted"`
	Archived bool   `json:"archived"`
}

type createThreadRequest struct {
	MachineID string `json:"machineId"`
	Title     string `json:"title"`
}

type startTurnRequest struct {
	Input string `json:"input"`
}

type approvalRespondRequest struct {
	Decision string         `json:"decision"`
	Answers  map[string]any `json:"answers"`
}

type environmentMutationRequest struct {
	MachineID string `json:"machineId"`
}

type environmentMCPUpsertRequest struct {
	MachineID  string         `json:"machineId"`
	ResourceID string         `json:"resourceId"`
	Config     map[string]any `json:"config"`
}

type activeTurnReader interface {
	ActiveTurnID(threadID string) (string, bool)
}

type threadUpdateEmitter interface {
	EmitThreadUpdated(payload protocol.ThreadUpdatedPayload, timestamp string)
}

func resolveThreadMachineID(router *routing.Router, threadID string) (string, bool) {
	if router == nil {
		return "", false
	}
	return router.ResolveThread(threadID)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func findThread(idx *runtimeindex.Store, threadID string) (domain.Thread, bool) {
	if idx == nil || strings.TrimSpace(threadID) == "" {
		return domain.Thread{}, false
	}
	for _, thread := range idx.Threads() {
		if thread.ThreadID == threadID {
			return thread, true
		}
	}
	return domain.Thread{}, false
}

func findEnvironmentResource(idx *runtimeindex.Store, kind domain.EnvironmentKind, machineID string, resourceID string) (domain.EnvironmentResource, bool) {
	if idx == nil || strings.TrimSpace(machineID) == "" || strings.TrimSpace(resourceID) == "" {
		return domain.EnvironmentResource{}, false
	}
	for _, resource := range idx.Environment(kind) {
		if resource.ResourceID == resourceID && resource.MachineID == machineID {
			return resource, true
		}
	}
	return domain.EnvironmentResource{}, false
}

func resolveEnvironmentMutationMachineID(r *http.Request) (string, error) {
	if r == nil {
		return "", nil
	}

	machineID := strings.TrimSpace(r.URL.Query().Get("machineId"))
	if machineID != "" {
		return machineID, nil
	}

	if r.Body == nil {
		return "", nil
	}

	var req environmentMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(req.MachineID), nil
}

func decodeEnvironmentMCPUpsertRequest(r *http.Request) (environmentMCPUpsertRequest, error) {
	if r == nil || r.Body == nil {
		return environmentMCPUpsertRequest{}, nil
	}

	var req environmentMCPUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return environmentMCPUpsertRequest{}, nil
		}
		return environmentMCPUpsertRequest{}, err
	}
	return req, nil
}

func stringDetail(resource domain.EnvironmentResource, key string) string {
	if resource.Details == nil || strings.TrimSpace(key) == "" {
		return ""
	}
	value, ok := resource.Details[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}

func machineIsOnline(reg *registry.Store, machineID string) bool {
	if reg == nil || strings.TrimSpace(machineID) == "" {
		return false
	}
	machine, ok := reg.Get(machineID)
	return ok && machine.Status == domain.MachineStatusOnline
}

func resolveActiveTurnID(sender CommandSender, threadID string) string {
	if sender == nil || strings.TrimSpace(threadID) == "" {
		return ""
	}

	reader, ok := sender.(activeTurnReader)
	if !ok {
		return ""
	}

	activeTurnID, ok := reader.ActiveTurnID(threadID)
	if !ok {
		return ""
	}
	return activeTurnID
}

func NewServer(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	mux := http.NewServeMux()
	var deletedThreadsMu sync.RWMutex
	deletedThreads := map[string]struct{}{}

	isDeleted := func(threadID string) bool {
		deletedThreadsMu.RLock()
		defer deletedThreadsMu.RUnlock()
		_, ok := deletedThreads[threadID]
		return ok
	}
	markDeleted := func(threadID string) {
		if strings.TrimSpace(threadID) == "" {
			return
		}
		deletedThreadsMu.Lock()
		deletedThreads[threadID] = struct{}{}
		deletedThreadsMu.Unlock()
	}
	clearDeleted := func(threadID string) {
		if strings.TrimSpace(threadID) == "" {
			return
		}
		deletedThreadsMu.Lock()
		delete(deletedThreads, threadID)
		deletedThreadsMu.Unlock()
	}

	if clientWS != nil {
		mux.Handle("/ws/client", clientWS)
	}
	if consoleWS != nil {
		mux.Handle("/ws", consoleWS)
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("GET /machines", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": reg.List()})
	})

	mux.HandleFunc("GET /machines/{machineId}", func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		machine, ok := reg.Get(machineID)
		if !ok {
			http.Error(w, "machine not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"machine": machine})
	})

	mux.HandleFunc("POST /machines/{machineId}/runtime/start", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), machineID, "runtime.start", protocol.RuntimeStartCommandPayload{}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"machineId": machineID})
	})

	mux.HandleFunc("POST /machines/{machineId}/runtime/stop", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), machineID, "runtime.stop", protocol.RuntimeStopCommandPayload{}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"machineId": machineID})
	})

	mux.HandleFunc("GET /threads", func(w http.ResponseWriter, _ *http.Request) {
		threads := make([]domain.Thread, 0)
		for _, thread := range idx.Threads() {
			if isDeleted(thread.ThreadID) {
				continue
			}
			threads = append(threads, thread)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": threads})
	})

	mux.HandleFunc("GET /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}
		if isDeleted(threadID) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		pendingApprovals := []protocol.ApprovalRequiredPayload{}
		if reg != nil {
			pendingApprovals = reg.PendingApprovalsForThread(threadID)
		}
		activeTurnID := resolveActiveTurnID(sender, threadID)

		if sender != nil {
			if machineID, ok := resolveThreadMachineID(router, threadID); ok {
				liveReadRequired := machineIsOnline(reg, machineID)
				completed, err := sender.SendCommand(r.Context(), machineID, "thread.read", protocol.ThreadReadCommandPayload{
					ThreadID: threadID,
				})
				if err == nil {
					var result protocol.ThreadReadCommandResult
					if err := json.Unmarshal(completed.Result, &result); err == nil {
						if result.Thread.MachineID == "" {
							result.Thread.MachineID = machineID
						}
						if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
							router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID)
						}
						clearDeleted(result.Thread.ThreadID)
						writeJSON(w, http.StatusOK, threadDetailResponse{
							Thread:           result.Thread,
							ActiveTurnID:     activeTurnID,
							PendingApprovals: pendingApprovals,
						})
						return
					}
					if liveReadRequired {
						http.Error(w, "invalid thread.read result", http.StatusBadGateway)
						return
					}
				}
				if err != nil && liveReadRequired {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
			}
		}

		thread, ok := findThread(idx, threadID)
		if !ok {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, threadDetailResponse{
			Thread:           thread,
			ActiveTurnID:     activeTurnID,
			PendingApprovals: pendingApprovals,
		})
	})

	mux.HandleFunc("POST /threads", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		var req createThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), req.MachineID, "thread.create", protocol.ThreadCreateCommandPayload{
			Title: req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadCreateCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.create result", http.StatusBadGateway)
			return
		}
		if result.Thread.MachineID == "" {
			result.Thread.MachineID = req.MachineID
		}
		if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID)
		}
		clearDeleted(result.Thread.ThreadID)

		writeJSON(w, http.StatusCreated, map[string]any{"thread": result.Thread})
	})

	mux.HandleFunc("POST /threads/{threadId}/resume", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolveThreadMachineID(router, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "thread.resume", protocol.ThreadResumeCommandPayload{
			ThreadID: threadID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadResumeCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.resume result", http.StatusBadGateway)
			return
		}
		if result.Thread.MachineID == "" {
			result.Thread.MachineID = machineID
		}
		if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID)
		}
		clearDeleted(result.Thread.ThreadID)

		writeJSON(w, http.StatusOK, map[string]any{"thread": result.Thread})
	})

	mux.HandleFunc("POST /threads/{threadId}/archive", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolveThreadMachineID(router, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "thread.archive", protocol.ThreadArchiveCommandPayload{
			ThreadID: threadID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadArchiveCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.archive result", http.StatusBadGateway)
			return
		}
		if result.ThreadID == "" {
			result.ThreadID = threadID
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"threadId": result.ThreadID})
	})

	mux.HandleFunc("DELETE /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}
		if isDeleted(threadID) {
			writeJSON(w, http.StatusOK, threadDeleteResponse{
				ThreadID: threadID,
				Deleted:  true,
				Archived: false,
			})
			return
		}
		thread, ok := findThread(idx, threadID)
		if !ok {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		archived := false
		if sender != nil {
			if machineID, ok := resolveThreadMachineID(router, threadID); ok {
				completed, err := sender.SendCommand(r.Context(), machineID, "thread.archive", protocol.ThreadArchiveCommandPayload{
					ThreadID: threadID,
				})
				if err == nil {
					var result protocol.ThreadArchiveCommandResult
					if err := json.Unmarshal(completed.Result, &result); err == nil {
						archived = true
					}
				}
			}
		}

		markDeleted(threadID)
		if emitter, ok := sender.(threadUpdateEmitter); ok {
			machineID := thread.MachineID
			if machineID == "" {
				machineID, _ = resolveThreadMachineID(router, threadID)
			}
			emitter.EmitThreadUpdated(protocol.ThreadUpdatedPayload{
				MachineID: machineID,
				ThreadID:  threadID,
			}, time.Now().UTC().Format(time.RFC3339))
		}
		writeJSON(w, http.StatusOK, threadDeleteResponse{
			ThreadID: threadID,
			Deleted:  true,
			Archived: archived,
		})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolveThreadMachineID(router, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		var req startTurnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "turn.start", protocol.TurnStartCommandPayload{
			ThreadID: threadID,
			Input:    req.Input,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnStartCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.start result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"turn": result})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns/{turnId}/steer", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		turnID := r.PathValue("turnId")
		if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
			http.Error(w, "threadId and turnId are required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolveThreadMachineID(router, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		var req startTurnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "turn.steer", protocol.TurnSteerCommandPayload{
			ThreadID: threadID,
			TurnID:   turnID,
			Input:    req.Input,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnSteerCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.steer result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"turn": result})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns/{turnId}/interrupt", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		turnID := r.PathValue("turnId")
		if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
			http.Error(w, "threadId and turnId are required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolveThreadMachineID(router, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "turn.interrupt", protocol.TurnInterruptCommandPayload{
			ThreadID: threadID,
			TurnID:   turnID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnInterruptCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.interrupt result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"turn": result.Turn})
	})

	mux.HandleFunc("POST /approvals/{requestId}/respond", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		resolver, ok := sender.(approvalRequestResolver)
		if !ok {
			http.Error(w, "approval resolver unavailable", http.StatusServiceUnavailable)
			return
		}

		requestID := r.PathValue("requestId")
		if strings.TrimSpace(requestID) == "" {
			http.Error(w, "requestId is required", http.StatusBadRequest)
			return
		}

		var req approvalRespondRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Decision) == "" {
			http.Error(w, "decision is required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolver.ResolveApprovalMachine(requestID)
		if !ok {
			http.Error(w, "approval route not found", http.StatusNotFound)
			return
		}

		storedApproval := protocol.ApprovalRequiredPayload{}
		if reg != nil {
			if payload, ok := reg.PendingApproval(requestID); ok {
				storedApproval = payload
			}
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "approval.respond", protocol.ApprovalRespondCommandPayload{
			RequestID: requestID,
			ThreadID:  storedApproval.ThreadID,
			TurnID:    storedApproval.TurnID,
			Decision:  req.Decision,
			Answers:   req.Answers,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if reg != nil {
			reg.RemovePendingApproval(requestID)
		}
		if cleaner, ok := sender.(approvalRequestCleaner); ok {
			cleaner.ClearApprovalRequest(requestID)
		}

		var result protocol.ApprovalRespondCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid approval.respond result", http.StatusBadGateway)
			return
		}
		if result.RequestID == "" {
			result.RequestID = requestID
		}
		if result.Decision == "" {
			result.Decision = req.Decision
		}

		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /environment/skills", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindSkill)})
	})

	mux.HandleFunc("GET /environment/mcps", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindMCP)})
	})

	mux.HandleFunc("GET /environment/plugins", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindPlugin)})
	})

	mux.HandleFunc("POST /environment/skills/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		skillID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindSkill, machineID, skillID)
		if !ok {
			http.Error(w, "skill not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.skill.enable", protocol.EnvironmentSkillSetEnabledCommandPayload{
			SkillID: skillID,
			Enabled: true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": skillID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/skills/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		skillID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindSkill, machineID, skillID)
		if !ok {
			http.Error(w, "skill not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.skill.disable", protocol.EnvironmentSkillSetEnabledCommandPayload{
			SkillID: skillID,
			Enabled: false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": skillID, "enabled": false})
	})

	mux.HandleFunc("DELETE /environment/plugins/{id}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, machineID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.uninstall", protocol.EnvironmentPluginUninstallCommandPayload{
			PluginID: pluginID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID})
	})

	mux.HandleFunc("POST /environment/mcps", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMCPUpsertRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.ResourceID) == "" {
			http.Error(w, "resourceId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), req.MachineID, "environment.mcp.upsert", protocol.EnvironmentMCPUpsertCommandPayload{
			ServerID: req.ResourceID,
			Config:   req.Config,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": req.ResourceID})
	})

	mux.HandleFunc("POST /environment/mcps/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, machineID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.enable", protocol.EnvironmentMCPSetEnabledCommandPayload{
			ServerID: serverID,
			Enabled:  true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/mcps/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, machineID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.disable", protocol.EnvironmentMCPSetEnabledCommandPayload{
			ServerID: serverID,
			Enabled:  false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID, "enabled": false})
	})

	mux.HandleFunc("DELETE /environment/mcps/{id}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, machineID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.remove", protocol.EnvironmentMCPRemoveCommandPayload{
			ServerID: serverID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/install", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, machineID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}
		marketplacePath := stringDetail(resource, "marketplacePath")
		pluginName := stringDetail(resource, "pluginName")
		if marketplacePath == "" || pluginName == "" {
			http.Error(w, "plugin install details unavailable", http.StatusConflict)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.install", protocol.EnvironmentPluginInstallCommandPayload{
			PluginID:        pluginID,
			MarketplacePath: marketplacePath,
			PluginName:      pluginName,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, machineID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.enable", protocol.EnvironmentPluginSetEnabledCommandPayload{
			PluginID: pluginID,
			Enabled:  true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID, err := resolveEnvironmentMutationMachineID(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, machineID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.disable", protocol.EnvironmentPluginSetEnabledCommandPayload{
			PluginID: pluginID,
			Enabled:  false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID, "enabled": false})
	})

	return mux
}
