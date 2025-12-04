package renderer

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// RenderPDF converts markdown to PDF using pandoc with LaTeX templates.
func RenderPDF(markdownPath, outputPath, templatePath, classPath string) (err error) {
	// Validate pandoc exists
	err = checkPandocExists()
	if err != nil {
		return err
	}

	// Validate input files exist
	err = validateFiles(markdownPath, templatePath, classPath)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	err = os.MkdirAll(outputDir, 0750)
	if err != nil {
		err = errors.Wrapf(err, "failed to create output directory: %s", outputDir)
		return err
	}

	// Build pandoc command
	//nolint:noctx // Context not available for exec.Command - pandoc is a long-running subprocess
	cmd := exec.Command(
		"pandoc",
		"-f", "markdown",
		"-t", "pdf",
		"-o", outputPath,
		"--template", templatePath,
		"--number-sections=false",
		markdownPath,
	)

	// Set TEXINPUTS to include directory with .cls file
	classDir := filepath.Dir(classPath)
	texinputs := classDir + ":" + os.Getenv("TEXINPUTS")
	cmd.Env = append(os.Environ(), "TEXINPUTS="+texinputs)

	// Capture output
	var output []byte
	output, err = cmd.CombinedOutput()
	if err != nil {
		err = errors.Wrapf(err, "pandoc failed: %s", string(output))
		return err
	}

	return err
}

// checkPandocExists verifies pandoc is installed.
func checkPandocExists() (err error) {
	//nolint:noctx // Context not available for version check
	cmd := exec.Command("pandoc", "--version")
	err = cmd.Run()
	if err != nil {
		err = errors.New("pandoc not found in PATH (install pandoc to generate PDFs)")
		return err
	}
	return err
}

// validateFiles checks that required files exist.
func validateFiles(paths ...string) (err error) {
	for _, path := range paths {
		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			err = errors.Errorf("file not found: %s", path)
			return err
		}
	}
	return err
}

// WriteMarkdown writes markdown content to a file.
func WriteMarkdown(content, outputPath string) (err error) {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	err = os.MkdirAll(outputDir, 0750)
	if err != nil {
		err = errors.Wrapf(err, "failed to create output directory: %s", outputDir)
		return err
	}

	// Write file
	err = os.WriteFile(outputPath, []byte(content), 0600)
	if err != nil {
		err = errors.Wrapf(err, "failed to write markdown file: %s", outputPath)
		return err
	}

	return err
}

// CleanupMarkdown removes markdown files after PDF generation.
func CleanupMarkdown(paths ...string) (err error) {
	for _, path := range paths {
		err = os.Remove(path)
		if err != nil {
			err = errors.Wrapf(err, "failed to remove markdown file: %s", path)
			return err
		}
	}
	return err
}
