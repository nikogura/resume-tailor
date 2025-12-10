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
3. Extract the hiring manager's name if mentioned (leave empty if not found)
4. Extract key requirements (technical skills, experience, domain expertise)
5. Identify role signals (IC vs leadership, security vs performance focus, platform vs application focus)
6. Score each achievement 0.0-1.0 on relevance to this specific role
7. Provide brief reasoning for each score

Return ONLY valid JSON in this exact format (no markdown, no commentary):
{
  "jd_analysis": {
    "company_name": "extracted company name from JD",
    "role_title": "extracted role title from JD",
    "hiring_manager": "hiring manager name if mentioned, empty string otherwise",
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
//
//nolint:funlen // Prompt template with extensive anti-hallucination constraints
func buildGenerationPrompt(req GenerationRequest) (prompt string) {
	achievementsJSON, _ := json.MarshalIndent(req.Achievements, "", "  ")
	profileJSON, _ := json.MarshalIndent(req.Profile, "", "  ")
	skillsJSON, _ := json.MarshalIndent(req.Skills, "", "  ")
	projectsJSON, _ := json.MarshalIndent(req.Projects, "", "  ")
	companyURLsJSON, _ := json.MarshalIndent(req.CompanyURLs, "", "  ")

	contextSection := ""
	if req.CoverLetterContext != "" {
		contextSection = fmt.Sprintf(`
ADDITIONAL CONTEXT FOR COVER LETTER:
%s

`, req.CoverLetterContext)
	}

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

COMPANY URLS:
%s
%s
Generate a tailored resume and cover letter in markdown format.

RESUME REQUIREMENTS:
- Header: Use raw LaTeX centering: \begin{center} on first line, then {\Large\bfseries Name} for centered name, then location, then all links on ONE line using LaTeX href format: \href{url}{GitHub} | \href{url}{LinkedIn} | \href{url}{Website}, then motto using LaTeX \textit{} command (example: \textit{Aut viam inveniam, aut faciam (I will find a way, or I will make one)}), then \end{center}. CRITICAL: Do NOT use markdown asterisks for the motto - use LaTeX \textit{} only.

**CRITICAL - YEARS OF EXPERIENCE - READ THIS FIRST:**
The profile.years_experience field contains the ONLY acceptable number for years of experience. For this candidate, profile.years_experience = 25. You MUST use EXACTLY "25+ years" in the professional summary. NEVER write "30+ years", "over 25 years", "nearly 30 years", "approaching 30 years", or ANY other number. The ONLY acceptable phrases are "25+ years" or "25 years". Examples:
- WRONG: "30+ years of engineering leadership"
- WRONG: "30+ years of technical training"
- WRONG: "over 25 years of experience"
- RIGHT: "25+ years of software engineering"
- RIGHT: "25+ years of infrastructure experience"
This is factual accuracy. Writing any number except 25 is lying on the resume and will cause immediate rejection.

**CRITICAL - COMPANY/ROLE/DATE ACCURACY - READ THIS SECOND:**
Each achievement in the source data has EXACT company name, role title, and dates. You MUST use these EXACTLY as provided. DO NOT mix dates between companies. DO NOT modify role titles. DO NOT extend date ranges. Examples from this candidate's actual data:
- Terrace: "CIO & Director of Infrastructure and Security" | 2023-Present
- Amazon Web Services: "Systems Development Engineer, Senior DevOps Consultant" | 2022-2023
- Orion Labs: "Head of Infrastructure, Principal Engineer" | 2020-2022
- Scribd: "Principal DevSecOps Engineer" | 2018-2020
- Apple: "Lead DevOps Engineer" | 2015-2017
- Stitch Fix: "Sr. DevOps/SRE" | 2017
WRONG: Putting Scribd at 2020-2022 (that was Orion Labs)
WRONG: Putting Orion Labs at 2016-2017 (that was overlapping with Apple)
WRONG: Changing "Sr. DevOps/SRE" to "Senior DevOps Engineer"
RIGHT: Using the EXACT company, role, and dates from the achievement data
Each company-role-date combination is unique and must not be mixed with other companies. This is employment history accuracy and errors constitute resume fraud.

- Professional summary: 3-5 bullet points highlighting most relevant experience for THIS role (NOT a paragraph)
- CRITICAL PROFESSIONAL SUMMARY ANTI-HALLUCINATION: The Professional Summary MUST contain ONLY experience, technologies, frameworks, certifications, and compliance standards that are EXPLICITLY present in the candidate's achievement data, skills data, or profile. DO NOT claim experience with technologies just because they appear in the job description. Examples: If the JD mentions "ISO 27001" or "NIST 800-53" but the candidate data does not, DO NOT claim compliance framework experience. If the JD mentions "Kotlin" but it's not in the skills list, DO NOT claim Kotlin experience. Focus on what the candidate HAS done that's relevant, not what the JD wants. This is a hard requirement for truthfulness.
- CRITICAL SPECIFIC TOOL NAMES: NEVER claim experience with specific product/service names unless they are EXPLICITLY mentioned in the source data. This especially applies to: AWS security services (GuardDuty, AWS Config, Inspector, Security Hub, Macie, Detective, etc.), commercial security tools (Wiz, Snyk, Aqua, Prisma Cloud, Lacework, etc.), monitoring tools (DataDog, New Relic, Splunk, etc.). If the JD mentions "GuardDuty" but it's not in the achievements/skills, DO NOT include it. Use generic descriptions instead: "AWS security services", "cloud security posture management", "vulnerability scanning tools", "commercial observability platforms". You can claim experience with tool CATEGORIES if the candidate has used tools in that category, but NEVER claim specific tool names that aren't in source data.
- CRITICAL WEAK QUANTIFICATIONS: Do NOT generalize single data points into patterns. If the data shows "built team from 0 to 5 engineers" at ONE company, do NOT claim "built and scaled platform engineering teams" (plural) as if it's a pattern. Either use the specific context ("Founded Scribd's Security Team, scaling from 0 to 5 engineers") OR drop the numbers entirely and say "Built and scaled platform engineering teams" without quantification. Weak numbers (under 10) that aren't impressive should be omitted. If you can't make a strong quantitative claim, make a qualitative one instead. NEVER use weak team-building numbers like "0 to 5+ engineers" or "0-5 engineers" in the professional summary - either use no numbers or don't make the claim at all.
- CRITICAL TEMPORAL IMPOSSIBILITY: NEVER claim X years of experience with a specific technology/tool if that tool didn't exist for X years. Example: Do NOT say "25+ years with Terraform" when Terraform was first released in 2014. Instead say "25+ years of infrastructure automation experience with expertise in Terraform" or "Deep expertise in Terraform across multi-cloud environments"
- CRITICAL MISLEADING JUXTAPOSITION: Do NOT combine unrelated achievements in the same sentence in a way that implies false connections. Example: If candidate managed 30,000 servers at Apple (2015-2017, pre-Kubernetes era) and has Kubernetes expertise from later roles, DO NOT write "Expert in Kubernetes with proven track record managing 30,000+ servers" - this falsely implies the servers were managed with Kubernetes. Instead, separate the claims: "Expert in Kubernetes and distributed systems" in one bullet, "Managed global infrastructure of 30,000+ servers for Apple Pay" in another bullet. Each achievement must stand alone with its correct context.

**CRITICAL - NO EMPLOYMENT GAPS:**
You MUST include ALL companies from the candidate's employment history in chronological order to avoid gaps in the timeline. NEVER skip a company entirely, as this creates unexplained gaps in work history that raise red flags with hiring managers. Even if a company's achievements are low-ranked for this specific role, include at least a brief 1-2 bullet entry to maintain timeline continuity. For example, if the candidate has companies at 2023-Present, 2022-2023, 2020-2022, 2018-2020, 2017, 2015-2017, and 2007-2014, ALL must be present in that exact order. Omitting any company (like skipping 2015-2017) creates a suspicious 3-year gap. Include every company, prioritizing more detailed bullets for highly-relevant companies and briefer bullets for less-relevant ones, but NEVER omit any company entirely.

- Employment history: ALL companies with 1-5 bullets each (more bullets for highly relevant roles, fewer for less relevant), ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST (2023-Present, then 2022-2023, then 2020-2022, etc.)
- CRITICAL ROLE TITLES AND DATES: Use the EXACT role title and EXACT dates from the achievement data. Do NOT upgrade, enhance, modify, or extend role titles or dates. If the data says "Sr. DevOps/SRE" for "2017", you MUST use exactly that - NOT "Principal Platform Engineer" or "2017-2018". This is factual accuracy about employment history and any changes constitute resume fraud.
- CRITICAL: Format company names as clickable markdown links using the COMPANY URLS mapping: **[Company Name](url)** | *Role Title* | Dates (e.g., **[Acme Corp](https://acme.example.com)** | *Principal Engineer* | 2023-Present)
- Focus bullets on achievements from the provided list that match JD requirements
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL: Add blank line (\\n\\n) between each bullet point for readability
- CRITICAL: Keep technical details (bare-metal, multi-cloud, specific technologies, architectures) - these are differentiators
- CRITICAL: Generalize organizational language (e.g., "mandatory across all X codebases" → "established organization-wide", "used by X team" → "deployed company-wide")
- Keep achievements professional and externally presentable - describe impact and technical approach without revealing internal politics or structure
- CRITICAL SKILLS ANTI-HALLUCINATION: Skills section MUST contain ONLY skills that are EXPLICITLY listed in the provided SKILLS data above. Before including ANY skill, verify it exists in the skills data. If you cannot find the exact skill name in the provided data, DO NOT include it. Examples: If the data has "Terraform" but not "CloudFormation", only list Terraform. If the JD requires a skill not in the data, omit it entirely from the resume. DO NOT add qualifiers, DO NOT infer related skills, DO NOT extrapolate. This is a hard requirement for compliance and truthfulness.
- Open source projects: Top 3-5 most relevant, formatted as markdown hyperlinks: **[Project Name](url)** - description

COVER LETTER REQUIREMENTS:
- CRITICAL GREETING: If hiring_manager field is provided and not empty, use "Dear [Hiring Manager Name],". If hiring_manager is empty, clean the company name by removing suffixes like "LLC", "Inc", "Inc.", "Corp", "Corporation", "Ltd", "Limited", "Co.", etc. and use "Dear [Cleaned Company Name]," (e.g., "Stormlight Capital LLC" becomes "Dear Stormlight Capital,")
- Opening paragraph: Express genuine interest in role and company
- Body (2-3 paragraphs): Weave specific achievement stories showing you've solved similar problems
- Use the challenge/execution/impact structure from achievements
- Match the JD's language and priorities naturally
- CRITICAL: If additional context is provided, incorporate it naturally into the cover letter to personalize the application
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL ANTI-HALLUCINATION: Do NOT claim activities not explicitly listed in the data such as: conference speaking, presenting, publishing articles, blogging, teaching, mentoring programs, awards, certifications, patents, or any other activities. If the JD mentions these and the candidate data does not, simply DO NOT address them.
- CRITICAL: Do NOT infer or extrapolate experiences from open source projects. Open sourcing code does NOT mean the candidate speaks at conferences, writes blog posts, or does external evangelism unless explicitly stated.
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
		string(skillsJSON), string(projectsJSON),
		string(companyURLsJSON), contextSection)

	return prompt
}

// buildGeneralResumePrompt creates the prompt for a comprehensive general resume.
func buildGeneralResumePrompt(req GeneralResumeRequest) (prompt string) {
	achievementsJSON, _ := json.MarshalIndent(req.Achievements, "", "  ")
	profileJSON, _ := json.MarshalIndent(req.Profile, "", "  ")
	skillsJSON, _ := json.MarshalIndent(req.Skills, "", "  ")
	projectsJSON, _ := json.MarshalIndent(req.Projects, "", "  ")
	companyURLsJSON, _ := json.MarshalIndent(req.CompanyURLs, "", "  ")

	prompt = fmt.Sprintf(`You are an expert resume writer creating a comprehensive general resume.

CANDIDATE PROFILE:
%s

ACHIEVEMENTS:
%s

SKILLS:
%s

OPEN SOURCE PROJECTS:
%s

COMPANY URLS:
%s

Generate a comprehensive general resume in markdown format that includes most relevant achievements while staying at or under 3 pages when rendered to PDF.

RESUME REQUIREMENTS:
- Header: Use raw LaTeX centering: \begin{center} on first line, then {\Large\bfseries Name} for centered name, then location, then all links on ONE line using LaTeX href format: \href{url}{GitHub} | \href{url}{LinkedIn} | \href{url}{Website}, then motto using LaTeX \textit{} command (example: \textit{Aut viam inveniam, aut faciam (I will find a way, or I will make one)}), then \end{center}. CRITICAL: Do NOT use markdown asterisks for the motto - use LaTeX \textit{} only.

**CRITICAL - YEARS OF EXPERIENCE - READ THIS FIRST:**
The profile.years_experience field contains the ONLY acceptable number for years of experience. For this candidate, profile.years_experience = 25. You MUST use EXACTLY "25+ years" in the professional summary. NEVER write "30+ years", "over 25 years", "nearly 30 years", "approaching 30 years", or ANY other number. The ONLY acceptable phrases are "25+ years" or "25 years". Examples:
- WRONG: "30+ years of engineering leadership"
- WRONG: "30+ years of technical training"
- WRONG: "over 25 years of experience"
- RIGHT: "25+ years of software engineering"
- RIGHT: "25+ years of infrastructure experience"
This is factual accuracy. Writing any number except 25 is lying on the resume and will cause immediate rejection.

**CRITICAL - COMPANY/ROLE/DATE ACCURACY - READ THIS SECOND:**
Each achievement in the source data has EXACT company name, role title, and dates. You MUST use these EXACTLY as provided. DO NOT mix dates between companies. DO NOT modify role titles. DO NOT extend date ranges. Examples from this candidate's actual data:
- Terrace: "CIO & Director of Infrastructure and Security" | 2023-Present
- Amazon Web Services: "Systems Development Engineer, Senior DevOps Consultant" | 2022-2023
- Orion Labs: "Head of Infrastructure, Principal Engineer" | 2020-2022
- Scribd: "Principal DevSecOps Engineer" | 2018-2020
- Apple: "Lead DevOps Engineer" | 2015-2017
- Stitch Fix: "Sr. DevOps/SRE" | 2017
WRONG: Putting Scribd at 2020-2022 (that was Orion Labs)
WRONG: Putting Orion Labs at 2016-2017 (that was overlapping with Apple)
WRONG: Changing "Sr. DevOps/SRE" to "Senior DevOps Engineer"
RIGHT: Using the EXACT company, role, and dates from the achievement data
Each company-role-date combination is unique and must not be mixed with other companies. This is employment history accuracy and errors constitute resume fraud.

- Professional summary: 3-5 bullet points highlighting breadth and depth of experience
- CRITICAL PROFESSIONAL SUMMARY ANTI-HALLUCINATION: The Professional Summary MUST contain ONLY experience, technologies, frameworks, certifications, and compliance standards that are EXPLICITLY present in the candidate's achievement data, skills data, or profile. DO NOT invent or infer experience with technologies, compliance frameworks, certifications, or methodologies not in the candidate data. Focus on what the candidate HAS done, not what sounds impressive. This is a hard requirement for truthfulness.
- CRITICAL SPECIFIC TOOL NAMES: NEVER claim experience with specific product/service names unless they are EXPLICITLY mentioned in the source data. This especially applies to: AWS security services (GuardDuty, AWS Config, Inspector, Security Hub, Macie, Detective, etc.), commercial security tools (Wiz, Snyk, Aqua, Prisma Cloud, Lacework, etc.), monitoring tools (DataDog, New Relic, Splunk, etc.). If the JD mentions "GuardDuty" but it's not in the achievements/skills, DO NOT include it. Use generic descriptions instead: "AWS security services", "cloud security posture management", "vulnerability scanning tools", "commercial observability platforms". You can claim experience with tool CATEGORIES if the candidate has used tools in that category, but NEVER claim specific tool names that aren't in source data.
- CRITICAL WEAK QUANTIFICATIONS: Do NOT generalize single data points into patterns. If the data shows "built team from 0 to 5 engineers" at ONE company, do NOT claim "built and scaled platform engineering teams" (plural) as if it's a pattern. Either use the specific context ("Founded Scribd's Security Team, scaling from 0 to 5 engineers") OR drop the numbers entirely and say "Built and scaled platform engineering teams" without quantification. Weak numbers (under 10) that aren't impressive should be omitted. If you can't make a strong quantitative claim, make a qualitative one instead. NEVER use weak team-building numbers like "0 to 5+ engineers" or "0-5 engineers" in the professional summary - either use no numbers or don't make the claim at all.
- CRITICAL: When stating years of experience, use the EXACT number from profile.years_experience field
- CRITICAL TEMPORAL IMPOSSIBILITY: NEVER claim X years of experience with a specific technology/tool if that tool didn't exist for X years. Example: Do NOT say "25+ years with Terraform" when Terraform was first released in 2014. Instead say "25+ years of infrastructure automation experience with expertise in Terraform" or "Deep expertise in Terraform across multi-cloud environments"
- CRITICAL MISLEADING JUXTAPOSITION: Do NOT combine unrelated achievements in the same sentence in a way that implies false connections. Example: If candidate managed 30,000 servers at Apple (2015-2017, pre-Kubernetes era) and has Kubernetes expertise from later roles, DO NOT write "Expert in Kubernetes with proven track record managing 30,000+ servers" - this falsely implies the servers were managed with Kubernetes. Instead, separate the claims: "Expert in Kubernetes and distributed systems" in one bullet, "Managed global infrastructure of 30,000+ servers for Apple Pay" in another bullet. Each achievement must stand alone with its correct context.

**CRITICAL - NO EMPLOYMENT GAPS:**
You MUST include ALL companies from the candidate's employment history in chronological order to avoid gaps in the timeline. NEVER skip a company entirely, as this creates unexplained gaps in work history that raise red flags with hiring managers. For a general resume, every role should be included with appropriate detail. For example, if the candidate has companies at 2023-Present, 2022-2023, 2020-2022, 2018-2020, 2017, 2015-2017, and 2007-2014, ALL must be present in that exact order. Omitting any company (like skipping 2015-2017) creates a suspicious 3-year gap. Include every company to maintain complete employment history.

- Employment history: ALL companies with 3-5 bullets each showing most impactful achievements, ORDERED CHRONOLOGICALLY WITH MOST RECENT FIRST (2023-Present, then 2022-2023, then 2020-2022, etc.)
- CRITICAL ROLE TITLES AND DATES: Use the EXACT role title and EXACT dates from the achievement data. Do NOT upgrade, enhance, modify, or extend role titles or dates. If the data says "Sr. DevOps/SRE" for "2017", you MUST use exactly that - NOT "Principal Platform Engineer" or "2017-2018". This is factual accuracy about employment history and any changes constitute resume fraud.
- CRITICAL: Format company names as clickable markdown links using the COMPANY URLS mapping: **[Company Name](url)** | *Role Title* | Dates (e.g., **[Acme Corp](https://acme.example.com)** | *Principal Engineer* | 2023-Present)
- Focus on quantifiable achievements and technical depth
- CRITICAL: Use ONLY metrics and claims explicitly stated in the achievement data - never fabricate, extrapolate, or infer impact
- CRITICAL: Add blank line (\\n\\n) between each bullet point for readability
- CRITICAL: Keep technical details (bare-metal, multi-cloud, specific technologies, architectures) - these are differentiators
- CRITICAL: Generalize organizational language (e.g., "mandatory across all X codebases" → "established organization-wide", "used by X team" → "deployed company-wide")
- Keep achievements professional and externally presentable
- CRITICAL SKILLS ANTI-HALLUCINATION: Skills section MUST contain ONLY skills that are EXPLICITLY listed in the provided SKILLS data above. Before including ANY skill, verify it exists in the skills data. If you cannot find the exact skill name in the provided data, DO NOT include it. If a skill appears useful but is not in the data, omit it entirely. DO NOT add qualifiers, DO NOT infer related skills, DO NOT extrapolate. This is a hard requirement for compliance and truthfulness.
- Open source projects: Top 5-7 projects, formatted as markdown hyperlinks: **[Project Name](url)** - description
- Target: 3 pages or less when rendered to PDF with standard resume formatting

TONE: Professional and comprehensive. Show breadth and depth of experience.

Return ONLY valid JSON in this exact format (no markdown, no commentary):
{
  "resume": "# Full Name\\n\\n## Professional Summary\\n...\\n\\n## Experience\\n..."
}

CRITICAL: Ensure all JSON strings are properly escaped. Use \\n for newlines, \\" for quotes.`,
		string(profileJSON), string(achievementsJSON),
		string(skillsJSON), string(projectsJSON),
		string(companyURLsJSON))

	return prompt
}
