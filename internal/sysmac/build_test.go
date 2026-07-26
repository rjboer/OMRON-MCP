package sysmac

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNexCCBuildRunsEachCxif2Input(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a portable test command; Windows command resolution is covered separately")
	}

	binDir := t.TempDir()
	argsFile := filepath.Join(binDir, "args.txt")
	tool := filepath.Join(binDir, "nexcc")
	writeExecutable(t, tool, "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \""+argsFile+"\"\n")
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "b.cxif2"), "{SET<OUTPUT-C-FILE=/tmp/b.c>}\n")
	writeFile(t, filepath.Join(project, "a.cxif2"), "{SET<OUTPUT-C-FILE=/tmp/a.c>}\n")
	writeFile(t, filepath.Join(project, "ignore.cxil2"), "not a nexcc input\n")

	result, err := (&NexCCRunner{Executable: tool}).Build(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 0 || result.Succeeded != 2 {
		t.Fatalf("result = %#v, want two successful inputs", result)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(args)
	if !strings.Contains(got, "--input-file") || !strings.Contains(got, "a.cxif2") || !strings.Contains(got, "b.cxif2") {
		t.Fatalf("nexcc arguments = %q", got)
	}
}

func TestNexCCBuildReturnsDiagnosticsForFailedInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a portable test command; Windows command resolution is covered separately")
	}

	binDir := t.TempDir()
	tool := filepath.Join(binDir, "nexcc")
	writeExecutable(t, tool, "#!/bin/sh\necho 'MainProgram:64:19: error: Undefined identifier SafetyBoundary' >&2\necho 'B0100200: error' >&2\nexit 63\n")
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "broken.cxif2"), "input\n")

	result, err := (&NexCCRunner{Executable: tool}).Build(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 0 || result.Failed != 1 {
		t.Fatalf("result = %#v, want one failed input", result)
	}
	if !strings.Contains(result.Diagnostics, "B0100200") {
		t.Fatalf("diagnostics = %q", result.Diagnostics)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("errors = %#v", result.Errors)
	}
	if result.Errors[0].File != "MainProgram" || result.Errors[0].Line != 64 || result.Errors[0].Column != 19 {
		t.Fatalf("location = %#v", result.Errors[0])
	}
	if result.Errors[0].Severity != "error" || result.Errors[0].Message != "Undefined identifier SafetyBoundary" {
		t.Fatalf("diagnostic = %#v", result.Errors[0])
	}
	if result.Errors[1].Code != "B0100200" {
		t.Fatalf("code = %#v", result.Errors[1])
	}
}

func TestNexCCBuildRejectsMissingProjectDirectory(t *testing.T) {
	_, err := (&NexCCRunner{Executable: "nexcc.exe"}).Build(context.Background(), filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected missing project directory error")
	}
}

func TestNexCCBuildManualInstalledSysmac(t *testing.T) {
	directory := os.Getenv("OMRON_MCP_NEX_BUILD_DIR")
	if directory == "" {
		t.Skip("OMRON_MCP_NEX_BUILD_DIR is not set")
	}

	result, err := (&NexCCRunner{}).Build(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("NexCC result: inputs=%d succeeded=%d failed=%d diagnostics=%s", len(result.Inputs), result.Succeeded, result.Failed, result.Diagnostics)
	if result.Failed != 0 {
		t.Fatalf("NexCC failed: %#v", result)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	writeFile(t, path, content)
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}
