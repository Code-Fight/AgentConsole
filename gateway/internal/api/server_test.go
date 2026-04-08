package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func TestServerServesEmptyControlPlaneViews(t *testing.T) {
	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), http.NotFoundHandler())

	for _, path := range []string{"/health", "/machines", "/threads", "/environment/skills", "/environment/mcps", "/environment/plugins"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, rec.Code)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("%s content-type = %q", path, got)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s invalid json: %v", path, err)
		}

		if path == "/health" {
			ok, exists := body["ok"].(bool)
			if !exists || !ok {
				t.Fatalf("%s unexpected body: %v", path, body)
			}
			continue
		}

		items, exists := body["items"].([]any)
		if !exists {
			t.Fatalf("%s items is not a json array: %T (%v)", path, body["items"], body["items"])
		}
		if len(items) != 0 {
			t.Fatalf("%s expected empty items, got: %v", path, items)
		}
	}
}

func TestServerMountsClientWebsocketRoute(t *testing.T) {
	called := false
	wsHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	handler := NewServer(registry.NewStore(), runtimeindex.NewStore(), wsHandler)
	req := httptest.NewRequest(http.MethodGet, "/ws/client", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("/ws/client returned %d", rec.Code)
	}
	if !called {
		t.Fatal("/ws/client handler was not invoked")
	}
}
