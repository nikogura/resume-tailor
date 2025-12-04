package summaries

// Data represents the complete summaries data structure.
type Data struct {
	Achievements       []Achievement       `json:"achievements"`
	Profile            Profile             `json:"profile"`
	Skills             Skills              `json:"skills"`
	OpensourceProjects []OpensourceProject `json:"opensource_projects"`
}

// Achievement represents a single career achievement.
type Achievement struct {
	ID         string   `json:"id"`
	Company    string   `json:"company"`
	Role       string   `json:"role"`
	Dates      string   `json:"dates"`
	Title      string   `json:"title"`
	Challenge  string   `json:"challenge"`
	Execution  string   `json:"execution"`
	Impact     string   `json:"impact"`
	Metrics    []string `json:"metrics"`
	Keywords   []string `json:"keywords"`
	Categories []string `json:"categories"`
}

// Profile represents personal information.
type Profile struct {
	Name     string            `json:"name"`
	Title    string            `json:"title"`
	Location string            `json:"location"`
	Motto    string            `json:"motto"`
	Profiles map[string]string `json:"profiles"`
}

// Skills represents organized skill categories.
type Skills struct {
	Languages  []string `json:"languages"`
	Cloud      []string `json:"cloud"`
	Kubernetes []string `json:"kubernetes"`
	Security   []string `json:"security"`
	Databases  []string `json:"databases"`
	CICD       []string `json:"cicd"`
	Networks   []string `json:"networks"`
}

// OpensourceProject represents an open source contribution.
type OpensourceProject struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Recognition string `json:"recognition"`
}
