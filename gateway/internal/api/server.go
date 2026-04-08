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

type createThreadRequest struct {
	MachineID string `json:"machineId"`
	Title     string `json:"title"`
}

type startTurnRequest struct {
	Input string `json:"input"`
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

		if router == nil {
			http.Error(w, "router unavailable", http.StatusServiceUnavailable)
			return
		}
		machineID, ok := router.ResolveThread(threadID)
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
