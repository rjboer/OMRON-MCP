package mcp

import (
	"fmt"
	"strings"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func sysmacInspectTextTools() []tool {
	return []tool{
		{Name: "sysmac_project_inspect", Description: "Inspect a native Sysmac Solution or supported project container.", InputSchema: objectSchema(map[string]any{"path": stringProperty("Sysmac Solution directory or project container")}, "path")},
		{Name: "sysmac_structured_text_read", Description: "Read one native Structured Text POU body by entity ID.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory"), "entity_id": stringProperty("Stable Sysmac entity ID")}, "project_path", "entity_id")},
		{Name: "sysmac_structured_text_preview", Description: "Preview a Structured Text change without modifying the native project.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory"), "entity_id": stringProperty("Stable Sysmac entity ID"), "source_text": stringProperty("Proposed Structured Text")}, "project_path", "entity_id", "source_text")},
		{Name: "sysmac_structured_text_update", Description: "Directly update one native Structured Text POU body. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac Solution directory"), "entity_id": stringProperty("Stable Sysmac entity ID"), "source_text": stringProperty("Replacement Structured Text"), "confirm_direct_write": map[string]any{"type": "boolean", "description": "Must be true to mutate project files"}, "protected_root": stringProperty("Optional root that must not be written")}, "project_path", "entity_id", "source_text", "confirm_direct_write")},
	}
}

func sysmacInspectTextToolHandlers() map[string]sysmacToolHandler {
	return map[string]sysmacToolHandler{
		"sysmac_project_inspect":         handleProjectInspect,
		"sysmac_structured_text_read":    handleStructuredTextRead,
		"sysmac_structured_text_preview": handleStructuredTextPreview,
		"sysmac_structured_text_update":  handleStructuredTextUpdate,
	}
}

func handleProjectInspect(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("path", true)
	if err != nil {
		return nil, err
	}
	inspection, err := sysmac.Inspect(path)
	if err != nil {
		return nil, err
	}
	return toolSuccess(inspection), nil
}

func handleStructuredTextRead(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	inspection, err := sysmac.Inspect(path)
	if err != nil {
		return nil, err
	}
	text, err := sysmac.ReadStructuredTextEntity(path, entityID)
	if err != nil {
		return nil, err
	}
	entity, err := findEntity(inspection, entityID)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"project_path": path, "entity_id": entityID, "entity_name": entity.Name, "source_text": text}), nil
}

func handleStructuredTextPreview(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	proposed, err := call.string("source_text", true)
	if err != nil {
		return nil, err
	}
	inspection, err := sysmac.Inspect(path)
	if err != nil {
		return nil, err
	}
	entity, err := findEntity(inspection, entityID)
	if err != nil {
		return nil, err
	}
	current, err := sysmac.ReadStructuredTextEntity(path, entityID)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{
		"status": "preview", "project_path": path, "entity_id": entityID, "entity_name": entity.Name,
		"changed": current != proposed, "source_hash_before": sourceHash(current), "source_hash_after": sourceHash(proposed),
		"current_source_text": current, "proposed_source_text": proposed, "diff": lineDiff(current, proposed),
	}), nil
}

func handleStructuredTextUpdate(call sysmacToolCall) (map[string]any, error) {
	path, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	entityID, err := call.string("entity_id", true)
	if err != nil {
		return nil, err
	}
	source, err := call.string("source_text", true)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_direct_write", "confirm_direct_write must be true for direct project mutation"); err != nil {
		return nil, err
	}
	protectedRoot, err := call.string("protected_root", false)
	if err != nil {
		return nil, err
	}
	payload, err := sysmac.StructuredTextEntityPayloadPath(path, entityID)
	if err != nil {
		return nil, err
	}
	before, err := sysmac.SHA256File(payload)
	if err != nil {
		return nil, err
	}
	if err := sysmac.WriteStructuredTextEntity(path, entityID, source, protectedRoot); err != nil {
		return nil, err
	}
	after, err := sysmac.SHA256File(payload)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "applied", "project_path": path, "entity_id": entityID, "changed_files": []string{payload, payload + ".backup"}, "source_hash_before": before, "source_hash_after": after, "backup_path": payload + ".backup"}), nil
}

func lineDiff(current, proposed string) string {
	left := strings.Split(strings.ReplaceAll(current, "\r\n", "\n"), "\n")
	right := strings.Split(strings.ReplaceAll(proposed, "\r\n", "\n"), "\n")
	limit := len(left)
	if len(right) > limit {
		limit = len(right)
	}
	var lines []string
	for index := 0; index < limit; index++ {
		var oldLine, newLine string
		if index < len(left) {
			oldLine = left[index]
		}
		if index < len(right) {
			newLine = right[index]
		}
		switch {
		case oldLine == newLine:
			lines = append(lines, "  "+oldLine)
		case oldLine == "":
			lines = append(lines, "+ "+newLine)
		case newLine == "":
			lines = append(lines, "- "+oldLine)
		default:
			lines = append(lines, "- "+oldLine, "+ "+newLine)
		}
	}
	return strings.Join(lines, "\n")
}

func findEntity(inspection sysmac.Inspection, id string) (sysmac.EntitySummary, error) {
	for _, entity := range inspection.Entities {
		if entity.ID == id {
			return entity, nil
		}
	}
	return sysmac.EntitySummary{}, fmt.Errorf("Sysmac entity %q not found", id)
}
