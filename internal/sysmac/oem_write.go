package sysmac

import (
	"crypto/rand"
	"errors"
	"fmt"
)

func (p *OEMProject) CreateEntity(entity OEMEntity, parentID string) (*OEMEntity, error) {
	if entity.ID == "" || entity.Name == "" || entity.Type == "" {
		return nil, errors.New("entity ID, name, and type are required")
	}
	if _, exists := p.EntitiesByID[entity.ID]; exists {
		return nil, fmt.Errorf("entity %q already exists", entity.ID)
	}
	parent, ok := p.EntitiesByID[parentID]
	if !ok {
		return nil, fmt.Errorf("parent entity %q not found", parentID)
	}
	if siblingNameExists(parent, entity.Name) {
		return nil, fmt.Errorf("entity name %q already exists under parent %q", entity.Name, parentID)
	}
	entity.ParentID = parentID
	entity.Children = nil
	created := &entity
	parent.Children = append(parent.Children, created)
	p.EntitiesByID[created.ID] = created
	return created, nil
}

func (p *OEMProject) RenameEntity(id, name string) error {
	entity, ok := p.EntitiesByID[id]
	if !ok {
		return fmt.Errorf("entity %q not found", id)
	}
	if name == "" {
		return errors.New("entity name is required")
	}
	if entity.ParentID != "" {
		parent := p.EntitiesByID[entity.ParentID]
		if siblingNameExistsExcept(parent, name, id) {
			return fmt.Errorf("entity name %q already exists under parent %q", name, entity.ParentID)
		}
	}
	entity.Name = name
	return nil
}

func (p *OEMProject) MoveEntity(id, parentID string) error {
	entity, ok := p.EntitiesByID[id]
	if !ok {
		return fmt.Errorf("entity %q not found", id)
	}
	parent, ok := p.EntitiesByID[parentID]
	if !ok {
		return fmt.Errorf("parent entity %q not found", parentID)
	}
	if id == p.Root.ID || id == parentID || containsEntity(entity, parentID) {
		return errors.New("entity move would create an invalid hierarchy")
	}
	if entity.ParentID == parentID {
		return nil
	}
	if siblingNameExists(parent, entity.Name) {
		return fmt.Errorf("entity name %q already exists under parent %q", entity.Name, parentID)
	}
	oldParent := p.EntitiesByID[entity.ParentID]
	if oldParent != nil {
		oldParent.Children = removeChild(oldParent.Children, id)
	}
	parent.Children = append(parent.Children, entity)
	entity.ParentID = parentID
	return nil
}

func (p *OEMProject) CloneEntity(id, parentID, name string) (*OEMEntity, error) {
	original, ok := p.EntitiesByID[id]
	if !ok {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if name == "" {
		return nil, errors.New("clone name is required")
	}
	clone := cloneEntityTree(original)
	clone.Name = name
	clone.ParentID = parentID
	parent, ok := p.EntitiesByID[parentID]
	if !ok {
		return nil, fmt.Errorf("parent entity %q not found", parentID)
	}
	if siblingNameExists(parent, name) {
		return nil, fmt.Errorf("entity name %q already exists under parent %q", name, parentID)
	}
	parent.Children = append(parent.Children, clone)
	indexOEMEntities(clone, p.EntitiesByID)
	return clone, nil
}

func (p *OEMProject) DeleteEntity(id string) error {
	entity, ok := p.EntitiesByID[id]
	if !ok {
		return fmt.Errorf("entity %q not found", id)
	}
	if entity == p.Root || entity.ParentID == "" {
		return errors.New("solution root cannot be deleted")
	}
	parent := p.EntitiesByID[entity.ParentID]
	if parent == nil {
		return fmt.Errorf("parent entity %q not found", entity.ParentID)
	}
	parent.Children = removeChild(parent.Children, id)
	deleteEntityIndex(entity, p.EntitiesByID)
	return nil
}

func siblingNameExists(parent *OEMEntity, name string) bool {
	return siblingNameExistsExcept(parent, name, "")
}

func siblingNameExistsExcept(parent *OEMEntity, name, exceptID string) bool {
	if parent == nil {
		return false
	}
	for _, child := range parent.Children {
		if child.ID != exceptID && child.Name == name {
			return true
		}
	}
	return false
}

func containsEntity(root *OEMEntity, id string) bool {
	for _, child := range root.Children {
		if child.ID == id || containsEntity(child, id) {
			return true
		}
	}
	return false
}

func removeChild(children []*OEMEntity, id string) []*OEMEntity {
	result := children[:0]
	for _, child := range children {
		if child.ID != id {
			result = append(result, child)
		}
	}
	return result
}

func deleteEntityIndex(entity *OEMEntity, index map[string]*OEMEntity) {
	delete(index, entity.ID)
	for _, child := range entity.Children {
		deleteEntityIndex(child, index)
	}
}

func cloneEntityTree(source *OEMEntity) *OEMEntity {
	clone := &OEMEntity{Type: source.Type, Subtype: source.Subtype, Name: source.Name, ID: newEntityID(), TrackingID: newEntityID()}
	for _, child := range source.Children {
		childClone := cloneEntityTree(child)
		childClone.ParentID = clone.ID
		clone.Children = append(clone.Children, childClone)
	}
	return clone
}

func newEntityID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	data[6] = (data[6] & 0x0f) | 0x40
	data[8] = (data[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", data[0:4], data[4:6], data[6:8], data[8:10], data[10:])
}
