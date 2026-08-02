package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rjboer/omron-mcp/internal/updater"
)

func main() {
	target := flag.String("target", "", "executable to replace")
	assetURL := flag.String("asset-url", "", "release asset URL")
	checksum := flag.String("sha256", "", "expected SHA-256 checksum")
	flag.Parse()
	if *target == "" || *assetURL == "" {
		fmt.Fprintln(os.Stderr, "target and asset-url are required")
		os.Exit(2)
	}
	wait := func() error {
		for range 60 {
			if file, err := os.OpenFile(*target, os.O_WRONLY, 0); err == nil {
				_ = file.Close()
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
		return fmt.Errorf("timed out waiting for %s to close", *target)
	}
	if err := updater.DownloadAndInstall(context.Background(), nil, *assetURL, *target, *checksum, wait); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	command := exec.Command(*target)
	command.Dir = filepath.Dir(*target)
	if err := command.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
