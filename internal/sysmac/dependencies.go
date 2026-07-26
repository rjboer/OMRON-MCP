package sysmac

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type SysmacDependencies struct {
	InstallationRoot string   `json:"installation_root"`
	SysmacStudio     string   `json:"sysmac_studio"`
	NexCC            string   `json:"nexcc"`
	NexBuilder2      string   `json:"nex_builder2"`
	NexProgramming   string   `json:"nex_programming"`
	GCC              string   `json:"gcc"`
	Missing          []string `json:"missing,omitempty"`
	Complete         bool     `json:"complete"`
}

func DiscoverSysmacDependencies(roots ...string) ([]SysmacDependencies, error) {
	if len(roots) == 0 {
		roots = defaultSysmacRoots()
	}
	var installations []SysmacDependencies
	seen := make(map[string]struct{})
	for _, root := range roots {
		root = filepath.Clean(root)
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || entry.Name() != "nexcc.exe" {
				return nil
			}
			installationRoot := filepath.Dir(filepath.Dir(path))
			key := filepath.Clean(installationRoot)
			if _, ok := seen[key]; ok {
				return nil
			}
			seen[key] = struct{}{}
			installations = append(installations, inspectSysmacDependencies(installationRoot))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(installations, func(i, j int) bool {
		return installations[i].InstallationRoot < installations[j].InstallationRoot
	})
	return installations, nil
}

func inspectSysmacDependencies(root string) SysmacDependencies {
	result := SysmacDependencies{
		InstallationRoot: root,
		SysmacStudio:     filepath.Join(root, "SysmacStudio.exe"),
		NexCC:            filepath.Join(root, "builder2", "nexcc.exe"),
		NexBuilder2:      filepath.Join(root, "builder2", "NexBuilder2.dll"),
		NexProgramming:   filepath.Join(root, "Modules", "Nex", "NexProgramming.dll"),
		GCC:              filepath.Join(root, "gcc", "smn002", "bin", "i686-pc-smn002-gcc.exe"),
	}
	paths := []struct {
		name string
		path string
	}{
		{"nexcc", result.NexCC},
		{"nex_builder2", result.NexBuilder2},
		{"nex_programming", result.NexProgramming},
		{"gcc", result.GCC},
	}
	for _, dependency := range paths {
		if _, err := os.Stat(dependency.path); err != nil {
			result.Missing = append(result.Missing, dependency.name)
		}
	}
	result.Complete = len(result.Missing) == 0
	return result
}

func defaultSysmacRoots() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	return []string{
		`C:\Program Files\OMRON`,
		`C:\Program Files (x86)\OMRON`,
		`C:\OMRON`,
	}
}
