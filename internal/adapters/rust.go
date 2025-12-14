package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/testgen/testgen/pkg/models"
)

// RustAdapter handles Rust source files
type RustAdapter struct {
	BaseAdapter
}

// NewRustAdapter creates a new Rust language adapter
func NewRustAdapter() *RustAdapter {
	return &RustAdapter{
		BaseAdapter: BaseAdapter{
			language:   "rust",
			frameworks: []string{"cargo-test"},
			defaultFW:  "cargo-test",
		},
	}
}

// CanHandle returns true if this adapter can handle the file
func (a *RustAdapter) CanHandle(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".rs")
}

// ParseFile parses Rust source code and extracts structure
func (a *RustAdapter) ParseFile(content string) (*models.AST, error) {
	ast := &models.AST{
		Language:    "rust",
		Definitions: make([]*models.Definition, 0),
		Imports:     make([]string, 0),
	}

	lines := strings.Split(content, "\n")

	// Extract use statements
	useRegex := regexp.MustCompile(`^use\s+([^;]+);`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := useRegex.FindStringSubmatch(trimmed); matches != nil {
			ast.Imports = append(ast.Imports, matches[1])
		}
	}

	// Extract function definitions
	// Pattern: pub? async? fn name<generics>(params) -> ReturnType
	funcRegex := regexp.MustCompile(`^(\s*)(pub\s+)?(async\s+)?fn\s+(\w+)(?:<[^>]+>)?\s*\(([^)]*)\)(?:\s*->\s*([^\{]+))?\s*\{?`)

	// Track impl blocks for methods
	implRegex := regexp.MustCompile(`^impl(?:<[^>]+>)?\s+(?:(\w+)\s+for\s+)?(\w+)`)
	var currentImpl string

	for i, line := range lines {
		// Check for impl block
		if matches := implRegex.FindStringSubmatch(line); matches != nil {
			if matches[1] != "" {
				currentImpl = matches[2] // trait impl
			} else {
				currentImpl = matches[2] // direct impl
			}
			continue
		}

		// Check for function
		if matches := funcRegex.FindStringSubmatch(line); matches != nil {
			def := &models.Definition{
				Name:      matches[4],
				StartLine: i + 1,
			}

			// Build signature
			sig := ""
			if matches[2] != "" {
				sig += "pub "
			}
			if matches[3] != "" {
				sig += "async "
			}
			sig += "fn " + matches[4] + "(" + matches[5] + ")"
			if matches[6] != "" {
				def.ReturnType = strings.TrimSpace(matches[6])
				sig += " -> " + def.ReturnType
			}
			def.Signature = sig

			// Parse parameters
			def.Parameters = parseRustParams(matches[5])

			// Check if inside impl block
			indent := len(matches[1])
			if currentImpl != "" && indent > 0 {
				def.IsMethod = true
				def.ClassName = currentImpl
			}

			// Find function end
			def.EndLine = findRustFunctionEnd(lines, i)
			if def.EndLine > def.StartLine {
				bodyLines := lines[def.StartLine-1 : def.EndLine]
				def.Body = strings.Join(bodyLines, "\n")
			}

			ast.Definitions = append(ast.Definitions, def)
		}
	}

	return ast, nil
}

// parseRustParams parses Rust function parameters
func parseRustParams(paramStr string) []models.Param {
	params := make([]models.Param, 0)
	if strings.TrimSpace(paramStr) == "" {
		return params
	}

	// Split by comma, handling generics
	parts := splitRustParams(paramStr)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "self" || part == "&self" || part == "&mut self" {
			continue
		}

		param := models.Param{}

		// Pattern: name: Type
		if colonIdx := strings.Index(part, ":"); colonIdx > 0 {
			param.Name = strings.TrimSpace(part[:colonIdx])
			param.Type = strings.TrimSpace(part[colonIdx+1:])
		} else {
			param.Name = part
		}

		params = append(params, param)
	}

	return params
}

// splitRustParams splits parameter string handling generics
func splitRustParams(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range s {
		switch ch {
		case '<', '(':
			depth++
			current.WriteRune(ch)
		case '>', ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// findRustFunctionEnd finds where a Rust function ends
func findRustFunctionEnd(lines []string, startIdx int) int {
	depth := 0
	started := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				depth++
				started = true
			} else if ch == '}' {
				depth--
				if started && depth == 0 {
					return i + 1
				}
			}
		}
	}

	return len(lines)
}

// ExtractDefinitions returns definitions from parsed AST
func (a *RustAdapter) ExtractDefinitions(ast *models.AST) ([]*models.Definition, error) {
	if ast == nil {
		return nil, fmt.Errorf("nil AST provided")
	}
	return ast.Definitions, nil
}

// SelectFramework determines the test framework to use
func (a *RustAdapter) SelectFramework(projectPath string) string {
	// Rust uses cargo test by default
	return a.defaultFW
}

// GenerateTestPath returns the expected path for a test file
func (a *RustAdapter) GenerateTestPath(sourcePath string, outputDir string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	name := strings.TrimSuffix(base, ".rs")

	// Rust convention: tests in same file (mod tests) or tests/ directory
	if outputDir != "" {
		return filepath.Join(outputDir, name+"_test.rs")
	}

	// Check if tests directory exists
	testsDir := filepath.Join(filepath.Dir(dir), "tests")
	if info, err := os.Stat(testsDir); err == nil && info.IsDir() {
		return filepath.Join(testsDir, name+"_test.rs")
	}

	// Return path for inline tests (append to same file)
	return sourcePath + ".test"
}

// FormatTestCode formats Rust test code using rustfmt
func (a *RustAdapter) FormatTestCode(code string) (string, error) {
	tmpFile, err := os.CreateTemp("", "testgen_*.rs")
	if err != nil {
		return code, nil
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return code, nil
	}
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rustfmt", tmpFile.Name())
	if err := cmd.Run(); err == nil {
		formatted, err := os.ReadFile(tmpFile.Name())
		if err == nil {
			return string(formatted), nil
		}
	}

	return code, nil
}

// GetPromptTemplate returns the prompt template for Rust tests
func (a *RustAdapter) GetPromptTemplate(testType string) string {
	basePrompt := `Generate idiomatic Rust tests for the following function.

Requirements:
- Use #[cfg(test)] mod tests block
- Use #[test] attribute for test functions
- Use assert!, assert_eq!, assert_ne! macros
- Handle Result<T, E> types properly
- Use #[should_panic] for panic tests
- Follow Rust naming conventions (snake_case)

Function to test:
%s

Module: %s
`

	switch testType {
	case "edge-cases":
		return basePrompt + `
Focus on edge cases and boundary conditions:
- Empty values (empty strings, vecs, etc.)
- Zero values
- Maximum/minimum numeric values
- Unicode and special characters
- Option::None handling
`

	case "negative":
		return basePrompt + `
Focus on error handling and negative test cases:
- Result::Err cases
- Invalid inputs
- Panic conditions with #[should_panic]
- Error type validation
`

	default: // unit
		return basePrompt + `
Generate comprehensive unit tests covering:
- Happy path scenarios
- Basic edge cases
- Error conditions

Example structure:
` + "```rust" + `
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_function_happy_path() {
        let result = function_name(valid_input);
        assert_eq!(result, Ok(expected_output));
    }

    #[test]
    fn test_function_edge_case() {
        let result = function_name(edge_input);
        assert!(result.is_ok());
    }

    #[test]
    fn test_function_error_case() {
        let result = function_name(invalid_input);
        assert!(result.is_err());
    }

    #[test]
    #[should_panic(expected = "error message")]
    fn test_function_panics() {
        function_that_panics();
    }
}
` + "```"
	}
}

// ValidateTests checks if generated tests compile
func (a *RustAdapter) ValidateTests(testCode string, testPath string) error {
	// For Rust, we need to be in a cargo project
	// This is a simplified check
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}
	defer os.Remove(testPath)

	// Try to compile with rustc (syntax check only)
	ctx, cancel := context.WithTimeout(context.Background(), 30*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rustc", "--edition", "2021", "--emit", "metadata", "-o", "/dev/null", testPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// May fail due to missing crate dependencies, which is OK for syntax check
		outputStr := string(output)
		if strings.Contains(outputStr, "error[E") && !strings.Contains(outputStr, "unresolved") {
			return fmt.Errorf("compilation error: %s", outputStr)
		}
	}

	return nil
}

// RunTests executes Rust tests and returns results
func (a *RustAdapter) RunTests(testDir string) (*models.TestResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*1e9) // 5 minutes for cargo
	defer cancel()

	// Find Cargo.toml
	cargoPath := testDir
	for cargoPath != "/" {
		if _, err := os.Stat(filepath.Join(cargoPath, "Cargo.toml")); err == nil {
			break
		}
		cargoPath = filepath.Dir(cargoPath)
	}

	cmd := exec.CommandContext(ctx, "cargo", "test", "--", "--nocapture")
	cmd.Dir = cargoPath

	output, err := cmd.CombinedOutput()

	results := &models.TestResults{
		Output:   string(output),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			results.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run tests: %w", err)
		}
	}

	// Parse output for pass/fail counts
	outputStr := string(output)
	
	// Pattern: test result: ok. X passed; Y failed; Z ignored
	resultRegex := regexp.MustCompile(`test result:.*?(\d+) passed.*?(\d+) failed`)
	if matches := resultRegex.FindStringSubmatch(outputStr); len(matches) > 2 {
		fmt.Sscanf(matches[1], "%d", &results.PassedCount)
		fmt.Sscanf(matches[2], "%d", &results.FailedCount)
	}

	return results, nil
}
