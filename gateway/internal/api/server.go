package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

type createThreadRequest struct {
	MachineID string `json:"machineId"`
	Title     string `json:"title"`
}

type startTurnRequest struct {
	Input string `json:"input"`
}

type approvalRespondRequest struct {
	Decision string `json:"decision"`
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

func NewServer(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	mux := http.NewServeMux()

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

	mux.HandleFunc("GET /threads", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Threads()})
	})

	mux.HandleFunc("GET /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
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

		completed, err := sender.SendCommand(r.Context(), machineID, "thread.read", protocol.ThreadReadCommandPayload{
			ThreadID: threadID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadReadCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.read result", http.StatusBadGateway)
			return
		}
		if result.Thread.MachineID == "" {
			result.Thread.MachineID = machineID
		}
		if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID)
		}

		writeJSON(w, http.StatusOK, map[string]any{"thread": result.Thread})
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

		completed, err := sender.SendCommand(r.Context(), machineID, "approval.respond", protocol.ApprovalRespondCommandPayload{
			RequestID: requestID,
			Decision:  req.Decision,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
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

	return mux
}
