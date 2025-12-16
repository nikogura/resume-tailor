package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nikogura/resume-tailor/pkg/config"
	"github.com/nikogura/resume-tailor/pkg/llm"
	"github.com/nikogura/resume-tailor/pkg/rag"
	"github.com/nikogura/resume-tailor/pkg/scorer"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//nolint:gochecknoglobals // Cobra boilerplate
var evaluateAll bool

//nolint:gochecknoglobals // Cobra boilerplate
var evaluateCmd = &cobra.Command{
	Use:   "evaluate [application-directory]",
	Short: "Evaluate generated resumes for hallucinations and quality",
	Long: `Evaluates generated resumes and cover letters against anti-fabrication rules.

Uses a separate Claude instance to check for:
- Number fabrications (invented metrics/headcounts)
- Industry fabrications (claiming industries not in experience)
- Domain fabrications (claiming technical domains not in experience)
- Pattern matching violations (claiming work "mirrors" domains candidate lacks)
- Weak quantifications (numbers that undermine credibility)
- Factual accuracy (company/role/date correctness)

Stores evaluation results in .evaluation.json alongside generated files.

Examples:
  # Evaluate a specific application
  resume-tailor evaluate ~/Documents/Applications/overstory

  # Evaluate all applications
  resume-tailor evaluate --all

  # Evaluate and show verbose output
  resume-tailor evaluate ~/Documents/Applications/overstory -v`,
	RunE: runEvaluate,
}

//nolint:gochecknoinits // Cobra boilerplate
func init() {
	rootCmd.AddCommand(evaluateCmd)
	evaluateCmd.Flags().BoolVar(&evaluateAll, "all", false, "Evaluate all applications in ~/Documents/Applications")
}

func runEvaluate(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()

	// Load config for API key
	var cfg config.Config
	cfg, err = config.Load(getConfigFile())
	if err != nil {
		err = fmt.Errorf("failed to load config: %w", err)
		return err
	}

	// Create evaluator
	var evaluator *llm.Evaluator
	evaluator, err = llm.NewEvaluator(cfg.AnthropicAPIKey, cfg.GetEvaluationModel())
	if err != nil {
		err = fmt.Errorf("failed to create evaluator: %w", err)
		return err
	}

	// Determine which applications to evaluate
	var appDirs []string
	if evaluateAll {
		appDirs, err = findAllApplications(cfg.Defaults.OutputDir)
		if err != nil {
			err = fmt.Errorf("failed to find applications: %w", err)
			return err
		}
	} else {
		if len(args) == 0 {
			err = errors.New("provide application directory or use --all")
			return err
		}
		appDirs = args
	}

	if getVerbose() {
		fmt.Printf("Evaluating %d application(s)...\n", len(appDirs))
	}

	// Evaluate each application
	successCount := 0
	for _, appDir := range appDirs {
		evalErr := evaluateApplication(ctx, evaluator, appDir)
		if evalErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to evaluate %s: %v\n", appDir, evalErr)
			continue
		}
		successCount++
	}

	fmt.Printf("Successfully evaluated %d/%d applications\n", successCount, len(appDirs))

	// Rebuild RAG index after evaluating
	if getVerbose() {
		fmt.Println("Rebuilding RAG index...")
	}

	var indexer *rag.Indexer
	indexer, err = rag.NewIndexer(cfg.Defaults.OutputDir)
	if err != nil {
		err = fmt.Errorf("failed to create indexer: %w", err)
		return err
	}

	var count int
	count, err = indexer.Index(ctx)
	if err != nil {
		err = fmt.Errorf("failed to build RAG index: %w", err)
		return err
	}

	if getVerbose() {
		fmt.Printf("Indexed %d evaluations\n", count)
	}

	return err
}

func findAllApplications(outputDir string) (dirs []string, err error) {
	var entries []os.DirEntry
	entries, err = os.ReadDir(outputDir)
	if err != nil {
		err = fmt.Errorf("failed to read output directory: %w", err)
		return dirs, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		appDir := filepath.Join(outputDir, entry.Name())
		dirs = append(dirs, appDir)
	}

	return dirs, err
}

func evaluateApplication(ctx context.Context, evaluator *llm.Evaluator, appDir string) (err error) {
	if getVerbose() {
		fmt.Printf("Evaluating %s...\n", filepath.Base(appDir))
	}

	// Find generated files
	var resumePath, coverPath, jdPath string
	resumePath, coverPath, jdPath, err = findGeneratedFiles(appDir)
	if err != nil {
		err = fmt.Errorf("failed to find generated files: %w", err)
		return err
	}

	// Load application files and source data
	var evalReq llm.EvaluationRequest
	var company, role string
	evalReq, company, role, err = loadAndBuildEvaluationRequest(appDir, resumePath, coverPath, jdPath)
	if err != nil {
		return err
	}

	// Run evaluation
	var evalResp llm.EvaluationResponse
	evalResp, err = evaluator.Evaluate(ctx, evalReq)
	if err != nil {
		err = fmt.Errorf("evaluation failed: %w", err)
		return err
	}

	// Process results and write evaluation
	var scores rag.Scores
	scores, err = processAndWriteEvaluation(appDir, company, role, evalResp)
	if err != nil {
		return err
	}

	// Print summary
	printEvaluationSummary(scores, evalResp)

	return err
}

func loadAndBuildEvaluationRequest(appDir, resumePath, coverPath, jdPath string) (evalReq llm.EvaluationRequest, company, role string, err error) {
	// Load config to get source data paths
	var cfg config.Config
	cfg, err = config.Load(getConfigFile())
	if err != nil {
		err = fmt.Errorf("failed to load config: %w", err)
		return evalReq, company, role, err
	}

	// Load generated content
	var resumeContent []byte
	resumeContent, err = os.ReadFile(resumePath)
	if err != nil {
		err = fmt.Errorf("failed to read resume: %w", err)
		return evalReq, company, role, err
	}

	var coverContent []byte
	coverContent, err = os.ReadFile(coverPath)
	if err != nil {
		err = fmt.Errorf("failed to read cover letter: %w", err)
		return evalReq, company, role, err
	}

	var jdContent []byte
	jdContent, err = os.ReadFile(jdPath)
	if err != nil {
		err = fmt.Errorf("failed to read job description: %w", err)
		return evalReq, company, role, err
	}

	// Load source data
	var achievementsJSON, profileJSON, skillsJSON string
	achievementsJSON, profileJSON, skillsJSON, err = loadSourceData(cfg)
	if err != nil {
		err = fmt.Errorf("failed to load source data: %w", err)
		return evalReq, company, role, err
	}

	// Extract company and role from path
	company, role = extractCompanyRole(appDir, resumePath)

	// Build evaluation request
	evalReq = llm.EvaluationRequest{
		Company:            company,
		Role:               role,
		JobDescription:     string(jdContent),
		Resume:             string(resumeContent),
		CoverLetter:        string(coverContent),
		SourceAchievements: achievementsJSON,
		SourceSkills:       skillsJSON,
		SourceProfile:      profileJSON,
	}

	return evalReq, company, role, err
}

func processAndWriteEvaluation(appDir, company, role string, evalResp llm.EvaluationResponse) (scores rag.Scores, err error) {
	// Calculate scores
	scr := scorer.NewScorer()
	scores, err = scr.CalculateScores(
		evalResp.ResumeViolations,
		evalResp.WeakQuantifications,
		evalResp.AccuracyViolations,
		evalResp.CoverLetterViolations,
		evalResp.VerifiedMetrics,
		evalResp.CompanyDatesCorrect,
		evalResp.RoleTitlesCorrect,
		evalResp.YearsExpCorrect,
	)
	if err != nil {
		err = fmt.Errorf("failed to calculate scores: %w", err)
		return scores, err
	}

	// Extract lessons
	lessons := scr.ExtractLessons(scores)
	lessons = append(lessons, evalResp.LessonsLearned...)

	// Generate RAG context
	ragContext := scr.GenerateRAGContext(company, role, scores, lessons)

	// Build full evaluation
	evaluation := rag.Evaluation{
		Company:     company,
		Role:        role,
		GeneratedAt: time.Now(), // TODO: Get from file metadata
		EvaluatedAt: time.Now(),
		Scores:      scores,
		JDMatch:     evalResp.JDMatch,
		Lessons:     lessons,
		RAGContext:  ragContext,
		Version:     "1.0.0",
	}

	// Write evaluation
	evalPath := filepath.Join(appDir, ".evaluation.json")
	err = writeEvaluation(evalPath, evaluation)
	if err != nil {
		err = fmt.Errorf("failed to write evaluation: %w", err)
		return scores, err
	}

	return scores, err
}

func printEvaluationSummary(scores rag.Scores, evalResp llm.EvaluationResponse) {
	fmt.Printf("  Overall Score: %d/100\n", scores.Overall)
	if len(evalResp.ResumeViolations) > 0 {
		fmt.Printf("  Resume Violations: %d\n", len(evalResp.ResumeViolations))
	}
	if len(evalResp.CoverLetterViolations) > 0 {
		fmt.Printf("  Cover Letter Violations: %d\n", len(evalResp.CoverLetterViolations))
	}
	if scores.Overall < 70 {
		fmt.Printf("  ⚠️  Score below threshold - review required\n")
	}
}

func findGeneratedFiles(appDir string) (resumePath, coverPath, jdPath string, err error) {
	var entries []os.DirEntry
	entries, err = os.ReadDir(appDir)
	if err != nil {
		err = fmt.Errorf("failed to read application directory: %w", err)
		return resumePath, coverPath, jdPath, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, "-resume.md") {
			resumePath = filepath.Join(appDir, name)
		}
		if strings.HasSuffix(name, "-cover.md") {
			coverPath = filepath.Join(appDir, name)
		}
		if strings.HasSuffix(name, "-jd.txt") {
			jdPath = filepath.Join(appDir, name)
		}
	}

	if resumePath == "" {
		err = errors.New("resume markdown file not found")
		return resumePath, coverPath, jdPath, err
	}
	if coverPath == "" {
		err = errors.New("cover letter markdown file not found")
		return resumePath, coverPath, jdPath, err
	}
	if jdPath == "" {
		err = errors.New("job description file not found")
		return resumePath, coverPath, jdPath, err
	}

	return resumePath, coverPath, jdPath, err
}

func loadSourceData(cfg config.Config) (achievementsJSON, profileJSON, skillsJSON string, err error) {
	// Load structured summaries
	var achievementsData []byte
	achievementsData, err = os.ReadFile(cfg.SummariesLocation)
	if err != nil {
		err = fmt.Errorf("failed to read summaries: %w", err)
		return achievementsJSON, profileJSON, skillsJSON, err
	}

	// Parse to extract achievements, profile, skills
	var summaries map[string]interface{}
	err = json.Unmarshal(achievementsData, &summaries)
	if err != nil {
		err = fmt.Errorf("failed to parse summaries: %w", err)
		return achievementsJSON, profileJSON, skillsJSON, err
	}

	// Extract and re-marshal each section
	if achievements, ok := summaries["achievements"]; ok {
		var achData []byte
		achData, err = json.MarshalIndent(achievements, "", "  ")
		if err != nil {
			return achievementsJSON, profileJSON, skillsJSON, err
		}
		achievementsJSON = string(achData)
	}

	if profile, ok := summaries["profile"]; ok {
		var profData []byte
		profData, err = json.MarshalIndent(profile, "", "  ")
		if err != nil {
			return achievementsJSON, profileJSON, skillsJSON, err
		}
		profileJSON = string(profData)
	}

	if skills, ok := summaries["skills"]; ok {
		var skillsData []byte
		skillsData, err = json.MarshalIndent(skills, "", "  ")
		if err != nil {
			return achievementsJSON, profileJSON, skillsJSON, err
		}
		skillsJSON = string(skillsData)
	}

	return achievementsJSON, profileJSON, skillsJSON, err
}

func extractCompanyRole(appDir, resumePath string) (company, role string) {
	// Extract from directory name
	company = filepath.Base(appDir)

	// Extract role from filename (e.g., "nik-ogura-overstory-chief-technology-officer-resume.md")
	filename := filepath.Base(resumePath)
	parts := strings.Split(filename, "-")

	// Remove prefix (nik-ogura) and suffix (resume.md)
	if len(parts) > 4 {
		// Skip first 2 (name), skip company name, collect rest until "resume"
		roleStart := 2
		// Find company name end
		companyLower := strings.ToLower(company)
		for i := 2; i < len(parts); i++ {
			if strings.Contains(companyLower, parts[i]) {
				roleStart = i + 1
				break
			}
		}

		roleParts := []string{}
		for i := roleStart; i < len(parts); i++ {
			if parts[i] == "resume.md" || parts[i] == "resume" {
				break
			}
			roleParts = append(roleParts, parts[i])
		}

		// Capitalize each word
		titleCaser := cases.Title(language.English)
		for i, part := range roleParts {
			roleParts[i] = titleCaser.String(part)
		}
		role = strings.Join(roleParts, " ")
	}

	return company, role
}

func writeEvaluation(path string, evaluation rag.Evaluation) (err error) {
	var data []byte
	data, err = json.MarshalIndent(evaluation, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal evaluation: %w", err)
		return err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write evaluation file: %w", err)
		return err
	}

	return err
}
