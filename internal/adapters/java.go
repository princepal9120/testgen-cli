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

// JavaAdapter handles Java source files
type JavaAdapter struct {
	BaseAdapter
}

// NewJavaAdapter creates a new Java language adapter
func NewJavaAdapter() *JavaAdapter {
	return &JavaAdapter{
		BaseAdapter: BaseAdapter{
			language:   "java",
			frameworks: []string{"junit5", "junit4", "testng"},
			defaultFW:  "junit5",
		},
	}
}

// CanHandle returns true if this adapter can handle the file
func (a *JavaAdapter) CanHandle(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".java"
}

// ParseFile parses Java source code
func (a *JavaAdapter) ParseFile(content string) (*models.AST, error) {
	ast := &models.AST{
		Definitions: make([]*models.Definition, 0),
		Language:    "java",
	}

	lines := strings.Split(content, "\n")

	// Extract package name
	packageRe := regexp.MustCompile(`^\s*package\s+([\w.]+)\s*;`)
	for _, line := range lines {
		if match := packageRe.FindStringSubmatch(line); match != nil {
			ast.Package = match[1]
			break
		}
	}

	// Extract imports
	importRe := regexp.MustCompile(`^\s*import\s+(static\s+)?([\w.]+)\s*;`)
	for _, line := range lines {
		if match := importRe.FindStringSubmatch(line); match != nil {
			ast.Imports = append(ast.Imports, match[2])
		}
	}

	// Extract class name (store locally)
	classRe := regexp.MustCompile(`(?:public\s+)?(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	var className string
	for _, line := range lines {
		if match := classRe.FindStringSubmatch(line); match != nil {
			className = match[1]
			break
		}
	}

	// Extract methods
	methodRe := regexp.MustCompile(`^\s*(public|private|protected)?\s*(static)?\s*(\w+(?:<[^>]+>)?)\s+(\w+)\s*\(([^)]*)\)`)

	for i, line := range lines {
		if match := methodRe.FindStringSubmatch(line); match != nil {
			isStatic := match[2] == "static"
			returnType := match[3]
			methodName := match[4]
			paramStr := match[5]

			// Skip constructors (method name equals class name)
			if methodName == className {
				continue
			}

			// Skip main methods
			if methodName == "main" && isStatic {
				continue
			}

			// Parse parameters
			params := parseJavaParams(paramStr)

			// Find method body
			startLine := i + 1
			endLine := findJavaMethodEnd(lines, i)
			body := strings.Join(lines[startLine:endLine], "\n")

			// Build signature
			signature := fmt.Sprintf("%s %s(%s)", returnType, methodName, paramStr)

			def := &models.Definition{
				Name:       methodName,
				Signature:  signature,
				StartLine:  startLine,
				EndLine:    endLine,
				Parameters: params,
				ReturnType: returnType,
				IsMethod:   true,
				ClassName:  className,
				Body:       body,
			}

			ast.Definitions = append(ast.Definitions, def)
		}
	}

	return ast, nil
}

// parseJavaParams parses Java method parameters
func parseJavaParams(paramStr string) []models.Param {
	params := []models.Param{}
	paramStr = strings.TrimSpace(paramStr)

	if paramStr == "" {
		return params
	}

	// Split by comma, handling generics
	parts := splitJavaParams(paramStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split type and name
		tokens := strings.Fields(part)
		if len(tokens) >= 2 {
			paramType := strings.Join(tokens[:len(tokens)-1], " ")
			paramName := tokens[len(tokens)-1]
			params = append(params, models.Param{
				Name: paramName,
				Type: paramType,
			})
		}
	}

	return params
}

// splitJavaParams splits parameter string handling generics
func splitJavaParams(paramStr string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, ch := range paramStr {
		switch ch {
		case '<':
			depth++
			current.WriteRune(ch)
		case '>':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// findJavaMethodEnd finds where a Java method ends
func findJavaMethodEnd(lines []string, startIdx int) int {
	braceCount := 0
	foundOpen := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]

		for _, ch := range line {
			if ch == '{' {
				braceCount++
				foundOpen = true
			} else if ch == '}' {
				braceCount--
				if foundOpen && braceCount == 0 {
					return i + 1
				}
			}
		}
	}

	return len(lines)
}

// ExtractDefinitions returns definitions from parsed AST
func (a *JavaAdapter) ExtractDefinitions(ast *models.AST) ([]*models.Definition, error) {
	return ast.Definitions, nil
}

// SelectFramework determines the test framework to use
func (a *JavaAdapter) SelectFramework(projectPath string) string {
	dir := filepath.Dir(projectPath)

	// Check for pom.xml (Maven)
	pomPath := filepath.Join(dir, "pom.xml")
	if content, err := os.ReadFile(pomPath); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "junit-jupiter") || strings.Contains(contentStr, "junit.jupiter") {
			return "junit5"
		}
		if strings.Contains(contentStr, "testng") {
			return "testng"
		}
		if strings.Contains(contentStr, "junit") {
			return "junit4"
		}
	}

	// Check for build.gradle
	gradlePath := filepath.Join(dir, "build.gradle")
	if content, err := os.ReadFile(gradlePath); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "junit-jupiter") || strings.Contains(contentStr, "useJUnitPlatform") {
			return "junit5"
		}
		if strings.Contains(contentStr, "testng") {
			return "testng"
		}
	}

	// Check in parent directories
	for i := 0; i < 3; i++ {
		dir = filepath.Dir(dir)
		pomPath = filepath.Join(dir, "pom.xml")
		if content, err := os.ReadFile(pomPath); err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "junit-jupiter") {
				return "junit5"
			}
		}
	}

	return a.defaultFW
}

// GenerateTestPath returns the expected path for a test file
func (a *JavaAdapter) GenerateTestPath(sourcePath string, outputDir string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	testName := name + "Test.java"

	if outputDir != "" {
		return filepath.Join(outputDir, testName)
	}

	// Maven/Gradle convention: src/main/java -> src/test/java
	if strings.Contains(dir, filepath.Join("src", "main", "java")) {
		testDir := strings.Replace(dir, filepath.Join("src", "main", "java"), filepath.Join("src", "test", "java"), 1)
		return filepath.Join(testDir, testName)
	}

	return filepath.Join(dir, testName)
}

// FormatTestCode formats Java test code
func (a *JavaAdapter) FormatTestCode(code string) (string, error) {
	// Try google-java-format if available
	cmd := exec.Command("google-java-format", "-")
	cmd.Stdin = strings.NewReader(code)
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	}

	// Basic cleanup if formatter not available
	lines := strings.Split(code, "\n")
	var result strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		result.WriteString(trimmed)
		result.WriteString("\n")
	}

	return result.String(), nil
}

// GetPromptTemplate returns the prompt template for Java tests
func (a *JavaAdapter) GetPromptTemplate(testType string) string {
	basePrompt := `Generate idiomatic Java tests for the following code.

Requirements:
- Use JUnit 5 (Jupiter) framework
- Use @Test annotation for test methods
- Use Assertions class (assertEquals, assertTrue, assertThrows, etc.)
- Follow Java naming conventions: testMethodName_condition_expectedResult
- Include @DisplayName annotations for readability
- Use @BeforeEach for common setup if needed
- Generate meaningful test data
- Handle exceptions properly with assertThrows
- Add comments explaining test purpose

Important:
- Import org.junit.jupiter.api.*
- Import static org.junit.jupiter.api.Assertions.*
- Keep the same package as source class
- Name test class as: {ClassName}Test
- Do NOT include markdown code blocks, return only valid Java code
`

	switch testType {
	case "unit":
		return basePrompt + `
Focus on:
- Testing each public method individually
- Positive test cases (valid inputs)
- Method return values
- State changes
`
	case "edge-cases":
		return basePrompt + `
Focus on:
- Null inputs
- Empty collections
- Boundary values (0, MAX_VALUE, MIN_VALUE)
- Empty strings
- Edge conditions in logic
`
	case "negative":
		return basePrompt + `
Focus on:
- Invalid inputs that should throw exceptions
- assertThrows for expected exceptions
- Null pointer scenarios
- Illegal argument scenarios
- Invalid state transitions
`
	case "integration":
		return basePrompt + `
Focus on:
- Testing component interactions
- Use @ExtendWith for extensions if needed
- Test with real dependencies when safe
- Verify side effects
`
	default:
		return basePrompt
	}
}

// ValidateTests checks if generated tests have valid syntax
func (a *JavaAdapter) ValidateTests(testCode string, testPath string) error {
	// Check for required imports
	if !strings.Contains(testCode, "import org.junit.jupiter") &&
		!strings.Contains(testCode, "import org.junit.") &&
		!strings.Contains(testCode, "import org.testng") {
		return fmt.Errorf("missing JUnit/TestNG imports")
	}

	// Check for test annotations
	if !strings.Contains(testCode, "@Test") {
		return fmt.Errorf("no @Test annotations found")
	}

	// Check for class definition
	if !strings.Contains(testCode, "class ") {
		return fmt.Errorf("no class definition found")
	}

	// Try to compile if javac is available
	tmpFile := filepath.Join(os.TempDir(), "TestValidation.java")
	if err := os.WriteFile(tmpFile, []byte(testCode), 0644); err != nil {
		return nil // Skip validation if we can't write temp file
	}
	defer os.Remove(tmpFile)

	// Check syntax with javac (don't fail if not available)
	cmd := exec.Command("javac", "-d", os.TempDir(), "-sourcepath", os.TempDir(), tmpFile)
	if err := cmd.Run(); err != nil {
		// Check if javac exists
		if _, pathErr := exec.LookPath("javac"); pathErr != nil {
			return nil // javac not available, skip validation
		}
		return fmt.Errorf("Java syntax error: %v", err)
	}

	return nil
}

// RunTests executes Java tests and returns results
func (a *JavaAdapter) RunTests(testDir string) (*models.TestResults, error) {
	results := &models.TestResults{
		Errors: []string{},
	}

	// Try Maven first
	if _, err := os.Stat(filepath.Join(testDir, "pom.xml")); err == nil {
		cmd := exec.CommandContext(context.Background(), "mvn", "test", "-f", testDir)
		output, err := cmd.CombinedOutput()
		results.Output = string(output)
		if err != nil {
			results.FailedCount = 1
			results.Errors = append(results.Errors, string(output))
			return results, nil
		}
		results.PassedCount = 1
		return results, nil
	}

	// Try Gradle
	if _, err := os.Stat(filepath.Join(testDir, "build.gradle")); err == nil {
		cmd := exec.CommandContext(context.Background(), "gradle", "test", "-p", testDir)
		output, err := cmd.CombinedOutput()
		results.Output = string(output)
		if err != nil {
			results.FailedCount = 1
			results.Errors = append(results.Errors, string(output))
			return results, nil
		}
		results.PassedCount = 1
		return results, nil
	}

	// Direct javac + java if no build tool
	return results, fmt.Errorf("no Maven or Gradle build file found")
}

// Ensure interface compliance
var _ LanguageAdapter = (*JavaAdapter)(nil)
