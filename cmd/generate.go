package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/princepal9120/testgen-cli/internal/adapters"
	"github.com/princepal9120/testgen-cli/internal/generator"
	"github.com/princepal9120/testgen-cli/internal/scanner"
	"github.com/princepal9120/testgen-cli/pkg/models"
)

var (
	// generate command flags
	genPath           string
	genFile           string
	genTypes          []string
	genFramework      string
	genOutput         string
	genRecursive      bool
	genParallel       int
	genDryRun         bool
	genValidate       bool
	genOutputFormat   string
	genIncludePattern string
	genExcludePattern string
	genBatchSize      int
	genReportUsage    bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate tests for source files",
	Long: `Generate tests for specified source files or directories.

TestGen analyzes your source code, extracts function definitions, and
generates comprehensive tests using AI. Tests follow language-specific
conventions and best practices.

Test Types:
  unit         - Basic unit tests covering happy path and common errors
  edge-cases   - Boundary conditions, nulls, extremes  
  negative     - Exception paths, invalid inputs
  table-driven - Parameterized tests (Go idiom)
  integration  - Tests with mocked external dependencies

Examples:
  # Generate unit tests for a single file
  testgen generate --file=./src/utils.py --type=unit

  # Generate multiple test types for a directory
  testgen generate --path=./src --type=unit,edge-cases --recursive

  # Preview without writing files
  testgen generate --path=./src --dry-run

  # Generate and validate tests
  testgen generate --path=./src --validate`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Path/file flags
	generateCmd.Flags().StringVarP(&genPath, "path", "p", "", "source directory to generate tests for")
	generateCmd.Flags().StringVar(&genFile, "file", "", "single source file to generate tests for")

	// Test configuration
	generateCmd.Flags().StringSliceVarP(&genTypes, "type", "t", []string{"unit"}, "test types: unit, edge-cases, negative, table-driven, integration")
	generateCmd.Flags().StringVarP(&genFramework, "framework", "f", "", "target test framework (auto-detected by default)")
	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "", "output directory for generated tests")

	// Processing options
	generateCmd.Flags().BoolVarP(&genRecursive, "recursive", "r", false, "process directories recursively")
	generateCmd.Flags().IntVarP(&genParallel, "parallel", "j", 2, "number of parallel workers")
	generateCmd.Flags().IntVar(&genBatchSize, "batch-size", 5, "batch size for API requests")

	// Output options
	generateCmd.Flags().BoolVar(&genDryRun, "dry-run", false, "preview output without writing files")
	generateCmd.Flags().BoolVar(&genValidate, "validate", false, "run generated tests after creation")
	generateCmd.Flags().StringVar(&genOutputFormat, "output-format", "text", "output format: text, json")

	// Filtering options
	generateCmd.Flags().StringVar(&genIncludePattern, "include-pattern", "", "glob pattern for files to include")
	generateCmd.Flags().StringVar(&genExcludePattern, "exclude-pattern", "", "glob pattern for files to exclude")

	// Reporting
	generateCmd.Flags().BoolVar(&genReportUsage, "report-usage", false, "generate usage/cost report")

	// Bind to viper
	viper.BindPFlag("generation.parallel_workers", generateCmd.Flags().Lookup("parallel"))
	viper.BindPFlag("generation.batch_size", generateCmd.Flags().Lookup("batch-size"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Validate inputs
	if genPath == "" && genFile == "" {
		return fmt.Errorf("either --path or --file is required")
	}

	// Determine target path
	targetPath := genPath
	if genFile != "" {
		targetPath = genFile
	}

	// Make path absolute
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	log.Info("starting test generation",
		slog.String("path", absPath),
		slog.Any("types", genTypes),
		slog.Bool("recursive", genRecursive),
		slog.Bool("dry-run", genDryRun),
	)

	// Initialize scanner
	scannerOpts := scanner.Options{
		Recursive:      genRecursive,
		IncludePattern: genIncludePattern,
		ExcludePattern: genExcludePattern,
	}

	s := scanner.New(scannerOpts)

	// Scan for source files
	sourceFiles, err := s.Scan(absPath)
	if err != nil {
		return fmt.Errorf("failed to scan path: %w", err)
	}

	if len(sourceFiles) == 0 {
		log.Warn("no source files found", slog.String("path", absPath))
		return nil
	}

	log.Info("found source files",
		slog.Int("count", len(sourceFiles)),
		slog.String("path", absPath),
	)

	// Group files by language for statistics
	langCounts := make(map[string]int)
	for _, f := range sourceFiles {
		langCounts[f.Language]++
	}
	for lang, count := range langCounts {
		log.Debug("files by language", slog.String("language", lang), slog.Int("count", count))
	}

	// Initialize the generator engine
	engine, err := generator.NewEngine(generator.EngineConfig{
		DryRun:      genDryRun,
		Validate:    genValidate,
		OutputDir:   genOutput,
		TestTypes:   genTypes,
		Framework:   genFramework,
		BatchSize:   genBatchSize,
		Parallelism: genParallel,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Process files
	results := processFiles(sourceFiles, engine, log)

	// Output results
	if err := outputResults(results, genOutputFormat, genDryRun); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	// Summary
	successCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Error != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	log.Info("generation complete",
		slog.Int("success", successCount),
		slog.Int("errors", errorCount),
		slog.Int("total", len(results)),
	)

	if errorCount > 0 {
		return fmt.Errorf("%d file(s) failed to generate tests", errorCount)
	}

	return nil
}

func processFiles(files []*models.SourceFile, engine *generator.Engine, log *slog.Logger) []*models.GenerationResult {
	results := make([]*models.GenerationResult, 0, len(files))
	var mu sync.Mutex

	// Get adapter registry
	registry := adapters.DefaultRegistry()

	// Process files (parallel processing will be added later)
	for _, file := range files {
		log.Debug("processing file", slog.String("path", file.Path), slog.String("language", file.Language))

		// Get appropriate adapter
		adapter := registry.GetAdapter(file.Language)
		if adapter == nil {
			mu.Lock()
			results = append(results, &models.GenerationResult{
				SourceFile: file,
				Error:      fmt.Errorf("no adapter for language: %s", file.Language),
			})
			mu.Unlock()
			continue
		}

		// Generate tests
		result, err := engine.Generate(file, adapter)
		if err != nil {
			mu.Lock()
			results = append(results, &models.GenerationResult{
				SourceFile: file,
				Error:      err,
			})
			mu.Unlock()
			continue
		}

		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}

	return results
}

func outputResults(results []*models.GenerationResult, format string, dryRun bool) error {
	switch strings.ToLower(format) {
	case "json":
		return outputJSON(results)
	default:
		return outputText(results, dryRun)
	}
}

func outputJSON(results []*models.GenerationResult) error {
	output := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		item := map[string]interface{}{
			"source_file": r.SourceFile.Path,
			"language":    r.SourceFile.Language,
			"success":     r.Error == nil,
		}
		if r.Error != nil {
			item["error"] = r.Error.Error()
		}
		if r.TestCode != "" {
			item["test_file"] = r.TestPath
			item["functions_tested"] = len(r.FunctionsTested)
		}
		output = append(output, item)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputText(results []*models.GenerationResult, dryRun bool) error {
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("✗ %s: %v\n", r.SourceFile.Path, r.Error)
			continue
		}

		if dryRun && r.TestCode != "" {
			fmt.Printf("\n--- %s (generated test) ---\n", r.SourceFile.Path)
			fmt.Println(r.TestCode)
			fmt.Println()
		} else if r.TestPath != "" {
			fmt.Printf("✓ %s → %s (%d functions)\n", r.SourceFile.Path, r.TestPath, len(r.FunctionsTested))
		}
	}
	return nil
}
