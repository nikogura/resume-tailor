package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/nikogura/resume-tailor/pkg/config"
	"github.com/nikogura/resume-tailor/pkg/llm"
	"github.com/nikogura/resume-tailor/pkg/renderer"
	"github.com/nikogura/resume-tailor/pkg/summaries"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra boilerplate
var generalOutputDir string

//nolint:gochecknoglobals // Cobra boilerplate
var generalKeepMarkdown bool

//nolint:gochecknoglobals // Cobra boilerplate
var generalFocus string

//nolint:gochecknoglobals // Cobra boilerplate
var generalCmd = &cobra.Command{
	Use:   "general",
	Short: "Generate a comprehensive general resume",
	Long: `Generate a comprehensive general resume that includes most relevant achievements
while keeping the output at or under 3 pages when rendered to PDF.

This creates a non-tailored resume suitable for general distribution or as a
master resume document.

Use --focus to create IC-focused or leadership-focused variants:
  --focus ic: Emphasizes hands-on technical work, architecture, implementation
  --focus leadership: Emphasizes team building, strategic initiatives, organizational impact
  --focus balanced: Balanced technical + leadership (default)

Example:
  resume-tailor general
  resume-tailor general --focus ic
  resume-tailor general --focus leadership --output-dir ~/Documents`,
	RunE: runGeneral,
}

//nolint:gochecknoinits // Cobra boilerplate
func init() {
	rootCmd.AddCommand(generalCmd)
	generalCmd.Flags().StringVar(&generalOutputDir, "output-dir", "", "Output directory (default from config)")
	generalCmd.Flags().BoolVar(&generalKeepMarkdown, "keep-markdown", true, "Keep markdown files after PDF generation")
	generalCmd.Flags().StringVar(&generalFocus, "focus", "balanced", "Resume focus: ic, leadership, or balanced (default)")
}

func runGeneral(cmd *cobra.Command, args []string) (err error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Load configuration
	var cfg config.Config
	cfg, err = config.Load(getConfigFile())
	if err != nil {
		err = errors.Wrap(err, "failed to load config")
		return err
	}

	// Validate focus parameter
	err = validateFocus(generalFocus)
	if err != nil {
		return err
	}

	// Use output dir from flag or config
	outDir := getOutputDir(generalOutputDir, cfg.Defaults.OutputDir)

	if getVerbose() {
		fmt.Printf("Loading summaries from: %s\n", cfg.SummariesLocation)
		fmt.Printf("Focus: %s\n", generalFocus)
	}

	// Load summaries
	var data summaries.Data
	data, err = summaries.Load(cfg.SummariesLocation)
	if err != nil {
		err = errors.Wrap(err, "failed to load summaries")
		return err
	}

	if getVerbose() {
		fmt.Printf("Loaded %d achievements\n", len(data.Achievements))
		fmt.Println("Generating comprehensive general resume...")
	}

	// Generate general resume
	var genResp llm.GeneralResumeResponse
	genResp, err = generateGeneralResume(ctx, cfg.AnthropicAPIKey, data, generalFocus)
	if err != nil {
		return err
	}

	// Generate output filenames
	var resumeMD, resumePDF string
	resumeMD, resumePDF = buildGeneralFilenames(data.Profile.Name, generalFocus, outDir)

	// Write and render
	err = writeAndRenderGeneral(genResp.Resume, resumeMD, resumePDF, cfg.Pandoc.TemplatePath, cfg.Pandoc.ClassFile)
	return err
}

func validateFocus(focus string) (err error) {
	validFocus := map[string]bool{"ic": true, "leadership": true, "balanced": true}
	if !validFocus[focus] {
		err = fmt.Errorf("invalid focus value '%s': must be 'ic', 'leadership', or 'balanced'", focus)
		return err
	}
	return err
}

func getOutputDir(flagValue, configValue string) (outDir string) {
	outDir = flagValue
	if outDir == "" {
		outDir = configValue
	}
	return outDir
}

func generateGeneralResume(ctx context.Context, apiKey string, data summaries.Data, focus string) (genResp llm.GeneralResumeResponse, err error) {
	// Convert achievements to maps for JSON
	achievementMaps := make([]map[string]interface{}, len(data.Achievements))
	for i, achievement := range data.Achievements {
		achievementMaps[i] = achievementToMap(achievement)
	}

	client := llm.NewClient(apiKey)
	genReq := llm.GeneralResumeRequest{
		Achievements: achievementMaps,
		Profile:      profileToMap(data.Profile),
		Skills:       skillsToMap(data.Skills),
		Projects:     projectsToMaps(data.OpensourceProjects),
		CompanyURLs:  data.CompanyURLs,
		Focus:        focus,
	}

	genResp, err = client.GenerateGeneral(ctx, genReq)
	if err != nil {
		err = errors.Wrap(err, "Claude API generation failed")
		return genResp, err
	}

	return genResp, err
}

func buildGeneralFilenames(name, focus, outDir string) (resumeMD, resumePDF string) {
	sanitizedName := sanitizeFilename(name)
	baseFilename := sanitizedName + "-general"
	// Add focus to filename if not balanced
	if focus != "balanced" {
		baseFilename += "-" + focus
	}
	baseFilename += "-resume"
	resumeMD = filepath.Join(outDir, baseFilename+".md")
	resumePDF = filepath.Join(outDir, baseFilename+".pdf")
	return resumeMD, resumePDF
}

func writeAndRenderGeneral(resume, resumeMD, resumePDF, templatePath, classPath string) (err error) {
	if getVerbose() {
		fmt.Println("Writing markdown file...")
	}

	// Write markdown file (unescape newlines that Claude may have escaped)
	resumeContent := unescapeNewlines(resume)
	err = renderer.WriteMarkdown(resumeContent, resumeMD)
	if err != nil {
		err = errors.Wrap(err, "failed to write resume markdown")
		return err
	}

	if getVerbose() {
		fmt.Println("Rendering PDF...")
	}

	err = renderAndCleanupGeneral(resumeMD, resumePDF, templatePath, classPath)
	return err
}

func renderAndCleanupGeneral(resumeMD, resumePDF, templatePath, classPath string) (err error) {
	// Render PDF
	err = renderer.RenderPDF(resumeMD, resumePDF, templatePath, classPath)
	if err != nil {
		fmt.Printf("Warning: Failed to render resume PDF: %v\n", err)
		fmt.Printf("Resume markdown saved at: %s\n", resumeMD)
	} else {
		fmt.Printf("General resume PDF saved at: %s\n", resumePDF)
	}

	// Clean up markdown files unless --keep-markdown is set
	if !generalKeepMarkdown {
		err = renderer.CleanupMarkdown(resumeMD)
		if err != nil {
			fmt.Printf("Warning: Failed to clean up markdown files: %v\n", err)
		}
	}

	fmt.Println("\nGeneration complete!")

	return err
}
