package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/princepal9120/testgen-cli/internal/scanner"
)

var (
	// analyze command flags
	anaPath         string
	anaCostEstimate bool
	anaDetail       string
	anaRecursive    bool
	anaOutputFormat string
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze codebase for test generation cost estimation",
	Long: `Analyze source files to estimate test generation costs and complexity.

This command scans your codebase and provides:
  • Estimated token usage for LLM API calls
  • Approximate cost in USD
  • File and function counts per language
  • Complexity metrics

Examples:
  # Get cost estimate for a directory
  testgen analyze --path=./src --cost-estimate

  # Detailed per-file analysis
  testgen analyze --path=./src --cost-estimate --detail=per-file

  # Summary only
  testgen analyze --path=./src --detail=summary`,
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().StringVarP(&anaPath, "path", "p", ".", "directory to analyze")
	analyzeCmd.Flags().BoolVar(&anaCostEstimate, "cost-estimate", false, "show estimated API costs")
	analyzeCmd.Flags().StringVar(&anaDetail, "detail", "summary", "detail level: summary, per-file, per-function")
	analyzeCmd.Flags().BoolVarP(&anaRecursive, "recursive", "r", true, "analyze recursively")
	analyzeCmd.Flags().StringVar(&anaOutputFormat, "output-format", "text", "output format: text, json")
}

type AnalysisResult struct {
	Path            string                `json:"path"`
	TotalFiles      int                   `json:"total_files"`
	TotalFunctions  int                   `json:"total_functions"`
	TotalLines      int                   `json:"total_lines"`
	ByLanguage      map[string]LangStats  `json:"by_language"`
	EstimatedTokens int                   `json:"estimated_tokens,omitempty"`
	EstimatedCost   float64               `json:"estimated_cost_usd,omitempty"`
	Files           []FileAnalysis        `json:"files,omitempty"`
}

type LangStats struct {
	Files     int `json:"files"`
	Lines     int `json:"lines"`
	Functions int `json:"functions"`
}

type FileAnalysis struct {
	Path      string `json:"path"`
	Language  string `json:"language"`
	Lines     int    `json:"lines"`
	Functions int    `json:"functions"`
	Tokens    int    `json:"estimated_tokens,omitempty"`
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Make path absolute
	absPath, err := filepath.Abs(anaPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	log.Info("analyzing codebase",
		slog.String("path", absPath),
		slog.Bool("cost-estimate", anaCostEstimate),
		slog.String("detail", anaDetail),
	)

	// Scan for source files
	s := scanner.New(scanner.Options{
		Recursive: anaRecursive,
	})

	sourceFiles, err := s.Scan(absPath)
	if err != nil {
		return fmt.Errorf("failed to scan path: %w", err)
	}

	// Analyze
	result := analyzeFiles(sourceFiles, absPath)

	// Add cost estimation if requested
	if anaCostEstimate {
		estimateCosts(result)
	}

	// Output results
	return outputAnalysisResults(result, anaOutputFormat, anaDetail)
}

func analyzeFiles(files []*scanner.SourceFile, basePath string) *AnalysisResult {
	result := &AnalysisResult{
		Path:       basePath,
		ByLanguage: make(map[string]LangStats),
		Files:      make([]FileAnalysis, 0),
	}

	for _, f := range files {
		// Read file to count lines
		content, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}

		lines := len(strings.Split(string(content), "\n"))
		// Rough estimate: 1 function per 20 lines on average
		estimatedFunctions := max(1, lines/20)

		result.TotalFiles++
		result.TotalLines += lines
		result.TotalFunctions += estimatedFunctions

		// Update language stats
		lang := f.Language
		stats := result.ByLanguage[lang]
		stats.Files++
		stats.Lines += lines
		stats.Functions += estimatedFunctions
		result.ByLanguage[lang] = stats

		// Add file analysis
		relPath, _ := filepath.Rel(basePath, f.Path)
		result.Files = append(result.Files, FileAnalysis{
			Path:      relPath,
			Language:  lang,
			Lines:     lines,
			Functions: estimatedFunctions,
		})
	}

	return result
}

func estimateCosts(result *AnalysisResult) {
	// Rough token estimation:
	// - Average 4 chars per token
	// - Source code: ~50 tokens per function for context
	// - Generated test: ~100 tokens per function
	// - System prompt overhead: ~500 tokens per request
	
	tokensPerFunction := 150  // input context
	outputPerFunction := 200  // generated test
	batchSize := 5
	systemPromptTokens := 500

	totalInputTokens := (result.TotalFunctions * tokensPerFunction) + 
		((result.TotalFunctions / batchSize) * systemPromptTokens)
	totalOutputTokens := result.TotalFunctions * outputPerFunction

	result.EstimatedTokens = totalInputTokens + totalOutputTokens

	// Claude 3.5 Sonnet pricing (as of late 2024):
	// Input: $3.00 per 1M tokens
	// Output: $15.00 per 1M tokens
	inputCost := float64(totalInputTokens) * 3.00 / 1_000_000
	outputCost := float64(totalOutputTokens) * 15.00 / 1_000_000
	result.EstimatedCost = inputCost + outputCost
}

func outputAnalysisResults(result *AnalysisResult, format, detail string) error {
	// Filter files if not detailed
	if detail == "summary" {
		result.Files = nil
	}

	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	default:
		fmt.Printf("\n=== Codebase Analysis ===\n\n")
		fmt.Printf("Path:            %s\n", result.Path)
		fmt.Printf("Total files:     %d\n", result.TotalFiles)
		fmt.Printf("Total lines:     %d\n", result.TotalLines)
		fmt.Printf("Est. functions:  %d\n", result.TotalFunctions)

		if len(result.ByLanguage) > 0 {
			fmt.Printf("\n--- By Language ---\n")
			for lang, stats := range result.ByLanguage {
				fmt.Printf("  %s: %d files, %d lines, ~%d functions\n",
					lang, stats.Files, stats.Lines, stats.Functions)
			}
		}

		if result.EstimatedTokens > 0 {
			fmt.Printf("\n--- Cost Estimate ---\n")
			fmt.Printf("Estimated tokens: %d\n", result.EstimatedTokens)
			fmt.Printf("Estimated cost:   $%.2f USD\n", result.EstimatedCost)
		}

		if detail == "per-file" && len(result.Files) > 0 {
			fmt.Printf("\n--- Per-File Details ---\n")
			for _, f := range result.Files {
				fmt.Printf("  %s (%s): %d lines, ~%d functions\n",
					f.Path, f.Language, f.Lines, f.Functions)
			}
		}

		fmt.Println()
		return nil
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
