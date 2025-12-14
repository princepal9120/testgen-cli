/*
Package adapters provides language-specific behavior for test generation.

Each language has a dedicated adapter implementing the LanguageAdapter interface,
handling parsing, framework selection, test generation, and validation.
*/
package adapters

import (
	"github.com/princepal9120/testgen-cli/pkg/models"
)

// LanguageAdapter defines the interface for language-specific test generation
type LanguageAdapter interface {
	// CanHandle returns true if this adapter handles the given file
	CanHandle(filePath string) bool

	// GetLanguage returns the language this adapter handles
	GetLanguage() string

	// GetDefaultFramework returns the default test framework for this language
	GetDefaultFramework() string

	// GetSupportedFrameworks returns all supported test frameworks
	GetSupportedFrameworks() []string

	// ParseFile parses source code and returns an AST
	ParseFile(content string) (*models.AST, error)

	// ExtractDefinitions extracts functions and methods from parsed AST
	ExtractDefinitions(ast *models.AST) ([]*models.Definition, error)

	// SelectFramework determines the test framework to use
	SelectFramework(projectPath string) string

	// GenerateTestPath returns the expected path for a test file
	GenerateTestPath(sourcePath string, outputDir string) string

	// FormatTestCode formats the generated test code
	FormatTestCode(code string) (string, error)

	// GetPromptTemplate returns the prompt template for the given test type
	GetPromptTemplate(testType string) string

	// ValidateTests checks if generated tests compile/parse correctly
	ValidateTests(testCode string, testPath string) error

	// RunTests executes tests and returns results
	RunTests(testDir string) (*models.TestResults, error)
}

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	language   string
	frameworks []string
	defaultFW  string
}

// GetLanguage returns the language
func (b *BaseAdapter) GetLanguage() string {
	return b.language
}

// GetDefaultFramework returns the default framework
func (b *BaseAdapter) GetDefaultFramework() string {
	return b.defaultFW
}

// GetSupportedFrameworks returns supported frameworks
func (b *BaseAdapter) GetSupportedFrameworks() []string {
	return b.frameworks
}
