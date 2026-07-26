package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

// TestManualProjectMCPEdit runs only when explicitly pointed at a local test
// project. It copies the source first, then exercises the real MCP mutation
// path against the temporary copy.
func TestManualProjectMCPEdit(t *testing.T) {
	source := os.Getenv("OMRON_MCP_SOURCE_PROJECT")
	if source == "" {
		t.Skip("OMRON_MCP_SOURCE_PROJECT is not set")
	}

	destination := os.Getenv("OMRON_MCP_DESTINATION_PROJECT")
	if destination == "" {
		destination = filepath.Join(t.TempDir(), "snapshot")
	} else if _, err := os.Stat(destination); err == nil {
		t.Fatalf("destination already exists: %s", destination)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat destination: %v", err)
	}
	snapshot, err := callArgs(t, "sysmac_project_snapshot", map[string]any{
		"source_path":      source,
		"destination_path": destination,
		"confirm_snapshot": true,
	})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if snapshot["structuredContent"].(map[string]any)["status"] != "copied" {
		t.Fatalf("snapshot response = %#v", snapshot)
	}

	inspection, err := sysmac.Inspect(destination)
	if err != nil {
		t.Fatalf("inspect snapshot failed: %v", err)
	}
	var entityID string
	for _, entity := range inspection.Entities {
		if entity.Type == "PouBody" && entity.Subtype == "StructuredText" {
			entityID = entity.ID
			break
		}
	}
	if entityID == "" {
		t.Fatalf("snapshot contains no Structured Text POU body; entities=%d", len(inspection.Entities))
	}

	read, err := callArgs(t, "sysmac_structured_text_read", map[string]any{
		"project_path": destination,
		"entity_id":    entityID,
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	current := read["structuredContent"].(map[string]any)["source_text"].(string)
	proposed := current
	if !strings.HasSuffix(proposed, "\n") {
		proposed += "\r\n"
	}
	proposed += "(* OMRON MCP integration test *)"

	preview, err := callArgs(t, "sysmac_structured_text_preview", map[string]any{
		"project_path": destination,
		"entity_id":    entityID,
		"source_text":  proposed,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	previewContent := preview["structuredContent"].(map[string]any)
	if previewContent["status"] != "preview" || previewContent["changed"] != true {
		t.Fatalf("preview response = %#v", previewContent)
	}

	updated, err := callArgs(t, "sysmac_structured_text_update", map[string]any{
		"project_path":         destination,
		"entity_id":            entityID,
		"source_text":          proposed,
		"confirm_direct_write": true,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	updatedContent := updated["structuredContent"].(map[string]any)
	if updatedContent["status"] != "applied" {
		t.Fatalf("update response = %#v", updatedContent)
	}

	verified, err := sysmac.ReadStructuredTextEntity(destination, entityID)
	if err != nil {
		t.Fatalf("verification read failed: %v", err)
	}
	if verified != proposed {
		t.Fatalf("verified source differs from proposed source")
	}
	if _, err := os.Stat(updatedContent["backup_path"].(string)); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	t.Logf("MCP edit passed: project=%s entity=%s source=%s snapshot=%s", inspection.ProjectName, entityID, source, destination)
}
