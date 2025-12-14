# TestGen CLI Reference

Complete reference for all TestGen commands and options.

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | | Path to config file | `.testgen.yaml` |
| `--verbose` | `-v` | Enable debug output | `false` |
| `--quiet` | `-q` | Suppress non-error output | `false` |

---

## `testgen generate`

Generate tests for source files.

### Usage
```bash
testgen generate [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--path` | `-p` | Source directory | - |
| `--file` | | Single source file | - |
| `--type` | `-t` | Test types (comma-separated) | `unit` |
| `--framework` | `-f` | Target test framework | auto-detect |
| `--output` | `-o` | Output directory | same as source |
| `--recursive` | `-r` | Process recursively | `false` |
| `--parallel` | `-j` | Number of workers | `2` |
| `--dry-run` | | Preview without writing | `false` |
| `--validate` | | Run tests after generation | `false` |
| `--output-format` | | Output format (text/json) | `text` |
| `--include-pattern` | | Glob pattern to include | - |
| `--exclude-pattern` | | Glob pattern to exclude | - |
| `--batch-size` | | API batch size | `5` |
| `--report-usage` | | Generate usage report | `false` |

### Test Types
- `unit` - Basic unit tests
- `edge-cases` - Boundary conditions
- `negative` - Error handling
- `table-driven` - Parameterized tests (Go)
- `integration` - With mocked dependencies

### Examples
```bash
# Single file
testgen generate --file=./src/utils.py

# Directory with multiple test types
testgen generate --path=./src -r --type=unit,edge-cases

# Dry run with JSON output
testgen generate --path=./src -r --dry-run --output-format=json
```

---

## `testgen validate`

Validate existing tests and coverage.

### Usage
```bash
testgen validate [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--path` | `-p` | Directory to validate | `.` |
| `--recursive` | `-r` | Check recursively | `true` |
| `--min-coverage` | | Minimum coverage % | `0` |
| `--fail-on-missing-tests` | | Exit 1 if tests missing | `false` |
| `--report-gaps` | | Show coverage gaps | `false` |
| `--output-format` | | Output format | `text` |

### Examples
```bash
# Basic validation
testgen validate --path=./src

# Enforce 80% coverage
testgen validate --path=./src --min-coverage=80 --fail-on-missing-tests
```

---

## `testgen analyze`

Analyze codebase before generation.

### Usage
```bash
testgen analyze [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--path` | `-p` | Directory to analyze | `.` |
| `--cost-estimate` | | Show estimated API cost | `false` |
| `--detail` | | Detail level | `summary` |
| `--recursive` | `-r` | Analyze recursively | `true` |
| `--output-format` | | Output format | `text` |

### Detail Levels
- `summary` - Total counts
- `per-file` - File-by-file breakdown
- `per-function` - Function-level detail

### Examples
```bash
# Quick cost estimate
testgen analyze --path=./src --cost-estimate

# Detailed per-file analysis
testgen analyze --path=./src --detail=per-file --output-format=json
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error during execution |
| `2` | Validation/coverage failure |
