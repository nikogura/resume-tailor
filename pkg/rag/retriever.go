package rag

import (
	"context"
	"fmt"
	"strings"
)

// Retriever retrieves relevant RAG context for new resume generation.
type Retriever struct {
	indexer *Indexer
}

// NewRetriever creates a new retriever instance.
func NewRetriever(indexer *Indexer) (retriever *Retriever) {
	retriever = &Retriever{
		indexer: indexer,
	}
	return retriever
}

// Retrieve finds relevant past evaluations for the given JD and role.
func (r *Retriever) Retrieve(ctx context.Context, company, role, jdText string) (ragCtx RAGContext, err error) {
	// Load index
	var index EvaluationIndex
	index, err = r.indexer.LoadIndex()
	if err != nil {
		err = fmt.Errorf("failed to load index: %w", err)
		return ragCtx, err
	}

	// Determine role level for this application
	roleLevel := r.indexer.inferRoleLevel(role)

	// Find similar applications
	var similar []IndexedEvaluation
	for _, eval := range index.Evaluations {
		score := r.calculateSimilarity(eval, roleLevel)
		if score > 0.3 { // Threshold for relevance
			similar = append(similar, eval)
		}
	}

	// Extract lessons and violations from similar applications
	ragCtx = r.buildRAGContext(similar)
	ragCtx.SimilarApplications = len(similar)

	return ragCtx, err
}

func (r *Retriever) calculateSimilarity(eval IndexedEvaluation, roleLevel string) (score float64) {
	score = 0.0

	// Role level match (highest weight)
	if eval.RoleLevel == roleLevel {
		score += 0.5
	}

	// Recent applications are more relevant
	// (applications within last 30 days get bonus)
	// daysSince := time.Since(eval.EvaluatedAt).Hours() / 24
	// if daysSince < 30 {
	//     score += 0.2
	// }

	// Low scores indicate problem areas - prioritize learning from failures
	if eval.OverallScore < 80 {
		score += 0.3
	}

	// Had critical violations - definitely want to learn from these
	if eval.CriticalViolations > 0 {
		score += 0.4
	}

	return score
}

func (r *Retriever) buildRAGContext(similar []IndexedEvaluation) (ctx RAGContext) {
	ctx = RAGContext{
		RelevantLessons:    []string{},
		CommonViolations:   []string{},
		SuccessfulPatterns: []string{},
	}

	// Track violations we've seen
	violationMap := make(map[string]int)

	for _, eval := range similar {
		// Collect lessons learned
		for _, lesson := range eval.LessonsLearned {
			// Avoid duplicates
			if !contains(ctx.RelevantLessons, lesson) {
				ctx.RelevantLessons = append(ctx.RelevantLessons, lesson)
			}
		}

		// Extract violation patterns from RAG context
		if strings.Contains(eval.RAGContext, "FORBIDDEN_NUMBER_FABRICATION") {
			violationMap["Number fabrication (inventing metrics/headcounts)"]++
		}
		if strings.Contains(eval.RAGContext, "FORBIDDEN_INDUSTRY_CLAIMS") {
			violationMap["Industry fabrication (claiming industries not in experience)"]++
		}
		if strings.Contains(eval.RAGContext, "FORBIDDEN_TECHNICAL_DOMAIN_CLAIMS") {
			violationMap["Domain fabrication (claiming technical domains not in experience)"]++
		}
		if strings.Contains(eval.RAGContext, "FORBIDDEN_PATTERN_MATCHING") {
			violationMap["Pattern matching (claiming work 'mirrors' domains candidate lacks)"]++
		}

		// Collect successful patterns (high scores)
		if eval.OverallScore >= 85 {
			ctx.SuccessfulPatterns = append(ctx.SuccessfulPatterns,
				fmt.Sprintf("%s application scored %d - good example", eval.Company, eval.OverallScore))
		}
	}

	// Convert violation map to list, sorted by frequency
	for violation, count := range violationMap {
		ctx.CommonViolations = append(ctx.CommonViolations,
			fmt.Sprintf("%s (occurred %d times)", violation, count))
	}

	return ctx
}

func contains(slice []string, item string) (found bool) {
	for _, s := range slice {
		if s == item {
			found = true
			return found
		}
	}
	found = false
	return found
}

// FormatForPrompt formats RAG context for injection into generation prompt.
func (r *Retriever) FormatForPrompt(ctx RAGContext) (formatted string) {
	if ctx.SimilarApplications == 0 {
		formatted = "No previous evaluation data available."
		return formatted
	}

	formatted = fmt.Sprintf("**LEARNING FROM %d PREVIOUS APPLICATIONS:**\n\n", ctx.SimilarApplications)

	if len(ctx.CommonViolations) > 0 {
		formatted += "**COMMON VIOLATIONS TO AVOID:**\n"
		for _, violation := range ctx.CommonViolations {
			formatted += fmt.Sprintf("- %s\n", violation)
		}
		formatted += "\n"
	}

	if len(ctx.RelevantLessons) > 0 {
		formatted += "**LESSONS LEARNED:**\n"
		for _, lesson := range ctx.RelevantLessons {
			formatted += fmt.Sprintf("- %s\n", lesson)
		}
		formatted += "\n"
	}

	if len(ctx.SuccessfulPatterns) > 0 {
		formatted += "**SUCCESSFUL PATTERNS:**\n"
		for _, pattern := range ctx.SuccessfulPatterns {
			formatted += fmt.Sprintf("- %s\n", pattern)
		}
		formatted += "\n"
	}

	return formatted
}
