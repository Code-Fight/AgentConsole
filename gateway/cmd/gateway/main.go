package main

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/config"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
	ws "code-agent-gateway/gateway/internal/websocket"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	handler := buildServerHandler()
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	log.Fatal(http.ListenAndServe(addr, handler))
}

func buildServerHandler() http.Handler {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	clientHub := ws.NewClientHubWithStores(reg, idx, router)
	return api.NewServer(reg, idx, router, clientHub, clientHub.Handler())
}
