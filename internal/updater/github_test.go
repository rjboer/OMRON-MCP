package updater

import "testing"

func TestWindowsAssetAndUpdateDetection(t *testing.T) {
	release := Release{
		TargetCommitish: "abc123",
		Assets:          []Asset{{Name: "omron-mcp-windows-amd64.exe", BrowserDownloadURL: "https://example.test/app.exe", Digest: "sha256:deadbeef"}},
	}
	asset, err := release.WindowsAsset()
	if err != nil || asset.Name != "omron-mcp-windows-amd64.exe" {
		t.Fatalf("WindowsAsset() = %#v, %v", asset, err)
	}
	if !release.HasUpdate("old456") || !release.HasUpdate("dev") || release.HasUpdate("abc123") {
		t.Fatalf("unexpected update detection")
	}
	if got := asset.SHA256(); got != "deadbeef" {
		t.Fatalf("SHA256() = %q", got)
	}
}
