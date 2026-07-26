package sysmac

import (
	"context"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed nexbuilder-headless.ps1
var embeddedHeadlessNexBuilderScript []byte

type ProjectBuildDiagnostic struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

type ProjectBuildResult struct {
	Status           string                   `json:"status"`
	ProjectPath      string                   `json:"project_path"`
	ManifestPath     string                   `json:"manifest_path,omitempty"`
	BuildResultsPath string                   `json:"build_results_path,omitempty"`
	ValidationPath   string                   `json:"validation_results_path,omitempty"`
	StartedAt        time.Time                `json:"started_at"`
	FinishedAt       time.Time                `json:"finished_at"`
	Diagnostics      []ProjectBuildDiagnostic `json:"diagnostics,omitempty"`
	Dependencies     []SysmacDependencies     `json:"dependencies,omitempty"`
	GeneratedInputs  []string                 `json:"generated_inputs,omitempty"`
	HostMessage      string                   `json:"host_message,omitempty"`
	BuilderResults   []HeadlessBuilderResult  `json:"builder_results,omitempty"`
}

type ProjectBuildOptions struct {
	Timeout time.Duration
}

type HeadlessBuilderMessage struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	File    string `json:"file"`
	Y       int    `json:"y"`
}

type HeadlessBuilderResult struct {
	File     string                   `json:"file"`
	Status   string                   `json:"status"`
	Messages []HeadlessBuilderMessage `json:"messages,omitempty"`
}

type headlessBuilderOutput struct {
	Results        []HeadlessBuilderResult `json:"results"`
	GeneratedCxif2 []string                `json:"generated_cxif2"`
}

// BuildProject runs the headless NexBuilder2 and nexcc stages for a regular
// Sysmac project. Sysmac Studio is not started or required.
func BuildProject(ctx context.Context, projectPath string, options ProjectBuildOptions) (ProjectBuildResult, error) {
	if options.Timeout <= 0 {
		options.Timeout = 2 * time.Minute
	}
	root, manifest, err := discoverProjectRoot(projectPath)
	if err != nil {
		return ProjectBuildResult{}, err
	}
	result := ProjectBuildResult{ProjectPath: root, ManifestPath: manifest, StartedAt: time.Now()}
	result.Dependencies, _ = DiscoverSysmacDependencies()
	outputDir, err := os.MkdirTemp("", "omron-mcp-nex-build-")
	if err != nil {
		return result, fmt.Errorf("create headless Nex output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)
	inputDirectory := findCxilInputDirectory(root)
	if inputDirectory == "" {
		result.Status = "headless_builder_failed"
		result.HostMessage = fmt.Sprintf("no .cxil2 sources found below %s", root)
		result.FinishedAt = time.Now()
		return result, nil
	}
	script := filepath.Join(outputDir, "nexbuilder-headless.ps1")
	if err := os.WriteFile(script, embeddedHeadlessNexBuilderScript, 0o600); err != nil {
		result.Status = "headless_builder_failed"
		result.HostMessage = fmt.Sprintf("write embedded NexBuilder script: %v", err)
		result.FinishedAt = time.Now()
		return result, nil
	}
	output, err := runHeadlessNexBuilder(ctx, script, inputDirectory, outputDir)
	if err != nil {
		result.Status = "headless_builder_failed"
		result.HostMessage = err.Error()
		result.FinishedAt = time.Now()
		return result, nil
	}
	result.BuilderResults = output.Results
	result.GeneratedInputs = output.GeneratedCxif2
	for _, build := range output.Results {
		for _, message := range build.Messages {
			result.Diagnostics = append(result.Diagnostics, ProjectBuildDiagnostic{Code: strconv.FormatUint(uint64(message.Code), 10), Message: message.Message, File: message.File, Line: message.Y})
		}
	}
	if len(output.GeneratedCxif2) == 0 {
		result.Status = "completed_with_errors"
		result.FinishedAt = time.Now()
		return result, nil
	}
	compiler, err := (&NexCCRunner{}).Build(ctx, outputDir)
	if err != nil {
		result.Status = "compiler_failed"
		result.HostMessage = err.Error()
	} else {
		result.Status = "succeeded"
		if compiler.Failed > 0 || len(result.Diagnostics) > 0 {
			result.Status = "completed_with_errors"
		}
		for _, diagnostic := range compiler.Errors {
			result.Diagnostics = append(result.Diagnostics, ProjectBuildDiagnostic{Code: diagnostic.Code, Message: diagnostic.Message, File: diagnostic.File, Line: diagnostic.Line, Column: diagnostic.Column})
		}
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func findCxilInputDirectory(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		matches, _ := filepath.Glob(filepath.Join(path, "*.cxil2"))
		if len(matches) > 0 {
			found = path
		}
		return nil
	})
	return found
}

func discoverProjectRoot(path string) (string, string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return "", "", errors.New("project path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("stat project path: %w", err)
	}
	if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".manifest") {
		return filepath.Dir(path), path, nil
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	manifests, _ := filepath.Glob(filepath.Join(path, "*.manifest"))
	if len(manifests) > 0 {
		return path, manifests[0], nil
	}
	for parent := filepath.Dir(path); parent != path; parent = filepath.Dir(parent) {
		manifests, _ = filepath.Glob(filepath.Join(parent, "*.manifest"))
		if len(manifests) > 0 {
			return parent, manifests[0], nil
		}
	}
	return "", "", fmt.Errorf("no Sysmac .manifest found at or above %q", path)
}

func runHeadlessNexBuilder(ctx context.Context, script, root, output string) (headlessBuilderOutput, error) {
	var result headlessBuilderOutput
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-ProjectDirectory", root, "-OutputDirectory", output, "-Target", "NX5")
	data, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = err.Error()
		}
		return result, fmt.Errorf("run headless NexBuilder: %s", message)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("parse headless NexBuilder output: %w", err)
	}
	return result, nil
}

func findProjectResults(root string) (string, string) {
	build, _ := filepath.Glob(filepath.Join(root, "*.NexBuildResults"))
	validation, _ := filepath.Glob(filepath.Join(root, "*.ValidationResults"))
	var buildPath, validationPath string
	if len(build) > 0 {
		buildPath = build[0]
	}
	if len(validation) > 0 {
		validationPath = validation[0]
	}
	return buildPath, validationPath
}

func fileModTime(path string) time.Time {
	if path == "" {
		return time.Time{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func changedAfter(path string, before time.Time) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && (before.IsZero() || info.ModTime().After(before))
}

func findGeneratedInputs(root string) []string {
	var paths []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".cxif2") {
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}

type buildXMLNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr     `xml:",any,attr"`
	Text    string         `xml:",chardata"`
	Nodes   []buildXMLNode `xml:",any"`
}

func parseProjectBuildDiagnostics(paths ...string) []ProjectBuildDiagnostic {
	var result []ProjectBuildDiagnostic
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var root buildXMLNode
		if xml.Unmarshal(data, &root) != nil {
			continue
		}
		collectBuildDiagnostics(root, &result)
	}
	return result
}

func collectBuildDiagnostics(node buildXMLNode, result *[]ProjectBuildDiagnostic) {
	var diagnostic ProjectBuildDiagnostic
	for _, attr := range node.Attrs {
		switch strings.ToLower(attr.Name.Local) {
		case "code", "errorcode":
			diagnostic.Code = attr.Value
		case "message", "description", "text":
			diagnostic.Message = attr.Value
		case "file", "filepath":
			diagnostic.File = attr.Value
		case "line", "lineid":
			diagnostic.Line, _ = strconv.Atoi(attr.Value)
		case "column", "columnid":
			diagnostic.Column, _ = strconv.Atoi(attr.Value)
		}
	}
	if diagnostic.Code != "" || diagnostic.Message != "" || diagnostic.File != "" {
		*result = append(*result, diagnostic)
	}
	for _, child := range node.Nodes {
		collectBuildDiagnostics(child, result)
	}
}
