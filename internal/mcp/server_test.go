package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerNegotiatesCurrentProtocolAndListsAllTools(t *testing.T) {
	ctx := context.Background()
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := NewServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "omron-mcp-test", Version: "0.1.0"},
		&sdkmcp.ClientOptions{Capabilities: &sdkmcp.ClientCapabilities{}},
	)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})

	initialize := clientSession.InitializeResult()
	if initialize.ProtocolVersion != "2025-11-25" {
		t.Fatalf("negotiated protocol = %q, want 2025-11-25", initialize.ProtocolVersion)
	}
	if initialize.ServerInfo == nil || initialize.ServerInfo.Name != "omron-sysmac-mcp" {
		t.Fatalf("server info = %#v", initialize.ServerInfo)
	}
	if initialize.Capabilities.Tools == nil {
		t.Fatal("server did not advertise tools capability")
	}
	if initialize.Capabilities.Resources != nil || initialize.Capabilities.Prompts != nil {
		t.Fatalf("server advertised unsupported capabilities: %#v", initialize.Capabilities)
	}

	list, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(list.Tools) != 18 {
		t.Fatalf("tools = %d, want 18", len(list.Tools))
	}
}

func TestRegisteredToolCanBeCalledAfterInitialization(t *testing.T) {
	ctx := context.Background()
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := NewServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "omron-mcp-test", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})

	result, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "sysmac_project_inspect",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatalf("missing required argument should be a tool error: %#v", result)
	}
}
