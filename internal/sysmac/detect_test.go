package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectNativeSolutionDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "demo.oem"), []byte(`<sysmac schemaVersion="1"/>`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ProjectKindSolutionDirectory {
		t.Fatalf("kind = %q, want %q", got.Kind, ProjectKindSolutionDirectory)
	}
}

func TestDetectContainerByExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.smc2")
	if err := os.WriteFile(path, []byte("not-a-zip-yet"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Detect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != ProjectKindContainer {
		t.Fatalf("kind = %q, want %q", got.Kind, ProjectKindContainer)
	}
}

func TestDetectRejectsUnsupportedPath(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "demo.txt"))
	if err == nil {
		t.Fatal("Detect() succeeded for unsupported path")
	}
}
