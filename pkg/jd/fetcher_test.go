package jd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchFromFile(t *testing.T) {
	// Create a test file.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "This is a test job description."

	err := os.WriteFile(testFile, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test fetching.
	content, err := fetchFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to fetch from file: %v", err)
	}

	if content != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}
}

func TestFetchFromFileNonexistent(t *testing.T) {
	_, err := fetchFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error fetching nonexistent file, got nil")
	}
}

func TestFetchFromFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(emptyFile, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = fetchFromFile(emptyFile)
	if err == nil {
		t.Error("Expected error fetching empty file, got nil")
	}
}

func TestFetchFromURL(t *testing.T) {
	// Create a test server.
	testContent := "<html><body><h1>Job Title</h1><p>Job description here.</p></body></html>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	ctx := context.Background()
	content, err := fetchFromURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch from URL: %v", err)
	}

	if content == "" {
		t.Error("Expected non-empty content")
	}

	// Should have stripped HTML tags.
	if content == testContent {
		t.Error("Expected HTML to be stripped")
	}
}

func TestFetchFromURL404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := fetchFromURL(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestFetchFromURLTimeout(t *testing.T) {
	// Create a server that takes too long.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		_, _ = w.Write([]byte("too slow"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := fetchFromURL(ctx, server.URL)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestFetchWithContext(t *testing.T) {
	// Test with file path.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Test job description"

	err := os.WriteFile(testFile, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	content, err := FetchWithContext(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if content != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, content)
	}
}

func TestFetchWithContextURL(t *testing.T) {
	// Test with URL.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>Test content</body></html>"))
	}))
	defer server.Close()

	ctx := context.Background()
	content, err := FetchWithContext(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch from URL: %v", err)
	}

	if content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestFetch(t *testing.T) {
	// Test the wrapper function.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("Test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := Fetch(testFile)
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if content != "Test" {
		t.Errorf("Expected 'Test', got '%s'", content)
	}
}

func TestStripBasicHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tags",
			input:    "<p>Hello <strong>world</strong></p>",
			expected: "Hello world",
		},
		{
			name:     "script tags",
			input:    "<p>Text</p><script>alert('hi')</script><p>More</p>",
			expected: "TextMore",
		},
		{
			name:     "style tags",
			input:    "<style>.class{color:red}</style><p>Content</p>",
			expected: "Content",
		},
		{
			name:     "no HTML",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripBasicHTML(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
