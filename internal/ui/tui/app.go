package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Screen int

const (
	ScreenOnboarding Screen = iota
	ScreenHome
	ScreenAPIKeySetup
	ScreenGenerateConfig
	ScreenAnalyzeConfig
	ScreenPreview
	ScreenRunning
	ScreenResults
)

type AppModel struct {
	screen         Screen
	width          int
	height         int
	onboarding     OnboardingModel
	home           HomeModel
	apiKeySetup    APIKeySetupModel
	generateConfig GenerateConfigModel
	analyzeConfig  AnalyzeConfigModel
	preview        PreviewModel
	running        RunningModel
	results        ResultsModel
	err            error
}

func NewAppModel() AppModel {
	// Check if this is a first-time user
	initialScreen := ScreenHome
	if IsFirstTimeUser() {
		initialScreen = ScreenOnboarding
	}

	return AppModel{
		screen:         initialScreen,
		onboarding:     NewOnboardingModel(),
		home:           NewHomeModel(),
		apiKeySetup:    NewAPIKeySetupModel(),
		generateConfig: NewGenerateConfigModel(),
		analyzeConfig:  NewAnalyzeConfigModel(),
		preview:        NewPreviewModel(),
		running:        NewRunningModel(),
		results:        NewResultsModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	if m.screen == ScreenOnboarding {
		return tea.Batch(
			m.onboarding.Init(),
			tea.EnterAltScreen,
		)
	}
	return tea.Batch(
		m.home.Init(),
		tea.EnterAltScreen,
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == ScreenHome {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case NavigateMsg:
		return m.handleNavigation(msg)

	case GenerateCompleteMsg:
		m.screen = ScreenResults
		m.results = m.results.SetResults(msg.Results, msg.Err)
		return m, nil

	case AnalyzeCompleteMsg:
		m.screen = ScreenResults
		m.results = m.results.SetAnalysis(msg.Result, msg.Err)
		return m, nil
	}

	// Delegate to current screen
	var cmd tea.Cmd
	switch m.screen {
	case ScreenOnboarding:
		m.onboarding, cmd = m.onboarding.Update(msg)
	case ScreenHome:
		m.home, cmd = m.home.Update(msg)
	case ScreenAPIKeySetup:
		m.apiKeySetup, cmd = m.apiKeySetup.Update(msg)
	case ScreenGenerateConfig:
		m.generateConfig, cmd = m.generateConfig.Update(msg)
	case ScreenAnalyzeConfig:
		m.analyzeConfig, cmd = m.analyzeConfig.Update(msg)
	case ScreenPreview:
		m.preview, cmd = m.preview.Update(msg)
	case ScreenRunning:
		m.running, cmd = m.running.Update(msg)
	case ScreenResults:
		m.results, cmd = m.results.Update(msg)
	}

	return m, cmd
}

func (m AppModel) handleNavigation(msg NavigateMsg) (tea.Model, tea.Cmd) {
	switch msg.To {
	case ScreenHome:
		m.screen = ScreenHome
		m.home = NewHomeModel()
		return m, m.home.Init()

	case ScreenAPIKeySetup:
		m.screen = ScreenAPIKeySetup
		m.apiKeySetup = NewAPIKeySetupModel()
		return m, m.apiKeySetup.Init()

	case ScreenGenerateConfig:
		m.screen = ScreenGenerateConfig
		m.generateConfig = NewGenerateConfigModel()
		return m, m.generateConfig.Init()

	case ScreenAnalyzeConfig:
		m.screen = ScreenAnalyzeConfig
		m.analyzeConfig = NewAnalyzeConfigModel()
		return m, m.analyzeConfig.Init()

	case ScreenPreview:
		m.screen = ScreenPreview
		if msg.Config != nil {
			m.preview = m.preview.SetConfig(*msg.Config)
		}
		return m, m.preview.Init()

	case ScreenRunning:
		m.screen = ScreenRunning
		if msg.Config != nil {
			m.running = m.running.SetConfig(*msg.Config)
		}
		return m, m.running.Init()

	case ScreenResults:
		m.screen = ScreenResults
		return m, m.results.Init()
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.screen {
	case ScreenOnboarding:
		return m.onboarding.View()
	case ScreenHome:
		return m.home.View()
	case ScreenAPIKeySetup:
		return m.apiKeySetup.View()
	case ScreenGenerateConfig:
		return m.generateConfig.View()
	case ScreenAnalyzeConfig:
		return m.analyzeConfig.View()
	case ScreenPreview:
		return m.preview.View()
	case ScreenRunning:
		return m.running.View()
	case ScreenResults:
		return m.results.View()
	}
	return ""
}

// Run starts the TUI application
func Run() error {
	p := tea.NewProgram(NewAppModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Messages for navigation and async operations
type NavigateMsg struct {
	To     Screen
	Config *RunConfig
}

type RunConfig struct {
	Mode      string // "generate" or "analyze"
	Path      string
	File      string
	Recursive bool
	Types     []string
	DryRun    bool
	Validate  bool
	Parallel  int
	CostEst   bool
	Detail    string
}

type GenerateCompleteMsg struct {
	Results interface{}
	Err     error
}

type AnalyzeCompleteMsg struct {
	Result interface{}
	Err    error
}

// Helper to check if API key is configured
func getConfiguredProvider() (string, bool) {
	providers := map[string]string{
		"groq":      "GROQ_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"gemini":    "GEMINI_API_KEY",
	}

	for name, envVar := range providers {
		if os.Getenv(envVar) != "" {
			return name, true
		}
	}
	return "", false
}
