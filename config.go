package main

import (
	"gopkg.in/yaml.v3"
	"os"
)

// InstallConfig holds fixed and dynamic install fields
// Fixed fields: Username, Password, SSHKeys
// Dynamic fields: stored in ExtraFields

type InstallConfig struct {
	Install     map[string]any `yaml:"install"`
	ExtraFields map[string]any `yaml:",inline"`
}

// NewInstallConfig creates a new config from model values
func NewInstallConfig(m model) *InstallConfig {
	return &InstallConfig{
		Install: map[string]any{
			"username": m.username,
			"password": m.password,
			"ssh_keys": m.sshKeys,
		},
		ExtraFields: make(map[string]any), // can be populated with dynamic fields
	}
}

// WriteYAML writes the config to a YAML file
func (c *InstallConfig) WriteYAML(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	defer enc.Close()
	return enc.Encode(c)
}
