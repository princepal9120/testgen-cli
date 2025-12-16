package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/princepal9120/testgen-cli/pkg/models"
)

// PythonAdapter handles Python source files
type PythonAdapter struct {
	BaseAdapter
}

// NewPythonAdapter creates a new Python language adapter
func NewPythonAdapter() *PythonAdapter {
	return &PythonAdapter{
		BaseAdapter: BaseAdapter{
			language:   "python",
			frameworks: []string{"pytest", "unittest"},
			defaultFW:  "pytest",
		},
	}
}

// CanHandle returns true if this adapter can handle the file
func (a *PythonAdapter) CanHandle(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".py")
}

// ParseFile parses Python source code and extracts structure
func (a *PythonAdapter) ParseFile(content string) (*models.AST, error) {
	ast := &models.AST{
		Language:    "python",
		Definitions: make([]*models.Definition, 0),
		Imports:     make([]string, 0),
	}

	lines := strings.Split(content, "\n")

	// Extract imports
	importRegex := regexp.MustCompile(`^(?:from\s+(\S+)\s+)?import\s+(.+)$`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := importRegex.FindStringSubmatch(trimmed); matches != nil {
			if matches[1] != "" {
				ast.Imports = append(ast.Imports, matches[1])
			} else {
				imports := strings.Split(matches[2], ",")
				for _, imp := range imports {
					ast.Imports = append(ast.Imports, strings.TrimSpace(imp))
				}
			}
		}
	}

	// Extract function definitions
	// Pattern: def function_name(params):
	funcRegex := regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(([^)]*)\)\s*(?:->\s*([^:]+))?\s*:`)

	// Extract class definitions for context
	classRegex := regexp.MustCompile(`^class\s+(\w+)`)

	var currentClass string
	var currentIndent int

	for i, line := range lines {
		// Check for class definition
		if matches := classRegex.FindStringSubmatch(line); matches != nil {
			currentClass = matches[1]
			currentIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			continue
		}

		// Check for function definition
		if matches := funcRegex.FindStringSubmatch(line); matches != nil {
			indent := len(matches[1])

			def := &models.Definition{
				Name:      matches[2],
				StartLine: i + 1,
			}

			// Build signature
			def.Signature = fmt.Sprintf("def %s(%s)", matches[2], matches[3])
			if matches[4] != "" {
				def.ReturnType = strings.TrimSpace(matches[4])
				def.Signature += " -> " + def.ReturnType
			}

			// Parse parameters
			def.Parameters = parsePythonParams(matches[3])

			// Check if it's a method (indented inside a class)
			if currentClass != "" && indent > currentIndent {
				def.IsMethod = true
				def.ClassName = currentClass
			}

			// Find function body (until dedent or EOF)
			def.EndLine = findPythonFunctionEnd(lines, i, indent)
			if def.EndLine > def.StartLine {
				bodyLines := lines[def.StartLine:def.EndLine]
				def.Body = strings.Join(bodyLines, "\n")
			}

			// Extract docstring if present
			def.Docstring = extractPythonDocstring(lines, i+1)

			ast.Definitions = append(ast.Definitions, def)
		}
	}

	return ast, nil
}

// parsePythonParams parses Python function parameters
func parsePythonParams(paramStr string) []models.Param {
	params := make([]models.Param, 0)
	if strings.TrimSpace(paramStr) == "" {
		return params
	}

	// Split by comma, handling default values
	parts := splitPythonParams(paramStr)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "self" || part == "cls" {
			continue
		}

		param := models.Param{}

		// Check for type annotation
		if colonIdx := strings.Index(part, ":"); colonIdx > 0 {
			param.Name = strings.TrimSpace(part[:colonIdx])
			typeAndDefault := part[colonIdx+1:]
			if eqIdx := strings.Index(typeAndDefault, "="); eqIdx > 0 {
				param.Type = strings.TrimSpace(typeAndDefault[:eqIdx])
			} else {
				param.Type = strings.TrimSpace(typeAndDefault)
			}
		} else if eqIdx := strings.Index(part, "="); eqIdx > 0 {
			param.Name = strings.TrimSpace(part[:eqIdx])
		} else {
			param.Name = part
		}

		params = append(params, param)
	}

	return params
}

// splitPythonParams splits parameter string handling nested brackets
func splitPythonParams(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range s {
		switch ch {
		case '[', '(':
			depth++
			current.WriteRune(ch)
		case ']', ')':
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

// findPythonFunctionEnd finds where a Python function ends
func findPythonFunctionEnd(lines []string, startIdx int, funcIndent int) int {
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Calculate indent
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// If we hit a line with same or less indent, function ended
		if indent <= funcIndent {
			return i
		}
	}
	return len(lines)
}

// extractPythonDocstring extracts docstring from function
func extractPythonDocstring(lines []string, startLine int) string {
	if startLine >= len(lines) {
		return ""
	}

	// Look for docstring in first non-empty line after def
	for i := startLine; i < len(lines) && i < startLine+3; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			quote := trimmed[:3]
			// Single line docstring
			if strings.HasSuffix(trimmed, quote) && len(trimmed) > 6 {
				return trimmed[3 : len(trimmed)-3]
			}
			// Multi-line docstring
			var doc strings.Builder
			doc.WriteString(trimmed[3:])
			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], quote) {
					idx := strings.Index(lines[j], quote)
					doc.WriteString(strings.TrimSpace(lines[j][:idx]))
					return doc.String()
				}
				doc.WriteString("\n")
				doc.WriteString(strings.TrimSpace(lines[j]))
			}
		}
		break
	}
	return ""
}

// ExtractDefinitions returns definitions from parsed AST
func (a *PythonAdapter) ExtractDefinitions(ast *models.AST) ([]*models.Definition, error) {
	if ast == nil {
		return nil, fmt.Errorf("nil AST provided")
	}
	return ast.Definitions, nil
}

// SelectFramework determines the test framework to use
func (a *PythonAdapter) SelectFramework(projectPath string) string {
	// Check for pytest in common config files
	configFiles := []string{"pytest.ini", "pyproject.toml", "setup.cfg"}
	for _, cfg := range configFiles {
		cfgPath := filepath.Join(projectPath, cfg)
		if content, err := os.ReadFile(cfgPath); err == nil {
			if strings.Contains(string(content), "pytest") {
				return "pytest"
			}
		}
	}

	// Check requirements.txt
	reqPath := filepath.Join(projectPath, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		if strings.Contains(string(content), "pytest") {
			return "pytest"
		}
	}

	return a.defaultFW
}

// GenerateTestPath returns the expected path for a test file
func (a *PythonAdapter) GenerateTestPath(sourcePath string, outputDir string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	name := strings.TrimSuffix(base, ".py")

	// Python convention: tests/test_<module>.py
	testDir := outputDir
	if testDir == "" {
		testDir = filepath.Join(dir, "..", "tests")
	}

	return filepath.Join(testDir, "test_"+name+".py")
}

// FormatTestCode formats Python test code
func (a *PythonAdapter) FormatTestCode(code string) (string, error) {
	// Try black, then autopep8
	formatters := []string{"black", "autopep8"}

	tmpFile, err := os.CreateTemp("", "testgen_*.py")
	if err != nil {
		return code, nil
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return code, nil
	}
	tmpFile.Close()

	for _, formatter := range formatters {
		ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
		defer cancel()

		var cmd *exec.Cmd
		if formatter == "black" {
			cmd = exec.CommandContext(ctx, "black", "--quiet", tmpFile.Name())
		} else {
			cmd = exec.CommandContext(ctx, "autopep8", "--in-place", tmpFile.Name())
		}

		if err := cmd.Run(); err == nil {
			// Formatter succeeded
			formatted, err := os.ReadFile(tmpFile.Name())
			if err == nil {
				return string(formatted), nil
			}
		}
	}

	return code, nil
}

// GetPromptTemplate returns the prompt template for Python tests
func (a *PythonAdapter) GetPromptTemplate(testType string) string {
	basePrompt := `Generate idiomatic Python tests using pytest for the following function.

Requirements:
- Use pytest conventions and fixtures
- Use descriptive test function names (test_<scenario>)
- Include docstrings for test functions
- Use assert statements (pytest style)
- Handle exceptions with pytest.raises
- Use @pytest.mark.parametrize for multiple test cases

Function to test:
%s

Module: %s
`

	switch testType {
	case "edge-cases":
		return basePrompt + `
Focus on edge cases and boundary conditions:
- None/empty inputs
- Empty strings, lists, dicts
- Zero values
- Very large values
- Unicode and special characters
- Type errors
`

	case "negative":
		return basePrompt + `
Focus on error handling and negative test cases:
- Invalid inputs that should raise exceptions
- Type errors
- Value errors
- Boundary violations
- Use pytest.raises for exception testing
`

	default: // unit
		return basePrompt + `
Generate comprehensive unit tests covering:
- Happy path scenarios
- Basic edge cases
- Error conditions

Example structure:
` + "```python" + `
import pytest
from module import function_name

class TestFunctionName:
    """Test suite for function_name."""

    def test_happy_path(self):
        """Should handle normal input correctly."""
        result = function_name(valid_input)
        assert result == expected_output

    @pytest.mark.parametrize("input,expected", [
        (input1, output1),
        (input2, output2),
    ])
    def test_various_inputs(self, input, expected):
        """Should handle various inputs correctly."""
        assert function_name(input) == expected

    def test_invalid_input_raises_error(self):
        """Should raise ValueError for invalid input."""
        with pytest.raises(ValueError):
            function_name(invalid_input)
` + "```"
	}
}

// ValidateTests checks if generated tests are valid Python
func (a *PythonAdapter) ValidateTests(testCode string, testPath string) error {
	// Write test file
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}
	defer os.Remove(testPath)

	// Check syntax with py_compile
	ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python", "-m", "py_compile", testPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(output))
	}

	return nil
}

// RunTests executes Python tests and returns results
func (a *PythonAdapter) RunTests(testDir string) (*models.TestResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python", "-m", "pytest", "-v", "--tb=short", testDir)
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
	passedRegex := regexp.MustCompile(`(\d+) passed`)
	failedRegex := regexp.MustCompile(`(\d+) failed`)

	if matches := passedRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &results.PassedCount)
	}
	if matches := failedRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &results.FailedCount)
	}

	return results, nil
}
