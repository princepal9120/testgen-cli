/*
Package generator implements the core test generation engine.

This package orchestrates the test generation process by coordinating
language adapters, LLM providers, and output formatting.
*/
package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/princepal9120/testgen-cli/internal/adapters"
	"github.com/princepal9120/testgen-cli/internal/llm"
	"github.com/princepal9120/testgen-cli/pkg/models"
)

// EngineConfig contains configuration for the generation engine
type EngineConfig struct {
	DryRun      bool
	Validate    bool
	OutputDir   string
	TestTypes   []string
	Framework   string
	BatchSize   int
	Parallelism int
	Provider    string // "anthropic" or "openai"
}

// Engine orchestrates test generation
type Engine struct {
	config   EngineConfig
	provider llm.Provider
	cache    *llm.Cache
	logger   *slog.Logger
}

// NewEngine creates a new generation engine
func NewEngine(config EngineConfig) (*Engine, error) {
	logger := slog.Default()

	// Initialize LLM provider
	var provider llm.Provider
	switch strings.ToLower(config.Provider) {
	case "openai":
		provider = llm.NewOpenAIProvider()
	case "gemini":
		provider = llm.NewGeminiProvider()
	case "groq":
		provider = llm.NewGroqProvider()
	default:
		// Default to Anthropic
		provider = llm.NewAnthropicProvider()
	}

	// Configure provider
	if err := provider.Configure(llm.ProviderConfig{}); err != nil {
		// Not configured, will fail on actual generation
		logger.Warn("LLM provider not configured", slog.String("error", err.Error()))
	}

	return &Engine{
		config:   config,
		provider: provider,
		cache:    llm.NewCache(10000),
		logger:   logger,
	}, nil
}

// Generate generates tests for a source file
func (e *Engine) Generate(sourceFile *models.SourceFile, adapter adapters.LanguageAdapter) (*models.GenerationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result := &models.GenerationResult{
		SourceFile: sourceFile,
	}

	// Read source file content
	content, err := os.ReadFile(sourceFile.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	// Parse file
	ast, err := adapter.ParseFile(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Extract definitions
	definitions, err := adapter.ExtractDefinitions(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to extract definitions: %w", err)
	}

	if len(definitions) == 0 {
		e.logger.Info("no functions found in file", slog.String("path", sourceFile.Path))
		return result, nil
	}

	e.logger.Debug("extracted definitions",
		slog.String("path", sourceFile.Path),
		slog.Int("count", len(definitions)),
	)

	// Generate tests for each definition
	var allTests strings.Builder
	functionsTested := make([]string, 0)

	for _, def := range definitions {
		for _, testType := range e.config.TestTypes {
			testCode, err := e.generateTestForDefinition(ctx, def, adapter, testType, ast.Package)
			if err != nil {
				e.logger.Warn("failed to generate test",
					slog.String("function", def.Name),
					slog.String("error", err.Error()),
				)
				continue
			}

			if testCode != "" {
				allTests.WriteString(testCode)
				allTests.WriteString("\n\n")
				functionsTested = append(functionsTested, def.Name)
			}
		}
	}

	if allTests.Len() == 0 {
		return result, nil
	}

	// Post-process: add imports, format
	finalCode := e.postProcess(allTests.String(), adapter, sourceFile.Language, ast)

	// Format code
	formattedCode, err := adapter.FormatTestCode(finalCode)
	if err != nil {
		e.logger.Warn("failed to format test code", slog.String("error", err.Error()))
		formattedCode = finalCode
	}

	result.TestCode = formattedCode
	result.FunctionsTested = functionsTested
	result.TestCount = len(functionsTested)

	// Determine test file path
	testPath := adapter.GenerateTestPath(sourceFile.Path, e.config.OutputDir)
	result.TestPath = testPath

	// Write file if not dry-run
	if !e.config.DryRun {
		if err := e.writeTestFile(testPath, formattedCode); err != nil {
			return nil, fmt.Errorf("failed to write test file: %w", err)
		}
		e.logger.Info("wrote test file", slog.String("path", testPath))
	}

	// Validate if requested
	if e.config.Validate && !e.config.DryRun {
		if err := adapter.ValidateTests(formattedCode, testPath); err != nil {
			result.Error = fmt.Errorf("validation failed: %w", err)
			e.logger.Warn("test validation failed", slog.String("error", err.Error()))
		}
	}

	return result, nil
}

func (e *Engine) generateTestForDefinition(
	ctx context.Context,
	def *models.Definition,
	adapter adapters.LanguageAdapter,
	testType string,
	packageName string,
) (string, error) {
	// Build prompt
	promptTemplate := adapter.GetPromptTemplate(testType)
	prompt := fmt.Sprintf(promptTemplate, def.Body, packageName)

	// Check cache
	cacheKey := e.cache.GenerateKey(prompt, "", e.provider.Name())
	if cached, hit := e.cache.Get(cacheKey); hit {
		e.logger.Debug("cache hit", slog.String("function", def.Name))
		return cached.Content, nil
	}

	// Call LLM
	systemRole := fmt.Sprintf("You are an expert %s developer. Generate production-quality tests that follow best practices. Output only the test code, no explanations.", adapter.GetLanguage())

	resp, err := e.provider.Complete(ctx, llm.CompletionRequest{
		Prompt:      prompt,
		SystemRole:  systemRole,
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	// Cache result
	e.cache.Set(cacheKey, resp)

	// Extract code from response
	code := extractCodeFromResponse(resp.Content, adapter.GetLanguage())

	return code, nil
}

// extractCodeFromResponse extracts code blocks from LLM response
func extractCodeFromResponse(response string, language string) string {
	// Try to extract from markdown code blocks
	patterns := []string{
		"```" + language + `\n([\s\S]*?)` + "```",
		"```" + `\n([\s\S]*?)` + "```",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(response); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// If no code blocks, return the whole response (might be plain code)
	return strings.TrimSpace(response)
}

func (e *Engine) postProcess(code string, adapter adapters.LanguageAdapter, language string, ast *models.AST) string {
	// Add standard imports based on language
	var imports string

	switch language {
	case "go":
		imports = `package ` + ast.Package + `_test

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

`
	case "python":
		imports = `import pytest
from unittest.mock import Mock, patch

`
	case "javascript", "typescript":
		// Imports depend on the source file
		imports = ""
	case "rust":
		imports = `#[cfg(test)]
mod tests {
    use super::*;

`
	}

	// For Go, check if package declaration exists
	if language == "go" && strings.Contains(code, "package ") {
		return code
	}

	return imports + code
}

func (e *Engine) writeTestFile(path string, content string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// GetUsage returns LLM usage metrics
func (e *Engine) GetUsage() *llm.UsageMetrics {
	return e.provider.GetUsage()
}

// GetCacheStats returns cache statistics
func (e *Engine) GetCacheStats() (size int, hits int, misses int, hitRate float64) {
	return e.cache.Stats()
}

// GeneratedTestJSON represents the expected JSON structure from LLM
type GeneratedTestJSON struct {
	TestName     string   `json:"test_name"`
	TestCode     string   `json:"test_code"`
	Imports      []string `json:"imports"`
	EdgeCases    []string `json:"edge_cases_covered"`
	Dependencies []string `json:"mocked_dependencies"`
}

// parseStructuredOutput attempts to parse structured JSON from LLM response
func parseStructuredOutput(response string) (*GeneratedTestJSON, error) {
	// Try to find JSON in response
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonMatch := jsonRegex.FindString(response)
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result GeneratedTestJSON
	if err := json.Unmarshal([]byte(jsonMatch), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}
