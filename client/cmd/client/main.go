package main

import (
	"fmt"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/snapshot"
)

func main() {
	cfg := config.Read()

	snap, err := snapshot.Build(codex.NewFakeAdapter())
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.GatewayURL, len(snap.Threads))
}
