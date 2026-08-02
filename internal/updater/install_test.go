package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadAndInstallVerifiesAndReplaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("new binary")) }))
	defer server.Close()
	target := filepath.Join(t.TempDir(), "omron-mcp.exe")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("new binary"))
	if err := DownloadAndInstall(context.Background(), server.Client(), server.URL, target, hex.EncodeToString(sum[:]), nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "new binary" {
		t.Fatalf("target = %q, %v", got, err)
	}
}
