package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nikogura/resume-tailor/pkg/config"
	"github.com/nikogura/resume-tailor/pkg/jd"
	"github.com/nikogura/resume-tailor/pkg/llm"
	"github.com/nikogura/resume-tailor/pkg/rag"
	"github.com/nikogura/resume-tailor/pkg/renderer"
	"github.com/nikogura/resume-tailor/pkg/summaries"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra boilerplate
var company string

//nolint:gochecknoglobals // Cobra boilerplate
var role string

//nolint:gochecknoglobals // Cobra boilerplate
var outputDir string

//nolint:gochecknoglobals // Cobra boilerplate
var keepMarkdown bool

//nolint:gochecknoglobals // Cobra boilerplate
var coverLetterContext string

//nolint:gochecknoglobals // Cobra boilerplate
var jobID string

//nolint:gochecknoglobals // Cobra boilerplate
var autoFix bool

//nolint:gochecknoglobals // Cobra boilerplate
var skipPDF bool

//nolint:gochecknoglobals // Cobra boilerplate
var generateCmd = &cobra.Command{
	Use:   "generate <jd-file-or-url>",
	Short: "Generate tailored resume and cover letter",
	Long: `Generate a tailored resume and cover letter based on a job description.

The job description can be provided as:
- A file path (e.g., jd.txt)
- A URL (e.g., https://example.com/jobs/123)

Example:
  resume-tailor generate jd.txt --company "Acme Corp" --role "Staff Engineer"
  resume-tailor generate https://example.com/jobs/123 --company "Acme" --role "SRE"
  resume-tailor generate jd.txt --company "Acme" --role "Staff Engineer" --job-id "req-12345"`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

//nolint:gochecknoinits // Cobra boilerplate
func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&company, "company", "", "Company name (extracted from JD if not provided)")
	generateCmd.Flags().StringVar(&role, "role", "", "Role title (extracted from JD if not provided)")
	generateCmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default from config)")
	generateCmd.Flags().StringVar(&jobID, "job-id", "", "Optional job/req ID to differentiate multiple applications (e.g., 'req-12345', '8886')")
	generateCmd.Flags().BoolVar(&keepMarkdown, "keep-markdown", true, "Keep markdown files after PDF generation")
	generateCmd.Flags().StringVar(&coverLetterContext, "context", "", "Additional context for cover letter generation")
	generateCmd.Flags().BoolVar(&autoFix, "auto-fix", true, "Automatically fix violations detected during evaluation")
	generateCmd.Flags().BoolVar(&skipPDF, "skip-pdf", false, "Skip PDF generation (useful for manual workflows)")
}

func runGenerate(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	jdInput := args[0]

	// Setup: load config, fetch JD, load summaries
	var cfg config.Config
	var jobDescription string
	var data summaries.Data
	var outDir string
	var client *llm.Client
	cfg, jobDescription, data, client, err = setupGeneration(jdInput)
	if err != nil {
		return err
	}

	// Convert achievements to maps for JSON
	achievementMaps := convertAchievements(data.Achievements)

	// Phase 1: Analyze
	var analysisResp llm.AnalysisResponse
	analysisResp, err = runAnalysisPhase(ctx, client, jobDescription, achievementMaps)
	if err != nil {
		return err
	}

	// Extract company/role and create output directory
	finalCompany, finalRole := extractCompanyAndRole(company, role, analysisResp.JDAnalysis)
	baseOutDir := getBaseOutputDir(cfg)
	outDir, err = createCompanyOutputDir(baseOutDir, finalCompany)
	if err != nil {
		return err
	}

	// Filter top achievements (score >= 0.6)
	topAchievements := filterTopAchievements(achievementMaps, analysisResp.RankedAchievements, 0.6)

	// Retrieve RAG context from past evaluations
	var ragContext string
	ragContext, err = retrieveRAGContext(ctx, baseOutDir, finalCompany, finalRole, jobDescription)
	if err != nil {
		// Log but don't fail if RAG retrieval fails
		if getVerbose() {
			fmt.Printf("Warning: RAG retrieval failed: %v\n", err)
		}
		ragContext = ""
	}

	// Phase 2: Generate
	var genResp llm.GenerationResponse
	genResp, err = runGenerationPhase(ctx, client, jobDescription, finalCompany, finalRole, coverLetterContext, ragContext, cfg.CompleteResumeURL, analysisResp.JDAnalysis, topAchievements, data)
	if err != nil {
		return err
	}

	// Generate filenames
	filenames := buildFilenames(outDir, cfg.Name, finalCompany, finalRole, jobID)

	// Write markdown files first (before evaluation)
	err = writeInitialFiles(genResp, jobDescription, filenames)
	if err != nil {
		return err
	}

	// Phase 3: Hybrid evaluation and fix
	finalEvaluation := runEvaluationPhase(ctx, cfg, finalCompany, finalRole, filenames, data)

	// Phase 4: Save evaluation to RAG for future learning
	if err == nil {
		ragErr := saveEvaluationToRAG(ctx, baseOutDir, finalCompany, finalRole, finalEvaluation, filenames)
		if ragErr != nil {
			if getVerbose() {
				fmt.Printf("Warning: Failed to save evaluation to RAG: %v\n", ragErr)
			}
		} else if getVerbose() {
			fmt.Println("✓ Evaluation saved to RAG for future learning")
		}
	}

	// Phase 5: Render PDFs (unless --skip-pdf)
	if !skipPDF {
		err = renderPDFs(filenames.resumeMD, filenames.resumePDF, filenames.coverMD, filenames.coverPDF, cfg.Pandoc.TemplatePath, cfg.Pandoc.ClassFile)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("\nMarkdown files saved (PDF generation skipped):")
		fmt.Printf("  Resume: %s\n", filenames.resumeMD)
		fmt.Printf("  Cover letter: %s\n", filenames.coverMD)
	}

	return err
}

func runAnalysisPhase(ctx context.Context, client *llm.Client, jobDescription string, achievementMaps []map[string]interface{}) (analysisResp llm.AnalysisResponse, err error) {
	// Show spinner during analysis unless in verbose mode
	var analysisSpinner *spinner
	if !getVerbose() {
		analysisSpinner = newSpinner("Analyzing job description with Claude API...")
		analysisSpinner.start()
	} else {
		fmt.Println("Analyzing job description with Claude API...")
	}

	analysisResp, err = client.Analyze(ctx, jobDescription, achievementMaps)

	if analysisSpinner != nil {
		analysisSpinner.stopSpinner()
	}

	if err != nil {
		err = errors.Wrap(err, "Claude API analysis failed")
		return analysisResp, err
	}

	if !getVerbose() {
		fmt.Println("✓ Analysis complete")
	}

	logAnalysisResults(analysisResp)

	return analysisResp, err
}

func runGenerationPhase(ctx context.Context, client *llm.Client, jobDescription, company, role, context, ragContext, completeResumeURL string, analysis llm.JDAnalysis, achievements []map[string]interface{}, data summaries.Data) (genResp llm.GenerationResponse, err error) {
	genReq := buildGenerationRequest(jobDescription, company, role, context, ragContext, completeResumeURL, analysis, achievements, data)

	// Show spinner during generation unless in verbose mode
	var genSpinner *spinner
	if !getVerbose() {
		genSpinner = newSpinner("Generating tailored resume and cover letter...")
		genSpinner.start()
	} else {
		fmt.Println("Generating tailored resume and cover letter...")
	}

	genResp, err = client.Generate(ctx, genReq)

	if genSpinner != nil {
		genSpinner.stopSpinner()
	}

	if err != nil {
		err = errors.Wrap(err, "Claude API generation failed")
		return genResp, err
	}

	if !getVerbose() {
		fmt.Println("✓ Generation complete")
	}

	return genResp, err
}

func writeMarkdownFiles(resume, coverLetter, resumeMD, coverMD string) (err error) {
	resumeContent := unescapeNewlines(resume)
	err = renderer.WriteMarkdown(resumeContent, resumeMD)
	if err != nil {
		err = errors.Wrap(err, "failed to write resume markdown")
		return err
	}

	coverContent := unescapeNewlines(coverLetter)
	err = renderer.WriteMarkdown(coverContent, coverMD)
	if err != nil {
		err = errors.Wrap(err, "failed to write cover letter markdown")
		return err
	}

	return err
}

func buildGenerationRequest(jobDescription, company, role, context, ragContext, completeResumeURL string, analysis llm.JDAnalysis, achievements []map[string]interface{}, data summaries.Data) (genReq llm.GenerationRequest) {
	genReq = llm.GenerationRequest{
		JobDescription:     jobDescription,
		Company:            company,
		Role:               role,
		HiringManager:      analysis.HiringManager,
		JDSummary:          buildJDSummary(analysis),
		CoverLetterContext: context,
		RAGContext:         ragContext,
		CompleteResumeURL:  completeResumeURL,
		Achievements:       achievements,
		Profile:            profileToMap(data.Profile),
		Skills:             skillsToMap(data.Skills),
		Projects:           projectsToMaps(data.OpensourceProjects),
		CompanyURLs:        data.CompanyURLs,
	}
	return genReq
}

func convertAchievements(achievements []summaries.Achievement) (maps []map[string]interface{}) {
	maps = make([]map[string]interface{}, len(achievements))
	for i, achievement := range achievements {
		maps[i] = achievementToMap(achievement)
	}
	return maps
}

func fetchAndLogJD(jdInput string) (jobDescription string, err error) {
	if getVerbose() {
		fmt.Printf("Loading job description from: %s\n", jdInput)
	}

	jobDescription, err = jd.Fetch(jdInput)
	if err != nil {
		// If fetching failed, offer to accept manual input
		fmt.Printf("\nWarning: Failed to fetch job description from URL: %v\n", err)
		fmt.Println("This often happens with JavaScript-rendered pages (Lever, Workable, etc.)")
		fmt.Println("\nPlease paste the job description text below.")
		fmt.Println("When finished, press Ctrl+D (Unix/Mac) or Ctrl+Z then Enter (Windows):")
		fmt.Println()

		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if scanner.Err() != nil {
			err = errors.Wrap(scanner.Err(), "failed to read job description from stdin")
			return jobDescription, err
		}

		jobDescription = strings.Join(lines, "\n")
		jobDescription = strings.TrimSpace(jobDescription)

		if jobDescription == "" {
			err = errors.New("no job description provided")
			return jobDescription, err
		}

		fmt.Printf("\nJob description received (%d characters)\n", len(jobDescription))
		err = nil
		return jobDescription, err
	}

	if getVerbose() {
		fmt.Printf("Job description loaded (%d characters)\n", len(jobDescription))
	}

	return jobDescription, err
}

func loadAndLogSummaries(path string) (data summaries.Data, err error) {
	if getVerbose() {
		fmt.Printf("Loading summaries from: %s\n", path)
	}

	data, err = summaries.Load(path)
	if err != nil {
		err = errors.Wrap(err, "failed to load summaries")
		return data, err
	}

	if getVerbose() {
		fmt.Printf("Loaded %d achievements\n", len(data.Achievements))
		fmt.Println("Analyzing job description with Claude API...")
	}

	return data, err
}

func logAnalysisResults(resp llm.AnalysisResponse) {
	if !getVerbose() {
		return
	}

	fmt.Printf("Analysis complete. Top requirements:\n")
	for _, req := range resp.JDAnalysis.KeyRequirements {
		fmt.Printf("  - %s\n", req)
	}
	fmt.Printf("Role focus: %s\n", resp.JDAnalysis.RoleFocus)
}

func extractCompanyAndRole(company, role string, analysis llm.JDAnalysis) (finalCompany, finalRole string) {
	finalCompany = company
	if finalCompany == "" {
		finalCompany = analysis.CompanyName
		if getVerbose() && finalCompany != "" {
			fmt.Printf("Extracted company from JD: %s\n", finalCompany)
		}
	}

	// Prompt for company if still empty
	if finalCompany == "" {
		finalCompany = promptForInput("Company name")
	}

	finalRole = role
	if finalRole == "" {
		finalRole = analysis.RoleTitle
		if getVerbose() && finalRole != "" {
			fmt.Printf("Extracted role from JD: %s\n", finalRole)
		}
	}

	// Prompt for role if still empty
	if finalRole == "" {
		finalRole = promptForInput("Role title")
	}

	return finalCompany, finalRole
}

func promptForInput(fieldName string) (input string) {
	fmt.Printf("%s could not be extracted from job description.\n", fieldName)
	fmt.Printf("Please enter %s: ", strings.ToLower(fieldName))

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input = strings.TrimSpace(scanner.Text())
	}

	return input
}

// spinner provides a simple text-based progress indicator.
type spinner struct {
	message string
	stop    chan bool
	done    chan bool
	mu      sync.Mutex
	active  bool
}

func newSpinner(message string) (s *spinner) {
	s = &spinner{
		message: message,
		stop:    make(chan bool),
		done:    make(chan bool),
	}
	return s
}

func (s *spinner) start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		chars := []string{"|", "/", "-", "\\"}
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		fmt.Printf("%s ", s.message)
		for {
			select {
			case <-s.stop:
				// Clear the line and ensure cursor is at start of new line
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.message)+2))
				s.done <- true
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s", s.message, chars[i%len(chars)])
				i++
			}
		}
	}()
}

func (s *spinner) stopSpinner() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.stop <- true
	<-s.done

	s.mu.Lock()
	s.active = false
	s.mu.Unlock()
}

func createCompanyOutputDir(baseOutDir, company string) (outDir string, err error) {
	companyDir := sanitizeFilename(company)
	outDir = filepath.Join(baseOutDir, companyDir)
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		err = errors.Wrapf(err, "failed to create output directory: %s", outDir)
		return outDir, err
	}
	return outDir, err
}

func achievementToMap(a summaries.Achievement) (result map[string]interface{}) {
	result = map[string]interface{}{
		"id":         a.ID,
		"company":    a.Company,
		"role":       a.Role,
		"dates":      a.Dates,
		"title":      a.Title,
		"challenge":  a.Challenge,
		"execution":  a.Execution,
		"impact":     a.Impact,
		"metrics":    a.Metrics,
		"keywords":   a.Keywords,
		"categories": a.Categories,
	}
	return result
}

func profileToMap(p summaries.Profile) (result map[string]interface{}) {
	result = map[string]interface{}{
		"name":     p.Name,
		"title":    p.Title,
		"location": p.Location,
		"motto":    p.Motto,
		"profiles": p.Profiles,
	}
	return result
}

func skillsToMap(s summaries.Skills) (result map[string]interface{}) {
	result = map[string]interface{}{
		"languages":  s.Languages,
		"cloud":      s.Cloud,
		"kubernetes": s.Kubernetes,
		"security":   s.Security,
		"databases":  s.Databases,
		"cicd":       s.CICD,
		"networks":   s.Networks,
	}
	return result
}

func projectsToMaps(projects []summaries.OpensourceProject) (result []map[string]interface{}) {
	result = make([]map[string]interface{}, len(projects))
	for i, project := range projects {
		result[i] = map[string]interface{}{
			"name":        project.Name,
			"url":         project.URL,
			"description": project.Description,
			"recognition": project.Recognition,
		}
	}
	return result
}

func filterTopAchievements(achievements []map[string]interface{}, ranked []llm.RankedAchievement, threshold float64) (filtered []map[string]interface{}) {
	filtered = make([]map[string]interface{}, 0)

	// Create map for quick lookup
	achievementMap := make(map[string]map[string]interface{})
	for _, achievement := range achievements {
		if id, ok := achievement["id"].(string); ok {
			achievementMap[id] = achievement
		}
	}

	// Add achievements above threshold
	for _, ranked := range ranked {
		if ranked.RelevanceScore >= threshold {
			if achievement, found := achievementMap[ranked.AchievementID]; found {
				filtered = append(filtered, achievement)
			}
		}
	}

	return filtered
}

func buildJDSummary(analysis llm.JDAnalysis) (summary string) {
	reqJSON, _ := json.Marshal(analysis.KeyRequirements)
	techJSON, _ := json.Marshal(analysis.TechnicalStack)

	summary = fmt.Sprintf(`Key Requirements: %s
Technical Stack: %s
Role Focus: %s
Company Signals: %s`,
		string(reqJSON), string(techJSON),
		analysis.RoleFocus, analysis.CompanySignals)

	return summary
}

func sanitizeFilename(name string) (sanitized string) {
	// Remove common company suffixes
	suffixes := []string{
		" LLC", " llc",
		" Inc.", " inc.",
		" Inc", " inc",
		" Corporation", " corporation",
		" Corp.", " corp.",
		" Corp", " corp",
		" Limited", " limited",
		" Ltd.", " ltd.",
		" Ltd", " ltd",
		" Co.", " co.",
		" Co", " co",
		", LLC", ", llc",
		", Inc.", ", inc.",
		", Inc", ", inc",
	}

	sanitized = name
	for _, suffix := range suffixes {
		sanitized = strings.TrimSuffix(sanitized, suffix)
	}

	// Convert to lowercase
	sanitized = strings.ToLower(sanitized)

	// Replace spaces and special chars with hyphens
	sanitized = strings.Map(func(r rune) (result rune) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = r
			return result
		}
		result = '-'
		return result
	}, sanitized)

	// Remove consecutive hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Trim hyphens from ends
	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}

// unescapeNewlines converts literal \n to actual newlines and removes emojis.
func unescapeNewlines(text string) (unescaped string) {
	// First unescape newlines
	unescaped = strings.ReplaceAll(text, "\\n", "\n")

	// Remove all emojis (LaTeX can't handle them)
	// Filter out runes in emoji ranges
	result := strings.Builder{}
	for _, r := range unescaped {
		// Skip emoji ranges (simplified - covers most common emojis)
		if r >= 0x1F300 && r <= 0x1F9FF { // Miscellaneous Symbols and Pictographs, Emoticons, etc.
			continue
		}
		if r >= 0x2600 && r <= 0x26FF { // Miscellaneous Symbols
			continue
		}
		if r >= 0x2700 && r <= 0x27BF { // Dingbats
			continue
		}
		result.WriteRune(r)
	}
	unescaped = result.String()

	// Clean up any double spaces left by emoji removal
	for strings.Contains(unescaped, "  ") {
		unescaped = strings.ReplaceAll(unescaped, "  ", " ")
	}

	return unescaped
}

// setupGeneration handles initial setup: config loading, JD fetching, and summaries loading.
func setupGeneration(jdInput string) (cfg config.Config, jobDescription string, data summaries.Data, client *llm.Client, err error) {
	// Load configuration
	cfg, err = config.Load(getConfigFile())
	if err != nil {
		err = errors.Wrap(err, "failed to load config")
		return cfg, jobDescription, data, client, err
	}

	// Fetch job description
	jobDescription, err = fetchAndLogJD(jdInput)
	if err != nil {
		return cfg, jobDescription, data, client, err
	}

	// Load summaries
	data, err = loadAndLogSummaries(cfg.SummariesLocation)
	if err != nil {
		return cfg, jobDescription, data, client, err
	}

	// Create client
	client = llm.NewClient(cfg.AnthropicAPIKey, cfg.GetGenerationModel())

	return cfg, jobDescription, data, client, err
}

// getBaseOutputDir returns the base output directory from flag or config.
func getBaseOutputDir(cfg config.Config) (baseOutDir string) {
	baseOutDir = outputDir
	if baseOutDir == "" {
		baseOutDir = cfg.Defaults.OutputDir
	}
	return baseOutDir
}

// retrieveRAGContext retrieves lessons learned from past evaluations.
func retrieveRAGContext(ctx context.Context, outputDir, company, role, jdText string) (context string, err error) {
	// Create indexer
	var indexer *rag.Indexer
	indexer, err = rag.NewIndexer(outputDir)
	if err != nil {
		return context, err
	}

	// Create retriever
	retriever := rag.NewRetriever(indexer)

	// Retrieve relevant evaluations
	var ragCtx rag.RAGContext
	ragCtx, err = retriever.Retrieve(ctx, company, role, jdText)
	if err != nil {
		return context, err
	}

	// Format for prompt
	context = retriever.FormatForPrompt(ragCtx)

	return context, err
}

// saveEvaluationToRAG saves the evaluation results for future learning.
func saveEvaluationToRAG(ctx context.Context, outputDir, company, role string, evalResp llm.EvaluationResponse, filenames outputFilenames) (err error) {
	// Build evaluation record
	evaluation := rag.Evaluation{
		Company:     company,
		Role:        role,
		GeneratedAt: time.Now(),
		EvaluatedAt: time.Now(),
		Scores: rag.Scores{
			Resume: rag.ResumeScore{
				Total: calculateResumeScore(evalResp),
				AntiFabrication: rag.AntiFabricationScore{
					Score:      len(evalResp.ResumeViolations),
					Violations: evalResp.ResumeViolations,
				},
				WeakQuantifications: rag.WeakQuantificationsScore{
					Score:  len(evalResp.WeakQuantifications),
					Issues: evalResp.WeakQuantifications,
				},
				Accuracy: rag.AccuracyScore{
					Score:               100, // Placeholder
					VerifiedMetrics:     evalResp.VerifiedMetrics,
					CompanyDatesCorrect: evalResp.CompanyDatesCorrect,
					RoleTitlesCorrect:   evalResp.RoleTitlesCorrect,
					YearsExpCorrect:     evalResp.YearsExpCorrect,
				},
			},
			CoverLetter: rag.CoverLetterScore{
				Total: calculateCoverLetterScore(evalResp),
				DomainClaims: rag.DomainClaimsScore{
					Score:      len(evalResp.CoverLetterViolations),
					Violations: evalResp.CoverLetterViolations,
				},
				Tone: rag.ToneScore{
					Score:    100, // Placeholder
					Feedback: []string{},
				},
			},
			Overall: calculateOverallScore(evalResp),
		},
		JDMatch:    evalResp.JDMatch,
		Lessons:    evalResp.LessonsLearned,
		RAGContext: formatRAGContext(evalResp),
		Version:    "1.0.0", // TODO: get from build version
	}

	// Write evaluation JSON file
	evalFilename := filepath.Join(filepath.Dir(filenames.resumeMD), sanitizeFilename(company)+"-"+sanitizeFilename(role)+".evaluation.json")
	var evalBytes []byte
	evalBytes, err = json.MarshalIndent(evaluation, "", "  ")
	if err != nil {
		err = errors.Wrap(err, "failed to marshal evaluation")
		return err
	}

	err = os.WriteFile(evalFilename, evalBytes, 0644)
	if err != nil {
		err = errors.Wrap(err, "failed to write evaluation file")
		return err
	}

	if getVerbose() {
		fmt.Printf("✓ Saved evaluation to %s\n", evalFilename)
	}

	// Rebuild RAG index
	var indexer *rag.Indexer
	indexer, err = rag.NewIndexer(outputDir)
	if err != nil {
		err = errors.Wrap(err, "failed to create RAG indexer")
		return err
	}

	var count int
	count, err = indexer.Index(ctx)
	if err != nil {
		err = errors.Wrap(err, "failed to rebuild RAG index")
		return err
	}

	if getVerbose() {
		fmt.Printf("✓ Rebuilt RAG index (%d evaluations indexed)\n", count)
	}

	return err
}

// calculateResumeScore calculates a simple resume score based on violations.
func calculateResumeScore(evalResp llm.EvaluationResponse) (score int) {
	score = 100
	for _, v := range evalResp.ResumeViolations {
		switch v.Severity {
		case "critical":
			score -= 30
		case "major":
			score -= 15
		case "minor":
			score -= 5
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// calculateCoverLetterScore calculates cover letter score.
func calculateCoverLetterScore(evalResp llm.EvaluationResponse) (score int) {
	score = 100
	for _, v := range evalResp.CoverLetterViolations {
		switch v.Severity {
		case "critical":
			score -= 30
		case "major":
			score -= 15
		case "minor":
			score -= 5
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// calculateOverallScore calculates overall weighted score.
func calculateOverallScore(evalResp llm.EvaluationResponse) (score int) {
	// Weighted average: resume 70%, cover letter 30%
	resumeScore := calculateResumeScore(evalResp)
	coverScore := calculateCoverLetterScore(evalResp)
	score = (resumeScore * 7 / 10) + (coverScore * 3 / 10)
	return score
}

// formatRAGContext creates a summary for RAG retrieval.
func formatRAGContext(evalResp llm.EvaluationResponse) (context string) {
	var builder strings.Builder

	// Add lessons learned
	if len(evalResp.LessonsLearned) > 0 {
		builder.WriteString("Lessons Learned:\n")
		for _, lesson := range evalResp.LessonsLearned {
			builder.WriteString("- ")
			builder.WriteString(lesson)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Add violation summaries
	if len(evalResp.ResumeViolations) > 0 {
		builder.WriteString("Resume Violations:\n")
		for _, v := range evalResp.ResumeViolations {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n", v.Rule, v.Severity, v.Fabricated))
		}
		builder.WriteString("\n")
	}

	if len(evalResp.CoverLetterViolations) > 0 {
		builder.WriteString("Cover Letter Violations:\n")
		for _, v := range evalResp.CoverLetterViolations {
			builder.WriteString(fmt.Sprintf("- %s (%s): %s\n", v.Rule, v.Severity, v.Fabricated))
		}
	}

	context = builder.String()
	return context
}

// outputFilenames holds all output file paths.
type outputFilenames struct {
	resumeMD  string
	resumePDF string
	coverMD   string
	coverPDF  string
	jdTXT     string
}

// buildFilenames generates all output file paths.
func buildFilenames(outDir, name, company, role, jobID string) (filenames outputFilenames) {
	sanitizedName := sanitizeFilename(name)
	sanitizedCompany := sanitizeFilename(company)

	// Truncate role to first 4 words to keep filename reasonable
	roleWords := strings.Fields(role)
	if len(roleWords) > 4 {
		role = strings.Join(roleWords[:4], " ")
	}
	sanitizedRole := sanitizeFilename(role)

	// Build base filename with optional job ID
	baseFilename := sanitizedName + "-" + sanitizedCompany + "-" + sanitizedRole
	if jobID != "" {
		sanitizedJobID := sanitizeFilename(jobID)
		baseFilename = baseFilename + "-" + sanitizedJobID
	}

	filenames = outputFilenames{
		resumeMD:  filepath.Join(outDir, baseFilename+"-resume.md"),
		resumePDF: filepath.Join(outDir, baseFilename+"-resume.pdf"),
		coverMD:   filepath.Join(outDir, baseFilename+"-cover.md"),
		coverPDF:  filepath.Join(outDir, baseFilename+"-cover.pdf"),
		jdTXT:     filepath.Join(outDir, baseFilename+"-jd.txt"),
	}

	return filenames
}

// writeInitialFiles writes markdown and JD files (before evaluation).
func writeInitialFiles(genResp llm.GenerationResponse, jobDescription string, filenames outputFilenames) (err error) {
	if getVerbose() {
		fmt.Println("Writing initial markdown files...")
	}

	// Write job description text file
	err = os.WriteFile(filenames.jdTXT, []byte(jobDescription), 0644)
	if err != nil {
		err = errors.Wrap(err, "failed to write job description file")
		return err
	}

	// Write markdown files
	err = writeMarkdownFiles(genResp.Resume, genResp.CoverLetter, filenames.resumeMD, filenames.coverMD)
	if err != nil {
		return err
	}

	if getVerbose() {
		fmt.Println("Initial markdown files written")
	}

	return err
}

// runEvaluationPhase runs the evaluation phase based on auto-fix setting.
func runEvaluationPhase(ctx context.Context, cfg config.Config, company, role string, filenames outputFilenames, data summaries.Data) (finalEval llm.EvaluationResponse) {
	var err error
	if autoFix {
		finalEval, err = runHybridEvaluationAndFix(ctx, cfg, company, role, filenames, data)
		if err != nil {
			fmt.Printf("Warning: Evaluation/fix phase failed: %v\n", err)
			fmt.Println("Continuing with generated content...")
		}
	} else {
		// If auto-fix is disabled, just evaluate once
		finalEval, err = runEvaluation(ctx, cfg, company, role, filenames, data)
		if err != nil {
			fmt.Printf("Warning: Evaluation failed: %v\n", err)
		}
	}
	return finalEval
}

// runHybridEvaluationAndFix implements the hybrid approach: eval #1 → fix → eval #2.
func runHybridEvaluationAndFix(ctx context.Context, cfg config.Config, company, role string, filenames outputFilenames, data summaries.Data) (finalEval llm.EvaluationResponse, err error) {
	// Evaluation #1: Detect violations
	fmt.Println("Phase 3a: Evaluating generated content (detecting violations)...")
	var evalResp llm.EvaluationResponse
	evalResp, err = runEvaluation(ctx, cfg, company, role, filenames, data)
	if err != nil {
		return finalEval, err
	}

	// Check if we have violations to fix
	totalViolations := len(evalResp.ResumeViolations) + len(evalResp.CoverLetterViolations)
	if totalViolations == 0 {
		fmt.Println("✓ No violations found - content looks good!")
		finalEval = evalResp
		return finalEval, err
	}

	fmt.Printf("Found %d violations, applying automated fixes...\n", totalViolations)

	if getVerbose() {
		fmt.Println("\nViolations detected:")
		for i, v := range evalResp.ResumeViolations {
			fmt.Printf("  [Resume %d] %s (severity: %s)\n", i+1, v.Rule, v.Severity)
			fmt.Printf("    Fabricated: %s\n", v.Fabricated)
			if v.SuggestedFix != "" {
				fmt.Printf("    Suggested fix: %s\n", v.SuggestedFix)
			}
		}
		for i, v := range evalResp.CoverLetterViolations {
			fmt.Printf("  [Cover %d] %s (severity: %s)\n", i+1, v.Rule, v.Severity)
			fmt.Printf("    Fabricated: %s\n", v.Fabricated)
		}
		fmt.Println()
	}

	// Apply and write fixes
	fmt.Println("Phase 3b: Applying automated fixes...")
	err = applyAndWriteFixes(filenames, evalResp)
	if err != nil {
		return finalEval, err
	}

	// Evaluation #2: Verify fixes and get final quality score
	fmt.Println("Phase 3c: Re-evaluating fixed content (verification)...")
	finalEval, err = runEvaluation(ctx, cfg, company, role, filenames, data)
	if err != nil {
		return finalEval, err
	}

	// Check if any violations remain
	remainingViolations := len(finalEval.ResumeViolations) + len(finalEval.CoverLetterViolations)
	if remainingViolations == 0 {
		fmt.Println("✓ All violations fixed! Content ready for PDF generation.")
	} else {
		fmt.Printf("⚠ Warning: %d violations remain after automated fixes\n", remainingViolations)
		if getVerbose() {
			fmt.Println("\nRemaining violations:")
			for i, v := range finalEval.ResumeViolations {
				fmt.Printf("  [Resume %d] %s: %s\n", i+1, v.Rule, v.Fabricated)
			}
			for i, v := range finalEval.CoverLetterViolations {
				fmt.Printf("  [Cover %d] %s: %s\n", i+1, v.Rule, v.Fabricated)
			}
		}
	}

	return finalEval, err
}

// runEvaluation runs the evaluation phase.
func runEvaluation(ctx context.Context, cfg config.Config, company, role string, filenames outputFilenames, data summaries.Data) (evalResp llm.EvaluationResponse, err error) {
	// Read the markdown files we just wrote
	var resumeBytes []byte
	resumeBytes, err = os.ReadFile(filenames.resumeMD)
	if err != nil {
		err = errors.Wrap(err, "failed to read resume markdown for evaluation")
		return evalResp, err
	}

	var coverBytes []byte
	coverBytes, err = os.ReadFile(filenames.coverMD)
	if err != nil {
		err = errors.Wrap(err, "failed to read cover letter markdown for evaluation")
		return evalResp, err
	}

	var jdBytes []byte
	jdBytes, err = os.ReadFile(filenames.jdTXT)
	if err != nil {
		err = errors.Wrap(err, "failed to read job description for evaluation")
		return evalResp, err
	}

	// Build evaluation request
	achievementsJSON, _ := json.Marshal(data.Achievements)
	skillsJSON, _ := json.Marshal(data.Skills)
	profileJSON, _ := json.Marshal(data.Profile)

	evalReq := llm.EvaluationRequest{
		Company:            company,
		Role:               role,
		JobDescription:     string(jdBytes),
		Resume:             string(resumeBytes),
		CoverLetter:        string(coverBytes),
		SourceAchievements: string(achievementsJSON),
		SourceSkills:       string(skillsJSON),
		SourceProfile:      string(profileJSON),
	}

	// Run evaluation with spinner
	var evalSpinner *spinner
	if !getVerbose() {
		evalSpinner = newSpinner("Evaluating generated content...")
		evalSpinner.start()
	} else {
		fmt.Println("Evaluating generated content...")
	}

	evaluator, _ := llm.NewEvaluator(cfg.AnthropicAPIKey, cfg.GetEvaluationModel())
	evalResp, err = evaluator.Evaluate(ctx, evalReq)

	if evalSpinner != nil {
		evalSpinner.stopSpinner()
	}

	if err != nil {
		err = errors.Wrap(err, "evaluation failed")
		return evalResp, err
	}

	if !getVerbose() {
		fmt.Println("✓ Evaluation complete")
	}

	return evalResp, err
}

// applyAndWriteFixes applies fixes and writes updated markdown files.
func applyAndWriteFixes(filenames outputFilenames, evalResp llm.EvaluationResponse) (err error) {
	// Read current markdown
	var resumeBytes []byte
	resumeBytes, err = os.ReadFile(filenames.resumeMD)
	if err != nil {
		err = errors.Wrap(err, "failed to read resume for fixing")
		return err
	}

	var coverBytes []byte
	coverBytes, err = os.ReadFile(filenames.coverMD)
	if err != nil {
		err = errors.Wrap(err, "failed to read cover letter for fixing")
		return err
	}

	// Apply fixes
	fixer := llm.NewFixer()
	var fixedResume string
	var fixedCover string
	var appliedFixes []string
	fixedResume, fixedCover, appliedFixes, err = fixer.ApplyFixes(string(resumeBytes), string(coverBytes), evalResp)
	if err != nil {
		err = errors.Wrap(err, "failed to apply fixes")
		return err
	}

	// Write fixed files if any fixes were applied
	if len(appliedFixes) == 0 {
		if getVerbose() {
			fmt.Println("No fixes could be automatically applied")
		}
		return err
	}

	fmt.Printf("✓ Applied %d automated fixes:\n", len(appliedFixes))
	for _, fix := range appliedFixes {
		fmt.Printf("  - %s\n", fix)
	}

	err = writeFixedMarkdown(filenames, fixedResume, fixedCover)
	return err
}

// writeFixedMarkdown writes the fixed markdown files.
func writeFixedMarkdown(filenames outputFilenames, fixedResume, fixedCover string) (err error) {
	err = os.WriteFile(filenames.resumeMD, []byte(fixedResume), 0644)
	if err != nil {
		err = errors.Wrap(err, "failed to write fixed resume")
		return err
	}

	err = os.WriteFile(filenames.coverMD, []byte(fixedCover), 0644)
	if err != nil {
		err = errors.Wrap(err, "failed to write fixed cover letter")
		return err
	}

	if getVerbose() {
		fmt.Println("Fixed markdown files written")
	}

	return err
}

// renderPDFs renders markdown files to PDFs.
func renderPDFs(resumeMD, resumePDF, coverMD, coverPDF, templatePath, classPath string) (err error) {
	if getVerbose() {
		fmt.Println("Rendering PDFs...")
	}

	// Render resume PDF
	err = renderer.RenderPDF(resumeMD, resumePDF, templatePath, classPath)
	if err != nil {
		fmt.Printf("Warning: Failed to render resume PDF: %v\n", err)
		fmt.Printf("Resume markdown saved at: %s\n", resumeMD)
	} else {
		fmt.Printf("Resume PDF saved at: %s\n", resumePDF)
	}

	// Render cover letter PDF
	err = renderer.RenderPDF(coverMD, coverPDF, templatePath, classPath)
	if err != nil {
		fmt.Printf("Warning: Failed to render cover letter PDF: %v\n", err)
		fmt.Printf("Cover letter markdown saved at: %s\n", coverMD)
	} else {
		fmt.Printf("Cover letter PDF saved at: %s\n", coverPDF)
	}

	// Clean up markdown files unless --keep-markdown is set
	if !keepMarkdown {
		err = renderer.CleanupMarkdown(resumeMD, coverMD)
		if err != nil {
			fmt.Printf("Warning: Failed to clean up markdown files: %v\n", err)
		}
	}

	fmt.Println("\nGeneration complete!")

	// Ensure stdout is flushed before exiting
	os.Stdout.Sync()

	return err
}
