package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func TestServerServesEmptyControlPlaneViews(t *testing.T) {
	handler := NewServer(registry.NewStore(), runtimeindex.NewStore())

	for _, path := range []string{"/health", "/machines", "/threads", "/environment/skills", "/environment/mcps", "/environment/plugins"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, rec.Code)
		}
	}
}
