package functions

// LoadProject inspects a native project and returns its current index.
func LoadProject(path string) (Inspection, error) { return Inspect(path) }
