package mcp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type HealthStatus string

const (
	HealthConnected    HealthStatus = "connected"
	HealthDisconnected HealthStatus = "disconnected"
	HealthError        HealthStatus = "error"
)

type Health struct {
	Status          HealthStatus
	ServerName      string
	Version         string
	ProtocolVersion string
	ToolCount       int
	CheckedAt       time.Time
	Message         string
}

// Probe starts the configured MCP stdio executable and validates its basic
// protocol handshake. It does not open or mutate a Sysmac project.
func Probe(ctx context.Context, executable, workingDir string) (Health, error) {
	return ProbeWithArgs(ctx, executable, workingDir)
}

// ProbeWithArgs performs a complete MCP initialization lifecycle followed by
// tools/list, using the official SDK client and the negotiated protocol version.
func ProbeWithArgs(ctx context.Context, executable, workingDir string, args ...string) (Health, error) {
	if strings.TrimSpace(executable) == "" {
		err := errors.New("MCP executable is not configured")
		return Health{Status: HealthDisconnected, Message: err.Error()}, err
	}

	command := exec.CommandContext(ctx, executable, args...)
	if workingDir != "" {
		command.Dir = workingDir
	}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "omron-mcp-health", Version: serverVersion},
		&sdkmcp.ClientOptions{Capabilities: &sdkmcp.ClientCapabilities{}},
	)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: command}, nil)
	if err != nil {
		wrapped := fmt.Errorf("connect to MCP server: %w", err)
		return Health{Status: HealthDisconnected, Message: wrapped.Error()}, wrapped
	}
	defer session.Close()

	initialize := session.InitializeResult()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		wrapped := fmt.Errorf("list MCP tools: %w", err)
		return Health{Status: HealthError, Message: wrapped.Error()}, wrapped
	}
	if initialize == nil || initialize.ServerInfo == nil || initialize.ProtocolVersion == "" {
		err := errors.New("MCP initialize response is missing server information")
		return Health{Status: HealthError, Message: err.Error()}, err
	}
	return Health{
		Status:          HealthConnected,
		ServerName:      initialize.ServerInfo.Name,
		Version:         initialize.ServerInfo.Version,
		ProtocolVersion: initialize.ProtocolVersion,
		ToolCount:       len(tools.Tools),
		CheckedAt:       time.Now(),
		Message:         "MCP initialize, initialized notification, and tools/list succeeded",
	}, nil
}
