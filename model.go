package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newLogger() *log.Logger {
	f, err := os.OpenFile("/tmp/kairos-installer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.New(os.Stdout, "", log.LstdFlags)
	}
	return log.New(f, "", log.LstdFlags)
}

// NextPageMsg is a custom message type for page navigation
type NextPageMsg struct{}

// GoToPageMsg is a custom message type for navigating to a specific page
// Updated to use PageID instead of PageIndex
type GoToPageMsg struct {
	PageID string
}

// Main application model
type model struct {
	pages           []Page
	currentPageID   string   // Track current page by ID
	navigationStack []string // Stack to track navigation history by ID
	width           int
	height          int
	title           string
	disk            string // Selected disk
	username        string
	sshKeys         []string // Store SSH keys
	password        string
	extraFields     map[string]any // Dynamic fields for customization
	log             *log.Logger
}

var mainModel model

// Initialize the application
func initialModel() model {
	pages := []Page{
		newDiskSelectionPage(),
		newConfirmationPage(),
		newInstallOptionsPage(),
		newCustomizationPage(),
		newUserPasswordPage(),
		newSSHKeysPage(),
		newInstallProcessPage(),
	}
	mainModel = model{
		pages:           pages,
		currentPageID:   pages[0].ID(), // Start with first page ID
		navigationStack: []string{},
		title:           "Kairos Interactive Installer",
		log:             newLogger(),
	}
	return mainModel
}

func (m model) Init() tea.Cmd {
	mainModel.log.Printf("Starting Kairos Interactive Installer")
	if len(mainModel.pages) > 0 {
		for _, p := range mainModel.pages {
			if p.ID() == mainModel.currentPageID {
				return p.Init()
			}
		}
	}

	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// For navigation, access the mainModel so we can modify from anywhere
	currentIdx := -1
	for i, p := range mainModel.pages {
		if p.ID() == mainModel.currentPageID {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return mainModel, nil
	}

	// Hijack all keys if on install process page
	if installPage, ok := mainModel.pages[currentIdx].(*installProcessPage); ok {
		if installPage.progress < len(installPage.steps)-1 {
			// Ignore all key events during install
			if _, isKey := msg.(tea.KeyMsg); isKey {
				return mainModel, nil
			}
		}
		if installPage.progress >= len(installPage.steps)-1 {
			// After install, any key exits
			if _, isKey := msg.(tea.KeyMsg); isKey {
				return mainModel, tea.Quit
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		mainModel.width = msg.Width
		mainModel.height = msg.Height
		return mainModel, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return mainModel, tea.Quit
		case "esc":
			// Go back to previous page if we have navigation history
			if len(mainModel.navigationStack) > 0 {
				// Pop the last page from the stack
				mainModel.currentPageID = mainModel.navigationStack[len(mainModel.navigationStack)-1]
				mainModel.navigationStack = mainModel.navigationStack[:len(mainModel.navigationStack)-1]
				return mainModel, mainModel.pages[currentIdx].Init()
			}
		}
	}

	// Handle page navigation
	if currentIdx < len(mainModel.pages) {
		updatedPage, cmd := mainModel.pages[currentIdx].Update(msg)
		mainModel.pages[currentIdx] = updatedPage

		// Check if we need to navigate to next page
		if _, ok := msg.(NextPageMsg); ok {
			if currentIdx < len(mainModel.pages)-1 {
				// Push current page to navigation stack
				mainModel.navigationStack = append(mainModel.navigationStack, mainModel.currentPageID)
				mainModel.currentPageID = mainModel.pages[currentIdx+1].ID()
				return mainModel, tea.Batch(cmd, mainModel.pages[currentIdx+1].Init())
			}
		}

		// Check if we need to navigate to a specific page
		if goToPageMsg, ok := msg.(GoToPageMsg); ok {
			if goToPageMsg.PageID != "" {
				for i, p := range mainModel.pages {
					if p.ID() == goToPageMsg.PageID {
						mainModel.navigationStack = append(mainModel.navigationStack, mainModel.currentPageID)
						mainModel.currentPageID = goToPageMsg.PageID
						return mainModel, tea.Batch(cmd, mainModel.pages[i].Init())
					}
				}
				mainModel.log.Printf("model.Update: pageID=%s not found in mainModel.pages", goToPageMsg.PageID)
			}
		}

		return mainModel, cmd
	}

	return mainModel, nil
}

func (m model) View() string {
	if mainModel.width == 0 || mainModel.height == 0 {
		return "Loading..."
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(kairosBorder).
		Background(kairosBg).
		Padding(1).
		Width(mainModel.width - 4).
		Height(mainModel.height - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(kairosHighlight).
		Background(kairosBg).
		Padding(1, 4).
		Align(lipgloss.Center).AlignHorizontal(lipgloss.Center)

	// Get current page content by ID
	content := ""
	help := ""
	for _, p := range mainModel.pages {
		if p.ID() == mainModel.currentPageID {
			content = p.View()
			help = p.Help()
			break
		}
	}

	title := titleStyle.Render(mainModel.title)

	helpStyle := lipgloss.NewStyle().
		Foreground(kairosText).
		Italic(true)

	var fullHelp string
	currentIdx := -1
	for i, p := range mainModel.pages {
		if p.ID() == mainModel.currentPageID {
			currentIdx = i
			break
		}
	}
	if currentIdx != -1 {
		if _, ok := mainModel.pages[currentIdx].(*installProcessPage); ok {
			fullHelp = help
		} else {
			fullHelp = help + " • ESC: back • q/ctrl+c: quit"
		}
	}

	helpText := helpStyle.Render(fullHelp)

	availableHeight := mainModel.height - 8
	contentHeight := availableHeight - 2
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
		content = strings.Join(contentLines, "\n")
	}

	pageContent := fmt.Sprintf("%s\n\n%s\n\n%s", title, content, helpText)

	return borderStyle.Render(pageContent)
}
