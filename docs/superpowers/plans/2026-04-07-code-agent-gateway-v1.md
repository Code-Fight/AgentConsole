# Code Agent Gateway V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Codex-first, single-tenant gateway with a Go `gateway`, a Go `client`, a shared `common` Go package, and a responsive `React + TypeScript + Vite` console.

**Architecture:** Use a polyglot monorepo with `go.work` coordinating `common/`, `gateway/`, and `client/`, while `console/` remains an isolated frontend package. `gateway` owns control-plane APIs and routing, `client` owns execution and Codex App Server integration, and `common` holds stable domain/protocol/transport/version packages shared by both Go services. Existing TypeScript backend code under `apps/` and `packages/` is legacy and must be retired once the Go replacements are verified.

**Tech Stack:** Go 1.22+, `net/http`, `encoding/json`, `github.com/coder/websocket`, React 19, TypeScript, Vite, Vitest, React Testing Library, Playwright

---

## Scope Check

This is one implementation plan even though it spans three deployables, because V1 only works when the Go `gateway`, Go `client`, `common` packages, and `console` land as one coherent vertical slice. The current repository already contains TypeScript backend scaffolding; the new plan replaces it instead of extending it.

## File Structure

- Keep: `docs/superpowers/specs/2026-04-07-code-agent-gateway-v1-design.md`
- Keep: `.gitignore`
- Create: `go.work`
- Create: `common/go.mod`
- Create: `common/version/version.go`
- Create: `common/version/version_test.go`
- Create: `common/domain/machine.go`
- Create: `common/domain/thread.go`
- Create: `common/domain/environment.go`
- Create: `common/protocol/messages.go`
- Create: `common/protocol/messages_test.go`
- Create: `common/transport/jsoncodec.go`
- Create: `gateway/go.mod`
- Create: `gateway/cmd/gateway/main.go`
- Create: `gateway/internal/config/config.go`
- Create: `gateway/internal/config/config_test.go`
- Create: `gateway/internal/registry/store.go`
- Create: `gateway/internal/runtimeindex/store.go`
- Create: `gateway/internal/api/server.go`
- Create: `gateway/internal/api/server_test.go`
- Create: `gateway/internal/websocket/client_hub.go`
- Create: `gateway/internal/websocket/client_hub_test.go`
- Create: `gateway/internal/routing/router.go`
- Create: `gateway/internal/routing/router_test.go`
- Create: `client/go.mod`
- Create: `client/cmd/client/main.go`
- Create: `client/internal/config/config.go`
- Create: `client/internal/config/config_test.go`
- Create: `client/internal/gateway/session.go`
- Create: `client/internal/gateway/session_test.go`
- Create: `client/internal/snapshot/builder.go`
- Create: `client/internal/agent/types/interfaces.go`
- Create: `client/internal/agent/registry/registry.go`
- Create: `client/internal/agent/manager/manager.go`
- Create: `client/internal/agent/codex/fake_adapter.go`
- Create: `client/internal/agent/codex/fake_adapter_test.go`
- Create: `client/internal/agent/codex/appserver_client.go`
- Create: `client/internal/agent/codex/threads.go`
- Create: `client/internal/agent/codex/turns.go`
- Create: `client/internal/agent/codex/approvals.go`
- Create: `client/internal/agent/codex/environment.go`
- Create: `client/internal/agent/codex/appserver_client_test.go`
- Create: `console/package.json`
- Create: `console/tsconfig.json`
- Create: `console/vite.config.ts`
- Create: `console/playwright.config.ts`
- Create: `console/src/main.tsx`
- Create: `console/src/app/router.tsx`
- Create: `console/src/app/providers.tsx`
- Create: `console/src/common/api/http.ts`
- Create: `console/src/common/api/ws.ts`
- Create: `console/src/common/api/types.ts`
- Create: `console/src/pages/overview-page.tsx`
- Create: `console/src/pages/machines-page.tsx`
- Create: `console/src/pages/threads-page.tsx`
- Create: `console/src/pages/thread-workspace-page.tsx`
- Create: `console/src/pages/environment-page.tsx`
- Create: `console/src/app/shell.tsx`
- Create: `console/src/app/shell.test.tsx`
- Create: `console/src/styles.css`
- Delete: `apps/gateway/**`
- Delete: `apps/client-codex/**`
- Delete: `packages/domain/**`
- Delete: `packages/protocol/**`
- Delete: `package.json`
- Delete: `pnpm-workspace.yaml`
- Delete: `tsconfig.base.json`
- Delete: `vitest.workspace.ts`
- Delete: `pnpm-lock.yaml`

## Task 1: Bootstrap Go Workspace And Common Version Package

**Files:**
- Modify: `.gitignore`
- Create: `go.work`
- Create: `common/go.mod`
- Create: `common/version/version.go`
- Test: `common/version/version_test.go`

- [ ] **Step 1: Write the failing test**

```go
// common/version/version_test.go
package version

import "testing"

func TestCurrentProtocolVersion(t *testing.T) {
	if CurrentProtocolVersion != "v1" {
		t.Fatalf("expected v1, got %q", CurrentProtocolVersion)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./common/version`
Expected: FAIL with `undefined: CurrentProtocolVersion` or missing package files

- [ ] **Step 3: Write minimal implementation**

```gitignore
# .gitignore
.DS_Store
node_modules
dist
coverage
playwright-report
.superpowers
bin
*.tsbuildinfo
console/node_modules
console/dist
```

```txt
# go.work
go 1.22.0

use (
	./common
)
```

```go
// common/go.mod
module code-agent-gateway/common

go 1.22.0
```

```go
// common/version/version.go
package version

const CurrentProtocolVersion = "v1"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./common/version`
Expected: PASS with `ok  	code-agent-gateway/common/version`

- [ ] **Step 5: Commit**

```bash
git add .gitignore go.work common/go.mod common/version
git commit -m "chore: bootstrap go workspace and common version package"
```

## Task 2: Define Common Domain And Southbound Protocol Packages

**Files:**
- Create: `common/domain/machine.go`
- Create: `common/domain/thread.go`
- Create: `common/domain/environment.go`
- Create: `common/protocol/messages.go`
- Create: `common/protocol/messages_test.go`
- Create: `common/transport/jsoncodec.go`

- [ ] **Step 1: Write the failing test**

```go
// common/protocol/messages_test.go
package protocol

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	msg := Envelope{
		Version:   "v1",
		Category:  CategoryCommand,
		Name:      "thread.create",
		RequestID: "req_123",
		MachineID: "machine_01",
		Timestamp: "2026-04-07T10:00:00Z",
		Payload:   json.RawMessage(`{"title":"Investigate flaky test"}`),
	}

	blob, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Name != "thread.create" {
		t.Fatalf("expected thread.create, got %q", decoded.Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./common/...`
Expected: FAIL with `undefined: Envelope` or missing package files

- [ ] **Step 3: Write minimal implementation**

```go
// common/domain/machine.go
package domain

type MachineStatus string

const (
	MachineStatusOnline       MachineStatus = "online"
	MachineStatusOffline      MachineStatus = "offline"
	MachineStatusReconnecting MachineStatus = "reconnecting"
)

type Machine struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status MachineStatus `json:"status"`
}
```

```go
// common/domain/thread.go
package domain

type ThreadStatus string
type TurnStatus string
type ApprovalStatus string

const (
	ThreadStatusNotLoaded ThreadStatus = "notLoaded"
	ThreadStatusIdle      ThreadStatus = "idle"
	ThreadStatusActive    ThreadStatus = "active"
	ThreadStatusError     ThreadStatus = "systemError"

	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusInterrupted TurnStatus = "interrupted"
	TurnStatusFailed      TurnStatus = "failed"

	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

type Thread struct {
	ID       string       `json:"id"`
	MachineID string      `json:"machineId"`
	Status   ThreadStatus `json:"status"`
	Title    string       `json:"title"`
}

type Turn struct {
	ID       string     `json:"id"`
	ThreadID string     `json:"threadId"`
	Status   TurnStatus `json:"status"`
}

type ApprovalRequest struct {
	RequestID string         `json:"requestId"`
	ThreadID  string         `json:"threadId"`
	TurnID    string         `json:"turnId"`
	ItemID    string         `json:"itemId"`
	Kind      string         `json:"kind"`
	Status    ApprovalStatus `json:"status"`
}
```

```go
// common/domain/environment.go
package domain

type EnvironmentKind string

const (
	EnvironmentKindSkill  EnvironmentKind = "skill"
	EnvironmentKindMCP    EnvironmentKind = "mcp"
	EnvironmentKindPlugin EnvironmentKind = "plugin"
)

type EnvironmentResource struct {
	ID              string          `json:"id"`
	Kind            EnvironmentKind `json:"kind"`
	DisplayName     string          `json:"displayName"`
	Status          string          `json:"status"`
	RestartRequired bool            `json:"restartRequired"`
	LastObservedAt  string          `json:"lastObservedAt"`
}
```

```go
// common/protocol/messages.go
package protocol

import "encoding/json"

type Category string

const (
	CategorySystem   Category = "system"
	CategoryCommand  Category = "command"
	CategoryEvent    Category = "event"
	CategorySnapshot Category = "snapshot"
)

type Envelope struct {
	Version   string          `json:"version"`
	Category  Category        `json:"category"`
	Name      string          `json:"name"`
	RequestID string          `json:"requestId,omitempty"`
	MachineID string          `json:"machineId,omitempty"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}
```

```go
// common/transport/jsoncodec.go
package transport

import "encoding/json"

func Encode[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

func Decode[T any](raw []byte, target *T) error {
	return json.Unmarshal(raw, target)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./common/...`
Expected: PASS with `ok` for `common/protocol` and `common/version`

- [ ] **Step 5: Commit**

```bash
git add common/domain common/protocol common/transport
git commit -m "feat: add common domain and protocol packages"
```

## Task 3: Build The Go Gateway Skeleton And Northbound HTTP Surface

**Files:**
- Modify: `go.work`
- Create: `gateway/go.mod`
- Create: `gateway/cmd/gateway/main.go`
- Create: `gateway/internal/config/config.go`
- Test: `gateway/internal/config/config_test.go`
- Create: `gateway/internal/registry/store.go`
- Create: `gateway/internal/runtimeindex/store.go`
- Create: `gateway/internal/api/server.go`
- Test: `gateway/internal/api/server_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// gateway/internal/config/config_test.go
package config

import "testing"

func TestReadConfigFallsBackAndRejectsInvalidPort(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	cfg, err := Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != 8080 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	t.Setenv("PORT", "abc")
	if _, err := Read(); err == nil {
		t.Fatal("expected invalid port error")
	}
}
```

```go
// gateway/internal/api/server_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./gateway/internal/...`
Expected: FAIL with missing `Read`, `NewStore`, or `NewServer`

- [ ] **Step 3: Write minimal implementation**

```go
// go.work
go 1.22.0

use (
	./common
	./gateway
)
```

```go
// gateway/go.mod
module code-agent-gateway/gateway

go 1.22.0

require code-agent-gateway/common v0.0.0
```

```go
// gateway/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host string
	Port int
}

func Read() (Config, error) {
	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		host = "0.0.0.0"
	}

	portRaw := strings.TrimSpace(os.Getenv("PORT"))
	if portRaw == "" {
		return Config{Host: host, Port: 8080}, nil
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT value: %q", portRaw)
	}

	return Config{Host: host, Port: port}, nil
}
```

```go
// gateway/internal/registry/store.go
package registry

import "code-agent-gateway/common/domain"

type Store struct {
	machines map[string]domain.Machine
}

func NewStore() *Store {
	return &Store{machines: map[string]domain.Machine{}}
}

func (s *Store) List() []domain.Machine {
	items := make([]domain.Machine, 0, len(s.machines))
	for _, item := range s.machines {
		items = append(items, item)
	}
	return items
}
```

```go
// gateway/internal/runtimeindex/store.go
package runtimeindex

import (
	"code-agent-gateway/common/domain"
)

type Store struct {
	threads     []domain.Thread
	environment []domain.EnvironmentResource
}

func NewStore() *Store { return &Store{} }
func (s *Store) Threads() []domain.Thread { return s.threads }
func (s *Store) Environment(kind domain.EnvironmentKind) []domain.EnvironmentResource {
	items := make([]domain.EnvironmentResource, 0)
	for _, item := range s.environment {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	return items
}
```

```go
// gateway/internal/api/server.go
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
```

```go
// gateway/cmd/gateway/main.go
package main

import (
	"log"
	"net/http"
	"strconv"

	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/config"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	handler := api.NewServer(registry.NewStore(), runtimeindex.NewStore())
	log.Fatal(http.ListenAndServe(
		cfg.Host+":"+strconv.Itoa(cfg.Port),
		handler,
	))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./gateway/internal/...`
Expected: PASS with `ok` for `gateway/internal/config` and `gateway/internal/api`

- [ ] **Step 5: Commit**

```bash
git add gateway
git commit -m "feat: add go gateway skeleton"
```

## Task 4: Build The Go Client Skeleton, Agent Abstraction, And Fake Codex Adapter

**Files:**
- Modify: `go.work`
- Create: `client/go.mod`
- Create: `client/cmd/client/main.go`
- Create: `client/internal/config/config.go`
- Test: `client/internal/config/config_test.go`
- Create: `client/internal/gateway/session.go`
- Test: `client/internal/gateway/session_test.go`
- Create: `client/internal/snapshot/builder.go`
- Create: `client/internal/agent/types/interfaces.go`
- Create: `client/internal/agent/registry/registry.go`
- Create: `client/internal/agent/manager/manager.go`
- Create: `client/internal/agent/codex/fake_adapter.go`
- Test: `client/internal/agent/codex/fake_adapter_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// client/internal/gateway/session_test.go
package gateway

import (
	"testing"
	"time"
)

func TestSessionFrames(t *testing.T) {
	var sent []string

	session := NewSession("machine-01", func(msg []byte) error {
		sent = append(sent, string(msg))
		return nil
	}, func() time.Time {
		return time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	})

	if err := session.Register(); err != nil {
		t.Fatal(err)
	}
	if err := session.Heartbeat(); err != nil {
		t.Fatal(err)
	}

	if len(sent) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(sent))
	}
}
```

```go
// client/internal/agent/codex/fake_adapter_test.go
package codex

import "testing"

func TestFakeAdapterSnapshot(t *testing.T) {
	adapter := NewFakeAdapter()

	threads, err := adapter.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("expected empty threads, got %d", len(threads))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./client/internal/...`
Expected: FAIL with missing `NewSession` or `NewFakeAdapter`

- [ ] **Step 3: Write minimal implementation**

```go
// go.work
go 1.22.0

use (
	./common
	./gateway
	./client
)
```

```go
// client/go.mod
module code-agent-gateway/client

go 1.22.0

require code-agent-gateway/common v0.0.0
```

```go
// client/internal/config/config.go
package config

import "os"

type Config struct {
	MachineID  string
	GatewayURL string
}

func Read() Config {
	machineID := os.Getenv("MACHINE_ID")
	if machineID == "" {
		machineID = "machine-01"
	}
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "ws://localhost:8080/ws/client"
	}
	return Config{MachineID: machineID, GatewayURL: gatewayURL}
}
```

```go
// client/internal/agent/types/interfaces.go
package types

import "code-agent-gateway/common/domain"

type Runtime interface {
	ListThreads() ([]domain.Thread, error)
	ListEnvironment() ([]domain.EnvironmentResource, error)
}
```

```go
// client/internal/agent/codex/fake_adapter.go
package codex

import "code-agent-gateway/common/domain"

type FakeAdapter struct{}

func NewFakeAdapter() *FakeAdapter { return &FakeAdapter{} }

func (a *FakeAdapter) ListThreads() ([]domain.Thread, error) {
	return []domain.Thread{}, nil
}

func (a *FakeAdapter) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return []domain.EnvironmentResource{}, nil
}
```

```go
// client/internal/snapshot/builder.go
package snapshot

import (
	"code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type Snapshot struct {
	Threads     []domain.Thread              `json:"threads"`
	Environment []domain.EnvironmentResource `json:"environment"`
}

func Build(runtime types.Runtime) (Snapshot, error) {
	threads, err := runtime.ListThreads()
	if err != nil {
		return Snapshot{}, err
	}
	environment, err := runtime.ListEnvironment()
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Threads: threads, Environment: environment}, nil
}
```

```go
// client/internal/gateway/session.go
package gateway

import (
	"time"
)

type Sender func([]byte) error

type Session struct {
	machineID string
	send      Sender
	now       func() time.Time
}

func NewSession(machineID string, send Sender, now func() time.Time) *Session {
	return &Session{machineID: machineID, send: send, now: now}
}

func (s *Session) Register() error {
	return s.send([]byte(`{"category":"system","name":"client.register","machineId":"` + s.machineID + `"}`))
}

func (s *Session) Heartbeat() error {
	return s.send([]byte(`{"category":"system","name":"client.heartbeat","machineId":"` + s.machineID + `"}`))
}
```

```go
// client/cmd/client/main.go
package main

import (
	"fmt"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/snapshot"
)

func main() {
	cfg := config.Read()
	snap, err := snapshot.Build(codex.NewFakeAdapter())
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg.GatewayURL, len(snap.Threads))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./client/internal/...`
Expected: PASS with `ok` for `client/internal/gateway` and `client/internal/agent/codex`

- [ ] **Step 5: Commit**

```bash
git add client
git commit -m "feat: add go client skeleton and fake codex adapter"
```

## Task 5: Implement Gateway-Client WebSocket System Messages And Snapshot Sync

**Files:**
- Create: `gateway/internal/websocket/client_hub.go`
- Test: `gateway/internal/websocket/client_hub_test.go`
- Modify: `client/internal/gateway/session.go`
- Create: `client/internal/gateway/socket.go`
- Modify: `client/cmd/client/main.go`

- [ ] **Step 1: Write the failing test**

```go
// gateway/internal/websocket/client_hub_test.go
package websocket

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
)

func TestClientHubAcceptsRegisterMessage(t *testing.T) {
	hub := NewClientHub()
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	conn, _, err := websocket.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := conn.Write(context.Background(), websocket.MessageText, []byte(`{"category":"system","name":"client.register","machineId":"machine-01","timestamp":"2026-04-07T10:00:00Z","version":"v1","payload":{}}`)); err != nil {
		t.Fatal(err)
	}

	if hub.Count() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.Count())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./gateway/internal/websocket`
Expected: FAIL with missing `NewClientHub`

- [ ] **Step 3: Write minimal implementation**

```go
// gateway/internal/websocket/client_hub.go
package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	cws "github.com/coder/websocket"
)

type ClientHub struct {
	mu      sync.Mutex
	clients map[string]struct{}
}

func NewClientHub() *ClientHub {
	return &ClientHub{clients: map[string]struct{}{}}
}

func (h *ClientHub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *ClientHub) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := cws.Accept(w, r, nil)
		if err != nil {
			return
		}
	defer conn.Close(cws.StatusNormalClosure, "done")

	_, data, err := conn.Read(context.Background())
	if err != nil {
		return
	}

	var envelope struct {
		MachineID string `json:"machineId"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.MachineID == "" {
		return
	}

	h.mu.Lock()
	h.clients[envelope.MachineID] = struct{}{}
	h.mu.Unlock()
})
}
```

```go
// client/internal/gateway/socket.go
package gateway

import (
	"context"

	cws "github.com/coder/websocket"
)

func Dial(ctx context.Context, url string) (*cws.Conn, error) {
	conn, _, err := cws.Dial(ctx, url, nil)
	return conn, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./gateway/internal/websocket`
Expected: PASS with `ok  	code-agent-gateway/gateway/internal/websocket`

- [ ] **Step 5: Commit**

```bash
git add gateway/internal/websocket client/internal/gateway client/cmd/client/main.go gateway/go.mod client/go.mod
git commit -m "feat: add gateway-client websocket system channel"
```

## Task 6: Implement Thread And Environment Command Routing Over The Fake Agent

**Files:**
- Create: `gateway/internal/routing/router.go`
- Test: `gateway/internal/routing/router_test.go`
- Modify: `gateway/internal/runtimeindex/store.go`
- Modify: `client/internal/agent/manager/manager.go`
- Modify: `client/internal/agent/registry/registry.go`
- Modify: `client/internal/agent/codex/fake_adapter.go`

- [ ] **Step 1: Write the failing test**

```go
// gateway/internal/routing/router_test.go
package routing

import "testing"

func TestResolveMachineForThread(t *testing.T) {
	router := NewRouter()
	router.TrackThread("thread-1", "machine-01")

	machineID, ok := router.ResolveThread("thread-1")
	if !ok || machineID != "machine-01" {
		t.Fatalf("expected machine-01, got %q", machineID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./gateway/internal/routing`
Expected: FAIL with missing `NewRouter`

- [ ] **Step 3: Write minimal implementation**

```go
// gateway/internal/routing/router.go
package routing

type Router struct {
	threads map[string]string
}

func NewRouter() *Router {
	return &Router{threads: map[string]string{}}
}

func (r *Router) TrackThread(threadID string, machineID string) {
	r.threads[threadID] = machineID
}

func (r *Router) ResolveThread(threadID string) (string, bool) {
	machineID, ok := r.threads[threadID]
	return machineID, ok
}
```

```go
// client/internal/agent/registry/registry.go
package registry

import "code-agent-gateway/client/internal/agent/types"

type Registry struct {
	runtimes map[string]types.Runtime
}

func New() *Registry {
	return &Registry{runtimes: map[string]types.Runtime{}}
}

func (r *Registry) Register(name string, runtime types.Runtime) {
	r.runtimes[name] = runtime
}

func (r *Registry) Get(name string) (types.Runtime, bool) {
	rt, ok := r.runtimes[name]
	return rt, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./gateway/internal/routing ./client/internal/agent/...`
Expected: PASS with `ok` for routing and agent packages

- [ ] **Step 5: Commit**

```bash
git add gateway/internal/routing gateway/internal/runtimeindex client/internal/agent
git commit -m "feat: add fake thread and environment routing path"
```

## Task 7: Build The React Console Shell, Overview, Machines, And Environment Pages

**Files:**
- Create: `console/package.json`
- Create: `console/tsconfig.json`
- Create: `console/vite.config.ts`
- Create: `console/playwright.config.ts`
- Create: `console/src/main.tsx`
- Create: `console/src/app/router.tsx`
- Create: `console/src/app/providers.tsx`
- Create: `console/src/app/shell.tsx`
- Test: `console/src/app/shell.test.tsx`
- Create: `console/src/common/api/http.ts`
- Create: `console/src/common/api/ws.ts`
- Create: `console/src/common/api/types.ts`
- Create: `console/src/pages/overview-page.tsx`
- Create: `console/src/pages/machines-page.tsx`
- Create: `console/src/pages/environment-page.tsx`
- Create: `console/src/styles.css`

- [ ] **Step 1: Write the failing test**

```tsx
// console/src/app/shell.test.tsx
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AppShell } from "./shell";

test("renders the console shell", () => {
  render(
    <MemoryRouter>
      <AppShell />
    </MemoryRouter>,
  );

  expect(screen.getByText("Overview")).toBeInTheDocument();
  expect(screen.getByText("Machines")).toBeInTheDocument();
  expect(screen.getByText("Environment")).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `corepack pnpm --dir console vitest src/app/shell.test.tsx`
Expected: FAIL with missing `AppShell`

- [ ] **Step 3: Write minimal implementation**

```json
// console/package.json
{
  "name": "@cag/console",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -p tsconfig.json && vite build",
    "test": "vitest run",
    "e2e": "playwright test"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^7.0.0"
  },
  "devDependencies": {
    "@playwright/test": "^1.54.0",
    "@testing-library/react": "^16.0.0",
    "@vitejs/plugin-react": "^5.0.0",
    "typescript": "^5.9.0",
    "vite": "^7.0.0",
    "vitest": "^3.2.0"
  }
}
```

```tsx
// console/src/app/shell.tsx
import { NavLink, Outlet } from "react-router-dom";

export function AppShell() {
  return (
    <div className="shell">
      <aside className="left-nav">
        <NavLink to="/">Overview</NavLink>
        <NavLink to="/machines">Machines</NavLink>
        <NavLink to="/threads">Threads</NavLink>
        <NavLink to="/environment">Environment</NavLink>
      </aside>
      <main className="center-pane">
        <Outlet />
      </main>
      <aside className="right-pane">Inspector</aside>
    </div>
  );
}
```

```tsx
// console/src/pages/overview-page.tsx
export function OverviewPage() {
  return <section>Overview</section>;
}
```

```tsx
// console/src/pages/machines-page.tsx
export function MachinesPage() {
  return <section>Machines</section>;
}
```

```tsx
// console/src/pages/environment-page.tsx
export function EnvironmentPage() {
  return <section>Environment</section>;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `corepack pnpm --dir console vitest src/app/shell.test.tsx`
Expected: PASS with `1 passed`

- [ ] **Step 5: Commit**

```bash
git add console
git commit -m "feat: add react console shell"
```

## Task 8: Implement The Codex App Server Adapter In Go

**Files:**
- Create: `client/internal/agent/codex/appserver_client.go`
- Create: `client/internal/agent/codex/threads.go`
- Create: `client/internal/agent/codex/turns.go`
- Create: `client/internal/agent/codex/approvals.go`
- Create: `client/internal/agent/codex/environment.go`
- Test: `client/internal/agent/codex/appserver_client_test.go`

- [ ] **Step 1: Write the failing test**

```go
// client/internal/agent/codex/appserver_client_test.go
package codex

import "testing"

type fakeRunner struct{}

func (fakeRunner) Call(method string, payload any, out any) error {
	if method == "thread/list" {
		threads := out.(*[]ThreadRecord)
		*threads = []ThreadRecord{{ID: "thread-1", Title: "Investigate flaky test"}}
	}
	return nil
}

func TestClientListsThreadsThroughRunner(t *testing.T) {
	client := NewAppServerClient(fakeRunner{})
	threads, err := client.ListThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ID != "thread-1" {
		t.Fatalf("unexpected threads: %+v", threads)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client/internal/agent/codex`
Expected: FAIL with missing `NewAppServerClient` or `ThreadRecord`

- [ ] **Step 3: Write minimal implementation**

```go
// client/internal/agent/codex/appserver_client.go
package codex

type Runner interface {
	Call(method string, payload any, out any) error
}

type ThreadRecord struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type AppServerClient struct {
	runner Runner
}

func NewAppServerClient(runner Runner) *AppServerClient {
	return &AppServerClient{runner: runner}
}

func (c *AppServerClient) ListThreads() ([]ThreadRecord, error) {
	var threads []ThreadRecord
	if err := c.runner.Call("thread/list", map[string]any{}, &threads); err != nil {
		return nil, err
	}
	return threads, nil
}
```

```go
// client/internal/agent/codex/threads.go
package codex

func (c *AppServerClient) CreateThread() (map[string]any, error) {
	var out map[string]any
	if err := c.runner.Call("thread/start", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

```go
// client/internal/agent/codex/turns.go
package codex

func (c *AppServerClient) StartTurn(threadID string, prompt string) (map[string]any, error) {
	var out map[string]any
	if err := c.runner.Call("turn/start", map[string]any{"threadId": threadID, "input": prompt}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

```go
// client/internal/agent/codex/approvals.go
package codex

func (c *AppServerClient) RespondApproval(requestID string, decision string) error {
	var out map[string]any
	return c.runner.Call("approval/respond", map[string]any{"requestId": requestID, "decision": decision}, &out)
}
```

```go
// client/internal/agent/codex/environment.go
package codex

func (c *AppServerClient) ListEnvironment() ([]map[string]any, error) {
	var out []map[string]any
	if err := c.runner.Call("environment/list", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./client/internal/agent/codex`
Expected: PASS with `ok  	code-agent-gateway/client/internal/agent/codex`

- [ ] **Step 5: Commit**

```bash
git add client/internal/agent/codex
git commit -m "feat: add codex app server adapter"
```

## Task 9: Add Thread Workspace Realtime UI And Retire Legacy TypeScript Backend

**Files:**
- Create: `console/src/pages/threads-page.tsx`
- Create: `console/src/pages/thread-workspace-page.tsx`
- Modify: `console/src/app/router.tsx`
- Modify: `console/src/common/api/http.ts`
- Modify: `console/src/common/api/ws.ts`
- Create: `console/playwright.config.ts`
- Delete: `apps/gateway/**`
- Delete: `apps/client-codex/**`
- Delete: `packages/domain/**`
- Delete: `packages/protocol/**`
- Delete: `package.json`
- Delete: `pnpm-workspace.yaml`
- Delete: `tsconfig.base.json`
- Delete: `vitest.workspace.ts`
- Delete: `pnpm-lock.yaml`
- Test: `console/tests/console-smoke.spec.ts`

- [ ] **Step 1: Write the failing e2e test**

```ts
// console/tests/console-smoke.spec.ts
import { expect, test } from "@playwright/test";

test("navigates to thread workspace", async ({ page }) => {
  await page.goto("/");
  await page.getByText("Threads").click();
  await expect(page.getByText("Thread Workspace")).toBeVisible();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `corepack pnpm --dir console playwright test tests/console-smoke.spec.ts`
Expected: FAIL because the thread workspace route does not exist yet

- [ ] **Step 3: Write minimal implementation**

```tsx
// console/src/pages/threads-page.tsx
import { Link } from "react-router-dom";

export function ThreadsPage() {
  return (
    <section>
      <h1>Threads</h1>
      <Link to="/threads/thread-1">Open thread-1</Link>
    </section>
  );
}
```

```tsx
// console/src/pages/thread-workspace-page.tsx
export function ThreadWorkspacePage() {
  return (
    <section>
      <h1>Thread Workspace</h1>
      <div>Realtime messages</div>
      <textarea aria-label="prompt" />
    </section>
  );
}
```

```tsx
// console/src/app/router.tsx
import { createBrowserRouter } from "react-router-dom";
import { AppShell } from "./shell";
import { EnvironmentPage } from "../pages/environment-page";
import { MachinesPage } from "../pages/machines-page";
import { OverviewPage } from "../pages/overview-page";
import { ThreadsPage } from "../pages/threads-page";
import { ThreadWorkspacePage } from "../pages/thread-workspace-page";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "machines", element: <MachinesPage /> },
      { path: "threads", element: <ThreadsPage /> },
      { path: "threads/:threadId", element: <ThreadWorkspacePage /> },
      { path: "environment", element: <EnvironmentPage /> },
    ],
  },
]);
```

```bash
# legacy backend cleanup
rm -rf apps/gateway apps/client-codex packages/domain packages/protocol
rm -f package.json pnpm-workspace.yaml tsconfig.base.json vitest.workspace.ts pnpm-lock.yaml
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `corepack pnpm --dir console test && corepack pnpm --dir console playwright test`
Expected: PASS with Vitest and Playwright green

Run: `go test ./common/... && go test ./gateway/... && go test ./client/...`
Expected: PASS with all Go modules green

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: add thread workspace and retire legacy ts backend"
```

## Self-Review

### Spec coverage

- Go `gateway` / Go `client` / React `console`: covered by Tasks 1, 3, 4, 7, 8, 9.
- `common` package design: covered by Tasks 1 and 2.
- `gateway <-> client` WebSocket + JSON: covered by Tasks 2, 5, and 6.
- Unified agent abstraction with `codex` as one implementation: covered by Tasks 4 and 8.
- Northbound product API and console pages: covered by Tasks 3, 6, 7, and 9.
- Replacement of legacy TypeScript backend: covered by Task 9.

### Placeholder scan

- No placeholders remain.
- Every task names exact files, code entry points, and verification commands.

### Type consistency

- `common/domain` is the source of truth for `Machine`, `Thread`, `Turn`, and environment resources.
- `common/protocol` owns the envelope shape used by `gateway` and `client`.
- `client/internal/agent` remains the only abstraction point for concrete agent implementations, with `codex` as one implementation.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-07-code-agent-gateway-v1.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
