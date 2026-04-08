package codex

import (
	"errors"
	"testing"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type fakeRunner struct {
	call func(method string, payload any, out any) error
}

func (r *fakeRunner) Call(method string, payload any, out any) error {
	if r.call != nil {
		return r.call(method, payload, out)
	}
	return nil
}

func TestClientListThreads(t *testing.T) {
	tests := []struct {
		name    string
		callErr error
	}{
		{
			name: "maps records",
		},
		{
			name:    "propagates runner error",
			callErr: errors.New("runner boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					if method != "thread/list" {
						t.Fatalf("unexpected method: %s", method)
					}
					if tt.callErr != nil {
						return tt.callErr
					}
					threads := out.(*[]ThreadRecord)
					*threads = []ThreadRecord{{ID: "thread-1", Title: "Investigate flaky test"}}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			threads, err := client.ListThreads()
			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(threads) != 1 {
				t.Fatalf("unexpected thread count: %d", len(threads))
			}
			if threads[0].ThreadID != "thread-1" {
				t.Fatalf("unexpected threads: %+v", threads)
			}
		})
	}
}

func TestClientListEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		callErr error
	}{
		{
			name: "maps records",
		},
		{
			name:    "propagates runner error",
			callErr: errors.New("environment list failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					if method != "environment/list" {
						t.Fatalf("unexpected method: %s", method)
					}
					if tt.callErr != nil {
						return tt.callErr
					}
					records := out.(*[]EnvironmentRecord)
					*records = []EnvironmentRecord{
						{
							ResourceID:      "skill-1",
							MachineID:       "machine-1",
							Kind:            domain.EnvironmentKindSkill,
							DisplayName:     "Skill A",
							Status:          domain.EnvironmentResourceStatusEnabled,
							RestartRequired: true,
							LastObservedAt:  "2026-04-08T10:00:00Z",
						},
					}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			environment, err := client.ListEnvironment()
			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(environment) != 1 {
				t.Fatalf("unexpected environment count: %d", len(environment))
			}
			if environment[0].ResourceID != "skill-1" || environment[0].DisplayName != "Skill A" {
				t.Fatalf("unexpected environment: %+v", environment)
			}
		})
	}
}

func TestClientCreateThreadUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		callErr error
	}{
		{
			name:  "uses thread/start payload keys",
			title: "Investigate flaky test",
		},
		{
			name:    "propagates runner error",
			title:   "Build failure",
			callErr: errors.New("thread start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					record := out.(*ThreadRecord)
					*record = ThreadRecord{ID: "thread-1", Title: tt.title, Status: domain.ThreadStatusIdle}
					return nil
				},
			}

			client := NewAppServerClient(runner)
			thread, err := client.CreateThread(agenttypes.CreateThreadParams{Title: tt.title})
			if gotMethod != "thread/start" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["title"] != tt.title {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if thread.ThreadID != "thread-1" || thread.Title != tt.title {
				t.Fatalf("unexpected thread: %+v", thread)
			}
		})
	}
}

func TestClientStartTurnUsesExpectedMethodAndPayload(t *testing.T) {
	tests := []struct {
		name     string
		threadID string
		prompt   string
		callErr  error
	}{
		{
			name:     "uses turn/start payload keys",
			threadID: "thread-1",
			prompt:   "run tests",
		},
		{
			name:     "propagates runner error",
			threadID: "thread-2",
			prompt:   "build",
			callErr:  errors.New("turn start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPayload any

			runner := &fakeRunner{
				call: func(method string, payload any, out any) error {
					gotMethod = method
					gotPayload = payload
					if tt.callErr != nil {
						return tt.callErr
					}
					result := out.(*struct {
						TurnID string `json:"turnId"`
					})
					result.TurnID = "turn-1"
					return nil
				},
			}

			client := NewAppServerClient(runner)
			result, err := client.StartTurn(agenttypes.StartTurnParams{
				ThreadID: tt.threadID,
				Input:    tt.prompt,
			})
			if gotMethod != "turn/start" {
				t.Fatalf("unexpected method: %s", gotMethod)
			}

			payloadMap, ok := gotPayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected payload type: %T", gotPayload)
			}
			if payloadMap["threadId"] != tt.threadID || payloadMap["input"] != tt.prompt {
				t.Fatalf("unexpected payload: %#v", payloadMap)
			}

			if tt.callErr != nil {
				if err == nil || err.Error() != tt.callErr.Error() {
					t.Fatalf("expected error %q, got %v", tt.callErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.TurnID != "turn-1" || result.ThreadID != tt.threadID {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}
