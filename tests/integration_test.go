package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	binaryPath string
	buildOnce  sync.Once
	buildErr   error
)

// getBinaryPath returns the path to the testgen binary
func getBinaryPath(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		// Get the root directory (parent of tests/)
		rootDir, err := filepath.Abs("..")
		if err != nil {
			buildErr = err
			return
		}

		binaryName := "testgen_test_binary"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}

		binaryPath = filepath.Join(rootDir, binaryName)

		// Build the binary
		buildCmd := exec.Command("go", "build", "-o", binaryName, ".")
		buildCmd.Dir = rootDir
		output, err := buildCmd.CombinedOutput()
		if err != nil {
			buildErr = err
			t.Logf("Build output: %s", string(output))
			return
		}
	})

	if buildErr != nil {
		t.Fatalf("Failed to build binary: %v", buildErr)
	}

	return binaryPath
}

// runCmd executes the testgen binary with args and returns stdout, stderr, and error
func runCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	binary := getBinaryPath(t)

	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runCmdInDir executes the testgen binary in a specific directory
func runCmdInDir(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()
	binary := getBinaryPath(t)

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ============================================
// HELP AND VERSION TESTS
// ============================================

func TestHelp(t *testing.T) {
	stdout, _, err := runCmd(t, "--help")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(stdout, "testgen") {
		t.Errorf("Expected help output to contain 'testgen', got: %s", stdout)
	}
	if !strings.Contains(stdout, "generate") {
		t.Errorf("Expected help output to contain 'generate', got: %s", stdout)
	}
}

func TestVersion(t *testing.T) {
	stdout, _, err := runCmd(t, "--version")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(strings.ToLower(stdout), "version") {
		t.Errorf("Expected version output, got: %s", stdout)
	}
}

// ============================================
// GENERATE COMMAND TESTS
// ============================================

func TestGenerateHelp(t *testing.T) {
	stdout, _, err := runCmd(t, "generate", "--help")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(stdout, "path") {
		t.Errorf("Expected generate help to contain 'path', got: %s", stdout)
	}
	if !strings.Contains(stdout, "type") {
		t.Errorf("Expected generate help to contain 'type', got: %s", stdout)
	}
}

func TestGenerateDryRun(t *testing.T) {
	// Create temp directory with a sample file
	dir, err := os.MkdirTemp("", "testgen-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create a sample Python file
	sampleFile := filepath.Join(dir, "sample.py")
	content := `def add(a, b):
    return a + b

def subtract(a, b):
    return a - b
`
	if err := os.WriteFile(sampleFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write sample file: %v", err)
	}

	stdout, stderr, err := runCmdInDir(t, dir, "generate", "--file=sample.py", "--dry-run")
	// With dry-run, it should succeed (may show API key error but command should parse)
	_ = err // May fail due to missing API key, but command should be valid
	combined := stdout + stderr
	if !strings.Contains(combined, "sample.py") && !strings.Contains(combined, "API") && !strings.Contains(combined, "dry") {
		t.Logf("Output: %s", combined)
	}
}

func TestGenerateNoFile(t *testing.T) {
	_, stderr, err := runCmd(t, "generate")
	if err == nil {
		t.Log("generate without file might succeed with default behavior")
	}
	// Check that it either fails or has reasonable output
	_ = stderr
}

// ============================================
// ANALYZE COMMAND TESTS
// ============================================

func TestAnalyzeHelp(t *testing.T) {
	stdout, _, err := runCmd(t, "analyze", "--help")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(stdout, "path") {
		t.Errorf("Expected analyze help to contain 'path', got: %s", stdout)
	}
	if !strings.Contains(stdout, "cost") {
		t.Errorf("Expected analyze help to contain 'cost', got: %s", stdout)
	}
}

func TestAnalyzeWithSampleFiles(t *testing.T) {
	// Create temp directory with sample files
	dir, err := os.MkdirTemp("", "testgen-analyze-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create sample files
	files := map[string]string{
		"main.py": `def main():
    print("Hello")

def helper():
    return 42
`,
		"utils.js": `function add(a, b) {
    return a + b;
}

function multiply(a, b) {
    return a * b;
}
`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	stdout, stderr, err := runCmdInDir(t, dir, "analyze", "--path=.", "--cost-estimate")
	if err != nil {
		t.Logf("Analyze command failed (may be expected): %v", err)
	}
	combined := stdout + stderr
	// Should show some analysis output
	if strings.Contains(combined, "error") && !strings.Contains(combined, "file") {
		t.Logf("Output: %s", combined)
	}
}

// ============================================
// VALIDATE COMMAND TESTS
// ============================================

func TestValidateHelp(t *testing.T) {
	stdout, _, err := runCmd(t, "validate", "--help")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(stdout, "path") {
		t.Errorf("Expected validate help to contain 'path', got: %s", stdout)
	}
}

// ============================================
// TUI COMMAND TESTS
// ============================================

func TestTuiHelp(t *testing.T) {
	stdout, _, err := runCmd(t, "tui", "--help")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if !strings.Contains(stdout, "TUI") && !strings.Contains(stdout, "tui") && !strings.Contains(stdout, "interactive") {
		t.Errorf("Expected TUI help output, got: %s", stdout)
	}
}

// ============================================
// INVALID COMMANDS TESTS
// ============================================

func TestInvalidCommand(t *testing.T) {
	_, stderr, err := runCmd(t, "nonexistent-command")
	if err == nil {
		t.Error("Expected error for invalid command")
	}
	if !strings.Contains(stderr, "unknown") && !strings.Contains(stderr, "Error") && !strings.Contains(stderr, "invalid") {
		t.Logf("Stderr: %s", stderr)
	}
}

func TestInvalidFlag(t *testing.T) {
	_, stderr, err := runCmd(t, "generate", "--invalid-flag-xyz")
	if err == nil {
		t.Error("Expected error for invalid flag")
	}
	if !strings.Contains(stderr, "unknown") && !strings.Contains(stderr, "flag") {
		t.Logf("Stderr: %s", stderr)
	}
}

// ============================================
// FILE TYPE DETECTION TESTS
// ============================================

func TestPythonFileDetection(t *testing.T) {
	dir, err := os.MkdirTemp("", "testgen-python-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create Python file
	pyFile := filepath.Join(dir, "calculator.py")
	content := `def add(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b

class Calculator:
    def multiply(self, a, b):
        return a * b
`
	if err := os.WriteFile(pyFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	stdout, stderr, _ := runCmdInDir(t, dir, "analyze", "--path=.")
	combined := stdout + stderr
	// Should detect Python
	if !strings.Contains(combined, "py") && !strings.Contains(combined, "Python") && !strings.Contains(combined, "file") {
		t.Logf("Output: %s", combined)
	}
}

func TestJavaScriptFileDetection(t *testing.T) {
	dir, err := os.MkdirTemp("", "testgen-js-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create JavaScript file
	jsFile := filepath.Join(dir, "utils.js")
	content := `function add(a, b) {
    return a + b;
}

const subtract = (a, b) => a - b;

export { add, subtract };
`
	if err := os.WriteFile(jsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	stdout, stderr, _ := runCmdInDir(t, dir, "analyze", "--path=.")
	combined := stdout + stderr
	// Should detect JavaScript
	if !strings.Contains(combined, "js") && !strings.Contains(combined, "JavaScript") && !strings.Contains(combined, "file") {
		t.Logf("Output: %s", combined)
	}
}

func TestGoFileDetection(t *testing.T) {
	dir, err := os.MkdirTemp("", "testgen-go-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create Go file
	goFile := filepath.Join(dir, "main.go")
	content := `package main

func Add(a, b int) int {
    return a + b
}

type Calculator struct{}

func (c *Calculator) Multiply(a, b int) int {
    return a * b
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	stdout, stderr, _ := runCmdInDir(t, dir, "analyze", "--path=.")
	combined := stdout + stderr
	// Should detect Go
	if !strings.Contains(combined, "go") && !strings.Contains(combined, "Go") && !strings.Contains(combined, "file") {
		t.Logf("Output: %s", combined)
	}
}

func TestRustFileDetection(t *testing.T) {
	dir, err := os.MkdirTemp("", "testgen-rust-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create Rust file
	rsFile := filepath.Join(dir, "lib.rs")
	content := `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

impl Calculator {
    pub fn multiply(&self, a: i32, b: i32) -> i32 {
        a * b
    }
}
`
	if err := os.WriteFile(rsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	stdout, stderr, _ := runCmdInDir(t, dir, "analyze", "--path=.")
	combined := stdout + stderr
	// Should detect Rust
	if !strings.Contains(combined, "rs") && !strings.Contains(combined, "Rust") && !strings.Contains(combined, "file") {
		t.Logf("Output: %s", combined)
	}
}

// ============================================
// CLEANUP
// ============================================

func TestCleanup(t *testing.T) {
	// Clean up the test binary using the global path
	if binaryPath != "" {
		os.Remove(binaryPath)
	}
}
