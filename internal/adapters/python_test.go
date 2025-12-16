package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPythonAdapter_ParseFile(t *testing.T) {
	adapter := NewPythonAdapter()

	t.Run("Parse basic function", func(t *testing.T) {
		code := `
def add(a, b):
    return a + b
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "add", ast.Definitions[0].Name)
		assert.Equal(t, 2, ast.Definitions[0].StartLine)
	})

	t.Run("Parse function with type hints", func(t *testing.T) {
		code := `
def greet(name: str, age: int = 20) -> str:
    return f"Hello {name}"
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)

		def := ast.Definitions[0]
		assert.Equal(t, "greet", def.Name)
		assert.Equal(t, "str", def.ReturnType)
		assert.Len(t, def.Parameters, 2)
		assert.Equal(t, "name", def.Parameters[0].Name)
		assert.Equal(t, "str", def.Parameters[0].Type)
	})

	t.Run("Parse class method", func(t *testing.T) {
		code := `
class Calculator:
    def add(self, a, b):
        return a + b
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)

		// Should find 1 definition (the method)
		assert.Len(t, ast.Definitions, 1)
		def := ast.Definitions[0]
		assert.Equal(t, "add", def.Name)
		assert.True(t, def.IsMethod)
		assert.Equal(t, "Calculator", def.ClassName)
	})
}

func TestPythonAdapter_GetPromptTemplate(t *testing.T) {
	adapter := NewPythonAdapter()

	t.Run("Unit test prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("unit")
		assert.Contains(t, prompt, "Generate idiomatic Python tests")
		assert.Contains(t, prompt, "pytest")
	})

	t.Run("Edge cases prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("edge-cases")
		assert.Contains(t, prompt, "Focus on edge cases")
	})
}

func TestPythonAdapter_GenerateTestPath(t *testing.T) {
	adapter := NewPythonAdapter()

	path := adapter.GenerateTestPath("/src/app/utils.py", "")
	// Expect ../tests/test_utils.py relative to file
	assert.Contains(t, path, "tests/test_utils.py")

	pathWithOutDir := adapter.GenerateTestPath("/src/app/utils.py", "/tmp/tests")
	assert.Equal(t, "/tmp/tests/test_utils.py", pathWithOutDir)
}
