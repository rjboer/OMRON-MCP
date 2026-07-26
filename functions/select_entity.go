package functions

import "fmt"

// SelectEntity returns one entity from an inspection by its stable entity ID.
func SelectEntity(inspection Inspection, entityID string) (EntitySummary, error) {
	for _, entity := range inspection.Entities {
		if entity.ID == entityID {
			return entity, nil
		}
	}
	return EntitySummary{}, fmt.Errorf("entity %q not found", entityID)
}
