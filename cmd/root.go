/*
Package cmd implements the CLI commands for TestGen.

This package uses Cobra for command-line parsing and Viper for configuration
management, providing a hierarchical command structure:

  - testgen generate: Generate tests for source files
  - testgen validate: Validate existing tests and coverage
  - testgen analyze: Analyze codebase for cost estimation
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is set at build time via ldflags
	// -ldflags="-X github.com/princepal9120/testgen-cli/cmd.Version=v1.0.0"
	Version = "dev"

	cfgFile string
	verbose bool
	quiet   bool
	logger  *slog.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "testgen",
	Short: "AI-powered test generation for multiple languages",
	Long: `TestGen is an AI-powered CLI tool that automatically generates 
production-ready tests for source code across multiple programming languages.

Supported languages:
  • JavaScript/TypeScript (Jest, Vitest, Mocha)
  • Python (pytest, unittest)
  • Go (testing + testify)
  • Rust (cargo test)

Examples:
  # Generate unit tests for a single file
  testgen generate --file=./src/utils.py --type=unit

  # Generate tests for entire directory recursively
  testgen generate --path=./src --recursive --type=unit,edge-cases

  # Analyze cost before generation
  testgen analyze --path=./src --cost-estimate

  # Validate tests and check coverage
  testgen validate --path=./src --min-coverage=80`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.testgen.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory and home directory
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.testgen")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".testgen")
	}

	// Read environment variables with TESTGEN_ prefix
	viper.SetEnvPrefix("TESTGEN")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Read in config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}

	// Initialize logger
	initLogger()

	return nil
}

// initLogger sets up the structured logger based on verbosity settings
func initLogger() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	if quiet {
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use JSON format if in CI or explicitly requested
	if os.Getenv("CI") != "" || viper.GetBool("log.json") {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, opts))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, opts))
	}

	slog.SetDefault(logger)
}

// GetLogger returns the configured logger
func GetLogger() *slog.Logger {
	if logger == nil {
		initLogger()
	}
	return logger
}
