package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// ReadStructuredText extracts normalized source from a StructuredTextModel XML payload.
func ReadStructuredText(path string) (string, error) { return sysmac.ReadStructuredText(path) }
