package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/princepal9120/testgen-cli/pkg/models"
)

type ResultsModel struct {
	results    []*models.GenerationResult
	analysis   interface{}
	err        error
	mode       string
	focusIndex int
	width      int
	height     int
}

func NewResultsModel() ResultsModel {
	return ResultsModel{}
}

func (m ResultsModel) SetResults(results interface{}, err error) ResultsModel {
	m.mode = "generate"
	m.err = err
	if r, ok := results.([]*models.GenerationResult); ok {
		m.results = r
	}
	return m
}

func (m ResultsModel) SetAnalysis(result interface{}, err error) ResultsModel {
	m.mode = "analyze"
	m.err = err
	m.analysis = result
	return m
}

func (m ResultsModel) Init() tea.Cmd {
	return nil
}

func (m ResultsModel) Update(msg tea.Msg) (ResultsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "enter":
			return m, func() tea.Msg { return NavigateMsg{To: ScreenHome} }

		case "r":
			// Rerun - go back to config
			if m.mode == "generate" {
				return m, func() tea.Msg { return NavigateMsg{To: ScreenGenerateConfig} }
			}
			return m, func() tea.Msg { return NavigateMsg{To: ScreenAnalyzeConfig} }

		case "tab":
			m.focusIndex = (m.focusIndex + 1) % 3
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m ResultsModel) View() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(titleStyle.Render("✖ Error"))
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
		b.WriteString("\n\n")
	} else if m.mode == "generate" {
		b.WriteString(m.generateResultsView())
	} else {
		b.WriteString(m.analyzeResultsView())
	}

	// Actions
	b.WriteString("\n")
	b.WriteString(m.renderButton(0, "Home"))
	b.WriteString("  ")
	b.WriteString(m.renderButton(1, "Rerun"))
	b.WriteString("  ")
	b.WriteString(m.renderButton(2, "Quit"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("r: rerun • q: quit • enter: home"))

	return b.String()
}

func (m ResultsModel) renderButton(idx int, label string) string {
	if m.focusIndex == idx {
		return activeButtonStyle.Render(label)
	}
	return buttonStyle.Render(label)
}

func (m ResultsModel) generateResultsView() string {
	var b strings.Builder

	success := 0
	failed := 0
	var paths []string

	for _, r := range m.results {
		if r.Error != nil {
			failed++
		} else {
			success++
			if r.TestPath != "" {
				paths = append(paths, r.TestPath)
			}
		}
	}

	if failed == 0 {
		b.WriteString(titleStyle.Render("✔ Generation Complete"))
	} else {
		b.WriteString(titleStyle.Render("⚠ Generation Complete (with errors)"))
	}
	b.WriteString("\n\n")

	// Stats box
	stats := fmt.Sprintf(
		"  Files Processed:  %d\n  Tests Generated:  %d\n  Errors:           %d",
		len(m.results), success, failed,
	)
	b.WriteString(boxStyle.Render(stats))
	b.WriteString("\n\n")

	// Generated paths
	if len(paths) > 0 {
		b.WriteString(subtitleStyle.Render("Generated test files:"))
		b.WriteString("\n")
		for _, p := range paths {
			if len(paths) > 5 {
				b.WriteString(fmt.Sprintf("  • %s\n", p))
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", p))
		}
		if len(paths) > 5 {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(paths)-5))
		}
	}

	return b.String()
}

func (m ResultsModel) analyzeResultsView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("✔ Analysis Complete"))
	b.WriteString("\n\n")

	if result, ok := m.analysis.(map[string]interface{}); ok {
		lines := []string{}
		if path, ok := result["path"].(string); ok {
			lines = append(lines, fmt.Sprintf("  Path:         %s", path))
		}
		if count, ok := result["total_files"].(int); ok {
			lines = append(lines, fmt.Sprintf("  Total Files:  %d", count))
		}
		b.WriteString(boxStyle.Render(strings.Join(lines, "\n")))
	}

	return b.String()
}
