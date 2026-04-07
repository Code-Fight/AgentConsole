package api

import (
	"encoding/json"
	"net/http"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func NewServer(reg *registry.Store, idx *runtimeindex.Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	mux.HandleFunc("GET /machines", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": reg.List()})
	})

	mux.HandleFunc("GET /threads", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": idx.Threads()})
	})

	mux.HandleFunc("GET /environment/skills", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": idx.Environment(domain.EnvironmentKindSkill)})
	})

	mux.HandleFunc("GET /environment/mcps", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": idx.Environment(domain.EnvironmentKindMCP)})
	})

	mux.HandleFunc("GET /environment/plugins", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": idx.Environment(domain.EnvironmentKindPlugin)})
	})

	return mux
}
