package sysmac

import (
	"os"
	"path/filepath"
	"testing"
)

func transactionFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.oem"), []byte(`<sysmac><Entity type="Solution" name="Demo" id="root" /></sysmac>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.slwd"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestProjectTransactionStagesAndCommitsAtomically(t *testing.T) {
	root := transactionFixture(t)
	tx, err := StartProjectTransaction(root)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := tx.StageFile("source.slwd", []byte("new")); err != nil {
		t.Fatal(err)
	}
	changes, err := tx.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].RelativePath != "source.slwd" {
		t.Fatalf("changes = %#v", changes)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "source.slwd"))
	if err != nil || string(data) != "new" {
		t.Fatalf("committed data = %q, err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "source.slwd.backup")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectTransactionRejectsHashConflict(t *testing.T) {
	root := transactionFixture(t)
	tx, err := StartProjectTransaction(root)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := tx.StageFile("source.slwd", []byte("staged")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.slwd"), []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err == nil {
		t.Fatal("hash conflict was not rejected")
	}
	data, err := os.ReadFile(filepath.Join(root, "source.slwd"))
	if err != nil || string(data) != "external" {
		t.Fatalf("conflicted data = %q, err=%v", data, err)
	}
}

func TestProjectTransactionDeletesFileWithBackup(t *testing.T) {
	root := transactionFixture(t)
	tx, err := StartProjectTransaction(root)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := tx.DeleteFile("source.slwd"); err != nil {
		t.Fatal(err)
	}
	changes, err := tx.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].RelativePath != "source.slwd" || changes[0].BeforeHash == "" || changes[0].AfterHash != "" {
		t.Fatalf("delete changes = %#v", changes)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "source.slwd")); !os.IsNotExist(err) {
		t.Fatalf("deleted source still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "source.slwd.backup")); err != nil {
		t.Fatalf("delete backup missing: %v", err)
	}
}

func TestProjectTransactionRollbackPreservesDeletedFile(t *testing.T) {
	root := transactionFixture(t)
	tx, err := StartProjectTransaction(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.DeleteFile("source.slwd"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "source.slwd"))
	if err != nil || string(data) != "old" {
		t.Fatalf("rollback data = %q, err=%v", data, err)
	}
}
