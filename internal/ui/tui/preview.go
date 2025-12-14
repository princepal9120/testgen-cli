package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type PreviewModel struct {
	config       RunConfig
	focusConfirm bool
	width        int
	height       int
}

func NewPreviewModel() PreviewModel {
	return PreviewModel{focusConfirm: true}
}

func (m PreviewModel) SetConfig(config RunConfig) PreviewModel {
	m.config = config
	return m
}

func (m PreviewModel) Init() tea.Cmd {
	return nil
}

func (m PreviewModel) Update(msg tea.Msg) (PreviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.config.Mode == "generate" {
				return m, func() tea.Msg { return NavigateMsg{To: ScreenGenerateConfig} }
			}
			return m, func() tea.Msg { return NavigateMsg{To: ScreenAnalyzeConfig} }

		case "enter":
			return m, func() tea.Msg {
				return NavigateMsg{To: ScreenRunning, Config: &m.config}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m PreviewModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📋 Preview Command"))
	b.WriteString("\n\n")

	// Build CLI command
	cmd := m.buildCommand()
	b.WriteString(boxStyle.Render(cmd))
	b.WriteString("\n\n")

	// Config summary
	b.WriteString(subtitleStyle.Render("Configuration:"))
	b.WriteString("\n")
	b.WriteString(m.configSummary())
	b.WriteString("\n")

	// API provider status
	provider, ok := getConfiguredProvider()
	if ok {
		b.WriteString(successStyle.Render(fmt.Sprintf("✔ API Provider: %s", provider)))
	} else {
		b.WriteString(errorStyle.Render("✖ No API key configured"))
	}
	b.WriteString("\n\n")

	// Buttons
	b.WriteString(activeButtonStyle.Render("Run ▶"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("enter: run • esc: back"))

	return b.String()
}

func (m PreviewModel) buildCommand() string {
	var parts []string
	parts = append(parts, "testgen")

	if m.config.Mode == "generate" {
		parts = append(parts, "generate")

		if m.config.Path != "" {
			parts = append(parts, fmt.Sprintf("--path=%s", m.config.Path))
		}
		if m.config.File != "" {
			parts = append(parts, fmt.Sprintf("--file=%s", m.config.File))
		}
		if m.config.Recursive {
			parts = append(parts, "--recursive")
		}
		if len(m.config.Types) > 0 {
			parts = append(parts, fmt.Sprintf("--type=%s", strings.Join(m.config.Types, ",")))
		}
		if m.config.DryRun {
			parts = append(parts, "--dry-run")
		}
		if m.config.Validate {
			parts = append(parts, "--validate")
		}
		if m.config.Parallel > 0 && m.config.Parallel != 2 {
			parts = append(parts, fmt.Sprintf("--parallel=%d", m.config.Parallel))
		}
	} else {
		parts = append(parts, "analyze")

		if m.config.Path != "" {
			parts = append(parts, fmt.Sprintf("--path=%s", m.config.Path))
		}
		if m.config.CostEst {
			parts = append(parts, "--cost-estimate")
		}
		if m.config.Detail != "" && m.config.Detail != "summary" {
			parts = append(parts, fmt.Sprintf("--detail=%s", m.config.Detail))
		}
	}

	return strings.Join(parts, " ")
}

func (m PreviewModel) configSummary() string {
	var lines []string

	if m.config.Mode == "generate" {
		lines = append(lines, fmt.Sprintf("  Mode:       %s", "Generate Tests"))
		lines = append(lines, fmt.Sprintf("  Path:       %s", m.config.Path))
		lines = append(lines, fmt.Sprintf("  Types:      %s", strings.Join(m.config.Types, ", ")))
		lines = append(lines, fmt.Sprintf("  Recursive:  %v", m.config.Recursive))
		lines = append(lines, fmt.Sprintf("  Dry Run:    %v", m.config.DryRun))
		lines = append(lines, fmt.Sprintf("  Validate:   %v", m.config.Validate))
		lines = append(lines, fmt.Sprintf("  Parallel:   %d", m.config.Parallel))
	} else {
		lines = append(lines, fmt.Sprintf("  Mode:         %s", "Analyze Codebase"))
		lines = append(lines, fmt.Sprintf("  Path:         %s", m.config.Path))
		lines = append(lines, fmt.Sprintf("  Cost Est:     %v", m.config.CostEst))
		lines = append(lines, fmt.Sprintf("  Detail:       %s", m.config.Detail))
	}

	return strings.Join(lines, "\n")
}
