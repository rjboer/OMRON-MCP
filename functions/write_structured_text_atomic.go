package functions

import "github.com/rjboer/omron-mcp/internal/sysmac"

// WriteStructuredTextAtomic replaces one Structured Text payload and creates a backup.
func WriteStructuredTextAtomic(path, text, protectedRoot string) error {
	return sysmac.WriteStructuredTextAtomic(path, text, protectedRoot)
}
