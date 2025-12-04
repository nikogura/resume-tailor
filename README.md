# resume-tailor

Generate tailored resumes and cover letters using Claude API to analyze job descriptions and select the most relevant achievements from your career history.

## Features

- **AI-Powered Analysis**: Uses Claude API to analyze job descriptions and rank your achievements by relevance
- **Two-Phase Generation**:
  - Phase 1: Analyze JD requirements and score achievements
  - Phase 2: Generate tailored resume and cover letter
- **PDF Rendering**: Automatic PDF generation using pandoc with custom LaTeX templates
- **Flexible Input**: Accept job descriptions as files or URLs
- **Standards Compliant**: Follows Terrace engineering standards with golangci-lint + namedreturns

## Prerequisites

- **Go 1.21+**: For building the tool
- **pandoc**: For PDF generation (`brew install pandoc` on macOS, `apt-get install pandoc` on Linux)
- **Claude API Key**: Get from https://console.anthropic.com/

## Installation

```bash
# Clone the repository
git clone https://github.com/nikogura/resume-tailor
cd resume-tailor

# Install dependencies
go mod download

# Build
make build

# Or install to $GOPATH/bin
make install
```

## Configuration

Create config file at `~/.resume-tailor/config.json`:

```json
{
  "anthropic_api_key": "sk-ant-api03-...",
  "summaries_location": "/home/you/Documents/nikogura.com/summaries/structured-summaries.json",
  "pandoc": {
    "template_path": "/home/you/Documents/nikogura.com/resume-template.latex",
    "class_file": "/home/you/Documents/nikogura.com/resume.cls"
  },
  "defaults": {
    "output_dir": "./applications"
  }
}
```

**Environment Variable Override**: Set `ANTHROPIC_API_KEY` to override config file.

## Summaries Data Structure

Your achievements must be in JSON format. Example:

```json
{
  "achievements": [
    {
      "id": "terrace-multicloud",
      "company": "Terrace",
      "role": "CIO & Head of Engineering",
      "dates": "2023-Present",
      "title": "Multi-Cloud Platform Architecture",
      "challenge": "Build enterprise-grade crypto trading platform spanning multiple clouds and on-premises.",
      "execution": "Architected multi-cloud, hybrid Kubernetes platform spanning AWS, GCP, and bare-metal...",
      "impact": "Infinite agility, no vendor lock-in, predictable costs, enterprise security",
      "metrics": ["100x data capacity", "600ms → microseconds latency"],
      "keywords": ["Multi-cloud", "Kubernetes", "GitOps", "AWS", "GCP"],
      "categories": ["Platform Engineering", "Cloud Architecture"]
    }
  ],
  "profile": {
    "name": "Your Name",
    "title": "Principal Engineer",
    "location": "City, State",
    "motto": "Your motto",
    "profiles": {
      "github": "https://github.com/username",
      "linkedin": "https://linkedin.com/in/username"
    }
  },
  "skills": {
    "languages": ["Go", "Python", "Java"],
    "cloud": ["AWS", "GCP", "Azure"],
    "kubernetes": ["EKS", "AKS", "Kubeadm"]
  },
  "opensource_projects": [
    {
      "name": "Project Name",
      "url": "https://github.com/username/project",
      "description": "Description",
      "recognition": "Listed in awesome-go"
    }
  ]
}
```

## Usage

### Generate Resume and Cover Letter

```bash
resume-tailor generate jd.txt \
  --company "Acme Corp" \
  --role "Staff DevOps Engineer" \
  --output-dir ./applications/acme

# Or from URL
resume-tailor generate https://company.com/jobs/123 \
  --company "Acme Corp" \
  --role "Staff Engineer"

# With additional context for the cover letter
resume-tailor generate jd.txt \
  --company "Acme Corp" \
  --role "Staff Engineer" \
  --context "Referred by Jane Smith, Engineering Manager. Excited about the company's recent Series B funding and growth plans."
```

**Generates:**
- `acme-corp-staff-devops-engineer-resume.md`
- `acme-corp-staff-devops-engineer-resume.pdf`
- `acme-corp-staff-devops-engineer-cover-letter.md`
- `acme-corp-staff-devops-engineer-cover-letter.pdf`

### Options

- `--company`: Company name (extracted from JD if not provided, prompts if extraction fails)
- `--role`: Role title (extracted from JD if not provided, prompts if extraction fails)
- `--context`: Additional context for cover letter generation (optional)
- `--output-dir`: Output directory (default from config)
- `--keep-markdown`: Keep markdown files after PDF generation
- `--config`: Config file path (default: `~/.resume-tailor/config.json`)
- `-v, --verbose`: Verbose output

## Development

### Build and Test

```bash
# Build
make build

# Run tests
make test

# Run linter (includes namedreturns)
make lint

# Clean build artifacts
make clean
```

### Code Standards

This project follows strict Terrace engineering standards:

- **Named returns mandatory**: All functions with multiple returns use named return values
- **No inline error handling**: Separate error declaration from checking
- **golangci-lint with namedreturns**: Custom linter enforces self-documenting code
- **Clear over clever**: Straightforward implementations, no over-engineering

## How It Works

1. **Load Configuration**: Reads config with API key and summaries location
2. **Fetch Job Description**: From file or URL, with basic HTML stripping
3. **Phase 1 - Analyze**:
   - Sends JD + all achievements to Claude
   - Claude scores each achievement 0.0-1.0 on relevance
   - Returns ranked list with reasoning
4. **Phase 2 - Generate**:
   - Sends top-ranked achievements (score ≥ 0.6) to Claude
   - Includes optional context (referral info, company research, etc.) if provided
   - Claude generates tailored resume and cover letter
   - Matches JD language naturally and incorporates context into cover letter
5. **Render**: Writes markdown and converts to PDF via pandoc

## Cost Estimate

- ~$0.10-0.50 per resume generation (Claude API usage)
- Depends on number of achievements and JD length

## Troubleshooting

**"pandoc not found"**: Install pandoc (`brew install pandoc` or `apt-get install pandoc`)

**"config file not found"**: Create `~/.resume-tailor/config.json` with your API key

**"summaries file not found"**: Ensure `summaries_location` in config points to valid JSON file

**Lint errors**: Run `make lint` to see specific issues. Focus on named returns and error handling patterns.

## License

MIT

## Author

Nik Ogura - https://nikogura.com
