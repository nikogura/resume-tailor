package summaries

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a test summaries file.
	tmpDir := t.TempDir()
	summariesPath := filepath.Join(tmpDir, "summaries.json")

	testData := Data{
		Achievements: []Achievement{
			{
				ID:         "test-1",
				Company:    "Test Corp",
				Role:       "Test Engineer",
				Dates:      "2020-2021",
				Title:      "Test Achievement",
				Challenge:  "Test challenge",
				Execution:  "Test execution",
				Impact:     "Test impact",
				Metrics:    []string{"100% success"},
				Keywords:   []string{"test", "golang"},
				Categories: []string{"Testing"},
			},
		},
		Profile: Profile{
			Name:     "Test User",
			Title:    "Test Engineer",
			Location: "Test City",
			Motto:    "Test motto",
			Profiles: map[string]string{
				"github": "https://github.com/test",
			},
		},
		Skills: Skills{
			Languages:  []string{"Go", "Python"},
			Cloud:      []string{"AWS", "GCP"},
			Kubernetes: []string{"EKS"},
		},
		OpensourceProjects: []OpensourceProject{
			{
				Name:        "Test Project",
				URL:         "https://github.com/test/project",
				Description: "Test description",
			},
		},
	}

	data, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = os.WriteFile(summariesPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test loading.
	loaded, err := Load(summariesPath)
	if err != nil {
		t.Fatalf("Failed to load summaries: %v", err)
	}

	if len(loaded.Achievements) != 1 {
		t.Errorf("Expected 1 achievement, got %d", len(loaded.Achievements))
	}

	if loaded.Achievements[0].ID != "test-1" {
		t.Errorf("Expected achievement ID 'test-1', got '%s'", loaded.Achievements[0].ID)
	}

	if loaded.Profile.Name != "Test User" {
		t.Errorf("Expected profile name 'Test User', got '%s'", loaded.Profile.Name)
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/summaries.json")
	if err == nil {
		t.Error("Expected error loading nonexistent file, got nil")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	summariesPath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(summariesPath, []byte("not valid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = Load(summariesPath)
	if err == nil {
		t.Error("Expected error loading invalid JSON, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		data      Data
		wantError bool
	}{
		{
			name: "valid data",
			data: Data{
				Achievements: []Achievement{
					{
						ID:      "test-1",
						Company: "Test Corp",
						Role:    "Engineer",
						Title:   "Test Achievement",
					},
				},
				Profile: Profile{
					Name: "Test User",
				},
			},
			wantError: false,
		},
		{
			name:      "empty achievements",
			data:      Data{},
			wantError: true,
		},
		{
			name: "achievement missing ID",
			data: Data{
				Achievements: []Achievement{
					{Company: "Test Corp", Role: "Engineer", Title: "Test"},
				},
				Profile: Profile{
					Name: "Test User",
				},
			},
			wantError: true,
		},
		{
			name: "achievement missing company",
			data: Data{
				Achievements: []Achievement{
					{ID: "test-1", Role: "Engineer", Title: "Test"},
				},
				Profile: Profile{
					Name: "Test User",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestFilterByScore(t *testing.T) {
	ranked := []RankedAchievement{
		{AchievementID: "high", RelevanceScore: 0.9},
		{AchievementID: "medium", RelevanceScore: 0.6},
		{AchievementID: "low", RelevanceScore: 0.3},
	}

	filtered := FilterByScore(ranked, 0.5)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 achievements, got %d", len(filtered))
	}

	// Verify they're the right ones.
	foundHigh := false
	foundMedium := false
	for _, a := range filtered {
		if a.AchievementID == "high" {
			foundHigh = true
		}
		if a.AchievementID == "medium" {
			foundMedium = true
		}
	}

	if !foundHigh {
		t.Error("Expected to find 'high' achievement")
	}
	if !foundMedium {
		t.Error("Expected to find 'medium' achievement")
	}
}
