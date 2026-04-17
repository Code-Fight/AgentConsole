package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/common/version"
	"code-agent-gateway/gateway/internal/config"
	cws "github.com/coder/websocket"
)

const testGatewayAPIKey = "test-key"

func TestBuildServerHandlerWiresClientSnapshotsIntoHTTPViews(t *testing.T) {
	serverHandler, err := buildServerHandler(config.Config{
		Host:             "0.0.0.0",
		Port:             8080,
		SettingsFilePath: t.TempDir() + "/settings.json",
		APIKey:           testGatewayAPIKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(serverHandler)
	defer server.Close()

	conn, _, err := cws.Dial(context.Background(), "ws"+server.URL[4:]+"/ws/client", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(cws.StatusNormalClosure, "done")

	if err := writeEnvelope(conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySystem,
		Name:      "client.register",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T13:00:00Z",
		Payload:   []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "machine.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T13:00:01Z",
		Payload:   []byte(`{"machine":{"id":"machine-01","name":"Dev Mac","status":"online"}}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "thread.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T13:00:02Z",
		Payload:   []byte(`{"threads":[{"threadId":"thread-01","machineId":"machine-01","status":"idle","title":"One"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(conn, protocol.Envelope{
		Version:   version.CurrentProtocolVersion,
		Category:  protocol.CategorySnapshot,
		Name:      "environment.snapshot",
		MachineID: "machine-01",
		Timestamp: "2026-04-08T13:00:03Z",
		Payload:   []byte(`{"environment":[{"resourceId":"skill-01","machineId":"machine-01","kind":"skill","displayName":"Skill A","status":"enabled","restartRequired":false,"lastObservedAt":"2026-04-08T13:00:03Z"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		machines := mustGetMachines(t, server.URL+"/machines")
		if len(machines) != 1 {
			return false
		}
		if machines[0].ID != "machine-01" || machines[0].Status != domain.MachineStatusOnline {
			return false
		}

		threads := mustGetThreads(t, server.URL+"/threads")
		if len(threads) != 1 {
			return false
		}
		if threads[0].ThreadID != "thread-01" || threads[0].MachineID != "machine-01" {
			return false
		}

		skills := mustGetEnvironment(t, server.URL+"/environment/skills")
		if len(skills) != 1 {
			return false
		}
		return skills[0].ResourceID == "skill-01" && skills[0].MachineID == "machine-01"
	})
}

func writeEnvelope(conn *cws.Conn, envelope protocol.Envelope) error {
	encoded, err := transport.Encode(envelope)
	if err != nil {
		return err
	}
	return conn.Write(context.Background(), cws.MessageText, encoded)
}

func mustGetMachines(t *testing.T, url string) []domain.Machine {
	t.Helper()

	resp, err := getWithAuth(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body struct {
		Items []domain.Machine `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Items
}

func mustGetThreads(t *testing.T, url string) []domain.Thread {
	t.Helper()

	resp, err := getWithAuth(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body struct {
		Items []domain.Thread `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Items
}

func mustGetEnvironment(t *testing.T, url string) []domain.EnvironmentResource {
	t.Helper()

	resp, err := getWithAuth(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var body struct {
		Items []domain.EnvironmentResource `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Items
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition did not become true before timeout")
}

func getWithAuth(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+testGatewayAPIKey)
	return http.DefaultClient.Do(req)
}
