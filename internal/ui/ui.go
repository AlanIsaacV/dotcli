package ui

import (
	"fmt"
	"strings"

	"github.com/alan/dotcli/internal/manager"
	"github.com/alan/dotcli/internal/models"
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
	allModules     []models.ModuleConfig
}

type FormData struct {
	moduleName   string
	description  string
	template     string
	packages     []string
	dotfilePaths []string
	currentField int
	inputValue   string
}

type installProgressMsg models.InstallationStatus
type installCompleteMsg struct{}
type installErrorMsg error

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

func NewModel(modules []models.ModuleConfig, mgr *manager.Manager) Model {
	moduleStates := make([]models.ModuleState, len(modules))
	for i, module := range modules {
		moduleStates[i] = models.ModuleState{
			Config:    module,
			Selected:  false,
			Installed: false,
		}
	}

	return Model{
		modules:    moduleStates,
		selected:   make(map[string]bool),
		manager:    mgr,
		allModules: modules,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			module := m.modules[m.cursor]
			if m.selected[module.Config.Name] {
				delete(m.selected, module.Config.Name)
				m.modules[m.cursor].Selected = false
			} else {
				m.selected[module.Config.Name] = true
				m.modules[m.cursor].Selected = true
				m.autoSelectDependencies(module.Config)
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
			if m.mode == "" {
				m.mode = "create"
				m.formData = FormData{template: "basic", currentField: 0}
				m.formData.inputValue = m.formData.moduleName
			}

		case "a":
			if m.mode == "" {
				m.mode = "add"
				m.formData = FormData{currentField: 0}
				m.formData.inputValue = m.formData.moduleName
			}

		case "esc":
			if m.mode != "" {
				m.mode = ""
				m.formData = FormData{}
			}
		}

		// Handle form input
		if m.mode == "create" || m.mode == "add" {
			return m.updateForm(msg)
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
		if m.mode == "create" && m.formData.currentField > 2 {
			m.formData.currentField = 0
		} else if m.mode == "add" && m.formData.currentField > 1 {
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
				m.formData.currentField = 1
			}
		}
		m.loadCurrentField()
	case "backspace":
		if len(m.formData.inputValue) > 0 {
			m.formData.inputValue = m.formData.inputValue[:len(m.formData.inputValue)-1]
		}
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
			m.formData.moduleName = m.formData.inputValue
		case 1:
			if len(m.formData.dotfilePaths) == 0 {
				m.formData.dotfilePaths = append(m.formData.dotfilePaths, "")
			}
			m.formData.dotfilePaths[0] = m.formData.inputValue
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
			m.formData.inputValue = m.formData.moduleName
		case 1:
			if len(m.formData.dotfilePaths) > 0 {
				m.formData.inputValue = m.formData.dotfilePaths[0]
			} else {
				m.formData.inputValue = ""
			}
		}
	}
}

func (m Model) handleCreateSubmit() (tea.Model, tea.Cmd) {
	if m.formData.moduleName == "" {
		m.error = fmt.Errorf("module name is required")
		return m, nil
	}

	if err := m.manager.CreateModule(m.formData.moduleName, m.formData.template); err != nil {
		m.error = err
		return m, nil
	}

	m.mode = ""
	m.formData = FormData{}
	return m, tea.Quit
}

func (m Model) handleAddSubmit() (tea.Model, tea.Cmd) {
	if m.formData.moduleName == "" || len(m.formData.dotfilePaths) == 0 {
		m.error = fmt.Errorf("module name and dotfile path are required")
		return m, nil
	}

	parts := strings.Split(m.formData.dotfilePaths[0], ":")
	if len(parts) != 2 {
		m.error = fmt.Errorf("format should be source:destination")
		return m, nil
	}

	if err := m.manager.AddDotfile(m.formData.moduleName, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])); err != nil {
		m.error = err
		return m, nil
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

		// Show packages if any
		if !m.hasNoPackages(module.Config.Packages) {
			var pkgList []string
			if len(module.Config.Packages.Brew) > 0 {
				pkgList = append(pkgList, fmt.Sprintf("brew:%s", strings.Join(module.Config.Packages.Brew, ",")))
			}
			if len(module.Config.Packages.Apt) > 0 {
				pkgList = append(pkgList, fmt.Sprintf("apt:%s", strings.Join(module.Config.Packages.Apt, ",")))
			}
			if len(module.Config.Packages.Pacman) > 0 {
				pkgList = append(pkgList, fmt.Sprintf("pacman:%s", strings.Join(module.Config.Packages.Pacman, ",")))
			}
			if len(pkgList) > 0 {
				line += helpStyle.Render(fmt.Sprintf(" (packages: %s)", strings.Join(pkgList, ", ")))
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
		nameValue = m.formData.inputValue
	} else {
		nameLabel = "  " + nameLabel
	}
	s.WriteString(nameLabel + " " + nameValue + "\n")

	// Description field
	descLabel := "Description:"
	descValue := m.formData.description
	if m.formData.currentField == 1 {
		descLabel = selectedStyle.Render("► " + descLabel)
		descValue = m.formData.inputValue
	} else {
		descLabel = "  " + descLabel
	}
	s.WriteString(descLabel + " " + descValue + "\n")

	// Template field
	templateLabel := "Template:"
	templateValue := m.formData.template
	if m.formData.currentField == 2 {
		templateLabel = selectedStyle.Render("► " + templateLabel)
		templateValue = m.formData.inputValue
	} else {
		templateLabel = "  " + templateLabel
	}
	s.WriteString(templateLabel + " " + templateValue + "\n")

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

	// Module name field
	nameLabel := "Module Name:"
	nameValue := m.formData.moduleName
	if m.formData.currentField == 0 {
		nameLabel = selectedStyle.Render("► " + nameLabel)
		nameValue = m.formData.inputValue
	} else {
		nameLabel = "  " + nameLabel
	}
	s.WriteString(nameLabel + " " + nameValue + "\n")

	// Dotfile path field
	pathLabel := "Dotfile Path (source:destination):"
	var pathValue string
	if m.formData.currentField == 1 {
		pathLabel = selectedStyle.Render("► " + pathLabel)
		pathValue = m.formData.inputValue
	} else {
		pathLabel = "  " + pathLabel
		if len(m.formData.dotfilePaths) > 0 {
			pathValue = m.formData.dotfilePaths[0]
		}
	}
	s.WriteString(pathLabel + " " + pathValue + "\n")

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
