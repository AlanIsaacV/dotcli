package ui

import (
	"fmt"
	"strings"

	"github.com/AlanIsaacV/dotcli/internal/manager"
	"github.com/AlanIsaacV/dotcli/internal/models"
	"github.com/AlanIsaacV/dotcli/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	modules        []models.ModuleState
	cursor         int
	selected       map[string]bool
	forceReinstall bool
	quitting       bool
	shouldInstall  bool
	error          error
	mode           string
	form           *huh.Form
	manager        *manager.Manager
	scanner        *scanner.Scanner
	allModules     []models.ModuleConfig
	dotfilesPath   string
	formData       FormData
}

type FormData struct {
	moduleName          string
	description         string
	dependencies        []string
	commonPackages      string
	hasSpecificPackages bool
	specificPackages    []SpecificPackageForm
	moduleChoice        string
	source              string
	destination         string
	isEditing           bool
	editModule          *models.ModuleConfig
}

type SpecificPackageForm struct {
	Name    string
	Manager string
}

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

func NewModel(modules []models.ModuleConfig, mgr *manager.Manager, dotfilesPath string) Model {
	moduleStates := make([]models.ModuleState, len(modules))
	for i, module := range modules {
		moduleStates[i] = models.ModuleState{
			Config:   module,
			Selected: false,
		}
	}

	return Model{
		modules:      moduleStates,
		selected:     make(map[string]bool),
		manager:      mgr,
		scanner:      scanner.New(dotfilesPath),
		allModules:   modules,
		dotfilesPath: dotfilesPath,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			if m.form.State == huh.StateCompleted {
				return m.handleFormCompletion()
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.modules)-1 {
				m.cursor++
			}

		case " ":
			if len(m.modules) > 0 {
				module := m.modules[m.cursor]
				if m.selected[module.Config.Name] {
					delete(m.selected, module.Config.Name)
					m.modules[m.cursor].Selected = false
				} else {
					m.selected[module.Config.Name] = true
					m.modules[m.cursor].Selected = true
					m.autoSelectDependencies(module.Config)
				}
			}

		case "enter":
			if len(m.selected) > 0 {
				m.shouldInstall = true
				m.quitting = true
				return m, tea.Quit
			}

		case "f":
			m.forceReinstall = !m.forceReinstall

		case "c":
			return m.createModuleForm()

		case "e":
			if len(m.modules) > 0 && m.cursor < len(m.modules) {
				return m.editModuleForm(m.modules[m.cursor].Config)
			} else if len(m.modules) == 0 {
				m.error = fmt.Errorf("no modules available to edit")
			}

		case "a":
			return m.addDotfileForm()

		case "esc":
			if m.form != nil {
				m.form = nil
				m.mode = ""
				m.error = nil
			}
		}
	}

	return m, nil
}

func (m Model) createModuleForm() (tea.Model, tea.Cmd) {
	m.formData = FormData{
		isEditing: false,
	}

	// Build list of available modules for dependencies
	var moduleOptions []huh.Option[string]
	for _, module := range m.modules {
		moduleOptions = append(moduleOptions, huh.NewOption(module.Config.Name, module.Config.Name))
	}

	var dependencyGroup *huh.Group
	if len(moduleOptions) > 0 {
		dependencyGroup = huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Dependencies").
				Description("Select modules that must be installed before this one").
				Options(moduleOptions...).
				Value(&m.formData.dependencies),
		)
	} else {
		dependencyGroup = huh.NewGroup(
			huh.NewNote().
				Title("Dependencies").
				Description("No existing modules available. Create other modules first to set up dependencies."),
		)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module Name").
				Description("Enter a unique identifier for your module (e.g., nvim, shell, git)").
				Value(&m.formData.moduleName).
				Validate(func(str string) error {
					trimmed := strings.TrimSpace(str)
					if trimmed == "" {
						return fmt.Errorf("module name is required")
					}
					// Check for conflicts with existing modules
					for _, module := range m.modules {
						if module.Config.Name == trimmed {
							return fmt.Errorf("module '%s' already exists", trimmed)
						}
					}
					return nil
				}),

			huh.NewInput().
				Title("Description").
				Description("Brief description of what this module configures (optional)").
				Value(&m.formData.description),
		),

		dependencyGroup,

		huh.NewGroup(
			huh.NewInput().
				Title("Common Packages").
				Description("Packages available in both brew and apt with same name (e.g., git, curl, vim, tmux)").
				Value(&m.formData.commonPackages),

			huh.NewConfirm().
				Title("Add Specific Packages?").
				Description("Need packages with different names on brew vs apt? (e.g., neovim vs nvim)").
				Affirmative("Yes, add specific").
				Negative("No, common only").
				Value(&m.formData.hasSpecificPackages),
		),
	).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "create"

	return m, m.form.Init()
}

func (m Model) editModuleForm(module models.ModuleConfig) (tea.Model, tea.Cmd) {
	m.formData = FormData{
		moduleName:     module.Name,
		description:    module.Description,
		dependencies:   module.Dependencies,
		commonPackages: strings.Join(module.Packages.Common, ", "),
		isEditing:      true,
		editModule:     &module,
	}

	// Convert specific packages to form format
	for _, pkg := range module.Packages.Specific {
		m.formData.specificPackages = append(m.formData.specificPackages, SpecificPackageForm{
			Name:    pkg.Name,
			Manager: pkg.Manager,
		})
	}

	// Build list of available modules for dependencies (excluding current module)
	var moduleOptions []huh.Option[string]
	for _, mod := range m.modules {
		if mod.Config.Name != module.Name {
			moduleOptions = append(moduleOptions, huh.NewOption(mod.Config.Name, mod.Config.Name))
		}
	}

	var dependencyGroup *huh.Group
	if len(moduleOptions) > 0 {
		dependencyGroup = huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Dependencies").
				Description("Select modules that must be installed before this one").
				Options(moduleOptions...).
				Value(&m.formData.dependencies),
		)
	} else {
		dependencyGroup = huh.NewGroup(
			huh.NewNote().
				Title("Dependencies").
				Description("No other modules available as dependencies"),
		)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Editing Module: "+module.Name).
				Description("Modify the configuration for this module"),

			huh.NewInput().
				Title("Description").
				Description("Brief description of what this module does").
				Value(&m.formData.description),
		),

		dependencyGroup,

		huh.NewGroup(
			huh.NewInput().
				Title("Common Packages").
				Description("Packages available in both brew and apt with same name").
				Value(&m.formData.commonPackages),
		),
	).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "edit"

	return m, m.form.Init()
}

func (m Model) addDotfileForm() (tea.Model, tea.Cmd) {
	if len(m.modules) == 0 {
		m.error = fmt.Errorf("no modules available. Create a module first")
		return m, nil
	}

	m.formData = FormData{}

	var moduleOptions []huh.Option[string]
	for _, module := range m.modules {
		moduleOptions = append(moduleOptions, huh.NewOption(module.Config.Name, module.Config.Name))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Module").
				Description("Choose which module to add the dotfile to").
				Options(moduleOptions...).
				Value(&m.formData.moduleChoice),

			huh.NewInput().
				Title("Source Path").
				Description("Path within the module (e.g., dotfiles/.bashrc)").
				Value(&m.formData.source).
				Validate(func(str string) error {
					if strings.TrimSpace(str) == "" {
						return fmt.Errorf("source path is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Destination Path").
				Description("Target path in home directory (e.g., .bashrc)").
				Value(&m.formData.destination).
				Validate(func(str string) error {
					if strings.TrimSpace(str) == "" {
						return fmt.Errorf("destination path is required")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "add"

	return m, m.form.Init()
}

func (m Model) handleFormCompletion() (tea.Model, tea.Cmd) {
	switch m.mode {
	case "create":
		return m.handleCreateModule()
	case "edit":
		return m.handleEditModule()
	case "add":
		return m.handleAddDotfile()
	}
	return m, nil
}

func (m Model) handleCreateModule() (tea.Model, tea.Cmd) {
	// Dependencies are already in the correct format from multi-select
	dependencies := m.formData.dependencies

	// Parse common packages
	var commonPackages []string
	if strings.TrimSpace(m.formData.commonPackages) != "" {
		for _, pkg := range strings.Split(m.formData.commonPackages, ",") {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" {
				commonPackages = append(commonPackages, pkg)
			}
		}
	}

	// Convert specific packages
	var specificPackages []models.SpecificPackage
	for _, pkg := range m.formData.specificPackages {
		if pkg.Name != "" && pkg.Manager != "" {
			specificPackages = append(specificPackages, models.SpecificPackage{
				Name:    pkg.Name,
				Manager: pkg.Manager,
			})
		}
	}

	packages := models.PackageManager{
		Common:   commonPackages,
		Specific: specificPackages,
	}

	// Create the module
	if err := m.manager.CreateModuleWithConfig(m.formData.moduleName, m.formData.description, dependencies, packages); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) handleEditModule() (tea.Model, tea.Cmd) {
	// Dependencies are already in the correct format from multi-select
	dependencies := m.formData.dependencies

	// Parse common packages
	var commonPackages []string
	if strings.TrimSpace(m.formData.commonPackages) != "" {
		for _, pkg := range strings.Split(m.formData.commonPackages, ",") {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" {
				commonPackages = append(commonPackages, pkg)
			}
		}
	}

	// Convert specific packages
	var specificPackages []models.SpecificPackage
	for _, pkg := range m.formData.specificPackages {
		if pkg.Name != "" && pkg.Manager != "" {
			specificPackages = append(specificPackages, models.SpecificPackage{
				Name:    pkg.Name,
				Manager: pkg.Manager,
			})
		}
	}

	packages := models.PackageManager{
		Common:   commonPackages,
		Specific: specificPackages,
	}

	// Update the module
	if err := m.manager.UpdateModule(m.formData.moduleName, m.formData.description, dependencies, packages); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) handleAddDotfile() (tea.Model, tea.Cmd) {
	if err := m.manager.AddDotfile(m.formData.moduleChoice, m.formData.source, m.formData.destination); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) reloadModules() (tea.Model, tea.Cmd) {
	modules, err := m.scanner.ScanModules()
	if err != nil {
		m.error = fmt.Errorf("operation completed but failed to reload: %w", err)
	} else {
		moduleStates := make([]models.ModuleState, len(modules))
		for i, module := range modules {
			moduleStates[i] = models.ModuleState{
				Config:   module,
				Selected: false,
			}
		}
		m.modules = moduleStates
		m.allModules = modules
		m.selected = make(map[string]bool)

		// Keep cursor in bounds
		if m.cursor >= len(m.modules) {
			m.cursor = len(m.modules) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}

	m.form = nil
	m.mode = ""
	return m, nil
}

func (m *Model) autoSelectDependencies(module models.ModuleConfig) {
	for _, dep := range module.Dependencies {
		if !m.selected[dep] {
			m.selected[dep] = true
			for i, modState := range m.modules {
				if modState.Config.Name == dep {
					m.modules[i].Selected = true
					m.autoSelectDependencies(modState.Config)
					break
				}
			}
		}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.form != nil {
		return m.form.View()
	}

	var s strings.Builder

	s.WriteString(titleStyle.Render("DotCLI - Dotfiles Manager"))
	s.WriteString("\n\n")

	if m.error != nil {
		s.WriteString(errorStyle.Render("Error: " + m.error.Error()))
		s.WriteString("\n\n")
		m.error = nil // Clear error after displaying
	}

	if len(m.modules) == 0 {
		s.WriteString("No modules found. Press 'c' to create your first module.\n\n")
		s.WriteString(helpStyle.Render("c: create module • q: quit"))
		return s.String()
	}

	s.WriteString("Select modules to install:\n\n")

	for i, module := range m.modules {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if module.Selected {
			checked = "✓"
		}

		line := fmt.Sprintf("%s [%s] %s", cursor, checked, module.Config.Name)
		if module.Config.Description != "" {
			line += " - " + module.Config.Description
		}

		if module.Selected {
			line = selectedStyle.Render(line)
		}

		if len(module.Config.Dependencies) > 0 {
			deps := strings.Join(module.Config.Dependencies, ", ")
			line += helpStyle.Render(fmt.Sprintf(" (requires: %s)", deps))
		}

		// Show packages count
		totalPkgs := len(module.Config.Packages.Common) + len(module.Config.Packages.Specific)
		if totalPkgs > 0 {
			line += helpStyle.Render(fmt.Sprintf(" (%d packages)", totalPkgs))
		}

		s.WriteString(line + "\n")
	}

	s.WriteString("\n")
	forceText := ""
	if m.forceReinstall {
		forceText = " • f: force ON"
	} else {
		forceText = " • f: force OFF"
	}
	s.WriteString(helpStyle.Render("↑/↓: navigate • space: select • c: create • e: edit • a: add dotfile • enter: install" + forceText + " • q: quit"))

	if len(m.selected) > 0 {
		s.WriteString("\n\n")
		s.WriteString(statusStyle.Render(fmt.Sprintf("Selected %d modules", len(m.selected))))
		if m.forceReinstall {
			s.WriteString(" " + errorStyle.Render("(force reinstall)"))
		}
	}

	return s.String()
}

func (m Model) ShouldInstall() bool {
	return m.shouldInstall && len(m.selected) > 0
}

func (m Model) GetSelected() []string {
	var selected []string
	for name := range m.selected {
		selected = append(selected, name)
	}
	return selected
}

func (m Model) GetForceReinstall() bool {
	return m.forceReinstall
}
