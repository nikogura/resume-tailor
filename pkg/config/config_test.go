package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := Config{
		Name:              "test-user",
		AnthropicAPIKey:   "test-key",
		SummariesLocation: tmpDir, // Use temp dir as it exists
		Pandoc: PandocConfig{
			TemplatePath: "test-template.latex",
			ClassFile:    "test-class.cls",
		},
		Defaults: DefaultConfig{
			OutputDir: "./test-output",
		},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading the config.
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.AnthropicAPIKey != testConfig.AnthropicAPIKey {
		t.Errorf("Expected API key %s, got %s", testConfig.AnthropicAPIKey, cfg.AnthropicAPIKey)
	}

	if cfg.SummariesLocation != testConfig.SummariesLocation {
		t.Errorf("Expected summaries location %s, got %s", testConfig.SummariesLocation, cfg.SummariesLocation)
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error loading nonexistent config, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid config",
			config: Config{
				Name:              "test-user",
				AnthropicAPIKey:   "test-key",
				SummariesLocation: os.TempDir(), //nolint:usetesting // Using os.TempDir() as known existing dir path for validation test, not for file I/O
				Pandoc: PandocConfig{
					TemplatePath: "template.latex",
					ClassFile:    "class.cls",
				},
				Defaults: DefaultConfig{
					OutputDir: "./output",
				},
			},
			wantError: false,
		},
		{
			name: "missing API key",
			config: Config{
				SummariesLocation: os.TempDir(), //nolint:usetesting // Using os.TempDir() as known existing dir path for validation test, not for file I/O
				Pandoc: PandocConfig{
					TemplatePath: "template.latex",
					ClassFile:    "class.cls",
				},
			},
			wantError: true,
		},
		{
			name: "missing summaries location",
			config: Config{
				AnthropicAPIKey: "test-key",
				Pandoc: PandocConfig{
					TemplatePath: "template.latex",
					ClassFile:    "class.cls",
				},
			},
			wantError: true,
		},
		{
			name: "nonexistent summaries file",
			config: Config{
				AnthropicAPIKey:   "test-key",
				SummariesLocation: "/nonexistent/file.json",
				Pandoc: PandocConfig{
					TemplatePath: "template.latex",
					ClassFile:    "class.cls",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	// Verify file was created.
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Read and verify the config structure without full validation.
	// Full validation would require all paths to exist, which isn't needed for this test.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.Defaults.OutputDir == "" {
		t.Error("Default output dir was not set")
	}

	if cfg.Name == "" {
		t.Error("Default name was not set")
	}
}

func TestInitConfigAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create file first.
	err := os.WriteFile(configPath, []byte("{}"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to init - should fail.
	err = InitConfig(configPath)
	if err == nil {
		t.Error("Expected error when config already exists, got nil")
	}
}
