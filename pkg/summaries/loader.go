package summaries

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// Load reads the summaries data from a JSON file.
func Load(path string) (data Data, err error) {
	// Read file
	var fileData []byte
	fileData, err = os.ReadFile(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to read summaries file: %s", path)
		return data, err
	}

	// Parse JSON
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		err = errors.Wrapf(err, "failed to parse summaries JSON: %s", path)
		return data, err
	}

	// Validate data
	err = data.Validate()
	if err != nil {
		err = errors.Wrap(err, "summaries validation failed")
		return data, err
	}

	return data, err
}

// Validate checks that the summaries data is well-formed.
func (d *Data) Validate() (err error) {
	if len(d.Achievements) == 0 {
		err = errors.New("no achievements found in summaries")
		return err
	}

	if d.Profile.Name == "" {
		err = errors.New("profile name is required")
		return err
	}

	// Validate each achievement has required fields
	for i, achievement := range d.Achievements {
		if achievement.ID == "" {
			err = errors.Errorf("achievement at index %d missing ID", i)
			return err
		}
		if achievement.Company == "" {
			err = errors.Errorf("achievement %s missing company", achievement.ID)
			return err
		}
		if achievement.Title == "" {
			err = errors.Errorf("achievement %s missing title", achievement.ID)
			return err
		}
	}

	return err
}

// FilterByScore returns achievements with relevance score above threshold.
func FilterByScore(achievements []RankedAchievement, threshold float64) (filtered []RankedAchievement) {
	filtered = make([]RankedAchievement, 0)
	for _, achievement := range achievements {
		if achievement.RelevanceScore >= threshold {
			filtered = append(filtered, achievement)
		}
	}
	return filtered
}

// RankedAchievement represents an achievement with score (defined here to avoid circular import).
type RankedAchievement struct {
	AchievementID  string
	RelevanceScore float64
	Reasoning      string
}
