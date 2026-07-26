package sysmac

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Detect(path string) (ProjectLocation, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ProjectLocation{}, fmt.Errorf("stat project path: %w", err)
	}
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return ProjectLocation{}, fmt.Errorf("read project directory: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".oem") {
				return ProjectLocation{Path: path, Kind: ProjectKindSolutionDirectory}, nil
			}
		}
		return ProjectLocation{}, errors.New("directory does not contain a Sysmac .oem project index")
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".smc2", ".csm2", ".smc", ".csm":
		return ProjectLocation{Path: path, Kind: ProjectKindContainer}, nil
	default:
		return ProjectLocation{}, fmt.Errorf("unsupported Sysmac project path %q", path)
	}
}
