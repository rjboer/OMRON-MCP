package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectSolutionReturnsProjectSummary(t *testing.T) {
	root := t.TempDir()
	oem := `<sysmac schemaVersion="1"><Entity type="Solution" name="Waterjet" id="solution-1" trackingId="track-solution"><ChildEntities><Entity type="Device" subtype="NJ501" name="Controller" id="controller-1" trackingId="track-controller" /></ChildEntities></Entity></sysmac>`
	if err := os.WriteFile(filepath.Join(root, "project.oem"), []byte(oem), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Location.Kind != ProjectKindSolutionDirectory || got.ProjectName != "Waterjet" || len(got.Entities) != 2 {
		t.Fatalf("inspection = %#v", got)
	}
}
