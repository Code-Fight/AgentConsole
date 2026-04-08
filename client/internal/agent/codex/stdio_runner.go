package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

type stdioProcess interface {
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type execCommandFunc func(ctx context.Context, name string, args ...string) (stdioProcess, error)

type StdioRunner struct {
	writer     io.WriteCloser
	writeMu    sync.Mutex
	pendingMu  sync.Mutex
	pending    map[int64]chan rpcCallResult
	nextID     int64
	closeOnce  sync.Once
	closed     chan struct{}
	terminalMu sync.Mutex
	terminal   error
	wait       func() error
}

type rpcCallResult struct {
	result json.RawMessage
	err    error
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonRPCResponse struct {
	ID     *int64          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *jsonRPCError   `json:"error"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewStdioRunner(ctx context.Context, codexBin string) (*StdioRunner, error) {
	return newStdioRunner(ctx, codexBin, execCommandFunc(startExecCommand))
}

func newStdioRunner(ctx context.Context, codexBin string, execFn execCommandFunc) (*StdioRunner, error) {
	process, err := execFn(ctx, codexBin, "app-server", "--listen", "stdio://")
	if err != nil {
		return nil, err
	}

	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := process.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}

	return newStdioRunnerFromStreams(stdout, stdin, process.Wait), nil
}

func newStdioRunnerFromStreams(stdout io.Reader, stdin io.WriteCloser, wait func() error) *StdioRunner {
	runner := &StdioRunner{
		writer:  stdin,
		pending: make(map[int64]chan rpcCallResult),
		closed:  make(chan struct{}),
		wait:    wait,
	}
	go runner.readLoop(stdout)
	return runner
}

func (r *StdioRunner) Call(method string, payload any, out any) error {
	requestID := atomic.AddInt64(&r.nextID, 1)
	responseCh := make(chan rpcCallResult, 1)

	r.pendingMu.Lock()
	r.pending[requestID] = responseCh
	r.pendingMu.Unlock()

	request := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  payload,
	}

	r.writeMu.Lock()
	err := json.NewEncoder(r.writer).Encode(request)
	r.writeMu.Unlock()
	if err != nil {
		r.removePending(requestID)
		return err
	}

	result := <-responseCh
	if result.err != nil {
		return fmt.Errorf("json-rpc %s failed: %w", method, result.err)
	}
	if out == nil || len(result.result) == 0 || string(result.result) == "null" {
		return nil
	}
	return json.Unmarshal(result.result, out)
}

func (r *StdioRunner) Close() error {
	r.closeOnce.Do(func() {
		close(r.closed)
		if r.writer != nil {
			_ = r.writer.Close()
		}
		if r.wait != nil {
			r.setTerminal(r.wait())
		}
	})
	return nil
}

func (r *StdioRunner) readLoop(stdout io.Reader) {
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				r.failPending(io.EOF)
				return
			}
			r.failPending(err)
			return
		}

		var response jsonRPCResponse
		if err := json.Unmarshal(line, &response); err != nil {
			continue
		}
		if response.ID == nil {
			continue
		}

		r.pendingMu.Lock()
		responseCh := r.pending[*response.ID]
		delete(r.pending, *response.ID)
		r.pendingMu.Unlock()
		if responseCh == nil {
			continue
		}

		if response.Error != nil {
			responseCh <- rpcCallResult{err: fmt.Errorf("%s", response.Error.Message)}
			continue
		}
		responseCh <- rpcCallResult{result: response.Result}
	}
}

func (r *StdioRunner) removePending(id int64) {
	r.pendingMu.Lock()
	delete(r.pending, id)
	r.pendingMu.Unlock()
}

func (r *StdioRunner) failPending(err error) {
	r.setTerminal(err)

	r.pendingMu.Lock()
	defer r.pendingMu.Unlock()
	for id, responseCh := range r.pending {
		responseCh <- rpcCallResult{err: err}
		delete(r.pending, id)
	}
}

func (r *StdioRunner) setTerminal(err error) {
	if err == nil {
		return
	}
	r.terminalMu.Lock()
	defer r.terminalMu.Unlock()
	if r.terminal == nil {
		r.terminal = err
	}
}

func startExecCommand(ctx context.Context, name string, args ...string) (stdioProcess, error) {
	return exec.CommandContext(ctx, name, args...), nil
}
