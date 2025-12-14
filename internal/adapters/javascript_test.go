package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaScriptAdapter_ParseFile(t *testing.T) {
	adapter := NewJavaScriptAdapter()

	t.Run("Parse standard function", func(t *testing.T) {
		code := `
function add(a, b) {
  return a + b;
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "add", ast.Definitions[0].Name)
	})

	t.Run("Parse arrow function", func(t *testing.T) {
		code := `
const multiply = (a, b) => {
  return a * b;
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "multiply", ast.Definitions[0].Name)
	})

	t.Run("Parse async function", func(t *testing.T) {
		code := `
async function fetchData() {
  return await api.get();
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "fetchData", ast.Definitions[0].Name)
	})
	
	t.Run("Parse class method", func(t *testing.T) {
		code := `
class User {
  constructor(name) {
    this.name = name;
  }
  
  getName() {
    return this.name;
  }
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		// Should find getName (constructor skipping depends on impl, but let's check what we find)
		// Our regex finds methods inside classes
		
		found := false
		for _, def := range ast.Definitions {
			if def.Name == "getName" {
				found = true
				assert.True(t, def.IsMethod)
				assert.Equal(t, "User", def.ClassName)
			}
		}
		assert.True(t, found, "Should find getName method")
	})
}

func TestJavaScriptAdapter_GetPromptTemplate(t *testing.T) {
	adapter := NewJavaScriptAdapter()
	
	prompt := adapter.GetPromptTemplate("unit")
	assert.Contains(t, prompt, "idiomatic JavaScript/TypeScript tests")
	assert.Contains(t, prompt, "Jest")
}

func TestJavaScriptAdapter_GenerateTestPath(t *testing.T) {
	adapter := NewJavaScriptAdapter()
	
	// JS file
	path := adapter.GenerateTestPath("/src/utils.js", "")
	assert.Contains(t, path, "utils.test.js")
	
	// TS file
	pathTS := adapter.GenerateTestPath("/src/components/Button.tsx", "")
	assert.Contains(t, pathTS, "Button.test.tsx")
}
