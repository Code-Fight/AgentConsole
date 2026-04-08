package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

func TestStdioRunnerCallDispatchesJSONRPCRequestAndDecodesResponse(t *testing.T) {
	serverReader, runnerWriter := io.Pipe()
	runnerReader, serverWriter := io.Pipe()

	runner := newStdioRunnerFromStreams(runnerReader, runnerWriter, nil)
	defer runner.Close()

	done := make(chan error, 1)
	go func() {
		defer close(done)

		var request struct {
			JSONRPC string         `json:"jsonrpc"`
			ID      int64          `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}
		if err := json.NewDecoder(serverReader).Decode(&request); err != nil {
			done <- err
			return
		}
		if request.JSONRPC != "2.0" {
			done <- errors.New("unexpected jsonrpc version")
			return
		}
		if request.Method != "thread/list" {
			done <- errors.New("unexpected method")
			return
		}
		if request.Params["cursor"] != "next-1" {
			done <- errors.New("unexpected params")
			return
		}

		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result": map[string]any{
				"data": []map[string]any{
					{"id": "thread-1", "name": "Bootstrap thread", "status": "idle"},
				},
				"nextCursor": nil,
			},
		}
		if err := json.NewEncoder(serverWriter).Encode(response); err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := runner.Call("thread/list", map[string]any{"cursor": "next-1"}, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].ID != "thread-1" {
		t.Fatalf("unexpected response: %+v", out)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive request")
	}
}

func TestStdioRunnerCallReturnsServerError(t *testing.T) {
	serverReader, runnerWriter := io.Pipe()
	runnerReader, serverWriter := io.Pipe()

	runner := newStdioRunnerFromStreams(runnerReader, runnerWriter, nil)
	defer runner.Close()

	go func() {
		defer serverReader.Close()
		var request struct {
			ID int64 `json:"id"`
		}
		_ = json.NewDecoder(serverReader).Decode(&request)
		_ = json.NewEncoder(serverWriter).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"error": map[string]any{
				"code":    -32000,
				"message": "boom",
			},
		})
	}()

	var out map[string]any
	err := runner.Call("initialize", map[string]any{"clientInfo": map[string]any{"name": "client"}}, &out)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if err.Error() != "json-rpc initialize failed: boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStdioRunnerDispatchesNotificationsWithoutConsumingResponses(t *testing.T) {
	serverReader, runnerWriter := io.Pipe()
	runnerReader, serverWriter := io.Pipe()

	runner := newStdioRunnerFromStreams(runnerReader, runnerWriter, nil)
	defer runner.Close()

	notifications := make(chan jsonRPCNotification, 1)
	runner.SetNotificationHandler(func(notification jsonRPCNotification) {
		notifications <- notification
	})

	go func() {
		defer serverReader.Close()

		var request struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(serverReader).Decode(&request); err != nil {
			t.Errorf("decode request failed: %v", err)
			return
		}

		if err := json.NewEncoder(serverWriter).Encode(map[string]any{
			"jsonrpc": "2.0",
			"method":  "turn/started",
			"params": map[string]any{
				"threadId": "thread-1",
				"turnId":   "turn-1",
			},
		}); err != nil {
			t.Errorf("encode notification failed: %v", err)
			return
		}

		if err := json.NewEncoder(serverWriter).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result": map[string]any{
				"turn": map[string]any{
					"id": "turn-1",
				},
			},
		}); err != nil {
			t.Errorf("encode response failed: %v", err)
		}
	}()

	var out turnStartResponse
	if err := runner.Call("turn/start", map[string]any{"threadId": "thread-1"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.Turn.ID != "turn-1" {
		t.Fatalf("unexpected response: %+v", out)
	}

	select {
	case notification := <-notifications:
		if notification.Method != "turn/started" {
			t.Fatalf("unexpected notification method: %q", notification.Method)
		}

		var payload struct {
			ThreadID string `json:"threadId"`
			TurnID   string `json:"turnId"`
		}
		if err := json.Unmarshal(notification.Params, &payload); err != nil {
			t.Fatalf("unmarshal notification failed: %v", err)
		}
		if payload.ThreadID != "thread-1" || payload.TurnID != "turn-1" {
			t.Fatalf("unexpected notification payload: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected notification")
	}
}

func TestNewStdioRunnerStartsConfiguredCommand(t *testing.T) {
	ctx := context.Background()
	var gotName string
	var gotArgs []string

	serverReader, runnerWriter := io.Pipe()
	runnerReader, serverWriter := io.Pipe()
	defer serverWriter.Close()
	defer serverReader.Close()

	runner, err := newStdioRunner(ctx, "/opt/homebrew/bin/codex", execCommandFunc(func(context.Context, string, ...string) (stdioProcess, error) {
		return &fakeStdioProcess{
			stdin: func() io.WriteCloser {
				gotName = "/opt/homebrew/bin/codex"
				gotArgs = []string{"app-server", "--listen", "stdio://"}
				return runnerWriter
			}(),
			stdout: runnerReader,
		}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()

	if gotName != "/opt/homebrew/bin/codex" {
		t.Fatalf("unexpected command: %q", gotName)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "app-server" || gotArgs[1] != "--listen" || gotArgs[2] != "stdio://" {
		t.Fatalf("unexpected args: %#v", gotArgs)
	}
}

type fakeStdioProcess struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *fakeStdioProcess) StdinPipe() (io.WriteCloser, error) {
	return p.stdin, nil
}

func (p *fakeStdioProcess) StdoutPipe() (io.ReadCloser, error) {
	return p.stdout, nil
}

func (p *fakeStdioProcess) Start() error {
	return nil
}

func (p *fakeStdioProcess) Wait() error {
	return nil
}
