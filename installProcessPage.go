package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Install Process Page
type installProcessPage struct {
	progress int
	step     string
	steps    []string
	done     chan bool   // Channel to signal when installation is complete
	output   chan string // Channel to receive output from the installer
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
			"Finalizing installation...",
			"Installation complete!",
		},
		done:   make(chan bool),
		output: make(chan string),
	}
}

func (p *installProcessPage) Init() tea.Cmd {
	// Start the actual installer binary as a background process
	go func() {
		defer close(p.done)

		// Build the command with arguments based on user selections
		diskArg := strings.Split(mainModel.disk, " ")[0] // Extract just the device path
		cmd := exec.Command("./fake.sh", diskArg, mainModel.username, mainModel.password)

		mainModel.log.Printf("Starting installer with disk=%s, user=%s", diskArg, mainModel.username)

		// Create pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			mainModel.log.Printf("Error creating stdout pipe: %v", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			mainModel.log.Printf("Error creating stderr pipe: %v", err)
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			mainModel.log.Printf("Error starting installer: %v", err)
			return
		}

		// Create a scanner to read stdout line by line
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))

		// Read output and send it to the channel
		go func() {
			for scanner.Scan() {
				line := scanner.Text()
				mainModel.log.Printf("Installer output: %s", line)

				// Send the line to the output channel
				p.output <- line

				// Parse output to determine current step based on keywords
				if strings.Contains(line, "Partitioning") {
					p.output <- "STEP:Partitioning disk..."
				} else if strings.Contains(line, "Formatting") {
					p.output <- "STEP:Formatting partitions..."
				} else if strings.Contains(line, "Installing base system") {
					p.output <- "STEP:Installing base system..."
				} else if strings.Contains(line, "Configuring bootloader") {
					p.output <- "STEP:Configuring bootloader..."
				} else if strings.Contains(line, "Finalizing") {
					p.output <- "STEP:Finalizing installation..."
				} else if strings.Contains(line, "Installation complete") || strings.Contains(line, "Success") {
					p.output <- "STEP:Installation complete!"
				}
			}
		}()

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			mainModel.log.Printf("Error waiting for installer: %v", err)
			p.output <- "ERROR:" + err.Error()
		} else {
			mainModel.log.Printf("Installation completed successfully")
			p.output <- "STEP:Installation complete!"
		}
	}()

	// Return a command that will check for output from the installer
	return func() tea.Msg {
		return CheckInstallerMsg{}
	}
}

// CheckInstallerMsg Message type to check for installer output
type CheckInstallerMsg struct{}

func (p *installProcessPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg.(type) {
	case CheckInstallerMsg:
		// Check for new output from the installer
		select {
		case output, ok := <-p.output:
			if !ok {
				// Channel closed, installer is done
				return p, nil
			}

			// Process the output
			if strings.HasPrefix(output, "STEP:") {
				// This is a step change notification
				stepName := strings.TrimPrefix(output, "STEP:")

				// Find the index of the step
				for i, s := range p.steps {
					if s == stepName {
						p.progress = i
						p.step = stepName
						break
					}
				}
			} else if strings.HasPrefix(output, "ERROR:") {
				// Handle error
				errorMsg := strings.TrimPrefix(output, "ERROR:")
				p.step = "Error: " + errorMsg
				return p, nil
			}

			// Continue checking for output
			return p, func() tea.Msg { return CheckInstallerMsg{} }

		case <-p.done:
			// Installer is finished
			p.progress = len(p.steps) - 1
			p.step = p.steps[len(p.steps)-1]
			return p, nil

		default:
			// No new output yet, check again after a short delay
			return p, tea.Tick(time.Millisecond*100, func(_ time.Time) tea.Msg {
				return CheckInstallerMsg{}
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
