package adapters

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoAdapter_ParseFile(t *testing.T) {
	adapter := NewGoAdapter()

	t.Run("Parse basic function", func(t *testing.T) {
		code := `
package main

func Add(a, b int) int {
	return a + b
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Equal(t, "main", ast.Package)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "Add", ast.Definitions[0].Name)
	})

	t.Run("Parse method", func(t *testing.T) {
		code := `
package models

func (u *User) GetName() string {
	return u.Name
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)

		def := ast.Definitions[0]
		assert.Equal(t, "GetName", def.Name)
		assert.True(t, def.IsMethod)
		assert.Equal(t, "User", def.ClassName)
	})
}

func TestGoAdapter_GetPromptTemplate(t *testing.T) {
	adapter := NewGoAdapter()

	t.Run("Unit test prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("unit")
		assert.Contains(t, prompt, "Generate idiomatic Go tests")
		assert.Contains(t, prompt, "testing")
	})

	t.Run("Table driven prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("table-driven")
		assert.Contains(t, prompt, "table-driven tests")
		assert.Contains(t, prompt, "struct slice")
	})
}

func TestGoAdapter_GenerateTestPath(t *testing.T) {
	adapter := NewGoAdapter()

	path := adapter.GenerateTestPath("/pkg/utils/math.go", "")
	assert.Contains(t, filepath.ToSlash(path), "math_test.go")

	pathWithOutDir := adapter.GenerateTestPath("/pkg/utils/math.go", "/tests")
	assert.Equal(t, "/tests/math_test.go", filepath.ToSlash(pathWithOutDir))
}
