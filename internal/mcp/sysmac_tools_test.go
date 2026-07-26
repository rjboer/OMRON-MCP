package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func testSolution(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	oem := `<sysmac><Entity type="Solution" name="Demo" id="solution"><ChildEntities><Entity type="PouBody" subtype="StructuredText" name="Body" id="body" /></ChildEntities></Entity></sysmac>`
	if err := os.WriteFile(filepath.Join(root, "project.oem"), []byte(oem), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "body.xml"), []byte(`<StructuredTextModel><Text>Old</Text></StructuredTextModel>`), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func callArgs(t *testing.T, name string, args map[string]any) (map[string]any, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"name": name, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	return callSysmacTool(raw)
}

func TestSysmacToolsExposeNexCCBuild(t *testing.T) {
	for _, candidate := range sysmacTools() {
		if candidate.Name == "sysmac_nexcc_build" {
			return
		}
	}
	t.Fatal("sysmac_nexcc_build tool is not registered")
}

func TestNexCCBuildToolManualInstalledSysmac(t *testing.T) {
	directory := os.Getenv("OMRON_MCP_NEX_BUILD_DIR")
	if directory == "" {
		t.Skip("OMRON_MCP_NEX_BUILD_DIR is not set")
	}
	result, err := callArgs(t, "sysmac_nexcc_build", map[string]any{"build_directory": directory})
	if err != nil {
		t.Fatal(err)
	}
	content := result["structuredContent"].(map[string]any)
	t.Logf("NexCC tool result: %#v", content)
	if content["status"] != "succeeded" || content["failed"] != 0 {
		t.Fatalf("build result = %#v", content)
	}
}

func TestStructuredTextReadAndConfirmedUpdate(t *testing.T) {
	root := testSolution(t)
	read, err := callArgs(t, "sysmac_structured_text_read", map[string]any{"project_path": root, "entity_id": "body"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read["content"].([]map[string]string)[0]["text"], "Old") {
		t.Fatalf("read = %#v", read)
	}
	if _, err := callArgs(t, "sysmac_structured_text_update", map[string]any{"project_path": root, "entity_id": "body", "source_text": "New", "confirm_direct_write": false}); err == nil {
		t.Fatal("unconfirmed update succeeded")
	}
	updated, err := callArgs(t, "sysmac_structured_text_update", map[string]any{"project_path": root, "entity_id": "body", "source_text": "New", "confirm_direct_write": true})
	if err != nil {
		t.Fatal(err)
	}
	if updated["structuredContent"].(map[string]any)["status"] != "applied" {
		t.Fatalf("update = %#v", updated)
	}
	if _, err := os.Stat(filepath.Join(root, "body.xml.backup")); err != nil {
		t.Fatal(err)
	}
}

func TestStructuredTextPreviewIsReadOnly(t *testing.T) {
	root := testSolution(t)
	preview, err := callArgs(t, "sysmac_structured_text_preview", map[string]any{
		"project_path": root,
		"entity_id":    "body",
		"source_text":  "New",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := preview["structuredContent"].(map[string]any)
	if content["status"] != "preview" {
		t.Fatalf("status = %#v", content["status"])
	}
	if content["changed"] != true {
		t.Fatalf("changed = %#v", content["changed"])
	}
	if content["current_source_text"] != "Old" {
		t.Fatalf("current source = %#v", content["current_source_text"])
	}
	if content["proposed_source_text"] != "New" {
		t.Fatalf("proposed source = %#v", content["proposed_source_text"])
	}
	if _, err := os.Stat(filepath.Join(root, "body.xml.backup")); !os.IsNotExist(err) {
		t.Fatalf("preview created backup: %v", err)
	}
}

func TestProjectSnapshotRequiresConfirmationAndCopiesSource(t *testing.T) {
	source := testSolution(t)
	destination := filepath.Join(t.TempDir(), "snapshot")
	if _, err := callArgs(t, "sysmac_project_snapshot", map[string]any{
		"source_path":      source,
		"destination_path": destination,
		"confirm_snapshot": false,
	}); err == nil {
		t.Fatal("unconfirmed snapshot succeeded")
	}

	result, err := callArgs(t, "sysmac_project_snapshot", map[string]any{
		"source_path":      source,
		"destination_path": destination,
		"confirm_snapshot": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["structuredContent"].(map[string]any)["status"] != "copied" {
		t.Fatalf("snapshot = %#v", result)
	}
	text, err := sysmac.ReadStructuredTextEntity(destination, "body")
	if err != nil || text != "Old" {
		t.Fatalf("snapshot read = %q, err=%v", text, err)
	}
	original, err := sysmac.ReadStructuredTextEntity(source, "body")
	if err != nil || original != "Old" {
		t.Fatalf("source read = %q, err=%v", original, err)
	}
}

func TestSLWDReadAndConfirmedWrite(t *testing.T) {
	root := testSolution(t)
	path := filepath.Join(root, "variables.slwd")
	original := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++X=Unknown PreserveMe\n++D=BOOL N=Start G=VAR\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	read, err := callArgs(t, "sysmac_slwd_read", map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	if len(read["structuredContent"].(map[string]any)["variables"].([]sysmac.SLWDVariable)) != 1 {
		t.Fatalf("read = %#v", read)
	}
	updated, err := callArgs(t, "sysmac_slwd_write", map[string]any{
		"path":                 path,
		"variables":            []sysmac.SLWDVariable{{Name: "Run", Type: "BOOL", Group: "VAR"}},
		"delete_names":         []string{"Start"},
		"confirm_direct_write": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated["structuredContent"].(map[string]any)["status"] != "applied" {
		t.Fatalf("update = %#v", updated)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "+X=Unknown PreserveMe") || !strings.Contains(string(data), "N=Run") {
		t.Fatalf("SLWD update lost data: %s", data)
	}
}

func TestSLWDWriteAndReadPreservesCommentWithSpaces(t *testing.T) {
	root := testSolution(t)
	path := filepath.Join(root, "variables.slwd")
	original := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++D=BOOL N=Start G=VAR\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	want := `Written through MCP integration test`
	if _, err := callArgs(t, "sysmac_slwd_write", map[string]any{
		"path":                 path,
		"variables":            []sysmac.SLWDVariable{{Name: "Probe", Type: "BOOL", Group: "VAR", Comment: want}},
		"confirm_direct_write": true,
	}); err != nil {
		t.Fatal(err)
	}
	read, err := callArgs(t, "sysmac_slwd_read", map[string]any{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	for _, variable := range read["structuredContent"].(map[string]any)["variables"].([]sysmac.SLWDVariable) {
		if variable.Name == "Probe" && variable.Comment == want {
			return
		}
	}
	t.Fatalf("MCP read did not preserve comment: %#v", read)
}

func TestProjectTransactionLifecycle(t *testing.T) {
	root := testSolution(t)
	started, err := callArgs(t, "sysmac_transaction_start", map[string]any{"project_path": root})
	if err != nil {
		t.Fatal(err)
	}
	id := started["structuredContent"].(map[string]any)["transaction_id"].(string)
	if _, err := callArgs(t, "sysmac_transaction_stage_file", map[string]any{"transaction_id": id, "relative_path": "body.xml", "content": "<StructuredTextModel><Text>Staged</Text></StructuredTextModel>"}); err != nil {
		t.Fatal(err)
	}
	preview, err := callArgs(t, "sysmac_transaction_preview", map[string]any{"transaction_id": id})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview["structuredContent"].(map[string]any)["changes"].([]sysmac.FileChange)) != 1 {
		t.Fatalf("preview = %#v", preview)
	}
	if _, err := callArgs(t, "sysmac_transaction_commit", map[string]any{"transaction_id": id, "confirm_commit": true}); err != nil {
		t.Fatal(err)
	}
	text, err := sysmac.ReadStructuredText(filepath.Join(root, "body.xml"))
	if err != nil || text != "Staged" {
		t.Fatalf("committed text = %q, err=%v", text, err)
	}
}

func TestProjectTransactionDeleteLifecycle(t *testing.T) {
	root := testSolution(t)
	path := filepath.Join(root, "body.xml")
	started, err := callArgs(t, "sysmac_transaction_start", map[string]any{"project_path": root})
	if err != nil {
		t.Fatal(err)
	}
	id := started["structuredContent"].(map[string]any)["transaction_id"].(string)
	if _, err := callArgs(t, "sysmac_transaction_delete_file", map[string]any{"transaction_id": id, "relative_path": "body.xml"}); err != nil {
		t.Fatal(err)
	}
	preview, err := callArgs(t, "sysmac_transaction_preview", map[string]any{"transaction_id": id})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview["structuredContent"].(map[string]any)["changes"].([]sysmac.FileChange)) != 1 {
		t.Fatalf("delete preview = %#v", preview)
	}
	if _, err := callArgs(t, "sysmac_transaction_commit", map[string]any{"transaction_id": id, "confirm_commit": true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("committed delete left body.xml: %v", err)
	}
}
