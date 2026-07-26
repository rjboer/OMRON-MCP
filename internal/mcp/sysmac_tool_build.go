package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func sysmacBuildTools() []tool {
	return []tool{
		{Name: "sysmac_nexcc_build", Description: "Run Omron nexcc.exe for prepared .cxif2 build inputs and return compiler diagnostics. This does not prepare .cxil2 files or perform a full controller build.", InputSchema: objectSchema(map[string]any{"build_directory": stringProperty("Directory containing prepared .cxif2 inputs"), "executable": stringProperty("Optional nexcc.exe path")}, "build_directory")},
		{Name: "sysmac_project_build", Description: "Trigger the complete regular Sysmac Studio controller build for a solution/project and return persisted build diagnostics. This uses Sysmac Studio's hosted build service; it does not bypass password prompts.", InputSchema: objectSchema(map[string]any{"project_path": stringProperty("Sysmac solution directory, project directory, or .manifest path"), "timeout_seconds": map[string]any{"type": "integer", "description": "Maximum seconds to wait for NexBuildResults/ValidationResults (default 120)"}}, "project_path")},
		{Name: "sysmac_project_snapshot", Description: "Copy a native Sysmac Solution to a Git-managed workspace. Requires explicit confirmation.", InputSchema: objectSchema(map[string]any{"source_path": stringProperty("Source Sysmac Solution directory"), "destination_path": stringProperty("New Git workspace destination directory"), "confirm_snapshot": map[string]any{"type": "boolean", "description": "Must be true to create the snapshot"}}, "source_path", "destination_path", "confirm_snapshot")},
	}
}

func sysmacBuildToolHandlers() map[string]sysmacToolHandler {
	return map[string]sysmacToolHandler{
		"sysmac_nexcc_build":      handleNexCCBuild,
		"sysmac_project_build":    handleProjectBuild,
		"sysmac_project_snapshot": handleProjectSnapshot,
	}
}

func handleNexCCBuild(call sysmacToolCall) (map[string]any, error) {
	buildDirectory, err := call.string("build_directory", true)
	if err != nil {
		return nil, err
	}
	executable, err := call.string("executable", false)
	if err != nil {
		return nil, err
	}
	result, err := (&sysmac.NexCCRunner{Executable: executable}).Build(call.context, buildDirectory)
	if err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": map[bool]string{true: "succeeded", false: "completed_with_errors"}[result.Failed == 0], "inputs": result.Inputs, "succeeded": result.Succeeded, "failed": result.Failed, "diagnostics": result.Diagnostics, "errors": result.Errors, "dependencies": result.Dependencies, "build_directory": buildDirectory}), nil
}

func handleProjectBuild(call sysmacToolCall) (map[string]any, error) {
	projectPath, err := call.string("project_path", true)
	if err != nil {
		return nil, err
	}
	timeout := 120 * time.Second
	if rawTimeout, ok := call.arguments["timeout_seconds"]; ok {
		var seconds int
		if json.Unmarshal(rawTimeout, &seconds) != nil || seconds <= 0 {
			return nil, errors.New("timeout_seconds must be a positive integer")
		}
		timeout = time.Duration(seconds) * time.Second
	}
	result, err := sysmac.BuildProject(call.context, projectPath, sysmac.ProjectBuildOptions{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	return toolSuccess(result), nil
}

func handleProjectSnapshot(call sysmacToolCall) (map[string]any, error) {
	source, err := call.string("source_path", true)
	if err != nil {
		return nil, err
	}
	destination, err := call.string("destination_path", true)
	if err != nil {
		return nil, err
	}
	if err := call.confirmed("confirm_snapshot", "confirm_snapshot must be true to create a project snapshot"); err != nil {
		return nil, err
	}
	absoluteSource, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("resolve source path: %w", err)
	}
	absoluteDestination, err := filepath.Abs(destination)
	if err != nil {
		return nil, fmt.Errorf("resolve destination path: %w", err)
	}
	if err := sysmac.CopySolutionDirectory(absoluteSource, absoluteDestination); err != nil {
		return nil, err
	}
	return toolSuccess(map[string]any{"status": "copied", "source_path": absoluteSource, "destination_path": absoluteDestination}), nil
}
