/*
Package config provides configuration management for TestGen.

This package uses Viper for loading configuration from:
- YAML config files (.testgen.yaml)
- Environment variables (TESTGEN_*)
- CLI flags
*/
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the full TestGen configuration
type Config struct {
	LLM        LLMConfig        `mapstructure:"llm"`
	Generation GenerationConfig `mapstructure:"generation"`
	Output     OutputConfig     `mapstructure:"output"`
	Languages  LanguagesConfig  `mapstructure:"languages"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	Model       string  `mapstructure:"model"`
	APIKeyEnv   string  `mapstructure:"api_key_env"`
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

// GenerationConfig contains test generation settings
type GenerationConfig struct {
	BatchSize       int `mapstructure:"batch_size"`
	ParallelWorkers int `mapstructure:"parallel_workers"`
	TimeoutSeconds  int `mapstructure:"timeout_seconds"`
}

// OutputConfig contains output settings
type OutputConfig struct {
	Format          string `mapstructure:"format"`
	IncludeCoverage bool   `mapstructure:"include_coverage"`
}

// LanguagesConfig contains per-language settings
type LanguagesConfig struct {
	JavaScript LanguageSettings `mapstructure:"javascript"`
	Python     LanguageSettings `mapstructure:"python"`
	Go         LanguageSettings `mapstructure:"go"`
	Rust       LanguageSettings `mapstructure:"rust"`
}

// LanguageSettings contains settings for a specific language
type LanguageSettings struct {
	Frameworks       []string `mapstructure:"frameworks"`
	DefaultFramework string   `mapstructure:"default_framework"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:    "anthropic",
			Model:       "claude-3-5-sonnet-20241022",
			APIKeyEnv:   "ANTHROPIC_API_KEY",
			Temperature: 0.3,
			MaxTokens:   4096,
		},
		Generation: GenerationConfig{
			BatchSize:       5,
			ParallelWorkers: 2,
			TimeoutSeconds:  30,
		},
		Output: OutputConfig{
			Format:          "text",
			IncludeCoverage: true,
		},
		Languages: LanguagesConfig{
			JavaScript: LanguageSettings{
				Frameworks:       []string{"jest", "vitest", "mocha"},
				DefaultFramework: "jest",
			},
			Python: LanguageSettings{
				Frameworks:       []string{"pytest", "unittest"},
				DefaultFramework: "pytest",
			},
			Go: LanguageSettings{
				Frameworks:       []string{"testing", "testify"},
				DefaultFramework: "testing",
			},
			Rust: LanguageSettings{
				Frameworks:       []string{"cargo-test"},
				DefaultFramework: "cargo-test",
			},
		},
	}
}

// Load loads configuration from files and environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set defaults in viper
	setDefaults(cfg)

	// Read config file if it exists
	_ = viper.ReadInConfig()

	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults(cfg *Config) {
	viper.SetDefault("llm.provider", cfg.LLM.Provider)
	viper.SetDefault("llm.model", cfg.LLM.Model)
	viper.SetDefault("llm.api_key_env", cfg.LLM.APIKeyEnv)
	viper.SetDefault("llm.temperature", cfg.LLM.Temperature)
	viper.SetDefault("llm.max_tokens", cfg.LLM.MaxTokens)

	viper.SetDefault("generation.batch_size", cfg.Generation.BatchSize)
	viper.SetDefault("generation.parallel_workers", cfg.Generation.ParallelWorkers)
	viper.SetDefault("generation.timeout_seconds", cfg.Generation.TimeoutSeconds)

	viper.SetDefault("output.format", cfg.Output.Format)
	viper.SetDefault("output.include_coverage", cfg.Output.IncludeCoverage)
}

// GetAPIKey retrieves the API key for the configured provider
func GetAPIKey(cfg *Config) string {
	envVar := cfg.LLM.APIKeyEnv
	if envVar == "" {
		switch cfg.LLM.Provider {
		case "anthropic":
			envVar = "ANTHROPIC_API_KEY"
		case "openai":
			envVar = "OPENAI_API_KEY"
		case "gemini":
			envVar = "GEMINI_API_KEY"
		case "groq":
			envVar = "GROQ_API_KEY"
		}
	}
	return os.Getenv(envVar)
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	// Check current directory
	if _, err := os.Stat(".testgen.yaml"); err == nil {
		return ".testgen.yaml"
	}

	// Check home directory
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".testgen", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return ""
}
