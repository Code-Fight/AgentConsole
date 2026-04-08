package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code-agent-gateway/client/internal/agent/codex"
	"code-agent-gateway/client/internal/agent/manager"
	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/config"
	"code-agent-gateway/client/internal/gateway"
	"code-agent-gateway/client/internal/snapshot"
	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	cws "github.com/coder/websocket"
)

func main() {
	const heartbeatInterval = 30 * time.Second
	const connectTimeout = 5 * time.Second
	const reconnectMaxBackoff = 5 * time.Second
	const runtimeName = "codex"

	cfg := config.Read()

	adapter := codex.NewFakeAdapter()
	adapter.SeedSnapshot(
		[]domain.Thread{
			{
				ThreadID:  "thread-01",
				MachineID: cfg.MachineID,
				Status:    domain.ThreadStatusIdle,
				Title:     "Gateway bootstrap thread",
			},
		},
		[]domain.EnvironmentResource{
			{
				ResourceID:      "skill-01",
				MachineID:       cfg.MachineID,
				Kind:            domain.EnvironmentKindSkill,
				DisplayName:     "Bootstrap Skill",
				Status:          domain.EnvironmentResourceStatusEnabled,
				RestartRequired: false,
				LastObservedAt:  time.Now().UTC().Format(time.RFC3339),
			},
		},
	)

	runtimeRegistry := agentregistry.New()
	runtimeRegistry.Register(runtimeName, adapter)
	agentManager := manager.New(runtimeRegistry)

	machine := domain.Machine{
		ID:     cfg.MachineID,
		Name:   cfg.MachineID,
		Status: domain.MachineStatusOnline,
	}

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
		if err := sendLiveSnapshot(session, machine, agentManager, runtimeName); err != nil {
			_ = conn.Close(cws.StatusNormalClosure, "snapshot-failed")
			backoff = nextBackoff(backoff, reconnectMaxBackoff)
			if !sleepWithContext(shutdownCtx, backoff) {
				return
			}
			continue
		}

		backoff = 0
		if err := runConnection(shutdownCtx, conn, session, agentManager, runtimeName, heartbeatInterval); err != nil {
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

func runConnection(ctx context.Context, conn *cws.Conn, session *gateway.Session, mgr *manager.Manager, runtimeName string, heartbeatInterval time.Duration) error {
	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- runHeartbeatLoop(loopCtx, session, heartbeatInterval)
	}()

	go func() {
		errCh <- runCommandLoop(loopCtx, conn, session, mgr, runtimeName)
	}()

	err := <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

func sendLiveSnapshot(session *gateway.Session, machine domain.Machine, mgr *manager.Manager, runtimeName string) error {
	snap, err := mgr.Snapshot(runtimeName)
	if err != nil {
		return err
	}

	return sendInitialSnapshot(session, machine, snap)
}

func sendInitialSnapshot(session *gateway.Session, machine domain.Machine, snap snapshot.Snapshot) error {
	if err := session.MachineSnapshot(machine); err != nil {
		return err
	}
	if err := session.ThreadSnapshot(snap.Threads); err != nil {
		return err
	}
	return session.EnvironmentSnapshot(snap.Environment)
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

func runCommandLoop(ctx context.Context, conn *cws.Conn, session *gateway.Session, mgr *manager.Manager, runtimeName string) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		var envelope protocol.Envelope
		if err := transport.Decode(data, &envelope); err != nil {
			continue
		}

		if envelope.Category != protocol.CategoryCommand {
			continue
		}

		if err := handleCommandEnvelope(session, mgr, runtimeName, envelope); err != nil {
			return err
		}
	}
}

func handleCommandEnvelope(session *gateway.Session, mgr *manager.Manager, runtimeName string, envelope protocol.Envelope) error {
	switch envelope.Name {
	case "thread.create":
		var payload protocol.ThreadCreateCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		thread, err := mgr.CreateThread(runtimeName, agenttypes.CreateThreadParams{
			Title: payload.Title,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.ThreadCreateCommandResult{
			Thread: thread,
		}); err != nil {
			return err
		}

		threads, err := mgr.Threads(runtimeName)
		if err != nil {
			return err
		}

		return session.ThreadSnapshot(threads)
	case "turn.start":
		var payload protocol.TurnStartCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), "")
		}

		result, err := mgr.StartTurn(runtimeName, agenttypes.StartTurnParams{
			ThreadID: payload.ThreadID,
			Input:    payload.Input,
		})
		if err != nil {
			return session.CommandRejected(envelope.RequestID, envelope.Name, err.Error(), payload.ThreadID)
		}

		if err := session.CommandCompleted(envelope.RequestID, envelope.Name, protocol.TurnStartCommandResult{
			TurnID:   result.TurnID,
			ThreadID: result.ThreadID,
		}); err != nil {
			return err
		}

		for _, delta := range result.Deltas {
			if err := session.TurnDelta(envelope.RequestID, protocol.TurnDeltaPayload{
				ThreadID: result.ThreadID,
				TurnID:   result.TurnID,
				Sequence: delta.Sequence,
				Delta:    delta.Delta,
			}); err != nil {
				return err
			}
		}

		return session.TurnCompleted(envelope.RequestID, protocol.TurnCompletedPayload{
			Turn: domain.Turn{
				TurnID:   result.TurnID,
				ThreadID: result.ThreadID,
				Status:   domain.TurnStatusCompleted,
			},
		})
	default:
		return session.CommandRejected(envelope.RequestID, envelope.Name, "unsupported command", "")
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
