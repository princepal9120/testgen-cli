/*
TestGen - AI-Powered Multi-Language Test Generation CLI

TestGen automatically generates production-ready tests for source code
across JavaScript/TypeScript, Python, Go, and Rust using LLM APIs.

Usage:

	testgen [command] [flags]

Commands:

	generate    Generate tests for source files
	validate    Validate existing tests and coverage
	analyze     Analyze codebase for test generation cost estimation

Copyright 2024 TestGen Authors. Licensed under Apache 2.0.
*/
package main

import (
	"os"

	"github.com/princepal9120/testgen-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
