package functions

// ListContainerEntries returns the names stored in an opened project container.
func ListContainerEntries(container *Container) []string { return container.Entries() }
