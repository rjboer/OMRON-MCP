package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func DownloadAndInstall(ctx context.Context, client *http.Client, url, target, expectedSHA string, wait func() error) error {
	if client == nil {
		client = http.DefaultClient
	}
	target = filepath.Clean(target)
	temporary := target + ".download"
	if err := download(ctx, client, url, temporary, expectedSHA); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if wait != nil {
		if err := wait(); err != nil {
			_ = os.Remove(temporary)
			return err
		}
	}
	backup := target + ".backup"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(temporary)
		return fmt.Errorf("backup existing executable: %w", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install downloaded executable: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

func download(ctx context.Context, client *http.Client, url, target, expectedSHA string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "omron-mcp-updater")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", response.Status)
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hash), response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if expectedSHA != "" && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimPrefix(expectedSHA, "sha256:")) {
		return fmt.Errorf("download checksum does not match")
	}
	return nil
}
