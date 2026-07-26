package sysmac

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// NexCCRunner invokes Omron's command-line compiler for the compiler inputs
// already prepared by Sysmac's Nex build pipeline. It deliberately does not
// pretend to turn .cxil2 files into .cxif2 files; that preparation belongs to
// NexProgramming.dll inside Sysmac Studio.
type NexCCRunner struct {
	Executable string
}

type NexCCBuildResult struct {
	Inputs       []string
	Succeeded    int
	Failed       int
	Diagnostics  string
	Errors       []NexCCDiagnostic
	Dependencies []SysmacDependencies `json:"dependencies,omitempty"`
}

type NexCCDiagnostic struct {
	Code      string `json:"code,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Message   string `json:"message,omitempty"`
	File      string `json:"file,omitempty"`
	InputFile string `json:"input_file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

var (
	nexCCLocationPattern = regexp.MustCompile(`^(.+?):([0-9]+):([0-9]+):\s*(error|warning|note):\s*(.*)$`)
	nexCCCodePattern     = regexp.MustCompile(`\b([A-Z][0-9]{7,8})\b`)
)

func (r *NexCCRunner) Build(ctx context.Context, directory string) (NexCCBuildResult, error) {
	var result NexCCBuildResult
	result.Dependencies, _ = DiscoverSysmacDependencies()
	directory = filepath.Clean(strings.TrimSpace(directory))
	info, err := os.Stat(directory)
	if err != nil {
		return result, fmt.Errorf("stat Nex build directory: %w", err)
	}
	if !info.IsDir() {
		return result, fmt.Errorf("Nex build path is not a directory: %q", directory)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return result, fmt.Errorf("read Nex build directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".cxif2") {
			continue
		}
		result.Inputs = append(result.Inputs, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(result.Inputs)
	if len(result.Inputs) == 0 {
		return result, fmt.Errorf("no .cxif2 inputs found in %q", directory)
	}

	executable := r.Executable
	if executable == "" {
		executable = defaultNexCCPath()
	}
	for _, input := range result.Inputs {
		var output bytes.Buffer
		cmd := exec.CommandContext(ctx, executable, "--input-file", input)
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		if output.Len() > 0 {
			text := output.String()
			result.Diagnostics += fmt.Sprintf("%s:\n%s", filepath.Base(input), text)
			result.Errors = append(result.Errors, parseNexCCDiagnostics(input, text)...)
		}
		if err != nil {
			result.Failed++
			continue
		}
		result.Succeeded++
	}
	return result, nil
}

func parseNexCCDiagnostics(input, output string) []NexCCDiagnostic {
	var diagnostics []NexCCDiagnostic
	for _, rawLine := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		diagnostic := NexCCDiagnostic{InputFile: input, Raw: line}
		if matches := nexCCLocationPattern.FindStringSubmatch(line); matches != nil {
			diagnostic.File = matches[1]
			diagnostic.Line, _ = strconv.Atoi(matches[2])
			diagnostic.Column, _ = strconv.Atoi(matches[3])
			diagnostic.Severity = matches[4]
			diagnostic.Message = matches[5]
		} else if strings.Contains(strings.ToLower(line), "error") {
			diagnostic.Severity = "error"
			diagnostic.Message = line
		}
		if matches := nexCCCodePattern.FindStringSubmatch(line); matches != nil {
			diagnostic.Code = matches[1]
		}
		if diagnostic.Severity != "" || diagnostic.Code != "" {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics
}

func defaultNexCCPath() string {
	if runtime.GOOS == "windows" {
		candidates := []string{
			`C:\Program Files\OMRON\Sysmac Studio\builder2\nexcc.exe`,
			`C:\Program Files (x86)\OMRON\Sysmac Studio\builder2\nexcc.exe`,
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return "nexcc.exe"
}
