package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const DefaultRepository = "rjboer/OMRON-MCP"

type Release struct {
	Name            string  `json:"name"`
	TagName         string  `json:"tag_name"`
	TargetCommitish string  `json:"target_commitish"`
	Body            string  `json:"body"`
	Assets          []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

func Latest(ctx context.Context, client *http.Client, repository string) (Release, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(repository) == "" {
		repository = DefaultRepository
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/releases/tags/continuous", nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "omron-mcp-updater")
	response, err := client.Do(request)
	if err != nil {
		return Release{}, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return Release{}, fmt.Errorf("GitHub release request failed: %s", response.Status)
	}
	var release Release
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		response.Body.Close()
		return Release{}, err
	}
	response.Body.Close()
	if release.TagName != "" {
		commit, err := tagCommit(ctx, client, repository, release.TagName)
		if err == nil {
			release.TargetCommitish = commit
		} else if strings.TrimSpace(release.TargetCommitish) == "" {
			return Release{}, err
		}
	}
	return release, nil
}

func tagCommit(ctx context.Context, client *http.Client, repository, tag string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/git/ref/tags/"+url.PathEscape(tag), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "omron-mcp-updater")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub tag request failed: %s", response.Status)
	}
	var reference struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := json.NewDecoder(response.Body).Decode(&reference); err != nil {
		return "", err
	}
	if strings.TrimSpace(reference.Object.SHA) == "" {
		return "", errors.New("GitHub tag did not resolve to a commit")
	}
	return reference.Object.SHA, nil
}

func (release Release) WindowsAsset() (Asset, error) {
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, "omron-mcp-windows-amd64.exe") {
			return asset, nil
		}
	}
	return Asset{}, errors.New("Windows AMD64 release asset was not found")
}

func (release Release) HasUpdate(currentCommit string) bool {
	currentCommit = strings.TrimSpace(currentCommit)
	return currentCommit == "dev" || (currentCommit != "" && !strings.EqualFold(currentCommit, strings.TrimSpace(release.TargetCommitish)))
}

func (asset Asset) SHA256() string {
	return strings.TrimPrefix(strings.TrimSpace(asset.Digest), "sha256:")
}
