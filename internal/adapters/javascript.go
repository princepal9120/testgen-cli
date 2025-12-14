package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/testgen/testgen/pkg/models"
)

// JavaScriptAdapter handles JavaScript and TypeScript source files
type JavaScriptAdapter struct {
	BaseAdapter
}

// NewJavaScriptAdapter creates a new JavaScript/TypeScript language adapter
func NewJavaScriptAdapter() *JavaScriptAdapter {
	return &JavaScriptAdapter{
		BaseAdapter: BaseAdapter{
			language:   "javascript",
			frameworks: []string{"jest", "vitest", "mocha"},
			defaultFW:  "jest",
		},
	}
}

// CanHandle returns true if this adapter can handle the file
func (a *JavaScriptAdapter) CanHandle(filePath string) bool {
	lower := strings.ToLower(filePath)
	extensions := []string{".js", ".jsx", ".ts", ".tsx"}
	for _, ext := range extensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// ParseFile parses JavaScript/TypeScript source code
func (a *JavaScriptAdapter) ParseFile(content string) (*models.AST, error) {
	ast := &models.AST{
		Language:    "javascript",
		Definitions: make([]*models.Definition, 0),
		Imports:     make([]string, 0),
	}

	lines := strings.Split(content, "\n")

	// Extract imports
	importRegex := regexp.MustCompile(`(?:import\s+.*\s+from\s+['"]([^'"]+)['"]|require\s*\(\s*['"]([^'"]+)['"]\s*\))`)
	for _, line := range lines {
		if matches := importRegex.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				if match[1] != "" {
					ast.Imports = append(ast.Imports, match[1])
				} else if match[2] != "" {
					ast.Imports = append(ast.Imports, match[2])
				}
			}
		}
	}

	// Extract function definitions
	// Patterns:
	// - function name(params) {}
	// - const/let/var name = function(params) {}
	// - const/let/var name = (params) => {}
	// - async function name(params) {}
	// - export function name(params) {}

	patterns := []*regexp.Regexp{
		// Standard function declaration
		regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`),
		// Arrow function assigned to variable
		regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(([^)]*)\)\s*=>`),
		// Function expression
		regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?function\s*\(([^)]*)\)`),
	}

	// TypeScript-specific: method declarations in classes
	methodPattern := regexp.MustCompile(`^\s+(?:public|private|protected)?\s*(?:async\s+)?(\w+)\s*\(([^)]*)\)`)

	var currentClass string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for class declaration
		classMatch := regexp.MustCompile(`class\s+(\w+)`).FindStringSubmatch(line)
		if classMatch != nil {
			currentClass = classMatch[1]
			continue
		}

		// Check for end of class (simplified)
		if currentClass != "" && trimmed == "}" && !strings.Contains(line, "=>") {
			// This might be end of class
			// For simplicity, we'll reset after a while
		}

		// Try each pattern
		for _, pattern := range patterns {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				def := &models.Definition{
					Name:      matches[1],
					StartLine: i + 1,
					Signature: strings.TrimSpace(line),
				}

				if len(matches) > 2 {
					def.Parameters = parseJSParams(matches[2])
				}

				// Find function end
				def.EndLine = findJSFunctionEnd(lines, i)
				if def.EndLine > def.StartLine {
					bodyLines := lines[def.StartLine-1 : def.EndLine]
					def.Body = strings.Join(bodyLines, "\n")
				}

				ast.Definitions = append(ast.Definitions, def)
				break
			}
		}

		// Check for methods inside classes
		if currentClass != "" {
			if matches := methodPattern.FindStringSubmatch(line); matches != nil {
				def := &models.Definition{
					Name:      matches[1],
					IsMethod:  true,
					ClassName: currentClass,
					StartLine: i + 1,
					Signature: strings.TrimSpace(line),
				}

				if len(matches) > 2 {
					def.Parameters = parseJSParams(matches[2])
				}

				def.EndLine = findJSFunctionEnd(lines, i)
				if def.EndLine > def.StartLine {
					bodyLines := lines[def.StartLine-1 : def.EndLine]
					def.Body = strings.Join(bodyLines, "\n")
				}

				ast.Definitions = append(ast.Definitions, def)
			}
		}
	}

	return ast, nil
}

// parseJSParams parses JavaScript function parameters
func parseJSParams(paramStr string) []models.Param {
	params := make([]models.Param, 0)
	if strings.TrimSpace(paramStr) == "" {
		return params
	}

	// Split by comma, handling default values
	parts := strings.Split(paramStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		param := models.Param{}

		// Handle TypeScript type annotations: name: Type
		if colonIdx := strings.Index(part, ":"); colonIdx > 0 {
			namePart := part[:colonIdx]
			typePart := part[colonIdx+1:]

			// Handle default value
			if eqIdx := strings.Index(namePart, "="); eqIdx > 0 {
				namePart = namePart[:eqIdx]
			}
			if eqIdx := strings.Index(typePart, "="); eqIdx > 0 {
				typePart = typePart[:eqIdx]
			}

			param.Name = strings.TrimSpace(namePart)
			param.Type = strings.TrimSpace(typePart)
		} else {
			// Handle default value
			if eqIdx := strings.Index(part, "="); eqIdx > 0 {
				param.Name = strings.TrimSpace(part[:eqIdx])
			} else {
				param.Name = part
			}
		}

		params = append(params, param)
	}

	return params
}

// findJSFunctionEnd finds where a JavaScript function ends
func findJSFunctionEnd(lines []string, startIdx int) int {
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
func (a *JavaScriptAdapter) ExtractDefinitions(ast *models.AST) ([]*models.Definition, error) {
	if ast == nil {
		return nil, fmt.Errorf("nil AST provided")
	}
	return ast.Definitions, nil
}

// SelectFramework determines the test framework to use
func (a *JavaScriptAdapter) SelectFramework(projectPath string) string {
	// Check package.json
	pkgPath := filepath.Join(projectPath, "package.json")
	if content, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(content, &pkg) == nil {
			// Check devDependencies
			if devDeps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
				if _, hasVitest := devDeps["vitest"]; hasVitest {
					return "vitest"
				}
				if _, hasJest := devDeps["jest"]; hasJest {
					return "jest"
				}
				if _, hasMocha := devDeps["mocha"]; hasMocha {
					return "mocha"
				}
			}
		}
	}

	return a.defaultFW
}

// GenerateTestPath returns the expected path for a test file
func (a *JavaScriptAdapter) GenerateTestPath(sourcePath string, outputDir string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	testDir := outputDir
	if testDir == "" {
		// Use __tests__ directory or same directory
		testsDir := filepath.Join(dir, "__tests__")
		if info, err := os.Stat(testsDir); err == nil && info.IsDir() {
			testDir = testsDir
		} else {
			testDir = dir
		}
	}

	// Keep the same extension for TypeScript
	return filepath.Join(testDir, name+".test"+ext)
}

// FormatTestCode formats JavaScript/TypeScript test code
func (a *JavaScriptAdapter) FormatTestCode(code string) (string, error) {
	// Try prettier
	tmpFile, err := os.CreateTemp("", "testgen_*.js")
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

	cmd := exec.CommandContext(ctx, "npx", "prettier", "--write", tmpFile.Name())
	if err := cmd.Run(); err == nil {
		formatted, err := os.ReadFile(tmpFile.Name())
		if err == nil {
			return string(formatted), nil
		}
	}

	return code, nil
}

// GetPromptTemplate returns the prompt template for JavaScript tests
func (a *JavaScriptAdapter) GetPromptTemplate(testType string) string {
	basePrompt := `Generate idiomatic JavaScript/TypeScript tests using Jest for the following function.

Requirements:
- Use describe/it blocks for test organization
- Use expect() assertions
- Include meaningful test descriptions
- Handle async functions with async/await
- Use jest.mock() for mocking dependencies
- Use it.each() for parameterized tests

Function to test:
%s

Module: %s
`

	switch testType {
	case "edge-cases":
		return basePrompt + `
Focus on edge cases and boundary conditions:
- null/undefined inputs
- Empty strings, arrays, objects
- NaN, Infinity for numbers
- Very large arrays/strings
- Unicode and special characters
`

	case "negative":
		return basePrompt + `
Focus on error handling and negative test cases:
- Invalid inputs that should throw
- Type errors
- Promise rejections
- Network failures (mock)
`

	default: // unit
		return basePrompt + `
Generate comprehensive unit tests covering:
- Happy path scenarios
- Basic edge cases
- Error conditions

Example structure:
` + "```javascript" + `
describe('functionName', () => {
  it('should handle normal input correctly', () => {
    const result = functionName(validInput);
    expect(result).toBe(expectedOutput);
  });

  it.each([
    ['case1', input1, expected1],
    ['case2', input2, expected2],
  ])('%s', (_, input, expected) => {
    expect(functionName(input)).toBe(expected);
  });

  it('should throw for invalid input', () => {
    expect(() => functionName(invalidInput)).toThrow();
  });

  it('should handle async operations', async () => {
    const result = await asyncFunction();
    expect(result).toBeDefined();
  });
});
` + "```"
	}
}

// ValidateTests checks if generated tests have valid syntax
func (a *JavaScriptAdapter) ValidateTests(testCode string, testPath string) error {
	// Write test file
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}
	defer os.Remove(testPath)

	// Use Node to check syntax
	ctx, cancel := context.WithTimeout(context.Background(), 10*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "--check", testPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(output))
	}

	return nil
}

// RunTests executes JavaScript tests and returns results
func (a *JavaScriptAdapter) RunTests(testDir string) (*models.TestResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npx", "jest", "--json", "--testPathPattern", testDir)
	output, err := cmd.CombinedOutput()

	results := &models.TestResults{
		Output:   string(output),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			results.ExitCode = exitErr.ExitCode()
		}
	}

	// Try to parse Jest JSON output
	var jestOutput struct {
		NumPassedTests  int `json:"numPassedTests"`
		NumFailedTests  int `json:"numFailedTests"`
		NumTotalTests   int `json:"numTotalTests"`
	}

	if json.Unmarshal(output, &jestOutput) == nil {
		results.PassedCount = jestOutput.NumPassedTests
		results.FailedCount = jestOutput.NumFailedTests
	}

	return results, nil
}
