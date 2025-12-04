package jd

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Fetch retrieves job description from file or URL.
func Fetch(input string) (content string, err error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	content, err = FetchWithContext(ctx, input)
	return content, err
}

// FetchWithContext retrieves job description with context.
func FetchWithContext(ctx context.Context, input string) (content string, err error) {
	// Check if input is a URL
	parsedURL, urlErr := url.Parse(input)
	if urlErr == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
		// It's a URL - fetch via HTTP
		content, err = fetchFromURL(ctx, input)
		if err != nil {
			err = errors.Wrapf(err, "failed to fetch JD from URL: %s", input)
			return content, err
		}
		return content, err
	}

	// It's a file path - read from disk
	content, err = fetchFromFile(input)
	if err != nil {
		err = errors.Wrapf(err, "failed to fetch JD from file: %s", input)
		return content, err
	}

	return content, err
}

// fetchFromFile reads job description from a file.
func fetchFromFile(path string) (content string, err error) {
	var data []byte
	data, err = os.ReadFile(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to read file: %s", path)
		return content, err
	}

	content = string(data)
	if content == "" {
		err = errors.New("file is empty")
		return content, err
	}

	return content, err
}

// fetchFromURL retrieves job description from a URL.
func fetchFromURL(ctx context.Context, urlStr string) (content string, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create HTTP request")
		return content, err
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "resume-tailor/1.0")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "HTTP request failed")
		return content, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("HTTP request failed with status: %d", resp.StatusCode)
		return content, err
	}

	// Read response body
	var bodyBytes []byte
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read response body")
		return content, err
	}

	content = string(bodyBytes)

	// Basic HTML stripping (simple approach - could be enhanced)
	content = stripBasicHTML(content)

	if content == "" {
		err = errors.New("fetched content is empty after processing")
		return content, err
	}

	return content, err
}

// stripBasicHTML removes basic HTML tags (simple implementation).
func stripBasicHTML(html string) (text string) {
	text = html

	// Remove script and style tags with their content
	text = removeTagAndContent(text, "script")
	text = removeTagAndContent(text, "style")

	// Remove HTML tags
	inTag := false
	result := strings.Builder{}
	for _, char := range text {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	text = result.String()

	// Clean up whitespace
	text = strings.TrimSpace(text)

	return text
}

// removeTagAndContent removes a specific HTML tag and its content.
func removeTagAndContent(html, tag string) (result string) {
	result = html
	openTag := "<" + tag
	closeTag := "</" + tag + ">"

	for {
		startIdx := strings.Index(result, openTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(result[startIdx:], closeTag)
		if endIdx == -1 {
			break
		}

		endIdx += startIdx + len(closeTag)
		result = result[:startIdx] + result[endIdx:]
	}

	return result
}
