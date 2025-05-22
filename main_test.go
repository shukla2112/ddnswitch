package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	execCommand = exec.Command
	// getSymlinkPath is already declared in core.go
	// installVersion is already declared in core.go
	// downloadBinary is already declared in core.go
)

func TestGetHomeDir(t *testing.T) {
	home, err := getHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	if home == "" {
		t.Fatal("Home directory is empty")
	}

	// Verify the directory exists
	if _, err := os.Stat(home); os.IsNotExist(err) {
		t.Fatalf("Home directory does not exist: %s", home)
	}
}

func TestGetInstallDir(t *testing.T) {
	installDir, err := getInstallDir()
	if err != nil {
		t.Fatalf("Failed to get install directory: %v", err)
	}

	if installDir == "" {
		t.Fatal("Install directory is empty")
	}

	// Should end with .ddnswitch
	if filepath.Base(installDir) != ".ddnswitch" {
		t.Fatalf("Install directory should end with .ddnswitch, got: %s", installDir)
	}
}

func TestEnsureInstallDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save the original function and restore it after the test
	originalGetInstallDir := getInstallDir
	defer func() {
		getInstallDir = originalGetInstallDir
	}()

	// Create a new variable of function type that can be assigned
	var getInstallDirFunc = func() (string, error) {
		return filepath.Join(tempDir, ".ddnswitch"), nil
	}

	// Assign our test function to the package-level function
	getInstallDir = getInstallDirFunc

	// Test that the function creates the directory structure
	if err := ensureInstallDir(); err != nil {
		t.Fatalf("Failed to ensure install directory: %v", err)
	}

	// Verify the directory was created
	installPath, err := getInstallDir()
	if err != nil {
		t.Fatalf("Failed to get install path: %v", err)
	}

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		t.Fatalf("Install directory was not created: %s", installPath)
	}
}

func TestDownloadBinary(t *testing.T) {
	// Create a test server that serves a mock binary
	testContent := "mock binary content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "ddn")

	// Download the mock binary
	err := downloadBinary(server.URL, destPath)
	if err != nil {
		t.Fatalf("Failed to download binary: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Fatalf("Binary was not downloaded to %s", destPath)
	}

	// Verify the content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Fatalf("Downloaded content doesn't match. Expected: %s, Got: %s", testContent, string(content))
	}

	// Verify permissions on Unix systems
	if runtime.GOOS != "windows" {
		info, err := os.Stat(destPath)
		if err != nil {
			t.Fatalf("Failed to stat downloaded file: %v", err)
		}

		if info.Mode().Perm() != 0755 {
			t.Fatalf("Expected file permissions 0755, got %v", info.Mode().Perm())
		}
	}
}

func TestIsCurrentVersionWithoutDDN(t *testing.T) {
	// Test when DDN CLI is not installed
	result := isCurrentVersion("v2.28.0")
	if result {
		t.Fatal("Should return false when DDN CLI is not available")
	}
}

func TestProgressReader(t *testing.T) {
	// Create a test string
	testData := "Hello, World! This is a test for the progress reader."
	testReader := strings.NewReader(testData)

	// Create progress reader
	pr := &progressReader{
		reader: testReader,
		size:   int64(len(testData)),
	}

	// Read data in chunks
	buffer := make([]byte, 10)
	totalRead := 0

	for {
		n, err := pr.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("Unexpected error: %v", err)
			}
			break
		}
		totalRead += n
	}

	if totalRead != len(testData) {
		t.Fatalf("Expected to read %d bytes, got %d", len(testData), totalRead)
	}

	if pr.read != int64(len(testData)) {
		t.Fatalf("Progress reader should track %d bytes read, got %d", len(testData), pr.read)
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary source file
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "source.txt")
	sourceContent := "This is test content for file copying."

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination
	destDir := t.TempDir()
	destFile := filepath.Join(destDir, "dest.txt")

	// Copy file
	if err := copyFile(sourceFile, destFile); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination file exists and has correct content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(destContent) != sourceContent {
		t.Fatalf("Destination file content mismatch. Expected: %s, Got: %s", sourceContent, string(destContent))
	}

	// Verify permissions are copied
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	destInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if sourceInfo.Mode() != destInfo.Mode() {
		t.Fatalf("File permissions not copied correctly. Source: %v, Dest: %v", sourceInfo.Mode(), destInfo.Mode())
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSwitchToVersion(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save the original functions and restore them after the test
	originalGetInstallDir := getInstallDir
	originalGetSymlinkPath := getSymlinkPath
	defer func() {
		getInstallDir = originalGetInstallDir
		getSymlinkPath = originalGetSymlinkPath
	}()

	// Create a new variable of function type that can be assigned
	getInstallDir = func() (string, error) {
		return tempDir, nil
	}

	// Create a mock symlink path in the temp directory
	symlinkPath := filepath.Join(tempDir, "ddn")
	getSymlinkPath = func() (string, error) {
		return symlinkPath, nil
	}

	// Create a mock version directory and binary
	testVersion := "v2.28.0"
	versionDir := filepath.Join(tempDir, testVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}

	// Create a mock binary that returns the correct version
	binPath := filepath.Join(versionDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Create a mock binary script
	mockBinaryContent := "#!/bin/sh\necho \"DDN CLI Version: " + testVersion + "\"\n"
	if runtime.GOOS == "windows" {
		mockBinaryContent = "@echo off\necho DDN CLI Version: " + testVersion
	}

	if err := os.WriteFile(binPath, []byte(mockBinaryContent), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test switching to the version
	if err := switchToVersion(testVersion); err != nil {
		t.Fatalf("Failed to switch to version: %v", err)
	}

	// Verify the symlink was created
	if _, err := os.Stat(symlinkPath); os.IsNotExist(err) {
		t.Fatalf("Symlink was not created at %s", symlinkPath)
	}

	// Verify the symlink points to the correct binary
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if target != binPath {
			t.Fatalf("Symlink points to %s, expected %s", target, binPath)
		}
	}
}

func TestSwitchToVersionWithIncorrectBinary(t *testing.T) {
	// Skip on Windows as this test relies on shell scripts
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save the original functions and restore them after the test
	originalGetInstallDir := getInstallDir
	originalGetSymlinkPath := getSymlinkPath
	originalInstallVersion := installVersion
	defer func() {
		getInstallDir = originalGetInstallDir
		getSymlinkPath = originalGetSymlinkPath
		installVersion = originalInstallVersion
	}()

	// Create a new variable of function type that can be assigned
	getInstallDir = func() (string, error) {
		return tempDir, nil
	}

	// Create a mock symlink path in the temp directory
	symlinkPath := filepath.Join(tempDir, "ddn")
	getSymlinkPath = func() (string, error) {
		return symlinkPath, nil
	}

	// Mock installVersion to create a correct binary
	installVersionCalled := false
	installVersion = func(version string) error {
		installVersionCalled = true
		versionDir := filepath.Join(tempDir, version)
		if err := os.MkdirAll(versionDir, 0755); err != nil {
			return err
		}
		binPath := filepath.Join(versionDir, binName)
		mockBinaryContent := "#!/bin/sh\necho \"DDN CLI Version: " + version + "\"\n"
		return os.WriteFile(binPath, []byte(mockBinaryContent), 0755)
	}

	// Create a mock version directory and binary with incorrect version
	testVersion := "v2.28.0"
	incorrectVersion := "v2.9.0"
	versionDir := filepath.Join(tempDir, testVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}

	// Create a mock binary that returns the incorrect version
	binPath := filepath.Join(versionDir, binName)
	mockBinaryContent := "#!/bin/sh\necho \"DDN CLI Version: " + incorrectVersion + "\"\n"
	if err := os.WriteFile(binPath, []byte(mockBinaryContent), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test switching to the version
	if err := switchToVersion(testVersion); err != nil {
		t.Fatalf("Failed to switch to version: %v", err)
	}

	// Verify installVersion was called to reinstall the correct version
	if !installVersionCalled {
		t.Fatal("installVersion was not called to reinstall the correct version")
	}

	// Verify the symlink was created
	if _, err := os.Stat(symlinkPath); os.IsNotExist(err) {
		t.Fatalf("Symlink was not created at %s", symlinkPath)
	}

	// Verify the symlink points to the correct binary
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	expectedBinPath := filepath.Join(tempDir, testVersion, binName)
	if target != expectedBinPath {
		t.Fatalf("Symlink points to %s, expected %s", target, expectedBinPath)
	}
}

func TestInstallVersion(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save the original functions and restore them after the test
	originalGetInstallDir := getInstallDir
	originalDownloadBinary := downloadBinary
	defer func() {
		getInstallDir = originalGetInstallDir
		downloadBinary = originalDownloadBinary
	}()

	// Create a new variable of function type that can be assigned
	getInstallDir = func() (string, error) {
		return tempDir, nil
	}

	// Mock downloadBinary to create a mock binary
	downloadBinaryCalled := false
	downloadBinary = func(url, destPath string) error {
		downloadBinaryCalled = true
		
		// Create a mock binary that returns the correct version
		testVersion := "v2.28.0"
		mockBinaryContent := "#!/bin/sh\necho \"DDN CLI Version: " + testVersion + "\"\n"
		if runtime.GOOS == "windows" {
			mockBinaryContent = "@echo off\necho DDN CLI Version: " + testVersion
		}
		
		// Create the directory if it doesn't exist
		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		
		return os.WriteFile(destPath, []byte(mockBinaryContent), 0755)
	}

	// Test installing a version
	testVersion := "v2.28.0"
	if err := installVersion(testVersion); err != nil {
		t.Fatalf("Failed to install version: %v", err)
	}

	// Verify downloadBinary was called
	if !downloadBinaryCalled {
		t.Fatal("downloadBinary was not called")
	}

	// Verify the version directory was created
	versionDir := filepath.Join(tempDir, testVersion)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		t.Fatalf("Version directory was not created at %s", versionDir)
	}

	// Verify the binary was created
	binPath := filepath.Join(versionDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatalf("Binary was not created at %s", binPath)
	}
}

func TestCreateSymlink(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save the original function and restore it after the test
	originalGetSymlinkPath := getSymlinkPath
	defer func() {
		getSymlinkPath = originalGetSymlinkPath
	}()

	// Create a mock symlink path in the temp directory
	symlinkPath := filepath.Join(tempDir, "ddn")
	getSymlinkPath = func() (string, error) {
		return symlinkPath, nil
	}

	// Create a mock target file
	targetPath := filepath.Join(tempDir, "target")
	if err := os.WriteFile(targetPath, []byte("test content"), 0755); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Test creating a symlink
	if err := createSymlink(targetPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Verify the symlink was created
	if _, err := os.Stat(symlinkPath); os.IsNotExist(err) {
		t.Fatalf("Symlink was not created at %s", symlinkPath)
	}

	// Verify the symlink points to the correct file on Unix systems
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if target != targetPath {
			t.Fatalf("Symlink points to %s, expected %s", target, targetPath)
		}
	} else {
		// On Windows, verify the file was copied
		content, err := os.ReadFile(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}
		if string(content) != "test content" {
			t.Fatalf("Copied file has incorrect content: %s", string(content))
		}
	}

	// Test creating a symlink when one already exists
	newTargetPath := filepath.Join(tempDir, "new_target")
	if err := os.WriteFile(newTargetPath, []byte("new test content"), 0755); err != nil {
		t.Fatalf("Failed to create new target file: %v", err)
	}

	if err := createSymlink(newTargetPath); err != nil {
		t.Fatalf("Failed to update symlink: %v", err)
	}

	// Verify the symlink points to the new file on Unix systems
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to read updated symlink: %v", err)
		}
		if target != newTargetPath {
			t.Fatalf("Updated symlink points to %s, expected %s", target, newTargetPath)
		}
	} else {
		// On Windows, verify the file was copied
		content, err := os.ReadFile(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to read updated copied file: %v", err)
		}
		if string(content) != "new test content" {
			t.Fatalf("Updated copied file has incorrect content: %s", string(content))
		}
	}
}

