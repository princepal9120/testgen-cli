package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRustAdapter_ParseFile(t *testing.T) {
	adapter := NewRustAdapter()

	t.Run("Parse basic function", func(t *testing.T) {
		code := `
fn add(a: i32, b: i32) -> i32 {
    a + b
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "add", ast.Definitions[0].Name)
	})

	t.Run("Parse public async function", func(t *testing.T) {
		code := `
pub async fn fetch_data() -> Result<String, Error> {
    Ok("data".to_string())
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)

		def := ast.Definitions[0]
		assert.Equal(t, "fetch_data", def.Name)
		assert.Contains(t, def.Signature, "async")
		assert.Contains(t, def.Signature, "pub")
	})

	t.Run("Parse impl method", func(t *testing.T) {
		code := `
impl User {
    pub fn new(name: String) -> Self {
        User { name }
    }
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)

		def := ast.Definitions[0]
		assert.Equal(t, "new", def.Name)
		assert.True(t, def.IsMethod)
		assert.Equal(t, "User", def.ClassName)
	})
}

func TestRustAdapter_GetPromptTemplate(t *testing.T) {
	adapter := NewRustAdapter()

	prompt := adapter.GetPromptTemplate("unit")
	assert.Contains(t, prompt, "idiomatic Rust tests")
	assert.Contains(t, prompt, "#[cfg(test)]")
}

func TestRustAdapter_GenerateTestPath(t *testing.T) {
	adapter := NewRustAdapter()

	// Inline tests (default behavior if no tests dir)
	path := adapter.GenerateTestPath("/src/lib.rs", "")
	assert.Contains(t, path, "lib.rs.test") // This is our fallback

	// Explicit output dir
	pathWithDir := adapter.GenerateTestPath("/src/lib.rs", "/tests")
	assert.Equal(t, "/tests/lib_test.rs", pathWithDir)
}
