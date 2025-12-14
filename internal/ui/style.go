package ui

import "github.com/charmbracelet/lipgloss"

// Minimalist Design System
// Colors: Monochrome + Yellow Accent (#1F2937, #FFFFFF, #F59E0B)

var (
	// Colors
	ColorBg      = lipgloss.Color("#000000") // Black
	ColorFg      = lipgloss.Color("#FFFFFF") // White
	ColorSub     = lipgloss.Color("#9CA3AF") // Gray-400
	ColorAccent  = lipgloss.Color("#F59E0B") // Amber-500
	ColorSuccess = lipgloss.Color("#10B981") // Emerald-500
	ColorError   = lipgloss.Color("#EF4444") // Red-500

	// Text Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSub).
			MarginBottom(1)

	// Status Indicators
	PassStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	FailStyle = lipgloss.NewStyle().Foreground(ColorError)
	InfoStyle = lipgloss.NewStyle().Foreground(ColorSub)

	// List Items
	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(ColorAccent).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(ColorAccent)

	// Box/Container
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSub).
			Padding(1, 2)

	// Code/Detail View
	DetailStyle = lipgloss.NewStyle().
			Foreground(ColorSub).
			PaddingLeft(4)
)
