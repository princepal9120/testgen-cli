package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AnalyzeConfigModel struct {
	focusIndex int
	pathInput  textinput.Model
	costEst    bool
	recursive  bool
	detail     string
	width      int
	height     int
}

func NewAnalyzeConfigModel() AnalyzeConfigModel {
	pathInput := textinput.New()
	pathInput.Placeholder = "./src"
	pathInput.Focus()
	pathInput.Width = 40
	pathInput.Prompt = "› "

	return AnalyzeConfigModel{
		pathInput: pathInput,
		costEst:   true,
		recursive: true,
		detail:    "summary",
	}
}

func (m AnalyzeConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AnalyzeConfigModel) Update(msg tea.Msg) (AnalyzeConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return NavigateMsg{To: ScreenHome} }

		case "tab", "down":
			m.focusIndex++
			if m.focusIndex > 4 {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 4
			}
			return m, m.updateFocus()

		case "enter":
			if m.focusIndex == 4 { // Confirm button
				config := m.buildConfig()
				return m, func() tea.Msg {
					return NavigateMsg{To: ScreenPreview, Config: &config}
				}
			}

		case " ":
			switch m.focusIndex {
			case 1: // cost estimate
				m.costEst = !m.costEst
			case 2: // recursive
				m.recursive = !m.recursive
			case 3: // detail level
				m.cycleDetail()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update path input
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m *AnalyzeConfigModel) updateFocus() tea.Cmd {
	if m.focusIndex == 0 {
		return m.pathInput.Focus()
	}
	m.pathInput.Blur()
	return nil
}

func (m *AnalyzeConfigModel) cycleDetail() {
	switch m.detail {
	case "summary":
		m.detail = "per-file"
	case "per-file":
		m.detail = "per-function"
	default:
		m.detail = "summary"
	}
}

func (m AnalyzeConfigModel) buildConfig() RunConfig {
	return RunConfig{
		Mode:      "analyze",
		Path:      m.pathInput.Value(),
		Recursive: m.recursive,
		CostEst:   m.costEst,
		Detail:    m.detail,
	}
}

func (m AnalyzeConfigModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📊 Analyze Codebase"))
	b.WriteString("\n\n")

	// Path input
	label := labelStyle.Render("Path:")
	if m.focusIndex == 0 {
		label = focusedInputStyle.Render("Path:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", label, m.pathInput.View()))

	// Booleans
	b.WriteString(m.renderBool(1, "Cost Estimate", m.costEst))
	b.WriteString(m.renderBool(2, "Recursive", m.recursive))

	// Detail level
	label = labelStyle.Render("Detail Level:")
	if m.focusIndex == 3 {
		label = focusedInputStyle.Render("Detail Level:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", label, m.detail))

	b.WriteString("\n")

	// Confirm button
	btn := buttonStyle.Render("Continue →")
	if m.focusIndex == 4 {
		btn = activeButtonStyle.Render("Continue →")
	}
	b.WriteString(btn)

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("tab: next • space: toggle • enter: confirm • esc: back"))

	return b.String()
}

func (m AnalyzeConfigModel) renderBool(idx int, label string, value bool) string {
	style := labelStyle
	if m.focusIndex == idx {
		style = focusedInputStyle
	}
	check := "[ ]"
	if value {
		check = "[✓]"
	}
	return fmt.Sprintf("%s %s\n", style.Render(label+":"), check)
}
