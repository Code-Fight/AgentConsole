package version

import "testing"

func TestCurrentProtocolVersion(t *testing.T) {
	if CurrentProtocolVersion != "v1" {
		t.Fatalf("expected v1, got %q", CurrentProtocolVersion)
	}
}
