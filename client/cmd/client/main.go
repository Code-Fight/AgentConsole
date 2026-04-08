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
	const connectTimeout = 5 * time.Second
	const reconnectMaxBackoff = 5 * time.Second

	cfg := config.Read()

	snap, err := snapshot.Build(codex.NewFakeAdapter())
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.GatewayURL, len(snap.Threads))

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	backoff := time.Duration(0)
	for {
		if shutdownCtx.Err() != nil {
			return
		}

		dialCtx, cancelDial := context.WithTimeout(shutdownCtx, connectTimeout)
		conn, err := gateway.Dial(dialCtx, cfg.GatewayURL)
		cancelDial()
		if err != nil {
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return
			}
			continue
		}

		session := gateway.NewSession(cfg.MachineID, func(msg []byte) error {
			return conn.Write(shutdownCtx, cws.MessageText, msg)
		}, time.Now)
		if err := session.Register(); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "register-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return
			}
			continue
		}

		backoff = 0
		if err := runHeartbeatLoop(shutdownCtx, session, heartbeatInterval); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "reconnect")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return
			}
			continue
		}

		_ = conn.Close(cws.StatusNormalClosure, "done")
		return
	}
}

func runHeartbeatLoop(ctx context.Context, session *gateway.Session, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := session.Heartbeat(); err != nil {
				return err
			}
		}
	}
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	if current <= 0 {
		return 1 * time.Second
	}

	next := current * 2
	if next > max {
		return max
	}

	return next
}
