package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type progressMsg float64
type doneMsg struct{}

type ProgressModel struct {
	spinner  spinner.Model
	progress progress.Model
	message  string
	percent  float64
	done     bool
	width    int
}

func NewProgressModel(message string) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return ProgressModel{
		spinner:  s,
		progress: p,
		message:  message,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = msg.Width - 10
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progressMsg:
		m.percent = float64(msg)
		if m.percent >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case doneMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m ProgressModel) View() string {
	if m.done {
		return ""
	}

	var s strings.Builder

	spin := m.spinner.View()
	s.WriteString(fmt.Sprintf("\n  %s %s\n\n", spin, InfoStyle.Render(m.message)))

	if m.percent > 0 {
		s.WriteString(fmt.Sprintf("  %s\n", m.progress.ViewAs(m.percent)))
	}

	return s.String()
}

// ProgressTracker manages a progress display
type ProgressTracker struct {
	program *tea.Program
	total   int
	current int
}

func NewProgressTracker(message string, total int) *ProgressTracker {
	model := NewProgressModel(message)
	p := tea.NewProgram(model)

	return &ProgressTracker{
		program: p,
		total:   total,
	}
}

func (t *ProgressTracker) Start() {
	go t.program.Run()
	time.Sleep(50 * time.Millisecond)
}

func (t *ProgressTracker) Increment() {
	t.current++
	if t.total > 0 {
		t.program.Send(progressMsg(float64(t.current) / float64(t.total)))
	}
}

func (t *ProgressTracker) Done() {
	t.program.Send(doneMsg{})
	time.Sleep(50 * time.Millisecond)
}

// ShowAPIKeyError displays a helpful error when API key is missing
func ShowAPIKeyError(provider string) {
	var s strings.Builder

	mark := errorMark.Render("✖")
	title := lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true).
		Render("API KEY NOT CONFIGURED")

	s.WriteString(fmt.Sprintf("  %s %s\n\n", mark, title))

	envVar := getEnvVarForProvider(provider)
	s.WriteString(fmt.Sprintf("  Provider: %s\n", statValue.Render(provider)))
	s.WriteString(fmt.Sprintf("  Required: %s\n\n", statValue.Render(envVar)))

	s.WriteString(statLabel.Render("  To fix, run:\n"))
	s.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().
		Foreground(ColorAccent).
		Render(fmt.Sprintf("export %s=\"your-api-key\"", envVar))))

	s.WriteString(statLabel.Render("  Get your API key:\n"))
	url := getAPIKeyURL(provider)
	s.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().
		Foreground(ColorAccent).
		Underline(true).
		Render(url)))

	fmt.Println(errorBox.Render(s.String()))
}

func getEnvVarForProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	default:
		return "API_KEY"
	}
}

func getAPIKeyURL(provider string) string {
	switch strings.ToLower(provider) {
	case "openai":
		return "https://platform.openai.com/api-keys"
	case "anthropic":
		return "https://console.anthropic.com/settings/keys"
	case "gemini":
		return "https://aistudio.google.com/apikey"
	case "groq":
		return "https://console.groq.com/keys"
	default:
		return ""
	}
}
