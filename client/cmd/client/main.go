package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/gateway"
	"code-agent-gateway/client/internal/snapshot"
	cws "github.com/coder/websocket"
)

func main() {
	const heartbeatInterval = 30 * time.Second

	cfg := config.Read()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := gateway.Dial(ctx, cfg.GatewayURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(cws.StatusNormalClosure, "done")

	session := gateway.NewSession(cfg.MachineID, func(msg []byte) error {
		return conn.Write(context.Background(), cws.MessageText, msg)
	}, time.Now)
	if err := session.Register(); err != nil {
		panic(err)
	}

	snap, err := snapshot.Build(codex.NewFakeAdapter())
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.GatewayURL, len(snap.Threads))

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownCtx.Done():
			return
		case <-ticker.C:
			if err := session.Heartbeat(); err != nil {
				panic(err)
			}
		}
	}
}
