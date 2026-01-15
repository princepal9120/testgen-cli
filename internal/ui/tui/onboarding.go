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

// Onboarding step
type OnboardingStep int

const (
	StepWelcome OnboardingStep = iota
	StepSelectProvider
	StepEnterKey
	StepComplete
)

type OnboardingModel struct {
	step        OnboardingStep
	providerIdx int
	textInput   textinput.Model
	err         error
	width       int
	height      int
}

func NewOnboardingModel() OnboardingModel {
	ti := textinput.New()
	ti.Placeholder = "Paste your API key here..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 50

	return OnboardingModel{
		step:        StepWelcome,
		providerIdx: 0,
		textInput:   ti,
	}
}

func (m OnboardingModel) Init() tea.Cmd {
	return nil
}

func (m OnboardingModel) Update(msg tea.Msg) (OnboardingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case StepWelcome:
			switch msg.String() {
			case "enter", " ":
				m.step = StepSelectProvider
				return m, nil
			case "q", "esc":
				return m, tea.Quit
			}

		case StepSelectProvider:
			switch msg.String() {
			case "up", "k":
				if m.providerIdx > 0 {
					m.providerIdx--
				}
			case "down", "j":
				if m.providerIdx < len(providers)-1 {
					m.providerIdx++
				}
			case "enter":
				m.step = StepEnterKey
				m.textInput.Focus()
				return m, textinput.Blink
			case "esc":
				m.step = StepWelcome
				return m, nil
			}

		case StepEnterKey:
			switch msg.String() {
			case "enter":
				if m.textInput.Value() != "" {
					m.err = m.saveAPIKey()
					if m.err == nil {
						m.step = StepComplete
					}
				}
				return m, nil
			case "esc":
				m.step = StepSelectProvider
				m.textInput.Reset()
				return m, nil
			}

		case StepComplete:
			switch msg.String() {
			case "enter", " ":
				return m, func() tea.Msg {
					return NavigateMsg{To: ScreenHome}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update text input
	if m.step == StepEnterKey {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m OnboardingModel) View() string {
	// Styles
	containerStyle := lipgloss.NewStyle().
		Padding(2, 4)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("15")).
		Padding(0, 3).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	var s strings.Builder

	switch m.step {
	case StepWelcome:
		// Clean welcome screen
		logo := `
  ████████╗███████╗███████╗████████╗ ██████╗ ███████╗███╗   ██╗
  ╚══██╔══╝██╔════╝██╔════╝╚══██╔══╝██╔════╝ ██╔════╝████╗  ██║
     ██║   █████╗  ███████╗   ██║   ██║  ███╗█████╗  ██╔██╗ ██║
     ██║   ██╔══╝  ╚════██║   ██║   ██║   ██║██╔══╝  ██║╚██╗██║
     ██║   ███████╗███████║   ██║   ╚██████╔╝███████╗██║ ╚████║
     ╚═╝   ╚══════╝╚══════╝   ╚═╝    ╚═════╝ ╚══════╝╚═╝  ╚═══╝`

		logoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

		s.WriteString(logoStyle.Render(logo) + "\n\n\n")

		tagline := headerStyle.Render("AI-Powered Test Generation")
		s.WriteString("       " + tagline + "\n\n")

		desc := subtitleStyle.Render("Generate comprehensive unit tests for your code using LLMs.\n       Supports Go, Python, Rust, JavaScript, and more.")
		s.WriteString("       " + desc + "\n\n\n")

		// Get started button
		s.WriteString("       " + buttonStyle.Render(" Get Started ") + "\n\n")
		s.WriteString("       " + dimStyle.Render("Press ENTER to continue • q to quit") + "\n")

	case StepSelectProvider:
		s.WriteString(headerStyle.Render("Select Your LLM Provider") + "\n\n")

		desc := subtitleStyle.Render("Choose the AI provider you'd like to use for test generation.\nYou can change this later in settings.")
		s.WriteString(desc + "\n\n")

		// Provider list with descriptions
		providerDetails := []struct {
			name   string
			desc   string
			badge  string
		}{
			{"Groq", "Ultra-fast inference, generous free tier", "RECOMMENDED"},
			{"Anthropic Claude", "Highest quality, best for complex code", "PREMIUM"},
			{"OpenAI GPT", "Most popular, reliable performance", ""},
			{"Google Gemini", "Good balance, free tier available", ""},
		}

		for i, p := range providerDetails {
			cursor := "  "
			nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
			descStyle := subtitleStyle

			if i == m.providerIdx {
				cursor = accentStyle.Render("▸ ")
				nameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
			}

			line := cursor + nameStyle.Render(p.name)

			if p.badge != "" {
				badgeStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("0")).
					Background(lipgloss.Color("6")).
					Padding(0, 1).
					MarginLeft(1)
				if p.badge == "PREMIUM" {
					badgeStyle = badgeStyle.Background(lipgloss.Color("5"))
				}
				line += " " + badgeStyle.Render(p.badge)
			}

			s.WriteString(line + "\n")
			s.WriteString("   " + descStyle.Render(p.desc) + "\n\n")
		}

		s.WriteString("\n" + dimStyle.Render("↑/↓ select • enter confirm • esc back") + "\n")

	case StepEnterKey:
		p := providers[m.providerIdx]

		s.WriteString(headerStyle.Render("Enter Your API Key") + "\n\n")

		providerLabel := accentStyle.Render(strings.Title(p.name))
		s.WriteString(fmt.Sprintf("Provider: %s\n\n", providerLabel))

		// API key input
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1).
			Width(54)

		s.WriteString(inputBox.Render(m.textInput.View()) + "\n\n")

		// Error
		if m.err != nil {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			s.WriteString(errStyle.Render("✗ " + m.err.Error()) + "\n\n")
		}

		// Instructions
		url := getProviderURL(p.name)
		s.WriteString(subtitleStyle.Render("Get your API key: ") + accentStyle.Render(url) + "\n\n")

		securityNote := dimStyle.Render("🔒 Your key is stored locally in ~/.config/testgen/env")
		s.WriteString(securityNote + "\n\n")

		s.WriteString(dimStyle.Render("enter save • esc back") + "\n")

	case StepComplete:
		// Success screen
		checkmark := `
     ██████╗ ██╗  ██╗
    ██╔═══██╗██║ ██╔╝
    ██║   ██║█████╔╝ 
    ██║   ██║██╔═██╗ 
    ╚██████╔╝██║  ██╗
     ╚═════╝ ╚═╝  ╚═╝`

		checkStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

		s.WriteString(checkStyle.Render(checkmark) + "\n\n")

		s.WriteString(headerStyle.Render("You're All Set!") + "\n\n")

		p := providers[m.providerIdx]
		s.WriteString(subtitleStyle.Render(fmt.Sprintf("API key for %s has been saved.\n", strings.Title(p.name))))
		s.WriteString(subtitleStyle.Render("You're ready to generate tests for your code.") + "\n\n\n")

		s.WriteString("     " + buttonStyle.Render(" Start Using TestGen ") + "\n\n")
		s.WriteString("     " + dimStyle.Render("Press ENTER to continue") + "\n")
	}

	return containerStyle.Render(s.String())
}

func (m OnboardingModel) saveAPIKey() error {
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

// IsFirstTimeUser checks if any API key is configured
func IsFirstTimeUser() bool {
	keys := []string{"GROQ_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY"}
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return false
		}
	}

	// Also check config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return true
	}

	envFile := filepath.Join(homeDir, ".config", "testgen", "env")
	if _, err := os.Stat(envFile); err == nil {
		// File exists, try to source it
		data, err := os.ReadFile(envFile)
		if err == nil && len(data) > 0 {
			return false
		}
	}

	return true
}
