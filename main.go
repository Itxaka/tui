package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mudler/go-pluggable"
	"github.com/sanity-io/litter"
	"log"
	"os"
	"strings"
	"time"
)

// Page interface that all pages must implement
type Page interface {
	Init() tea.Cmd
	Update(tea.Msg) (Page, tea.Cmd)
	View() string
	Title() string
	Help() string
}

// Main application model
type model struct {
	pages           []Page
	currentPage     int
	navigationStack []int // Stack to track navigation history
	width           int
	height          int
	title           string
	disk            string // Selected disk
	username        string
	sshKeys         []string // Store SSH keys
	password        string
	log             *log.Logger
}

var mainModel model

func newLogger() *log.Logger {
	f, err := os.OpenFile("/tmp/kairos-installer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return log.New(os.Stdout, "", log.LstdFlags)
	}
	return log.New(f, "", log.LstdFlags)
}

const genericNavigationHelp = "↑/k: up • ↓/j: down • enter: select"

// Initialize the application
func initialModel() model {
	mainModel = model{
		currentPage:     0,
		navigationStack: []int{}, // Initialize empty stack
		title:           "Kairos Interactive Installer",
		log:             newLogger(),
	}
	// Create installation workflow pages
	pages := []Page{
		newDiskSelectionPage(),
		newConfirmationPage(),
		newInstallOptionsPage(),
		newCustomizationPage(),
		newUserPasswordPage(),
		newSSHKeysPage(),
		newInstallProcessPage(),
	}

	mainModel.pages = pages
	return mainModel
}

func (m model) Init() tea.Cmd {
	mainModel.log.Printf("Starting Kairos Interactive Installer")
	if len(m.pages) > 0 {
		return m.pages[m.currentPage].Init()
	}

	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Hijack all keys if on install process page
	if installPage, ok := m.pages[m.currentPage].(*installProcessPage); ok {
		if installPage.progress < len(installPage.steps)-1 {
			// Ignore all key events during install
			if _, isKey := msg.(tea.KeyMsg); isKey {
				return m, nil
			}
		}
		if installPage.progress >= len(installPage.steps)-1 {
			// After install, any key exits
			if _, isKey := msg.(tea.KeyMsg); isKey {
				return m, tea.Quit
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			// Go back to previous page if we have navigation history
			if len(m.navigationStack) > 0 {
				// Pop the last page from the stack
				m.currentPage = m.navigationStack[len(m.navigationStack)-1]
				m.navigationStack = m.navigationStack[:len(m.navigationStack)-1]
				return m, m.pages[m.currentPage].Init()
			}
		}
	}

	// Handle page navigation
	if m.currentPage < len(m.pages) {
		updatedPage, cmd := m.pages[m.currentPage].Update(msg)
		m.pages[m.currentPage] = updatedPage

		// Check if we need to navigate to next page
		if _, ok := msg.(NextPageMsg); ok {
			if m.currentPage < len(m.pages)-1 {
				// Push current page to navigation stack
				m.navigationStack = append(m.navigationStack, m.currentPage)
				m.currentPage++
				return m, tea.Batch(cmd, m.pages[m.currentPage].Init())
			}
		}

		// Check if we need to navigate to a specific page
		if goToPageMsg, ok := msg.(GoToPageMsg); ok {
			if goToPageMsg.PageIndex >= 0 && goToPageMsg.PageIndex < len(m.pages) {
				// Push current page to navigation stack
				m.navigationStack = append(m.navigationStack, m.currentPage)
				m.currentPage = goToPageMsg.PageIndex
				return m, tea.Batch(cmd, m.pages[m.currentPage].Init())
			}
		}

		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Kairos.io themed border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(kairosBorder).
		Background(kairosBg).
		Padding(1).
		Width(m.width - 4).
		Height(m.height - 4)

	// Kairos.io themed title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(kairosHighlight).
		Background(kairosBg).
		Padding(1, 4).
		Align(lipgloss.Center)

	// Get current page content
	content := ""
	help := ""
	if m.currentPage < len(m.pages) {
		content = m.pages[m.currentPage].View()
		help = m.pages[m.currentPage].Help()
	}

	// Create the bordered content
	title := titleStyle.Render(m.title)

	// Add help text at the bottom
	helpStyle := lipgloss.NewStyle().
		Foreground(kairosText).
		Italic(true)

	// If install process, show minimal help
	var fullHelp string
	if _, ok := m.pages[m.currentPage].(*installProcessPage); ok {
		fullHelp = help
	} else {
		fullHelp = help + " • ESC: back • q/ctrl+c: quit"
	}

	helpText := helpStyle.Render(fullHelp)

	// Calculate available space for content
	availableHeight := m.height - 8 // Account for border, padding, title, and help
	contentHeight := availableHeight - 2

	// Ensure content fits within available space
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
		content = strings.Join(contentLines, "\n")
	}

	// Combine title, content, and help
	pageContent := fmt.Sprintf("%s\n\n%s\n\n%s", title, content, helpText)

	return borderStyle.Render(pageContent)
}

// NextPageMsg is a custom message type for page navigation
type NextPageMsg struct{}

// GoToPageMsg is a custom message type for navigating to a specific page
type GoToPageMsg struct {
	PageIndex int
}

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
			return p, func() tea.Msg { return GoToPageMsg{PageIndex: 1} }
		}
	}
	return p, nil
}

func (p *diskSelectionPage) View() string {
	s := "Select target disk for installation:\n\n"
	s += "⚠️  WARNING: All data on the selected disk will be DESTROYED!\n\n"

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
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 2} }
			} else {
				// No - clear selected disk and go back to disk selection
				mainModel.disk = ""
				mainModel.log.Printf("Installation cancelled, going back to disk selection")
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 0} }
			}
		}
	}
	return p, nil
}

func (p *confirmationPage) View() string {
	s := "⚠️  FINAL CONFIRMATION ⚠️\n\n"
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

// Install Options Page
type installOptionsPage struct {
	cursor  int
	options []string
}

func newInstallOptionsPage() *installOptionsPage {
	return &installOptionsPage{
		options: []string{
			"Start Install",
			"Customize Further",
		},
		cursor: 0,
	}
}

func (p *installOptionsPage) Init() tea.Cmd {
	return nil
}

func (p *installOptionsPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.options)-1 {
				p.cursor++
			}
		case "enter":
			if p.cursor == 0 {
				// Start Install - go to install process
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 6} }
			} else {
				// Customize Further - go to customization page
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 3} }
			}
		}
	}
	return p, nil
}

func (p *installOptionsPage) View() string {
	s := "Installation Options\n\n"
	s += "Choose how to proceed:\n\n"

	for i, option := range p.options {
		cursor := " "
		if p.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
		}
		s += fmt.Sprintf("%s %s\n", cursor, option)
	}

	return s
}

func (p *installOptionsPage) Title() string {
	return "Install Options"
}

func (p *installOptionsPage) Help() string {
	return genericNavigationHelp
}

// Customization Page

type YAMLPrompt struct {
	YAMLSection string
	Bool        bool
	Prompt      string
	Default     string
	AskFirst    bool
	AskPrompt   string
	IfEmpty     string
	PlaceHolder string
}

type EventPayload struct {
	Config string `json:"config"`
}

// Discover and run plugins for customization
func runCustomizationPlugins() ([]YAMLPrompt, error) {
	Manager.Initialize()
	var r []YAMLPrompt
	Manager.Response("agent.interactive-install", func(p *pluggable.Plugin, resp *pluggable.EventResponse) {
		mainModel.log.Printf("Received response from plugin %s at %s: %s", p.Name, p.Executable, resp.Data)
		err := json.Unmarshal([]byte(resp.Data), &r)
		if err != nil {
			fmt.Println(err)
		}
	})

	_, err := Manager.Publish("agent.interactive-install", EventPayload{})
	if err != nil {
		return r, err
	}

	return r, nil

}

type customizationPage struct {
	cursor  int
	options []string
}

func (p *customizationPage) Title() string {
	return "Customization"
}

func (p *customizationPage) Help() string {
	return genericNavigationHelp
}

func newCustomizationPage() *customizationPage {
	return &customizationPage{
		options: []string{
			"User & Password",
			"SSH Keys",
			"Finish Customization",
		},
		cursor: 0,
	}
}

func (p *customizationPage) Init() tea.Cmd {
	// Create the customization plugins pages
	mainModel.log.Printf("Running customization plugins...")
	yaML, err := runCustomizationPlugins()
	mainModel.log.Printf("Customization plugins returned: %s", litter.Sdump(yaML))
	if err != nil {
		mainModel.log.Printf("Error running customization plugins: %v", err)
		fmt.Println("Error running customization plugins:", err)
		return nil
	}
	return nil
}

func (p *customizationPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.options)-1 {
				p.cursor++
			}
		case "enter":
			switch p.cursor {
			case 0:
				// User & Password
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 4} }
			case 1:
				// SSH Keys
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 5} }
			case 2:
				// Finish Customization - go to install process
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 6} }
			}
		}
	}
	return p, nil
}

func (p *customizationPage) View() string {
	s := "Customization Options\n\n"
	s += "Configure additional settings:\n\n"

	for i, option := range p.options {
		cursor := " "
		if p.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
		}
		tick := ""
		if i == 0 {
			// User & Password
			if p.isUserConfigured() {
				tick = lipgloss.NewStyle().Foreground(kairosAccent).Render("✓")
			}
		}
		if i == 1 {
			// SSH Keys
			if p.isSSHConfigured() {
				tick = lipgloss.NewStyle().Foreground(kairosAccent).Render("✓")
			}
		}
		s += fmt.Sprintf("%s %s %s\n", cursor, option, tick)
	}

	return s
}

// Helper methods to check configuration
func (p *customizationPage) isUserConfigured() bool {
	if &mainModel != nil {
		return mainModel.username != "" && mainModel.password != ""
	}
	return false
}

func (p *customizationPage) isSSHConfigured() bool {
	if &mainModel != nil {
		return len(mainModel.sshKeys) > 0
	}
	return false
}

// User Password Page
type userPasswordPage struct {
	focusedField  int // 0 = username, 1 = password
	usernameInput textinput.Model
	passwordInput textinput.Model
	username      string
	password      string
}

func newUserPasswordPage() *userPasswordPage {
	usernameInput := textinput.New()
	usernameInput.Placeholder = "Kairos"
	usernameInput.Width = 20
	usernameInput.Focus()

	passwordInput := textinput.New()
	passwordInput.Width = 20
	passwordInput.Placeholder = "Kairos"
	passwordInput.EchoMode = textinput.EchoPassword

	return &userPasswordPage{
		focusedField:  0,
		usernameInput: usernameInput,
		passwordInput: passwordInput,
	}
}

func (p *userPasswordPage) Init() tea.Cmd {
	return textinput.Blink
}

func (p *userPasswordPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if p.focusedField == 0 {
				p.focusedField = 1
				p.usernameInput.Blur()
				p.passwordInput.Focus()
				return p, p.passwordInput.Focus()
			} else {
				p.focusedField = 0
				p.passwordInput.Blur()
				p.usernameInput.Focus()
				return p, p.usernameInput.Focus()
			}
		case "enter":
			if p.usernameInput.Value() != "" && p.passwordInput.Value() != "" {
				p.username = p.usernameInput.Value()
				mainModel.username = p.username
				p.password = p.passwordInput.Value()
				mainModel.password = p.password
				// Save and go back to customization
				return p, func() tea.Msg { return GoToPageMsg{PageIndex: 3} }
			}
		}
	}

	if p.focusedField == 0 {
		p.usernameInput, cmd = p.usernameInput.Update(msg)
	} else {
		p.passwordInput, cmd = p.passwordInput.Update(msg)
	}

	return p, cmd
}

func (p *userPasswordPage) View() string {
	s := "User Account Setup\n\n"
	s += "Username:\n"
	s += p.usernameInput.View() + "\n\n"
	s += "Password:\n"
	s += p.passwordInput.View() + "\n\n"

	if p.username != "" {
		s += fmt.Sprintf("✓ User configured: %s\n", p.username)
	}

	if p.usernameInput.Value() == "" || p.passwordInput.Value() == "" {
		s += "\nBoth fields are required to continue."
	}

	return s
}

func (p *userPasswordPage) Title() string {
	return "User & Password"
}

func (p *userPasswordPage) Help() string {
	return "tab: switch fields • enter: save and continue"
}

// SSH Keys Page
type sshKeysPage struct {
	mode     int // 0 = list view, 1 = add key input
	cursor   int
	sshKeys  []string
	keyInput textinput.Model
}

func newSSHKeysPage() *sshKeysPage {
	keyInput := textinput.New()
	keyInput.Placeholder = "github:USERNAME or "
	keyInput.Width = 60

	return &sshKeysPage{
		mode:     0,
		cursor:   0,
		sshKeys:  []string{},
		keyInput: keyInput,
	}
}

func (p *sshKeysPage) Init() tea.Cmd {
	return nil
}

func (p *sshKeysPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.mode == 0 { // List view
			switch msg.String() {
			case "up", "k":
				if p.cursor > 0 {
					p.cursor--
				}
			case "down", "j":
				if p.cursor < len(p.sshKeys) { // +1 for "Add new key" option
					p.cursor++
				}
			case "d":
				// Delete selected key
				if p.cursor < len(p.sshKeys) {
					p.sshKeys = append(p.sshKeys[:p.cursor], p.sshKeys[p.cursor+1:]...)
					mainModel.sshKeys = append(mainModel.sshKeys[:p.cursor], mainModel.sshKeys[p.cursor+1:]...)
					if p.cursor >= len(p.sshKeys) && p.cursor > 0 {
						p.cursor--
					}
				}
			case "a", "enter":
				if p.cursor == len(p.sshKeys) {
					// Add new key
					p.mode = 1
					p.keyInput.Focus()
					return p, textinput.Blink
				}
			}
		} else { // Add key input mode
			switch msg.String() {
			case "esc":
				p.mode = 0
				p.keyInput.Blur()
				p.keyInput.SetValue("")
				return p, nil
			case "enter":
				if p.keyInput.Value() != "" {
					p.sshKeys = append(p.sshKeys, p.keyInput.Value())
					mainModel.sshKeys = append(mainModel.sshKeys, p.keyInput.Value())
					p.mode = 0
					p.keyInput.Blur()
					p.keyInput.SetValue("")
					p.cursor = len(p.sshKeys) // Point to "Add new key" option
					return p, nil
				}
			}
			p.keyInput, cmd = p.keyInput.Update(msg)
		}
	}

	return p, cmd
}

func (p *sshKeysPage) View() string {
	s := "SSH Keys Management\n\n"

	if p.mode == 0 {
		s += "Current SSH Keys:\n\n"

		for i, key := range p.sshKeys {
			cursor := " "
			if p.cursor == i {
				cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
			}
			// Truncate long keys for display
			displayKey := key
			if len(displayKey) > 50 {
				displayKey = displayKey[:47] + "..."
			}
			s += fmt.Sprintf("%s %s\n", cursor, displayKey)
		}

		// Add "Add new key" option
		cursor := " "
		if p.cursor == len(p.sshKeys) {
			cursor = lipgloss.NewStyle().Foreground(kairosAccent).Render(">")
		}
		s += fmt.Sprintf("%s + Add new SSH key\n", cursor)

		s += "\nPress 'd' to delete selected key"
	} else {
		s += "Add SSH Public Key:\n\n"
		s += p.keyInput.View() + "\n\n"
		s += "Paste your SSH public key above."
	}

	return s
}

func (p *sshKeysPage) Title() string {
	return "SSH Keys"
}

func (p *sshKeysPage) Help() string {
	if p.mode == 0 {
		return "↑/k: up • ↓/j: down • enter/a: add key • d: delete • esc: back"
	}
	return "Type SSH key • enter: add • esc: cancel"
}

// Install Process Page
type installProcessPage struct {
	progress int
	step     string
	steps    []string
}

func newInstallProcessPage() *installProcessPage {
	return &installProcessPage{
		progress: 0,
		step:     "Preparing installation...",
		steps: []string{
			"Preparing installation...",
			"Partitioning disk...",
			"Formatting partitions...",
			"Installing base system...",
			"Configuring bootloader...",
			"Installing packages...",
			"Configuring user account...",
			"Setting up SSH keys...",
			"Finalizing installation...",
			"Installation complete!",
		},
	}
}

func (p *installProcessPage) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return "tick"
	})
}

func (p *installProcessPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	// TODO: Make some magic here and call the binary installer?
	// 2 ways fo dealing with this, either just stream the log output
	// Or read the output and update the progress bar accordingly depending ont he messages
	// we get
	switch msg := msg.(type) {
	case string:
		if msg == "tick" && p.progress < len(p.steps)-1 {
			p.progress++
			p.step = p.steps[p.progress]
			return p, tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
				return "tick"
			})
		}
	}
	return p, nil
}

func (p *installProcessPage) View() string {
	s := "Installation in Progress\n\n"

	// Progress bar
	totalSteps := len(p.steps)
	progressPercent := (p.progress * 100) / (totalSteps - 1)
	barWidth := 40 // Make progress bar wider
	filled := barWidth * progressPercent / 100
	progressBar := lipgloss.NewStyle().Foreground(kairosHighlight2).Background(kairosBg).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(kairosBorder).Background(kairosBg).Render(strings.Repeat("░", barWidth-filled))

	s += "Progress:" + progressBar + lipgloss.NewStyle().Background(kairosBg).Render(" ")
	s += lipgloss.NewStyle().Foreground(kairosText).Background(kairosBg).Bold(true).Render(fmt.Sprintf("%d%%", progressPercent))
	s += "\n\n"
	s += fmt.Sprintf("Current step: %s\n\n", p.step)

	// Show completed steps
	s += "Completed steps:\n"
	for i := 0; i < p.progress; i++ {
		s += fmt.Sprintf("✓ %s\n", p.steps[i])
	}

	if p.progress < len(p.steps)-1 {
		s += "\n⚠️  Do not power off the system during installation!"
	} else {
		s += "\n🎉 Installation completed successfully!"
		s += "\nYou can now reboot your system."
	}

	return s
}

func (p *installProcessPage) Title() string {
	return "Installing"
}

func (p *installProcessPage) Help() string {
	if p.progress >= len(p.steps)-1 {
		return "Press any key to exit"
	}
	return "Installation in progress - please wait..."
}

var (
	// Updated Kairos.io color palette
	kairosBg         = lipgloss.Color("#03153a") // Deep blue background
	kairosHighlight  = lipgloss.Color("#e56a44") // Orange highlight
	kairosHighlight2 = lipgloss.Color("#d54b11") // Red-orange highlight
	kairosAccent     = lipgloss.Color("#ee5007") // Accent orange
	kairosBorder     = lipgloss.Color("#e56a44") // Use highlight for border
	kairosText       = lipgloss.Color("#ffffff") // White text for contrast
)

// Main function
func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
