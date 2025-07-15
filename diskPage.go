package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Disk Selection Page
type diskSelectionPage struct {
	disks  []string
	cursor int
}

func newDiskSelectionPage() *diskSelectionPage {
	// TODO: Get the disks and maybe filter them somehow?
	return &diskSelectionPage{
		disks: []string{
			"/dev/sda - 500GB SSD",
			"/dev/sdb - 1TB HDD",
			"/dev/nvme0n1 - 256GB NVMe",
			"/dev/sdc - 2TB HDD",
		},
		cursor: 0,
	}
}

func (p *diskSelectionPage) Init() tea.Cmd {
	return nil
}

func (p *diskSelectionPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.disks)-1 {
				p.cursor++
			}
		case "enter":
			// Store selected disk in mainModel
			if p.cursor >= 0 && p.cursor < len(p.disks) {
				mainModel.disk = p.disks[p.cursor]
				mainModel.log.Printf("Selected disk: %s", mainModel.disk)
			}
			// Go to confirmation page
			return p, func() tea.Msg { return GoToPageMsg{PageID: "confirmation"} }
		}
	}
	return p, nil
}

func (p *diskSelectionPage) View() string {
	s := "Select target disk for installation:\n\n"
	s += "⚠  WARNING: All data on the selected disk will be DESTROYED!\n\n"

	for i, disk := range p.disks {
		cursor := " "
		if p.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
		}
		s += fmt.Sprintf("%s %s\n", cursor, disk)
	}

	return s
}

func (p *diskSelectionPage) Title() string {
	return "Disk Selection"
}

func (p *diskSelectionPage) Help() string {
	return genericNavigationHelp
}

func (p *diskSelectionPage) ID() string { return "disk_selection" }

// Confirmation Page
type confirmationPage struct {
	cursor  int
	options []string
}

func newConfirmationPage() *confirmationPage {
	return &confirmationPage{
		options: []string{"Yes, continue", "No, go back"},
		cursor:  1, // Default to "No"
	}
}

func (p *confirmationPage) Init() tea.Cmd {
	return nil
}

func (p *confirmationPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.cursor = 0
		case "down", "j":
			p.cursor = 1
		case "enter":
			if p.cursor == 0 {
				// Yes - go to install options
				return p, func() tea.Msg { return GoToPageMsg{PageID: "install_options"} }
			} else {
				// No - clear selected disk and go back to disk selection
				mainModel.disk = ""
				mainModel.log.Printf("Installation cancelled, going back to disk selection")
				return p, func() tea.Msg { return GoToPageMsg{PageID: "disk_selection"} }
			}
		}
	}
	return p, nil
}

func (p *confirmationPage) View() string {
	s := "⚠ FINAL CONFIRMATION ⚠\n\n"
	s += fmt.Sprintf("You are about to install Linux on the selected disk (%s).\n", mainModel.disk)
	s += "This will PERMANENTLY DELETE all existing data!\n\n"
	s += "Are you sure you want to continue?\n\n"

	for i, option := range p.options {
		cursor := " "
		if p.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
		}
		s += fmt.Sprintf("%s %s\n", cursor, option)
	}

	return s
}

func (p *confirmationPage) Title() string {
	return "Confirmation"
}

func (p *confirmationPage) Help() string {
	return genericNavigationHelp
}

func (p *confirmationPage) ID() string { return "confirmation" }
