package sysmac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteStructuredTextAtomicCreatesBackup(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "body.xml")
	original := `<StructuredTextModel xmlns="http://schemas.datacontract.org/2004/07/Omron.Cxap.Modules.StructuredText.Core"><Text>Old</Text></StructuredTextModel>`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteStructuredTextAtomic(path, "New\r\nLine", filepath.Join(root, "protected")); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".backup")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != original {
		t.Fatalf("backup = %q, want original", backup)
	}
	got, err := ReadStructuredText(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "New\r\nLine" {
		t.Fatalf("source = %q", got)
	}
}

func TestWriteStructuredTextAtomicPreservesSysmacNamespace(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "body.xml")
	original := `<StructuredTextModel xmlns="http://schemas.datacontract.org/2004/07/Omron.Cxap.Modules.StructuredText.Core"><Text>Old</Text></StructuredTextModel>`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteStructuredTextAtomic(path, "New", filepath.Join(root, "protected")); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `xmlns="http://schemas.datacontract.org/2004/07/Omron.Cxap.Modules.StructuredText.Core"`) {
		t.Fatalf("updated XML lost Sysmac namespace: %s", updated)
	}
}

func TestWriteStructuredTextAtomicRejectsProtectedRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "body.xml")
	if err := os.WriteFile(path, []byte(`<StructuredTextModel><Text>Old</Text></StructuredTextModel>`), 0600); err != nil {
		t.Fatal(err)
	}

	err := WriteStructuredTextAtomic(path, "New", root)
	if err == nil || !strings.Contains(err.Error(), "protected") {
		t.Fatalf("error = %v, want protected-root error", err)
	}
}
