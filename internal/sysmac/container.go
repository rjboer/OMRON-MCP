package sysmac

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

type Container struct {
	path string
	file *zip.ReadCloser
}

func OpenContainer(path string) (*Container, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open Sysmac container: %w", err)
	}
	return &Container{path: path, file: archive}, nil
}

func (c *Container) Close() error {
	if c == nil || c.file == nil {
		return nil
	}
	return c.file.Close()
}

func (c *Container) Entries() []string {
	entries := make([]string, 0, len(c.file.File))
	for _, entry := range c.file.File {
		entries = append(entries, entry.Name)
	}
	return entries
}

func (c *Container) ReadEntry(name string) ([]byte, error) {
	for _, entry := range c.file.File {
		if entry.Name != name {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open container entry %q: %w", name, err)
		}
		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read container entry %q: %w", name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close container entry %q: %w", name, closeErr)
		}
		return data, nil
	}
	return nil, fmt.Errorf("container entry %q not found", name)
}

func CopyContainer(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read container: %w", err)
	}
	if err := os.WriteFile(target, data, 0600); err != nil {
		return fmt.Errorf("write container copy: %w", err)
	}
	return nil
}
