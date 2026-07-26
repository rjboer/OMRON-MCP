package sysmac

import (
	"encoding/xml"
	"fmt"
	"os"
)

type OEMProject struct {
	Name         string
	Root         *OEMEntity
	EntitiesByID map[string]*OEMEntity
}

type OEMEntity struct {
	Type       string
	Subtype    string
	Name       string
	ID         string
	TrackingID string
	ParentID   string
	Children   []*OEMEntity
}

type oemXML struct {
	XMLName xml.Name  `xml:"sysmac"`
	Entity  oemEntity `xml:"Entity"`
}

type oemEntity struct {
	Type       string      `xml:"type,attr"`
	Subtype    string      `xml:"subtype,attr"`
	Name       string      `xml:"name,attr"`
	ID         string      `xml:"id,attr"`
	TrackingID string      `xml:"trackingId,attr"`
	Children   oemChildren `xml:"ChildEntities"`
}

type oemChildren struct {
	Entities []oemEntity `xml:"Entity"`
}

func ParseOEM(path string) (OEMProject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OEMProject{}, fmt.Errorf("read OEM index: %w", err)
	}
	var doc oemXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return OEMProject{}, fmt.Errorf("parse OEM index: %w", err)
	}
	root := convertOEMEntity(doc.Entity, "")
	project := OEMProject{Name: root.Name, Root: root, EntitiesByID: make(map[string]*OEMEntity)}
	indexOEMEntities(root, project.EntitiesByID)
	return project, nil
}

func convertOEMEntity(entity oemEntity, parentID string) *OEMEntity {
	result := &OEMEntity{Type: entity.Type, Subtype: entity.Subtype, Name: entity.Name, ID: entity.ID, TrackingID: entity.TrackingID, ParentID: parentID}
	for _, child := range entity.Children.Entities {
		result.Children = append(result.Children, convertOEMEntity(child, entity.ID))
	}
	return result
}

func indexOEMEntities(entity *OEMEntity, index map[string]*OEMEntity) {
	if entity == nil {
		return
	}
	index[entity.ID] = entity
	for _, child := range entity.Children {
		indexOEMEntities(child, index)
	}
}
