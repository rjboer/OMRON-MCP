package mcp

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProbeRejectsMissingExecutable(t *testing.T) {
	health, err := Probe(context.Background(), "", "")
	if err == nil {
		t.Fatal("Probe() error = nil, want configuration error")
	}
	if health.Status != HealthDisconnected {
		t.Fatalf("health status = %q, want disconnected", health.Status)
	}
}

func TestProbeStartsConfiguredServer(t *testing.T) {
	executable := os.Getenv("OMRON_MCP_TEST_EXECUTABLE")
	if executable == "" {
		t.Skip("OMRON_MCP_TEST_EXECUTABLE is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	health, err := ProbeWithArgs(ctx, executable, "", "--mcp")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if health.Status != HealthConnected || health.ToolCount != 18 || health.ProtocolVersion != "2025-11-25" {
		t.Fatalf("health = %#v", health)
	}
}
