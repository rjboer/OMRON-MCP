package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopySolutionDirectoryAndReadEntity(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "project.oem"), []byte(`<sysmac><Entity type="Solution" name="Demo" id="solution"><ChildEntities><Entity type="PouBody" subtype="StructuredText" name="Body" id="body" /></ChildEntities></Entity></sysmac>`), 0600); err != nil {
		t.Fatal(err)
	}
	original := []byte(`<StructuredTextModel><Text>Old</Text></StructuredTextModel>`)
	payload := filepath.Join(source, "body.xml")
	if err := os.WriteFile(payload, original, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := SHA256File(payload)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "copy")
	if err := CopySolutionDirectory(source, destination); err != nil {
		t.Fatal(err)
	}
	text, err := ReadStructuredTextEntity(destination, "body")
	if err != nil || text != "Old" {
		t.Fatalf("read copied entity = %q, err=%v", text, err)
	}
	if err := WriteStructuredTextEntity(destination, "body", text, ""); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(filepath.Join(destination, "body.xml.backup"))
	if err != nil || string(backup) != string(original) {
		t.Fatalf("backup = %q, err=%v", backup, err)
	}
	after, err := SHA256File(payload)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("source hash changed: before=%s after=%s", before, after)
	}
}

func TestCopySolutionDirectoryRejectsOverlap(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "project.oem"), []byte(`<sysmac/>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CopySolutionDirectory(source, filepath.Join(source, "copy")); err == nil {
		t.Fatal("copy succeeded inside source")
	}
}
