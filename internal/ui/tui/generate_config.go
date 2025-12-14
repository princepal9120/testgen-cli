package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type GenerateConfigModel struct {
	focusIndex int
	inputs     []textinput.Model
	booleans   map[string]bool
	types      []string
	width      int
	height     int
}

const (
	genPathIdx = iota
	genParallelIdx
)

var testTypes = []string{"unit", "edge-cases", "negative", "table-driven", "integration"}

func NewGenerateConfigModel() GenerateConfigModel {
	m := GenerateConfigModel{
		booleans: map[string]bool{
			"recursive": false,
			"dry-run":   false,
			"validate":  false,
		},
		types: []string{"unit"},
	}

	pathInput := textinput.New()
	pathInput.Placeholder = "./src or ./file.py"
	pathInput.Focus()
	pathInput.Width = 40
	pathInput.Prompt = "› "

	parallelInput := textinput.New()
	parallelInput.Placeholder = "2"
	parallelInput.Width = 10
	parallelInput.Prompt = "› "

	m.inputs = []textinput.Model{pathInput, parallelInput}

	return m
}

func (m GenerateConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m GenerateConfigModel) Update(msg tea.Msg) (GenerateConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return NavigateMsg{To: ScreenHome} }

		case "tab", "down":
			m.focusIndex++
			if m.focusIndex > 6 {
				m.focusIndex = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 6
			}
			return m, m.updateFocus()

		case "enter":
			if m.focusIndex == 6 { // Confirm button
				config := m.buildConfig()
				return m, func() tea.Msg {
					return NavigateMsg{To: ScreenPreview, Config: &config}
				}
			}

		case " ":
			// Toggle booleans or types
			switch m.focusIndex {
			case 2: // recursive
				m.booleans["recursive"] = !m.booleans["recursive"]
			case 3: // dry-run
				m.booleans["dry-run"] = !m.booleans["dry-run"]
			case 4: // validate
				m.booleans["validate"] = !m.booleans["validate"]
			case 5: // types - cycle through
				m.cycleTypes()
			}

		case "1", "2", "3", "4", "5":
			if m.focusIndex == 5 {
				idx, _ := strconv.Atoi(msg.String())
				idx--
				if idx >= 0 && idx < len(testTypes) {
					m.toggleType(testTypes[idx])
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update focused input
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *GenerateConfigModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *GenerateConfigModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *GenerateConfigModel) cycleTypes() {
	// Simple cycle: add next type or reset
	if len(m.types) >= len(testTypes) {
		m.types = []string{"unit"}
	} else {
		for _, t := range testTypes {
			if !m.hasType(t) {
				m.types = append(m.types, t)
				break
			}
		}
	}
}

func (m *GenerateConfigModel) toggleType(t string) {
	if m.hasType(t) {
		// Remove
		newTypes := []string{}
		for _, existing := range m.types {
			if existing != t {
				newTypes = append(newTypes, existing)
			}
		}
		if len(newTypes) == 0 {
			newTypes = []string{"unit"}
		}
		m.types = newTypes
	} else {
		m.types = append(m.types, t)
	}
}

func (m GenerateConfigModel) hasType(t string) bool {
	for _, existing := range m.types {
		if existing == t {
			return true
		}
	}
	return false
}

func (m GenerateConfigModel) buildConfig() RunConfig {
	parallel := 2
	if p, err := strconv.Atoi(m.inputs[genParallelIdx].Value()); err == nil && p > 0 {
		parallel = p
	}

	return RunConfig{
		Mode:      "generate",
		Path:      m.inputs[genPathIdx].Value(),
		Recursive: m.booleans["recursive"],
		Types:     m.types,
		DryRun:    m.booleans["dry-run"],
		Validate:  m.booleans["validate"],
		Parallel:  parallel,
	}
}

func (m GenerateConfigModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚡ Generate Tests"))
	b.WriteString("\n\n")

	// Path input
	b.WriteString(m.renderField(0, "Path", m.inputs[genPathIdx].View()))

	// Parallel input
	b.WriteString(m.renderField(1, "Parallel", m.inputs[genParallelIdx].View()))

	// Booleans
	b.WriteString(m.renderBool(2, "Recursive", m.booleans["recursive"]))
	b.WriteString(m.renderBool(3, "Dry Run", m.booleans["dry-run"]))
	b.WriteString(m.renderBool(4, "Validate", m.booleans["validate"]))

	// Types
	typesStr := strings.Join(m.types, ", ")
	focused := m.focusIndex == 5
	style := labelStyle
	if focused {
		style = focusedInputStyle
	}
	b.WriteString(fmt.Sprintf("%s %s\n", style.Render("Test Types:"), typesStr))
	if focused {
		b.WriteString(infoStyle.Render("  Press 1-5 to toggle: unit, edge-cases, negative, table-driven, integration\n"))
	}

	b.WriteString("\n")

	// Confirm button
	btn := buttonStyle.Render("Continue →")
	if m.focusIndex == 6 {
		btn = activeButtonStyle.Render("Continue →")
	}
	b.WriteString(btn)

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("tab: next • space: toggle • enter: confirm • esc: back"))

	return b.String()
}

func (m GenerateConfigModel) renderField(idx int, label, value string) string {
	style := labelStyle
	if m.focusIndex == idx {
		style = focusedInputStyle
	}
	return fmt.Sprintf("%s %s\n", style.Render(label+":"), value)
}

func (m GenerateConfigModel) renderBool(idx int, label string, value bool) string {
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
