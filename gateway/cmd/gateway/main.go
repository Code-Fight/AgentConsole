package main

import (
	"log"
	"net/http"
	"strconv"

	"code-agent-gateway/gateway/internal/api"
	"code-agent-gateway/gateway/internal/config"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/runtimeindex"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	handler := api.NewServer(registry.NewStore(), runtimeindex.NewStore())
	log.Fatal(http.ListenAndServe(
		cfg.Host+":"+strconv.Itoa(cfg.Port),
		handler,
	))
}
