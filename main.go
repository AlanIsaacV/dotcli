package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AlanIsaacV/dotcli/internal/installer"
	"github.com/AlanIsaacV/dotcli/internal/manager"
	"github.com/AlanIsaacV/dotcli/internal/models"
	"github.com/AlanIsaacV/dotcli/internal/scanner"
	"github.com/AlanIsaacV/dotcli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	dotfilesPath := filepath.Join(homeDir, "dotfiles")

	if _, err := os.Stat(dotfilesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(dotfilesPath, "modules"), 0755); err != nil {
			log.Fatal("Failed to create dotfiles directory:", err)
		}
	}

	mgr := manager.New(dotfilesPath)
	scanner := scanner.New(dotfilesPath)
	modules, err := scanner.ScanModules()
	if err != nil {
		log.Fatal("Failed to scan modules:", err)
	}

	if len(modules) == 0 {
		fmt.Println("No modules found. Use 'c' in the interface to create your first module.")
	}

	model := ui.NewModel(modules, mgr)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal("Error running program:", err)
	}

	if finalModel.(ui.Model).ShouldInstall() {
		selected := finalModel.(ui.Model).GetSelected()
		forceReinstall := finalModel.(ui.Model).GetForceReinstall()

		if len(selected) > 0 {
			fmt.Printf("\n🚀 Installing %d modules...\n", len(selected))
			if forceReinstall {
				fmt.Println("⚡ Force reinstall enabled - will reinstall even if packages exist")
			}
			fmt.Println("📋 Installation order:")

			installer, err := installer.New()
			if err != nil {
				log.Fatal("Failed to create installer:", err)
			}

			installOrder, err := installer.GetInstallationOrder(modules, selected)
			if err != nil {
				log.Fatal("Failed to resolve dependencies:", err)
			}

			for i, moduleName := range installOrder {
				fmt.Printf("  %d. %s\n", i+1, moduleName)
			}
			fmt.Println()

			statusCh := make(chan models.InstallationStatus, 100)
			var installationFailed bool

			go func() {
				defer close(statusCh)

				moduleMap := make(map[string]models.ModuleConfig)
				for _, module := range modules {
					moduleMap[module.Name] = module
				}

				options := models.InstallOptions{
					ForceReinstall: forceReinstall,
				}

				for _, moduleName := range installOrder {
					if module, exists := moduleMap[moduleName]; exists {
						if err := installer.InstallModuleWithOptions(module, statusCh, options); err != nil {
							statusCh <- models.InstallationStatus{
								Module: moduleName,
								Status: fmt.Sprintf("Installation failed: %v", err),
								Error:  err,
							}
							installationFailed = true
							break
						}
					}
				}
			}()

			for status := range statusCh {
				timestamp := fmt.Sprintf("[%s]", "DOTCLI")
				if status.Error != nil {
					fmt.Printf("❌ %s %s: %s\n", timestamp, status.Module, status.Status)
				} else if status.Progress == 1.0 {
					fmt.Printf("✅ %s %s: %s\n", timestamp, status.Module, status.Status)
				} else {
					fmt.Printf("📦 %s %s: %s\n", timestamp, status.Module, status.Status)
				}
			}

			if installationFailed {
				fmt.Println("\n❌ Installation failed!")
				os.Exit(1)
			} else {
				fmt.Println("\n🎉 Installation completed successfully!")
				fmt.Println("💡 You may need to restart your terminal or reload your shell configuration.")
			}
		}
	}
}
