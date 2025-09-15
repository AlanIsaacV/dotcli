package ui

import (
	"fmt"
	"strings"

	"github.com/AlanIsaacV/dotcli/internal/manager"
	"github.com/AlanIsaacV/dotcli/internal/models"
	"github.com/AlanIsaacV/dotcli/internal/scanner"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
)

type Model struct {
	list           list.Model
	selected       map[string]bool
	exportMode     bool
	quitting       bool
	shouldInstall  bool
	error          error
	mode           string
	form           *huh.Form
	manager        *manager.Manager
	scanner        *scanner.Scanner
	allModules     []models.ModuleConfig
	dotfilesPath   string
	createFormData *CreateFormData
	editFormData   *EditFormData
	addFormData    *AddFormData
	importFormData *ImportFormData
	keys           *listKeyMap
	delegateKeys   *delegateKeyMap
}

type moduleItem struct {
	module   models.ModuleConfig
	selected bool
}

func (i moduleItem) Title() string {
	prefix := "  "
	if i.selected {
		prefix = "✓ "
	}
	title := prefix + i.module.Name
	if i.module.Description != "" {
		title += " - " + i.module.Description
	}
	return title
}

func (i moduleItem) Description() string {
	var parts []string

	if len(i.module.Dependencies) > 0 {
		deps := strings.Join(i.module.Dependencies, ", ")
		parts = append(parts, fmt.Sprintf("requires: %s", deps))
	}

	totalPkgs := len(i.module.Packages.Common) + len(i.module.Packages.Specific)
	if totalPkgs > 0 {
		parts = append(parts, fmt.Sprintf("%d packages", totalPkgs))
	}

	return strings.Join(parts, " • ")
}

func (i moduleItem) FilterValue() string {
	return i.module.Name + " " + i.module.Description
}

type delegateKeyMap struct {
	choose     key.Binding
	edit       key.Binding
	add        key.Binding
	importFile key.Binding
	export     key.Binding
}

func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
		d.edit,
		d.add,
		d.importFile,
		d.export,
	}
}

func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.choose,
			d.edit,
		},
		{
			d.add,
			d.importFile,
			d.export,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add dotfile"),
		),
		importFile: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import"),
		),
		export: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "toggle export"),
		),
	}
}

type listKeyMap struct {
	create  key.Binding
	install key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create module"),
		),
		install: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "install"),
		),
	}
}

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		if _, ok := m.SelectedItem().(moduleItem); !ok {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.choose):
				// This will be handled in the main model
				return nil
			case key.Matches(msg, keys.edit):
				// This will be handled in the main model
				return nil
			case key.Matches(msg, keys.add):
				// This will be handled in the main model
				return nil
			case key.Matches(msg, keys.importFile):
				// This will be handled in the main model
				return nil
			case key.Matches(msg, keys.export):
				// This will be handled in the main model
				return nil
			}
		}

		return nil
	}

	d.ShortHelpFunc = func() []key.Binding {
		return keys.ShortHelp()
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return keys.FullHelp()
	}

	return d
}

type CreateFormData struct {
	ModuleName       *string
	Description      *string
	Dependencies     *[]string
	CommonPackages   *string
	SpecificPackages []SpecificPackageForm
	CommonCommands   *string
	SpecificCommands []SpecificCommandForm
}

type EditFormData struct {
	ModuleName       string
	Description      *string
	Dependencies     *[]string
	CommonPackages   *string
	CommonCommands   *string
	SpecificPackages []SpecificPackageForm
	SpecificCommands []SpecificCommandForm
}

type AddFormData struct {
	ModuleChoice *string
	Source       *string
	Destination  *string
}

type ImportFormData struct {
	ModuleChoice    *string
	SourcePath      *string
	DestinationPath *string
}

type SpecificPackageForm struct {
	Name    string
	Manager string
}

type SpecificCommandForm struct {
	Command string
	OS      string
}

func NewModel(modules []models.ModuleConfig, mgr *manager.Manager, dotfilesPath string) Model {
	items := make([]list.Item, len(modules))
	for i, module := range modules {
		items[i] = moduleItem{
			module:   module,
			selected: false,
		}
	}

	delegateKeys := newDelegateKeyMap()
	listKeys := newListKeyMap()

	delegate := newItemDelegate(delegateKeys)
	l := list.New(items, delegate, 0, 0)
	l.Title = "DotCLI - Dotfiles Manager"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.create,
			listKeys.install,
		}
	}

	return Model{
		list:         l,
		selected:     make(map[string]bool),
		manager:      mgr,
		scanner:      scanner.New(dotfilesPath),
		allModules:   modules,
		dotfilesPath: dotfilesPath,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) getCurrentModule() *models.ModuleConfig {
	if item, ok := m.list.SelectedItem().(moduleItem); ok {
		return &item.module
	}
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
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.delegateKeys.choose):
			if item, ok := m.list.SelectedItem().(moduleItem); ok {
				if m.selected[item.module.Name] {
					delete(m.selected, item.module.Name)
				} else {
					m.selected[item.module.Name] = true
					m.autoSelectDependencies(item.module)
				}
				m.updateListItem(item.module.Name)
			}
			return m, nil

		case key.Matches(msg, m.delegateKeys.edit):
			if currentModule := m.getCurrentModule(); currentModule != nil {
				return m.editModuleForm(*currentModule)
			} else {
				m.error = fmt.Errorf("no module selected")
			}
			return m, nil

		case key.Matches(msg, m.delegateKeys.add):
			m.error = nil
			return m.addDotfileForm()

		case key.Matches(msg, m.delegateKeys.importFile):
			m.error = nil
			return m.importDotfileForm()

		case key.Matches(msg, m.delegateKeys.export):
			m.exportMode = !m.exportMode
			return m, nil

		case key.Matches(msg, m.keys.create):
			m.error = nil
			return m.createModuleForm()

		case key.Matches(msg, m.keys.install):
			if len(m.selected) > 0 {
				m.shouldInstall = true
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case msg.String() == "ctrl+c" || msg.String() == "q":
			m.quitting = true
			return m, tea.Quit

		case msg.String() == "esc":
			if m.form != nil {
				m.form = nil
				m.mode = ""
				m.error = nil
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) updateListItem(moduleName string) {
	items := m.list.Items()
	for i, item := range items {
		if moduleItem, ok := item.(moduleItem); ok {
			if moduleItem.module.Name == moduleName {
				moduleItem.selected = m.selected[moduleName]
				items[i] = moduleItem
			}
		}
	}
	m.list.SetItems(items)
}

func (m Model) createModuleForm() (tea.Model, tea.Cmd) {
	// Form data variables - these will be directly bound to huh
	var (
		moduleName     string
		description    string
		dependencies   []string
		commonPackages string
		commonCommands string
	)

	// Initialize specific packages and commands
	specificPackages := []SpecificPackageForm{{}, {}}
	specificCommands := []SpecificCommandForm{{}, {}}

	// Build list of available modules for dependencies
	var moduleOptions []huh.Option[string]
	for _, item := range m.list.Items() {
		if moduleItem, ok := item.(moduleItem); ok {
			moduleOptions = append(moduleOptions, huh.NewOption(moduleItem.module.Name, moduleItem.module.Name))
		}
	}

	var groups []*huh.Group

	// Basic info group
	basicGroup := huh.NewGroup(
		huh.NewInput().
			Title("Module Name").
			Description("Enter a unique identifier for your module (e.g., nvim, shell, git)").
			Value(&moduleName).
			Validate(func(str string) error {
				trimmed := strings.TrimSpace(str)
				if trimmed == "" {
					return nil // Allow empty during typing
				}
				// Check for conflicts with existing modules
				for _, item := range m.list.Items() {
					if moduleItem, ok := item.(moduleItem); ok {
						if moduleItem.module.Name == trimmed {
							return fmt.Errorf("module '%s' already exists", trimmed)
						}
					}
				}
				return nil
			}),

		huh.NewInput().
			Title("Description").
			Description("Brief description of what this module configures (optional)").
			Value(&description),
	)

	// Dependencies group
	var dependencyGroup *huh.Group
	if len(moduleOptions) > 0 {
		dependencyGroup = huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Dependencies").
				Description("Select modules that must be installed before this one").
				Options(moduleOptions...).
				Value(&dependencies),
		)
	} else {
		dependencyGroup = huh.NewGroup(
			huh.NewNote().
				Title("Dependencies").
				Description("No existing modules available. Create other modules first to set up dependencies."),
		)
	}

	// Package group
	packageGroup := huh.NewGroup(
		huh.NewInput().
			Title("Common Packages").
			Description("Packages with same name on both macOS and Ubuntu (e.g., git, curl, vim, tmux)").
			Value(&commonPackages),
	)

	// Commands group
	commandGroup := huh.NewGroup(
		huh.NewInput().
			Title("Common Commands").
			Description("Commands that work on both macOS and Ubuntu (optional)").
			Value(&commonCommands),
	)

	// Specific packages group
	specificPkgGroup := huh.NewGroup(
		huh.NewInput().
			Title("Specific Package 1 - Name").
			Description("Package with different name on macOS vs Ubuntu").
			Value(&specificPackages[0].Name),

		huh.NewSelect[string]().
			Title("Specific Package 1 - Manager").
			Description("Package manager for this package").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificPackages[0].Manager),

		huh.NewInput().
			Title("Specific Package 2 - Name").
			Description("Another package with different name on macOS vs Ubuntu").
			Value(&specificPackages[1].Name),

		huh.NewSelect[string]().
			Title("Specific Package 2 - Manager").
			Description("Package manager for this package").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificPackages[1].Manager),
	)

	// Specific commands group
	specificCmdGroup := huh.NewGroup(
		huh.NewInput().
			Title("Specific Command 1 - Command").
			Description("Command only for macOS or Ubuntu").
			Value(&specificCommands[0].Command),

		huh.NewSelect[string]().
			Title("Specific Command 1 - OS").
			Description("Operating system for this command").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificCommands[0].OS),

		huh.NewInput().
			Title("Specific Command 2 - Command").
			Description("Another command only for macOS or Ubuntu").
			Value(&specificCommands[1].Command),

		huh.NewSelect[string]().
			Title("Specific Command 2 - OS").
			Description("Operating system for this command").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificCommands[1].OS),
	)

	groups = append(groups, basicGroup, dependencyGroup, packageGroup, commandGroup, specificPkgGroup, specificCmdGroup)

	form := huh.NewForm(groups...).WithTheme(huh.ThemeCharm())

	// Store the form and local variables in the model context
	m.form = form
	m.mode = "create"

	// Store form variables in a temporary struct for later use
	m.createFormData = &CreateFormData{
		ModuleName:       &moduleName,
		Description:      &description,
		Dependencies:     &dependencies,
		CommonPackages:   &commonPackages,
		CommonCommands:   &commonCommands,
		SpecificPackages: specificPackages,
		SpecificCommands: specificCommands,
	}

	return m, m.form.Init()
}

func (m Model) editModuleForm(module models.ModuleConfig) (tea.Model, tea.Cmd) {
	// Form data variables initialized with current values
	var (
		description    = module.Description
		dependencies   = module.Dependencies
		commonPackages = strings.Join(module.Packages.Common, ", ")
		commonCommands = ""
	)

	// Build common commands string
	var commonCmds []string
	for _, cmd := range module.Commands {
		if cmd.OS == "" {
			commonCmds = append(commonCmds, cmd.Command)
		}
	}
	commonCommands = strings.Join(commonCmds, "; ")

	// Build list of available modules for dependencies (excluding current module)
	var moduleOptions []huh.Option[string]
	for _, item := range m.list.Items() {
		if moduleItem, ok := item.(moduleItem); ok {
			if moduleItem.module.Name != module.Name {
				moduleOptions = append(moduleOptions, huh.NewOption(moduleItem.module.Name, moduleItem.module.Name))
			}
		}
	}

	var groups []*huh.Group

	// Basic info group
	basicGroup := huh.NewGroup(
		huh.NewNote().
			Title("Editing Module: "+module.Name).
			Description("Modify the configuration for this module"),

		huh.NewInput().
			Title("Description").
			Description("Brief description of what this module configures").
			Value(&description),
	)

	// Dependencies group
	var dependencyGroup *huh.Group
	if len(moduleOptions) > 0 {
		dependencyGroup = huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Dependencies").
				Description("Select modules that must be installed before this one").
				Options(moduleOptions...).
				Value(&dependencies),
		)
	} else {
		dependencyGroup = huh.NewGroup(
			huh.NewNote().
				Title("Dependencies").
				Description("No other modules available as dependencies"),
		)
	}

	// Package group
	packageGroup := huh.NewGroup(
		huh.NewInput().
			Title("Common Packages").
			Description("Packages with same name on both macOS and Ubuntu").
			Value(&commonPackages),
	)

	// Commands group
	commandGroup := huh.NewGroup(
		huh.NewInput().
			Title("Common Commands").
			Description("Installation commands that work on all systems (optional)").
			Value(&commonCommands),
	)

	// Initialize form data with existing specific packages and commands
	specificPackages := []SpecificPackageForm{{}, {}}
	specificCommands := []SpecificCommandForm{{}, {}}

	// Load existing specific packages
	for i, pkg := range module.Packages.Specific {
		if i < 2 {
			specificPackages[i] = SpecificPackageForm{
				Name:    pkg.Name,
				Manager: pkg.Manager,
			}
		}
	}

	// Load existing specific commands
	specificCmdIndex := 0
	for _, cmd := range module.Commands {
		if cmd.OS != "" && specificCmdIndex < 2 {
			specificCommands[specificCmdIndex] = SpecificCommandForm{
				Command: cmd.Command,
				OS:      cmd.OS,
			}
			specificCmdIndex++
		}
	}

	// Specific packages group
	specificPkgGroup := huh.NewGroup(
		huh.NewInput().
			Title("Specific Package 1 - Name").
			Description("Package with different name on macOS vs Ubuntu").
			Value(&specificPackages[0].Name),

		huh.NewSelect[string]().
			Title("Specific Package 1 - Manager").
			Description("Package manager for this package").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificPackages[0].Manager),

		huh.NewInput().
			Title("Specific Package 2 - Name").
			Description("Another package with different name on macOS vs Ubuntu").
			Value(&specificPackages[1].Name),

		huh.NewSelect[string]().
			Title("Specific Package 2 - Manager").
			Description("Package manager for this package").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificPackages[1].Manager),
	)

	// Specific commands group
	specificCmdGroup := huh.NewGroup(
		huh.NewInput().
			Title("Specific Command 1 - Command").
			Description("Command only for macOS or Ubuntu").
			Value(&specificCommands[0].Command),

		huh.NewSelect[string]().
			Title("Specific Command 1 - OS").
			Description("Operating system for this command").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificCommands[0].OS),

		huh.NewInput().
			Title("Specific Command 2 - Command").
			Description("Another command only for macOS or Ubuntu").
			Value(&specificCommands[1].Command),

		huh.NewSelect[string]().
			Title("Specific Command 2 - OS").
			Description("Operating system for this command").
			Options(
				huh.NewOption("None", ""),
				huh.NewOption("Homebrew (macOS)", "brew"),
				huh.NewOption("APT (Debian/Ubuntu)", "apt"),
			).
			Value(&specificCommands[1].OS),
	)

	groups = append(groups, basicGroup, dependencyGroup, packageGroup, commandGroup, specificPkgGroup, specificCmdGroup)

	form := huh.NewForm(groups...).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "edit"

	// Store form variables for later use
	m.editFormData = &EditFormData{
		ModuleName:       module.Name,
		Description:      &description,
		Dependencies:     &dependencies,
		CommonPackages:   &commonPackages,
		CommonCommands:   &commonCommands,
		SpecificPackages: specificPackages,
		SpecificCommands: specificCommands,
	}

	return m, m.form.Init()
}

func (m Model) addDotfileForm() (tea.Model, tea.Cmd) {
	if len(m.allModules) == 0 {
		m.error = fmt.Errorf("no modules available. Create a module first")
		return m, nil
	}

	// Use current selected module directly
	currentModule := m.getCurrentModule()
	if currentModule == nil {
		m.error = fmt.Errorf("no module selected")
		return m, nil
	}

	// Form data variables
	var (
		source      string
		destination string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Adding dotfile to: "+currentModule.Name).
				Description("Configure the dotfile mapping for this module"),

			huh.NewInput().
				Title("Source Path").
				Description("Path within the module (e.g., dotfiles/.bashrc)").
				Value(&source),

			huh.NewInput().
				Title("Destination Path").
				Description("Target path in home directory (e.g., .bashrc)").
				Value(&destination),
		),
	).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "add"

	// Store form variables for later use
	moduleChoice := currentModule.Name
	m.addFormData = &AddFormData{
		ModuleChoice: &moduleChoice,
		Source:       &source,
		Destination:  &destination,
	}

	return m, m.form.Init()
}

func (m Model) importDotfileForm() (tea.Model, tea.Cmd) {
	if len(m.allModules) == 0 {
		m.error = fmt.Errorf("no modules available. Create a module first")
		return m, nil
	}

	// Use current selected module directly
	currentModule := m.getCurrentModule()
	if currentModule == nil {
		m.error = fmt.Errorf("no module selected")
		return m, nil
	}

	// Form data variables
	var (
		sourcePath      string
		destinationPath string
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Importing dotfile to: "+currentModule.Name).
				Description("Import an existing file/directory into this module"),

			huh.NewInput().
				Title("Source Path").
				Description("Path to existing file/directory (e.g., ~/.bashrc, ~/.config/nvim)").
				Value(&sourcePath),

			huh.NewInput().
				Title("Destination Path").
				Description("Path within module dotfiles directory (e.g., .bashrc, nvim/)").
				Value(&destinationPath),
		),
	).WithTheme(huh.ThemeCharm())

	m.form = form
	m.mode = "import"

	// Store form variables for later use
	moduleChoice := currentModule.Name
	m.importFormData = &ImportFormData{
		ModuleChoice:    &moduleChoice,
		SourcePath:      &sourcePath,
		DestinationPath: &destinationPath,
	}

	return m, m.form.Init()
}

func (m Model) handleFormCompletion() (tea.Model, tea.Cmd) {
	switch m.mode {
	case "create":
		if m.createFormData != nil {
			return m.handleCreateModuleCompletion()
		}
	case "edit":
		if m.editFormData != nil {
			return m.handleEditModuleCompletion()
		}
	case "add":
		if m.addFormData != nil {
			return m.handleAddDotfileCompletion()
		}
	case "import":
		if m.importFormData != nil {
			return m.handleImportDotfileCompletion()
		}
	}
	return m, nil
}

func (m Model) handleCreateModuleCompletion() (tea.Model, tea.Cmd) {
	if m.createFormData == nil {
		m.error = fmt.Errorf("form data not available")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	data := m.createFormData
	moduleName := *data.ModuleName
	description := *data.Description
	dependencies := *data.Dependencies
	commonPackages := *data.CommonPackages
	commonCommands := *data.CommonCommands

	// Validate required fields
	if strings.TrimSpace(moduleName) == "" {
		m.error = fmt.Errorf("module name is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	// Parse common packages
	var commonPkgs []string
	if strings.TrimSpace(commonPackages) != "" {
		for _, pkg := range strings.Split(commonPackages, ",") {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" {
				commonPkgs = append(commonPkgs, pkg)
			}
		}
	}

	// Build package manager config
	packages := models.PackageManager{
		Common: commonPkgs,
	}

	// Add specific packages
	for _, pkg := range data.SpecificPackages {
		if strings.TrimSpace(pkg.Name) != "" && strings.TrimSpace(pkg.Manager) != "" && pkg.Manager != "" {
			packages.Specific = append(packages.Specific, models.SpecificPackage{
				Name:    strings.TrimSpace(pkg.Name),
				Manager: strings.TrimSpace(pkg.Manager),
			})
		}
	}

	// Parse commands
	var commands []models.InstallCommand
	if strings.TrimSpace(commonCommands) != "" {
		for _, cmd := range strings.Split(commonCommands, ";") {
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				commands = append(commands, models.InstallCommand{
					Command: cmd,
					OS:      "", // Empty means common
				})
			}
		}
	}

	// Add specific commands
	for _, cmd := range data.SpecificCommands {
		if strings.TrimSpace(cmd.Command) != "" && strings.TrimSpace(cmd.OS) != "" && cmd.OS != "" {
			commands = append(commands, models.InstallCommand{
				Command: strings.TrimSpace(cmd.Command),
				OS:      strings.TrimSpace(cmd.OS),
			})
		}
	}

	// Create the module
	if err := m.manager.CreateModuleWithConfig(moduleName, description, dependencies, packages, commands); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) handleEditModuleCompletion() (tea.Model, tea.Cmd) {
	if m.editFormData == nil {
		m.error = fmt.Errorf("form data not available")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	data := m.editFormData
	moduleName := data.ModuleName
	description := *data.Description
	dependencies := *data.Dependencies
	commonPackages := *data.CommonPackages
	commonCommands := *data.CommonCommands

	// Parse common packages
	var commonPkgs []string
	if strings.TrimSpace(commonPackages) != "" {
		for _, pkg := range strings.Split(commonPackages, ",") {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" {
				commonPkgs = append(commonPkgs, pkg)
			}
		}
	}

	packages := models.PackageManager{
		Common: commonPkgs,
	}

	// Add specific packages from edit form
	for _, pkg := range data.SpecificPackages {
		if strings.TrimSpace(pkg.Name) != "" && strings.TrimSpace(pkg.Manager) != "" && pkg.Manager != "" {
			packages.Specific = append(packages.Specific, models.SpecificPackage{
				Name:    strings.TrimSpace(pkg.Name),
				Manager: strings.TrimSpace(pkg.Manager),
			})
		}
	}

	// Parse commands
	var commands []models.InstallCommand
	if strings.TrimSpace(commonCommands) != "" {
		for _, cmd := range strings.Split(commonCommands, ";") {
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				commands = append(commands, models.InstallCommand{
					Command: cmd,
					OS:      "", // Empty means common
				})
			}
		}
	}

	// Add specific commands
	for _, cmd := range data.SpecificCommands {
		if strings.TrimSpace(cmd.Command) != "" && strings.TrimSpace(cmd.OS) != "" {
			commands = append(commands, models.InstallCommand{
				Command: strings.TrimSpace(cmd.Command),
				OS:      strings.TrimSpace(cmd.OS),
			})
		}
	}

	// Update the module
	if err := m.manager.UpdateModule(moduleName, description, dependencies, packages, commands); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) handleAddDotfileCompletion() (tea.Model, tea.Cmd) {
	if m.addFormData == nil {
		m.error = fmt.Errorf("form data not available")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	data := m.addFormData
	moduleChoice := *data.ModuleChoice
	source := *data.Source
	destination := *data.Destination

	// Validate required fields
	if strings.TrimSpace(moduleChoice) == "" {
		m.error = fmt.Errorf("module selection is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}
	if strings.TrimSpace(source) == "" {
		m.error = fmt.Errorf("source path is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}
	if strings.TrimSpace(destination) == "" {
		m.error = fmt.Errorf("destination path is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	if err := m.manager.AddDotfile(moduleChoice, source, destination); err != nil {
		m.error = err
		m.form = nil
		m.mode = ""
		return m, nil
	}

	return m.reloadModules()
}

func (m Model) handleImportDotfileCompletion() (tea.Model, tea.Cmd) {
	if m.importFormData == nil {
		m.error = fmt.Errorf("form data not available")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	data := m.importFormData
	moduleChoice := *data.ModuleChoice
	sourcePath := *data.SourcePath
	destinationPath := *data.DestinationPath

	// Validate required fields
	if strings.TrimSpace(moduleChoice) == "" {
		m.error = fmt.Errorf("module selection is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}
	if strings.TrimSpace(sourcePath) == "" {
		m.error = fmt.Errorf("source path is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}
	if strings.TrimSpace(destinationPath) == "" {
		m.error = fmt.Errorf("destination path is required")
		m.form = nil
		m.mode = ""
		return m, nil
	}

	if err := m.manager.ImportDotfileWithDestination(moduleChoice, sourcePath, destinationPath); err != nil {
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
		items := make([]list.Item, len(modules))
		for i, module := range modules {
			items[i] = moduleItem{
				module:   module,
				selected: false,
			}
		}
		m.list.SetItems(items)
		m.allModules = modules
		m.selected = make(map[string]bool)
	}

	m.form = nil
	m.mode = ""
	return m, nil
}

func (m *Model) autoSelectDependencies(module models.ModuleConfig) {
	for _, dep := range module.Dependencies {
		if !m.selected[dep] {
			m.selected[dep] = true
			for _, mod := range m.allModules {
				if mod.Name == dep {
					m.updateListItem(dep)
					m.autoSelectDependencies(mod)
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
		formView := m.form.View()
		if m.error != nil {
			formView += "\n\n" + errorStyle.Render("Error: "+m.error.Error())
		}
		return formView
	}

	if len(m.allModules) == 0 {
		var s strings.Builder
		s.WriteString(titleStyle.Render("DotCLI - Dotfiles Manager"))
		s.WriteString("\n\n")
		if m.error != nil {
			s.WriteString(errorStyle.Render("Error: " + m.error.Error()))
			s.WriteString("\n\n")
		}
		s.WriteString("No modules found. Press 'c' to create your first module.\n\n")
		s.WriteString(helpStyle.Render("c: create module • q: quit"))
		return s.String()
	}

	// Update list items to show selection status
	items := m.list.Items()
	for i, item := range items {
		if moduleItem, ok := item.(moduleItem); ok {
			moduleItem.selected = m.selected[moduleItem.module.Name]
			items[i] = moduleItem
		}
	}
	m.list.SetItems(items)

	var s strings.Builder

	if m.error != nil {
		s.WriteString(errorStyle.Render("Error: " + m.error.Error()))
		s.WriteString("\n")
	}

	s.WriteString(m.list.View())

	if len(m.selected) > 0 {
		s.WriteString("\n")
		statusText := fmt.Sprintf("Selected %d modules", len(m.selected))
		if m.exportMode {
			statusText += " (dotfiles only)"
		}
		s.WriteString(statusStyle.Render(statusText))
	}

	return appStyle.Render(s.String())
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

func (m Model) GetExportMode() bool {
	return m.exportMode
}
