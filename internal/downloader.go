package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/DaniilSokolyuk/sing-vnet/ut/shell"
)

type Release struct {
	TagName    string  `json:"tag_name"`
	Assets     []Asset `json:"assets"`
	Prerelease bool    `json:"prerelease"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) DownloadLatestSingBoxIfNotExists() (string, error) {
	executableName := "sing-box"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}

	// Get latest release info
	resp, err := http.Get("https://api.github.com/repos/SagerNet/sing-box/releases")
	if err != nil {
		return "", fmt.Errorf("failed to get releases: %v", err)
	}
	defer resp.Body.Close()

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to decode releases: %v", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	var stableRelease *Release
	for _, release := range releases {
		if app.Cfg.Sing.ForceVersion != "" {
			if release.TagName == app.Cfg.Sing.ForceVersion {
				stableRelease = &release
				break
			}

			continue
		}

		if !release.Prerelease {
			stableRelease = &release
			break
		}
	}

	if stableRelease == nil {
		return "", fmt.Errorf("no stable release found")
	}

	versionDir := fmt.Sprintf("sing-box-%s-%s-%s",
		strings.TrimPrefix(stableRelease.TagName, "v"),
		runtime.GOOS,
		runtime.GOARCH)

	if _, err := os.Stat(versionDir); err == nil {
		return filepath.Join(versionDir, executableName), nil // Directory exists
	}

	// Determine required asset name pattern
	assetPattern := versionDir
	if runtime.GOOS == "windows" {
		assetPattern += ".zip"
	} else {
		assetPattern += ".tar.gz"
	}

	// Find matching asset
	var downloadURL string
	for _, asset := range stableRelease.Assets {
		if strings.Contains(asset.Name, assetPattern) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("no matching asset found for %s", assetPattern)
	}

	// Download the asset
	resp, err = http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download asset: %v", err)
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile := filepath.Join(os.TempDir(), filepath.Base(downloadURL))
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer f.Close()

	// Save to temporary file
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save download: %v", err)
	}

	// Extract the archive
	var extractCmd string
	var extractArgs []string

	if runtime.GOOS == "windows" {
		// Use PowerShell to extract ZIP
		extractCmd = "powershell"
		extractArgs = []string{
			"-command",
			fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath .", tmpFile),
		}
	} else {
		// Use tar for Unix-like systems
		extractCmd = "tar"
		extractArgs = []string{
			"xzf",
			tmpFile,
		}
	}

	cmd := exec.Command(extractCmd, extractArgs...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract archive: %v", err)
	}

	// Clean up
	os.Remove(tmpFile)

	execPath := filepath.Join(versionDir, executableName)

	err = chmod(versionDir, execPath)
	if err != nil {
		return "", err
	}

	deleteFromQuarantine(execPath)

	return execPath, nil
}

func chmod(versionDir string, execPath string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	if err := os.Chmod(versionDir, 0777); err != nil {
		return fmt.Errorf("failed to set folder permissions: %v", err)
	}

	cmd := shell.Exec("chmod", "+x", execPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to chmod +x: %v", err)
	}

	return nil
}

func deleteFromQuarantine(execPath string) {
	if runtime.GOOS != "darwin" {
		return
	}

	_ = shell.Exec("xattr", "-d", "com.apple.quarantine", execPath).Run()

	return
}
