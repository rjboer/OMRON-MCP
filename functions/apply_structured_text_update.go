package functions

// ApplyStructuredTextUpdate applies an explicit Structured Text mutation.
func ApplyStructuredTextUpdate(projectRoot, entityID, text, protectedRoot string) error {
	return WriteStructuredTextEntity(projectRoot, entityID, text, protectedRoot)
}
