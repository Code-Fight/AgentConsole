package websocket

import (
	"context"
	"net/http"
	"sync"

	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	cws "github.com/coder/websocket"
)

type ClientHub struct {
	mu      sync.Mutex
	clients map[string]struct{}
}

func NewClientHub() *ClientHub {
	return &ClientHub{
		clients: map[string]struct{}{},
	}
}

func (h *ClientHub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.clients)
}

func (h *ClientHub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/client", func(w http.ResponseWriter, r *http.Request) {
		conn, err := cws.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(cws.StatusNormalClosure, "done")

		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			return
		}
		if envelope.Name != "client.register" || envelope.MachineID == "" {
			return
		}

		h.mu.Lock()
		h.clients[envelope.MachineID] = struct{}{}
		h.mu.Unlock()
	})

	return mux
}
