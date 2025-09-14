package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlanIsaacV/dotcli/internal/models"
	"gopkg.in/yaml.v3"
)

type Scanner struct {
	dotfilesPath string
}

func New(dotfilesPath string) *Scanner {
	return &Scanner{
		dotfilesPath: dotfilesPath,
	}
}

func (s *Scanner) ScanModules() ([]models.ModuleConfig, error) {
	modulesPath := filepath.Join(s.dotfilesPath, "modules")

	if _, err := os.Stat(modulesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("modules directory not found at %s", modulesPath)
	}

	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules directory: %w", err)
	}

	var modules []models.ModuleConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modulePath := filepath.Join(modulesPath, entry.Name())
		module, err := s.loadModule(modulePath)
		if err != nil {
			fmt.Printf("Warning: failed to load module %s: %v\n", entry.Name(), err)
			continue
		}

		modules = append(modules, module)
	}

	return modules, nil
}

func (s *Scanner) loadModule(modulePath string) (models.ModuleConfig, error) {
	configPath := filepath.Join(modulePath, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return models.ModuleConfig{}, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var config models.ModuleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return models.ModuleConfig{}, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	config.Path = modulePath
	return config, nil
}

func (s *Scanner) ValidateModule(modulePath string) error {
	configPath := filepath.Join(modulePath, "config.yaml")
	installPath := filepath.Join(modulePath, "install.sh")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config.yaml not found in module")
	}

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return fmt.Errorf("install.sh not found in module")
	}

	return nil
}
