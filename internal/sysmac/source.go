package sysmac

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type structuredTextModel struct {
	Text string `xml:"Text"`
}

type SLWDVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	InitialValue string `json:"initialValue,omitempty"`
	Group        string `json:"group,omitempty"`
	Comment      string `json:"comment,omitempty"`
}

func ReadStructuredText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read Structured Text: %w", err)
	}
	var model structuredTextModel
	if err := xml.Unmarshal(data, &model); err != nil {
		return "", fmt.Errorf("parse Structured Text: %w", err)
	}
	// Sysmac stores CR as an XML character reference and LF as the literal
	// line separator. Normalize the decoded text to Windows line endings.
	text := strings.ReplaceAll(model.Text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\n", "\r\n"), nil
}

// ReadStructuredTextEntity resolves a Structured Text POU body by entity ID and reads its source.
func ReadStructuredTextEntity(projectRoot, entityID string) (string, error) {
	project, err := loadOEMProject(projectRoot)
	if err != nil {
		return "", err
	}
	entity, ok := project.EntitiesByID[entityID]
	if !ok {
		return "", fmt.Errorf("Sysmac entity %q not found", entityID)
	}
	if entity.Type != "PouBody" || entity.Subtype != "StructuredText" {
		return "", fmt.Errorf("Sysmac entity %q is %s/%s, not a Structured Text body", entityID, entity.Type, entity.Subtype)
	}
	payload, err := findEntityPayload(projectRoot, entityID)
	if err != nil {
		return "", err
	}
	return ReadStructuredText(payload)
}

func ReadSLWDVariables(path string) ([]SLWDVariable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SLWD variables: %w", err)
	}
	document, err := ParseSLWDDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse SLWD variables: %w", err)
	}
	return document.Variables(), nil
}
