# resume-tailor

Generate tailored resumes and cover letters using Claude API to analyze job descriptions and select the most relevant achievements from your career history.

I've been working in tech professionaly for 25+ years.  It's really difficult to get all relevant experiences into a resume that is short enough that people will actually read it.

Hiring managers want to see your qualifications for their exact jobs, everything else is noise.

With the structured summaries of your job experiences and achievements, an agent can quickly match what you've done with what they're looking for and put your best foot forward.

## Features

- **AI-Powered Analysis**: Uses Claude API to analyze job descriptions and rank your achievements by relevance
- **Two-Phase Generation**:
  - Phase 1: Analyze JD requirements and score achievements
  - Phase 2: Generate tailored resume and cover letter
- **Self-Improving RAG System**: Evaluates generated resumes for hallucinations and learns from mistakes
  - Separate Claude instance evaluates outputs for fabrications
  - Scores resumes on accuracy and anti-fabrication rules
  - Indexes lessons learned and injects them into future generations
- **Anti-Hallucination Engine**: Strict rules prevent fabricated numbers, industries, and domains
- **PDF Rendering**: Automatic PDF generation using pandoc with custom LaTeX templates
- **Flexible Input**: Accept job descriptions as files or URLs
- **Standards Compliant**: Follows [Nik Ogura's engineering standards](https://nikogura.com/EngineeringStandards.html) with golangci-lint + namedreturns

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
  "name": "your-name",
  "anthropic_api_key": "sk-ant-api03-...",
  "summaries_location": "~/.resume-tailor/structured-summaries.json",
  "complete_resume_url": "https://github.com/username/repo/blob/master/general-resume.pdf",
  "models": {
    "generation": "claude-sonnet-4-20250514",
    "evaluation": "claude-sonnet-4-5-20250929"
  },
  "pandoc": {
    "template_path": "~/.resume-tailor/resume-template.latex",
    "class_file": "~/.resume-tailor/resume.cls"
  },
  "defaults": {
    "output_dir": "~/Documents/Applications"
  }
}
```

**Configuration Fields:**
- `name`: Used in output filenames (e.g., `your-name-acme-corp-staff-engineer-resume.pdf`)
- `anthropic_api_key`: Your Claude API key (can be overridden with `ANTHROPIC_API_KEY` env var)
- `summaries_location`: Path to your structured achievements JSON file
- `complete_resume_url`: (Optional) URL to your complete general resume - will be linked in cover letters
- `models.generation`: (Optional) Claude model for resume generation (default: `claude-sonnet-4-20250514`)
- `models.evaluation`: (Optional) Claude model for evaluation (default: `claude-sonnet-4-5-20250929`)
- `pandoc.template_path`: Path to LaTeX template for PDF generation
- `pandoc.class_file`: Path to LaTeX class file
- `defaults.output_dir`: Default output directory for generated resumes

**Model Selection:**

The `models` configuration allows you to customize which Claude models are used for different purposes:

- **Generation Model**: Used for analyzing job descriptions and generating tailored resumes/cover letters. Default is Claude Sonnet 4 (`claude-sonnet-4-20250514`) which balances quality and speed.
- **Evaluation Model**: Used for evaluating generated content against anti-fabrication rules. Default is Claude Sonnet 4.5 (`claude-sonnet-4-5-20250929`) for more thorough fact-checking.

Using separate models ensures the evaluator acts as an independent auditor and won't defend the generator's work. You can customize these based on your needs:
- Use faster models (like Haiku) for quicker iterations during testing
- Use more capable models (like Opus) for higher-quality output in production
- Keep both models the same if you prefer consistency over separation

Available models include:
- `claude-sonnet-4-20250514` - Balanced performance (recommended for generation)
- `claude-sonnet-4-5-20250929` - Enhanced capabilities (recommended for evaluation)
- `claude-opus-4-5-20251101` - Highest quality (slower, more expensive)
- `claude-haiku-3-7-20250122` - Fastest, most economical

If the `models` section is omitted, the system uses the defaults above.

### LaTeX Templates

The project includes default LaTeX templates in the `templates/` directory:
- `templates/resume-template.latex` - Pandoc template for resume formatting
- `templates/resume.cls` - LaTeX class file with custom styling

To use these templates, copy them to your config directory:

```bash
cp templates/resume-template.latex ~/.resume-tailor/
cp templates/resume.cls ~/.resume-tailor/
```

Or customize them and use your own paths in the config file.

## Summaries Data Structure

Your achievements must be in JSON format. Example:

```json
{
  "achievements": [
    {
      "id": "acme-multicloud",
      "company": "Acme Corp",
      "role": "Principal Engineer",
      "dates": "2023-Present",
      "title": "Multi-Cloud Platform Architecture",
      "challenge": "Build enterprise-grade platform spanning multiple clouds and on-premises.",
      "execution": "Architected multi-cloud, hybrid Kubernetes platform spanning AWS, GCP, and bare-metal...",
      "impact": "Infinite agility, no vendor lock-in, predictable costs, enterprise security",
      "metrics": ["100x data capacity", "600ms → microseconds latency"],
      "keywords": ["Multi-cloud", "Kubernetes", "GitOps", "AWS", "GCP"],
      "categories": ["Platform Engineering", "Cloud Architecture"]
    }
  ],
  "profile": {
    "name": "Your Name",
    "role_titles": ["Principal Engineer", "CTO"],
    "years_experience": 15,
    "location": "City, State",
    "motto": "Your motto, if you have one",
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
      "name": "Dynamic Binary Toolkit (DBT)",
      "url": "https://github.com/nikogura/dbt",
      "description": "Self-updating signed binary distribution system. Builds, tests, publishes, and updates itself in flight. Used for distributing tools in cloud, containers, and laptops.",
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

# Multiple applications to same company/role - use job-id to differentiate
resume-tailor generate jd-platform.txt \
  --company "Acme Corp" \
  --role "Principal Engineer" \
  --job-id "req-8886"

resume-tailor generate jd-infrastructure.txt \
  --company "Acme Corp" \
  --role "Principal Engineer" \
  --job-id "req-8492"
```

**Generates:**
- `your-name-acme-corp-staff-devops-engineer-resume.md`
- `your-name-acme-corp-staff-devops-engineer-resume.pdf`
- `your-name-acme-corp-staff-devops-engineer-cover.md`
- `your-name-acme-corp-staff-devops-engineer-cover.pdf`

**Using `--job-id` for Multiple Applications:**

When applying to multiple positions at the same company with similar role titles, use the `--job-id` flag to prevent filename collisions. The job ID is appended to the filename:

- Without `--job-id`: `your-name-acme-corp-principal-engineer-resume.pdf`
- With `--job-id "req-8886"`: `your-name-acme-corp-principal-engineer-req-8886-resume.pdf`

Common job ID formats:
- Requisition numbers: `req-12345`, `8886`, `JR-2024-001`
- Team/focus area: `platform`, `infrastructure`, `api`
- Short descriptors: `backend`, `fullstack`, `ml`

### Evaluate Generated Resumes

After generating resumes, evaluate them for hallucinations and quality:

```bash
# Evaluate a specific application
resume-tailor evaluate ~/Documents/Applications/acme-corp

# Evaluate all applications in the output directory
resume-tailor evaluate --all

# Verbose evaluation output
resume-tailor evaluate ~/Documents/Applications/acme-corp -v
```

The evaluation system:
- Uses a separate Claude instance to objectively score the resume
- Checks for fabricated numbers, industries, and domains
- Verifies company names, role titles, and dates
- Scores resumes 0-100 based on accuracy and anti-fabrication rules
- Stores results in `.evaluation.json` alongside generated files
- Builds a RAG index of lessons learned from all evaluations
- Future generations automatically learn from past mistakes

**Evaluation Output:**
- `.evaluation.json`: Full evaluation with violations, scores, and lessons learned
- `.rag-index.json`: Searchable index of all evaluations (in output directory root)

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

This project follows strict [Nik Ogura's engineering standards](https://nikogura.com/EngineeringStandards.html):

- **Named returns mandatory**: All functions with multiple returns use named return values
- **No inline error handling**: Separate error declaration from checking
- **golangci-lint with namedreturns**: Custom linter enforces self-documenting code
- **Clear over clever**: Straightforward implementations, no over-engineering

## How It Works

### Generation Flow

1. **Load Configuration**: Reads config with API key and summaries location
2. **Fetch Job Description**: From file or URL, with basic HTML stripping
3. **Retrieve RAG Context**: Queries past evaluations for similar roles and lessons learned
4. **Phase 1 - Analyze**:
   - Sends JD + all achievements to Claude
   - Claude scores each achievement 0.0-1.0 on relevance
   - Returns ranked list with reasoning
5. **Phase 2 - Generate**:
   - Injects RAG lessons learned at top of prompt
   - Sends top-ranked achievements (score ≥ 0.6) to Claude
   - Includes optional context (referral info, company research, etc.) if provided
   - Claude generates tailored resume and cover letter with anti-hallucination rules
   - Matches JD language naturally and incorporates context into cover letter
6. **Render**: Writes markdown and converts to PDF via pandoc

### Evaluation Flow (Self-Improvement)

1. **Load Generated Files**: Reads resume, cover letter, and job description
2. **Load Source Data**: Reads original achievements, profile, and skills
3. **Evaluate with Separate Claude Instance**:
   - Checks every number against source data (detects fabrications)
   - Verifies company names, role titles, and employment dates
   - Checks for industry/domain fabrications
   - Detects pattern matching violations (claiming work "mirrors" domains you lack)
   - Scores weak quantifications (numbers that undermine credibility)
4. **Calculate Scores**:
   - Anti-fabrication rules: 50% weight (critical violations)
   - Accuracy rules: 30% weight (dates, titles, metrics)
   - Quality rules: 20% weight (weak numbers, structure)
5. **Generate Lessons Learned**: Extracts patterns from violations
6. **Build RAG Index**: Indexes evaluation with lessons for future retrieval
7. **Store Results**: Writes `.evaluation.json` with full scoring details

## Cost Estimate

Based on Claude API pricing (~$3/M input tokens, ~$15/M output tokens):

- **Per Resume Generation**: $0.40-1.00 typical, up to $1.50 with re-evaluation
  - JD analysis: 15-40K input, 5-15K output
  - Resume generation: 20-50K input, 3-8K output
  - Evaluation: 30-60K input, 5-15K output
  - RAG indexing: 5-15K input, minimal output
  - Re-evaluation (if fixes needed): +30-60K input, +5-15K output

- **Typical Flow**: $0.60-1.20 per complete application (includes evaluation and fixes)
- **Factors Affecting Cost**:
  - Number of achievements in your library (more = higher analysis cost)
  - JD complexity and length
  - Number of violations requiring fixes and re-evaluation
  - Model selection (Sonnet 4 vs Sonnet 4.5 vs Opus vs Haiku)

**Cost Optimization Tips:**
- Use Haiku for testing/iteration ($0.25/M input, $1.25/M output - ~80% cheaper)
- Keep achievement library focused on recent, high-quality items
- Use cached model responses where possible
- Configure generation model separately from evaluation model for cost control

## Troubleshooting

**"pandoc not found"**: Install pandoc (`brew install pandoc` or `apt-get install pandoc`)

**"config file not found"**: Create `~/.resume-tailor/config.json` with your API key

**"summaries file not found"**: Ensure `summaries_location` in config points to valid JSON file

**Lint errors**: Run `make lint` to see specific issues. Focus on named returns and error handling patterns.

## License

MIT

## Author

Nik Ogura - https://nikogura.com
