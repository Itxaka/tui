package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
)

var version = "0.0.1" // Placeholder for version, can be set during build

// Main function
func main() {
	// if we have an arg and that arg is version or v, print the version and exit
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "v") {
		fmt.Println(version)
		os.Exit(0)
	}
	mainModel = initialModel()
	p := tea.NewProgram(mainModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
