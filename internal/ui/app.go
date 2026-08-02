package ui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/rjboer/omron-mcp/internal/mcp"
	"github.com/rjboer/omron-mcp/internal/model"
	"github.com/rjboer/omron-mcp/internal/sysmac"
)

func Run(logger *log.Logger, initialPath, mcpExecutable string) error {
	a := app.NewWithID("rjboer.omron-sysmac-mcp-workbench")
	w := a.NewWindow("OMRON Sysmac MCP Workbench")
	w.Resize(fyne.NewSize(800, 520))
	w.CenterOnScreen()
	savedPath := a.Preferences().String("discovery.last-folder")
	state := &model.Workbench{
		ProjectPath:        InitialDiscoveryPath(savedPath, initialPath),
		MCPExecutable:      mcpExecutable,
		MCPStatus:          string(mcp.HealthDisconnected),
		DiscoveryPathValid: true,
	}
	applicationExecutable, _ := os.Executable()
	w.SetContent(buildWorkbench(w, state, logger, applicationExecutable, a.Preferences()))
	w.ShowAndRun()
	return nil
}

func buildWorkbench(w fyne.Window, state *model.Workbench, logger *log.Logger, applicationExecutable string, prefs fyne.Preferences) fyne.CanvasObject {
	path := widget.NewEntry()
	path.SetText(state.ProjectPath)
	path.SetPlaceHolder(`C:\OMRON\Data\Solution`)
	status := widget.NewLabel("Ready — native project operations are capability-dependent")
	mcpHeaderStatus := widget.NewLabel("● MCP Disconnected")
	mcpHeaderDetails := widget.NewLabel("MCP health has not been checked")
	mcpHeaderDetails.Wrapping = fyne.TextWrapWord
	mcpPanelDetails := widget.NewLabel("MCP health has not been checked")
	mcpPanelDetails.Wrapping = fyne.TextWrapWord
	resolvedExecutableLabel := widget.NewLabel("Executable: not resolved")
	resolvedExecutableLabel.Wrapping = fyne.TextWrapWord
	workingDirectoryLabel := widget.NewLabel("Working directory: not resolved")
	workingDirectoryLabel.Wrapping = fyne.TextWrapWord
	projectSummary := widget.NewLabel("No project inspected")
	projectSummary.Wrapping = fyne.TextWrapWord
	discoveryMessage := widget.NewLabel(DiscoveryMessage(false, false, 0, false, state.DiscoveryPathValid))
	discoveryMessage.Wrapping = fyne.TextWrapWord

	filter := widget.NewEntry()
	filter.SetPlaceHolder("Filter by type, subtype, name, or entity ID")
	filtered := []sysmac.EntitySummary{}
	entityList := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= 0 && id < len(filtered) {
				obj.(*widget.Label).SetText(entityLabel(filtered[id]))
			}
		},
	)
	entityList.HideSeparators = true

	selectionLabel := widget.NewLabel("No entity selected")
	selectionLabel.Wrapping = fyne.TextWrapWord

	addActivity := func(operation, id, statusText, detail string) {
		if logger != nil {
			logger.Printf("%s entity=%s status=%s detail=%s", operation, id, statusText, detail)
		}
	}
	refreshMCP := func(health mcp.Health) {
		state.MCPStatus = string(health.Status)
		state.MCPServerName = health.ServerName
		state.MCPVersion = health.Version
		state.MCPProtocol = health.ProtocolVersion
		state.MCPToolCount = health.ToolCount
		state.MCPCheckedAt = health.CheckedAt
		state.MCPMessage = health.Message
		label := "MCP " + strings.Title(string(health.Status))
		if health.Status == mcp.HealthConnected {
			label = "● MCP Connected"
		} else if health.Status == mcp.HealthError {
			label = "● MCP Error"
		} else {
			label = "● MCP Disconnected"
		}
		mcpHeaderStatus.SetText(label)
		details := health.Message
		if health.ServerName != "" {
			details = fmt.Sprintf("%s %s · %d tools", health.ServerName, health.Version, health.ToolCount)
		}
		if !health.CheckedAt.IsZero() {
			details += " · checked " + health.CheckedAt.Local().Format("15:04:05")
		}
		mcpHeaderDetails.SetText(details)
		mcpPanelDetails.SetText(details)
	}
	checkMCP := func() {
		state.MCPStatus = "checking"
		mcpHeaderStatus.SetText("● MCP Checking…")
		mcpHeaderDetails.SetText("Starting MCP stdio server and validating initialize/tools/list…")
		mcpPanelDetails.SetText("Starting MCP stdio server and validating initialize/tools/list…")
		addActivity("MCP Health", "", "checking", state.MCPExecutable)
		go func() {
			executable, workingDir, err := mcp.ResolveExecutable(state.MCPExecutable, applicationExecutable)
			if err != nil {
				health := mcp.Health{Status: mcp.HealthError, Message: err.Error()}
				fyne.Do(func() { refreshMCP(health); addActivity("MCP Health", "", "failed", err.Error()) })
				return
			}
			state.MCPResolvedExecutable = executable
			state.MCPWorkingDirectory = workingDir
			fyne.Do(func() {
				resolvedExecutableLabel.SetText("Executable: " + executable)
				workingDirectoryLabel.SetText("Working directory: " + workingDir)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			health, probeErr := mcp.ProbeWithArgs(ctx, executable, workingDir, "--mcp")
			cancel()
			fyne.Do(func() {
				refreshMCP(health)
				if probeErr != nil {
					addActivity("MCP Health", "", "failed", probeErr.Error())
					status.SetText("MCP server connection failed")
					return
				}
				addActivity("MCP Health", "", "connected", health.Message)
				status.SetText("MCP server connected")
			})
		}()
	}
	refreshEntities := func() {
		filtered = FilteredEntities(state.Inspection.Entities, state.Filter)
		entityList.Refresh()
	}
	updateSelectionPanel := func(entity sysmac.EntitySummary) {
		selectionLabel.SetText(fmt.Sprintf(
			"Selected entity\nName: %s\nType: %s\nSubtype: %s\nID: %s\nTracking ID: %s",
			entity.Name, entity.Type, entity.Subtype, entity.ID, entity.TrackingID,
		))
	}
	selectedCandidate := -1
	refreshDiscoveryMessage := func() {
		validCount := 0
		for _, candidate := range state.DiscoveryCandidates {
			if candidate.Status == sysmac.ProjectStatusValid {
				validCount++
			}
		}
		discoveryMessage.SetText(DiscoveryMessage(state.DiscoveryScanned, state.DiscoveryScanning, validCount, selectedCandidate >= 0, state.DiscoveryPathValid))
	}
	projectList := widget.NewList(
		func() int { return len(state.DiscoveryCandidates) },
		func() fyne.CanvasObject { return newProjectCandidateRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(state.DiscoveryCandidates) {
				return
			}
			setProjectCandidateRow(obj.(*fyne.Container), state.DiscoveryCandidates[id])
		},
	)
	projectList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(state.DiscoveryCandidates) {
			return
		}
		selectedCandidate = id
		candidate := state.DiscoveryCandidates[id]
		state.SelectedProjectFolder = candidate.Folder
		projectSummary.SetText(projectCandidateLabel(candidate))
		refreshDiscoveryMessage()
		status.SetText("Selected project: " + candidate.Name)
	}
	saveDiscoveryPath := func(value string) {
		if prefs != nil && strings.TrimSpace(value) != "" {
			prefs.SetString("discovery.last-folder", strings.TrimSpace(value))
		}
	}
	scanFolder := func() {
		root := strings.TrimSpace(path.Text)
		if root == "" {
			state.DiscoveryScanned = true
			state.DiscoveryPathValid = false
			state.DiscoveryCandidates = nil
			selectedCandidate = -1
			refreshDiscoveryMessage()
			return
		}
		saveDiscoveryPath(root)
		state.DiscoveryScanning = true
		state.DiscoveryScanned = false
		state.DiscoveryPathValid = true
		state.DiscoveryCandidates = nil
		selectedCandidate = -1
		refreshDiscoveryMessage()
		projectList.Refresh()
		go func() {
			candidates, err := sysmac.DiscoverProjects(root)
			fyne.Do(func() {
				state.DiscoveryScanning = false
				state.DiscoveryScanned = true
				state.DiscoveryCandidates = candidates
				state.DiscoveryPathValid = err == nil
				if err != nil {
					addActivity("Scan Folder", "", "failed", err.Error())
					status.SetText("Folder scan failed")
				} else {
					addActivity("Scan Folder", "", "applied", fmt.Sprintf("%d candidate folders inspected", len(candidates)))
					status.SetText("Folder scan completed")
				}
				refreshDiscoveryMessage()
				projectList.Refresh()
			})
		}()
	}

	loadProject := func(projectPath string) {
		projectPath = strings.TrimSpace(projectPath)
		if projectPath == "" {
			err := fmt.Errorf("project path is required")
			addActivity("Inspect", "", "failed", err.Error())
			dialog.ShowError(err, w)
			return
		}
		status.SetText("Inspecting project…")
		inspection, err := sysmac.Inspect(projectPath)
		if err != nil {
			addActivity("Inspect", "", "failed", err.Error())
			status.SetText("Inspection failed")
			dialog.ShowError(err, w)
			return
		}
		state.ProjectPath, state.Inspection, state.SelectedID = projectPath, inspection, ""
		state.SelectedProjectFolder = projectPath
		selectionLabel.SetText("No entity selected")
		if inspection.ProjectName != "" {
			projectSummary.SetText(fmt.Sprintf("Project: %s\nKind: %s\nEntities: %d\nPath: %s", inspection.ProjectName, inspection.Location.Kind, len(inspection.Entities), projectPath))
		} else {
			projectSummary.SetText(fmt.Sprintf("Container: %s\nEntries: %d\nPath: %s", inspection.Location.Kind, len(inspection.Entries), projectPath))
		}
		refreshEntities()
		addActivity("Inspect", inspection.ProjectName, "applied", fmt.Sprintf("%d entities indexed", len(inspection.Entities)))
		status.SetText("Project inspected; select an entity")
	}
	openSelectedProject := func() {
		if selectedCandidate < 0 || selectedCandidate >= len(state.DiscoveryCandidates) {
			dialog.ShowInformation("No project selected", "Projects were found, but none has been selected.", w)
			return
		}
		candidate := state.DiscoveryCandidates[selectedCandidate]
		if candidate.Status != sysmac.ProjectStatusValid {
			dialog.ShowInformation("Project cannot be opened", candidate.Error, w)
			return
		}
		loadProject(candidate.Folder)
	}
	browseFolder := func() {
		folderDialog := dialog.NewFolderOpen(func(selected fyne.ListableURI, err error) {
			if err != nil {
				addActivity("Browse Folder", "", "failed", err.Error())
				return
			}
			if selected == nil {
				return
			}
			selectedPath := selected.Path()
			path.SetText(selectedPath)
			saveDiscoveryPath(selectedPath)
			state.DiscoveryPathValid = true
			state.DiscoveryScanned = false
			state.DiscoveryCandidates = nil
			selectedCandidate = -1
			projectList.Refresh()
			refreshDiscoveryMessage()
			status.SetText("Folder selected; click Scan Folder")
		}, w)
		if selectedPath := strings.TrimSpace(path.Text); selectedPath != "" {
			if info, err := os.Stat(selectedPath); err == nil && info.IsDir() {
				if location, err := storage.ListerForURI(storage.NewFileURI(selectedPath)); err == nil {
					folderDialog.SetLocation(location)
				}
			}
		}
		folderDialog.Show()
	}

	filter.OnChanged = func(value string) {
		state.Filter = value
		refreshEntities()
	}
	entityList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filtered) {
			return
		}
		state.SelectedID = filtered[id].ID
		updateSelectionPanel(filtered[id])
		status.SetText("Selected entity: " + filtered[id].ID)
	}

	browseButton := widget.NewButton("Browse...", browseFolder)
	scanButton := widget.NewButton("Scan Folder", scanFolder)
	openSelectedButton := widget.NewButton("Open Selected Project", openSelectedProject)
	capButton := widget.NewButton("Show Capabilities", func() {
		dialog.ShowInformation("Capability boundary", "The GUI provides read-only project discovery and entity metadata. Native project mutations are available only through confirmed MCP tool calls. PLC online operations are outside this application.", w)
	})

	pathRow := container.NewBorder(nil, nil, nil, browseButton, path)
	discoveryActions := container.NewHBox(scanButton, openSelectedButton)
	discovery := container.NewBorder(
		container.NewVBox(widget.NewLabel("Sysmac project folder or solution root"), pathRow, discoveryActions, discoveryMessage),
		container.NewVBox(projectSummary, widget.NewLabel("Scan is read-only. Select a valid project before opening it.")),
		nil, nil,
		projectList,
	)
	explorer := container.NewBorder(
		container.NewVBox(widget.NewLabel("Project Explorer"), filter, widget.NewLabel("Select an entity to view read-only metadata.")),
		container.NewVBox(widget.NewSeparator(), selectionLabel), nil, nil,
		entityList,
	)
	dependencies, _ := sysmac.DiscoverSysmacDependencies()
	dependencyText := "No Sysmac Studio installation found under the configured C:\\ roots."
	if len(dependencies) > 0 {
		var lines []string
		for _, dependency := range dependencies {
			status := "complete"
			if !dependency.Complete {
				status = "missing: " + strings.Join(dependency.Missing, ", ")
			}
			lines = append(lines,
				fmt.Sprintf("[%s] %s\nnexcc: %s\nNexBuilder2: %s\nNexProgramming: %s\nGCC: %s", status, dependency.InstallationRoot, dependency.NexCC, dependency.NexBuilder2, dependency.NexProgramming, dependency.GCC),
			)
		}
		dependencyText = strings.Join(lines, "\n\n")
	}
	validation := container.NewVBox(widget.NewLabel("NexCC build adapter available"), widget.NewLabel("Prepared .cxif2 inputs can be compiled through the sysmac_nexcc_build tool. Full .cxil2 preparation/controller build through NexProgramming.dll and PLC online operations are not enabled yet."), widget.NewSeparator(), widget.NewLabel("Detected build dependencies"), widget.NewLabel(dependencyText), capButton)
	gitlab := container.NewVBox(widget.NewLabel("Git / GitLab"), widget.NewLabel("Repository status, commits, pushes, and merge requests remain capability-gated; no automatic push is performed."))
	mcpConnection := container.NewVBox(
		widget.NewLabel("MCP Connection"),
		widget.NewLabel("The MCP status validates the local stdio server only. It does not indicate PLC or controller connectivity."),
		widget.NewSeparator(),
		widget.NewLabel("Configured executable: "+state.MCPExecutable),
		resolvedExecutableLabel,
		workingDirectoryLabel,
		mcpPanelDetails,
		widget.NewButton("Test MCP Connection", checkMCP),
	)

	mainTabs := newMainTabs(discovery, explorer, validation, gitlab, mcpConnection)
	go checkMCP()
	return container.NewBorder(nil, container.NewVBox(mcpHeaderStatus, mcpHeaderDetails, status), nil, nil, mainTabs)
}
