package sysmac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverProjectRootFindsManifestFromProjectDirectory(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project-id")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "solution.manifest")
	if err := os.WriteFile(manifest, []byte("manifest"), 0o600); err != nil {
		t.Fatal(err)
	}
	foundRoot, foundManifest, err := discoverProjectRoot(project)
	if err != nil {
		t.Fatal(err)
	}
	if foundRoot != root || foundManifest != manifest {
		t.Fatalf("got %q, %q", foundRoot, foundManifest)
	}
}

func TestEmbeddedHeadlessNexBuilderScriptIsPresent(t *testing.T) {
	if len(embeddedHeadlessNexBuilderScript) == 0 || !strings.Contains(string(embeddedHeadlessNexBuilderScript), "NexBuilder2") {
		t.Fatal("embedded headless NexBuilder script is missing")
	}
}

func TestParseProjectBuildDiagnosticsReadsOnlyPersistedFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.xml")
	data := `<Root><Error Code="E123" Message="Undefined identifier" File="MainProgram" Line="64" Column="19"/></Root>`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got := parseProjectBuildDiagnostics(path)
	if len(got) != 1 {
		t.Fatalf("diagnostics = %#v", got)
	}
	if got[0].Code != "E123" || got[0].Message != "Undefined identifier" || got[0].File != "MainProgram" || got[0].Line != 64 || got[0].Column != 19 {
		t.Fatalf("diagnostic = %#v", got[0])
	}
}

func TestFindCxilInputDirectoryFindsNestedProjectDirectory(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project-id")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "program.cxil2"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := findCxilInputDirectory(root); got != project {
		t.Fatalf("input directory = %q, want %q", got, project)
	}
}
