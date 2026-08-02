package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func TestNewMainTabsExposesApprovedFiveTabsInOrder(t *testing.T) {
	tabs := newMainTabs(
		widget.NewLabel("discovery"),
		widget.NewLabel("explorer"),
		widget.NewLabel("validation"),
		widget.NewLabel("gitlab"),
		widget.NewLabel("mcp"),
	)

	want := []string{"Project Discovery", "Project Explorer", "Validation / Build", "GitLab", "MCP Connection"}
	if len(tabs.Items) != len(want) {
		t.Fatalf("tab count = %d, want %d", len(tabs.Items), len(want))
	}
	for i, item := range tabs.Items {
		if item.Text != want[i] {
			t.Errorf("tab %d = %q, want %q", i, item.Text, want[i])
		}
	}
}

func TestFilteredEntitiesMatchesNameTypeSubtypeAndID(t *testing.T) {
	entities := []sysmac.EntitySummary{
		{Type: "Program", Subtype: "StructuredText", Name: "MainProgram", ID: "program-1"},
		{Type: "PouBody", Subtype: "StructuredText", Name: "PressureControl", ID: "body-2"},
	}

	if got := FilteredEntities(entities, "pressure"); !reflect.DeepEqual(got, entities[1:]) {
		t.Fatalf("filter by name = %#v, want %#v", got, entities[1:])
	}
	if got := FilteredEntities(entities, "PROGRAM"); !reflect.DeepEqual(got, entities[:1]) {
		t.Fatalf("filter by type/name = %#v, want %#v", got, entities[:1])
	}
	if got := FilteredEntities(entities, "body-2"); !reflect.DeepEqual(got, entities[1:]) {
		t.Fatalf("filter by ID = %#v, want %#v", got, entities[1:])
	}
}

func TestDiscoveryMessageReturnsExplicitStates(t *testing.T) {
	tests := []struct {
		name      string
		scanned   bool
		scanning  bool
		count     int
		selected  bool
		pathValid bool
		want      string
	}{
		{name: "no scan", pathValid: true, want: "Select a folder and click Scan Folder."},
		{name: "scanning", scanning: true, pathValid: true, want: "Scanning for Sysmac projects..."},
		{name: "invalid path", scanned: true, pathValid: false, want: "The selected path is invalid or unreadable."},
		{name: "none found", scanned: true, pathValid: true, want: "No valid Sysmac projects were found in this folder."},
		{name: "found but unselected", scanned: true, count: 3, pathValid: true, want: "3 Sysmac projects found. Select a project to continue."},
		{name: "selected", scanned: true, count: 1, selected: true, pathValid: true, want: "Project selected. Open it to continue."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DiscoveryMessage(test.scanned, test.scanning, test.count, test.selected, test.pathValid); got != test.want {
				t.Fatalf("DiscoveryMessage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInitialDiscoveryPathPrefersSavedPath(t *testing.T) {
	if got := InitialDiscoveryPath(`C:\saved`, `C:\OMRON\Data\Solution`); got != `C:\saved` {
		t.Fatalf("saved path = %q, want C:\\saved", got)
	}
	if got := InitialDiscoveryPath("", `C:\OMRON\Data\Solution`); got != `C:\OMRON\Data\Solution` {
		t.Fatalf("fallback path = %q", got)
	}
}

func TestProjectDetailsUseOnlyCandidateData(t *testing.T) {
	candidate := sysmac.ProjectCandidate{
		Name: "Waterjet", ID: "project-1", Folder: `C:\OMRON\Data\Solution\project-1`,
		Status: sysmac.ProjectStatusValid,
	}
	details := projectDetailsText(candidate)
	for _, want := range []string{"Name: Waterjet", "ID: project-1", "Folder: C:\\OMRON\\Data\\Solution\\project-1", "Status: valid"} {
		if !strings.Contains(details, want) {
			t.Fatalf("project details %q does not contain %q", details, want)
		}
	}
	for _, absent := range []string{"Entities:", "Duplicates:", "About:"} {
		if strings.Contains(details, absent) {
			t.Fatalf("project details %q contains unsupported field %q", details, absent)
		}
	}
}

func TestProjectCandidateLinesKeepRowsStructured(t *testing.T) {
	candidate := sysmac.ProjectCandidate{
		Name:     "Waterjet",
		ID:       "22994b64-903d-4e2d-ae31-a6fb4e029729",
		Folder:   `C:\OMRON\Data\Solution\22994b64-903d-4e2d-ae31-a6fb4e029729`,
		Modified: time.Date(2026, time.July, 23, 2, 12, 0, 0, time.UTC),
		Status:   sysmac.ProjectStatusValid,
	}

	lines := projectCandidateLines(candidate)
	if len(lines) != 3 {
		t.Fatalf("got %d project row lines, want 3", len(lines))
	}
	if lines[0] != "Waterjet" {
		t.Fatalf("title line = %q", lines[0])
	}
	if !strings.Contains(lines[1], "ID: "+candidate.ID) || !strings.Contains(lines[1], "Status: valid") {
		t.Fatalf("metadata line = %q", lines[1])
	}
	if lines[2] != "Folder: "+candidate.Folder {
		t.Fatalf("folder line = %q", lines[2])
	}
}

func TestProjectCandidateRowUsesSingleLineEllipsis(t *testing.T) {
	row := newProjectCandidateRow()
	if len(row.Objects) != 3 {
		t.Fatalf("row contains %d objects, want 3", len(row.Objects))
	}
	for i, object := range row.Objects {
		label, ok := object.(*widget.Label)
		if !ok {
			t.Fatalf("row object %d has type %T, want *widget.Label", i, object)
		}
		if label.Wrapping != fyne.TextWrapOff {
			t.Errorf("row label %d wrapping = %v, want TextWrapOff", i, label.Wrapping)
		}
		if label.Truncation != fyne.TextTruncateEllipsis {
			t.Errorf("row label %d truncation = %v, want TextTruncateEllipsis", i, label.Truncation)
		}
	}
}
