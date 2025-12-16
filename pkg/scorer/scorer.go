package scorer

import (
	"github.com/nikogura/resume-tailor/pkg/rag"
)

// Scorer calculates scores from evaluation data.
type Scorer struct{}

// NewScorer creates a new scorer instance.
func NewScorer() (scorer *Scorer) {
	scorer = &Scorer{}
	return scorer
}

// CalculateScores computes all scores from violations and issues.
func (s *Scorer) CalculateScores(antiFabViolations []rag.Violation, weakIssues []rag.WeakNumberIssue,
	accuracyViolations []rag.Violation, domainViolations []rag.Violation,
	verifiedMetrics []string, companyDatesOK, roleTitlesOK, yearsExpOK bool) (scores rag.Scores, err error) {

	// Calculate Resume Anti-Fabrication Score
	antiFabScore := s.calculateAntiFabricationScore(antiFabViolations)

	// Calculate Weak Quantifications Score
	weakScore := s.calculateWeakQuantificationsScore(weakIssues)

	// Calculate Accuracy Score
	accuracyScore := s.calculateAccuracyScore(accuracyViolations, verifiedMetrics,
		companyDatesOK, roleTitlesOK, yearsExpOK)

	// Calculate Resume Total (weighted average)
	resumeTotal := int(float64(antiFabScore)*0.50 + float64(weakScore)*0.20 + float64(accuracyScore)*0.30)

	// Calculate Cover Letter Domain Claims Score
	domainScore := s.calculateDomainClaimsScore(domainViolations)

	// Cover Letter Total (simplified for now)
	coverLetterTotal := domainScore

	// Overall Score (weighted by category)
	overall := int(float64(resumeTotal)*0.70 + float64(coverLetterTotal)*0.30)

	scores = rag.Scores{
		Resume: rag.ResumeScore{
			Total: resumeTotal,
			AntiFabrication: rag.AntiFabricationScore{
				Score:      antiFabScore,
				Violations: antiFabViolations,
			},
			WeakQuantifications: rag.WeakQuantificationsScore{
				Score:  weakScore,
				Issues: weakIssues,
			},
			Accuracy: rag.AccuracyScore{
				Score:               accuracyScore,
				VerifiedMetrics:     verifiedMetrics,
				CompanyDatesCorrect: companyDatesOK,
				RoleTitlesCorrect:   roleTitlesOK,
				YearsExpCorrect:     yearsExpOK,
			},
		},
		CoverLetter: rag.CoverLetterScore{
			Total: coverLetterTotal,
			DomainClaims: rag.DomainClaimsScore{
				Score:      domainScore,
				Violations: domainViolations,
			},
			Tone: rag.ToneScore{
				Score:    100, // TODO: Implement tone scoring
				Feedback: []string{},
			},
		},
		Overall: overall,
	}

	return scores, err
}

func (s *Scorer) calculateAntiFabricationScore(violations []rag.Violation) (score int) {
	score = 100

	for _, v := range violations {
		rule, exists := ScoringRules[v.Rule]
		if !exists {
			continue
		}

		if rule.Category == "anti_fabrication" {
			score -= rule.Weight
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *Scorer) calculateWeakQuantificationsScore(issues []rag.WeakNumberIssue) (score int) {
	score = 100

	for range issues {
		score -= ScoringRules["WEAK_QUANTIFICATIONS"].Weight
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *Scorer) calculateAccuracyScore(violations []rag.Violation, verifiedMetrics []string,
	companyDatesOK, roleTitlesOK, yearsExpOK bool) (score int) {

	score = 100

	// Deduct for violations
	for _, v := range violations {
		rule, exists := ScoringRules[v.Rule]
		if !exists {
			continue
		}

		if rule.Category == "accuracy" {
			score -= rule.Weight
		}
	}

	// Deduct for incorrect metadata
	if !companyDatesOK {
		score -= ScoringRules["COMPANY_DATE_MISMATCH"].Weight
	}
	if !roleTitlesOK {
		score -= ScoringRules["ROLE_TITLE_MISMATCH"].Weight
	}
	if !yearsExpOK {
		score -= ScoringRules["YEARS_EXPERIENCE_WRONG"].Weight
	}

	// Bonus for verified metrics (up to +10)
	metricsBonus := len(verifiedMetrics)
	if metricsBonus > 10 {
		metricsBonus = 10
	}
	score += metricsBonus

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func (s *Scorer) calculateDomainClaimsScore(violations []rag.Violation) (score int) {
	score = 100

	for _, v := range violations {
		rule, exists := ScoringRules[v.Rule]
		if !exists {
			continue
		}

		score -= rule.Weight
	}

	if score < 0 {
		score = 0
	}

	return score
}

// ExtractLessons generates lessons learned from evaluation.
func (s *Scorer) ExtractLessons(scores rag.Scores) (lessons []string) {
	lessons = []string{}

	// Check for critical violations
	if len(scores.Resume.AntiFabrication.Violations) > 0 {
		for _, v := range scores.Resume.AntiFabrication.Violations {
			if v.Severity == "critical" {
				lesson := "Fabrication detected: " + v.Rule + " - " + v.Fabricated
				lessons = append(lessons, lesson)
			}
		}
	}

	// Check for weak quantifications
	if len(scores.Resume.WeakQuantifications.Issues) > 0 {
		lessons = append(lessons, "Weak quantifications found that undermine credibility")
	}

	// Check for domain violations in cover letter
	if len(scores.CoverLetter.DomainClaims.Violations) > 0 {
		lessons = append(lessons, "Cover letter made domain claims not supported by achievements")
	}

	// Check overall score
	if scores.Overall < 70 {
		lessons = append(lessons, "Overall quality below acceptable threshold - multiple issues detected")
	}

	return lessons
}

// GenerateRAGContext creates the RAG context string for future generations.
func (s *Scorer) GenerateRAGContext(company, role string, scores rag.Scores, lessons []string) (context string) {
	context = "Application: " + company + " - " + role + "\n"
	context += "Overall Score: " + string(rune(scores.Overall)) + "/100\n\n"

	if len(lessons) > 0 {
		context += "Key Issues:\n"
		for _, lesson := range lessons {
			context += "- " + lesson + "\n"
		}
	}

	// Add specific violation patterns
	if len(scores.Resume.AntiFabrication.Violations) > 0 {
		context += "\nFabrication Patterns to Avoid:\n"
		for _, v := range scores.Resume.AntiFabrication.Violations {
			context += "- " + v.Rule + ": " + v.Fabricated + "\n"
		}
	}

	return context
}
