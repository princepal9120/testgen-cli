package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type HomeModel struct {
	list   list.Model
	choice string
}

func NewHomeModel() HomeModel {
	items := []list.Item{
		menuItem{title: "Generate Tests", desc: "Generate unit tests for source files"},
		menuItem{title: "Analyze Codebase", desc: "Analyze files and estimate costs"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.NormalTitle = itemStyle

	l := list.New(items, delegate, 50, 10)
	l.Title = "⚡ TestGen TUI"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	return HomeModel{list: l}
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				m.choice = item.title
				if m.choice == "Generate Tests" {
					return m, func() tea.Msg {
						return NavigateMsg{To: ScreenGenerateConfig}
					}
				} else if m.choice == "Analyze Codebase" {
					return m, func() tea.Msg {
						return NavigateMsg{To: ScreenAnalyzeConfig}
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m HomeModel) View() string {
	provider, ok := getConfiguredProvider()
	status := errorStyle.Render("✖ No API key configured")
	if ok {
		status = successStyle.Render(fmt.Sprintf("✔ Using %s", provider))
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s",
		m.list.View(),
		status,
		helpStyle.Render("enter: select • q: quit"),
	)
}
