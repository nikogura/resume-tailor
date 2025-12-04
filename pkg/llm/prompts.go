package llm

import (
	"encoding/json"
	"fmt"
)

// buildAnalysisPrompt creates the Phase 1 prompt.
func buildAnalysisPrompt(jd string, achievements []map[string]interface{}) (prompt string) {
	achievementsJSON, _ := json.MarshalIndent(achievements, "", "  ")

	prompt = fmt.Sprintf(`You are an expert career consultant analyzing a job description to identify the most relevant achievements from a candidate's background.

JOB DESCRIPTION:
%s

CANDIDATE ACHIEVEMENTS:
%s

Analyze the job description and:
1. Extract the company name from the job description
2. Extract the role title from the job description
3. Extract key requirements (technical skills, experience, domain expertise)
4. Identify role signals (IC vs leadership, security vs performance focus, platform vs application focus)
5. Score each achievement 0.0-1.0 on relevance to this specific role
6. Provide brief reasoning for each score

Return ONLY valid JSON in this exact format (no markdown, no commentary):
{
  "jd_analysis": {
    "company_name": "extracted company name from JD",
    "role_title": "extracted role title from JD",
    "key_requirements": ["requirement1", "requirement2"],
    "technical_stack": ["tech1", "tech2"],
    "role_focus": "description of role focus",
    "company_signals": "insights about company culture/stage"
  },
  "ranked_achievements": [
    {
      "achievement_id": "achievement-id-here",
      "relevance_score": 0.95,
      "reasoning": "why this is relevant"
    }
  ]
}`, jd, string(achievementsJSON))

	return prompt
}

// buildGenerationPrompt creates the Phase 2 prompt.
func buildGenerationPrompt(req GenerationRequest) (prompt string) {
	achievementsJSON, _ := json.MarshalIndent(req.Achievements, "", "  ")
	profileJSON, _ := json.MarshalIndent(req.Profile, "", "  ")
	skillsJSON, _ := json.MarshalIndent(req.Skills, "", "  ")
	projectsJSON, _ := json.MarshalIndent(req.Projects, "", "  ")

	prompt = fmt.Sprintf(`You are an expert resume writer creating tailored application materials.

JOB DESCRIPTION:
%s

COMPANY: %s
ROLE: %s

CANDIDATE PROFILE:
%s

TOP ACHIEVEMENTS (pre-ranked by relevance):
%s

SKILLS:
%s

OPEN SOURCE PROJECTS:
%s

Generate a tailored resume and cover letter in markdown format.

RESUME REQUIREMENTS:
- Header: Use raw LaTeX centering: \begin{center} on first line, then {\Large\bfseries Name} for centered name, then location, then all links on ONE line using LaTeX href format: \href{url}{GitHub} | \href{url}{LinkedIn} | \href{url}{Website}, then motto in italics, then \end{center}
- Professional summary: 3-5 bullet points highlighting most relevant experience for THIS role (NOT a paragraph)
- CRITICAL: When stating years of experience, use the EXACT number from profile.years_experience field (not "15+" when the candidate has 25+ years)
- Employment history: Top 5-7 most relevant companies with 3-5 bullets each, ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST (2023-Present, then 2022-2023, then 2020-2022, etc.)
- Focus bullets on achievements from the provided list that match JD requirements
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL: Add blank line (\\n\\n) between each bullet point for readability
- CRITICAL: Keep technical details (bare-metal, multi-cloud, specific technologies, architectures) - these are differentiators
- CRITICAL: Generalize organizational language (e.g., "mandatory across all X codebases" → "established organization-wide", "used by X team" → "deployed company-wide")
- Keep achievements professional and externally presentable - describe impact and technical approach without revealing internal politics or structure
- Skills section: Only include skills relevant to this JD, organized by category
- Open source projects: Top 3-5 most relevant, formatted as markdown hyperlinks: **[Project Name](url)** - description

COVER LETTER REQUIREMENTS:
- Opening paragraph: Express genuine interest in role and company
- Body (2-3 paragraphs): Weave specific achievement stories showing you've solved similar problems
- Use the challenge/execution/impact structure from achievements
- Match the JD's language and priorities naturally
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL: Avoid overly internal language - keep stories externally appropriate and professional
- Closing: Clear call to action
- CRITICAL: End with proper letter format: "Sincerely,\\n\\n[Name]" or "Best regards,\\n\\n[Name]" (blank line between closing and name)

TONE: Professional but authentic. Show "I've solved YOUR exact problems before."

Return ONLY valid JSON in this exact format (no markdown, no commentary):
{
  "resume": "# Full Name\\n\\n## Professional Summary\\n...\\n\\n## Experience\\n...",
  "cover_letter": "Dear Hiring Manager,\\n\\n..."
}

CRITICAL: Ensure all JSON strings are properly escaped. Use \\n for newlines, \\" for quotes.`,
		req.JobDescription, req.Company, req.Role,
		string(profileJSON), string(achievementsJSON),
		string(skillsJSON), string(projectsJSON))

	return prompt
}

// buildGeneralResumePrompt creates the prompt for a comprehensive general resume.
func buildGeneralResumePrompt(req GeneralResumeRequest) (prompt string) {
	achievementsJSON, _ := json.MarshalIndent(req.Achievements, "", "  ")
	profileJSON, _ := json.MarshalIndent(req.Profile, "", "  ")
	skillsJSON, _ := json.MarshalIndent(req.Skills, "", "  ")
	projectsJSON, _ := json.MarshalIndent(req.Projects, "", "  ")

	prompt = fmt.Sprintf(`You are an expert resume writer creating a comprehensive general resume.

CANDIDATE PROFILE:
%s

ACHIEVEMENTS:
%s

SKILLS:
%s

OPEN SOURCE PROJECTS:
%s

Generate a comprehensive general resume in markdown format that includes most relevant achievements while staying at or under 3 pages when rendered to PDF.

RESUME REQUIREMENTS:
- Header: Use raw LaTeX centering: \begin{center} on first line, then {\Large\bfseries Name} for centered name, then location, then all links on ONE line using LaTeX href format: \href{url}{GitHub} | \href{url}{LinkedIn} | \href{url}{Website}, then motto in italics, then \end{center}
- Professional summary: 3-5 bullet points highlighting breadth and depth of experience
- CRITICAL: When stating years of experience, use the EXACT number from profile.years_experience field
- Employment history: Include all major roles with 3-5 bullets each showing most impactful achievements, ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST (2023-Present, then 2022-2023, then 2020-2022, etc.)
- Focus on quantifiable achievements and technical depth
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL: Add blank line (\\n\\n) between each bullet point for readability
- CRITICAL: Keep technical details (bare-metal, multi-cloud, specific technologies, architectures) - these are differentiators
- CRITICAL: Generalize organizational language (e.g., "mandatory across all X codebases" → "established organization-wide", "used by X team" → "deployed company-wide")
- Keep achievements professional and externally presentable
- Skills section: Organize by category, include all relevant skills
- Open source projects: Top 5-7 projects, formatted as markdown hyperlinks: **[Project Name](url)** - description
- Target: 3 pages or less when rendered to PDF with standard resume formatting

TONE: Professional and comprehensive. Show breadth and depth of experience.

Return ONLY valid JSON in this exact format (no markdown, no commentary):
{
  "resume": "# Full Name\\n\\n## Professional Summary\\n...\\n\\n## Experience\\n..."
}

CRITICAL: Ensure all JSON strings are properly escaped. Use \\n for newlines, \\" for quotes.`,
		string(profileJSON), string(achievementsJSON),
		string(skillsJSON), string(projectsJSON))

	return prompt
}
