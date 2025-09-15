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

func (m *Manager) CreateModuleWithConfig(name, description string, dependencies []string, packages models.PackageManager, commands []models.InstallCommand) error {
	modulePath := filepath.Join(m.modulesPath, name)

	if _, err := os.Stat(modulePath); !os.IsNotExist(err) {
		return fmt.Errorf("module '%s' already exists", name)
	}

	if err := os.MkdirAll(modulePath, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	// Create config.yaml with provided configuration
	config := models.ModuleConfig{
		Name:         name,
		Description:  description,
		Dependencies: dependencies,
		Packages:     packages,
		Commands:     commands,
		Dotfiles:     []models.DotfileMapping{},
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(modulePath, "config.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Only create install.sh if there are actual custom commands (not empty)
	hasCustomCommands := false
	for _, cmd := range commands {
		if strings.TrimSpace(cmd.Command) != "" {
			hasCustomCommands = true
			break
		}
	}

	if hasCustomCommands {
		installScript := fmt.Sprintf(`#!/bin/bash

echo "Setting up %s configuration..."

# Packages are handled by DotCLI package manager integration
# Custom commands will be executed by DotCLI

echo "%s setup completed!"
`, name, name)

		installPath := filepath.Join(modulePath, "install.sh")
		if err := os.WriteFile(installPath, []byte(installScript), 0755); err != nil {
			return fmt.Errorf("failed to write install script: %w", err)
		}
	}

	// Create dotfiles directory
	dotfilesDir := filepath.Join(modulePath, "dotfiles")
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create dotfiles directory: %w", err)
	}

	return nil
}

func (m *Manager) UpdateModule(name, description string, dependencies []string, packages models.PackageManager, commands []models.InstallCommand) error {
	modulePath := filepath.Join(m.modulesPath, name)
	configPath := filepath.Join(modulePath, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("module '%s' does not exist", name)
	}

	// Read existing config to preserve dotfiles
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var existingConfig models.ModuleConfig
	if err := yaml.Unmarshal(data, &existingConfig); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Update config with new values, preserving dotfiles
	config := models.ModuleConfig{
		Name:         name,
		Description:  description,
		Dependencies: dependencies,
		Packages:     packages,
		Commands:     commands,
		Dotfiles:     existingConfig.Dotfiles, // Preserve existing dotfiles
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
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
	sourcePath = m.expandPath(sourcePath)

	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Check if source is already a symlink pointing to our module
	if linkTarget, err := os.Readlink(sourcePath); err == nil {
		// It's already a symlink, check if it points to our module
		if strings.Contains(linkTarget, modulePath) {
			return fmt.Errorf("file is already managed by this module")
		}
	}

	// Determine destination in module
	basename := filepath.Base(sourcePath)
	moduleDestPath := filepath.Join(modulePath, "dotfiles", basename)

	// Check if file already exists in module
	if _, err := os.Stat(moduleDestPath); err == nil {
		return fmt.Errorf("file already exists in module: %s", basename)
	}

	// Copy file to module
	if err := m.copyFile(sourcePath, moduleDestPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Remove original file
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	// Create symlink from original location to module
	absModuleDestPath, err := filepath.Abs(moduleDestPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := os.Symlink(absModuleDestPath, sourcePath); err != nil {
		// Try to restore original file if symlink creation fails
		m.copyFile(absModuleDestPath, sourcePath)
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Add to config
	relativeSource := filepath.Join("dotfiles", basename)
	homeDir, _ := os.UserHomeDir()

	// Calculate destination as relative to home (e.g., ".bashrc" instead of full path)
	var relativeDest string
	if strings.HasPrefix(sourcePath, homeDir) {
		relativeDest, _ = filepath.Rel(homeDir, sourcePath)
	} else {
		// If not in home directory, use the basename
		relativeDest = basename
	}

	if err := m.AddDotfile(moduleName, relativeSource, relativeDest); err != nil {
		return fmt.Errorf("failed to add to config: %w", err)
	}

	return nil
}

func (m *Manager) expandPath(path string) string {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if we can't get home dir
	}

	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}

	// Handle exact ~ (home directory)
	if path == "~" {
		return homeDir
	}

	// Handle $HOME expansion
	if strings.HasPrefix(path, "$HOME/") {
		return filepath.Join(homeDir, path[6:])
	}

	// Handle exact $HOME
	if path == "$HOME" {
		return homeDir
	}

	// Expand environment variables
	expanded := os.ExpandEnv(path)

	// If path is relative and doesn't start with . or /, assume it's relative to home
	if !filepath.IsAbs(expanded) && !strings.HasPrefix(expanded, ".") && !strings.HasPrefix(expanded, "/") {
		return filepath.Join(homeDir, expanded)
	}

	return expanded
}

func (m *Manager) ImportDotfileWithDestination(moduleName, sourcePath, destinationPath string) error {
	modulePath := filepath.Join(m.modulesPath, moduleName)

	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return fmt.Errorf("module '%s' does not exist", moduleName)
	}

	// Expand source path
	sourcePath = m.expandPath(sourcePath)

	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Check if source is already a symlink pointing to our module
	if linkTarget, err := os.Readlink(sourcePath); err == nil {
		if strings.Contains(linkTarget, modulePath) {
			return fmt.Errorf("file is already managed by this module")
		}
	}

	// Determine destination in module based on provided destination path
	moduleDestPath := filepath.Join(modulePath, "dotfiles", destinationPath)

	// Check if file already exists in module
	if _, err := os.Stat(moduleDestPath); err == nil {
		return fmt.Errorf("file already exists in module: %s", destinationPath)
	}

	// Create directory structure if needed
	moduleDestDir := filepath.Dir(moduleDestPath)
	if err := os.MkdirAll(moduleDestDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory structure: %w", err)
	}

	// Handle both files and directories
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.IsDir() {
		// Copy directory recursively
		if err := m.copyDir(sourcePath, moduleDestPath); err != nil {
			return fmt.Errorf("failed to copy directory: %w", err)
		}
	} else {
		// Copy single file
		if err := m.copyFile(sourcePath, moduleDestPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Remove original file/directory
	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original: %w", err)
	}

	// Create symlink from original location to module
	absModuleDestPath, err := filepath.Abs(moduleDestPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := os.Symlink(absModuleDestPath, sourcePath); err != nil {
		// Try to restore original file/directory if symlink creation fails
		if fileInfo.IsDir() {
			m.copyDir(absModuleDestPath, sourcePath)
		} else {
			m.copyFile(absModuleDestPath, sourcePath)
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Calculate relative destination path for config
	homeDir, _ := os.UserHomeDir()

	// Calculate destination as relative to home (e.g., ".bashrc" instead of full path)
	var relativeDest string
	if strings.HasPrefix(sourcePath, homeDir) {
		relativeDest, _ = filepath.Rel(homeDir, sourcePath)
	} else {
		// If not in home directory, use the basename
		relativeDest = filepath.Base(sourcePath)
	}
	relativeSource := filepath.Join("dotfiles", destinationPath)

	// Add to config
	if err := m.AddDotfile(moduleName, relativeSource, relativeDest); err != nil {
		return fmt.Errorf("failed to add to config: %w", err)
	}

	return nil
}

func (m *Manager) copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := m.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := m.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
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
				Commands:     []models.InstallCommand{},
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
					Common: []string{"zsh"},
				},
				Commands: []models.InstallCommand{},
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
					Common: []string{},
					Specific: []models.SpecificPackage{
						{Name: "neovim", Manager: "brew"},
						{Name: "neovim", Manager: "apt"},
					},
				},
				Commands: []models.InstallCommand{},
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
					Common: []string{"git", "curl"},
				},
				Commands: []models.InstallCommand{},
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
