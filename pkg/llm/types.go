package llm

// AnalysisRequest represents Phase 1: Analyze + Rank request.
type AnalysisRequest struct {
	JobDescription string                   `json:"job_description"`
	Achievements   []map[string]interface{} `json:"achievements"`
}

// AnalysisResponse represents Phase 1: Analyze + Rank response.
type AnalysisResponse struct {
	JDAnalysis         JDAnalysis          `json:"jd_analysis"`
	RankedAchievements []RankedAchievement `json:"ranked_achievements"`
}

// JDAnalysis represents extracted insights from job description.
type JDAnalysis struct {
	CompanyName     string   `json:"company_name"`
	RoleTitle       string   `json:"role_title"`
	HiringManager   string   `json:"hiring_manager,omitempty"`
	KeyRequirements []string `json:"key_requirements"`
	TechnicalStack  []string `json:"technical_stack"`
	RoleFocus       string   `json:"role_focus"`
	CompanySignals  string   `json:"company_signals"`
}

// RankedAchievement represents an achievement with relevance score.
type RankedAchievement struct {
	AchievementID  string  `json:"achievement_id"`
	RelevanceScore float64 `json:"relevance_score"`
	Reasoning      string  `json:"reasoning"`
}

// GenerationRequest represents Phase 2: Generate request.
type GenerationRequest struct {
	JobDescription     string                   `json:"job_description"`
	Company            string                   `json:"company"`
	Role               string                   `json:"role"`
	HiringManager      string                   `json:"hiring_manager,omitempty"`
	JDSummary          string                   `json:"jd_summary"`
	CoverLetterContext string                   `json:"cover_letter_context,omitempty"`
	Achievements       []map[string]interface{} `json:"achievements"`
	Profile            map[string]interface{}   `json:"profile"`
	Skills             map[string]interface{}   `json:"skills"`
	Projects           []map[string]interface{} `json:"projects"`
}

// GenerationResponse represents Phase 2: Generate response.
type GenerationResponse struct {
	Resume      string `json:"resume"`
	CoverLetter string `json:"cover_letter"`
}

// GeneralResumeRequest represents a request to generate a comprehensive general resume.
type GeneralResumeRequest struct {
	Achievements []map[string]interface{} `json:"achievements"`
	Profile      map[string]interface{}   `json:"profile"`
	Skills       map[string]interface{}   `json:"skills"`
	Projects     []map[string]interface{} `json:"projects"`
}

// GeneralResumeResponse represents the response for a general resume.
type GeneralResumeResponse struct {
	Resume string `json:"resume"`
}

// ClaudeRequest represents the Claude API request format.
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// ClaudeResponse represents the Claude API response format.
type ClaudeResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Role    string    `json:"role"`
	Content []Content `json:"content"`
	Model   string    `json:"model"`
	Usage   Usage     `json:"usage"`
}

// Message represents a message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Content represents content in the response.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
