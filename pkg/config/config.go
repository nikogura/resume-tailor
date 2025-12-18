package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Config represents the application configuration.
type Config struct {
	Name              string        `json:"name"`
	AnthropicAPIKey   string        `json:"anthropic_api_key"`
	SummariesLocation string        `json:"summaries_location"`
	CompleteResumeURL string        `json:"complete_resume_url,omitempty"`
	LinkedInURL       string        `json:"linkedin_url,omitempty"`
	Models            ModelsConfig  `json:"models,omitempty"`
	Pandoc            PandocConfig  `json:"pandoc"`
	Defaults          DefaultConfig `json:"defaults"`
}

// ModelsConfig holds model selection for generation and evaluation.
type ModelsConfig struct {
	Generation string `json:"generation,omitempty"`
	Evaluation string `json:"evaluation,omitempty"`
}

// PandocConfig holds pandoc-related configuration.
type PandocConfig struct {
	TemplatePath string `json:"template_path"`
	ClassFile    string `json:"class_file"`
}

// DefaultConfig holds default values for commands.
type DefaultConfig struct {
	OutputDir string `json:"output_dir"`
}

// GetGenerationModel returns the generation model or default if not specified.
func (c *Config) GetGenerationModel() (model string) {
	if c.Models.Generation != "" {
		model = c.Models.Generation
		return model
	}
	model = "claude-sonnet-4-20250514" // Default to Sonnet 4
	return model
}

// GetEvaluationModel returns the evaluation model or default if not specified.
func (c *Config) GetEvaluationModel() (model string) {
	if c.Models.Evaluation != "" {
		model = c.Models.Evaluation
		return model
	}
	model = "claude-sonnet-4-5-20250929" // Default to Sonnet 4.5
	return model
}

// Load reads configuration from file with environment variable overrides.
func Load(configPath string) (cfg Config, err error) {
	// Determine config file location
	path := configPath
	if path == "" {
		var homeDir string
		homeDir, err = os.UserHomeDir()
		if err != nil {
			err = errors.Wrap(err, "failed to get user home directory")
			return cfg, err
		}
		path = filepath.Join(homeDir, ".resume-tailor", "config.json")
	}

	// Read config file
	var data []byte
	data, err = os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Errorf("config file not found: %s (run 'resume-tailor init' to create)", path)
			return cfg, err
		}
		err = errors.Wrapf(err, "failed to read config file: %s", path)
		return cfg, err
	}

	// Parse JSON
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse config file: %s", path)
		return cfg, err
	}

	// Override with environment variable if set
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.AnthropicAPIKey = apiKey
	}

	// Validate required fields
	err = cfg.Validate()
	if err != nil {
		err = errors.Wrap(err, "config validation failed")
		return cfg, err
	}

	return cfg, err
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() (err error) {
	if c.Name == "" {
		err = errors.New("name is required in config")
		return err
	}

	if c.AnthropicAPIKey == "" {
		err = errors.New("anthropic_api_key is required (set in config or ANTHROPIC_API_KEY env var)")
		return err
	}

	if c.SummariesLocation == "" {
		err = errors.New("summaries_location is required in config")
		return err
	}

	// Check summaries file exists
	_, err = os.Stat(c.SummariesLocation)
	if os.IsNotExist(err) {
		err = errors.Errorf("summaries file not found: %s", c.SummariesLocation)
		return err
	}

	if c.Pandoc.TemplatePath == "" {
		err = errors.New("pandoc.template_path is required in config")
		return err
	}

	if c.Pandoc.ClassFile == "" {
		err = errors.New("pandoc.class_file is required in config")
		return err
	}

	// Set default output_dir if not specified
	if c.Defaults.OutputDir == "" {
		c.Defaults.OutputDir = "./applications"
	}

	return err
}

// InitConfig creates a default configuration file.
func InitConfig(configPath string) (err error) {
	// Determine config file location
	path := configPath
	if path == "" {
		var homeDir string
		homeDir, err = os.UserHomeDir()
		if err != nil {
			err = errors.Wrap(err, "failed to get user home directory")
			return err
		}
		path = filepath.Join(homeDir, ".resume-tailor", "config.json")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0750)
	if err != nil {
		err = errors.Wrapf(err, "failed to create config directory: %s", dir)
		return err
	}

	// Check if file already exists
	_, err = os.Stat(path)
	if err == nil {
		err = errors.Errorf("config file already exists: %s", path)
		return err
	}

	// Create default config
	var homeDir string
	homeDir, err = os.UserHomeDir()
	if err != nil {
		err = errors.Wrap(err, "failed to get user home directory")
		return err
	}

	defaultConfig := Config{
		Name:              "your-name",
		AnthropicAPIKey:   "sk-ant-api03-...",
		SummariesLocation: filepath.Join(homeDir, ".resume-tailor", "structured-summaries.json"),
		CompleteResumeURL: "",
		LinkedInURL:       "",
		Pandoc: PandocConfig{
			TemplatePath: filepath.Join(homeDir, ".resume-tailor", "resume-template.latex"),
			ClassFile:    filepath.Join(homeDir, ".resume-tailor", "resume.cls"),
		},
		Defaults: DefaultConfig{
			OutputDir: filepath.Join(homeDir, "Documents", "Applications"),
		},
	}

	// Write to file
	var data []byte
	data, err = json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		err = errors.Wrap(err, "failed to marshal default config")
		return err
	}

	err = os.WriteFile(path, data, 0600)
	if err != nil {
		err = errors.Wrapf(err, "failed to write config file: %s", path)
		return err
	}

	return err
}
