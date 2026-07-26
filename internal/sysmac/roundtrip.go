package sysmac

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopySolutionDirectory copies a native Solution directory without touching the source.
func CopySolutionDirectory(source, destination string) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}
	if sameOrWithin(source, destination) || sameOrWithin(destination, source) {
		return fmt.Errorf("source and destination overlap: %q and %q", source, destination)
	}
	if _, err := Detect(source); err != nil {
		return fmt.Errorf("validate source Solution: %w", err)
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination already exists: %q", destination)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat destination: %w", err)
	}
	if err := os.MkdirAll(destination, 0700); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not supported in Solution copy: %q", path)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, err = io.Copy(output, input)
		if closeErr := input.Close(); err == nil {
			err = closeErr
		}
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("copy Solution: %w", err)
	}
	return nil
}

// SHA256File returns the content hash of one file.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sameOrWithin(root, path string) bool {
	root = strings.TrimRight(filepath.Clean(root), string(filepath.Separator)) + string(filepath.Separator)
	path = strings.TrimRight(filepath.Clean(path), string(filepath.Separator)) + string(filepath.Separator)
	return strings.EqualFold(strings.TrimRight(path, string(filepath.Separator)), strings.TrimRight(root, string(filepath.Separator))) || strings.HasPrefix(strings.ToLower(path), strings.ToLower(root))
}
