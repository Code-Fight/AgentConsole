package websocket

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	cws "github.com/coder/websocket"
)

type ConsoleHub struct {
	mu      sync.RWMutex
	clients map[*consoleConn]struct{}
}

type consoleConn struct {
	threadID  string
	conn      *cws.Conn
	outbound  chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

const consoleOutboundBufferSize = 32

func NewConsoleHub() *ConsoleHub {
	return &ConsoleHub{
		clients: map[*consoleConn]struct{}{},
	}
}

func (h *ConsoleHub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := cws.Accept(w, r, nil)
		if err != nil {
			return
		}

		client := &consoleConn{
			threadID: strings.TrimSpace(r.URL.Query().Get("threadId")),
			conn:     conn,
			outbound: make(chan []byte, consoleOutboundBufferSize),
			done:     make(chan struct{}),
		}

		h.mu.Lock()
		h.clients[client] = struct{}{}
		h.mu.Unlock()

		go h.writeLoop(client)

		defer func() {
			h.removeClient(client, cws.StatusNormalClosure, "done")
		}()

		for {
			if _, _, err := conn.Read(context.Background()); err != nil {
				return
			}
		}
	})

	return mux
}

func (h *ConsoleHub) Broadcast(envelope protocol.Envelope) error {
	encoded, err := transport.Encode(envelope)
	if err != nil {
		return err
	}

	h.mu.RLock()
	clients := make([]*consoleConn, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if !shouldDeliverEnvelope(client.threadID, envelope) {
			continue
		}

		select {
		case client.outbound <- encoded:
		default:
			h.removeClient(client, cws.StatusPolicyViolation, "console client too slow")
		}
	}

	return nil
}

func (h *ConsoleHub) writeLoop(client *consoleConn) {
	if client == nil || client.conn == nil {
		return
	}

	for {
		select {
		case <-client.done:
			return
		case data := <-client.outbound:
			if err := client.conn.Write(context.Background(), cws.MessageText, data); err != nil {
				h.removeClient(client, cws.StatusInternalError, "write failed")
				return
			}
		}
	}
}

func (h *ConsoleHub) removeClient(client *consoleConn, status cws.StatusCode, reason string) {
	if client == nil {
		return
	}

	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()

	client.closeOnce.Do(func() {
		if client.done != nil {
			close(client.done)
		}
		if client.conn != nil {
			_ = client.conn.Close(status, reason)
		}
	})
}

func shouldDeliverEnvelope(threadID string, envelope protocol.Envelope) bool {
	if threadID == "" {
		return true
	}

	eventThreadID := envelopeThreadID(envelope)
	return eventThreadID != "" && eventThreadID == threadID
}

func envelopeThreadID(envelope protocol.Envelope) string {
	switch envelope.Name {
	case "approval.required":
		var payload protocol.ApprovalRequiredPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.ThreadID
		}
	case "approval.resolved":
		var payload protocol.ApprovalResolvedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.ThreadID
		}
	case "turn.delta":
		var payload protocol.TurnDeltaPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.ThreadID
		}
	case "turn.started":
		var payload protocol.TurnStartedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.ThreadID
		}
	case "turn.completed", "turn.failed":
		var payload protocol.TurnCompletedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.Turn.ThreadID
		}
	case "command.completed":
		var payload protocol.CommandCompletedPayload
		if err := transport.Decode(envelope.Payload, &payload); err != nil {
			return ""
		}
		switch payload.CommandName {
		case "turn.start":
			var result protocol.TurnStartCommandResult
			if err := transport.Decode(payload.Result, &result); err == nil {
				return result.ThreadID
			}
		case "thread.create":
			var result protocol.ThreadCreateCommandResult
			if err := transport.Decode(payload.Result, &result); err == nil {
				return result.Thread.ThreadID
			}
		}
	case "command.rejected":
		var payload protocol.CommandRejectedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			return payload.ThreadID
		}
	case "thread.updated":
		var payload protocol.ThreadUpdatedPayload
		if err := transport.Decode(envelope.Payload, &payload); err == nil {
			if payload.Thread != nil && payload.Thread.ThreadID != "" {
				return payload.Thread.ThreadID
			}
			return payload.ThreadID
		}
	}

	return ""
}
