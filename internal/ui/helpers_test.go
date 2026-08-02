package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
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
	if row.Content == nil {
		t.Fatal("project row card has no content")
	}
	paddedRow, ok := row.Content.(*fyne.Container)
	if !ok || len(paddedRow.Objects) != 2 {
		t.Fatalf("row content has unexpected shape: %#v", row.Content)
	}
	content, ok := paddedRow.Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
	if !ok || len(content.Objects) != 3 {
		t.Fatalf("row text content has unexpected shape: %#v", content)
	}
	for i, object := range content.Objects {
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

func TestProjectCandidateStatusBadgeHasFixedMinimumSize(t *testing.T) {
	row := newProjectCandidateRow()
	if got := row.statusBackground.MinSize(); got.Width < 72 || got.Height < 24 {
		t.Fatalf("status badge minimum size = %v, want at least 72x24", got)
	}
}

func TestEntityRowSeparatesNameClassificationAndSystemID(t *testing.T) {
	entity := sysmac.EntitySummary{
		Name: "CPU Rack", Type: "NexPC21Configuration", Subtype: "CpuRackProxy", ID: "413b92e0-be71-4f4f-a8c1-123456789abc",
	}
	lines := entityRowLines(entity)
	if lines[0] != "CPU Rack" || lines[1] != "NexPC21Configuration · CpuRackProxy" || lines[2] != "ID: "+entity.ID {
		t.Fatalf("entity row lines = %#v", lines)
	}
}

func TestEntityDetailsFieldsKeepSystemMetadataSeparate(t *testing.T) {
	entity := sysmac.EntitySummary{Name: "Programs", Type: "Group", Subtype: "IecPrograms", ID: "entity-1", TrackingID: "tracking-1"}
	fields := entityDetailsFields(entity)
	if len(fields) != 3 || fields[0].Label != "Name" || fields[0].Value != "Programs" || fields[1].Value != "Group" || fields[2].Value != "IecPrograms" {
		t.Fatalf("entity detail fields = %#v", fields)
	}
	metadata := entityMetadataText(entity)
	if !strings.Contains(metadata, "ID: entity-1") || !strings.Contains(metadata, "Tracking ID: tracking-1") {
		t.Fatalf("entity metadata = %q", metadata)
	}
}

func TestEntityDetailsFormUsesOneItemPerVisibleField(t *testing.T) {
	testApp := app.New()
	defer testApp.Quit()
	entity := sysmac.EntitySummary{Name: "Programs", Type: "Group", Subtype: "IecPrograms"}
	form := newEntityDetailsForm(entity)
	if len(form.Items) != 3 {
		t.Fatalf("form item count = %d, want 3", len(form.Items))
	}
	for i, want := range []string{"Name", "Type", "Subtype"} {
		if form.Items[i].Text != want {
			t.Errorf("form item %d label = %q, want %q", i, form.Items[i].Text, want)
		}
	}
}

func TestAppendLogLineKeepsNewestEntriesWithinLimit(t *testing.T) {
	lines := []string{"one", "two", "three"}
	lines = appendLogLine(lines, "four", 3)
	if !reflect.DeepEqual(lines, []string{"two", "three", "four"}) {
		t.Fatalf("log lines = %#v", lines)
	}
}

func TestAppendLogLineKeepsFiveThousandNewestEntries(t *testing.T) {
	if mcpLogLimit != 5000 {
		t.Fatalf("MCP log limit = %d, want 5000", mcpLogLimit)
	}
	lines := make([]string, 5000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%d", i)
	}
	lines = appendLogLine(lines, "line-5000", 5000)
	if len(lines) != 5000 || lines[0] != "line-1" || lines[len(lines)-1] != "line-5000" {
		t.Fatalf("rolling log boundaries = len %d, first %q, last %q", len(lines), lines[0], lines[len(lines)-1])
	}
}

func TestMCPLogUsesConsoleHeight(t *testing.T) {
	if mcpLogRows != 12 {
		t.Fatalf("MCP log rows = %d, want 12", mcpLogRows)
	}
}

func TestBuildOutputUsesConsoleHeight(t *testing.T) {
	if buildOutputRows != 10 {
		t.Fatalf("build output rows = %d, want 10", buildOutputRows)
	}
	if buildOutputHeight != 280 || dependencyListHeight != 240 {
		t.Fatalf("panel heights = build %v, dependencies %v", buildOutputHeight, dependencyListHeight)
	}
}

func TestCompactTabPaddingIsSmallerThanDefault(t *testing.T) {
	if got := compactTabPadding(); got != 8 {
		t.Fatalf("compact tab padding = %v, want 8", got)
	}
}

func TestDiscoveryButtonStatePrioritizesScanBeforeAndOpenAfterScan(t *testing.T) {
	scan, open, openDisabled := discoveryButtonState(false, false, false)
	if scan != widget.HighImportance || open != widget.LowImportance || !openDisabled {
		t.Fatalf("initial button state = %v, %v, disabled=%v", scan, open, openDisabled)
	}
	scan, open, openDisabled = discoveryButtonState(true, false, false)
	if scan != widget.LowImportance || open != widget.HighImportance || openDisabled {
		t.Fatalf("scanned button state = %v, %v, disabled=%v", scan, open, openDisabled)
	}
	scan, open, openDisabled = discoveryButtonState(true, false, true)
	if scan != widget.LowImportance || open != widget.LowImportance || !openDisabled {
		t.Fatalf("opened button state = %v, %v, disabled=%v", scan, open, openDisabled)
	}
}

func TestDependencyItemsFlattenInstallationsIntoScannableRows(t *testing.T) {
	items := dependencyItems([]sysmac.SysmacDependencies{{InstallationRoot: `C:\OMRON\Sysmac`, Complete: true, NexCC: `C:\OMRON\Sysmac\nexcc.exe`, GCC: `C:\gcc.exe`}})
	if len(items) != 2 || items[0].Name != "nexcc" || items[1].Name != "gcc" {
		t.Fatalf("dependency items = %#v", items)
	}
	if !items[0].Complete || items[0].Path == "" {
		t.Fatalf("dependency item = %#v", items[0])
	}
}

func TestSetProjectCandidateRowIgnoresMalformedCard(t *testing.T) {
	card := widget.NewCard("", "", widget.NewLabel("unexpected list item"))

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("setProjectCandidateRow panicked for a malformed list item: %v", recovered)
		}
	}()

	setProjectCandidateRow(card, sysmac.ProjectCandidate{Name: "Waterjet"})
}

func TestWorkbenchThemeUsesLayeredSurfacesAndSubtleDividers(t *testing.T) {
	workbench := newWorkbenchTheme()
	if got := workbench.Size(theme.SizeNameSplitThickness); got != 1 {
		t.Fatalf("split thickness = %v, want 1", got)
	}
	background := workbench.Color(theme.ColorNameBackground, theme.VariantDark)
	panel := workbench.Color(theme.ColorNameInputBackground, theme.VariantDark)
	border := workbench.Color(theme.ColorNameSeparator, theme.VariantDark)
	if reflect.DeepEqual(background, panel) || reflect.DeepEqual(panel, border) {
		t.Fatalf("theme surfaces are not layered: background=%v panel=%v border=%v", background, panel, border)
	}
	if got := workbench.Size(theme.SizeNamePadding); got != 6 {
		t.Fatalf("theme padding = %v, want 6", got)
	}
	if got := workbench.Size(theme.SizeNameInnerPadding); got != 6 {
		t.Fatalf("theme inner padding = %v, want 6", got)
	}
}
