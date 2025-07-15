package main

import tea "github.com/charmbracelet/bubbletea"

// Page interface that all pages must implement
type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
	Title() string
	Help() string
	ID() string // Unique identifier for the page
}
