package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// CopyContainer copies a project container to target without interpreting it.
func CopyContainer(source, target string) error { return sysmac.CopyContainer(source, target) }
