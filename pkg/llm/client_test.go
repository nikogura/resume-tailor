package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	apiKey := "test-api-key"
	model := "claude-sonnet-4-20250514"
	client := NewClient(apiKey, model)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.apiKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, client.apiKey)
	}

	if client.model != model {
		t.Errorf("Expected model '%s', got '%s'", model, client.model)
	}

	if client.endpoint != ClaudeAPIEndpoint {
		t.Errorf("Expected endpoint '%s', got '%s'", ClaudeAPIEndpoint, client.endpoint)
	}

	if client.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
}

func TestAnalyze(t *testing.T) {
	// Create mock analysis response.
	mockResponse := AnalysisResponse{
		JDAnalysis: JDAnalysis{
			CompanyName:     "Acme Corp",
			RoleTitle:       "Senior Engineer",
			KeyRequirements: []string{"Go", "Kubernetes"},
			TechnicalStack:  []string{"Go", "Docker"},
			RoleFocus:       "Platform engineering",
			CompanySignals:  "Fast-growing startup",
		},
		RankedAchievements: []RankedAchievement{
			{
				AchievementID:  "test-1",
				RelevanceScore: 0.9,
				Reasoning:      "Highly relevant",
			},
		},
	}

	// Create test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request.
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Error("Missing or incorrect API key header")
		}

		if r.Header.Get("Anthropic-Version") != ClaudeAPIVersion {
			t.Error("Missing or incorrect API version header")
		}

		// Return mock Claude response.
		responseJSON, _ := json.Marshal(mockResponse)
		claudeResp := ClaudeResponse{
			ID:   "test-id",
			Type: "message",
			Role: "assistant",
			Content: []Content{
				{
					Type: "text",
					Text: string(responseJSON),
				},
			},
			Model: ClaudeModel,
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client pointing to test server.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test Analyze.
	ctx := context.Background()
	achievements := []map[string]interface{}{
		{"id": "test-1", "title": "Test Achievement"},
	}

	response, err := client.Analyze(ctx, "Test job description", achievements)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if response.JDAnalysis.CompanyName != "Acme Corp" {
		t.Errorf("Expected company 'Acme Corp', got '%s'", response.JDAnalysis.CompanyName)
	}

	if len(response.RankedAchievements) != 1 {
		t.Errorf("Expected 1 ranked achievement, got %d", len(response.RankedAchievements))
	}
}

func TestGenerate(t *testing.T) {
	// Create mock generation response.
	mockResponse := GenerationResponse{
		Resume:      "# Test Resume\n\nTest content",
		CoverLetter: "Dear Hiring Manager,\n\nTest letter",
	}

	// Create test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock Claude response.
		responseJSON, _ := json.Marshal(mockResponse)
		claudeResp := ClaudeResponse{
			Content: []Content{
				{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test Generate.
	ctx := context.Background()
	req := GenerationRequest{
		JobDescription: "Test JD",
		Company:        "Test Corp",
		Role:           "Test Role",
		Profile:        map[string]interface{}{"name": "Test User"},
		Achievements:   []map[string]interface{}{{"id": "test-1"}},
		Skills:         map[string]interface{}{"languages": []string{"Go"}},
		Projects:       []map[string]interface{}{{"name": "Test Project"}},
	}

	response, err := client.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(response.Resume, "Test Resume") {
		t.Error("Resume doesn't contain expected content")
	}

	if !strings.Contains(response.CoverLetter, "Dear Hiring Manager") {
		t.Error("Cover letter doesn't contain expected content")
	}
}

func TestGenerateGeneral(t *testing.T) {
	// Create mock general resume response.
	mockResponse := GeneralResumeResponse{
		Resume: "# Test General Resume\n\nComprehensive content",
	}

	// Create test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock Claude response.
		responseJSON, _ := json.Marshal(mockResponse)
		claudeResp := ClaudeResponse{
			Content: []Content{
				{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test GenerateGeneral.
	ctx := context.Background()
	req := GeneralResumeRequest{
		Profile:      map[string]interface{}{"name": "Test User"},
		Achievements: []map[string]interface{}{{"id": "test-1"}},
		Skills:       map[string]interface{}{"languages": []string{"Go"}},
		Projects:     []map[string]interface{}{{"name": "Test Project"}},
	}

	response, err := client.GenerateGeneral(ctx, req)
	if err != nil {
		t.Fatalf("GenerateGeneral failed: %v", err)
	}

	if !strings.Contains(response.Resume, "Test General Resume") {
		t.Error("Resume doesn't contain expected content")
	}
}

func TestAPIError(t *testing.T) {
	// Create test server that returns an error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid request"}`))
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test that Analyze returns error.
	ctx := context.Background()
	_, err := client.Analyze(ctx, "Test JD", []map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for bad request, got nil")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Error should mention status code 400: %v", err)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	// Create test server that returns invalid JSON in content.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeResp := ClaudeResponse{
			Content: []Content{
				{
					Type: "text",
					Text: "not valid json",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test that Analyze returns error.
	ctx := context.Background()
	_, err := client.Analyze(ctx, "Test JD", []map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestEmptyContent(t *testing.T) {
	// Create test server that returns empty content array.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeResp := ClaudeResponse{
			Content: []Content{},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test that Analyze returns error.
	ctx := context.Background()
	_, err := client.Analyze(ctx, "Test JD", []map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}

	if !strings.Contains(err.Error(), "no content") {
		t.Errorf("Error should mention 'no content': %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	// Create test server that delays response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Create context that cancels immediately.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that request is cancelled.
	_, err := client.Analyze(ctx, "Test JD", []map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
}

func TestStripMarkdownCodeFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with json code fence",
			input:    "```json\n{\"test\": \"value\"}\n```",
			expected: "{\"test\": \"value\"}",
		},
		{
			name:     "without code fence",
			input:    "{\"test\": \"value\"}",
			expected: "{\"test\": \"value\"}",
		},
		{
			name:     "with extra whitespace",
			input:    "```json\n{\"test\": \"value\"}\n\n```",
			expected: "{\"test\": \"value\"}",
		},
		{
			name:     "multiline json",
			input:    "```json\n{\n  \"test\": \"value\",\n  \"nested\": {\n    \"key\": \"data\"\n  }\n}\n```",
			expected: "{\n  \"test\": \"value\",\n  \"nested\": {\n    \"key\": \"data\"\n  }\n}",
		},
		{
			name:     "plain text",
			input:    "This is plain text",
			expected: "This is plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeFences(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAnalyzeWithCodeFences(t *testing.T) {
	// Create mock response wrapped in code fences.
	mockResponse := AnalysisResponse{
		JDAnalysis: JDAnalysis{
			CompanyName: "Test Corp",
			RoleTitle:   "Engineer",
		},
		RankedAchievements: []RankedAchievement{
			{
				AchievementID:  "test-1",
				RelevanceScore: 0.8,
			},
		},
	}

	responseJSON, _ := json.Marshal(mockResponse)
	wrappedJSON := "```json\n" + string(responseJSON) + "\n```"

	// Create test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claudeResp := ClaudeResponse{
			Content: []Content{
				{
					Type: "text",
					Text: wrappedJSON,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("test-key", "")
	client.endpoint = server.URL

	// Test that Analyze handles code fences.
	ctx := context.Background()
	response, err := client.Analyze(ctx, "Test JD", []map[string]interface{}{})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if response.JDAnalysis.CompanyName != "Test Corp" {
		t.Errorf("Expected company 'Test Corp', got '%s'", response.JDAnalysis.CompanyName)
	}
}

func TestHTTPClientTimeout(t *testing.T) {
	client := NewClient("test-key", "")

	// Verify timeout is set.
	if client.httpClient.Timeout != 120*time.Second {
		t.Errorf("Expected timeout 120s, got %v", client.httpClient.Timeout)
	}
}

func TestRequestHeaders(t *testing.T) {
	// Create test server that checks headers.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers.
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing Content-Type header")
		}

		if r.Header.Get("X-Api-Key") != "my-api-key" {
			t.Errorf("Expected API key 'my-api-key', got '%s'", r.Header.Get("X-Api-Key"))
		}

		if r.Header.Get("Anthropic-Version") != ClaudeAPIVersion {
			t.Errorf("Expected version '%s', got '%s'", ClaudeAPIVersion, r.Header.Get("Anthropic-Version"))
		}

		// Return minimal valid response.
		claudeResp := ClaudeResponse{
			Content: []Content{
				{
					Type: "text",
					Text: "{}",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	// Create client.
	client := NewClient("my-api-key", "")
	client.endpoint = server.URL

	// Make request - header checks are in server handler.
	ctx := context.Background()
	_, _ = client.Analyze(ctx, "Test", []map[string]interface{}{})
}
