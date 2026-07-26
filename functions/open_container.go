package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// OpenContainer opens a ZIP-compatible Sysmac project container.
func OpenContainer(path string) (*Container, error) { return sysmac.OpenContainer(path) }
