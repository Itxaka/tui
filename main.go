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
	ID() string // Unique identifier for the page
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
		Align(lipgloss.Center)

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

// NextPageMsg is a custom message type for page navigation
type NextPageMsg struct{}

// GoToPageMsg is a custom message type for navigating to a specific page
// Updated to use PageID instead of PageIndex
type GoToPageMsg struct {
	PageID string
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
			return p, func() tea.Msg { return GoToPageMsg{PageID: "confirmation"} }
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

func (p *confirmationPage) ID() string { return "confirmation" }

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
				return p, func() tea.Msg { return GoToPageMsg{PageID: "install_process"} }
			} else {
				// Customize Further - go to customization page
				return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
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

func (p *installOptionsPage) ID() string { return "install_options" }

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
	cursor        int
	options       []string
	cursorWithIds map[int]string
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
			"Finish Customization",
			"User & Password",
			"SSH Keys",
		},

		cursor: 0,
		cursorWithIds: map[int]string{
			0: "install_process",
			1: "user_password",
			2: "ssh_keys",
		},
	}
}

func checkPageExists(pageID string, options map[int]string) bool {
	for _, opt := range options {
		if strings.Contains(opt, pageID) {
			return true
		}
	}
	return false
}

func (p *customizationPage) Init() tea.Cmd {
	mainModel.log.Printf("Running customization plugins...")
	yaML, err := runCustomizationPlugins()
	mainModel.log.Printf("Customization plugins returned: %s", litter.Sdump(yaML))
	if err != nil {
		mainModel.log.Printf("Error running customization plugins: %v", err)
		fmt.Println("Error running customization plugins:", err)
		return nil
	}
	if len(yaML) > 0 {
		startIdx := len(p.options)
		for i, prompt := range yaML {
			// Check if its already added to the options!
			if checkPageExists(idFromSection(prompt), p.cursorWithIds) {
				mainModel.log.Printf("Customization page for %s already exists, skipping", prompt.YAMLSection)
				continue
			}
			optIdx := startIdx + i
			if prompt.Bool == false {
				p.options = append(p.options, fmt.Sprintf("Configure %s", prompt.YAMLSection))
				pageID := idFromSection(prompt)
				p.cursorWithIds[optIdx] = pageID
				newPage := newGenericQuestionPage(prompt)
				mainModel.pages = append(mainModel.pages, newPage)
			} else {
				p.options = append(p.options, fmt.Sprintf("Configure %s", prompt.YAMLSection))
				pageID := idFromSection(prompt)
				p.cursorWithIds[optIdx] = pageID
				newPage := newGenericBoolPage(prompt)
				mainModel.pages = append(mainModel.pages, newPage)
			}
		}
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
			if pageID, ok := p.cursorWithIds[p.cursor]; ok {
				return p, func() tea.Msg { return GoToPageMsg{PageID: pageID} }
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

func (p *customizationPage) ID() string { return "customization" }

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
				return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
			}
		case "esc":
			// Go back to customization page
			return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
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

func (p *userPasswordPage) ID() string { return "user_password" }

// SSH Keys Page
type sshKeysPage struct {
	mode     int // 0 = list view, 1 = add key input
	cursor   int
	sshKeys  []string
	keyInput textinput.Model
}

func newSSHKeysPage() *sshKeysPage {
	keyInput := textinput.New()
	keyInput.Placeholder = "github:USERNAME or gitlab:USERNAME"
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
			case "esc":
				// Go back to customization page
				return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
			}
		} else { // Add key input mode
			switch msg.String() {
			case "esc":
				p.mode = 0
				p.keyInput.Blur()
				p.keyInput.SetValue("")
				// Go back to customization page
				return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
			case "enter":
				if p.keyInput.Value() != "" {
					p.sshKeys = append(p.sshKeys, p.keyInput.Value())
					mainModel.sshKeys = append(mainModel.sshKeys, p.keyInput.Value())
					p.mode = 0
					p.keyInput.Blur()
					p.keyInput.SetValue("")
					p.cursor = len(p.sshKeys) // Point to "Add new key" option
					// Go back to customization page
					return p, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
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

func (p *sshKeysPage) ID() string { return "ssh_keys" }

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

func (p *installProcessPage) ID() string { return "install_process" }

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
	mainModel = initialModel()
	p := tea.NewProgram(mainModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
