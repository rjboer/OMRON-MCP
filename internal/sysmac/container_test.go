package sysmac

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenContainerListsAndReadsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.smc2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("project.oem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(`<sysmac/>`)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	container, err := OpenContainer(path)
	if err != nil {
		t.Fatal(err)
	}
	defer container.Close()
	entries := container.Entries()
	if len(entries) != 1 || entries[0] != "project.oem" {
		t.Fatalf("entries = %#v", entries)
	}
	data, err := container.ReadEntry("project.oem")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "<sysmac/>" {
		t.Fatalf("data = %q", data)
	}
}
