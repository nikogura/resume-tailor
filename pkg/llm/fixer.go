package llm

import (
	"fmt"
	"regexp"
	"strings"
)

// Fixer applies automated fixes to resumes and cover letters based on evaluation violations.
type Fixer struct {
	// Fix patterns organized by rule type
	temporalImpossibilityPatterns []FixPattern
	domainExpertPatterns          []FixPattern
	coverLetterPatterns           []FixPattern
}

// FixPattern defines a search-and-fix pattern.
type FixPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
	RuleMatch   string // Which violation rule this fixes
}

// NewFixer creates a new fixer with predefined fix patterns.
func NewFixer() (fixer *Fixer) {
	fixer = &Fixer{
		temporalImpossibilityPatterns: buildTemporalImpossibilityPatterns(),
		domainExpertPatterns:          buildDomainExpertPatterns(),
		coverLetterPatterns:           buildCoverLetterPatterns(),
	}
	return fixer
}

// ApplyFixes applies automated fixes to resume and cover letter based on violations.
func (f *Fixer) ApplyFixes(resumeMD, coverLetterMD string, evalResp EvaluationResponse) (fixedResume, fixedCoverLetter string, appliedFixes []string, err error) {
	fixedResume = resumeMD
	fixedCoverLetter = coverLetterMD
	appliedFixes = []string{}

	// Fix resume violations
	fixedResume, appliedFixes = f.fixResumeViolations(fixedResume, evalResp, appliedFixes)

	// Fix cover letter violations
	fixedCoverLetter = f.fixCoverLetterViolations(fixedCoverLetter, evalResp)

	return fixedResume, fixedCoverLetter, appliedFixes, err
}

// fixResumeViolations applies all resume fixes.
func (f *Fixer) fixResumeViolations(resume string, evalResp EvaluationResponse, appliedFixes []string) (fixed string, fixes []string) {
	fixed = resume
	fixes = appliedFixes

	// Fix temporal impossibility violations
	for _, violation := range evalResp.ResumeViolations {
		if strings.Contains(violation.Rule, "TEMPORAL") {
			var applied bool
			fixed, applied = f.applyTemporalFixes(fixed)
			if applied {
				fixes = append(fixes, fmt.Sprintf("Fixed temporal impossibility: %s", violation.Fabricated))
			}
		}
	}

	// Fix domain expert claims
	for _, violation := range evalResp.ResumeViolations {
		if strings.Contains(violation.Rule, "DOMAIN") || strings.Contains(violation.Fabricated, "Expert") {
			var applied bool
			fixed, applied = f.applyDomainExpertFixes(fixed)
			if applied {
				fixes = append(fixes, fmt.Sprintf("Fixed domain expert claim: %s", violation.Fabricated))
			}
		}
	}

	// Fix weak quantifications
	fixed = f.applyCoverLetterWording(fixed)

	return fixed, fixes
}

// fixCoverLetterViolations applies all cover letter fixes.
func (f *Fixer) fixCoverLetterViolations(coverLetter string, evalResp EvaluationResponse) (fixed string) {
	fixed = coverLetter

	// Fix domain expert claims
	for _, violation := range evalResp.CoverLetterViolations {
		if strings.Contains(violation.Rule, "DOMAIN") || strings.Contains(violation.Fabricated, "Expert") {
			fixed, _ = f.applyDomainExpertFixes(fixed)
		}
	}

	// Fix weak quantifications and wording patterns
	fixed = f.applyCoverLetterWording(fixed)

	return fixed
}

// applyTemporalFixes fixes temporal impossibility violations.
func (f *Fixer) applyTemporalFixes(content string) (fixed string, applied bool) {
	fixed = content
	applied = false

	for _, pattern := range f.temporalImpossibilityPatterns {
		if pattern.Pattern.MatchString(fixed) {
			fixed = pattern.Pattern.ReplaceAllString(fixed, pattern.Replacement)
			applied = true
			fmt.Printf("  ✓ Applied pattern: %s\n", pattern.Name)
		}
	}

	return fixed, applied
}

// applyDomainExpertFixes fixes domain expert positioning violations.
func (f *Fixer) applyDomainExpertFixes(content string) (fixed string, applied bool) {
	fixed = content
	applied = false

	for _, pattern := range f.domainExpertPatterns {
		if pattern.Pattern.MatchString(fixed) {
			fixed = pattern.Pattern.ReplaceAllString(fixed, pattern.Replacement)
			applied = true
			fmt.Printf("  ✓ Applied pattern: %s\n", pattern.Name)
		}
	}

	return fixed, applied
}

// applyCoverLetterWording fixes standard cover letter wording patterns.
func (f *Fixer) applyCoverLetterWording(content string) (fixed string) {
	fixed = content

	for _, pattern := range f.coverLetterPatterns {
		if pattern.Pattern.MatchString(fixed) {
			fixed = pattern.Pattern.ReplaceAllString(fixed, pattern.Replacement)
		}
	}

	return fixed
}

// buildTemporalImpossibilityPatterns creates patterns for fixing temporal impossibility violations.
func buildTemporalImpossibilityPatterns() (patterns []FixPattern) {
	// Pattern: "25+ years of experience building [tech]" → "25+ years in [domain], with deep expertise in [tech]"
	patterns = []FixPattern{
		{
			Name:        "Temporal - building platform engineering",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting) (enterprise-scale |scalable |production )?platform engineering([,\n])`),
			Replacement: `$1$2 in software engineering and infrastructure** with deep expertise in $4platform engineering$5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - building AWS/cloud",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting) (AWS|Azure|GCP|multi-cloud|cloud-native) ([^,\n]+)`),
			Replacement: `$1$2 in distributed systems and platform engineering** with deep expertise in $4 $5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - building Kubernetes",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting) (Kubernetes|K8s|containerized|container-native) ([^,\n]+)`),
			Replacement: `$1$2 in platform engineering and distributed systems** with extensive $4 $5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - SRE/DevOps",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (in|of|building|architecting) (site reliability engineering|SRE|DevOps) ([^,\n]+)`),
			Replacement: `$1$2 in operational excellence and infrastructure automation** with deep $4 expertise $5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - AI-powered",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting|in) (AI-powered|AI-driven|machine learning) ([^,\n]+)`),
			Replacement: `$1$2 in system architecture and automation** with expertise in $4 $5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - DeFi/Blockchain",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting) (distributed DeFi|DeFi|blockchain|cryptocurrency) ([^,\n]+)`),
			Replacement: `$1$2 in distributed systems and platform engineering** with deep expertise building infrastructure for $4 $5`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
		{
			Name:        "Temporal - general tech prefix",
			Pattern:     regexp.MustCompile(`(?i)(\*\*[^*]+with )(25\+ years of experience\*\*) (building|architecting|developing) (enterprise-grade|scalable|production) ([^\n]*?) (AWS|Kubernetes|SRE|AI|DeFi|cloud-native|blockchain) ([^,\n]+)`),
			Replacement: `$1$2 in $4 $5 systems** with expertise in $6 $7`,
			RuleMatch:   "TEMPORAL_IMPOSSIBILITY",
		},
	}

	return patterns
}

// buildDomainExpertPatterns creates patterns for fixing domain expert claims.
func buildDomainExpertPatterns() (patterns []FixPattern) {
	patterns = []FixPattern{
		{
			Name:        "DeFi/Crypto Expert (any combination) → Infrastructure Architect",
			Pattern:     regexp.MustCompile(`(?i)\*\*([^*]*?)(DeFi|Cryptocurrency|Crypto)([^*]*?) Expert\*\* specializing in ([^\n]+)`),
			Replacement: `**Multi-Cloud Infrastructure Architect** specializing in Kubernetes platforms supporting cryptocurrency trading systems, blockchain infrastructure, and $4`,
			RuleMatch:   "FORBIDDEN_DOMAIN_CLAIM",
		},
		{
			Name:        "DeFi/Crypto Expert (any combo, no specializing) → Infrastructure Architect",
			Pattern:     regexp.MustCompile(`(?i)\*\*([^*]*?)(DeFi|Cryptocurrency|Crypto)([^*]*?) Expert\*\*`),
			Replacement: `**Multi-Cloud Infrastructure Architect**`,
			RuleMatch:   "FORBIDDEN_DOMAIN_CLAIM",
		},
		{
			Name:        "Domain Expert (specific domains) → Infrastructure role",
			Pattern:     regexp.MustCompile(`(?i)\*\*([^*]*?)(Climate|Gaming|Healthcare|Real Estate|Satellite|Geospatial)([^*]*?) Expert\*\*`),
			Replacement: `**Infrastructure Architect** with experience in $2 platforms`,
			RuleMatch:   "FORBIDDEN_DOMAIN_CLAIM",
		},
	}

	return patterns
}

// buildCoverLetterPatterns creates patterns for fixing cover letter wording.
func buildCoverLetterPatterns() (patterns []FixPattern) {
	patterns = []FixPattern{
		{
			Name:        "Targeted resume wording",
			Pattern:     regexp.MustCompile(`This is a targeted resume highlighting`),
			Replacement: `The resume submitted for this role highlights`,
			RuleMatch:   "COVER_LETTER_WORDING",
		},
		{
			Name:        "Weak quantification - 5 continents",
			Pattern:     regexp.MustCompile(`(?i)(across|spanning) 5 continents`),
			Replacement: `$1 North America, South America, Europe, Africa, and India`,
			RuleMatch:   "WEAK_QUANTIFICATION",
		},
		{
			Name:        "Weak quantification - 7 clusters",
			Pattern:     regexp.MustCompile(`(?i)(\d+\+? (?:WAF )?(?:security )?(?:events|logs) daily (?:across|over) )7 distributed clusters`),
			Replacement: `${1}multi-cluster distributed infrastructure`,
			RuleMatch:   "WEAK_QUANTIFICATION",
		},
	}

	return patterns
}
