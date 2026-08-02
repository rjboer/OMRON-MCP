package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func newMainTabs(discovery, explorer, validation, gitlab, mcpConnection fyne.CanvasObject) *container.AppTabs {
	return container.NewAppTabs(
		container.NewTabItem("Project Discovery", container.NewPadded(discovery)),
		container.NewTabItem("Project Explorer", container.NewPadded(explorer)),
		container.NewTabItem("Validation / Build", container.NewPadded(validation)),
		container.NewTabItem("GitLab", container.NewPadded(gitlab)),
		container.NewTabItem("MCP Connection", container.NewPadded(mcpConnection)),
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

func newProjectCandidateRow() *widget.Card {
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
	content := container.NewVBox(title, metadata, folder)
	row := container.NewBorder(nil, nil, nil, container.NewPadded(statusBadge), container.NewPadded(content))
	return widget.NewCard("", "", row)
}

func setProjectCandidateRow(card *widget.Card, candidate sysmac.ProjectCandidate) {
	lines := projectCandidateLines(candidate)
	paddedRow := card.Content.(*fyne.Container)
	row := paddedRow.Objects[0].(*fyne.Container)
	content := row.Objects[0].(*fyne.Container).Objects[0].(*fyne.Container)
	for i, line := range lines {
		content.Objects[i].(*widget.Label).SetText(line)
	}
	badge := row.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
	badge.Objects[1].(*widget.Label).SetText(strings.Title(string(candidate.Status)))
	background := badge.Objects[0].(*canvas.Rectangle)
	background.FillColor = projectStatusColor(candidate.Status)
	background.Refresh()
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
