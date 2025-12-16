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

// GoAdapter handles Go source files
type GoAdapter struct {
	BaseAdapter
}

// NewGoAdapter creates a new Go language adapter
func NewGoAdapter() *GoAdapter {
	return &GoAdapter{
		BaseAdapter: BaseAdapter{
			language:   "go",
			frameworks: []string{"testing", "testify"},
			defaultFW:  "testing",
		},
	}
}

// CanHandle returns true if this adapter can handle the file
func (a *GoAdapter) CanHandle(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".go")
}

// ParseFile parses Go source code and extracts structure
func (a *GoAdapter) ParseFile(content string) (*models.AST, error) {
	ast := &models.AST{
		Language:    "go",
		Definitions: make([]*models.Definition, 0),
		Imports:     make([]string, 0),
	}

	// Extract package name
	pkgRegex := regexp.MustCompile(`(?m)^package\s+(\w+)`)
	if matches := pkgRegex.FindStringSubmatch(content); len(matches) > 1 {
		ast.Package = matches[1]
	}

	// Extract imports
	importRegex := regexp.MustCompile(`(?m)import\s+(?:\(\s*([\s\S]*?)\s*\)|"([^"]+)")`)
	if matches := importRegex.FindAllStringSubmatch(content, -1); matches != nil {
		for _, match := range matches {
			if match[1] != "" {
				// Multi-line import
				lines := strings.Split(match[1], "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "//") {
						// Extract import path from quotes
						if idx := strings.Index(line, `"`); idx >= 0 {
							end := strings.LastIndex(line, `"`)
							if end > idx {
								ast.Imports = append(ast.Imports, line[idx+1:end])
							}
						}
					}
				}
			} else if match[2] != "" {
				// Single import
				ast.Imports = append(ast.Imports, match[2])
			}
		}
	}

	// Extract function definitions
	// Pattern: func (receiver) FunctionName(params) (returns) {
	funcRegex := regexp.MustCompile(`(?m)^func\s+(?:\((\w+)\s+\*?(\w+)\)\s+)?(\w+)\s*\(([^)]*)\)\s*(?:\(([^)]*)\)|(\w+))?\s*\{`)

	lines := strings.Split(content, "\n")
	matches := funcRegex.FindAllStringSubmatchIndex(content, -1)

	for _, matchIdx := range matches {
		if len(matchIdx) < 2 {
			continue
		}

		fullMatch := content[matchIdx[0]:matchIdx[1]]

		// Calculate line number
		lineNum := strings.Count(content[:matchIdx[0]], "\n") + 1

		// Extract function components
		submatches := funcRegex.FindStringSubmatch(fullMatch)
		if len(submatches) < 4 {
			continue
		}

		def := &models.Definition{
			StartLine: lineNum,
		}

		// Check if it's a method (has receiver)
		if submatches[1] != "" && submatches[2] != "" {
			def.IsMethod = true
			def.ClassName = submatches[2]
		}

		def.Name = submatches[3]
		def.Signature = strings.TrimSuffix(strings.TrimSpace(fullMatch), "{")

		// Parse parameters
		if submatches[4] != "" {
			def.Parameters = parseGoParams(submatches[4])
		}

		// Parse return type
		if len(submatches) > 5 && submatches[5] != "" {
			def.ReturnType = submatches[5]
		} else if len(submatches) > 6 && submatches[6] != "" {
			def.ReturnType = submatches[6]
		}

		// Find function body (simplified - find matching brace)
		bodyStart := matchIdx[1] - 1 // Position of opening brace
		if bodyStart < len(content) {
			endLine := findMatchingBrace(content, bodyStart, lines)
			def.EndLine = endLine
			if endLine > lineNum && endLine <= len(lines) {
				bodyLines := lines[lineNum-1 : endLine]
				def.Body = strings.Join(bodyLines, "\n")
			}
		}

		ast.Definitions = append(ast.Definitions, def)
	}

	return ast, nil
}

// parseGoParams parses Go function parameters
func parseGoParams(paramStr string) []models.Param {
	params := make([]models.Param, 0)
	if strings.TrimSpace(paramStr) == "" {
		return params
	}

	// Split by comma, handling grouped types
	parts := strings.Split(paramStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split into name and type
		fields := strings.Fields(part)
		if len(fields) >= 2 {
			params = append(params, models.Param{
				Name: fields[0],
				Type: strings.Join(fields[1:], " "),
			})
		} else if len(fields) == 1 {
			// Type only (e.g., in func(int, int))
			params = append(params, models.Param{
				Type: fields[0],
			})
		}
	}

	return params
}

// findMatchingBrace finds the line number of the matching closing brace
func findMatchingBrace(content string, start int, lines []string) int {
	depth := 1
	for i := start + 1; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.Count(content[:i], "\n") + 1
			}
		}
	}
	return len(lines)
}

// ExtractDefinitions returns definitions from parsed AST
func (a *GoAdapter) ExtractDefinitions(ast *models.AST) ([]*models.Definition, error) {
	if ast == nil {
		return nil, fmt.Errorf("nil AST provided")
	}
	return ast.Definitions, nil
}

// SelectFramework determines the test framework to use
func (a *GoAdapter) SelectFramework(projectPath string) string {
	// Check go.mod for testify
	goModPath := filepath.Join(projectPath, "go.mod")
	if content, err := os.ReadFile(goModPath); err == nil {
		if strings.Contains(string(content), "github.com/stretchr/testify") {
			return "testify"
		}
	}
	return a.defaultFW
}

// GenerateTestPath returns the expected path for a test file
func (a *GoAdapter) GenerateTestPath(sourcePath string, outputDir string) string {
	// Go tests are in the same directory with _test.go suffix
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	name := strings.TrimSuffix(base, ".go")

	if outputDir != "" {
		dir = outputDir
	}

	return filepath.Join(dir, name+"_test.go")
}

// FormatTestCode formats Go test code using gofmt
func (a *GoAdapter) FormatTestCode(code string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "testgen_*.go")
	if err != nil {
		return code, nil // Return unformatted if can't create temp file
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return code, nil
	}
	tmpFile.Close()

	// Run gofmt
	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9) // 5 seconds
	defer cancel()

	cmd := exec.CommandContext(ctx, "gofmt", "-w", tmpFile.Name())
	if err := cmd.Run(); err != nil {
		return code, nil // Return unformatted if gofmt fails
	}

	// Read formatted content
	formatted, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return code, nil
	}

	return string(formatted), nil
}

// GetPromptTemplate returns the prompt template for Go tests
func (a *GoAdapter) GetPromptTemplate(testType string) string {
	basePrompt := `Generate idiomatic Go tests for the following function.

Requirements:
- Use Go's testing package
- Use testify/assert for assertions
- Follow table-driven test pattern with t.Run() for subtests
- Include meaningful test case names
- Cover happy path, edge cases, and error conditions
- Use t.Helper() for helper functions
- Handle errors explicitly

Function to test:
%s

Package: %s
`

	switch testType {
	case "table-driven":
		return basePrompt + `
Focus on table-driven tests with comprehensive test cases:
- Use a struct slice for test cases
- Include name, input, expected output, and wantErr fields
- Use t.Run() for each test case
- Use testify assert.Equal and require.NoError

Example structure:
` + "```go" + `
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"happy path", validInput, expectedOutput, false},
        {"edge case", edgeInput, edgeOutput, false},
        {"error case", invalidInput, zeroValue, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
` + "```"

	case "edge-cases":
		return basePrompt + `
Focus on edge cases and boundary conditions:
- Nil/empty inputs
- Zero values
- Maximum/minimum values
- Unicode and special characters
- Concurrent access if applicable
`

	case "negative":
		return basePrompt + `
Focus on error handling and negative test cases:
- Invalid inputs that should return errors
- Nil pointer handling
- Out of bounds conditions
- Invalid state scenarios
`

	default: // unit
		return basePrompt + `
Generate comprehensive unit tests covering:
- Happy path scenarios
- Basic edge cases
- Error conditions
`
	}
}

// ValidateTests checks if generated tests compile
func (a *GoAdapter) ValidateTests(testCode string, testPath string) error {
	// Write test file temporarily
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}
	defer os.Remove(testPath)

	// Try to compile
	ctx, cancel := context.WithTimeout(context.Background(), 30*1e9) // 30 seconds
	defer cancel()

	dir := filepath.Dir(testPath)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", "/dev/null", "./...")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %s", string(output))
	}

	return nil
}

// RunTests executes Go tests and returns results
func (a *GoAdapter) RunTests(testDir string) (*models.TestResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*1e9) // 2 minutes
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-cover", "-json", "./...")
	cmd.Dir = testDir

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

	// Parse output for pass/fail counts (simplified)
	outputStr := string(output)
	results.PassedCount = strings.Count(outputStr, `"Action":"pass"`)
	results.FailedCount = strings.Count(outputStr, `"Action":"fail"`)

	// Extract coverage
	coverageRegex := regexp.MustCompile(`coverage:\s+([\d.]+)%`)
	if matches := coverageRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%f", &results.Coverage)
	}

	return results, nil
}
