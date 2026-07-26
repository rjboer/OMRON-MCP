package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// WriteStructuredTextEntity writes source for a Structured Text POU body selected by entity ID.
func WriteStructuredTextEntity(projectRoot, entityID, text, protectedRoot string) error {
	return sysmac.WriteStructuredTextEntity(projectRoot, entityID, text, protectedRoot)
}
