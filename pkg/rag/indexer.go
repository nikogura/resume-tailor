package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Indexer indexes evaluation files for RAG retrieval.
type Indexer struct {
	applicationsPath string // ~/Documents/Applications
	indexPath        string // ~/Documents/Applications/.rag-index.json
}

// NewIndexer creates a new indexer instance.
func NewIndexer(applicationsPath string) (indexer *Indexer, err error) {
	if applicationsPath == "" {
		err = errors.New("applications path is required")
		return indexer, err
	}

	indexPath := filepath.Join(applicationsPath, ".rag-index.json")

	indexer = &Indexer{
		applicationsPath: applicationsPath,
		indexPath:        indexPath,
	}

	return indexer, err
}

// processEvaluationFile processes a single evaluation file during directory walk.
func (idx *Indexer) processEvaluationFile(path string, info os.FileInfo, walkErr error, evaluations *[]IndexedEvaluation, count *int) (err error) {
	if walkErr != nil {
		err = walkErr
		return err
	}

	// Skip if not .evaluation.json
	if info.IsDir() || !strings.HasSuffix(info.Name(), ".evaluation.json") {
		return err
	}

	// Load evaluation
	var eval Evaluation
	eval, err = idx.loadEvaluation(path)
	if err != nil {
		// Log but don't fail - skip bad evaluations
		err = nil
		//nolint:nilerr // Intentionally swallowing error to skip bad evaluations
		return err
	}

	// Extract industry from company name (simple heuristic)
	industry := idx.inferIndustry(eval.Company)

	// Determine role level
	roleLevel := idx.inferRoleLevel(eval.Role)

	// Count critical violations
	criticalCount := 0
	for _, v := range eval.Scores.Resume.AntiFabrication.Violations {
		if v.Severity == "critical" {
			criticalCount++
		}
	}
	for _, v := range eval.Scores.CoverLetter.DomainClaims.Violations {
		if v.Severity == "critical" {
			criticalCount++
		}
	}

	// Create indexed entry
	indexed := IndexedEvaluation{
		Company:            eval.Company,
		Role:               eval.Role,
		RoleLevel:          roleLevel,
		Industry:           industry,
		EvaluatedAt:        eval.EvaluatedAt,
		OverallScore:       eval.Scores.Overall,
		CriticalViolations: criticalCount,
		LessonsLearned:     eval.Lessons,
		RAGContext:         eval.RAGContext,
		Path:               path,
	}

	*evaluations = append(*evaluations, indexed)
	*count++

	return err
}

// Index scans all .evaluation.json files and builds searchable index.
func (idx *Indexer) Index(ctx context.Context) (count int, err error) {
	evaluations := []IndexedEvaluation{}

	// Walk the applications directory
	walkErr := filepath.Walk(idx.applicationsPath, func(path string, info os.FileInfo, walkErr error) (walkFuncErr error) {
		walkFuncErr = idx.processEvaluationFile(path, info, walkErr, &evaluations, &count)
		return walkFuncErr
	})

	if walkErr != nil {
		err = fmt.Errorf("failed to walk applications directory: %w", walkErr)
		return count, err
	}

	// Build index
	index := EvaluationIndex{
		Evaluations: evaluations,
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
	}

	// Write index
	err = idx.writeIndex(index)
	if err != nil {
		err = fmt.Errorf("failed to write index: %w", err)
		return count, err
	}

	return count, err
}

func (idx *Indexer) loadEvaluation(path string) (eval Evaluation, err error) {
	var data []byte
	data, err = os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("failed to read evaluation file: %w", err)
		return eval, err
	}

	err = json.Unmarshal(data, &eval)
	if err != nil {
		err = fmt.Errorf("failed to parse evaluation JSON: %w", err)
		return eval, err
	}

	return eval, err
}

func (idx *Indexer) writeIndex(index EvaluationIndex) (err error) {
	var data []byte
	data, err = json.MarshalIndent(index, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal index: %w", err)
		return err
	}

	err = os.WriteFile(idx.indexPath, data, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write index file: %w", err)
		return err
	}

	return err
}

// inferIndustry extracts industry from company name (simple heuristics).
func (idx *Indexer) inferIndustry(company string) (industry string) {
	lower := strings.ToLower(company)

	if strings.Contains(lower, "bank") || strings.Contains(lower, "capital") {
		industry = "fintech"
		return industry
	}
	if strings.Contains(lower, "tech") || strings.Contains(lower, "soft") {
		industry = "technology"
		return industry
	}
	if strings.Contains(lower, "cloud") || strings.Contains(lower, "aws") {
		industry = "cloud"
		return industry
	}
	if strings.Contains(lower, "pay") {
		industry = "payments"
		return industry
	}

	industry = "unknown"
	return industry
}

// inferRoleLevel determines role level from title.
func (idx *Indexer) inferRoleLevel(role string) (level string) {
	lower := strings.ToLower(role)

	if strings.Contains(lower, "cto") || strings.Contains(lower, "chief") {
		level = "CTO"
		return level
	}
	if strings.Contains(lower, "vp") || strings.Contains(lower, "vice president") {
		level = "VP"
		return level
	}
	if strings.Contains(lower, "director") {
		level = "Director"
		return level
	}
	if strings.Contains(lower, "senior") || strings.Contains(lower, "sr") || strings.Contains(lower, "principal") {
		level = "Senior IC"
		return level
	}
	if strings.Contains(lower, "lead") || strings.Contains(lower, "staff") {
		level = "IC"
		return level
	}

	level = "IC"
	return level
}

// LoadIndex loads the existing index from disk.
func (idx *Indexer) LoadIndex() (index EvaluationIndex, err error) {
	var data []byte
	data, err = os.ReadFile(idx.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty index
			index = EvaluationIndex{
				Evaluations: []IndexedEvaluation{},
				UpdatedAt:   time.Now(),
				Version:     "1.0.0",
			}
			err = nil
			return index, err
		}
		err = fmt.Errorf("failed to read index file: %w", err)
		return index, err
	}

	err = json.Unmarshal(data, &index)
	if err != nil {
		err = fmt.Errorf("failed to parse index JSON: %w", err)
		return index, err
	}

	return index, err
}
