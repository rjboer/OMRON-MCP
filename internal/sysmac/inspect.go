package sysmac

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Inspection struct {
	Location    ProjectLocation `json:"location"`
	ProjectName string          `json:"projectName,omitempty"`
	Entities    []EntitySummary `json:"entities,omitempty"`
	Entries     []string        `json:"containerEntries,omitempty"`
}

type EntitySummary struct {
	Type       string `json:"type"`
	Subtype    string `json:"subtype,omitempty"`
	Name       string `json:"name,omitempty"`
	ID         string `json:"id"`
	TrackingID string `json:"trackingId,omitempty"`
}

func Inspect(path string) (Inspection, error) {
	location, err := Detect(path)
	if err != nil {
		return Inspection{}, err
	}
	inspection := Inspection{Location: location}
	if location.Kind == ProjectKindContainer {
		container, err := OpenContainer(path)
		if err != nil {
			return Inspection{}, err
		}
		defer container.Close()
		inspection.Entries = container.Entries()
		sort.Strings(inspection.Entries)
		return inspection, nil
	}

	oemPath, err := findOEM(path)
	if err != nil {
		return Inspection{}, err
	}
	project, err := ParseOEM(oemPath)
	if err != nil {
		return Inspection{}, err
	}
	inspection.ProjectName = project.Name
	for _, entity := range project.EntitiesByID {
		inspection.Entities = append(inspection.Entities, EntitySummary{Type: entity.Type, Subtype: entity.Subtype, Name: entity.Name, ID: entity.ID, TrackingID: entity.TrackingID})
	}
	sort.Slice(inspection.Entities, func(i, j int) bool { return inspection.Entities[i].ID < inspection.Entities[j].ID })
	return inspection, nil
}

func findOEM(root string) (string, error) {
	var candidates []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".oem" {
			candidates = append(candidates, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("find OEM index: %w", err)
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no .oem index found below %q", root)
	}
	sort.Strings(candidates)
	return candidates[0], nil
}
