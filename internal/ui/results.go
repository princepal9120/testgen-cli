package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/princepal9120/testgen-cli/pkg/models"
)

type ResultsModel struct {
	results  []*models.GenerationResult
	cursor   int
	scroll   int
	height   int
	width    int
	expanded map[int]bool
	quitting bool
}

func NewResultsModel(results []*models.GenerationResult) ResultsModel {
	return ResultsModel{
		results:  results,
		expanded: make(map[int]bool),
		height:   24,
	}
}

func (m ResultsModel) Init() tea.Cmd {
	return nil
}

func (m ResultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				visibleLines := m.height - 8 // Adjust for header/footer
				if m.cursor >= m.scroll+visibleLines {
					m.scroll++
				}
			}
		case "enter", " ":
			m.expanded[m.cursor] = !m.expanded[m.cursor]
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m ResultsModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// 1. Header
	success := 0
	failed := 0
	for _, r := range m.results {
		if r.Error != nil {
			failed++
		} else {
			success++
		}
	}

	// Minimalist Header: [ TITLE ] Stats
	title := TitleStyle.Render("TEST RESULTS")
	stats := SubtitleStyle.Render(fmt.Sprintf("%d passed · %d failed", success, failed))
	s.WriteString(fmt.Sprintf("%s  %s\n\n", title, stats))

	// 2. Results List
	visibleLines := m.height - 8
	if visibleLines < 5 {
		visibleLines = 5
	}

	endIdx := m.scroll + visibleLines
	if endIdx > len(m.results) {
		endIdx = len(m.results)
	}

	for i := m.scroll; i < endIdx; i++ {
		r := m.results[i]
		line := m.renderResultLine(r, i)
		s.WriteString(line)
		s.WriteString("\n")

		// Expanded Details
		if m.expanded[i] {
			s.WriteString(m.renderExpanded(r))
			s.WriteString("\n")
		}
	}

	// 3. Footer
	s.WriteString("\n")
	s.WriteString(SubtitleStyle.Render("Press q to quit · Enter to expand"))

	return s.String()
}

func (m ResultsModel) renderResultLine(r *models.GenerationResult, idx int) string {
	// Status Bullet
	bullet := PassStyle.Render("●")
	if r.Error != nil {
		bullet = FailStyle.Render("●")
	}

	// Filename
	name := filepath.Base(r.SourceFile.Path)

	// Layout
	content := fmt.Sprintf("%s  %s", bullet, name)

	// Selection Style
	if idx == m.cursor {
		return SelectedItemStyle.Render(content)
	}
	return ItemStyle.Render(content)
}

func (m ResultsModel) renderExpanded(r *models.GenerationResult) string {
	var s strings.Builder

	if r.Error != nil {
		return DetailStyle.Render(FailStyle.Render("Error: " + r.Error.Error()))
	}

	// Output Path
	if r.TestPath != "" {
		s.WriteString(DetailStyle.Render(fmt.Sprintf("→ %s", filepath.Base(r.TestPath))))
		s.WriteString("\n")
	}

	// Functions List
	if len(r.FunctionsTested) > 0 {
		funcs := strings.Join(r.FunctionsTested, ", ")
		s.WriteString(DetailStyle.Render(fmt.Sprintf("fn: %s", funcs)))
	}

	return s.String()
}

func ShowResults(results []*models.GenerationResult) error {
	if len(results) == 0 {
		return nil
	}
	p := tea.NewProgram(NewResultsModel(results), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
