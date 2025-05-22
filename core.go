package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Masterminds/semver/v3"
)

const (
	//githubAPIURL = "https://api.github.com/repos/hasura/ddn/releases"
	releasesURL = "https://gist.githubusercontent.com/shukla2112/7cab141a3eafab4d4565d7347eec9029/raw/bf04120a4cc99ed239dddcef7ff1d7aee82c69fd/releases.json"
	installDir  = ".ddnswitch"
	binName     = "ddn"
)

type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Assets     []Asset `json:"assets"`
	PreRelease bool    `json:"prerelease"`
	Draft      bool    `json:"draft"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func getHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func getInstallDir() (string, error) {
	home, err := getHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, installDir), nil
}

func ensureInstallDir() error {
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(installPath, 0755)
}

// Add a cache for versions to avoid repeated network calls
var (
	versionCache     []Release
	versionCacheMux  sync.RWMutex
	versionCacheTime time.Time
	cacheTTL         = 1 * time.Hour
)

func fetchAvailableVersions() ([]Release, error) {
	// Check cache first
	versionCacheMux.RLock()
	if time.Since(versionCacheTime) < cacheTTL && len(versionCache) > 0 {
		cachedVersions := versionCache
		versionCacheMux.RUnlock()
		return cachedVersions, nil
	}
	versionCacheMux.RUnlock()

	// Set a timeout for the HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", releasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Releases API returned status: %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	// Filter out drafts and pre-releases, then sort by version
	var validReleases []Release
	for _, release := range releases {
		if !release.Draft && !release.PreRelease {
			validReleases = append(validReleases, release)
		}
	}

	// Sort by semantic version (newest first)
	sort.Slice(validReleases, func(i, j int) bool {
		vi, err1 := semver.NewVersion(strings.TrimPrefix(validReleases[i].TagName, "v"))
		vj, err2 := semver.NewVersion(strings.TrimPrefix(validReleases[j].TagName, "v"))
		if err1 != nil || err2 != nil {
			// Fallback to string comparison if semver parsing fails
			return validReleases[i].TagName > validReleases[j].TagName
		}
		return vi.GreaterThan(vj)
	})

	// Update cache
	versionCacheMux.Lock()
	versionCache = validReleases
	versionCacheTime = time.Now()
	versionCacheMux.Unlock()

	return validReleases, nil
}

func listAvailableVersions() error {
	fmt.Println("Fetching available DDN CLI versions...")
	
	releases, err := fetchAvailableVersions()
	if err != nil {
		return err
	}

	fmt.Println("\nAvailable DDN CLI versions:")
	for i, release := range releases {
		current := ""
		if isCurrentVersion(release.TagName) {
			current = " (current)"
		}
		fmt.Printf("%2d. %s%s\n", i+1, release.TagName, current)
	}
	
	return nil
}

func listAndSelectVersion() error {
	fmt.Println("Fetching available DDN CLI versions...")
	
	releases, err := fetchAvailableVersions()
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return fmt.Errorf("no DDN CLI releases found")
	}

	// Prepare options for selection
	var options []string
	for _, release := range releases {
		current := ""
		if isCurrentVersion(release.TagName) {
			current = " (current)"
		}
		options = append(options, fmt.Sprintf("%s%s", release.TagName, current))
	}

	// Interactive selection
	var selectedOption string
	prompt := &survey.Select{
		Message: "Select DDN CLI version to install:",
		Options: options,
	}
	
	if err := survey.AskOne(prompt, &selectedOption); err != nil {
		return err
	}

	// Extract version from selected option
	selectedVersion := strings.Split(selectedOption, " ")[0]
	
	return switchToVersion(selectedVersion)
}

func switchToVersion(version string) error {
	if err := ensureInstallDir(); err != nil {
		return err
	}

	// Check if version is already installed
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}

	versionDir := filepath.Join(installPath, version)
	binPath := filepath.Join(versionDir, binName)

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		fmt.Printf("Version %s not found locally. Installing...\n", version)
		if err := installVersion(version); err != nil {
			return err
		}
	}

	// Create or update symlink
	return createSymlink(binPath)
}

func installVersion(version string) error {
	if err := ensureInstallDir(); err != nil {
		return err
	}

	// Check for unsupported platforms
	osName := runtime.GOOS
	archName := runtime.GOARCH
	
	// ARM-based Linux systems are not supported
	if osName == "linux" && (archName == "arm64" || archName == "arm") {
		return fmt.Errorf("DDN CLI does not support ARM-based Linux systems")
	}
	
	suffix := fmt.Sprintf("-%s-%s", osName, archName)
	
	// Use the same URL pattern as in download_cli.sh
	downloadURL := fmt.Sprintf("https://graphql-engine-cdn.hasura.io/ddn/cli/v4/%s/cli-ddn%s", version, suffix)
	
	fmt.Printf("Downloading DDN CLI %s...\n", version)
	
	// Download the binary
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}

	versionDir := filepath.Join(installPath, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return err
	}

	binPath := filepath.Join(versionDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Download the binary directly
	return downloadBinary(downloadURL, binPath)
}

func downloadBinary(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: status %d", resp.StatusCode)
	}

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create a progress reader
	progressReader := &progressReader{
		reader: resp.Body,
		size:   resp.ContentLength,
	}

	// Copy the binary data to the file
	_, err = io.Copy(outFile, progressReader)
	if err != nil {
		return err
	}

	// Make executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
	}

	fmt.Println("\nInstallation completed successfully!")
	return nil
}

func createSymlink(targetPath string) error {
	// Find a directory in PATH to create symlink
	pathDirs := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	
	var symlinkDir string
	homeDir, _ := getHomeDir()
	
	// Prefer user-writable directories
	preferredDirs := []string{
		filepath.Join(homeDir, "bin"),
		filepath.Join(homeDir, ".local", "bin"),
		"/usr/local/bin",
	}

	for _, preferred := range preferredDirs {
		for _, pathDir := range pathDirs {
			if pathDir == preferred {
				symlinkDir = pathDir
				goto found
			}
		}
	}

	// If no preferred directory found, use first writable directory in PATH
	for _, pathDir := range pathDirs {
		if pathDir == "" || pathDir == "." {
			continue
		}
		
		// Test if directory is writable
		testFile := filepath.Join(pathDir, ".ddnswitch_test")
		if file, err := os.Create(testFile); err == nil {
			file.Close()
			os.Remove(testFile)
			symlinkDir = pathDir
			break
		}
	}

found:
	if symlinkDir == "" {
		// Create ~/bin if no suitable directory found
		symlinkDir = filepath.Join(homeDir, "bin")
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			return err
		}
		fmt.Printf("Created %s directory. Please add it to your PATH.\n", symlinkDir)
	}

	symlinkPath := filepath.Join(symlinkDir, binName)
	if runtime.GOOS == "windows" {
		symlinkPath += ".exe"
	}

	// Remove existing symlink if it exists
	os.Remove(symlinkPath)

	// Create new symlink
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		// On Windows or if symlink fails, try copying the file
		return copyFile(targetPath, symlinkPath)
	}

	fmt.Printf("Switched to DDN CLI version %s\n", filepath.Base(filepath.Dir(targetPath)))
	fmt.Printf("Active binary: %s\n", symlinkPath)
	
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func showCurrentVersion() error {
	// Try to get version from currently active DDN CLI
	cmd := exec.Command("ddn", "--version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("No DDN CLI found in PATH or unable to determine version")
		return nil
	}

	fmt.Printf("Current DDN CLI version: %s", strings.TrimSpace(string(output)))
	return nil
}

func isCurrentVersion(version string) bool {
	cmd := exec.Command("ddn", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	currentVersion := strings.TrimSpace(string(output))
	return strings.Contains(currentVersion, version)
}

func uninstallVersion(version string) error {
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}

	versionDir := filepath.Join(installPath, version)
	
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", version)
	}

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to uninstall version %s: %w", version, err)
	}

	fmt.Printf("Successfully uninstalled DDN CLI version %s\n", version)
	return nil
}

// progressReader implements io.Reader with progress tracking
type progressReader struct {
	reader io.Reader
	size   int64
	read   int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	
	if pr.size > 0 {
		percent := float64(pr.read) / float64(pr.size) * 100
		fmt.Printf("\rProgress: %.1f%%", percent)
	}
	
	return n, err
}
