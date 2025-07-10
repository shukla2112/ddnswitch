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
	//releasesURL = "https://gist.githubusercontent.com/shukla2112/7cab141a3eafab4d4565d7347eec9029/raw/d49a91cc321133ccac15f17ad09f749d6bec37c3/releases.json"
	releasesURL = "https://gist.githubusercontent.com/shukla2112/7cab141a3eafab4d4565d7347eec9029/raw/releases.json"
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

// Define getInstallDir as a variable of function type
var getInstallDir = func() (string, error) {
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
	cachePrerelease  bool // Store whether the cache includes prereleases
)

func fetchAvailableVersions() ([]Release, error) {
	// Check cache first
	versionCacheMux.RLock()
	if time.Since(versionCacheTime) < cacheTTL && len(versionCache) > 0 && cachePrerelease == includePrerelease {
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

	// Filter out drafts and pre-releases (if not included), then sort by version
	var validReleases []Release
	for _, release := range releases {
		if !release.Draft && (includePrerelease || !release.PreRelease) {
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
	cachePrerelease = includePrerelease // Store the prerelease flag state
	versionCacheMux.Unlock()

	return validReleases, nil
}

// Add a debug function to check cache status
func debugCacheStatus() {
	if !debugMode {
		return
	}

	versionCacheMux.RLock()
	defer versionCacheMux.RUnlock()

	cacheAge := time.Since(versionCacheTime)
	cacheValid := cacheAge < cacheTTL && len(versionCache) > 0

	fmt.Printf("\n[DEBUG] Cache status:\n")
	fmt.Printf("  Cache age: %v\n", cacheAge)
	fmt.Printf("  Cache TTL: %v\n", cacheTTL)
	fmt.Printf("  Cache size: %d items\n", len(versionCache))
	fmt.Printf("  Cache includes prereleases: %v\n", cachePrerelease)
	fmt.Printf("  Current prerelease flag: %v\n", includePrerelease)
	fmt.Printf("  Cache valid: %v\n", cacheValid)
}

func listAvailableVersions() error {
	fmt.Println("Fetching available DDN CLI versions...")

	// Uncomment this line for debugging
	// debugCacheStatus()

	releases, err := fetchAvailableVersions()
	if err != nil {
		return err
	}

	// Uncomment this line for debugging
	// debugCacheStatus()

	fmt.Println("\nAvailable DDN CLI versions:")
	for i, release := range releases {
		current := ""
		if isCurrentVersion(release.TagName) {
			current = " (current)"
		}

		prerelease := ""
		if release.PreRelease {
			prerelease = " [pre-release]"
		}

		fmt.Printf("%2d. %s%s%s\n", i+1, release.TagName, prerelease, current)
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

		prerelease := ""
		if release.PreRelease {
			prerelease = " [pre-release]"
		}

		options = append(options, fmt.Sprintf("%s%s%s", release.TagName, prerelease, current))
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
	debugLog("Starting switchToVersion for %s", version)

	if err := ensureInstallDir(); err != nil {
		return err
	}

	// Get install directory
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}
	debugLog("Install directory is %s", installPath)

	// Construct paths
	versionDir := filepath.Join(installPath, version)
	binPath := filepath.Join(versionDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	debugLog("Binary path should be %s", binPath)

	// Check if the version is already installed
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		fmt.Printf("Version %s not found locally. Installing...\n", version)
		if err := installVersion(version); err != nil {
			return fmt.Errorf("failed to install version %s: %w", version, err)
		}
	} else {
		debugLog("Binary exists at %s", binPath)

		// Check if the binary is executable
		if runtime.GOOS != "windows" {
			info, err := os.Stat(binPath)
			if err == nil {
				debugLog("Binary permissions: %v", info.Mode().Perm())
				if info.Mode().Perm()&0111 == 0 {
					debugLog("Binary is not executable, fixing permissions")
					if err := os.Chmod(binPath, 0755); err != nil {
						debugLog("Failed to make binary executable: %v", err)
					}
				}
			}
		}

		// Verify the binary version
		cmd := exec.Command(binPath, "version")
		output, err := cmd.CombinedOutput() // Use CombinedOutput to capture stderr too
		if err != nil {
			debugLog("Failed to execute binary: %v", err)
			debugLog("Command output: %s", string(output))
			fmt.Printf("Reinstalling version %s due to verification failure\n", version)
			if err := installVersion(version); err != nil {
				return fmt.Errorf("failed to reinstall version %s: %w", version, err)
			}
		} else {
			installedVersion := strings.TrimSpace(string(output))
			debugLog("Binary reports version: %s", installedVersion)

			if !strings.Contains(installedVersion, version) {
				debugLog("Version mismatch! Expected %s, got %s", version, installedVersion)
				fmt.Printf("Reinstalling version %s due to version mismatch\n", version)
				if err := installVersion(version); err != nil {
					return fmt.Errorf("failed to reinstall version %s: %w", version, err)
				}
			} else {
				debugLog("Version verification successful")
			}
		}
	}

	// Get the symlink path
	symlinkPath, err := getSymlinkPath()
	if err != nil {
		return fmt.Errorf("failed to determine symlink path: %w", err)
	}
	debugLog("Symlink path is %s", symlinkPath)

	// Check if symlink exists and where it points
	if target, err := os.Readlink(symlinkPath); err == nil {
		debugLog("Current symlink points to %s", target)
		if target == binPath {
			debugLog("Symlink already points to the correct version")
			fmt.Printf("Already using DDN CLI version %s\n", version)
			return nil
		}
	} else {
		debugLog("Failed to read symlink: %v", err)
	}

	// Create or update symlink
	debugLog("Creating symlink from %s to %s", symlinkPath, binPath)
	if err := createSymlink(binPath); err != nil {
		return fmt.Errorf("failed to create symlink for version %s: %w", version, err)
	}

	// Verify the symlink is working correctly
	cmd := exec.Command("ddn", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		debugLog("Failed to execute ddn command: %v", err)
		debugLog("Command output: %s", string(output))
	} else {
		activeVersion := strings.TrimSpace(string(output))
		debugLog("Active ddn reports version: %s", activeVersion)

		if !strings.Contains(activeVersion, version) {
			fmt.Printf("WARNING: Active DDN CLI reports version %s, expected %s\n",
				activeVersion, version)
		} else {
			fmt.Printf("Verified: Active DDN CLI is now version %s\n", version)
		}
	}

	return nil
}

var installVersion = func(version string) error {
	return installVersionImpl(version)
}

func installVersionImpl(version string) error {
	debugLog("Starting installVersion for %s", version)

	if err := ensureInstallDir(); err != nil {
		return err
	}

	// Check for unsupported platforms
	osName := runtime.GOOS
	archName := runtime.GOARCH
	debugLog("Platform: %s, Architecture: %s", osName, archName)

	// ARM-based Linux systems are not supported
	if osName == "linux" && (archName == "arm64" || archName == "arm") {
		return fmt.Errorf("DDN CLI does not support ARM-based Linux systems")
	}

	// Clean up any existing installation for this version
	installPath, err := getInstallDir()
	if err != nil {
		return err
	}

	versionDir := filepath.Join(installPath, version)
	debugLog("Version directory: %s", versionDir)

	if _, err := os.Stat(versionDir); err == nil {
		debugLog("Removing existing version directory")
		if err := os.RemoveAll(versionDir); err != nil {
			debugLog("Failed to remove existing directory: %v", err)
			return fmt.Errorf("failed to clean up existing installation: %w", err)
		}
	}

	debugLog("Creating version directory")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for version %s: %w", version, err)
	}

	suffix := fmt.Sprintf("-%s-%s", osName, archName)
	downloadURL := fmt.Sprintf("https://graphql-engine-cdn.hasura.io/ddn/cli/v4/%s/cli-ddn%s", version, suffix)
	debugLog("Download URL: %s", downloadURL)

	binPath := filepath.Join(versionDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	debugLog("Binary path: %s", binPath)

	// Download the binary directly
	debugLog("Downloading binary")
	if err := downloadBinary(downloadURL, binPath); err != nil {
		return fmt.Errorf("failed to download binary for version %s: %w", version, err)
	}

	// Verify the downloaded binary
	debugLog("Verifying downloaded binary")
	cmd := exec.Command(binPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		debugLog("Failed to execute binary: %v", err)
		debugLog("Command output: %s", string(output))
		return fmt.Errorf("failed to verify downloaded binary: %w", err)
	}

	installedVersion := strings.TrimSpace(string(output))
	debugLog("Binary reports version: %s", installedVersion)

	if !strings.Contains(installedVersion, version) {
		debugLog("Version mismatch! Expected %s, got %s", version, installedVersion)
		return fmt.Errorf("downloaded binary reports version %s, expected %s",
			installedVersion, version)
	}

	fmt.Printf("Successfully installed DDN CLI %s\n", version)
	return nil
}

var downloadBinary = func(url, destPath string) error {
	return downloadBinaryImpl(url, destPath)
}

func downloadBinaryImpl(url, destPath string) error {
	fmt.Printf("Downloading from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP status %d", resp.StatusCode)
	}

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
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
		return fmt.Errorf("failed to write binary data: %w", err)
	}

	// Make executable on Unix systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
	}

	fmt.Println("\nDownload completed successfully!")
	return nil
}

func createSymlink(targetPath string) error {
	debugLog("Creating symlink to %s", targetPath)

	symlinkPath, err := getSymlinkPath()
	if err != nil {
		return err
	}

	debugLog("Symlink path: %s", symlinkPath)

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		debugLog("Removing existing symlink or file")
		if err := os.Remove(symlinkPath); err != nil {
			debugLog("Failed to remove existing symlink: %v", err)
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	debugLog("Creating new symlink")
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		debugLog("Failed to create symlink: %v", err)
		// On Windows or if symlink fails, try copying the file
		debugLog("Falling back to file copy")
		return copyFile(targetPath, symlinkPath)
	}

	// Verify the symlink was created correctly
	if target, err := os.Readlink(symlinkPath); err == nil {
		debugLog("Symlink created, points to %s", target)
		if target != targetPath {
			debugLog("WARNING: Symlink points to %s, expected %s", target, targetPath)
		}
	} else {
		debugLog("Failed to read created symlink: %v", err)
	}

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

// Helper function to get the symlink path
var getSymlinkPath = func() (string, error) {
	homeDir, err := getHomeDir()
	if err != nil {
		return "", err
	}
	return getSymlinkPathImpl(homeDir)
}

func getSymlinkPathImpl(homeDir string) (string, error) {
	// Find a directory in PATH to create symlink
	pathDirs := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	var symlinkDir string
	preferredDirs := []string{
		filepath.Join(homeDir, "bin"),
		filepath.Join(homeDir, ".local", "bin"),
		"/usr/local/bin",
	}

	for _, preferred := range preferredDirs {
		for _, pathDir := range pathDirs {
			if pathDir == preferred {
				symlinkDir = pathDir
				debugLog("Found preferred directory in PATH: %s", preferred)
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
			debugLog("Found writable directory in PATH: %s", pathDir)
			break
		}
	}

found:
	if symlinkDir == "" {
		// Create ~/bin if no suitable directory found
		symlinkDir = filepath.Join(homeDir, "bin")
		debugLog("No suitable directory found in PATH, using %s", symlinkDir)
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			return "", err
		}
	}

	symlinkPath := filepath.Join(symlinkDir, binName)
	if runtime.GOOS == "windows" {
		symlinkPath += ".exe"
	}

	return symlinkPath, nil
}
