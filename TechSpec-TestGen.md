# Technical Specification
## TestGen: AI-Powered Test Generation CLI Tool

---

## 1. System Architecture

### 1.1 High-Level Architecture Overview

TestGen follows a modular, layered architecture designed for extensibility and language-agnostic test generation:

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Layer (User Interface)                │
│  - Command parsing (Cobra)                                   │
│  - Config management (Viper)                                 │
│  - Output formatting (JSON, HTML, TEXT)                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              Orchestration Layer                             │
│  - File discovery and filtering                              │
│  - Language detection                                        │
│  - Test generation pipeline coordination                     │
│  - Parallel batch processing                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│         Language-Specific Adapter Layer                      │
│  - Language detectors                                        │
│  - Framework selectors                                       │
│  - Test generators (per language)                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│            Core Generation Engine                            │
│  - AST parsing (Tree-sitter)                                 │
│  - Prompt template management                                │
│  - LLM orchestration layer                                   │
│  - Output validation & structuring                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              LLM Integration Layer                           │
│  - Provider abstraction (OpenAI, Anthropic)                  │
│  - Token caching & optimization                              │
│  - Request batching & rate limiting                          │
│  - Cost tracking                                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│        Validation & Testing Layer                            │
│  - Compilation validation (language-specific)                │
│  - Test execution via framework runners                      │
│  - Coverage analysis                                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              Storage & Reporting Layer                       │
│  - Test file writing                                         │
│  - Metrics collection                                        │
│  - Report generation                                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. CLI Design

### 2.1 Command Hierarchy

```
testgen
├── generate          # Main test generation command
│   ├── --path        # Source file/directory
│   ├── --type        # Test types (unit, edge-cases, negative, table-driven, integration, regression)
│   ├── --output      # Output directory
│   ├── --framework   # Target framework (auto-detected)
│   ├── --dry-run     # Preview without writing
│   ├── --validate    # Run tests after generation
│   ├── --recursive   # Process directories recursively
│   ├── --parallel    # Worker count for parallel processing
│   └── [other options]
│
├── validate          # Test validation command
│   ├── --path        # Directory to validate
│   ├── --min-coverage
│   ├── --fail-on-missing-tests
│   └── --report-gaps
│
├── analyze           # Cost and coverage analysis
│   ├── --path        # Directory to analyze
│   ├── --cost-estimate
│   ├── --detail      # per-file, per-function, summary
│   └── --recursive
│
└── config            # Configuration management (v1.1)
    ├── set           # Set configuration values
    ├── get           # Get configuration values
    └── list          # List all configuration
```

### 2.2 CLI UX Principles

1. **Progressive Disclosure**: Simple commands for common cases, advanced flags for power users
2. **Sensible Defaults**: Works with `testgen generate --path=.` for standard projects
3. **Clear Feedback**: Real-time progress, detailed error messages, actionable suggestions
4. **Machine-Readable Output**: JSON support for tool integration
5. **Graceful Degradation**: Continues processing on non-critical errors, reports all issues

---

## 3. Technology Stack

### 3.1 Core Language: Go 1.22+

**Rationale:**
- Single binary distribution (no runtime dependencies)
- Excellent CLI performance and startup time
- Strong static typing catches errors early
- Rich ecosystem for DevTools (kubectl, Docker, Hugo use Cobra)
- Cross-platform compilation (Windows, macOS, Linux)

### 3.2 CLI Framework: Cobra v1.8+

**Features Used:**
- Command hierarchies with subcommands
- Automatic help generation
- POSIX-compliant flag parsing
- Persistent flags for parent-child inheritance
- Pre/Post-run hooks for lifecycle management

**Example Integration:**
```go
var generateCmd = &cobra.Command{
    Use:   "generate [path]",
    Short: "Generate tests for source files",
    Long:  `Generate tests for specified files or directories...`,
    RunE:  runGenerate,
}

func init() {
    generateCmd.Flags().StringP("type", "t", "unit", "Test types to generate")
    generateCmd.Flags().StringP("output", "o", "./tests", "Output directory")
    generateCmd.MarkFlagRequired("path")
}
```

### 3.3 Configuration Management: Viper v1.18+

**Features:**
- YAML/JSON config file support (`.testgen.yaml`)
- Environment variable binding (`TESTGEN_*`)
- Config precedence: CLI flags > env vars > config file > defaults
- Automatic config discovery in home directory and project root

**Example Config File (.testgen.yaml):**
```yaml
testgen:
  llm:
    provider: anthropic        # openai or anthropic
    model: claude-3-5-sonnet-20241022
    api_key_env: ANTHROPIC_API_KEY
    temperature: 0.3
    
  generation:
    batch_size: 5
    parallel_workers: 4
    timeout_seconds: 30
    
  output:
    format: text              # text, json, html
    include_coverage: true
    
  languages:
    javascript:
      frameworks: [jest, vitest]
      default_framework: jest
      
    python:
      frameworks: [pytest, unittest]
      default_framework: pytest
      
    go:
      frameworks: [testing]
      test_package: standard
```

### 3.4 Logging: Structured Logging with `slog` (stdlib, Go 1.21+)

**Why Stdlib slog:**
- No external dependencies
- Structured logging with key-value pairs
- Multiple log levels (DEBUG, INFO, WARN, ERROR)
- Easy JSON output for log aggregation

**Configuration:**
```go
import "log/slog"

// Initialize logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Structured logging
logger.Info("test generation started", 
    slog.String("file", "/path/to/file.py"),
    slog.String("language", "python"),
    slog.Int("tokens", 15000),
)
```

### 3.5 AST Parsing: Tree-sitter Bindings

**Rationale:**
- Supports 30+ languages with single parser
- Incremental parsing for performance
- Accurate scope and symbol detection
- Used by GitHub (codespace, semantic search)

**Alternative:** Language-specific AST parsers (astexplorer, ast module) as fallback

### 3.6 Observability

**Metrics Collection:**
- Token usage (input, output, cached)
- API latency per request
- Test generation time per file
- Coverage improvement metrics

**Storage:** Local JSON files in `.testgen/metrics/`
```json
{
  "timestamp": "2025-03-15T14:22:33Z",
  "run_id": "gen-20250315-142233",
  "total_files": 12,
  "tokens_input": 48000,
  "tokens_output": 12000,
  "tokens_cached": 8000,
  "cache_hit_rate": 0.625,
  "total_cost_usd": 0.24,
  "execution_time_seconds": 18,
  "success_count": 12,
  "error_count": 0
}
```

---

## 4. Multi-Language Architecture

### 4.1 Plugin-Based Language Adapter Pattern

TestGen uses an adapter pattern where each language has a dedicated adapter implementing common interfaces:

```go
type LanguageAdapter interface {
    // Detect if this adapter handles the given file
    CanHandle(filePath string) bool
    
    // Parse source code to AST
    ParseFile(content string) (*AST, error)
    
    // Extract function/method definitions
    ExtractDefinitions(ast *AST) ([]*Definition, error)
    
    // Select appropriate test framework
    SelectFramework(ast *AST) string
    
    // Generate test code
    GenerateTests(def *Definition, testType string) (string, error)
    
    // Validate generated tests compile
    ValidateTests(testCode string, testType string) error
    
    // Run generated tests and capture results
    RunTests(testDir string) (*TestResults, error)
}
```

### 4.2 Language Implementations

#### 4.2.1 JavaScript/TypeScript Adapter

```go
type JavaScriptAdapter struct {
    framework string // jest, vitest, mocha
}

func (a *JavaScriptAdapter) SelectFramework(ast *AST) string {
    // Check package.json for hints
    // Default to Jest (most popular)
    return "jest"
}

func (a *JavaScriptAdapter) GenerateTests(def *Definition, testType string) (string, error) {
    // Uses template + LLM to generate Jest test
    // Includes: describe blocks, it() tests, async/await support
    return a.llmClient.Generate(prompts.JestUnitTestTemplate, def), nil
}
```

#### 4.2.2 Python Adapter

```go
type PythonAdapter struct {
    framework string // pytest, unittest
}

func (a *PythonAdapter) GenerateTests(def *Definition, testType string) (string, error) {
    // Uses pytest fixtures, parametrize decorator
    // Fixtures for setup/teardown
    // Parametrize for table-driven tests
    return a.llmClient.Generate(prompts.PytestUnitTestTemplate, def), nil
}

func (a *PythonAdapter) ValidateTests(testCode string, testType string) error {
    // Run: python -m pytest --collect-only
    // Validates syntax and test discovery
    cmd := exec.Command("python", "-m", "pytest", "--collect-only", "-q")
    // ...
}
```

#### 4.2.3 Go Adapter

```go
type GoAdapter struct{}

func (a *GoAdapter) GenerateTests(def *Definition, testType string) (string, error) {
    // For table-driven tests, uses Go idiom:
    // - Struct array of test cases
    // - t.Run() for subtests
    // - testify assertions
    if testType == "table-driven" {
        return a.generateTableDrivenTests(def)
    }
    return a.generateStandardTests(def)
}

func (a *GoAdapter) ValidateTests(testCode string, testType string) error {
    // Run: go test -compile-only
    cmd := exec.Command("go", "test", "-compile", ".")
    // ...
}
```

#### 4.2.4 Rust Adapter

```go
type RustAdapter struct{}

func (a *RustAdapter) GenerateTests(def *Definition, testType string) (string, error) {
    // Generates #[test] modules
    // Handles Result<T, E> patterns
    // Property-based testing (v1.1+)
    return a.llmClient.Generate(prompts.RustUnitTestTemplate, def), nil
}

func (a *RustAdapter) ValidateTests(testCode string, testType string) error {
    // Run: cargo test --no-run --lib
    cmd := exec.Command("cargo", "test", "--no-run", "--lib")
    // ...
}
```

### 4.3 Adapter Registry

```go
type AdapterRegistry struct {
    adapters map[string]LanguageAdapter
}

var registry = &AdapterRegistry{
    adapters: map[string]LanguageAdapter{
        "javascript": &JavaScriptAdapter{},
        "typescript": &JavaScriptAdapter{},
        "python":     &PythonAdapter{},
        "go":         &GoAdapter{},
        "rust":       &RustAdapter{},
    },
}

func (r *AdapterRegistry) GetAdapter(language string) LanguageAdapter {
    return r.adapters[language]
}

func (r *AdapterRegistry) Register(language string, adapter LanguageAdapter) {
    r.adapters[language] = adapter
}
```

---

## 5. File and Folder Scanning

### 5.1 Recursive File Discovery

```go
type FileScanner struct {
    ignorePatterns []string
    includePattern string
    excludePattern string
}

func (s *FileScanner) ScanDirectory(rootPath string, recursive bool) ([]*SourceFile, error) {
    var files []*SourceFile
    
    if recursive {
        err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
            if s.shouldIgnore(path) {
                if info.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
            
            if s.isSourceFile(path) {
                files = append(files, &SourceFile{Path: path, Language: detectLanguage(path)})
            }
            return nil
        })
        return files, err
    }
    
    // Non-recursive: single-level only
    entries, _ := os.ReadDir(rootPath)
    for _, entry := range entries {
        path := filepath.Join(rootPath, entry.Name())
        if !s.shouldIgnore(path) && s.isSourceFile(path) {
            files = append(files, &SourceFile{Path: path, Language: detectLanguage(path)})
        }
    }
    return files, nil
}
```

### 5.2 Ignore Rules (Hierarchical)

TestGen respects ignore patterns in this order:
1. **Global ignore** (hardcoded): `node_modules/`, `venv/`, `.venv/`, `vendor/`, `target/`, `__pycache__/`, `.git/`
2. **Project-level** `.testgenignore` file (similar to `.gitignore`)
3. **Command-line flags**: `--exclude-pattern`, `--include-pattern`

**Example .testgenignore:**
```
# Ignore test files (already tested)
*_test.go
*_test.py
*.spec.js

# Ignore generated code
generated/
dist/
build/

# Ignore third-party
vendor/
node_modules/
```

### 5.3 Large File Handling

Files >1000 LOC are chunked to prevent context window overflow:

```go
type FileChunker struct {
    maxLinesPerChunk int // default: 500
}

func (c *FileChunker) ChunkFile(filePath string, language string) ([]*CodeChunk, error) {
    content, _ := ioutil.ReadFile(filePath)
    lines := strings.Split(string(content), "\n")
    
    var chunks []*CodeChunk
    for i := 0; i < len(lines); i += c.maxLinesPerChunk {
        end := i + c.maxLinesPerChunk
        if end > len(lines) {
            end = len(lines)
        }
        
        chunk := &CodeChunk{
            StartLine: i,
            EndLine:   end,
            Content:   strings.Join(lines[i:end], "\n"),
        }
        chunks = append(chunks, chunk)
    }
    return chunks, nil
}
```

---

## 6. Test Generation Engine

### 6.1 Prompt Template Architecture

Each language and test type has dedicated prompt templates:

```go
type PromptTemplate struct {
    Name       string
    Language   string
    TestType   string
    Template   string  // Contains {{.Placeholder}} variables
    Examples   string  // Few-shot examples
    SystemRole string  // System message for LLM
}

var templates = map[string]*PromptTemplate{
    "go_unit": {
        Language: "go",
        TestType: "unit",
        Template: `Generate a unit test for the following Go function:
        
{{ .FunctionSignature }}

{{ .FunctionBody }}

Requirements:
- Use Go's testing package
- Use testify/assert for assertions
- Follow table-driven test pattern
- Include edge cases
{{ .AdditionalContext }}`,
        
        Examples: `Example:
func TestAdd(t *testing.T) {
    tests := []struct {
        name    string
        a, b    int
        want    int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -2, -3},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, Add(tt.a, tt.b))
        })
    }
}`,
    },
    
    // Similar templates for pytest_unit, jest_unit, etc.
}
```

### 6.2 LLM Prompt Orchestration

```go
type PromptOrchestrator struct {
    cache   *PromptCache           // Semantic deduplication
    batches *RequestBatcher        // Batch multiple requests
    limiter *RateLimiter           // API rate limiting
}

func (o *PromptOrchestrator) GenerateTest(definition *Definition, testType string, language string) (string, error) {
    // Step 1: Check cache for similar definitions
    cacheKey := o.cache.GenerateKey(definition, testType, language)
    if cached, hit := o.cache.Get(cacheKey); hit {
        metrics.RecordCacheHit()
        return cached.(string), nil
    }
    
    // Step 2: Build prompt
    template := templates[fmt.Sprintf("%s_%s", language, testType)]
    prompt := o.buildPrompt(template, definition)
    
    // Step 3: Batch or send immediately
    if o.batches.CanAdd(prompt) {
        result := o.batches.Add(prompt)
        output := <-result.Chan
        o.cache.Set(cacheKey, output)
        return output, nil
    }
    
    // Step 4: Send directly if batching not applicable
    output, err := o.llmClient.Complete(prompt)
    o.cache.Set(cacheKey, output)
    return output, err
}
```

### 6.3 Structured Output Enforcement

Tests must follow a strict JSON structure before post-processing:

```go
type GeneratedTest struct {
    TestName    string   `json:"test_name"`
    TestCode    string   `json:"test_code"`
    Imports     []string `json:"imports"`
    Coverage    []string `json:"coverage_lines"`
    EdgeCases   []string `json:"edge_cases_covered"`
    Dependencies []string `json:"mocked_dependencies"`
}

func (e *Engine) ValidateStructuredOutput(rawOutput string) (*GeneratedTest, error) {
    var test GeneratedTest
    
    // First attempt: Parse as JSON
    if err := json.Unmarshal([]byte(rawOutput), &test); err == nil {
        return &test, nil
    }
    
    // Second attempt: Extract JSON from markdown code blocks
    extracted := extractJSONFromMarkdown(rawOutput)
    if err := json.Unmarshal([]byte(extracted), &test); err == nil {
        return &test, nil
    }
    
    // Third attempt: Fallback parsing if structured output fails
    return e.parseUnstructuredOutput(rawOutput)
}
```

### 6.4 Post-Processing and Formatting

```go
func (e *Engine) PostProcessTest(test *GeneratedTest, language string, framework string) string {
    code := test.TestCode
    
    // Add imports at top
    code = e.addImports(code, test.Imports, language)
    
    // Format according to language conventions
    switch language {
    case "python":
        code = e.formatPython(code)  // black formatter
    case "javascript":
        code = e.formatJavaScript(code)  // prettier
    case "go":
        code = e.formatGo(code)  // gofmt
    case "rust":
        code = e.formatRust(code)  // rustfmt
    }
    
    // Add comments/metadata
    code = e.addMetadata(code, language, test)
    
    return code
}
```

---

## 7. LLM Strategy and Integration

### 7.1 LLM Provider Architecture

TestGen supports multiple LLM providers with a unified interface:

```go
type LLMProvider interface {
    // Configuration
    Configure(apiKey string, model string) error
    
    // Single request
    Complete(prompt string, temperature float32) (string, error)
    
    // Batch requests
    BatchComplete(prompts []string, temperature float32) ([]string, error)
    
    // Stream responses
    CompleteStream(prompt string) (<-chan string, error)
    
    // Token counting
    CountTokens(text string) int
    
    // Usage metrics
    GetUsageMetrics() *UsageMetrics
}
```

### 7.2 Anthropic Claude Integration (Primary)

**Why Claude 3.5 Sonnet:**
- Excellent code understanding (trained on massive code corpus)
- Supports prompt caching (90% cost savings for repeated context)
- 200K token context window (handles large files)
- Structured output via JSON mode
- 50% lower cost vs. GPT-4 for similar quality

```go
type AnthropicProvider struct {
    apiKey      string
    model       string // claude-3-5-sonnet-20241022
    httpClient  *http.Client
}

func (p *AnthropicProvider) Complete(prompt string, temperature float32) (string, error) {
    req := &AnthropicRequest{
        Model: p.model,
        Messages: []AnthropicMessage{
            {
                Role:    "user",
                Content: prompt,
            },
        },
        Temperature: temperature,
        MaxTokens:   2000,
        
        // Prompt caching for repeated prompts
        SystemPrompt: []SystemContent{
            {
                Type:          "text",
                Text:          systemPrompt,
                CacheControl:  &CacheControl{Type: "ephemeral"},
            },
        },
    }
    
    resp, err := p.httpClient.Do(req)
    // Parse response and extract generated text
    return extractText(resp), nil
}
```

### 7.3 OpenAI GPT-4 Fallback

```go
type OpenAIProvider struct {
    apiKey string
    model  string // gpt-4-turbo
}

func (p *OpenAIProvider) Complete(prompt string, temperature float32) (string, error) {
    // Similar implementation to Anthropic
    // Uses OpenAI API format
    return "", nil
}
```

### 7.4 Token Caching Strategy

**Prompt-Level Caching (Anthropic prompt_caching):**
- System prompt cached once: <10 tokens/request
- Repeated function definitions cached: up to 90% savings

**Semantic Caching:**
- Deduplicate similar function patterns
- Store results in local LRU cache (10K entries)
- Hit rate: 50-70% in typical projects

```go
type PromptCache struct {
    cache  *lru.Cache  // Semantic hashing
    prefix string      // Invariant context (framework docs, patterns)
}

func (c *PromptCache) GenerateKey(definition *Definition, testType string) string {
    // Hash: function signature + parameters + return type
    hasher := sha256.New()
    hasher.Write([]byte(definition.Signature))
    hasher.Write([]byte(testType))
    return hex.EncodeToString(hasher.Sum(nil))
}

func (c *PromptCache) Hit(key string) bool {
    _, exists := c.cache.Get(key)
    return exists
}
```

### 7.5 Request Batching

```go
type RequestBatcher struct {
    batchSize  int
    maxWaitMs  int
    pendingReqs []*LLMRequest
}

func (b *RequestBatcher) BatchComplete(requests []*LLMRequest) {
    for i := 0; i < len(requests); i += b.batchSize {
        end := i + b.batchSize
        if end > len(requests) {
            end = len(requests)
        }
        
        batch := requests[i:end]
        prompts := extractPrompts(batch)
        
        // Single API call for multiple prompts
        results, err := b.llmClient.BatchComplete(prompts)
        
        for j, result := range results {
            batch[j].Result = result
        }
    }
}
```

### 7.6 Cost Optimization Metrics

```
Target Cost Reduction: 50-70% vs. naive LLM usage

Strategies:
1. Prompt caching: 90% reduction for repeated patterns
2. Batching: Amortizes overhead across multiple requests
3. Structured output: Reduces output tokens by 40%
4. Chunking: Prevents re-generating context for each chunk
5. Smart model selection: Use smaller models for simple functions

Example:
- Naive single-file generation: 50,000 tokens = $0.50
- Optimized (caching + batching): 15,000 tokens = $0.15
- Cost reduction: 70%
```

---

## 8. Validation Layer

### 8.1 Language-Specific Compilation Checks

```go
type CompilationValidator struct {
    language string
    framework string
}

func (v *CompilationValidator) Validate(testCode string, testPath string) error {
    switch v.language {
    case "go":
        return v.validateGo(testCode, testPath)
    case "python":
        return v.validatePython(testCode, testPath)
    case "javascript":
        return v.validateJavaScript(testCode, testPath)
    case "rust":
        return v.validateRust(testCode, testPath)
    }
    return nil
}

func (v *CompilationValidator) validateGo(testCode string, testPath string) error {
    // Write test to temp file
    tempFile := filepath.Join(os.TempDir(), "test_validation.go")
    ioutil.WriteFile(tempFile, []byte(testCode), 0644)
    defer os.Remove(tempFile)
    
    // Run: go test -compile-only
    cmd := exec.Command("go", "test", "-v", "-compile", ".")
    out, err := cmd.CombinedOutput()
    
    if err != nil {
        return fmt.Errorf("compilation error: %s", string(out))
    }
    return nil
}

func (v *CompilationValidator) validatePython(testCode string, testPath string) error {
    // Run: python -m py_compile
    cmd := exec.Command("python", "-m", "py_compile", testPath)
    _, err := cmd.CombinedOutput()
    return err
}
```

### 8.2 Dry-Run Mode

```go
func (e *Engine) GenerateWithDryRun(files []*SourceFile, dryRun bool) error {
    for _, file := range files {
        testCode, err := e.generateTest(file)
        
        if dryRun {
            // Print to stdout, don't write files
            fmt.Printf("--- %s (generated test) ---\n", file.Path)
            fmt.Println(testCode)
            fmt.Println()
        } else {
            // Write to disk
            err := ioutil.WriteFile(file.TestPath, []byte(testCode), 0644)
            if err != nil {
                e.logger.Error("failed to write test file", slog.String("path", file.TestPath))
            }
        }
    }
    return nil
}
```

### 8.3 Test Execution and Coverage

```go
type TestRunner struct {
    language  string
    framework string
}

func (r *TestRunner) RunAndCapture(testDir string) (*TestResults, error) {
    var cmd *exec.Cmd
    
    switch r.language {
    case "python":
        cmd = exec.Command("pytest", "--cov", "--cov-report=json", testDir)
    case "javascript":
        cmd = exec.Command("npm", "test", "--", "--coverage")
    case "go":
        cmd = exec.Command("go", "test", "-v", "-cover", "./...")
    case "rust":
        cmd = exec.Command("cargo", "test", "--lib")
    }
    
    output, err := cmd.CombinedOutput()
    
    results := &TestResults{
        ExitCode:     cmd.ProcessState.ExitCode(),
        Output:       string(output),
        Coverage:     parseCoverage(string(output), r.language),
        PassedCount:  countPassed(string(output)),
        FailedCount:  countFailed(string(output)),
    }
    
    return results, nil
}
```

---

## 9. Performance and Scalability

### 9.1 Parallel Test Generation

```go
type ParallelGenerator struct {
    workerCount int
    taskQueue   chan *SourceFile
    results     chan *GenerationResult
}

func (g *ParallelGenerator) Generate(files []*SourceFile) ([]*GenerationResult, error) {
    g.taskQueue = make(chan *SourceFile, len(files))
    g.results = make(chan *GenerationResult, len(files))
    
    // Spawn workers
    for i := 0; i < g.workerCount; i++ {
        go g.worker()
    }
    
    // Enqueue tasks
    for _, file := range files {
        g.taskQueue <- file
    }
    close(g.taskQueue)
    
    // Collect results
    var results []*GenerationResult
    for i := 0; i < len(files); i++ {
        results = append(results, <-g.results)
    }
    
    return results, nil
}

func (g *ParallelGenerator) worker() {
    for file := range g.taskQueue {
        testCode, err := g.generateTest(file)
        g.results <- &GenerationResult{
            File:     file,
            TestCode: testCode,
            Error:    err,
        }
    }
}
```

### 9.2 Rate Limiting

```go
type RateLimiter struct {
    requestsPerMinute int
    limiter           *time.Ticker
}

func (l *RateLimiter) WaitForSlot() {
    <-l.limiter.C
}

// Usage in batch processing
for _, prompt := range prompts {
    l.WaitForSlot()
    results = append(results, callLLM(prompt))
}
```

### 9.3 Resource Management

- **Memory**: Streamed processing to avoid loading entire files into memory
- **Disk I/O**: Batch writes, temporary file cleanup
- **Network**: Connection pooling, keep-alive for HTTP clients

---

## 10. Security and Privacy

### 10.1 Source Code Handling

**Default Behavior (API-based):**
- Code transmitted only to configured LLM provider (OpenAI/Anthropic)
- HTTPS/TLS for all API communications
- No code storage or logging on TestGen infrastructure

**Local LLM Option (v1.1+):**
```go
type LocalLLMProvider struct {
    endpoint string // http://localhost:8000 (Ollama, LM Studio)
}

// All processing stays on local machine
```

### 10.2 API Key Management

```go
type CredentialManager struct {
    keyring keyring.Keyring
}

func (cm *CredentialManager) StoreAPIKey(provider string, apiKey string) error {
    // Store in OS keyring (Keychain/Credential Manager/Secret Service)
    return cm.keyring.Set(
        service="testgen",
        user=provider,
        passwd=apiKey,
    )
}

func (cm *CredentialManager) GetAPIKey(provider string) (string, error) {
    return cm.keyring.Get(service="testgen", user=provider)
}
```

### 10.3 Redaction for Logging

```go
func (l *Logger) SanitizeForLogging(text string) string {
    // Redact API keys, credentials
    text = redactPattern(text, `api[_-]key[=:]\s*["']?([^\s"']+)`, "***")
    text = redactPattern(text, `password[=:]\s*["']?([^\s"']+)`, "***")
    return text
}
```

---

## 11. Best Practices and Design Principles

### 11.1 Clean Architecture

```
testgen/
├── cmd/
│   ├── root.go          # CLI entry point
│   ├── generate.go      # generate command
│   ├── validate.go      # validate command
│   └── analyze.go       # analyze command
├── internal/
│   ├── generator/       # Core generation logic
│   │   ├── engine.go
│   │   └── orchestrator.go
│   ├── adapters/        # Language adapters
│   │   ├── golang.go
│   │   ├── python.go
│   │   ├── javascript.go
│   │   └── rust.go
│   ├── llm/             # LLM integration
│   │   ├── provider.go
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   └── cache.go
│   ├── validation/      # Test validation
│   │   └── validator.go
│   ├── scanner/         # File scanning
│   │   └── scanner.go
│   └── config/          # Configuration
│       └── config.go
└── pkg/
    ├── models/          # Data structures (external API)
    └── util/            # Utilities
```

### 11.2 SOLID Principles

**Single Responsibility:** Each adapter handles one language; each provider handles one LLM API

**Open/Closed:** New language support by implementing LanguageAdapter interface

**Liskov Substitution:** All providers implement LLMProvider interface identically

**Interface Segregation:** Small focused interfaces (Scanner, Validator, Generator)

**Dependency Inversion:** Core logic depends on interfaces, not concrete implementations

### 11.3 Error Handling

```go
// Custom error types for clarity
type GenerationError struct {
    Code    string
    Message string
    File    string
    Line    int
    Err     error
}

func (e *GenerationError) Error() string {
    return fmt.Sprintf("[%s] %s (file: %s, line: %d)", e.Code, e.Message, e.File, e.Line)
}

// Usage
if err != nil {
    return &GenerationError{
        Code:    "PARSE_ERROR",
        Message: "Failed to parse AST",
        File:    filePath,
        Err:     err,
    }
}
```

### 11.4 Testing the Tool Itself

TestGen includes comprehensive tests:

```go
// cmd/generate_test.go
func TestGenerateCommand(t *testing.T) {
    // Test with mock files
    tmpDir := t.TempDir()
    createMockFile(tmpDir, "math.py", pythonCode)
    
    // Run generation
    cmd := rootCmd
    cmd.SetArgs([]string{"generate", "--path", tmpDir})
    err := cmd.Execute()
    
    assert.NoError(t, err)
    assert.FileExists(t, filepath.Join(tmpDir, "test_math.py"))
}
```

---

## 12. Open-Source and Community Strategy

### 12.1 Repository Structure

```
testgen/
├── README.md                    # Project overview
├── CONTRIBUTING.md              # Contribution guidelines
├── LICENSE                      # Apache 2.0
├── CODE_OF_CONDUCT.md          # Community standards
├── docs/
│   ├── architecture.md          # This document
│   ├── cli-reference.md
│   ├── contributing.md
│   └── language-adapters.md
├── examples/
│   ├── javascript/
│   ├── python/
│   ├── go/
│   └── rust/
├── cmd/
├── internal/
├── pkg/
├── tests/
└── Makefile
```

### 12.2 Contribution Guidelines

- Language-specific adapter PRs welcome (new language support)
- Bug reports with reproducible examples
- Feature requests with clear use cases
- Code review standards: 2 approvals, passing CI/CD

### 12.3 Licensing

**Apache 2.0** chosen for:
- Commercial-friendly (allows proprietary derivatives)
- Strong patent protection
- Wide adoption in DevTools ecosystem (Kubernetes, Docker, etc.)

### 12.4 Version Strategy

Semantic versioning:
- **v1.0.0**: Initial release (4-language core)
- **v1.1.0**: IDE plugins + Java support
- **v2.0.0**: SaaS platform + advanced features

---

## 13. Deployment and Distribution

### 13.1 Build and Release

**Supported Platforms:**
- Linux (x86_64, ARM64)
- macOS (x86_64, ARM64)
- Windows (x86_64)

**Distribution:**
- GitHub Releases (binary downloads)
- Package managers: homebrew (macOS), apt/yum (Linux), chocolatey (Windows)
- Docker image: `testgen:latest`
- Language package managers: pip (Python), npm (Node.js), crates.io (Rust)

### 13.2 Installation

```bash
# macOS
brew install testgen

# Linux
curl -sSL https://install.testgen.dev | bash

# Docker
docker pull testgen:latest
docker run --rm testgen generate --path=/code

# From source
go install github.com/testgen/testgen@latest
```

---

## 14. Monitoring and Observability

### 14.1 Metrics Exported

- Token usage per run
- API latency percentiles (p50, p95, p99)
- Test generation success rate
- Cache hit rate
- Cost tracking (USD)

### 14.2 Logging Integration

Structured JSON logs compatible with:
- ELK Stack (Elasticsearch)
- Datadog
- CloudWatch
- Splunk

---

## Document Control

| Item | Value |
|------|-------|
| **Document Version** | 1.0 |
| **Last Updated** | 2025-03-15 |
| **Owner** | Engineering Lead |
| **Status** | Approved for Implementation |
| **Next Review** | 2025-06-15 |

---
