package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlanIsaacV/dotcli/internal/models"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	dotfilesPath string
	modulesPath  string
}

type Template struct {
	Name        string
	Description string
	Config      models.ModuleConfig
	Files       map[string]string
}

func New(dotfilesPath string) *Manager {
	return &Manager{
		dotfilesPath: dotfilesPath,
		modulesPath:  filepath.Join(dotfilesPath, "modules"),
	}
}

func (m *Manager) CreateModule(name, templateName string) error {
	modulePath := filepath.Join(m.modulesPath, name)

	if _, err := os.Stat(modulePath); !os.IsNotExist(err) {
		return fmt.Errorf("module '%s' already exists", name)
	}

	template, err := m.getTemplate(templateName)
	if err != nil {
		return fmt.Errorf("template '%s' not found: %w", templateName, err)
	}

	if err := os.MkdirAll(modulePath, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	// Create config.yaml
	config := template.Config
	config.Name = name
	if config.Description == "" {
		config.Description = fmt.Sprintf("Configuration for %s", name)
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(modulePath, "config.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create install.sh
	installScript := template.Files["install.sh"]
	if installScript == "" {
		installScript = fmt.Sprintf(`#!/bin/bash

echo "Setting up %s configuration..."

# Packages are handled by DotCLI package manager integration
# Add your custom setup logic here

echo "%s setup completed!"
`, name, name)
	}

	installPath := filepath.Join(modulePath, "install.sh")
	if err := os.WriteFile(installPath, []byte(installScript), 0755); err != nil {
		return fmt.Errorf("failed to write install script: %w", err)
	}

	// Create dotfiles directory
	dotfilesDir := filepath.Join(modulePath, "dotfiles")
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create dotfiles directory: %w", err)
	}

	// Create template files
	for filename, content := range template.Files {
		if filename == "install.sh" {
			continue // Already handled
		}

		filePath := filepath.Join(dotfilesDir, filename)
		fileDir := filepath.Dir(filePath)
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fileDir, err)
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return nil
}

func (m *Manager) AddDotfileFromString(input string) error {
	parts := strings.Split(input, ":")
	if len(parts) != 3 {
		return fmt.Errorf("format should be module:source:destination")
	}

	moduleName := strings.TrimSpace(parts[0])
	source := strings.TrimSpace(parts[1])
	destination := strings.TrimSpace(parts[2])

	return m.AddDotfile(moduleName, source, destination)
}

func (m *Manager) AddDotfile(moduleName, source, destination string) error {
	modulePath := filepath.Join(m.modulesPath, moduleName)
	configPath := filepath.Join(modulePath, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("module '%s' does not exist", moduleName)
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config models.ModuleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Add new dotfile mapping
	newMapping := models.DotfileMapping{
		Source:      source,
		Destination: destination,
	}

	// Check if mapping already exists
	for _, existing := range config.Dotfiles {
		if existing.Source == source && existing.Destination == destination {
			return fmt.Errorf("dotfile mapping already exists")
		}
	}

	config.Dotfiles = append(config.Dotfiles, newMapping)

	// Write updated config
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	return nil
}

func (m *Manager) ImportDotfile(moduleName, sourcePath string) error {
	modulePath := filepath.Join(m.modulesPath, moduleName)

	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return fmt.Errorf("module '%s' does not exist", moduleName)
	}

	// Expand source path
	if strings.HasPrefix(sourcePath, "~/") {
		homeDir, _ := os.UserHomeDir()
		sourcePath = filepath.Join(homeDir, sourcePath[2:])
	}

	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Determine destination in module
	basename := filepath.Base(sourcePath)
	moduleDestPath := filepath.Join(modulePath, "dotfiles", basename)

	// Copy file to module
	if err := m.copyFile(sourcePath, moduleDestPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Add to config
	relativeSource := filepath.Join("dotfiles", basename)
	homeDir, _ := os.UserHomeDir()
	relativeDest, _ := filepath.Rel(homeDir, sourcePath)

	if err := m.AddDotfile(moduleName, relativeSource, relativeDest); err != nil {
		return fmt.Errorf("failed to add to config: %w", err)
	}

	return nil
}

func (m *Manager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

func (m *Manager) getTemplate(name string) (Template, error) {
	templates := map[string]Template{
		"basic": {
			Name:        "basic",
			Description: "Basic module template",
			Config: models.ModuleConfig{
				Dependencies: []string{},
				Packages:     models.PackageManager{},
				Dotfiles:     []models.DotfileMapping{},
			},
			Files: map[string]string{},
		},
		"shell": {
			Name:        "shell",
			Description: "Shell configuration template",
			Config: models.ModuleConfig{
				Dependencies: []string{},
				Packages: models.PackageManager{
					Brew: []string{"zsh"},
					Apt:  []string{"zsh"},
				},
				Dotfiles: []models.DotfileMapping{
					{Source: "dotfiles/.zshrc", Destination: ".zshrc"},
				},
			},
			Files: map[string]string{
				".zshrc": "# Shell configuration\nexport PATH=$HOME/.local/bin:$PATH\n",
			},
		},
		"editor": {
			Name:        "editor",
			Description: "Editor configuration template",
			Config: models.ModuleConfig{
				Dependencies: []string{},
				Packages: models.PackageManager{
					Brew: []string{"neovim"},
					Apt:  []string{"neovim"},
				},
				Dotfiles: []models.DotfileMapping{
					{Source: "dotfiles/nvim/", Destination: ".config/nvim/"},
				},
			},
			Files: map[string]string{
				"nvim/init.lua": "-- Editor configuration\nvim.opt.number = true\n",
			},
		},
		"cli-tool": {
			Name:        "cli-tool",
			Description: "CLI tool configuration template",
			Config: models.ModuleConfig{
				Dependencies: []string{},
				Packages: models.PackageManager{
					Brew: []string{"git", "curl"},
					Apt:  []string{"git", "curl"},
				},
				Dotfiles: []models.DotfileMapping{
					{Source: "dotfiles/.toolrc", Destination: ".toolrc"},
				},
			},
			Files: map[string]string{
				".toolrc": "# Tool configuration\n",
			},
		},
	}

	template, exists := templates[name]
	if !exists {
		return Template{}, fmt.Errorf("template '%s' not found. Available: basic, shell, editor, cli-tool", name)
	}

	return template, nil
}
