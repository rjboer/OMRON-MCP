package functions

// ReadContainerEntry reads one named entry from an opened project container.
func ReadContainerEntry(container *Container, name string) ([]byte, error) {
	return container.ReadEntry(name)
}
