package main

import (
	"testing"
	"time"
)

func TestNextBackoffStartsAtOneSecond(t *testing.T) {
	got := nextBackoff(0, 5*time.Second)
	if got != 1*time.Second {
		t.Fatalf("expected initial backoff to be 1s, got %s", got)
	}
}

func TestNextBackoffCapsAtMax(t *testing.T) {
	got := nextBackoff(4*time.Second, 5*time.Second)
	if got != 5*time.Second {
		t.Fatalf("expected capped backoff of 5s, got %s", got)
	}
}
