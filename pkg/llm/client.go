package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	// ClaudeAPIEndpoint is the Anthropic API endpoint.
	ClaudeAPIEndpoint = "https://api.anthropic.com/v1/messages"
	// ClaudeModel is the model to use.
	ClaudeModel = "claude-sonnet-4-20250514"
	// ClaudeAPIVersion is the API version.
	ClaudeAPIVersion = "2023-06-01"
)

// Client represents a Claude API client.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	endpoint   string
}

// NewClient creates a new Claude API client.
func NewClient(apiKey, model string) (client *Client) {
	if model == "" {
		model = ClaudeModel // Default to Sonnet 4
	}
	client = &Client{
		apiKey:   apiKey,
		model:    model,
		endpoint: ClaudeAPIEndpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	return client
}

// Analyze performs Phase 1: Analyze + Rank.
func (c *Client) Analyze(ctx context.Context, jd string, achievements []map[string]interface{}) (response AnalysisResponse, err error) {
	prompt := buildAnalysisPrompt(jd, achievements)

	var responseText string
	responseText, err = c.sendRequest(ctx, prompt)
	if err != nil {
		err = errors.Wrap(err, "analysis request failed")
		return response, err
	}

	// Clean markdown code fences if present
	cleanedText := stripMarkdownCodeFences(responseText)

	// Parse JSON response
	err = json.Unmarshal([]byte(cleanedText), &response)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse analysis response: %s", responseText)
		return response, err
	}

	return response, err
}

// Generate performs Phase 2: Generate Resume + Cover Letter.
func (c *Client) Generate(ctx context.Context, req GenerationRequest) (response GenerationResponse, err error) {
	prompt := buildGenerationPrompt(req)

	var responseText string
	responseText, err = c.sendRequest(ctx, prompt)
	if err != nil {
		err = errors.Wrap(err, "generation request failed")
		return response, err
	}

	// Clean markdown code fences if present
	cleanedText := stripMarkdownCodeFences(responseText)

	// Parse JSON response
	err = json.Unmarshal([]byte(cleanedText), &response)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse generation response: %s", responseText)
		return response, err
	}

	return response, err
}

// GenerateGeneral generates a comprehensive general resume.
func (c *Client) GenerateGeneral(ctx context.Context, req GeneralResumeRequest) (response GeneralResumeResponse, err error) {
	prompt := buildGeneralResumePrompt(req)

	var responseText string
	responseText, err = c.sendRequest(ctx, prompt)
	if err != nil {
		err = errors.Wrap(err, "general resume generation request failed")
		return response, err
	}

	// Clean markdown code fences if present
	cleanedText := stripMarkdownCodeFences(responseText)

	// Parse JSON response
	err = json.Unmarshal([]byte(cleanedText), &response)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse general resume response: %s", responseText)
		return response, err
	}

	return response, err
}

// sendRequest sends a request to Claude API.
func (c *Client) sendRequest(ctx context.Context, prompt string) (responseText string, err error) {
	// Build request
	claudeReq := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 4096,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	var reqBody []byte
	reqBody, err = json.Marshal(claudeReq)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal request")
		return responseText, err
	}

	// Create HTTP request
	var httpReq *http.Request
	httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		err = errors.Wrap(err, "failed to create HTTP request")
		return responseText, err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", ClaudeAPIVersion)

	// Send request
	var resp *http.Response
	resp, err = c.httpClient.Do(httpReq)
	if err != nil {
		err = errors.Wrap(err, "HTTP request failed")
		return responseText, err
	}
	defer resp.Body.Close()

	// Read response body
	var respBody []byte
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read response body")
		return responseText, err
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
		return responseText, err
	}

	// Parse Claude response
	var claudeResp ClaudeResponse
	err = json.Unmarshal(respBody, &claudeResp)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse Claude response: %s", string(respBody))
		return responseText, err
	}

	// Extract text content
	if len(claudeResp.Content) == 0 {
		err = errors.New("no content in Claude response")
		return responseText, err
	}

	responseText = claudeResp.Content[0].Text

	return responseText, err
}

// stripMarkdownCodeFences removes markdown code fences from JSON responses.
func stripMarkdownCodeFences(text string) (cleaned string) {
	cleaned = text

	// Check if text starts with ```json and ends with ```
	if len(cleaned) > 7 && cleaned[:7] == "```json" {
		// Find first newline after ```json
		start := 7
		for start < len(cleaned) && cleaned[start] != '\n' {
			start++
		}
		start++ // skip the newline

		// Find last ```
		end := len(cleaned)
		if end > 3 && cleaned[end-3:] == "```" {
			end -= 3
		}

		// Remove trailing whitespace before ```
		for end > 0 && (cleaned[end-1] == '\n' || cleaned[end-1] == ' ' || cleaned[end-1] == '\r') {
			end--
		}

		cleaned = cleaned[start:end]
	}

	return cleaned
}
