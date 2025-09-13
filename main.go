package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alan/dotcli/internal/installer"
	"github.com/alan/dotcli/internal/manager"
	"github.com/alan/dotcli/internal/models"
	"github.com/alan/dotcli/internal/scanner"
	"github.com/alan/dotcli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func printHelp() {
	fmt.Println("DotCLI - Enhanced Dotfiles Manager")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  dotcli                                     # Interactive module selection")
	fmt.Println("  dotcli -create <name>                      # Create new module")
	fmt.Println("  dotcli -create <name> -template <type>     # Create with template")
	fmt.Println("  dotcli -add <module:source:dest>           # Add dotfile mapping")
	fmt.Println("  dotcli -import <module:path>               # Import existing file")
	fmt.Println("  dotcli -list                               # List modules")
	fmt.Println("  dotcli -templates                          # List templates")
	fmt.Println("  dotcli -help                               # Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dotcli -create nvim -template editor")
	fmt.Println("  dotcli -add nvim:dotfiles/init.lua:.config/nvim/init.lua")
	fmt.Println("  dotcli -import zsh:~/.zshrc")
	fmt.Println()
	fmt.Println("Templates: basic, shell, editor, cli-tool")
}

func main() {
	var (
		create     = flag.String("create", "", "Create new module with name")
		addDotfile = flag.String("add", "", "Add dotfile to existing module (format: module:source:destination)")
		importFile = flag.String("import", "", "Import existing file to module (format: module:path)")
		template   = flag.String("template", "basic", "Template to use when creating module")
		listCmd    = flag.Bool("list", false, "List available modules")
		templates  = flag.Bool("templates", false, "List available templates")
		editModule = flag.String("edit", "", "Edit module config in $EDITOR")
	)

	// Check for help before parsing
	for _, arg := range os.Args[1:] {
		if arg == "-help" || arg == "--help" || arg == "-h" {
			printHelp()
			return
		}
	}

	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	dotfilesPath := filepath.Join(homeDir, ".dotfiles")

	// Create dotfiles directory if it doesn't exist
	if _, err := os.Stat(dotfilesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(dotfilesPath, "modules"), 0755); err != nil {
			log.Fatal("Failed to create dotfiles directory:", err)
		}
		fmt.Printf("Created dotfiles directory at %s\n", dotfilesPath)
	}

	mgr := manager.New(dotfilesPath)

	// Handle commands
	switch {

	case *create != "":
		if err := mgr.CreateModule(*create, *template); err != nil {
			log.Fatal("Failed to create module:", err)
		}
		fmt.Printf("✅ Module '%s' created successfully\n", *create)
		return

	case *addDotfile != "":
		if err := mgr.AddDotfileFromString(*addDotfile); err != nil {
			log.Fatal("Failed to add dotfile:", err)
		}
		fmt.Println("✅ Dotfile added successfully")
		return

	case *importFile != "":
		if err := mgr.ImportDotfileFromString(*importFile); err != nil {
			log.Fatal("Failed to import file:", err)
		}
		fmt.Println("✅ File imported successfully")
		return

	case *templates:
		mgr.ListTemplates()
		return

	case *editModule != "":
		if err := mgr.EditModule(*editModule); err != nil {
			log.Fatal("Failed to edit module:", err)
		}
		return

	case *listCmd:
		scanner := scanner.New(dotfilesPath)
		modules, err := scanner.ScanModules()
		if err != nil {
			log.Fatal("Failed to scan modules:", err)
		}

		fmt.Println("Available modules:")
		for _, module := range modules {
			fmt.Printf("  📁 %s - %s\n", module.Name, module.Description)
		}
		return
	}

	// Default: run interactive UI
	scanner := scanner.New(dotfilesPath)
	modules, err := scanner.ScanModules()
	if err != nil {
		log.Fatal("Failed to scan modules:", err)
	}

	if len(modules) == 0 {
		fmt.Println("No modules found in", filepath.Join(dotfilesPath, "modules"))
		fmt.Println()
		fmt.Println("🚀 Get started:")
		fmt.Println("  dotcli -create <name>                    # Create basic module")
		fmt.Println("  dotcli -create <name> -template shell    # Create shell config")
		fmt.Println("  dotcli -templates                        # Show all templates")
		fmt.Println("  dotcli -help                             # Show detailed help")
		return
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
					SkipInstalled:  false,
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
