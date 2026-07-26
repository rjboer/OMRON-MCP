package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// Detect identifies the native Sysmac storage mode at path.
func Detect(path string) (ProjectLocation, error) { return sysmac.Detect(path) }
