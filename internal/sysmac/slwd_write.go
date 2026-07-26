package sysmac

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type slwdLine struct {
	raw      string
	variable *SLWDVariable
	deleted  bool
}

// SLWDDocument is a loss-aware editable SLWD source. Lines that are not
// understood by the current parser are retained verbatim.
type SLWDDocument struct {
	lines       []slwdLine
	trailingNew bool
}

func ParseSLWDDocument(data []byte) (SLWDDocument, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	doc := SLWDDocument{trailingNew: strings.HasSuffix(text, "\n")}
	for _, raw := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		line := slwdLine{raw: raw}
		if strings.HasPrefix(strings.TrimSpace(raw), "++D=") || strings.HasPrefix(strings.TrimSpace(raw), "+++D=") {
			variable, err := parseSLWDVariableLine(raw)
			if err != nil {
				return SLWDDocument{}, err
			}
			line.variable = &variable
		}
		doc.lines = append(doc.lines, line)
	}
	return doc, nil
}

func (d SLWDDocument) Variables() []SLWDVariable {
	var result []SLWDVariable
	for _, line := range d.lines {
		if line.deleted || line.variable == nil {
			continue
		}
		result = append(result, *line.variable)
	}
	return result
}

func (d *SLWDDocument) AddVariable(variable SLWDVariable) error {
	if variable.Name == "" || variable.Type == "" {
		return errors.New("SLWD variable name and type are required")
	}
	if d.findVariable(variable.Name) >= 0 {
		return fmt.Errorf("SLWD variable %q already exists", variable.Name)
	}
	if variable.Group == "" {
		variable.Group = "VAR"
	}
	d.lines = append(d.lines, slwdLine{variable: &variable})
	return nil
}

func (d *SLWDDocument) UpdateVariable(variable SLWDVariable) error {
	if variable.Name == "" || variable.Type == "" {
		return errors.New("SLWD variable name and type are required")
	}
	index := d.findVariable(variable.Name)
	if index < 0 {
		return fmt.Errorf("SLWD variable %q not found", variable.Name)
	}
	if variable.Group == "" {
		variable.Group = d.lines[index].variable.Group
	}
	d.lines[index].variable = &variable
	return nil
}

func (d *SLWDDocument) RenameVariable(oldName, newName string) error {
	if newName == "" {
		return errors.New("new SLWD variable name is required")
	}
	index := d.findVariable(oldName)
	if index < 0 {
		return fmt.Errorf("SLWD variable %q not found", oldName)
	}
	if oldName != newName && d.findVariable(newName) >= 0 {
		return fmt.Errorf("SLWD variable %q already exists", newName)
	}
	updated := *d.lines[index].variable
	updated.Name = newName
	d.lines[index].variable = &updated
	return nil
}

func (d *SLWDDocument) DeleteVariable(name string) error {
	index := d.findVariable(name)
	if index < 0 {
		return fmt.Errorf("SLWD variable %q not found", name)
	}
	d.lines[index].deleted = true
	return nil
}

func MarshalSLWDDocument(document SLWDDocument) ([]byte, error) {
	lines := make([]string, 0, len(document.lines))
	for _, line := range document.lines {
		if line.deleted {
			continue
		}
		if line.variable != nil {
			lines = append(lines, formatSLWDVariable(*line.variable))
		} else {
			lines = append(lines, line.raw)
		}
	}
	text := strings.Join(lines, "\n")
	if document.trailingNew {
		text += "\n"
	}
	return []byte(text), nil
}

func (d *SLWDDocument) findVariable(name string) int {
	for index, line := range d.lines {
		if !line.deleted && line.variable != nil && line.variable.Name == name {
			return index
		}
	}
	return -1
}

func parseSLWDVariableLine(raw string) (SLWDVariable, error) {
	var variable SLWDVariable
	fields, err := tokenizeSLWDFields(strings.TrimSpace(raw))
	if err != nil {
		return SLWDVariable{}, fmt.Errorf("invalid SLWD variable line %q: %w", raw, err)
	}
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch strings.TrimLeft(key, "+") {
		case "D":
			variable.Type = value
		case "N":
			variable.Name = value
		case "IV":
			variable.InitialValue = value
		case "G":
			variable.Group = value
		case "Com":
			variable.Comment = value
		}
	}
	if variable.Name == "" || variable.Type == "" {
		return SLWDVariable{}, fmt.Errorf("invalid SLWD variable line %q", raw)
	}
	return variable, nil
}

func formatSLWDVariable(variable SLWDVariable) string {
	line := "++D=" + variable.Type + " N=" + variable.Name
	if variable.InitialValue != "" {
		line += " IV=" + variable.InitialValue
	}
	if variable.Group != "" {
		line += " G=" + variable.Group
	}
	if variable.Comment != "" {
		line += " Com=" + formatSLWDValue(variable.Comment)
	}
	return line
}

func tokenizeSLWDFields(raw string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range raw {
		if escaped {
			if r != '\\' && r != '"' {
				return nil, fmt.Errorf("unsupported escape \\%c", r)
			}
			current.WriteRune(r)
			escaped = false
			continue
		}
		if inQuotes && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && (r == ' ' || r == '\t' || r == '\r' || r == '\n') {
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if escaped {
		return nil, errors.New("dangling escape")
	}
	if inQuotes {
		return nil, errors.New("unterminated quote")
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func formatSLWDValue(value string) string {
	if !strings.ContainsAny(value, " \t\r\n\\\"") {
		return value
	}
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}

func WriteSLWDAtomic(path string, document SLWDDocument, protectedRoot string) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve SLWD path: %w", err)
	}
	if protectedRoot != "" {
		protected, err := filepath.Abs(protectedRoot)
		if err != nil {
			return fmt.Errorf("resolve protected root: %w", err)
		}
		if isWithin(protected, resolved) {
			return errors.New("refusing write inside protected Sysmac project root")
		}
	}
	original, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("read SLWD before write: %w", err)
	}
	updated, err := MarshalSLWDDocument(document)
	if err != nil {
		return err
	}
	backup := resolved + ".backup"
	if err := os.WriteFile(backup, original, 0600); err != nil {
		return fmt.Errorf("write SLWD backup: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(resolved), ".sysmac-slwd-*.tmp")
	if err != nil {
		return fmt.Errorf("create SLWD temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(updated); err != nil {
		tmp.Close()
		return fmt.Errorf("write SLWD temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("flush SLWD temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close SLWD temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, resolved); err != nil {
		return fmt.Errorf("replace SLWD atomically: %w", err)
	}
	return nil
}
