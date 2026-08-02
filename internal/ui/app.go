package ui

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	"github.com/rjboer/omron-mcp/internal/updater"
)

func Run(logger *log.Logger, initialPath, mcpExecutable, version string) error {
	a := app.NewWithID("rjboer.omron-sysmac-mcp-workbench")
	a.Settings().SetTheme(newWorkbenchTheme())
	w := a.NewWindow("OMRON Sysmac MCP Workbench")
	w.Resize(fyne.NewSize(1100, 720))
	w.CenterOnScreen()
	savedPath := a.Preferences().String("discovery.last-folder")
	state := &model.Workbench{
		ProjectPath:        InitialDiscoveryPath(savedPath, initialPath),
		MCPExecutable:      mcpExecutable,
		MCPStatus:          string(mcp.HealthDisconnected),
		DiscoveryPathValid: true,
	}
	applicationExecutable, _ := os.Executable()
	if err := updater.EnsureHelperVersion(applicationExecutable, updater.HelperPath(applicationExecutable), version); err != nil && logger != nil {
		logger.Printf("updater helper is unavailable: %v", err)
	}
	w.SetContent(buildWorkbench(w, state, logger, applicationExecutable, version, a.Preferences()))
	w.ShowAndRun()
	return nil
}

func buildWorkbench(w fyne.Window, state *model.Workbench, logger *log.Logger, applicationExecutable, version string, prefs fyne.Preferences) fyne.CanvasObject {
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
	projectDetails := widget.NewLabel("Select a project to view its details.")
	projectDetails.Wrapping = fyne.TextWrapWord
	projectCount := widget.NewLabel("Discovered Projects (0)")
	projectCount.TextStyle = fyne.TextStyle{Bold: true}
	discoveryMessage := widget.NewLabel(DiscoveryMessage(false, false, 0, false, state.DiscoveryPathValid))
	discoveryMessage.Wrapping = fyne.TextWrapWord
	activityLines := []string{}
	activityView := widget.NewEntry()
	activityView.MultiLine = true
	activityView.Wrapping = fyne.TextWrapOff
	activityView.Disable()
	appendActivity := func(line string) {
		activityLines = appendLogLine(activityLines, line, 300)
		activityView.SetText(strings.Join(activityLines, "\n"))
	}

	filter := widget.NewEntry()
	filter.SetPlaceHolder("Filter by type, subtype, name, or entity ID")
	filtered := []sysmac.EntitySummary{}
	entityList := widget.NewList(
		func() int { return len(filtered) },
		func() fyne.CanvasObject { return newEntityRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= 0 && id < len(filtered) {
				setEntityRow(obj, filtered[id], id)
			}
		},
	)
	entityList.HideSeparators = true

	selectedEntityPanel := container.NewVBox(widget.NewLabel("Select an entity to inspect its metadata."))

	addActivity := func(operation, id, statusText, detail string) {
		if logger != nil {
			logger.Printf("%s entity=%s status=%s detail=%s", operation, id, statusText, detail)
		}
		line := fmt.Sprintf("%s [%s] %s", operation, statusText, detail)
		fyne.Do(func() { appendActivity(line) })
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
		selectedEntityPanel.RemoveAll()
		metadata := widget.NewLabel(entityMetadataText(entity))
		metadata.Wrapping = fyne.TextWrapOff
		selectedEntityPanel.Add(newEntityDetailsForm(entity))
		selectedEntityPanel.Add(widget.NewCard("System Metadata", "", metadata))
		selectedEntityPanel.Refresh()
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
		projectCount.SetText(fmt.Sprintf("Discovered Projects (%d)", len(state.DiscoveryCandidates)))
	}
	projectList := widget.NewList(
		func() int { return len(state.DiscoveryCandidates) },
		func() fyne.CanvasObject { return newProjectCandidateRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(state.DiscoveryCandidates) {
				return
			}
			setProjectCandidateRow(obj, state.DiscoveryCandidates[id])
		},
	)
	projectList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(state.DiscoveryCandidates) {
			return
		}
		selectedCandidate = id
		candidate := state.DiscoveryCandidates[id]
		state.SelectedProjectFolder = candidate.Folder
		projectDetails.SetText(projectDetailsText(candidate))
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
		selectedEntityPanel.RemoveAll()
		selectedEntityPanel.Add(widget.NewLabel("Select an entity to inspect its metadata."))
		selectedEntityPanel.Refresh()
		details := "Folder: " + projectPath
		if inspection.ProjectName != "" {
			details = "Name: " + inspection.ProjectName + "\n" + details
		}
		projectDetails.SetText(details)
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
	openSelectedButton.Importance = widget.HighImportance
	browseButton.Importance = widget.LowImportance
	capButton := widget.NewButton("Show Capabilities", func() {
		dialog.ShowInformation("Capability boundary", "The GUI provides read-only project discovery and entity metadata. Native project mutations are available only through confirmed MCP tool calls. PLC online operations are outside this application.", w)
	})

	controlHeight := path.MinSize().Height
	pathRow := container.NewBorder(nil, nil, nil, controlWithHeight(browseButton, controlHeight), path)
	discoveryActions := container.NewHBox(
		controlWithHeight(scanButton, controlHeight),
		controlWithHeight(openSelectedButton, controlHeight),
	)
	scanInfo := widget.NewCard("", "", widget.NewLabel("Read-only scan · Projects are not modified. Select a valid project to open."))
	discoveryHeader := container.NewVBox(
		widget.NewLabelWithStyle("Discover Sysmac Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Scan a folder or open an existing Sysmac Studio project."),
		widget.NewLabel("Sysmac project folder or solution root"),
		pathRow,
		discoveryActions,
		discoveryMessage,
		widget.NewSeparator(),
		projectCount,
	)
	discoveryList := container.NewBorder(discoveryHeader, nil, nil, nil, projectList)
	discoveryDetails := container.NewPadded(container.NewVBox(scanInfo, widget.NewCard("Project Details", "", projectDetails)))
	discoverySplit := container.NewHSplit(discoveryList, discoveryDetails)
	discoverySplit.SetOffset(0.68)
	discovery := discoverySplit
	explorerHeader := container.NewVBox(
		widget.NewLabelWithStyle("Project Explorer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Inspect read-only metadata from the selected Sysmac project."),
		filter,
	)
	entityListPanel := container.NewBorder(widget.NewLabelWithStyle("Entities", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, entityList)
	explorerLeft := container.NewBorder(explorerHeader, nil, nil, nil, entityListPanel)
	explorerRight := container.NewPadded(widget.NewCard("Selected entity", "", selectedEntityPanel))
	explorerSplit := container.NewHSplit(explorerLeft, explorerRight)
	explorerSplit.SetOffset(0.62)
	explorer := explorerSplit
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
	validationDescription := widget.NewLabel("Build the selected project through the headless NexBuilder2 and NexCC pipeline. PLC online operations remain disabled.")
	validationDescription.Wrapping = fyne.TextWrapWord
	dependencyLabel := widget.NewLabel(dependencyText)
	dependencyLabel.Wrapping = fyne.TextWrapWord
	buildOutput := widget.NewEntry()
	buildOutput.MultiLine = true
	buildOutput.Wrapping = fyne.TextWrapOff
	buildOutput.Disable()
	buildOutput.SetText("Build output will appear here.")
	buildButton := widget.NewButton("Build Selected Project", func() {
		projectPath := strings.TrimSpace(state.SelectedProjectFolder)
		if projectPath == "" {
			dialog.ShowInformation("No project selected", "Select a valid project before starting a build.", w)
			return
		}
		buildOutput.SetText("Building " + projectPath + "…")
		addActivity("Build", projectPath, "started", "headless NexBuilder2/NexCC")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			result, err := sysmac.BuildProject(ctx, projectPath, sysmac.ProjectBuildOptions{Timeout: 2 * time.Minute})
			cancel()
			output := fmt.Sprintf("Status: %s\nProject: %s\nGenerated inputs: %d", result.Status, result.ProjectPath, len(result.GeneratedInputs))
			if result.HostMessage != "" {
				output += "\nMessage: " + result.HostMessage
			}
			for _, diagnostic := range result.Diagnostics {
				output += fmt.Sprintf("\n[%s] %s", diagnostic.Code, diagnostic.Message)
			}
			if err != nil {
				output += "\nError: " + err.Error()
			}
			fyne.Do(func() {
				buildOutput.SetText(output)
				if err != nil {
					status.SetText("Build failed")
					addActivity("Build", projectPath, "failed", err.Error())
				} else {
					status.SetText("Build " + result.Status)
					addActivity("Build", projectPath, result.Status, fmt.Sprintf("%d diagnostics", len(result.Diagnostics)))
				}
			})
		}()
	})
	validation := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("Validation / Build", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Review the available local build adapter and detected dependencies."),
		widget.NewCard("NexCC build adapter", "Available", validationDescription),
		buildButton,
		widget.NewCard("Build output", "Live", container.NewVScroll(buildOutput)),
		widget.NewCard("Detected build dependencies", "", dependencyLabel),
		capButton,
	))
	gitlab := container.NewVBox(
		widget.NewLabelWithStyle("Git / GitLab", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Repository status, commits, pushes, and merge requests remain capability-gated; no automatic push is performed."),
		widget.NewCard("Repository actions", "Not connected", widget.NewLabel("No Git or GitLab operation is available from this screen.")),
	)
	executableEntry := widget.NewEntry()
	executableEntry.SetText(state.MCPExecutable)
	applyExecutableButton := widget.NewButton("Apply executable", func() {
		state.MCPExecutable = strings.TrimSpace(executableEntry.Text)
		addActivity("MCP Configuration", "", "applied", state.MCPExecutable)
		status.SetText("MCP executable configured")
	})
	mcpConnection := container.NewVBox(
		widget.NewLabelWithStyle("MCP Connection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Validates the local stdio server; it does not indicate PLC or controller connectivity."),
		widget.NewCard("Connection status", "", mcpPanelDetails),
		widget.NewCard("Configured executable", "", container.NewVBox(executableEntry, applyExecutableButton, resolvedExecutableLabel, workingDirectoryLabel)),
		widget.NewCard("MCP log (latest 300 entries)", "Rolling", container.NewVScroll(activityView)),
	)
	updateButton := widget.NewButton("Check for updates", func() {
		go func() {
			release, err := updater.Latest(context.Background(), http.DefaultClient, updater.DefaultRepository)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, w) })
				return
			}
			asset, err := release.WindowsAsset()
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, w) })
				return
			}
			if !release.HasUpdate(version) {
				fyne.Do(func() {
					dialog.ShowInformation("No update available", "This installation is already running the latest published build.", w)
				})
				return
			}
			if asset.SHA256() == "" {
				fyne.Do(func() { dialog.ShowError(fmt.Errorf("GitHub release did not provide a SHA-256 checksum"), w) })
				return
			}
			helper := filepath.Join(filepath.Dir(applicationExecutable), "omron-mcp-updater.exe")
			if _, err := os.Stat(helper); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("updater executable not found next to the application: %s", helper), w)
				})
				return
			}
			command := exec.Command(helper, "--update-helper", "--target", applicationExecutable, "--asset-url", asset.BrowserDownloadURL, "--sha256", asset.SHA256())
			if err := command.Start(); err != nil {
				fyne.Do(func() { dialog.ShowError(err, w) })
				return
			}
			fyne.Do(func() { status.SetText("Update downloaded; restarting..."); w.Close() })
		}()
	})
	mcpConnection.Add(container.NewHBox(widget.NewButton("Test MCP Connection", checkMCP), updateButton))

	mainTabs := newMainTabs(discovery, explorer, validation, gitlab, mcpConnection)
	go checkMCP()
	statusContent := container.NewBorder(nil, nil, mcpHeaderStatus, status, mcpHeaderDetails)
	statusBar := widget.NewCard("", "", statusContent)
	return container.NewBorder(nil, statusBar, nil, nil, mainTabs)
}
