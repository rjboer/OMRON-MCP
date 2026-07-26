package model

import (
	"time"

	"github.com/rjboer/omron-mcp/internal/sysmac"
)

type Workbench struct {
	ProjectPath           string
	DiscoveryCandidates   []sysmac.ProjectCandidate
	SelectedProjectFolder string
	DiscoveryScanned      bool
	DiscoveryScanning     bool
	DiscoveryPathValid    bool
	MCPExecutable         string
	MCPResolvedExecutable string
	MCPWorkingDirectory   string
	MCPStatus             string
	MCPServerName         string
	MCPVersion            string
	MCPProtocol           string
	MCPToolCount          int
	MCPCheckedAt          time.Time
	MCPMessage            string
	Inspection            sysmac.Inspection
	SelectedID            string
	Filter                string
}

func New(path string) Workbench { return Workbench{ProjectPath: path} }
