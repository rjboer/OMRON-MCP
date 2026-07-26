package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

type ProjectKind = sysmac.ProjectKind
type ProjectLocation = sysmac.ProjectLocation
type Inspection = sysmac.Inspection
type EntitySummary = sysmac.EntitySummary
type OEMProject = sysmac.OEMProject
type OEMEntity = sysmac.OEMEntity
type SLWDVariable = sysmac.SLWDVariable
type Container = sysmac.Container

const (
	ProjectKindSolutionDirectory = sysmac.ProjectKindSolutionDirectory
	ProjectKindContainer         = sysmac.ProjectKindContainer
)
