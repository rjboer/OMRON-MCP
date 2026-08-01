package mcp

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExecutableUsesApplicationRootForRelativeBinPath(t *testing.T) {
	applicationRoot := t.TempDir()
	applicationExecutable := filepath.Join(applicationRoot, "bin", "omron-mcp.exe")
	configured := filepath.Join("bin", "omron-mcp.exe")

	executable, workingDir, err := ResolveExecutable(configured, applicationExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if executable != applicationExecutable {
		t.Fatalf("executable = %q", executable)
	}
	if workingDir != filepath.Join(applicationRoot, "bin") {
		t.Fatalf("working directory = %q", workingDir)
	}
}

func TestResolveExecutableDoesNotAppendBinTwice(t *testing.T) {
	applicationRoot := t.TempDir()
	applicationExecutable := filepath.Join(applicationRoot, "bin", "omron-mcp.exe")
	configured := filepath.Join("bin", "omron-mcp.exe")

	executable, _, err := ResolveExecutable(configured, applicationExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(executable), filepath.Join("bin", "bin")) {
		t.Fatalf("duplicate bin in %q", executable)
	}
}

func TestResolveExecutablePreservesAbsolutePath(t *testing.T) {
	absoluteExecutable := filepath.Join(t.TempDir(), "tools", "omron-mcp.exe")
	applicationExecutable := filepath.Join(t.TempDir(), "bin", "omron-mcp.exe")

	executable, workingDir, err := ResolveExecutable(absoluteExecutable, applicationExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if executable != absoluteExecutable || workingDir != filepath.Dir(absoluteExecutable) {
		t.Fatalf("resolved absolute path = %q, %q", executable, workingDir)
	}
}
