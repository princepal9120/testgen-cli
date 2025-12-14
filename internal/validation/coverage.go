package validation

import (
	"regexp"
	"strconv"
	"strings"
)

// CoverageParser parses coverage output from different test runners
type CoverageParser struct{}

// NewCoverageParser creates a new coverage parser
func NewCoverageParser() *CoverageParser {
	return &CoverageParser{}
}

// ParseGoCoverage parses Go coverage output
// Expected format: "coverage: 80.5% of statements"
func (p *CoverageParser) ParseGoCoverage(output string) float64 {
	re := regexp.MustCompile(`coverage:\s+([\d.]+)%`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}
	return 0
}

// ParsePytestCoverage parses pytest-cov output
// Expected format: "TOTAL ... 85%"
func (p *CoverageParser) ParsePytestCoverage(output string) float64 {
	re := regexp.MustCompile(`TOTAL\s+\d+\s+\d+\s+(\d+)%`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}
	return 0
}

// ParseJestCoverage parses Jest coverage output
// Expected format: "All files | 85.5 | 80.2 | 90.1 | 85.5 |"
func (p *CoverageParser) ParseJestCoverage(output string) float64 {
	re := regexp.MustCompile(`All files\s*\|\s*([\d.]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}
	return 0
}

// ParseCargoCoverage parses cargo-tarpaulin output
// Expected format: "85.50% coverage, 171/200 lines covered"
func (p *CoverageParser) ParseCargoCoverage(output string) float64 {
	re := regexp.MustCompile(`([\d.]+)%\s+coverage`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}
	return 0
}

// ParseCoverage auto-detects language and parses coverage
func (p *CoverageParser) ParseCoverage(output string, language string) float64 {
	switch strings.ToLower(language) {
	case "go":
		return p.ParseGoCoverage(output)
	case "python":
		return p.ParsePytestCoverage(output)
	case "javascript", "typescript":
		return p.ParseJestCoverage(output)
	case "rust":
		return p.ParseCargoCoverage(output)
	default:
		// Try all parsers
		if cov := p.ParseGoCoverage(output); cov > 0 {
			return cov
		}
		if cov := p.ParsePytestCoverage(output); cov > 0 {
			return cov
		}
		if cov := p.ParseJestCoverage(output); cov > 0 {
			return cov
		}
		return p.ParseCargoCoverage(output)
	}
}
