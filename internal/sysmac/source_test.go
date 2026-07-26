package sysmac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadStructuredTextSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.xml")
	content := `<StructuredTextModel xmlns="http://schemas.datacontract.org/2004/07/Omron.Cxap.Modules.StructuredText.Core"><Text>IF Start THEN&#xD;
  Output := TRUE;
END_IF;</Text></StructuredTextModel>`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadStructuredText(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "IF Start THEN\r\n  Output := TRUE;\r\nEND_IF;" {
		t.Fatalf("source = %q", got)
	}
}

func TestParseSLWDVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variables.xml")
	content := `[SLWD version=1.0]
_EN=Variables
+GN=VAR GVT=DefaultGroup
++D=BOOL N=Start IV=False G=VAR Com=StartSignal
++D=DINT N=Pressure G=VAR`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadSLWDVariables(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Start" || got[0].Type != "BOOL" || got[1].Name != "Pressure" {
		t.Fatalf("variables = %#v", got)
	}
}

func TestSLWDRoundTripPreservesUnknownLines(t *testing.T) {
	content := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++X=Unknown PreserveMe\n++D=BOOL N=Start IV=False G=VAR Com=StartSignal\n"
	doc, err := ParseSLWDDocument([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.RenameVariable("Start", "RunRequest"); err != nil {
		t.Fatal(err)
	}
	output, err := MarshalSLWDDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "+X=Unknown PreserveMe") {
		t.Fatalf("unknown line was lost: %q", text)
	}
	if !strings.Contains(text, "N=RunRequest") || strings.Contains(text, "N=Start") {
		t.Fatalf("rename not serialized: %q", text)
	}
}

func TestSLWDVariableCRUD(t *testing.T) {
	content := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++D=BOOL N=Start G=VAR\n"
	doc, err := ParseSLWDDocument([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddVariable(SLWDVariable{Name: "Pressure", Type: "DINT", Group: "VAR"}); err != nil {
		t.Fatal(err)
	}
	if err := doc.UpdateVariable(SLWDVariable{Name: "Pressure", Type: "REAL", Group: "VAR"}); err != nil {
		t.Fatal(err)
	}
	if err := doc.DeleteVariable("Start"); err != nil {
		t.Fatal(err)
	}
	vars := doc.Variables()
	if len(vars) != 1 || vars[0].Name != "Pressure" || vars[0].Type != "REAL" {
		t.Fatalf("variables = %#v", vars)
	}
}

func TestSLWDCommentRoundTripPreservesWhitespaceAndEscapes(t *testing.T) {
	content := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++D=BOOL N=Start G=VAR Com=\"Hydraulic filter pressure \\\"delta\\\" \\\\sensor\"\n"
	doc, err := ParseSLWDDocument([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	want := `Hydraulic filter pressure "delta" \sensor`
	vars := doc.Variables()
	if len(vars) != 1 || vars[0].Comment != want {
		t.Fatalf("parsed comment = %#v, want %q", vars, want)
	}
	serialized, err := MarshalSLWDDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := ParseSLWDDocument(serialized)
	if err != nil {
		t.Fatal(err)
	}
	if got := reparsed.Variables()[0].Comment; got != want {
		t.Fatalf("round-trip comment = %q, want %q; serialized=%q", got, want, serialized)
	}
}

func TestWriteSLWDAtomicCreatesBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variables.slwd")
	original := "[SLWD version=1.0]\n_EN=Variables\n+GN=VAR GVT=DefaultGroup\n++D=BOOL N=Start G=VAR\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseSLWDDocument([]byte(original))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.RenameVariable("Start", "Run"); err != nil {
		t.Fatal(err)
	}
	if err := WriteSLWDAtomic(path, doc, filepath.Join(t.TempDir(), "protected")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".backup"); err != nil {
		t.Fatal(err)
	}
	updated, err := ReadSLWDVariables(path)
	if err != nil || len(updated) != 1 || updated[0].Name != "Run" {
		t.Fatalf("updated = %#v, err=%v", updated, err)
	}
}
