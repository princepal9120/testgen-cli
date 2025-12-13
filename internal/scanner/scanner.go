/*
Package scanner provides file discovery and language detection for TestGen.
*/
package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/testgen/testgen/pkg/models"
)

// Options configures the scanner behavior
type Options struct {
	Recursive      bool
	IncludePattern string
	ExcludePattern string
	IgnoreFile     string // Path to .testgenignore
}

// Scanner discovers and filters source files
type Scanner struct {
	opts          Options
	ignoreRules   []string
	hardcodedDirs []string
}

// SourceFile is an alias for the models.SourceFile for package-local use
type SourceFile = models.SourceFile

// New creates a new Scanner with the given options
func New(opts Options) *Scanner {
	s := &Scanner{
		opts: opts,
		hardcodedDirs: []string{
			"node_modules",
			"venv",
			".venv",
			"vendor",
			"target",
			"__pycache__",
			".git",
			".idea",
			".vscode",
			"dist",
			"build",
			"coverage",
			".pytest_cache",
			".mypy_cache",
		},
	}

	// Load ignore rules
	s.loadIgnoreRules()

	return s
}

// Scan discovers source files in the given path
func (s *Scanner) Scan(rootPath string) ([]*SourceFile, error) {
	var files []*SourceFile

	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, err
	}

	// Single file
	if !info.IsDir() {
		if s.isSourceFile(rootPath) && !s.isTestFile(rootPath) {
			lang := DetectLanguage(rootPath)
			if lang != "" {
				files = append(files, &SourceFile{
					Path:     rootPath,
					Language: lang,
				})
			}
		}
		return files, nil
	}

	// Directory
	if s.opts.Recursive {
		err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Skip ignored directories
			if info.IsDir() {
				if s.shouldIgnoreDir(path) {
					return filepath.SkipDir
				}
				return nil
			}

			// Process files
			if s.shouldInclude(path) {
				if file := s.processFile(path); file != nil {
					files = append(files, file)
				}
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(rootPath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(rootPath, entry.Name())
			if s.shouldInclude(path) {
				if file := s.processFile(path); file != nil {
					files = append(files, file)
				}
			}
		}
	}

	return files, err
}

func (s *Scanner) processFile(path string) *SourceFile {
	if !s.isSourceFile(path) {
		return nil
	}

	if s.isTestFile(path) {
		return nil
	}

	lang := DetectLanguage(path)
	if lang == "" {
		return nil
	}

	return &SourceFile{
		Path:     path,
		Language: lang,
	}
}

func (s *Scanner) loadIgnoreRules() {
	// Try to load .testgenignore from current directory
	ignoreFile := s.opts.IgnoreFile
	if ignoreFile == "" {
		ignoreFile = ".testgenignore"
	}

	file, err := os.Open(ignoreFile)
	if err != nil {
		return // No ignore file, that's OK
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			s.ignoreRules = append(s.ignoreRules, line)
		}
	}
}

func (s *Scanner) shouldIgnoreDir(path string) bool {
	base := filepath.Base(path)

	// Hardcoded ignores
	for _, dir := range s.hardcodedDirs {
		if base == dir {
			return true
		}
	}

	// Custom ignore rules (directory patterns)
	for _, rule := range s.ignoreRules {
		rule = strings.TrimSuffix(rule, "/")
		if matched, _ := filepath.Match(rule, base); matched {
			return true
		}
	}

	return false
}

func (s *Scanner) shouldInclude(path string) bool {
	base := filepath.Base(path)

	// Check exclude pattern
	if s.opts.ExcludePattern != "" {
		if matched, _ := filepath.Match(s.opts.ExcludePattern, base); matched {
			return false
		}
	}

	// Check custom ignore rules
	for _, rule := range s.ignoreRules {
		if matched, _ := filepath.Match(rule, base); matched {
			return false
		}
	}

	// Check include pattern
	if s.opts.IncludePattern != "" {
		if matched, _ := filepath.Match(s.opts.IncludePattern, base); !matched {
			return false
		}
	}

	return true
}

func (s *Scanner) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	sourceExts := []string{
		".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".rs",
	}
	for _, e := range sourceExts {
		if ext == e {
			return true
		}
	}
	return false
}

func (s *Scanner) isTestFile(path string) bool {
	base := filepath.Base(path)
	lower := strings.ToLower(base)

	// Go test files
	if strings.HasSuffix(lower, "_test.go") {
		return true
	}

	// Python test files
	if strings.HasPrefix(lower, "test_") && strings.HasSuffix(lower, ".py") {
		return true
	}
	if strings.HasSuffix(lower, "_test.py") {
		return true
	}

	// JavaScript/TypeScript test files
	if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
		return true
	}

	// Rust test files in tests/ directory
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "tests" && strings.HasSuffix(lower, ".rs") {
		return true
	}

	return false
}
