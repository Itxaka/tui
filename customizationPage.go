package main

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mudler/go-pluggable"
	"github.com/sanity-io/litter"
)

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
