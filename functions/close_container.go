package functions

// CloseContainer closes an opened project container.
func CloseContainer(container *Container) error { return container.Close() }
