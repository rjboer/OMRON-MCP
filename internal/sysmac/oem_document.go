package sysmac

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type oemNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*oemNode
}

func loadOEMDocument(path string) (*oemNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OEM document: %w", err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *oemNode
	stack := []*oemNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse OEM document: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			node := &oemNode{Name: value.Name, Attrs: append([]xml.Attr(nil), value.Attr...)}
			if len(stack) == 0 {
				root = node
			} else {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, errors.New("invalid OEM XML nesting")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(value)
			}
		}
	}
	if root == nil {
		return nil, errors.New("OEM document is empty")
	}
	return root, nil
}

func saveOEMDocument(path string, root *oemNode, protectedRoot string) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if protectedRoot != "" {
		protected, err := filepath.Abs(protectedRoot)
		if err != nil {
			return err
		}
		if isWithin(protected, resolved) {
			return errors.New("refusing write inside protected Sysmac project root")
		}
	}
	original, err := os.ReadFile(resolved)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(resolved), ".sysmac-oem-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	encoder := xml.NewEncoder(tmp)
	if err := encoder.Encode(root); err != nil {
		tmp.Close()
		return err
	}
	if err := encoder.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(resolved+".backup", original, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, resolved)
}

func attr(node *oemNode, name string) string {
	for _, item := range node.Attrs {
		if item.Name.Local == name {
			return item.Value
		}
	}
	return ""
}

func setAttr(node *oemNode, name, value string) {
	for index := range node.Attrs {
		if node.Attrs[index].Name.Local == name {
			node.Attrs[index].Value = value
			return
		}
	}
	node.Attrs = append(node.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func entityNode(node *oemNode) bool { return node.Name.Local == "Entity" }

func findNode(node *oemNode, id string) *oemNode {
	if entityNode(node) && attr(node, "id") == id {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, id); found != nil {
			return found
		}
	}
	return nil
}

func findParent(node *oemNode, id string) (*oemNode, *oemNode) {
	for _, child := range node.Children {
		if entityNode(child) && attr(child, "id") == id {
			return node, child
		}
		if parent, found := findParent(child, id); found != nil {
			return parent, found
		}
	}
	return nil, nil
}

func findChildEntities(node *oemNode, create bool) *oemNode {
	for _, child := range node.Children {
		if child.Name.Local == "ChildEntities" {
			return child
		}
	}
	if !create {
		return nil
	}
	child := &oemNode{Name: xml.Name{Local: "ChildEntities"}}
	node.Children = append(node.Children, child)
	return child
}

func appendEntity(parent *oemNode, entity *oemNode) {
	findChildEntities(parent, true).Children = append(findChildEntities(parent, true).Children, entity)
}

func removeNode(parent, target *oemNode) bool {
	children := findChildEntities(parent, false)
	if children == nil {
		return false
	}
	for index, child := range children.Children {
		if child == target {
			children.Children = append(children.Children[:index], children.Children[index+1:]...)
			return true
		}
	}
	return false
}

func cloneNode(node *oemNode) *oemNode {
	clone := &oemNode{Name: node.Name, Attrs: append([]xml.Attr(nil), node.Attrs...), Text: node.Text}
	for _, child := range node.Children {
		clone.Children = append(clone.Children, cloneNode(child))
	}
	return clone
}

type OEMMutation struct {
	Operation string
	EntityID  string
	ParentID  string
	Name      string
	Type      string
	Subtype   string
}

func ApplyOEMMutation(path string, mutation OEMMutation, protectedRoot string) (string, error) {
	root, err := loadOEMDocument(path)
	if err != nil {
		return "", err
	}
	entity := findNode(root, mutation.EntityID)
	var changedID = mutation.EntityID
	switch mutation.Operation {
	case "rename":
		if entity == nil {
			return "", fmt.Errorf("OEM entity %q not found", mutation.EntityID)
		}
		if mutation.Name == "" {
			return "", errors.New("entity name is required")
		}
		setAttr(entity, "name", mutation.Name)
	case "create":
		parent := findNode(root, mutation.ParentID)
		if parent == nil {
			return "", fmt.Errorf("parent entity %q not found", mutation.ParentID)
		}
		if mutation.Name == "" || mutation.Type == "" {
			return "", errors.New("entity name and type are required")
		}
		if findNode(root, mutation.EntityID) != nil {
			return "", fmt.Errorf("entity %q already exists", mutation.EntityID)
		}
		created := &oemNode{Name: xml.Name{Local: "Entity"}, Attrs: []xml.Attr{
			{Name: xml.Name{Local: "type"}, Value: mutation.Type},
			{Name: xml.Name{Local: "subtype"}, Value: mutation.Subtype},
			{Name: xml.Name{Local: "name"}, Value: mutation.Name},
			{Name: xml.Name{Local: "id"}, Value: mutation.EntityID},
			{Name: xml.Name{Local: "trackingId"}, Value: newEntityID()},
		}}
		appendEntity(parent, created)
	case "move":
		if entity == nil {
			return "", fmt.Errorf("OEM entity %q not found", mutation.EntityID)
		}
		parent := findNode(root, mutation.ParentID)
		if parent == nil {
			return "", fmt.Errorf("parent entity %q not found", mutation.ParentID)
		}
		oldParent, _ := findParent(root, mutation.EntityID)
		if oldParent == nil || entity == root || entity == parent || containsNode(entity, parent) {
			return "", errors.New("invalid OEM move hierarchy")
		}
		if oldParent != parent {
			removeNode(oldParent, entity)
			appendEntity(parent, entity)
		}
	case "clone":
		if entity == nil {
			return "", fmt.Errorf("OEM entity %q not found", mutation.EntityID)
		}
		parent := findNode(root, mutation.ParentID)
		if parent == nil {
			return "", fmt.Errorf("parent entity %q not found", mutation.ParentID)
		}
		clone := cloneNode(entity)
		resetCloneIDs(clone)
		if mutation.Name != "" {
			setAttr(clone, "name", mutation.Name)
		}
		changedID = attr(clone, "id")
		appendEntity(parent, clone)
	case "delete":
		if entity == nil {
			return "", fmt.Errorf("OEM entity %q not found", mutation.EntityID)
		}
		parent, _ := findParent(root, mutation.EntityID)
		if parent == nil {
			return "", errors.New("OEM solution root cannot be deleted")
		}
		removeNode(parent, entity)
	default:
		return "", fmt.Errorf("unsupported OEM mutation %q", mutation.Operation)
	}
	if err := saveOEMDocument(path, root, protectedRoot); err != nil {
		return "", err
	}
	return changedID, nil
}

func containsNode(root, target *oemNode) bool {
	for _, child := range root.Children {
		if child == target || containsNode(child, target) {
			return true
		}
	}
	return false
}

func resetCloneIDs(node *oemNode) {
	if entityNode(node) {
		setAttr(node, "id", newEntityID())
		setAttr(node, "trackingId", newEntityID())
	}
	for _, child := range node.Children {
		resetCloneIDs(child)
	}
}
