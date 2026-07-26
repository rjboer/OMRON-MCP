package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSysmacDependenciesFindsInstalledComponents(t *testing.T) {
	root := filepath.Join(t.TempDir(), "OMRON")
	install := filepath.Join(root, "Sysmac Studio")
	writeDependencyFile(t, filepath.Join(install, "builder2", "nexcc.exe"))
	writeDependencyFile(t, filepath.Join(install, "builder2", "NexBuilder2.dll"))
	writeDependencyFile(t, filepath.Join(install, "Modules", "Nex", "NexProgramming.dll"))
	writeDependencyFile(t, filepath.Join(install, "gcc", "smn002", "bin", "i686-pc-smn002-gcc.exe"))

	installations, err := DiscoverSysmacDependencies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(installations) != 1 {
		t.Fatalf("installations = %#v", installations)
	}
	found := installations[0]
	if !found.Complete || found.InstallationRoot != install {
		t.Fatalf("installation = %#v", found)
	}
	if found.NexCC == "" || found.NexBuilder2 == "" || found.NexProgramming == "" || found.GCC == "" {
		t.Fatalf("dependency paths = %#v", found)
	}
}

func TestDiscoverSysmacDependenciesReportsMissingComponents(t *testing.T) {
	root := filepath.Join(t.TempDir(), "OMRON")
	install := filepath.Join(root, "Sysmac Studio")
	writeDependencyFile(t, filepath.Join(install, "builder2", "nexcc.exe"))

	installations, err := DiscoverSysmacDependencies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(installations) != 1 || installations[0].Complete {
		t.Fatalf("installations = %#v", installations)
	}
	if len(installations[0].Missing) != 3 {
		t.Fatalf("missing = %#v", installations[0].Missing)
	}
}

func writeDependencyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
}
