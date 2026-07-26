package sysmac

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ProjectStatus string

const (
	ProjectStatusValid       ProjectStatus = "valid"
	ProjectStatusIncomplete  ProjectStatus = "incomplete"
	ProjectStatusUnsupported ProjectStatus = "unsupported"
	ProjectStatusUnreadable  ProjectStatus = "unreadable"
)

type ProjectCandidate struct {
	Name     string
	ID       string
	Folder   string
	OEMPath  string
	Modified time.Time
	Status   ProjectStatus
	Error    string
}

// DiscoverProjects scans one selected directory and its immediate child
// directories. It never recursively scans sibling levels from the selected
// root, so a Solution directory is not mistaken for one project.
func DiscoverProjects(root string) ([]ProjectCandidate, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat discovery root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("discovery root is not a directory: %q", root)
	}

	if oemPath, ok, err := directOEM(root); err != nil {
		return nil, fmt.Errorf("read discovery root: %w", err)
	} else if ok {
		return []ProjectCandidate{inspectCandidate(root, oemPath, info.ModTime())}, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read discovery root: %w", err)
	}
	candidates := make([]ProjectCandidate, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		folder := filepath.Join(root, entry.Name())
		childInfo, statErr := os.Stat(folder)
		if statErr != nil {
			candidates = append(candidates, ProjectCandidate{
				Name:   entry.Name(),
				ID:     entry.Name(),
				Folder: folder,
				Status: ProjectStatusUnreadable,
				Error:  statErr.Error(),
			})
			continue
		}
		oemPath, ok, oemErr := directOEM(folder)
		if oemErr != nil {
			candidates = append(candidates, ProjectCandidate{
				Name:     entry.Name(),
				ID:       entry.Name(),
				Folder:   folder,
				Modified: childInfo.ModTime(),
				Status:   ProjectStatusUnreadable,
				Error:    oemErr.Error(),
			})
			continue
		}
		if !ok {
			candidates = append(candidates, ProjectCandidate{
				Name:     entry.Name(),
				ID:       entry.Name(),
				Folder:   folder,
				Modified: childInfo.ModTime(),
				Status:   ProjectStatusIncomplete,
				Error:    "no .oem project index found directly in the folder",
			})
			continue
		}
		candidates = append(candidates, inspectCandidate(folder, oemPath, childInfo.ModTime()))
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		leftName := strings.ToLower(candidates[i].Name)
		rightName := strings.ToLower(candidates[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return strings.ToLower(candidates[i].Folder) < strings.ToLower(candidates[j].Folder)
	})
	return candidates, nil
}

func directOEM(folder string) (string, bool, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".oem") {
			return filepath.Join(folder, entry.Name()), true, nil
		}
	}
	return "", false, nil
}

func inspectCandidate(folder, oemPath string, modified time.Time) ProjectCandidate {
	name := filepath.Base(folder)
	candidate := ProjectCandidate{
		Name:     name,
		ID:       name,
		Folder:   folder,
		OEMPath:  oemPath,
		Modified: modified,
		Status:   ProjectStatusIncomplete,
	}
	project, err := ParseOEM(oemPath)
	if err != nil {
		candidate.Error = err.Error()
		return candidate
	}
	if project.Name != "" {
		candidate.Name = project.Name
	}
	if project.Root != nil && project.Root.ID != "" {
		candidate.ID = project.Root.ID
	}
	candidate.Status = ProjectStatusValid
	return candidate
}
