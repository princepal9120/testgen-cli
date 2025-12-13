package adapters

import (
	"sync"

	"github.com/testgen/testgen/internal/scanner"
)

// Registry manages language adapters
type Registry struct {
	adapters map[string]LanguageAdapter
	mu       sync.RWMutex
}

var (
	defaultRegistry *Registry
	once            sync.Once
)

// DefaultRegistry returns the singleton registry with all default adapters
func DefaultRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = NewRegistry()
		// Register all built-in adapters
		defaultRegistry.Register(NewGoAdapter())
		defaultRegistry.Register(NewPythonAdapter())
		defaultRegistry.Register(NewJavaScriptAdapter())
		defaultRegistry.Register(NewRustAdapter())
	})
	return defaultRegistry
}

// NewRegistry creates a new empty adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]LanguageAdapter),
	}
}

// Register adds an adapter to the registry
func (r *Registry) Register(adapter LanguageAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.GetLanguage()] = adapter
}

// GetAdapter returns the adapter for a language
func (r *Registry) GetAdapter(language string) LanguageAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize language name
	lang := scanner.NormalizeLanguage(language)

	// TypeScript uses JavaScript adapter
	if lang == scanner.LangTypeScript {
		lang = scanner.LangJavaScript
	}

	return r.adapters[lang]
}

// GetAdapterForFile returns the adapter for a file based on its extension
func (r *Registry) GetAdapterForFile(filePath string) LanguageAdapter {
	lang := scanner.DetectLanguage(filePath)
	if lang == "" {
		return nil
	}
	return r.GetAdapter(lang)
}

// ListLanguages returns all registered languages
func (r *Registry) ListLanguages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	langs := make([]string, 0, len(r.adapters))
	for lang := range r.adapters {
		langs = append(langs, lang)
	}
	return langs
}

// HasAdapter returns true if an adapter exists for the language
func (r *Registry) HasAdapter(language string) bool {
	return r.GetAdapter(language) != nil
}
