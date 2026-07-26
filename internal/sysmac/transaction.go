package sysmac

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileChange struct {
	RelativePath string `json:"relativePath"`
	BeforeHash   string `json:"beforeHash"`
	AfterHash    string `json:"afterHash"`
}

type ProjectTransaction struct {
	SourceRoot string
	StageRoot  string
	tempParent string
	original   map[string]string
	closed     bool
}

func StartProjectTransaction(sourceRoot string) (*ProjectTransaction, error) {
	absolute, err := filepath.Abs(sourceRoot)
	if err != nil {
		return nil, err
	}
	if _, err := Detect(absolute); err != nil {
		return nil, fmt.Errorf("validate transaction source: %w", err)
	}
	parent, err := os.MkdirTemp("", "sysmac-transaction-")
	if err != nil {
		return nil, err
	}
	stage := filepath.Join(parent, "stage")
	if err := CopySolutionDirectory(absolute, stage); err != nil {
		os.RemoveAll(parent)
		return nil, err
	}
	hashes, err := hashProjectFiles(absolute)
	if err != nil {
		os.RemoveAll(parent)
		return nil, err
	}
	return &ProjectTransaction{SourceRoot: absolute, StageRoot: stage, tempParent: parent, original: hashes}, nil
}

func (tx *ProjectTransaction) StageFile(relativePath string, data []byte) error {
	if err := tx.ensureOpen(); err != nil {
		return err
	}
	path, err := tx.safeStagePath(relativePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (tx *ProjectTransaction) DeleteFile(relativePath string) error {
	if err := tx.ensureOpen(); err != nil {
		return err
	}
	path, err := tx.safeStagePath(relativePath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("staged file %q does not exist: %w", relativePath, err)
	}
	return os.Remove(path)
}

func (tx *ProjectTransaction) Preview() ([]FileChange, error) {
	if err := tx.ensureOpen(); err != nil {
		return nil, err
	}
	current, err := hashProjectFiles(tx.StageRoot)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for path := range tx.original {
		paths[path] = true
	}
	for path := range current {
		paths[path] = true
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	var result []FileChange
	for _, path := range ordered {
		before, beforeOK := tx.original[path]
		after, afterOK := current[path]
		if beforeOK && afterOK && before == after {
			continue
		}
		result = append(result, FileChange{RelativePath: path, BeforeHash: before, AfterHash: after})
	}
	return result, nil
}

func (tx *ProjectTransaction) Commit() error {
	if err := tx.ensureOpen(); err != nil {
		return err
	}
	currentSource, err := hashProjectFiles(tx.SourceRoot)
	if err != nil {
		return err
	}
	for path, expected := range tx.original {
		if currentSource[path] != expected {
			return fmt.Errorf("transaction hash conflict for %q", path)
		}
	}
	changes, err := tx.Preview()
	if err != nil {
		return err
	}
	for _, change := range changes {
		stagePath := filepath.Join(tx.StageRoot, filepath.FromSlash(change.RelativePath))
		targetPath := filepath.Join(tx.SourceRoot, filepath.FromSlash(change.RelativePath))
		if _, err := os.Stat(stagePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if original, err := os.ReadFile(targetPath); err == nil {
			if err := os.WriteFile(targetPath+".backup", original, 0600); err != nil {
				return err
			}
		}
	}
	for _, change := range changes {
		stagePath := filepath.Join(tx.StageRoot, filepath.FromSlash(change.RelativePath))
		targetPath := filepath.Join(tx.SourceRoot, filepath.FromSlash(change.RelativePath))
		data, err := os.ReadFile(stagePath)
		if os.IsNotExist(err) {
			if err := os.Remove(targetPath); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".sysmac-transaction-*.tmp")
		if err != nil {
			return err
		}
		tmpPath := tmp.Name()
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
		if err := tmp.Sync(); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmpPath)
			return err
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			os.Remove(tmpPath)
			return err
		}
	}
	tx.closed = true
	return os.RemoveAll(tx.tempParent)
}

func (tx *ProjectTransaction) Rollback() error {
	if tx == nil || tx.closed {
		return nil
	}
	tx.closed = true
	return os.RemoveAll(tx.tempParent)
}

func (tx *ProjectTransaction) ensureOpen() error {
	if tx == nil || tx.closed {
		return fmt.Errorf("transaction is closed")
	}
	return nil
}

func (tx *ProjectTransaction) safeStagePath(relativePath string) (string, error) {
	clean := filepath.Clean(relativePath)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid transaction relative path %q", relativePath)
	}
	return filepath.Join(tx.StageRoot, clean), nil
}

func hashProjectFiles(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".backup") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, err := SHA256File(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = hash
		return nil
	})
	return result, err
}
