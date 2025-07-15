package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sanity-io/litter"
	"strings"
)

type genericQuestionPage struct {
	genericInput textinput.Model
	section      YAMLPrompt
}

func (g genericQuestionPage) Init() tea.Cmd {
	return textinput.Blink
}

func (g genericQuestionPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if g.genericInput.Value() == "" && g.section.IfEmpty != "" {
				// If the input is empty and IfEmpty is set, use IfEmpty value
				g.genericInput.SetValue(g.section.IfEmpty)
			}
			// Now if the input is not empty, we can proceed
			if g.genericInput.Value() != "" {
				// Set the value in the mainModel extraFields
				// Split the section YAMLSection to use as a key. Each field separated by a dot is a nested field.
				// for example, if the section is "network.token", we will use "network" as a key and "token" as a subkey.
				// This allows us to store nested values in the mainModel.extraFields map.

				sections := strings.Split(g.section.YAMLSection, ".")
				if len(sections) > 1 {
					// If there are multiple sections, we need to create a nested map structure
					currentMap := mainModel.extraFields
					for i, section := range sections {
						if i == len(sections)-1 {
							// If it's the last section, set the value
							currentMap[section] = g.genericInput.Value()
						} else {
							// If it's not the last section, ensure the map exists
							if _, exists := currentMap[section]; !exists {
								currentMap[section] = make(map[string]interface{})
							}
							// Move to the next level in the map
							if nextMap, ok := currentMap[section].(map[string]interface{}); ok {
								currentMap = nextMap
							} else {
								// If the next level is not a map, we need to create it
								currentMap[section] = make(map[string]interface{})
								currentMap = currentMap[section].(map[string]interface{})
							}
						}

					}
				} else {
					// If there's only one section, just set the value directly
					mainModel.extraFields[g.section.YAMLSection] = g.genericInput.Value()
				}

				mainModel.log.Println(litter.Sdump(mainModel.extraFields))
				return g, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
			}
		case "esc":
			// Go back to customization page
			return g, func() tea.Msg { return GoToPageMsg{PageID: "customization"} }
		}
	}

	return g, cmd
}

func (g genericQuestionPage) View() string {
	s := g.section.Prompt + "\n\n"
	s += g.genericInput.View() + "\n\n"

	return s
}

func (g genericQuestionPage) Title() string {
	return idFromSection(g.section)
}

func (g genericQuestionPage) Help() string {
	return "Press Enter to submit your answer, or esc to cancel."
}

func (g genericQuestionPage) ID() string {
	return idFromSection(g.section)
}

func idFromSection(section YAMLPrompt) string {
	// Generate a unique ID based on the section's YAMLSection.
	// This could be a simple hash or just the section name.
	return section.YAMLSection
}

// newGenericQuestionPage initializes a new generic question page with a text input model.
// Uses the provided section to set up the input model.
func newGenericQuestionPage(section YAMLPrompt) *genericQuestionPage {
	genericInput := textinput.New()
	genericInput.Placeholder = section.PlaceHolder
	genericInput.Width = 20
	genericInput.Focus()

	return &genericQuestionPage{
		genericInput: genericInput,
		section:      section,
	}
}
