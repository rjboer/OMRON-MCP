package sysmac

// ProjectKind identifies the native storage representation detected on disk.
type ProjectKind string

const (
	ProjectKindSolutionDirectory ProjectKind = "solution-directory"
	ProjectKindContainer         ProjectKind = "container"
)

type ProjectLocation struct {
	Path string      `json:"path"`
	Kind ProjectKind `json:"kind"`
}
