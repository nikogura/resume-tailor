package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra boilerplate
var verbose bool

//nolint:gochecknoglobals // Cobra boilerplate
var configFile string

//nolint:gochecknoglobals // Cobra boilerplate
var rootCmd = &cobra.Command{
	Use:   "resume-tailor",
	Short: "Generate tailored resumes and cover letters",
	Long: `resume-tailor analyzes job descriptions and generates tailored resumes
and cover letters by selecting the most relevant achievements from your career history.

Uses Claude API to analyze requirements and craft compelling applications.`,
}

// Execute runs the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

//nolint:gochecknoinits // Cobra boilerplate
func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.resume-tailor/config.json)")
}

// getVerbose returns the verbose flag value.
func getVerbose() (result bool) {
	result = verbose
	return result
}

// getConfigFile returns the config file path.
func getConfigFile() (result string) {
	result = configFile
	return result
}
