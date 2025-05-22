package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	result := isCurrentVersion("v3.0.1")
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
