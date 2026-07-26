package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// ParseOEM parses the Sysmac .oem entity index at path.
func ParseOEM(path string) (OEMProject, error) { return sysmac.ParseOEM(path) }
