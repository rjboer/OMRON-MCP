package mcp

import (
	"errors"
	"path/filepath"
	"strings"
)

// ResolveExecutable turns the configured MCP path into an absolute executable
// path and returns the directory that should be used as its working directory.
// Relative paths are rooted at the application directory, not the process CWD.
func ResolveExecutable(configured, applicationExecutable string) (string, string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return "", "", errors.New("MCP executable is not configured")
	}

	if filepath.IsAbs(configured) {
		executable := filepath.Clean(configured)
		return executable, filepath.Dir(executable), nil
	}

	applicationExecutable = strings.TrimSpace(applicationExecutable)
	if applicationExecutable == "" {
		return "", "", errors.New("application executable is required to resolve a relative MCP path")
	}
	applicationDir := filepath.Dir(filepath.Clean(applicationExecutable))
	applicationRoot := applicationDir
	if strings.EqualFold(filepath.Base(applicationDir), "bin") {
		applicationRoot = filepath.Dir(applicationDir)
	}
	executable := filepath.Clean(filepath.Join(applicationRoot, configured))
	return executable, filepath.Dir(executable), nil
}
