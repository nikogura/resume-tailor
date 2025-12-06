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
var generateCmd = &cobra.Command{
	Use:   "generate <jd-file-or-url>",
	Short: "Generate tailored resume and cover letter",
	Long: `Generate a tailored resume and cover letter based on a job description.

The job description can be provided as:
- A file path (e.g., jd.txt)
- A URL (e.g., https://example.com/jobs/123)

Example:
  resume-tailor generate jd.txt --company "Acme Corp" --role "Staff Engineer"
  resume-tailor generate https://example.com/jobs/123 --company "Acme" --role "SRE"`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

//nolint:gochecknoinits // Cobra boilerplate
func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&company, "company", "", "Company name (extracted from JD if not provided)")
	generateCmd.Flags().StringVar(&role, "role", "", "Role title (extracted from JD if not provided)")
	generateCmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default from config)")
	generateCmd.Flags().BoolVar(&keepMarkdown, "keep-markdown", false, "Keep markdown files after PDF generation")
	generateCmd.Flags().StringVar(&coverLetterContext, "context", "", "Additional context for cover letter generation")
}

func runGenerate(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	jdInput := args[0]

	// Load configuration
	var cfg config.Config
	cfg, err = config.Load(getConfigFile())
	if err != nil {
		err = errors.Wrap(err, "failed to load config")
		return err
	}

	// Use output dir from flag or config
	baseOutDir := outputDir
	if baseOutDir == "" {
		baseOutDir = cfg.Defaults.OutputDir
	}

	// Fetch job description
	var jobDescription string
	jobDescription, err = fetchAndLogJD(jdInput)
	if err != nil {
		return err
	}

	// Load summaries
	var data summaries.Data
	data, err = loadAndLogSummaries(cfg.SummariesLocation)
	if err != nil {
		return err
	}

	// Convert achievements to maps for JSON
	achievementMaps := convertAchievements(data.Achievements)

	// Phase 1: Analyze
	client := llm.NewClient(cfg.AnthropicAPIKey)

	var analysisResp llm.AnalysisResponse
	analysisResp, err = runAnalysisPhase(ctx, client, jobDescription, achievementMaps)
	if err != nil {
		return err
	}

	// Use extracted company and role if not provided
	finalCompany, finalRole := extractCompanyAndRole(company, role, analysisResp.JDAnalysis)

	// Create company subdirectory and get output dir
	var outDir string
	outDir, err = createCompanyOutputDir(baseOutDir, finalCompany)
	if err != nil {
		return err
	}

	// Filter top achievements (score >= 0.6)
	topAchievements := filterTopAchievements(achievementMaps, analysisResp.RankedAchievements, 0.6)

	// Phase 2: Generate
	var genResp llm.GenerationResponse
	genResp, err = runGenerationPhase(ctx, client, jobDescription, finalCompany, finalRole, coverLetterContext, analysisResp.JDAnalysis, topAchievements, data)
	if err != nil {
		return err
	}

	// Write and render output files
	err = writeAndRenderOutput(genResp, outDir, cfg.Name, finalCompany, finalRole, cfg.Pandoc.TemplatePath, cfg.Pandoc.ClassFile)
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

func runGenerationPhase(ctx context.Context, client *llm.Client, jobDescription, company, role, context string, analysis llm.JDAnalysis, achievements []map[string]interface{}, data summaries.Data) (genResp llm.GenerationResponse, err error) {
	genReq := buildGenerationRequest(jobDescription, company, role, context, analysis, achievements, data)

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

func writeAndRenderOutput(genResp llm.GenerationResponse, outDir, name, company, role, templatePath, classPath string) (err error) {
	// Generate output filenames: name-company-role-{resume,cover}.pdf
	sanitizedName := sanitizeFilename(name)
	sanitizedCompany := sanitizeFilename(company)

	// Truncate role to first 4 words to keep filename reasonable
	roleWords := strings.Fields(role)
	if len(roleWords) > 4 {
		role = strings.Join(roleWords[:4], " ")
	}
	sanitizedRole := sanitizeFilename(role)
	baseFilename := sanitizedName + "-" + sanitizedCompany + "-" + sanitizedRole

	resumeMD := filepath.Join(outDir, baseFilename+"-resume.md")
	resumePDF := filepath.Join(outDir, baseFilename+"-resume.pdf")
	coverMD := filepath.Join(outDir, baseFilename+"-cover.md")
	coverPDF := filepath.Join(outDir, baseFilename+"-cover.pdf")

	if getVerbose() {
		fmt.Println("Writing markdown files...")
	}

	// Write markdown files
	err = writeMarkdownFiles(genResp.Resume, genResp.CoverLetter, resumeMD, coverMD)
	if err != nil {
		return err
	}

	if getVerbose() {
		fmt.Println("Rendering PDFs...")
	}

	err = renderAndCleanupGenerate(resumeMD, resumePDF, coverMD, coverPDF, templatePath, classPath)
	return err
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

func renderAndCleanupGenerate(resumeMD, resumePDF, coverMD, coverPDF, templatePath, classPath string) (err error) {
	// Render PDFs
	err = renderer.RenderPDF(resumeMD, resumePDF, templatePath, classPath)
	if err != nil {
		fmt.Printf("Warning: Failed to render resume PDF: %v\n", err)
		fmt.Printf("Resume markdown saved at: %s\n", resumeMD)
	} else {
		fmt.Printf("Resume PDF saved at: %s\n", resumePDF)
	}

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

func buildGenerationRequest(jobDescription, company, role, context string, analysis llm.JDAnalysis, achievements []map[string]interface{}, data summaries.Data) (genReq llm.GenerationRequest) {
	genReq = llm.GenerationRequest{
		JobDescription:     jobDescription,
		Company:            company,
		Role:               role,
		HiringManager:      analysis.HiringManager,
		JDSummary:          buildJDSummary(analysis),
		CoverLetterContext: context,
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
