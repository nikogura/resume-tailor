package renderer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	testContent := "# Test Markdown\n\nThis is a test."

	err := WriteMarkdown(testContent, testFile)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	// Verify file exists.
	_, err = os.Stat(testFile)
	if os.IsNotExist(err) {
		t.Error("Markdown file was not created")
	}

	// Verify content.
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(data))
	}
}

func TestWriteMarkdownCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "test.md")

	err := WriteMarkdown("test", nestedPath)
	if err != nil {
		t.Fatalf("Failed to write markdown: %v", err)
	}

	// Verify file exists.
	_, err = os.Stat(nestedPath)
	if os.IsNotExist(err) {
		t.Error("Markdown file was not created in nested directory")
	}
}

func TestCleanupMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.md")
	testFile2 := filepath.Join(tmpDir, "test2.md")

	// Create files.
	err := os.WriteFile(testFile1, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.WriteFile(testFile2, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup.
	err = CleanupMarkdown(testFile1, testFile2)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Verify files are gone.
	_, err = os.Stat(testFile1)
	if !os.IsNotExist(err) {
		t.Error("File 1 was not deleted")
	}

	_, err = os.Stat(testFile2)
	if !os.IsNotExist(err) {
		t.Error("File 2 was not deleted")
	}
}

func TestCleanupMarkdownNonexistent(t *testing.T) {
	err := CleanupMarkdown("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error cleaning up nonexistent file, got nil")
	}
}

func TestValidateFiles(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")

	err := os.WriteFile(existingFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with existing file.
	err = validateFiles(existingFile)
	if err != nil {
		t.Errorf("Expected no error for existing file, got %v", err)
	}

	// Test with nonexistent file.
	err = validateFiles("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	// Test with multiple files.
	err = validateFiles(existingFile, "/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error when one file doesn't exist, got nil")
	}
}

func TestCheckPandocExists(t *testing.T) {
	// This test will pass if pandoc is installed, skip otherwise.
	err := checkPandocExists()
	if err != nil {
		t.Skip("Pandoc not installed, skipping test")
	}
}
