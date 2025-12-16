package scorer

// Rule represents a scoring rule.
type Rule struct {
	Name        string
	Category    string // anti_fabrication, accuracy, quality
	Severity    string // critical, major, minor
	Description string
	Weight      int // Points deducted for violation
}

//nolint:gochecknoglobals // Scoring configuration constants
var ScoringRules = map[string]Rule{
	// Anti-Fabrication Rules (Critical)
	"FORBIDDEN_NUMBER_FABRICATION": {
		Name:        "FORBIDDEN_NUMBER_FABRICATION",
		Category:    "anti_fabrication",
		Severity:    "critical",
		Description: "Numbers invented that don't exist in source achievement metrics",
		Weight:      30,
	},
	"FORBIDDEN_INDUSTRY_CLAIMS": {
		Name:        "FORBIDDEN_INDUSTRY_CLAIMS",
		Category:    "anti_fabrication",
		Severity:    "critical",
		Description: "Industry claims (climate-tech, gaming, etc.) not in achievement companies",
		Weight:      25,
	},
	"FORBIDDEN_TECHNICAL_DOMAIN_CLAIMS": {
		Name:        "FORBIDDEN_TECHNICAL_DOMAIN_CLAIMS",
		Category:    "anti_fabrication",
		Severity:    "critical",
		Description: "Technical domain claims (satellite imagery, geospatial) not in achievements",
		Weight:      25,
	},
	"FORBIDDEN_PATTERN_MATCHING": {
		Name:        "FORBIDDEN_PATTERN_MATCHING",
		Category:    "anti_fabrication",
		Severity:    "critical",
		Description: "Claims that work 'mirrors' or is 'similar to' JD domain candidate lacks",
		Weight:      20,
	},
	"SKILL_FABRICATION": {
		Name:        "SKILL_FABRICATION",
		Category:    "anti_fabrication",
		Severity:    "major",
		Description: "Skills listed that are not in source skills data",
		Weight:      15,
	},
	"WEAK_QUANTIFICATIONS": {
		Name:        "WEAK_QUANTIFICATIONS",
		Category:    "anti_fabrication",
		Severity:    "minor",
		Description: "Numbers under 10-20 that undermine credibility (7 clusters, 3 regions, etc.)",
		Weight:      5,
	},

	// Accuracy Rules
	"COMPANY_DATE_MISMATCH": {
		Name:        "COMPANY_DATE_MISMATCH",
		Category:    "accuracy",
		Severity:    "critical",
		Description: "Company employment dates don't match source achievement data",
		Weight:      25,
	},
	"ROLE_TITLE_MISMATCH": {
		Name:        "ROLE_TITLE_MISMATCH",
		Category:    "accuracy",
		Severity:    "critical",
		Description: "Role titles modified from source achievement data",
		Weight:      20,
	},
	"YEARS_EXPERIENCE_WRONG": {
		Name:        "YEARS_EXPERIENCE_WRONG",
		Category:    "accuracy",
		Severity:    "critical",
		Description: "Years of experience doesn't match profile.years_experience",
		Weight:      25,
	},
	"METRIC_FABRICATION": {
		Name:        "METRIC_FABRICATION",
		Category:    "accuracy",
		Severity:    "critical",
		Description: "Metrics (percentages, dollar amounts) not in achievement metrics",
		Weight:      20,
	},
	"TEMPORAL_IMPOSSIBILITY": {
		Name:        "TEMPORAL_IMPOSSIBILITY",
		Category:    "accuracy",
		Severity:    "major",
		Description: "Claims X years experience with tool that didn't exist for X years",
		Weight:      15,
	},

	// Quality Rules
	"POOR_JD_ALIGNMENT": {
		Name:        "POOR_JD_ALIGNMENT",
		Category:    "quality",
		Severity:    "minor",
		Description: "Resume doesn't emphasize JD-relevant achievements",
		Weight:      5,
	},
	"INAPPROPRIATE_TONE": {
		Name:        "INAPPROPRIATE_TONE",
		Category:    "quality",
		Severity:    "minor",
		Description: "Cover letter tone doesn't match company culture signals",
		Weight:      5,
	},
}

//nolint:gochecknoglobals // Scoring configuration constants
var CategoryWeights = map[string]float64{
	"anti_fabrication": 0.50, // 50%
	"accuracy":         0.30, // 30%
	"quality":          0.20, // 20%
}

//nolint:gochecknoglobals // Scoring configuration constants
var SeverityThresholds = map[string]int{
	"critical": 60, // Any critical violation must keep score above 60
	"major":    70, // Multiple major violations must keep score above 70
}
