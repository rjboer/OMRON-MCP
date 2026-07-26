package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// Inspect returns a JSON-friendly summary of a native project or container.
func Inspect(path string) (Inspection, error) { return sysmac.Inspect(path) }
