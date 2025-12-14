package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	successBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 3).
			Margin(1, 0)

	errorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 3).
			Margin(1, 0)

	successCheck = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	errorMark = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	statLabel = lipgloss.NewStyle().
			Foreground(ColorSub)

	statValue = lipgloss.NewStyle().
			Foreground(ColorFg).
			Bold(true)
)

type SuccessStats struct {
	FilesProcessed int
	TestsGenerated int
	FunctionsFound int
}

func ShowSuccess(stats SuccessStats) {
	var s strings.Builder

	check := successCheck.Render("✔")
	title := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true).
		Render("SUCCESS")

	s.WriteString(fmt.Sprintf("  %s %s\n\n", check, title))

	s.WriteString(fmt.Sprintf("  %s %s\n",
		statValue.Render(fmt.Sprintf("%d", stats.FilesProcessed)),
		statLabel.Render("files processed")))

	s.WriteString(fmt.Sprintf("  %s %s\n",
		statValue.Render(fmt.Sprintf("%d", stats.TestsGenerated)),
		statLabel.Render("test files generated")))

	s.WriteString(fmt.Sprintf("  %s %s\n",
		statValue.Render(fmt.Sprintf("%d", stats.FunctionsFound)),
		statLabel.Render("functions tested")))

	fmt.Println(successBox.Render(s.String()))
}

func ShowError(message string, details string) {
	var s strings.Builder

	mark := errorMark.Render("✖")
	title := lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true).
		Render("ERROR")

	s.WriteString(fmt.Sprintf("  %s %s\n\n", mark, title))
	s.WriteString(fmt.Sprintf("  %s\n", message))

	if details != "" {
		s.WriteString(fmt.Sprintf("\n  %s\n", statLabel.Render(details)))
	}

	fmt.Println(errorBox.Render(s.String()))
}

func ShowSimpleSuccess(message string) {
	check := successCheck.Render("✔")
	fmt.Printf("\n  %s %s\n\n", check, PassStyle.Render(message))
}

func ShowSimpleError(message string) {
	mark := errorMark.Render("✖")
	fmt.Printf("\n  %s %s\n\n", mark, FailStyle.Render(message))
}
