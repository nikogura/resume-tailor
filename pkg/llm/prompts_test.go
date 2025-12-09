package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildAnalysisPrompt(t *testing.T) {
	jd := "We are looking for a Staff Engineer with Go experience at Acme Corp."
	achievements := []map[string]interface{}{
		{
			"id":      "test-1",
			"company": "Test Corp",
			"role":    "Engineer",
			"title":   "Built API",
		},
	}

	prompt := buildAnalysisPrompt(jd, achievements)

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	// Should contain job description.
	if !strings.Contains(prompt, jd) {
		t.Error("Prompt should contain job description")
	}

	// Should contain achievement data.
	if !strings.Contains(prompt, "test-1") {
		t.Error("Prompt should contain achievement ID")
	}

	// Should request JSON format.
	if !strings.Contains(prompt, "jd_analysis") {
		t.Error("Prompt should specify jd_analysis in response format")
	}

	if !strings.Contains(prompt, "ranked_achievements") {
		t.Error("Prompt should specify ranked_achievements in response format")
	}

	// Should request company extraction.
	if !strings.Contains(prompt, "company name") {
		t.Error("Prompt should request company name extraction")
	}

	// Should request role extraction.
	if !strings.Contains(prompt, "role title") {
		t.Error("Prompt should request role title extraction")
	}
}

func TestBuildAnalysisPromptWithMultipleAchievements(t *testing.T) {
	jd := "Job description here"
	achievements := []map[string]interface{}{
		{"id": "ach-1", "title": "First achievement"},
		{"id": "ach-2", "title": "Second achievement"},
		{"id": "ach-3", "title": "Third achievement"},
	}

	prompt := buildAnalysisPrompt(jd, achievements)

	// Should contain all achievement IDs.
	for _, ach := range achievements {
		id := ach["id"].(string)
		if !strings.Contains(prompt, id) {
			t.Errorf("Prompt should contain achievement ID '%s'", id)
		}
	}
}

func TestBuildGenerationPrompt(t *testing.T) {
	req := GenerationRequest{
		JobDescription: "Looking for a Go engineer at Acme Corp for Senior Engineer role",
		Company:        "Acme Corp",
		Role:           "Senior Engineer",
		Profile: map[string]interface{}{
			"name":             "Test User",
			"years_experience": 10,
		},
		Achievements: []map[string]interface{}{
			{
				"id":      "test-1",
				"title":   "Test Achievement",
				"company": "Test Corp",
			},
		},
		Skills: map[string]interface{}{
			"languages": []string{"Go", "Python"},
		},
		Projects: []map[string]interface{}{
			{
				"name": "Test Project",
				"url":  "https://github.com/test/project",
			},
		},
	}

	prompt := buildGenerationPrompt(req)

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	// Should contain all key elements.
	if !strings.Contains(prompt, req.JobDescription) {
		t.Error("Prompt should contain job description")
	}

	if !strings.Contains(prompt, req.Company) {
		t.Error("Prompt should contain company name")
	}

	if !strings.Contains(prompt, req.Role) {
		t.Error("Prompt should contain role title")
	}

	// Should contain profile data.
	if !strings.Contains(prompt, "Test User") {
		t.Error("Prompt should contain profile name")
	}

	// Should contain achievement data.
	if !strings.Contains(prompt, "test-1") {
		t.Error("Prompt should contain achievement ID")
	}

	// Should contain skills data.
	if !strings.Contains(prompt, "Go") {
		t.Error("Prompt should contain skills")
	}

	// Should contain project data.
	if !strings.Contains(prompt, "Test Project") {
		t.Error("Prompt should contain project name")
	}

	// Should specify resume requirements.
	if !strings.Contains(prompt, "RESUME REQUIREMENTS") {
		t.Error("Prompt should contain resume requirements")
	}

	// Should specify cover letter requirements.
	if !strings.Contains(prompt, "COVER LETTER REQUIREMENTS") {
		t.Error("Prompt should contain cover letter requirements")
	}

	// Should request JSON response.
	if !strings.Contains(prompt, `"resume"`) {
		t.Error("Prompt should specify resume in response format")
	}

	if !strings.Contains(prompt, `"cover_letter"`) {
		t.Error("Prompt should specify cover_letter in response format")
	}

	// Should include critical anti-fabrication rules.
	if !strings.Contains(prompt, "Use ONLY metrics and claims explicitly stated") {
		t.Error("Prompt should include anti-fabrication rule")
	}

	// Should include years_experience rule.
	if !strings.Contains(prompt, "use the EXACT number from profile.years_experience") {
		t.Error("Prompt should include years_experience rule")
	}

	// Should include blank line rule.
	if !strings.Contains(prompt, "Add blank line") {
		t.Error("Prompt should include blank line formatting rule")
	}

	// Should include chronological ordering rule.
	if !strings.Contains(prompt, "ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST") {
		t.Error("Prompt should include chronological ordering rule")
	}
}

func TestBuildGeneralResumePrompt(t *testing.T) {
	req := GeneralResumeRequest{
		Profile: map[string]interface{}{
			"name":             "Test User",
			"years_experience": 15,
		},
		Achievements: []map[string]interface{}{
			{"id": "ach-1", "title": "Achievement 1"},
			{"id": "ach-2", "title": "Achievement 2"},
		},
		Skills: map[string]interface{}{
			"languages": []string{"Go", "Python", "Java"},
			"cloud":     []string{"AWS", "GCP"},
		},
		Projects: []map[string]interface{}{
			{
				"name":        "Project One",
				"url":         "https://github.com/test/one",
				"description": "First project",
			},
			{
				"name":        "Project Two",
				"url":         "https://github.com/test/two",
				"description": "Second project",
			},
		},
	}

	prompt := buildGeneralResumePrompt(req)

	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	// Should contain profile data.
	if !strings.Contains(prompt, "Test User") {
		t.Error("Prompt should contain profile name")
	}

	// Should contain all achievements.
	if !strings.Contains(prompt, "ach-1") {
		t.Error("Prompt should contain first achievement")
	}

	if !strings.Contains(prompt, "ach-2") {
		t.Error("Prompt should contain second achievement")
	}

	// Should contain skills data.
	if !strings.Contains(prompt, "Go") || !strings.Contains(prompt, "Python") {
		t.Error("Prompt should contain skills")
	}

	// Should contain projects data.
	if !strings.Contains(prompt, "Project One") {
		t.Error("Prompt should contain first project")
	}

	if !strings.Contains(prompt, "Project Two") {
		t.Error("Prompt should contain second project")
	}

	// Should specify it's a general resume.
	if !strings.Contains(prompt, "comprehensive general resume") {
		t.Error("Prompt should specify this is a general resume")
	}

	// Should mention 3 pages target.
	if !strings.Contains(prompt, "3 pages") {
		t.Error("Prompt should mention 3 pages target")
	}

	// Should request JSON response with resume only (no cover letter).
	if !strings.Contains(prompt, `"resume"`) {
		t.Error("Prompt should specify resume in response format")
	}

	if strings.Contains(prompt, `"cover_letter"`) {
		t.Error("General resume prompt should not include cover_letter")
	}

	// Should include anti-fabrication rules.
	if !strings.Contains(prompt, "Use ONLY metrics and claims explicitly stated") {
		t.Error("Prompt should include anti-fabrication rule")
	}

	// Should include chronological ordering rule.
	if !strings.Contains(prompt, "ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST") {
		t.Error("Prompt should include chronological ordering rule")
	}
}

func TestBuildAnalysisPromptJSONValidity(t *testing.T) {
	// Test that achievements are properly JSON-encoded.
	achievements := []map[string]interface{}{
		{
			"id":          "test-1",
			"title":       "Test with \"quotes\" and special chars",
			"description": "Line 1\nLine 2",
		},
	}

	prompt := buildAnalysisPrompt("Job description", achievements)

	// Extract the JSON portion (this is a rough check).
	// The achievements should be valid JSON within the prompt.
	if !strings.Contains(prompt, "test-1") {
		t.Error("Prompt should contain achievement ID")
	}

	// Verify the marshaled JSON is present.
	expectedJSON, _ := json.MarshalIndent(achievements, "", "  ")
	if !strings.Contains(prompt, string(expectedJSON)) {
		t.Error("Prompt should contain properly marshaled achievements JSON")
	}
}

func TestBuildGenerationPromptJSONValidity(t *testing.T) {
	// Test that all fields are properly JSON-encoded.
	req := GenerationRequest{
		JobDescription: "Test JD",
		Company:        "Test Corp",
		Role:           "Test Role",
		Profile: map[string]interface{}{
			"name": "Test \"Nickname\" User",
		},
		Achievements: []map[string]interface{}{
			{"id": "test-1"},
		},
		Skills:   map[string]interface{}{"languages": []string{"Go"}},
		Projects: []map[string]interface{}{{"name": "Test"}},
	}

	prompt := buildGenerationPrompt(req)

	// Verify all marshaled JSONs are present.
	profileJSON, _ := json.MarshalIndent(req.Profile, "", "  ")
	achievementsJSON, _ := json.MarshalIndent(req.Achievements, "", "  ")

	if !strings.Contains(prompt, string(profileJSON)) {
		t.Error("Prompt should contain properly marshaled profile JSON")
	}

	if !strings.Contains(prompt, string(achievementsJSON)) {
		t.Error("Prompt should contain properly marshaled achievements JSON")
	}
}

func TestPromptsCriticalRules(t *testing.T) {
	// Verify that all prompts contain critical anti-fabrication rules.
	tests := []struct {
		name       string
		promptFunc func() (prompt string)
		shouldHave []string
	}{
		{
			name: "generation prompt",
			promptFunc: func() (prompt string) {
				prompt = buildGenerationPrompt(GenerationRequest{
					JobDescription: "test",
					Company:        "test",
					Role:           "test",
					Profile:        map[string]interface{}{},
					Achievements:   []map[string]interface{}{},
					Skills:         map[string]interface{}{},
					Projects:       []map[string]interface{}{},
				})
				return prompt
			},
			shouldHave: []string{
				"Use ONLY metrics and claims explicitly stated",
				"never fabricate",
				"Add blank line",
				"use the EXACT number from profile.years_experience",
				"Use the EXACT role title and EXACT dates",
				"ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST",
			},
		},
		{
			name: "general resume prompt",
			promptFunc: func() (prompt string) {
				prompt = buildGeneralResumePrompt(GeneralResumeRequest{
					Profile:      map[string]interface{}{},
					Achievements: []map[string]interface{}{},
					Skills:       map[string]interface{}{},
					Projects:     []map[string]interface{}{},
				})
				return prompt
			},
			shouldHave: []string{
				"Use ONLY metrics and claims explicitly stated",
				"never fabricate",
				"Add blank line",
				"Use the EXACT role title and EXACT dates",
				"use the EXACT number from profile.years_experience",
				"ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := tt.promptFunc()
			for _, rule := range tt.shouldHave {
				if !strings.Contains(prompt, rule) {
					t.Errorf("Prompt missing critical rule: '%s'", rule)
				}
			}
		})
	}
}
