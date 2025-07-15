package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
)

// Main function
func main() {
	mainModel = initialModel()
	p := tea.NewProgram(mainModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
