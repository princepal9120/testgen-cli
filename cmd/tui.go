package cmd

import (
	"github.com/princepal9120/testgen-cli/internal/ui/tui"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch the interactive Terminal User Interface (TUI) for TestGen.

The TUI provides a visual, keyboard-driven interface for:
  • Generating tests for source files
  • Analyzing codebases for cost estimation
  • Previewing commands before execution
  • Viewing results in a formatted display

Controls:
  • Arrow keys / Tab: Navigate
  • Enter: Select / Confirm
  • Space: Toggle options
  • Esc: Go back
  • q: Quit

Examples:
  # Launch TUI
  testgen tui`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	return tui.Run()
}
