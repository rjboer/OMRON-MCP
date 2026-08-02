package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/rjboer/omron-mcp/internal/sysmac"
)

const (
	mcpLogLimit = 5000
	mcpLogRows  = 12
)

func newMainTabs(discovery, explorer, validation, gitlab, mcpConnection fyne.CanvasObject) *container.AppTabs {
	return container.NewAppTabs(
		container.NewTabItem("Project Discovery", compactTabContent(discovery)),
		container.NewTabItem("Project Explorer", compactTabContent(explorer)),
		container.NewTabItem("Validation / Build", compactTabContent(validation)),
		container.NewTabItem("GitLab", compactTabContent(gitlab)),
		container.NewTabItem("MCP Connection", compactTabContent(mcpConnection)),
	)
}

func compactTabPadding() float32 {
	return 8
}

func compactTabContent(content fyne.CanvasObject) fyne.CanvasObject {
	padding := compactTabPadding()
	spacer := func(horizontal, vertical float32) fyne.CanvasObject {
		rectangle := canvas.NewRectangle(color.NRGBA{A: 0})
		rectangle.SetMinSize(fyne.NewSize(horizontal, vertical))
		return rectangle
	}
	return container.NewBorder(
		spacer(0, padding), spacer(0, padding), spacer(padding, 0), spacer(padding, 0), content,
	)
}

func DiscoveryMessage(scanned, scanning bool, count int, selected, pathValid bool) string {
	switch {
	case scanning:
		return "Scanning for Sysmac projects..."
	case !pathValid:
		return "The selected path is invalid or unreadable."
	case !scanned:
		return "Select a folder and click Scan Folder."
	case count == 0:
		return "No valid Sysmac projects were found in this folder."
	case !selected:
		return fmt.Sprintf("%d Sysmac projects found. Select a project to continue.", count)
	default:
		return "Project selected. Open it to continue."
	}
}

func discoveryButtonState(scanned, scanning, opened bool) (scan, open widget.Importance, openDisabled bool) {
	if scanning || opened {
		return widget.LowImportance, widget.LowImportance, true
	}
	if scanned {
		return widget.LowImportance, widget.HighImportance, false
	}
	return widget.HighImportance, widget.LowImportance, true
}

func InitialDiscoveryPath(saved, fallback string) string {
	if strings.TrimSpace(saved) != "" {
		return saved
	}
	return fallback
}

func projectCandidateLabel(candidate sysmac.ProjectCandidate) string {
	text := strings.Join(projectCandidateLines(candidate), "\n")
	if candidate.Error != "" {
		text += "\nError: " + candidate.Error
	}
	return text
}

func projectDetailsText(candidate sysmac.ProjectCandidate) string {
	modified := "unknown"
	if !candidate.Modified.IsZero() {
		modified = candidate.Modified.Local().Format("2006-01-02 15:04")
	}
	details := fmt.Sprintf("Name: %s\nID: %s\nFolder: %s\nModified: %s\nStatus: %s", candidate.Name, candidate.ID, candidate.Folder, modified, candidate.Status)
	if candidate.Error != "" {
		details += "\nError: " + candidate.Error
	}
	return details
}

func projectCandidateLines(candidate sysmac.ProjectCandidate) []string {
	modified := "unknown"
	if !candidate.Modified.IsZero() {
		modified = candidate.Modified.Local().Format("2006-01-02 15:04")
	}
	return []string{
		candidate.Name,
		fmt.Sprintf("ID: %s · Modified: %s · Status: %s", candidate.ID, modified, candidate.Status),
		"Folder: " + candidate.Folder,
	}
}

type projectCandidateRow struct {
	*widget.Card
	title            *widget.Label
	metadata         *widget.Label
	folder           *widget.Label
	status           *widget.Label
	statusBackground *canvas.Rectangle
	statusBadge      *fyne.Container
}

func newProjectCandidateRow() *projectCandidateRow {
	title := widget.NewLabel("")
	title.TextStyle.Bold = true
	metadata := widget.NewLabel("")
	folder := widget.NewLabel("")
	for _, label := range []*widget.Label{title, metadata, folder} {
		label.Wrapping = fyne.TextWrapOff
		label.Truncation = fyne.TextTruncateEllipsis
	}
	status := widget.NewLabel("")
	status.TextStyle = fyne.TextStyle{Bold: true}
	status.Alignment = fyne.TextAlignCenter
	status.Wrapping = fyne.TextWrapOff
	statusBackground := canvas.NewRectangle(color.NRGBA{R: 0x2A, G: 0x32, B: 0x3D, A: 0xFF})
	statusBackground.CornerRadius = 8
	statusBadge := container.NewStack(statusBackground, status)
	statusBackground.SetMinSize(fyne.NewSize(72, 24))
	content := container.NewVBox(title, metadata, folder)
	row := container.NewBorder(nil, nil, nil, container.NewPadded(statusBadge), container.NewPadded(content))
	return &projectCandidateRow{
		Card:             widget.NewCard("", "", row),
		title:            title,
		metadata:         metadata,
		folder:           folder,
		status:           status,
		statusBackground: statusBackground,
		statusBadge:      statusBadge,
	}
}

func setProjectCandidateRow(obj fyne.CanvasObject, candidate sysmac.ProjectCandidate) {
	row, ok := obj.(*projectCandidateRow)
	if !ok || row == nil {
		return
	}
	lines := projectCandidateLines(candidate)
	row.title.SetText(lines[0])
	row.metadata.SetText(lines[1])
	row.folder.SetText(lines[2])
	row.status.SetText(strings.Title(string(candidate.Status)))
	row.statusBackground.FillColor = projectStatusColor(candidate.Status)
	row.statusBackground.Refresh()
}

func projectStatusColor(status sysmac.ProjectStatus) color.Color {
	switch status {
	case sysmac.ProjectStatusValid:
		return color.NRGBA{R: 0x17, G: 0x5C, B: 0x3A, A: 0xFF}
	case sysmac.ProjectStatusIncomplete:
		return color.NRGBA{R: 0x7A, G: 0x5A, B: 0x14, A: 0xFF}
	default:
		return color.NRGBA{R: 0x7A, G: 0x2E, B: 0x35, A: 0xFF}
	}
}

type entityRow struct {
	*widget.Card
	name           *widget.Label
	classification *widget.Label
	id             *canvas.Text
	background     *canvas.Rectangle
}

func entityRowLines(entity sysmac.EntitySummary) [3]string {
	classification := entity.Type
	if entity.Subtype != "" {
		classification += " · " + entity.Subtype
	}
	return [3]string{entity.Name, classification, "ID: " + entity.ID}
}

func newEntityRow() *entityRow {
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	name.Wrapping = fyne.TextWrapOff
	name.Truncation = fyne.TextTruncateEllipsis
	classification := widget.NewLabel("")
	classification.Wrapping = fyne.TextWrapOff
	classification.Truncation = fyne.TextTruncateEllipsis
	id := canvas.NewText("", color.NRGBA{R: 0x88, G: 0x8F, B: 0x9B, A: 0xFF})
	id.TextSize = theme.TextSize() - 1
	id.TextStyle = fyne.TextStyle{Monospace: true}
	id.Alignment = fyne.TextAlignTrailing
	background := canvas.NewRectangle(color.NRGBA{R: 0x1B, G: 0x20, B: 0x28, A: 0xFF})
	content := container.NewBorder(nil, nil, nil, container.NewPadded(id), container.NewVBox(name, classification))
	body := container.NewStack(background, container.NewPadded(content))
	return &entityRow{
		Card:           widget.NewCard("", "", body),
		name:           name,
		classification: classification,
		id:             id,
		background:     background,
	}
}

func setEntityRow(obj fyne.CanvasObject, entity sysmac.EntitySummary, index int) {
	row, ok := obj.(*entityRow)
	if !ok || row == nil {
		return
	}
	lines := entityRowLines(entity)
	row.name.SetText(lines[0])
	row.classification.SetText(lines[1])
	row.id.Text = lines[2]
	row.background.FillColor = color.NRGBA{R: 0x1B, G: 0x20, B: 0x28, A: 0xFF}
	if index%2 == 1 {
		row.background.FillColor = color.NRGBA{R: 0x20, G: 0x25, B: 0x2E, A: 0xFF}
	}
	row.id.Refresh()
	row.background.Refresh()
}

type entityDetailField struct {
	Label string
	Value string
}

func entityDetailsFields(entity sysmac.EntitySummary) []entityDetailField {
	fields := []entityDetailField{{Label: "Name", Value: entity.Name}, {Label: "Type", Value: entity.Type}}
	if entity.Subtype != "" {
		fields = append(fields, entityDetailField{Label: "Subtype", Value: entity.Subtype})
	}
	return fields
}

func entityMetadataText(entity sysmac.EntitySummary) string {
	return fmt.Sprintf("ID: %s\nTracking ID: %s", entity.ID, entity.TrackingID)
}

func newEntityDetailsForm(entity sysmac.EntitySummary) *widget.Form {
	form := widget.NewForm()
	for _, field := range entityDetailsFields(entity) {
		value := widget.NewLabel(field.Value)
		value.TextStyle = fyne.TextStyle{Bold: true}
		value.Wrapping = fyne.TextWrapOff
		value.Truncation = fyne.TextTruncateEllipsis
		form.Append(field.Label, value)
	}
	return form
}

func appendLogLine(lines []string, line string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	lines = append(lines, line)
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines
}

type dependencyItem struct {
	Name     string
	Path     string
	Complete bool
}

func dependencyItems(dependencies []sysmac.SysmacDependencies) []dependencyItem {
	var items []dependencyItem
	for _, dependency := range dependencies {
		for _, item := range []struct {
			name string
			path string
		}{
			{"sysmac studio", dependency.SysmacStudio},
			{"nexcc", dependency.NexCC},
			{"nexbuilder2", dependency.NexBuilder2},
			{"nexprogramming", dependency.NexProgramming},
			{"gcc", dependency.GCC},
		} {
			if item.path == "" {
				continue
			}
			items = append(items, dependencyItem{Name: item.name, Path: item.path, Complete: dependency.Complete})
		}
	}
	return items
}

type dependencyRow struct {
	*widget.Card
	name            *widget.Label
	path            *widget.Label
	badge           *widget.Label
	badgeBackground *canvas.Rectangle
	background      *canvas.Rectangle
}

func newDependencyRow() *dependencyRow {
	name := widget.NewLabel("")
	name.TextStyle.Bold = true
	path := widget.NewLabel("")
	path.TextStyle = fyne.TextStyle{Monospace: true}
	path.Wrapping = fyne.TextWrapOff
	path.Truncation = fyne.TextTruncateEllipsis
	badge := widget.NewLabel("")
	badge.Alignment = fyne.TextAlignCenter
	badge.TextStyle.Bold = true
	badgeBackground := canvas.NewRectangle(color.NRGBA{R: 0x17, G: 0x5C, B: 0x3A, A: 0xFF})
	badgeBackground.SetMinSize(fyne.NewSize(64, 22))
	badgeStack := container.NewStack(badgeBackground, badge)
	content := container.NewBorder(nil, nil, nil, container.NewPadded(badgeStack), container.NewVBox(name, path))
	background := canvas.NewRectangle(color.NRGBA{R: 0x17, G: 0x1C, B: 0x23, A: 0xFF})
	return &dependencyRow{Card: widget.NewCard("", "", container.NewStack(background, container.NewPadded(content))), name: name, path: path, badge: badge, badgeBackground: badgeBackground, background: background}
}

func setDependencyRow(obj fyne.CanvasObject, item dependencyItem) {
	row, ok := obj.(*dependencyRow)
	if !ok || row == nil {
		return
	}
	row.name.SetText(item.Name)
	row.path.SetText(item.Path)
	if item.Complete {
		row.badge.SetText("Ready")
		row.badgeBackground.FillColor = color.NRGBA{R: 0x17, G: 0x5C, B: 0x3A, A: 0xFF}
		row.background.FillColor = color.NRGBA{R: 0x17, G: 0x1C, B: 0x23, A: 0xFF}
	} else {
		row.badge.SetText("Missing")
		row.badgeBackground.FillColor = color.NRGBA{R: 0x7A, G: 0x2E, B: 0x35, A: 0xFF}
		row.background.FillColor = color.NRGBA{R: 0x2B, G: 0x20, B: 0x23, A: 0xFF}
	}
	row.badgeBackground.Refresh()
	row.background.Refresh()
}

func controlWithHeight(control fyne.CanvasObject, height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.NRGBA{A: 0})
	spacer.SetMinSize(fyne.NewSize(0, height))
	return container.NewStack(spacer, control)
}

func FilteredEntities(entities []sysmac.EntitySummary, query string) []sysmac.EntitySummary {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entities
	}
	filtered := make([]sysmac.EntitySummary, 0, len(entities))
	for _, entity := range entities {
		fields := []string{entity.Type, entity.Subtype, entity.Name, entity.ID, entity.TrackingID}
		matched := false
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), query) {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, entity)
		}
	}
	return filtered
}

func entityLabel(entity sysmac.EntitySummary) string {
	parts := []string{entity.Type}
	if entity.Subtype != "" {
		parts = append(parts, entity.Subtype)
	}
	if entity.Name != "" {
		parts = append(parts, entity.Name)
	}
	return fmt.Sprintf("%s [%s]", strings.Join(parts, " / "), entity.ID)
}
