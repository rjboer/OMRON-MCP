package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type sysmacToolCall struct {
	context   context.Context
	arguments map[string]json.RawMessage
}

type sysmacToolHandler func(sysmacToolCall) (map[string]any, error)

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func callSysmacTool(raw json.RawMessage) (map[string]any, error) {
	return callSysmacToolContext(context.Background(), raw)
}

func callSysmacToolContext(ctx context.Context, raw json.RawMessage) (map[string]any, error) {
	var params struct {
		Name      string                     `json:"name"`
		Arguments map[string]json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}
	if params.Name == "" {
		return nil, errors.New("tool name is required")
	}
	handler, ok := sysmacToolHandlers()[params.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
	return handler(sysmacToolCall{context: ctx, arguments: params.Arguments})
}

func (call sysmacToolCall) string(name string, required bool) (string, error) {
	value, ok := call.arguments[name]
	if !ok {
		if required {
			return "", fmt.Errorf("argument %q is required", name)
		}
		return "", nil
	}
	var result string
	if err := json.Unmarshal(value, &result); err != nil || (required && strings.TrimSpace(result) == "") {
		return "", fmt.Errorf("argument %q must be a non-empty string", name)
	}
	return result, nil
}

func (call sysmacToolCall) confirmed(name, message string) error {
	var confirmed bool
	if rawConfirm, ok := call.arguments[name]; !ok || json.Unmarshal(rawConfirm, &confirmed) != nil || !confirmed {
		return errors.New(message)
	}
	return nil
}

func sourceHash(value string) string {
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", hash[:])
}

func toolSuccess(value any) map[string]any {
	data, _ := json.Marshal(value)
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(data)}}, "structuredContent": value, "isError": false}
}
