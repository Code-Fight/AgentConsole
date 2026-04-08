package websocket

import (
	"context"
	"net/http"
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
	conn    *cws.Conn
	writeMu sync.Mutex
}

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

		client := &consoleConn{conn: conn}

		h.mu.Lock()
		h.clients[client] = struct{}{}
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			_ = conn.Close(cws.StatusNormalClosure, "done")
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
		client.writeMu.Lock()
		err := client.conn.Write(context.Background(), cws.MessageText, encoded)
		client.writeMu.Unlock()
		if err != nil {
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			_ = client.conn.Close(cws.StatusInternalError, "write failed")
		}
	}

	return nil
}
