package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	serverName    = "omron-sysmac-mcp"
	serverVersion = "0.1.0"
)

type Server struct {
	sdk *sdkmcp.Server
}

type tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

func NewServer() *Server {
	sdk := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: serverName, Version: serverVersion},
		&sdkmcp.ServerOptions{
			Instructions: "Direct project mutation is available only through an explicit confirmed tool call; PLC operations are not exposed.",
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{},
			},
		},
	)
	for _, definition := range sysmacTools() {
		definition := definition
		sdkmcp.AddTool(sdk, &sdkmcp.Tool{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
		}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, arguments map[string]any) (*sdkmcp.CallToolResult, any, error) {
			rawArguments, err := json.Marshal(arguments)
			if err != nil {
				return nil, nil, err
			}
			raw, err := json.Marshal(map[string]any{
				"name":      definition.Name,
				"arguments": json.RawMessage(rawArguments),
			})
			if err != nil {
				return nil, nil, err
			}
			value, err := callSysmacToolContext(ctx, raw)
			if err != nil {
				return nil, nil, err
			}
			return nil, value["structuredContent"], nil
		})
	}
	return &Server{sdk: sdk}
}

func (s *Server) Connect(ctx context.Context, transport sdkmcp.Transport, options *sdkmcp.ServerSessionOptions) (*sdkmcp.ServerSession, error) {
	return s.sdk.Connect(ctx, transport, options)
}

func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	return s.sdk.Run(ctx, transport)
}

func (s *Server) RunStdio(ctx context.Context) error {
	return s.Run(ctx, &sdkmcp.StdioTransport{})
}
