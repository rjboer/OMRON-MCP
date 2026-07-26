package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOEMBuildsEntityIndex(t *testing.T) {
	root := t.TempDir()
	oem := `<sysmac schemaVersion="1"><Entity type="Solution" name="Waterjet" id="solution-1" trackingId="track-solution"><ChildEntities><Entity type="Device" subtype="NJ501" name="new_Controller_0" id="controller-1" trackingId="track-controller"><ChildEntities><Entity type="Program" subtype="StructuredText" name="MainProgram" id="program-1" trackingId="track-program"><ChildEntities><Entity type="PouBody" subtype="StructuredText" name="Program Body" id="body-1" trackingId="track-body" /></ChildEntities></Entity></ChildEntities></Entity></ChildEntities></Entity></sysmac>`
	path := filepath.Join(root, "project.oem")
	if err := os.WriteFile(path, []byte(oem), 0600); err != nil {
		t.Fatal(err)
	}

	project, err := ParseOEM(path)
	if err != nil {
		t.Fatal(err)
	}
	if project.Name != "Waterjet" || project.Root == nil {
		t.Fatalf("project = %#v", project)
	}
	body, ok := project.EntitiesByID["body-1"]
	if !ok || body.Name != "Program Body" || body.TrackingID != "track-body" {
		t.Fatalf("body = %#v", body)
	}
	if len(project.Root.Children) != 1 || project.Root.Children[0].Children[0].Name != "MainProgram" {
		t.Fatalf("unexpected entity tree: %#v", project.Root)
	}
}

func TestOEMEntityCRUDMaintainsParentAndChildren(t *testing.T) {
	project := OEMProject{
		Root:         &OEMEntity{Type: "Solution", Name: "Demo", ID: "root"},
		EntitiesByID: map[string]*OEMEntity{},
	}
	indexOEMEntities(project.Root, project.EntitiesByID)
	created, err := project.CreateEntity(OEMEntity{Type: "Program", Name: "Main", ID: "program"}, "root")
	if err != nil {
		t.Fatal(err)
	}
	if created.ParentID != "root" {
		t.Fatalf("created parent = %q", created.ParentID)
	}
	if err := project.RenameEntity("program", "Renamed"); err != nil {
		t.Fatal(err)
	}
	if err := project.MoveEntity("program", "root"); err != nil {
		t.Fatal(err)
	}
	clone, err := project.CloneEntity("program", "root", "Clone")
	if err != nil {
		t.Fatal(err)
	}
	if clone.ID == "program" || len(project.Root.Children) != 2 {
		t.Fatalf("clone = %#v children = %d", clone, len(project.Root.Children))
	}
	if err := project.DeleteEntity("program"); err != nil {
		t.Fatal(err)
	}
	if _, ok := project.EntitiesByID["program"]; ok {
		t.Fatal("deleted entity remains indexed")
	}
}
