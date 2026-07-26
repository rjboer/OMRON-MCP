package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// ReadSLWDVariables reads supported variable declarations from an SLWD payload.
func ReadSLWDVariables(path string) ([]SLWDVariable, error) {
	return sysmac.ReadSLWDVariables(path)
}
