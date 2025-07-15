package main

import "github.com/charmbracelet/lipgloss"

const (
	// Updated Kairos.io color palette
	kairosBg              = lipgloss.Color("#03153a") // Deep blue background
	kairosHighlight       = lipgloss.Color("#e56a44") // Orange highlight
	kairosHighlight2      = lipgloss.Color("#d54b11") // Red-orange highlight
	kairosAccent          = lipgloss.Color("#ee5007") // Accent orange
	kairosBorder          = lipgloss.Color("#e56a44") // Use highlight for border
	kairosText            = lipgloss.Color("#ffffff") // White text for contrast
	genericNavigationHelp = "↑/k: up • ↓/j: down • enter: select"
	StepPrefix            = "STEP:"
	ErrorPrefix           = "ERROR:"
)

// Installation steps for show
const (
	InstallDefaultStep       = "Preparing installation"
	InstallPartitionStep     = "Partitioning disk"
	InstallBeforeInstallStep = "Running before-install"
	InstallActiveStep        = "Installing Active"
	InstallBootloaderStep    = "Configuring bootloader"
	InstallRecoveryStep      = "Creating Recovery"
	InstallPassiveStep       = "Creating Passive"
	InstallAfterInstallStep  = "Running after-install"
	InstallCompleteStep      = "Installation complete!"
)

// Installation steps to identify installer to UI
const (
	AgentPartitionLog     = "Partitioning device"
	AgentBeforeInstallLog = "Running stage: before-install"
	AgentActiveLog        = "Creating file system image"
	AgentBootloaderLog    = "Installing GRUB"
	AgentRecoveryLog      = "Copying /run/cos/state/cOS/active.img source to /run/cos/recovery/cOS/recovery.img"
	AgentPassiveLog       = "Copying /run/cos/state/cOS/active.img source to /run/cos/state/cOS/passive.img"
	AgentAfterInstallLog  = "Running stage: after-install"
	AgentCompleteLog      = "Installation complete" // This is not reported by the agent, we should add it.
)
