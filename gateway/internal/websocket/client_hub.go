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
	clients map[string]*cws.Conn
}

func NewClientHub() *ClientHub {
	return &ClientHub{
		clients: map[string]*cws.Conn{},
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

		registeredMachineID := ""
		defer func() {
			if registeredMachineID == "" {
				return
			}

			h.mu.Lock()
			if h.clients[registeredMachineID] == conn {
				delete(h.clients, registeredMachineID)
			}
			h.mu.Unlock()
		}()

		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}

			var envelope protocol.Envelope
			if err := transport.Decode(data, &envelope); err != nil {
				continue
			}
			if envelope.Category != protocol.CategorySystem {
				continue
			}

			switch envelope.Name {
			case "client.register":
				if envelope.MachineID == "" {
					continue
				}
				h.mu.Lock()
				if registeredMachineID != "" && registeredMachineID != envelope.MachineID && h.clients[registeredMachineID] == conn {
					delete(h.clients, registeredMachineID)
				}
				h.clients[envelope.MachineID] = conn
				registeredMachineID = envelope.MachineID
				h.mu.Unlock()
			case "client.heartbeat":
				continue
			}
		}
	})

	return mux
}
