# TestGen Architecture

## Overview

TestGen follows **Clean Architecture** principles with clear separation between CLI/TUI, business logic, and external services.

```
┌─────────────────────────────────────────────────────┐
│               Presentation Layer                     │
│  ┌─────────────────────┐  ┌─────────────────────┐   │
│  │   CLI (cmd/)        │  │   TUI (internal/    │   │
│  │   Cobra commands    │  │   ui/tui/)          │   │
│  │   flags, user I/O   │  │   Bubble Tea models │   │
│  └──────────┬──────────┘  └──────────┬──────────┘   │
└─────────────┼────────────────────────┼──────────────┘
              │                        │
              └───────────┬────────────┘
                          ▼
┌─────────────────────────────────────────────────────┐
│              Core Engine (internal/generator/)       │
│         Orchestrates adapters, LLM, output          │
└──────────────────────┬──────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
┌───────────┐   ┌───────────┐   ┌───────────┐
│  Scanner  │   │  Adapters │   │    LLM    │
│(internal/)│   │(internal/)│   │(internal/)│
└───────────┘   └───────────┘   └───────────┘
```

---

## Package Responsibilities

### `cmd/`
- Cobra command definitions
- Flag parsing and validation
- Calls `internal/` packages
- **No business logic**

### `internal/ui/tui/`
- Bubble Tea TUI application
- Screen models (Home, Config, Preview, Running, Results)
- State machine for navigation
- Uses lipgloss for styling

### `internal/ui/`
- Shared UI components (spinner, banner, progress)
- Style definitions

### `internal/scanner/`
- File discovery
- Language detection
- Ignore pattern handling

### `internal/adapters/`
- `LanguageAdapter` interface
- Language-specific implementations (Go, Python, JS, Rust)
- Parsing, prompts, formatting

### `internal/llm/`
- `Provider` interface
- Anthropic/OpenAI implementations
- Caching, rate limiting, batching

### `internal/generator/`
- Core orchestration
- Worker pool for parallelism
- Output handling

### `internal/validation/`
- Test compilation checks
- Coverage parsing

### `pkg/models/`
- Shared data structures
- DTOs between packages

---

## Key Interfaces

### LanguageAdapter
```go
type LanguageAdapter interface {
    ParseFile(content string) (*models.AST, error)
    GetPromptTemplate(testType string) string
    GenerateTestPath(sourcePath string, outputDir string) string
    FormatTestCode(code string) (string, error)
    ValidateTests(testCode string, testPath string) error
    RunTests(testDir string) (*models.TestResults, error)
}
```

### LLM Provider
```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    BatchComplete(ctx context.Context, reqs []CompletionRequest) ([]*CompletionResponse, error)
}
```

---

## Data Flow

```
Source File → Scanner → Adapter.Parse → Engine → LLM → Adapter.Format → Output
```

1. **Scanner** discovers source files
2. **Adapter** parses file into AST
3. **Engine** builds prompts using adapter templates
4. **LLM** generates test code
5. **Adapter** formats and validates output
6. **Engine** writes test files

---

## Adding a New Language

1. Create `internal/adapters/<lang>.go`
2. Implement `LanguageAdapter` interface
3. Register in `internal/adapters/registry.go`:
   ```go
   defaultRegistry.Register(NewRubyAdapter())
   ```

No changes needed in CLI, Engine, or LLM layers.
