package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	colorPrimary   = lipgloss.Color("#4F46E5") // Indigo
	colorSecondary = lipgloss.Color("#10B981") // Emerald
	colorError     = lipgloss.Color("#EF4444") // Red
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorBg        = lipgloss.Color("#1F2937") // Dark bg
	colorFg        = lipgloss.Color("#F9FAFB") // Light fg

	// Base styles
	titleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginBottom(1)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	focusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary).
			Padding(1, 2)

	// Form styles
	inputStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	focusedInputStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(20)

	// Button styles
	buttonStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorPrimary).
			Padding(0, 2).
			MarginRight(1)

	activeButtonStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorSecondary).
				Padding(0, 2).
				Bold(true)

	// List item styles
	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(colorSecondary).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(colorSecondary)

	// Status styles
	successStyle = lipgloss.NewStyle().Foreground(colorSecondary)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError)
	infoStyle    = lipgloss.NewStyle().Foreground(colorMuted)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)
)
