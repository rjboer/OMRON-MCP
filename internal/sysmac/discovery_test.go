package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProjectsFindsImmediateChildProjects(t *testing.T) {
	root := t.TempDir()
	writeDiscoveryOEM(t, filepath.Join(root, "project-a", "project-a.oem"), "Alpha", "alpha-id")
	writeDiscoveryOEM(t, filepath.Join(root, "project-b", "project-b.oem"), "Beta", "beta-id")

	candidates, err := DiscoverProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(candidates))
	}
	if candidates[0].Name != "Alpha" || candidates[1].Name != "Beta" {
		t.Fatalf("candidate names = %#v", candidates)
	}
	if candidates[0].Status != ProjectStatusValid || candidates[1].Status != ProjectStatusValid {
		t.Fatalf("candidate statuses = %#v", candidates)
	}
}

func TestDiscoverProjectsKeepsInvalidSiblingCandidate(t *testing.T) {
	root := t.TempDir()
	writeDiscoveryOEM(t, filepath.Join(root, "valid", "valid.oem"), "Valid", "valid-id")
	if err := os.Mkdir(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}

	candidates, err := DiscoverProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(candidates))
	}
	if candidates[0].Status != ProjectStatusIncomplete || candidates[1].Status != ProjectStatusValid {
		t.Fatalf("candidate statuses = %#v", candidates)
	}
}

func TestDiscoverProjectsAcceptsSelectedProjectDirectory(t *testing.T) {
	root := t.TempDir()
	writeDiscoveryOEM(t, filepath.Join(root, "project.oem"), "Direct", "direct-id")

	candidates, err := DiscoverProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Folder != root {
		t.Fatalf("candidates = %#v", candidates)
	}
}

func TestDiscoverProjectsUsesFolderFallbacks(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "folder-fallback")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "project.oem"), []byte(`<sysmac><Entity type="Solution" /></sysmac>`), 0o600); err != nil {
		t.Fatal(err)
	}

	candidates, err := DiscoverProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(candidates))
	}
	if candidates[0].Name != "folder-fallback" || candidates[0].ID != "folder-fallback" {
		t.Fatalf("fallback candidate = %#v", candidates[0])
	}
}

func TestDiscoverProjectsRejectsInvalidRoot(t *testing.T) {
	_, err := DiscoverProjects(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected invalid root error")
	}
}

func TestDiscoverProjectsManualSolutionRoot(t *testing.T) {
	root := os.Getenv("OMRON_MCP_DISCOVERY_ROOT")
	if root == "" {
		t.Skip("OMRON_MCP_DISCOVERY_ROOT is not set")
	}

	candidates, err := DiscoverProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	valid := 0
	for _, candidate := range candidates {
		if candidate.Status == ProjectStatusValid {
			valid++
		}
	}
	if valid != 3 {
		t.Fatalf("valid candidates = %d, all candidates = %#v", valid, candidates)
	}
}

func writeDiscoveryOEM(t *testing.T, path, name, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `<sysmac><Entity type="Solution" name="` + name + `" id="` + id + `"><ChildEntities><Entity type="Device" name="Controller" id="controller-id" /></ChildEntities></Entity></sysmac>`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
