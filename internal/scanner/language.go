package scanner

import (
	"path/filepath"
	"strings"
)

// Language constants
const (
	LangGo         = "go"
	LangPython     = "python"
	LangJavaScript = "javascript"
	LangTypeScript = "typescript"
	LangRust       = "rust"
	LangJava       = "java"
)

// extensionMap maps file extensions to languages
var extensionMap = map[string]string{
	".go":   LangGo,
	".py":   LangPython,
	".js":   LangJavaScript,
	".jsx":  LangJavaScript,
	".ts":   LangTypeScript,
	".tsx":  LangTypeScript,
	".rs":   LangRust,
	".java": LangJava,
}

// DetectLanguage determines the programming language from a file path
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	return extensionMap[ext]
}

// IsJavaScriptFamily returns true if the language is JS or TS
func IsJavaScriptFamily(lang string) bool {
	return lang == LangJavaScript || lang == LangTypeScript
}

// GetSupportedExtensions returns all supported file extensions
func GetSupportedExtensions() []string {
	exts := make([]string, 0, len(extensionMap))
	for ext := range extensionMap {
		exts = append(exts, ext)
	}
	return exts
}

// GetLanguagesForExtension returns languages that use the given extension
func GetLanguagesForExtension(ext string) []string {
	if lang, ok := extensionMap[strings.ToLower(ext)]; ok {
		return []string{lang}
	}
	return nil
}

// NormalizeLanguage converts language aliases to standard names
func NormalizeLanguage(lang string) string {
	lower := strings.ToLower(lang)
	switch lower {
	case "golang":
		return LangGo
	case "py", "python3":
		return LangPython
	case "js", "node", "nodejs":
		return LangJavaScript
	case "ts":
		return LangTypeScript
	case "rs":
		return LangRust
	case "jdk", "openjdk", "jvm":
		return LangJava
	default:
		return lower
	}
}
