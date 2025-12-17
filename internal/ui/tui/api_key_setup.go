package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Provider info
type provider struct {
	name   string
	envVar string
	desc   string
}

var providers = []provider{
	{name: "groq", envVar: "GROQ_API_KEY", desc: "Groq (fastest, free tier)"},
	{name: "anthropic", envVar: "ANTHROPIC_API_KEY", desc: "Anthropic Claude (best quality)"},
	{name: "openai", envVar: "OPENAI_API_KEY", desc: "OpenAI GPT"},
	{name: "gemini", envVar: "GEMINI_API_KEY", desc: "Google Gemini (free tier)"},
}

type APIKeySetupModel struct {
	providerIdx int
	textInput   textinput.Model
	saved       bool
	err         error
	width       int
	height      int
}

func NewAPIKeySetupModel() APIKeySetupModel {
	ti := textinput.New()
	ti.Placeholder = "Enter your API key..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.Width = 50

	return APIKeySetupModel{
		providerIdx: 0,
		textInput:   ti,
	}
}

func (m APIKeySetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m APIKeySetupModel) Update(msg tea.Msg) (APIKeySetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return NavigateMsg{To: ScreenHome}
			}

		case "tab", "shift+tab":
			// Cycle through providers
			if msg.String() == "tab" {
				m.providerIdx = (m.providerIdx + 1) % len(providers)
			} else {
				m.providerIdx = (m.providerIdx - 1 + len(providers)) % len(providers)
			}
			return m, nil

		case "up":
			if m.providerIdx > 0 {
				m.providerIdx--
			}
			return m, nil

		case "down":
			if m.providerIdx < len(providers)-1 {
				m.providerIdx++
			}
			return m, nil

		case "enter":
			if m.textInput.Value() != "" {
				m.err = m.saveAPIKey()
				if m.err == nil {
					m.saved = true
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m APIKeySetupModel) View() string {
	var s strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		MarginBottom(1).
		Render("🔑 Configure API Key")
	s.WriteString(title + "\n\n")

	// Success message
	if m.saved {
		successBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2).
			Render(successStyle.Render("✓ API key saved successfully!\n\n") +
				"Press ESC to return to home screen.")
		s.WriteString(successBox + "\n")
		return s.String()
	}

	// Error message
	if m.err != nil {
		s.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n\n")
	}

	// Provider selection
	s.WriteString("Select Provider:\n")
	for i, p := range providers {
		cursor := "  "
		style := itemStyle
		if i == m.providerIdx {
			cursor = "▸ "
			style = selectedItemStyle
		}
		line := fmt.Sprintf("%s%s", cursor, p.desc)
		s.WriteString(style.Render(line) + "\n")
	}

	s.WriteString("\n")

	// API Key input
	s.WriteString("API Key:\n")
	s.WriteString(m.textInput.View() + "\n\n")

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Get your API key from:\n" + getProviderURL(providers[m.providerIdx].name))
	s.WriteString(instructions + "\n\n")

	// Help
	s.WriteString(helpStyle.Render("↑/↓: select provider • enter: save • esc: back"))

	return s.String()
}

func (m APIKeySetupModel) saveAPIKey() error {
	p := providers[m.providerIdx]
	apiKey := m.textInput.Value()

	// Create config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "testgen")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	// Write to env file
	envFile := filepath.Join(configDir, "env")
	content := fmt.Sprintf("export %s=%s\n", p.envVar, apiKey)

	if err := os.WriteFile(envFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("could not save API key: %w", err)
	}

	// Also set in current process
	os.Setenv(p.envVar, apiKey)

	return nil
}

func getProviderURL(name string) string {
	switch name {
	case "groq":
		return "https://console.groq.com/keys"
	case "anthropic":
		return "https://console.anthropic.com/"
	case "openai":
		return "https://platform.openai.com/api-keys"
	case "gemini":
		return "https://aistudio.google.com/app/apikey"
	default:
		return ""
	}
}
