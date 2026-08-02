package updater

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const HelperName = "omron-mcp-updater.exe"

func HelperPath(applicationExecutable string) string {
	return filepath.Join(filepath.Dir(filepath.Clean(applicationExecutable)), HelperName)
}

func EnsureHelper(source, helper string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open application payload: %w", err)
	}
	defer sourceFile.Close()
	temporary := helper + ".new"
	temporaryFile, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("create updater payload: %w", err)
	}
	_, copyErr := io.Copy(temporaryFile, sourceFile)
	closeErr := temporaryFile.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("copy updater payload: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("close updater payload: %w", closeErr)
	}
	backup := helper + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(helper, backup); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(temporary)
		return fmt.Errorf("replace old updater payload: %w", err)
	}
	if err := os.Rename(temporary, helper); err != nil {
		_ = os.Rename(backup, helper)
		_ = os.Remove(temporary)
		return fmt.Errorf("install updater payload: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

func EnsureHelperVersion(source, helper, version string) error {
	if output, err := exec.Command(helper, "--version").Output(); err == nil && strings.TrimSpace(string(output)) == strings.TrimSpace(version) {
		return nil
	}
	return EnsureHelper(source, helper)
}
