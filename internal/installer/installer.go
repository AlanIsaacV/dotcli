package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AlanIsaacV/dotcli/internal/models"
)

type Installer struct {
	homeDir string
}

func New() (*Installer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Installer{
		homeDir: homeDir,
	}, nil
}

func (i *Installer) ResolveDependencies(modules []models.ModuleConfig, selected []string) ([]string, error) {
	moduleMap := make(map[string]models.ModuleConfig)
	for _, module := range modules {
		moduleMap[module.Name] = module
	}

	var resolved []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var resolve func(string) error
	resolve = func(moduleName string) error {
		if visited[moduleName] {
			return nil
		}

		if visiting[moduleName] {
			return fmt.Errorf("circular dependency detected: %s", moduleName)
		}

		module, exists := moduleMap[moduleName]
		if !exists {
			return fmt.Errorf("module not found: %s", moduleName)
		}

		visiting[moduleName] = true

		for _, dep := range module.Dependencies {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		visiting[moduleName] = false
		visited[moduleName] = true
		resolved = append(resolved, moduleName)

		return nil
	}

	for _, moduleName := range selected {
		if err := resolve(moduleName); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}

func (i *Installer) InstallModule(module models.ModuleConfig, statusCh chan<- models.InstallationStatus) error {
	return i.InstallModuleWithOptions(module, statusCh, models.InstallOptions{})
}

func (i *Installer) InstallModuleWithOptions(module models.ModuleConfig, statusCh chan<- models.InstallationStatus, options models.InstallOptions) error {
	statusCh <- models.InstallationStatus{
		Module:   module.Name,
		Status:   "Starting installation",
		Progress: 0.0,
	}

	// Install packages first
	packagesInstalled := false
	if !i.hasNoPackages(module.Packages) {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "Installing packages",
		}
		if err := i.installPackages(module, statusCh); err != nil {
			statusCh <- models.InstallationStatus{
				Module: module.Name,
				Status: "Failed to install packages",
				Error:  err,
			}
			return err
		}
		statusCh <- models.InstallationStatus{
			Module:   module.Name,
			Status:   "Packages installed",
			Progress: 0.3,
		}
		packagesInstalled = true
	}

	// Run custom install script if it exists
	scriptPath := filepath.Join(module.Path, "install.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "Running install script",
		}
		if err := i.runInstallScript(module); err != nil {
			statusCh <- models.InstallationStatus{
				Module: module.Name,
				Status: "Failed to run install script",
				Error:  err,
			}
			return err
		}
		progress := 0.5
		if packagesInstalled {
			progress = 0.7
		}
		statusCh <- models.InstallationStatus{
			Module:   module.Name,
			Status:   "Install script completed",
			Progress: progress,
		}
	}

	// Run custom commands
	if len(module.Commands) > 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "Running custom commands",
		}
		if err := i.runCustomCommands(module, statusCh); err != nil {
			statusCh <- models.InstallationStatus{
				Module: module.Name,
				Status: "Failed to run custom commands",
				Error:  err,
			}
			return err
		}
		progress := 0.8
		if packagesInstalled {
			progress = 0.9
		}
		statusCh <- models.InstallationStatus{
			Module:   module.Name,
			Status:   "Custom commands completed",
			Progress: progress,
		}
	}

	// Create symlinks
	if len(module.Dotfiles) > 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "Creating symlinks",
		}
		if err := i.createSymlinks(module); err != nil {
			statusCh <- models.InstallationStatus{
				Module: module.Name,
				Status: "Failed to create symlinks",
				Error:  err,
			}
			return err
		}
	}

	statusCh <- models.InstallationStatus{
		Module:   module.Name,
		Status:   "Installation completed",
		Progress: 1.0,
	}

	return nil
}

func (i *Installer) installPackages(module models.ModuleConfig, statusCh chan<- models.InstallationStatus) error {
	if i.hasNoPackages(module.Packages) {
		return nil
	}

	pm := i.detectPackageManager()
	if pm == "" {
		return fmt.Errorf("no supported package manager found")
	}

	var packages []string

	// Add common packages (work with all package managers)
	packages = append(packages, module.Packages.Common...)

	// Add specific packages for this package manager
	for _, pkg := range module.Packages.Specific {
		if pkg.Manager == pm {
			packages = append(packages, pkg.Name)
		}
	}

	if len(packages) == 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: fmt.Sprintf("No packages defined for %s", pm),
		}
		return nil
	}

	if len(packages) == 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: fmt.Sprintf("No packages defined for %s", pm),
		}
		return nil
	}

	statusCh <- models.InstallationStatus{
		Module: module.Name,
		Status: fmt.Sprintf("Installing packages via %s: %s", pm, strings.Join(packages, ", ")),
	}

	return i.installWithPackageManager(pm, packages, statusCh, module.Name)
}

func (i *Installer) hasNoPackages(pm models.PackageManager) bool {
	return len(pm.Common) == 0 && len(pm.Specific) == 0
}

func (i *Installer) detectPackageManager() string {
	managers := []string{"brew", "apt"}

	for _, manager := range managers {
		if i.commandExists(manager) {
			return manager
		}
	}

	return ""
}

func (i *Installer) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (i *Installer) installWithPackageManager(pm string, packages []string, statusCh chan<- models.InstallationStatus, moduleName string) error {
	var cmd *exec.Cmd

	switch pm {
	case "brew":
		args := append([]string{"install"}, packages...)
		cmd = exec.Command("brew", args...)
	case "apt":
		// Update package list first for apt
		statusCh <- models.InstallationStatus{
			Module: moduleName,
			Status: "Updating package list (apt update)",
		}
		updateCmd := exec.Command("sudo", "apt-get", "update")
		if err := updateCmd.Run(); err != nil {
			statusCh <- models.InstallationStatus{
				Module: moduleName,
				Status: "Warning: apt update failed, continuing...",
			}
		}
		cmd = exec.Command("sudo", append([]string{"apt-get", "install", "-y"}, packages...)...)
	default:
		return fmt.Errorf("unsupported package manager: %s", pm)
	}

	// Capture output to avoid mixing with our status messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("package installation failed: %w\nCommand: %s\nOutput: %s", err, cmd.String(), string(output))
	}

	return nil
}

func (i *Installer) runInstallScript(module models.ModuleConfig) error {
	scriptPath := filepath.Join(module.Path, "install.sh")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", scriptPath)
	} else {
		cmd = exec.Command("bash", scriptPath)
	}

	cmd.Dir = module.Path
	return cmd.Run()
}

func (i *Installer) createSymlinks(module models.ModuleConfig) error {
	for _, dotfile := range module.Dotfiles {
		sourcePath := filepath.Join(module.Path, dotfile.Source)
		destPath := i.expandPath(dotfile.Destination)

		if err := i.createSymlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", dotfile.Source, err)
		}
	}

	return nil
}

func (i *Installer) expandPath(path string) string {
	// Handle ~ expansion
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(i.homeDir, path[2:])
	}

	// Handle exact ~ (home directory)
	if path == "~" {
		return i.homeDir
	}

	// Handle $HOME expansion
	if strings.HasPrefix(path, "$HOME/") {
		return filepath.Join(i.homeDir, path[6:])
	}

	// Handle exact $HOME
	if path == "$HOME" {
		return i.homeDir
	}

	// Expand environment variables
	expanded := os.ExpandEnv(path)

	// If path is relative and doesn't start with . or /, assume it's relative to home
	if !filepath.IsAbs(expanded) && !strings.HasPrefix(expanded, ".") {
		return filepath.Join(i.homeDir, expanded)
	}

	return expanded
}

func (i *Installer) createSymlink(source, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if _, err := os.Lstat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return os.Symlink(absSource, dest)
}

func (i *Installer) runCustomCommands(module models.ModuleConfig, statusCh chan<- models.InstallationStatus) error {
	pm := i.detectPackageManager()
	if pm == "" {
		return fmt.Errorf("no supported package manager found")
	}

	for _, command := range module.Commands {
		// Skip commands that don't match current package manager (OS)
		if command.OS != "" && command.OS != pm {
			continue
		}

		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: fmt.Sprintf("Running command: %s", command.Command),
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", command.Command)
		} else {
			cmd = exec.Command("bash", "-c", command.Command)
		}

		cmd.Dir = module.Path
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %w\nCommand: %s\nOutput: %s", err, command.Command, string(output))
		}
	}

	return nil
}

func (i *Installer) GetInstallationOrder(modules []models.ModuleConfig, selected []string) ([]string, error) {
	moduleMap := make(map[string]models.ModuleConfig)
	for _, module := range modules {
		moduleMap[module.Name] = module
	}

	selectedMap := make(map[string]bool)
	for _, name := range selected {
		selectedMap[name] = true
	}

	// Add missing dependencies
	var allSelected []string
	for _, name := range selected {
		if module, exists := moduleMap[name]; exists {
			for _, dep := range module.Dependencies {
				if !selectedMap[dep] {
					allSelected = append(allSelected, dep)
					selectedMap[dep] = true
				}
			}
		}
	}
	allSelected = append(allSelected, selected...)

	return i.ResolveDependencies(modules, allSelected)
}

func (i *Installer) InstallDotfilesOnly(module models.ModuleConfig, statusCh chan<- models.InstallationStatus) error {
	statusCh <- models.InstallationStatus{
		Module:   module.Name,
		Status:   "Installing dotfiles only",
		Progress: 0.0,
	}

	// Only create symlinks
	if len(module.Dotfiles) > 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "Creating symlinks",
		}
		if err := i.createSymlinks(module); err != nil {
			statusCh <- models.InstallationStatus{
				Module: module.Name,
				Status: "Failed to create symlinks",
				Error:  err,
			}
			return err
		}
	}

	statusCh <- models.InstallationStatus{
		Module:   module.Name,
		Status:   "Dotfiles installation completed",
		Progress: 1.0,
	}

	return nil
}
