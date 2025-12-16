package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nikogura/resume-tailor/pkg/rag"
)

// Evaluator is a separate Claude instance for evaluating generated resumes.
type Evaluator struct {
	client *Client
	model  string
}

// NewEvaluator creates a new evaluator instance.
func NewEvaluator(apiKey, model string) (evaluator *Evaluator, err error) {
	if apiKey == "" {
		err = errors.New("ANTHROPIC_API_KEY is required")
		return evaluator, err
	}

	if model == "" {
		model = "claude-sonnet-4-5-20250929" // Default to Sonnet 4.5
	}

	evaluator = &Evaluator{
		client: NewClient(apiKey, model),
		model:  model,
	}

	return evaluator, err
}

// EvaluationRequest contains all data needed for evaluation.
type EvaluationRequest struct {
	Company            string
	Role               string
	JobDescription     string
	Resume             string
	CoverLetter        string
	SourceAchievements string // JSON
	SourceSkills       string // JSON
	SourceProfile      string // JSON
}

// EvaluationResponse is what Claude returns.
type EvaluationResponse struct {
	ResumeViolations      []rag.Violation       `json:"resume_violations"`
	WeakQuantifications   []rag.WeakNumberIssue `json:"weak_quantifications"`
	AccuracyViolations    []rag.Violation       `json:"accuracy_violations"`
	CoverLetterViolations []rag.Violation       `json:"cover_letter_violations"`
	VerifiedMetrics       []string              `json:"verified_metrics"`
	CompanyDatesCorrect   bool                  `json:"company_dates_correct"`
	RoleTitlesCorrect     bool                  `json:"role_titles_correct"`
	YearsExpCorrect       bool                  `json:"years_exp_correct"`
	JDMatch               rag.JDMatch           `json:"jd_match"`
	LessonsLearned        []string              `json:"lessons_learned"`
}

// Evaluate runs the evaluation using Claude.
func (e *Evaluator) Evaluate(ctx context.Context, req EvaluationRequest) (resp EvaluationResponse, err error) {
	prompt := e.buildEvaluationPrompt(req)

	// Call Claude API directly using sendRequest (need to expose it or use a helper)
	// For now, use the same pattern as the client but adapted for evaluation
	responseText, callErr := e.callClaude(ctx, prompt)
	if callErr != nil {
		err = fmt.Errorf("failed to call Claude API: %w", callErr)
		return resp, err
	}

	// Strip markdown code fences if present
	cleanedText := stripMarkdownCodeFences(responseText)

	// Parse JSON response
	err = json.Unmarshal([]byte(cleanedText), &resp)
	if err != nil {
		err = fmt.Errorf("failed to parse evaluation response: %w\nResponse: %s", err, cleanedText)
		return resp, err
	}

	return resp, err
}

// callClaude makes a direct call to Claude API for evaluation.
func (e *Evaluator) callClaude(ctx context.Context, prompt string) (responseText string, err error) {
	// Build Claude API request
	claudeReq := ClaudeRequest{
		Model:     e.model,
		MaxTokens: 16000, // Evaluations need more tokens
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
		err = fmt.Errorf("failed to marshal request: %w", err)
		return responseText, err
	}

	var httpReq *http.Request
	httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, ClaudeAPIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		return responseText, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", e.client.apiKey)
	httpReq.Header.Set("Anthropic-Version", ClaudeAPIVersion)

	var httpResp *http.Response
	httpResp, err = e.client.httpClient.Do(httpReq)
	if err != nil {
		err = fmt.Errorf("HTTP request failed: %w", err)
		return responseText, err
	}
	defer httpResp.Body.Close()

	var respBody []byte
	respBody, err = io.ReadAll(httpResp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response: %w", err)
		return responseText, err
	}

	if httpResp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(respBody))
		return responseText, err
	}

	var claudeResp ClaudeResponse
	err = json.Unmarshal(respBody, &claudeResp)
	if err != nil {
		err = fmt.Errorf("failed to parse response: %w", err)
		return responseText, err
	}

	if len(claudeResp.Content) == 0 {
		err = errors.New("empty response from API")
		return responseText, err
	}

	responseText = claudeResp.Content[0].Text
	return responseText, err
}

func (e *Evaluator) buildEvaluationPrompt(req EvaluationRequest) (prompt string) {
	prompt = fmt.Sprintf(`You are a resume evaluation specialist. Your job is to score generated resumes and cover letters for FACTUAL ACCURACY and compliance with anti-fabrication rules.

CRITICAL: You are NOT the generator. You are the EVALUATOR. Your job is to find problems, not defend the output.

JOB DESCRIPTION:
%s

SOURCE ACHIEVEMENTS (GROUND TRUTH):
%s

SOURCE SKILLS (GROUND TRUTH):
%s

SOURCE PROFILE (GROUND TRUTH):
%s

GENERATED RESUME:
%s

GENERATED COVER LETTER:
%s

YOUR TASK: Evaluate the generated resume and cover letter against these CRITICAL ANTI-FABRICATION RULES:

**RULE 1: FORBIDDEN NUMBER FABRICATION**
Check every number in the resume/cover letter. If a number appears that is NOT in the source achievements' metrics array, it is FABRICATED.
Examples of violations:
- Resume says "managed 70+ engineers" but source has NO team size number
- Resume says "7 distributed clusters" when source only says "distributed clusters"
- Resume says "15 team members" but source has no headcount

**RULE 2: FORBIDDEN INDUSTRY CLAIMS**
Check every industry mentioned. If resume/cover claim "climate-tech", "gaming", "healthcare", "real estate", etc. but source achievement companies have NONE of those industries, it is FABRICATED.
Examples of violations:
- Resume says "climate-tech experience" but all companies are fintech/crypto
- Cover letter says "real estate technology" but source has no real estate companies

**RULE 3: FORBIDDEN TECHNICAL DOMAIN CLAIMS**
Check for domain-specific technical terms. If resume/cover claim "satellite imagery processing", "geospatial analysis", "vegetation risk" etc. but source achievements have ZERO work in those domains, it is FABRICATED.
Examples of violations:
- Resume skills list "Satellite Data Processing" but no satellite work in achievements
- Cover letter says "similar to processing satellite imagery" when source is about security events

**RULE 4: FORBIDDEN PATTERN MATCHING**
Check if resume/cover make claims like "mirrors", "similar to", "translates to" connecting their work to JD domains they don't have.
Examples of violations:
- "This mirrors your need to process satellite imagery" when candidate processed security logs
- "Similar patterns needed for vegetation analysis" when candidate has no vegetation work

**RULE 5: WEAK QUANTIFICATIONS**
Numbers under 10-20 that undermine credibility should be flagged (but are minor violations, not fabrications).
Examples: "7 clusters", "3 regions", "5 team members", "2 weeks"

**RULE 6: ACCURACY CHECKS**
- Years of experience: Must exactly match profile.years_experience (check for "25+ years", "30+ years", etc.)
- Company/Role/Dates: Must exactly match source achievements
- Metrics: Every percentage, dollar amount, must be in source achievements metrics

For EACH violation you find, you MUST provide:
{
  "rule": "FORBIDDEN_NUMBER_FABRICATION",
  "severity": "critical|major|minor",
  "location": "resume.md:line_number or cover.md:line_number",
  "fabricated": "exact text that was fabricated",
  "evidence_checked": "what you checked in source and didn't find",
  "suggested_fix": "how to fix it"
}

Return ONLY valid JSON in this format (no markdown, no commentary):
{
  "resume_violations": [],
  "weak_quantifications": [],
  "accuracy_violations": [],
  "cover_letter_violations": [],
  "verified_metrics": ["list of metrics you verified ARE in source"],
  "company_dates_correct": true|false,
  "role_titles_correct": true|false,
  "years_exp_correct": true|false,
  "jd_match": {
    "matched": ["requirements from JD that candidate meets"],
    "unmatched": ["requirements from JD candidate lacks"],
    "fabrications_to_match": ["things that were fabricated to match JD"]
  },
  "lessons_learned": ["key takeaways about what went wrong"]
}

BE THOROUGH. Check EVERY number, EVERY industry claim, EVERY domain term. Your job is to catch fabrications.`,
		req.JobDescription,
		req.SourceAchievements,
		req.SourceSkills,
		req.SourceProfile,
		req.Resume,
		req.CoverLetter,
	)

	return prompt
}
