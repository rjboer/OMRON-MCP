package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureHelperCopiesCurrentExecutableWhenMissing(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "omron-mcp.exe")
	helper := filepath.Join(dir, "omron-mcp-updater.exe")
	if err := os.WriteFile(source, []byte("payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := EnsureHelper(source, helper); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(helper)
	if err != nil || string(got) != "payload" {
		t.Fatalf("helper = %q, %v", got, err)
	}
}
