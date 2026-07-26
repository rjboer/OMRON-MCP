package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func newMainTabs(discovery, explorer, validation, gitlab, mcpConnection fyne.CanvasObject) *container.AppTabs {
	return container.NewAppTabs(
		container.NewTabItem("Project Discovery", discovery),
		container.NewTabItem("Project Explorer", explorer),
		container.NewTabItem("Validation / Build", validation),
		container.NewTabItem("GitLab", gitlab),
		container.NewTabItem("MCP Connection", mcpConnection),
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
	modified := "unknown"
	if !candidate.Modified.IsZero() {
		modified = candidate.Modified.Local().Format("2006-01-02 15:04")
	}
	text := fmt.Sprintf("%s\nID: %s\nFolder: %s\nModified: %s\nStatus: %s", candidate.Name, candidate.ID, candidate.Folder, modified, candidate.Status)
	if candidate.Error != "" {
		text += "\nError: " + candidate.Error
	}
	return text
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
