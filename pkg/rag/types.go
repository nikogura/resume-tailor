package rag

import "time"

// Evaluation represents a complete evaluation of a generated resume and cover letter.
type Evaluation struct {
	Company     string    `json:"company"`
	Role        string    `json:"role"`
	GeneratedAt time.Time `json:"generated_at"`
	EvaluatedAt time.Time `json:"evaluated_at"`
	Scores      Scores    `json:"scores"`
	JDMatch     JDMatch   `json:"jd_requirements"`
	Lessons     []string  `json:"lessons_learned"`
	RAGContext  string    `json:"rag_context"`
	Version     string    `json:"version"` // resume-tailor version
}

// Scores contains all scoring categories.
type Scores struct {
	Resume      ResumeScore      `json:"resume"`
	CoverLetter CoverLetterScore `json:"cover_letter"`
	Overall     int              `json:"overall"` // Weighted average
}

// ResumeScore contains resume-specific scoring.
type ResumeScore struct {
	Total               int                      `json:"total"`
	AntiFabrication     AntiFabricationScore     `json:"anti_fabrication"`
	WeakQuantifications WeakQuantificationsScore `json:"weak_quantifications"`
	Accuracy            AccuracyScore            `json:"accuracy"`
}

// CoverLetterScore contains cover letter-specific scoring.
type CoverLetterScore struct {
	Total        int               `json:"total"`
	DomainClaims DomainClaimsScore `json:"domain_claims"`
	Tone         ToneScore         `json:"tone"`
}

// AntiFabricationScore tracks fabrication violations.
type AntiFabricationScore struct {
	Score      int         `json:"score"`
	Violations []Violation `json:"violations"`
}

// WeakQuantificationsScore tracks weak numbers.
type WeakQuantificationsScore struct {
	Score  int               `json:"score"`
	Issues []WeakNumberIssue `json:"issues"`
}

// AccuracyScore tracks factual accuracy.
type AccuracyScore struct {
	Score               int      `json:"score"`
	VerifiedMetrics     []string `json:"verified_metrics"`
	CompanyDatesCorrect bool     `json:"company_dates_correct"`
	RoleTitlesCorrect   bool     `json:"role_titles_correct"`
	YearsExpCorrect     bool     `json:"years_exp_correct"`
}

// DomainClaimsScore tracks domain expertise claims.
type DomainClaimsScore struct {
	Score      int         `json:"score"`
	Violations []Violation `json:"violations"`
}

// ToneScore tracks cover letter tone quality.
type ToneScore struct {
	Score    int      `json:"score"`
	Feedback []string `json:"feedback"`
}

// Violation represents a rule violation.
type Violation struct {
	Rule            string `json:"rule"`
	Severity        string `json:"severity"` // critical, major, minor
	Location        string `json:"location"` // file:line
	Fabricated      string `json:"fabricated"`
	EvidenceChecked string `json:"evidence_checked"`
	FixApplied      string `json:"fix_applied,omitempty"`
	SuggestedFix    string `json:"suggested_fix,omitempty"`
}

// WeakNumberIssue represents a weak quantification.
type WeakNumberIssue struct {
	Location   string `json:"location"`
	WeakNumber string `json:"weak_number"`
	Suggested  string `json:"suggested"`
	Fixed      bool   `json:"fixed"`
}

// JDMatch tracks how well resume matches JD requirements.
type JDMatch struct {
	Matched             []string `json:"matched"`
	Unmatched           []string `json:"unmatched"`
	FabricationsToMatch []string `json:"fabrications_to_match"`
}

// EvaluationIndex is the searchable index of all evaluations.
type EvaluationIndex struct {
	Evaluations []IndexedEvaluation `json:"evaluations"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Version     string              `json:"version"`
}

// IndexedEvaluation is a summary for RAG retrieval.
type IndexedEvaluation struct {
	Company            string    `json:"company"`
	Role               string    `json:"role"`
	RoleLevel          string    `json:"role_level"` // IC, Director, VP, CTO
	Industry           string    `json:"industry"`   // Extracted from JD
	EvaluatedAt        time.Time `json:"evaluated_at"`
	OverallScore       int       `json:"overall_score"`
	CriticalViolations int       `json:"critical_violations"`
	LessonsLearned     []string  `json:"lessons_learned"`
	RAGContext         string    `json:"rag_context"`
	Path               string    `json:"path"` // Path to full evaluation
}

// RAGContext is what gets injected into generation prompts.
type RAGContext struct {
	RelevantLessons     []string `json:"relevant_lessons"`
	CommonViolations    []string `json:"common_violations"`
	SuccessfulPatterns  []string `json:"successful_patterns"`
	SimilarApplications int      `json:"similar_applications"`
}
