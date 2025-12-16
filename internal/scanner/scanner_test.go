package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner_IsTestFile(t *testing.T) {
	s := New(Options{})

	tests := []struct {
		path string
		want bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"utils.py", false},
		{"test_utils.py", true},
		{"utils_test.py", true},
		{"app.js", false},
		{"app.test.js", true},
		{"app.spec.js", true},
		{"component.tsx", false},
		{"component.test.tsx", true},
		{"lib.rs", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, s.isTestFile(tt.path))
		})
	}
}

func TestScanner_ShouldInclude(t *testing.T) {
	s := New(Options{
		ExcludePattern: "excluded_*",
		IncludePattern: "included_*",
	})

	// Default hardcoded ignores
	assert.True(t, s.shouldIgnoreDir("node_modules"))
	assert.True(t, s.shouldIgnoreDir(".git"))
	assert.False(t, s.shouldIgnoreDir("src"))

	// Patterns
	assert.False(t, s.shouldInclude("excluded_file.go"))
	assert.True(t, s.shouldInclude("included_file.go"))
	assert.False(t, s.shouldInclude("other_file.go"))
}

func TestScanner_Scan_Integration(t *testing.T) {
	// Create temp dir structure
	tmpDir, err := os.MkdirTemp("", "scanner_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create files
	// - valid.py
	// - ignored.txt
	// - subdir/valid.js
	// - node_modules/ignored.js

	createFile(t, tmpDir, "valid.py")
	createFile(t, tmpDir, "ignored.txt")

	subDir := filepath.Join(tmpDir, "subdir")
	assert.NoError(t, os.Mkdir(subDir, 0755))
	createFile(t, subDir, "valid.js")

	ignoredDir := filepath.Join(tmpDir, "node_modules")
	assert.NoError(t, os.Mkdir(ignoredDir, 0755))
	createFile(t, ignoredDir, "ignored.js")

	// Scan
	s := New(Options{Recursive: true})
	files, err := s.Scan(tmpDir)

	assert.NoError(t, err)
	assert.Len(t, files, 2) // valid.py, subdir/valid.js

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = filepath.Base(f.Path)
	}
	assert.Contains(t, paths, "valid.py")
	assert.Contains(t, paths, "valid.js")
}

func createFile(t *testing.T, dir, name string) {
	err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644)
	assert.NoError(t, err)
}
