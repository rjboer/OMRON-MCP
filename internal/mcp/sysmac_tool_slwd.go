package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func sysmacSLWDTools() []tool {
	return []tool{
		{Name: "sysmac_slwd_read", Description: "Read an SLWD variable source.", InputSchema: objectSchema(map[string]any{"path": stringProperty("SLWD variable source path")}, "path")},
		{Name: "sysmac_slwd_write", Description: "Update an SLWD variable source while preserving unknown lines. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"path": stringProperty("SLWD variable source path"), "variables": map[string]any{"type": "array", "description": "Desired variable records", "items": map[string]any{"type": "object"}}, "delete_names": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "confirm_direct_write": map[string]any{"type": "boolean", "description": "Must be true to mutate project files"}, "protected_root": stringProperty("Optional root that must not be written")}, "path", "variables", "confirm_direct_write")},
	}
}

func sysmacSLWDToolHandlers() map[string]sysmacToolHandler {
	return map[string]sysmacToolHandler{
		"sysmac_slwd_read":  handleSLWDRead,
		"sysmac_slwd_write": handleSLWDWrite,
	}
}

func handleSLWDRead(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("path", true)
	if err != nil {
		return nil, err
	}
	variables, err := sysmac.ReadSLWDVariables(path)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "read", "path": path, "variables": variables}), nil
}

func handleSLWDWrite(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("path", true)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_direct_write", "confirm_direct_write must be true for direct project mutation"); err != nil {
		return nil, err
	}
	var variables []sysmac.SLWDVariable
	if rawVariables, ok := call.arguments["variables"]; !ok || json.Unmarshal(rawVariables, &variables) != nil {
		return nil, errors.New("variables must be an array of SLWD variable records")
	}
	var deleteNames []string
	if rawDelete, ok := call.arguments["delete_names"]; ok && json.Unmarshal(rawDelete, &deleteNames) != nil {
		return nil, errors.New("delete_names must be an array of strings")
	}
	protectedRoot, err := call.string("protected_root", false)
	if err != nil {
		return nil, err
	}
	before, err := sysmac.SHA256File(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SLWD source: %w", err)
	}
	document, err := sysmac.ParseSLWDDocument(data)
	if err != nil {
		return nil, err
	}
	for _, variable := range variables {
		if err := document.UpdateVariable(variable); err != nil {
			if err := document.AddVariable(variable); err != nil {
				return nil, err
			}
		}
	}
	for _, name := range deleteNames {
		if err := document.DeleteVariable(name); err != nil {
			return nil, err
		}
	}
	if err := sysmac.WriteSLWDAtomic(path, document, protectedRoot); err != nil {
		return nil, err
	}
	after, err := sysmac.SHA256File(path)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "applied", "path": path, "changed_files": []string{path, path + ".backup"}, "source_hash_before": before, "source_hash_after": after, "backup_path": path + ".backup"}), nil
}
