package main

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/config"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
	ws "code-agent-gateway/gateway/internal/websocket"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	clientHub := ws.NewClientHub()
	handler := api.NewServer(registry.NewStore(), runtimeindex.NewStore(), clientHub.Handler())
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	log.Fatal(http.ListenAndServe(addr, handler))
}
