package ui

import (
	"reflect"
	"testing"

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
