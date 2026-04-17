package main

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/config"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
	"code-agent-gateway/gateway/internal/settings"
	ws "code-agent-gateway/gateway/internal/websocket"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	handler, err := buildServerHandler(cfg)
	if err != nil {
		log.Fatal(err)
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	log.Fatal(http.ListenAndServe(addr, handler))
}

func buildServerHandler(cfg config.Config) (http.Handler, error) {
	reg := registry.NewStore()
	idx := runtimeindex.NewStore()
	router := routing.NewRouter()
	consoleHub := ws.NewConsoleHub()
	clientHub := ws.NewClientHubWithStores(reg, idx, router)
	clientHub.SetConsoleHub(consoleHub)
	settingsStore, err := settings.NewFileStore(cfg.SettingsFilePath, []domain.AgentDescriptor{
		{AgentType: domain.AgentTypeCodex, DisplayName: "Codex"},
	})
	if err != nil {
		return nil, err
	}
	return api.NewServerWithSettingsAndAPIKey(reg, idx, router, clientHub, settingsStore, cfg.APIKey, clientHub.Handler(), consoleHub.Handler()), nil
}
