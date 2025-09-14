package ui

import (
	"fmt"
	"strings"

	"github.com/AlanIsaacV/dotcli/internal/manager"
	"github.com/AlanIsaacV/dotcli/internal/models"
	"github.com/AlanIsaacV/dotcli/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
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
	formData       FormData
	manager        *manager.Manager
	scanner        *scanner.Scanner
	allModules     []models.ModuleConfig
	dotfilesPath   string
}

type FormData struct {
	moduleName   string
	description  string
	template     string
	moduleChoice string
	source       string
	destination  string
	currentField int
	inputValue   string
	showingHelp  bool
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
	fieldStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == "create" || m.mode == "add" {
			return m.updateForm(msg)
		}

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
			m.mode = "create"
			m.formData = FormData{
				template:     "basic",
				currentField: 0,
			}
			m.loadCurrentField()

		case "a":
			if len(m.modules) == 0 {
				m.error = fmt.Errorf("no modules available. Create a module first")
				return m, nil
			}
			m.mode = "add"
			m.formData = FormData{
				currentField: 0,
			}
			m.loadCurrentField()

		case "esc":
			m.mode = ""
			m.formData = FormData{}
			m.error = nil
		}
	}

	return m, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.mode == "create" {
			return m.handleCreateSubmit()
		} else if m.mode == "add" {
			return m.handleAddSubmit()
		}
	case "tab", "down":
		m.saveCurrentField()
		m.formData.currentField++
		maxField := 2 // create form has 3 fields (0,1,2)
		if m.mode == "add" {
			maxField = 2 // add form has 3 fields (0,1,2)
		}
		if m.formData.currentField > maxField {
			m.formData.currentField = 0
		}
		m.loadCurrentField()
	case "shift+tab", "up":
		m.saveCurrentField()
		m.formData.currentField--
		if m.formData.currentField < 0 {
			if m.mode == "create" {
				m.formData.currentField = 2
			} else {
				m.formData.currentField = 2
			}
		}
		m.loadCurrentField()
	case "backspace":
		if len(m.formData.inputValue) > 0 {
			m.formData.inputValue = m.formData.inputValue[:len(m.formData.inputValue)-1]
		}
	case "esc":
		m.mode = ""
		m.formData = FormData{}
		m.error = nil
	default:
		if len(msg.String()) == 1 {
			m.formData.inputValue += msg.String()
		}
	}

	return m, nil
}

func (m *Model) saveCurrentField() {
	switch m.mode {
	case "create":
		switch m.formData.currentField {
		case 0:
			m.formData.moduleName = m.formData.inputValue
		case 1:
			m.formData.description = m.formData.inputValue
		case 2:
			m.formData.template = m.formData.inputValue
		}
	case "add":
		switch m.formData.currentField {
		case 0:
			m.formData.moduleChoice = m.formData.inputValue
		case 1:
			m.formData.source = m.formData.inputValue
		case 2:
			m.formData.destination = m.formData.inputValue
		}
	}
}

func (m *Model) loadCurrentField() {
	switch m.mode {
	case "create":
		switch m.formData.currentField {
		case 0:
			m.formData.inputValue = m.formData.moduleName
		case 1:
			m.formData.inputValue = m.formData.description
		case 2:
			m.formData.inputValue = m.formData.template
		}
	case "add":
		switch m.formData.currentField {
		case 0:
			m.formData.inputValue = m.formData.moduleChoice
		case 1:
			m.formData.inputValue = m.formData.source
		case 2:
			m.formData.inputValue = m.formData.destination
		}
	}
}

func (m Model) handleCreateSubmit() (tea.Model, tea.Cmd) {
	m.saveCurrentField()

	if strings.TrimSpace(m.formData.moduleName) == "" {
		m.error = fmt.Errorf("module name is required")
		return m, nil
	}

	template := m.formData.template
	if template == "" {
		template = "basic"
	}

	// Create the module
	if err := m.manager.CreateModule(m.formData.moduleName, template); err != nil {
		m.error = err
		return m, nil
	}

	// Reload modules after creation
	modules, err := m.scanner.ScanModules()
	if err != nil {
		m.error = fmt.Errorf("module created but failed to reload: %w", err)
	} else {
		// Update the model with new modules
		moduleStates := make([]models.ModuleState, len(modules))
		for i, module := range modules {
			moduleStates[i] = models.ModuleState{
				Config:   module,
				Selected: false,
			}
		}
		m.modules = moduleStates
		m.allModules = modules
		// Clear selection to prevent accidental installations
		m.selected = make(map[string]bool)
		m.cursor = 0
	}

	m.mode = ""
	m.formData = FormData{}
	return m, nil
}

func (m Model) handleAddSubmit() (tea.Model, tea.Cmd) {
	m.saveCurrentField()

	if strings.TrimSpace(m.formData.moduleChoice) == "" {
		m.error = fmt.Errorf("module name is required")
		return m, nil
	}

	if strings.TrimSpace(m.formData.source) == "" {
		m.error = fmt.Errorf("source path is required")
		return m, nil
	}

	if strings.TrimSpace(m.formData.destination) == "" {
		m.error = fmt.Errorf("destination path is required")
		return m, nil
	}

	// Add the dotfile
	if err := m.manager.AddDotfile(m.formData.moduleChoice, m.formData.source, m.formData.destination); err != nil {
		m.error = err
		return m, nil
	}

	// Reload modules after adding dotfile
	modules, err := m.scanner.ScanModules()
	if err != nil {
		m.error = fmt.Errorf("dotfile added but failed to reload: %w", err)
	} else {
		// Update the model with reloaded modules
		moduleStates := make([]models.ModuleState, len(modules))
		for i, module := range modules {
			moduleStates[i] = models.ModuleState{
				Config:   module,
				Selected: false,
			}
		}
		m.modules = moduleStates
		m.allModules = modules
	}

	m.mode = ""
	m.formData = FormData{}
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

	if m.mode == "create" {
		return m.renderCreateForm()
	}

	if m.mode == "add" {
		return m.renderAddForm()
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
			line += helpStyle.Render(fmt.Sprintf(" (deps: %s)", deps))
		}

		// Show packages count
		if !m.hasNoPackages(module.Config.Packages) {
			totalPkgs := len(module.Config.Packages.Brew) + len(module.Config.Packages.Apt) +
				len(module.Config.Packages.Pacman) + len(module.Config.Packages.Yum) + len(module.Config.Packages.Snap)
			if totalPkgs > 0 {
				line += helpStyle.Render(fmt.Sprintf(" (%d packages)", totalPkgs))
			}
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
	s.WriteString(helpStyle.Render("↑/↓: navigate • space: select • c: create • a: add dotfile • enter: install" + forceText + " • q: quit"))

	if len(m.selected) > 0 {
		s.WriteString("\n\n")
		s.WriteString(statusStyle.Render(fmt.Sprintf("Selected %d modules", len(m.selected))))
		if m.forceReinstall {
			s.WriteString(" " + errorStyle.Render("(force reinstall)"))
		}
	}

	return s.String()
}

func (m Model) renderCreateForm() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Create New Module"))
	s.WriteString("\n\n")

	// Module name field
	nameLabel := "Module Name:"
	nameValue := m.formData.moduleName
	if m.formData.currentField == 0 {
		nameLabel = selectedStyle.Render("► " + nameLabel)
		nameValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		nameLabel = "  " + nameLabel
		nameValue = fieldStyle.Render(nameValue)
	}
	s.WriteString(nameLabel + " " + nameValue + "\n")

	// Description field
	descLabel := "Description:"
	descValue := m.formData.description
	if m.formData.currentField == 1 {
		descLabel = selectedStyle.Render("► " + descLabel)
		descValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		descLabel = "  " + descLabel
		descValue = fieldStyle.Render(descValue)
	}
	s.WriteString(descLabel + " " + descValue + "\n")

	// Template field
	templateLabel := "Template:"
	templateValue := m.formData.template
	if m.formData.currentField == 2 {
		templateLabel = selectedStyle.Render("► " + templateLabel)
		templateValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		templateLabel = "  " + templateLabel
		templateValue = fieldStyle.Render(templateValue)
	}
	s.WriteString(templateLabel + " " + templateValue + "\n")

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Available templates: basic, shell, editor, cli-tool"))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("tab/↓: next field • shift+tab/↑: prev • enter: create • esc: cancel"))

	if m.error != nil {
		s.WriteString("\n\n")
		s.WriteString(errorStyle.Render("Error: " + m.error.Error()))
	}

	return s.String()
}

func (m Model) renderAddForm() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Add Dotfile to Module"))
	s.WriteString("\n\n")

	// Available modules display
	if m.formData.currentField == 0 {
		s.WriteString(helpStyle.Render("Available modules: "))
		var moduleNames []string
		for _, module := range m.modules {
			moduleNames = append(moduleNames, module.Config.Name)
		}
		s.WriteString(helpStyle.Render(strings.Join(moduleNames, ", ")))
		s.WriteString("\n\n")
	}

	// Module name field
	nameLabel := "Module Name:"
	nameValue := m.formData.moduleChoice
	if m.formData.currentField == 0 {
		nameLabel = selectedStyle.Render("► " + nameLabel)
		nameValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		nameLabel = "  " + nameLabel
		nameValue = fieldStyle.Render(nameValue)
	}
	s.WriteString(nameLabel + " " + nameValue + "\n")

	// Source path field
	sourceLabel := "Source Path:"
	sourceValue := m.formData.source
	if m.formData.currentField == 1 {
		sourceLabel = selectedStyle.Render("► " + sourceLabel)
		sourceValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		sourceLabel = "  " + sourceLabel
		sourceValue = fieldStyle.Render(sourceValue)
	}
	s.WriteString(sourceLabel + " " + sourceValue + "\n")

	// Destination path field
	destLabel := "Destination Path:"
	destValue := m.formData.destination
	if m.formData.currentField == 2 {
		destLabel = selectedStyle.Render("► " + destLabel)
		destValue = cursorStyle.Render(m.formData.inputValue + "█")
	} else {
		destLabel = "  " + destLabel
		destValue = fieldStyle.Render(destValue)
	}
	s.WriteString(destLabel + " " + destValue + "\n")

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Example: dotfiles/.bashrc → .bashrc"))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("tab/↓: next field • shift+tab/↑: prev • enter: add • esc: cancel"))

	if m.error != nil {
		s.WriteString("\n\n")
		s.WriteString(errorStyle.Render("Error: " + m.error.Error()))
	}

	return s.String()
}

func (m Model) hasNoPackages(pm models.PackageManager) bool {
	return len(pm.Brew) == 0 && len(pm.Apt) == 0 && len(pm.Pacman) == 0 && len(pm.Yum) == 0 && len(pm.Snap) == 0
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
