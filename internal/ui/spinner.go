package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
}

func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = PassStyle
	return SpinnerModel{spinner: s, message: message}
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case spinnerDoneMsg:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m SpinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

type spinnerDoneMsg struct{}

type StatusSpinner struct {
	program *tea.Program
}

func NewStatusSpinner(message string) *StatusSpinner {
	p := tea.NewProgram(NewSpinner(message))
	return &StatusSpinner{program: p}
}

func (s *StatusSpinner) Start() {
	go s.program.Run()
}

func (s *StatusSpinner) Stop() {
	s.program.Send(spinnerDoneMsg{})
	time.Sleep(50 * time.Millisecond)
}

func (s *StatusSpinner) UpdateMessage(msg string) {
	// For future use if we want to update the message dynamically
}
