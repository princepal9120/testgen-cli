package adapters

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaAdapter_CanHandle(t *testing.T) {
	adapter := NewJavaAdapter()

	t.Run("Handles .java files", func(t *testing.T) {
		assert.True(t, adapter.CanHandle("Calculator.java"))
		assert.True(t, adapter.CanHandle("src/main/java/com/example/Service.java"))
	})

	t.Run("Does not handle other files", func(t *testing.T) {
		assert.False(t, adapter.CanHandle("calculator.py"))
		assert.False(t, adapter.CanHandle("calculator.go"))
		assert.False(t, adapter.CanHandle("calculator.js"))
	})
}

func TestJavaAdapter_ParseFile(t *testing.T) {
	adapter := NewJavaAdapter()

	t.Run("Parse basic class with methods", func(t *testing.T) {
		code := `
package com.example.calculator;

import java.util.List;

public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
    
    public int subtract(int a, int b) {
        return a - b;
    }
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Equal(t, "com.example.calculator", ast.Package)
		assert.Len(t, ast.Definitions, 2)
		assert.Equal(t, "add", ast.Definitions[0].Name)
		assert.Equal(t, "subtract", ast.Definitions[1].Name)
		// ClassName is stored in definitions, not AST
		assert.Equal(t, "Calculator", ast.Definitions[0].ClassName)
	})

	t.Run("Parse method with generics", func(t *testing.T) {
		code := `
package com.example;

public class ListUtils {
    public List<String> filter(List<String> items, String prefix) {
        return items;
    }
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		assert.Len(t, ast.Definitions, 1)
		
		def := ast.Definitions[0]
		assert.Equal(t, "filter", def.Name)
		assert.Equal(t, "List<String>", def.ReturnType)
		assert.Len(t, def.Parameters, 2)
	})

	t.Run("Skip constructor and main", func(t *testing.T) {
		code := `
package com.example;

public class App {
    public App() {
        // constructor
    }
    
    public static void main(String[] args) {
        // main method
    }
    
    public void doWork() {
        // actual method
    }
}
`
		ast, err := adapter.ParseFile(code)
		assert.NoError(t, err)
		// Should only have doWork, not constructor or main
		assert.Len(t, ast.Definitions, 1)
		assert.Equal(t, "doWork", ast.Definitions[0].Name)
	})
}

func TestJavaAdapter_GetPromptTemplate(t *testing.T) {
	adapter := NewJavaAdapter()

	t.Run("Unit test prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("unit")
		assert.Contains(t, prompt, "JUnit 5")
		assert.Contains(t, prompt, "@Test")
		assert.Contains(t, prompt, "Assertions")
	})

	t.Run("Edge cases prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("edge-cases")
		assert.Contains(t, prompt, "Null")
		assert.Contains(t, prompt, "Boundary")
	})

	t.Run("Negative test prompt", func(t *testing.T) {
		prompt := adapter.GetPromptTemplate("negative")
		assert.Contains(t, prompt, "assertThrows")
		assert.Contains(t, prompt, "exception")
	})
}

func TestJavaAdapter_GenerateTestPath(t *testing.T) {
	adapter := NewJavaAdapter()

	t.Run("Without output dir", func(t *testing.T) {
		path := adapter.GenerateTestPath("/pkg/Calculator.java", "")
		assert.Contains(t, filepath.ToSlash(path), "CalculatorTest.java")
	})

	t.Run("With output dir", func(t *testing.T) {
		path := adapter.GenerateTestPath("/pkg/Calculator.java", "/tests")
		assert.Equal(t, "/tests/CalculatorTest.java", filepath.ToSlash(path))
	})

	t.Run("Maven convention", func(t *testing.T) {
		path := adapter.GenerateTestPath("/project/src/main/java/com/example/Service.java", "")
		expected := filepath.ToSlash(path)
		assert.Contains(t, expected, "src/test/java")
		assert.Contains(t, expected, "ServiceTest.java")
	})
}

func TestJavaAdapter_GetLanguage(t *testing.T) {
	adapter := NewJavaAdapter()
	assert.Equal(t, "java", adapter.GetLanguage())
}

func TestJavaAdapter_GetFrameworks(t *testing.T) {
	adapter := NewJavaAdapter()
	
	frameworks := adapter.GetSupportedFrameworks()
	assert.Contains(t, frameworks, "junit5")
	assert.Contains(t, frameworks, "junit4")
	assert.Contains(t, frameworks, "testng")
	
	assert.Equal(t, "junit5", adapter.GetDefaultFramework())
}
