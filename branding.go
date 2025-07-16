package main

import (
	"os"
	"path/filepath"
)

// This sets the text for the installer, allowing to override it with custom branding

func DefaultTitle() string {
	// Load it from a text file or something
	branding, err := os.ReadFile(filepath.Join("/etc", "kairos", "branding", "interactive_install_text"))
	if err == nil {
		return string(branding)
	} else {
		return "Kairos Interactive Installer"
	}
}
