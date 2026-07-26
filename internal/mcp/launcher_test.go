package mcp

import (
	"strings"
	"testing"
)

func TestResolveExecutableUsesApplicationRootForRelativeBinPath(t *testing.T) {
	executable, workingDir, err := ResolveExecutable(`.\bin\omron-mcp.exe`, `C:\OMRON-MCP\bin\omron-mcp.exe`)
	if err != nil {
		t.Fatal(err)
	}
	if executable != `C:\OMRON-MCP\bin\omron-mcp.exe` {
		t.Fatalf("executable = %q", executable)
	}
	if workingDir != `C:\OMRON-MCP\bin` {
		t.Fatalf("working directory = %q", workingDir)
	}
}

func TestResolveExecutableDoesNotAppendBinTwice(t *testing.T) {
	executable, _, err := ResolveExecutable(`.\bin\omron-mcp.exe`, `C:\OMRON-MCP\bin\omron-mcp.exe`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(executable), `bin\bin`) {
		t.Fatalf("duplicate bin in %q", executable)
	}
}

func TestResolveExecutablePreservesAbsolutePath(t *testing.T) {
	executable, workingDir, err := ResolveExecutable(`C:\tools\omron-mcp.exe`, `C:\OMRON-MCP\bin\omron-mcp.exe`)
	if err != nil {
		t.Fatal(err)
	}
	if executable != `C:\tools\omron-mcp.exe` || workingDir != `C:\tools` {
		t.Fatalf("resolved absolute path = %q, %q", executable, workingDir)
	}
}
