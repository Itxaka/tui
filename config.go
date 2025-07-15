package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// InstallConfig holds fixed and dynamic install fields
// Fixed fields: Username, Password, SSHKeys
// Dynamic fields: stored in ExtraFields

type InstallConfig struct {
	Install     map[string]any `yaml:"install,omitempty"`
	Stages      map[string]any `yaml:"stages,omitempty"`
	ExtraFields map[string]any `yaml:",inline,omitempty"`
}

// NewInstallConfig creates a new config from model values
func NewInstallConfig(m model) *InstallConfig {
	installConfig := InstallConfig{
		Install:     map[string]any{},
		Stages:      map[string]any{},
		ExtraFields: make(map[string]any),
	}

	installConfig.Install["device"] = m.disk

	if m.username != "" && m.password != "" {
		stage := "initramfs"

		// If we have ssh keys we need to delay the user creation to the network stage so we can get those keys
		if m.sshKeys != nil && len(m.sshKeys) > 0 {
			stage = "network"
		}
		installConfig.Stages[stage] = []map[string]any{
			{
				"name": "Set user and password",
				"users": map[string]any{
					"kairos": map[string]any{
						"passwd":              m.password,
						"groups":              []string{"admin"},
						"ssh_authorized_keys": m.sshKeys,
					},
				},
			},
		}
	} else {
		// No users set, we need to skip the user validation
		installConfig.Install["nousers"] = true
	}

	// Always set the extra fields
	installConfig.ExtraFields = m.extraFields

	return &installConfig
}

// WriteYAML writes the config to a YAML file
func (c *InstallConfig) WriteYAML(path string) error {
	mainModel.log.Printf("Writing install config to %s", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	defer enc.Close()
	return enc.Encode(c)
}
