package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStructuredTextEntityResolvesPayloadByEntityID(t *testing.T) {
	root := t.TempDir()
	oem := `<sysmac schemaVersion="1"><Entity type="Solution" name="Waterjet" id="solution-1"><ChildEntities><Entity type="PouBody" subtype="StructuredText" name="Program Body" id="body-1" /></ChildEntities></Entity></sysmac>`
	if err := os.WriteFile(filepath.Join(root, "project.oem"), []byte(oem), 0600); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(root, "body-1.xml")
	if err := os.WriteFile(payload, []byte(`<StructuredTextModel><Text>Old</Text></StructuredTextModel>`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteStructuredTextEntity(root, "body-1", "New", filepath.Join(root, "protected")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStructuredText(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got != "New" {
		t.Fatalf("source = %q", got)
	}
}
