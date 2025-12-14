package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/princepal9120/testgen-cli/internal/adapters"
	"github.com/princepal9120/testgen-cli/internal/generator"
	"github.com/princepal9120/testgen-cli/internal/scanner"
	"github.com/princepal9120/testgen-cli/pkg/models"
	"github.com/spf13/viper"
)

type RunningModel struct {
	config   RunConfig
	spinner  spinner.Model
	viewport viewport.Model
	logs     []string
	running  bool
	done     bool
	cancel   context.CancelFunc
	width    int
	height   int
}

func NewRunningModel() RunningModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = successStyle

	return RunningModel{
		spinner: s,
		logs:    []string{},
	}
}

func (m RunningModel) SetConfig(config RunConfig) RunningModel {
	m.config = config
	return m
}

func (m RunningModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startExecution(),
	)
}

func (m RunningModel) Update(msg tea.Msg) (RunningModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+x":
			if m.cancel != nil {
				m.cancel()
				m.logs = append(m.logs, "Cancelling...")
			}
		case "esc", "enter":
			if m.done {
				return m, func() tea.Msg { return NavigateMsg{To: ScreenHome} }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case logMsg:
		m.logs = append(m.logs, string(msg))
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case GenerateCompleteMsg:
		m.done = true
		m.running = false
		return m, func() tea.Msg { return msg }

	case AnalyzeCompleteMsg:
		m.done = true
		m.running = false
		return m, func() tea.Msg { return msg }
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m RunningModel) View() string {
	var b strings.Builder

	if m.config.Mode == "generate" {
		b.WriteString(titleStyle.Render("⚡ Generating Tests"))
	} else {
		b.WriteString(titleStyle.Render("📊 Analyzing Codebase"))
	}
	b.WriteString("\n\n")

	if !m.done {
		b.WriteString(fmt.Sprintf("%s Running...\n\n", m.spinner.View()))
	} else {
		b.WriteString(successStyle.Render("✔ Complete"))
		b.WriteString("\n\n")
	}

	// Logs viewport
	b.WriteString(boxStyle.Render(strings.Join(m.logs, "\n")))
	b.WriteString("\n\n")

	if m.done {
		b.WriteString(helpStyle.Render("enter: continue • esc: home"))
	} else {
		b.WriteString(helpStyle.Render("ctrl+x: cancel"))
	}

	return b.String()
}

type logMsg string

func (m RunningModel) startExecution() tea.Cmd {
	return func() tea.Msg {
		if m.config.Mode == "generate" {
			return m.runGenerate()
		}
		return m.runAnalyze()
	}
}

func (m *RunningModel) runGenerate() tea.Msg {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	defer cancel()

	// Resolve path
	absPath, err := filepath.Abs(m.config.Path)
	if err != nil {
		return GenerateCompleteMsg{Err: err}
	}

	// Scan files
	s := scanner.New(scanner.Options{
		Recursive: m.config.Recursive,
	})

	sourceFiles, err := s.Scan(absPath)
	if err != nil {
		return GenerateCompleteMsg{Err: err}
	}

	if len(sourceFiles) == 0 {
		return GenerateCompleteMsg{Err: fmt.Errorf("no source files found")}
	}

	// Initialize engine
	engine, err := generator.NewEngine(generator.EngineConfig{
		DryRun:      m.config.DryRun,
		Validate:    m.config.Validate,
		TestTypes:   m.config.Types,
		Parallelism: m.config.Parallel,
		Provider:    viper.GetString("llm.provider"),
	})
	if err != nil {
		return GenerateCompleteMsg{Err: err}
	}

	// Get adapter registry
	registry := adapters.DefaultRegistry()

	// Process files
	var results []*models.GenerationResult
	for _, file := range sourceFiles {
		select {
		case <-ctx.Done():
			return GenerateCompleteMsg{Err: ctx.Err()}
		default:
		}

		adapter := registry.GetAdapter(file.Language)
		if adapter == nil {
			continue
		}

		result, err := engine.Generate(file, adapter)
		if err != nil {
			results = append(results, &models.GenerationResult{
				SourceFile: file,
				Error:      err,
			})
			continue
		}
		results = append(results, result)
	}

	return GenerateCompleteMsg{Results: results}
}

func (m *RunningModel) runAnalyze() tea.Msg {
	// Resolve path
	absPath, err := filepath.Abs(m.config.Path)
	if err != nil {
		return AnalyzeCompleteMsg{Err: err}
	}

	// Scan files
	s := scanner.New(scanner.Options{
		Recursive: m.config.Recursive,
	})

	sourceFiles, err := s.Scan(absPath)
	if err != nil {
		return AnalyzeCompleteMsg{Err: err}
	}

	// Basic analysis
	result := map[string]interface{}{
		"path":        absPath,
		"total_files": len(sourceFiles),
	}

	return AnalyzeCompleteMsg{Result: result}
}
