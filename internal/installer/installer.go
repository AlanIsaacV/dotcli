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
		if err := i.installPackagesWithOptions(module, statusCh, options); err != nil {
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
	return i.installPackagesWithOptions(module, statusCh, models.InstallOptions{})
}

func (i *Installer) installPackagesWithOptions(module models.ModuleConfig, statusCh chan<- models.InstallationStatus, options models.InstallOptions) error {
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

	// Check which packages are already installed
	packageStatus := i.checkPackagesStatus(pm, packages)
	var toInstall []string
	var alreadyInstalled []string

	for _, status := range packageStatus {
		if status.Installed && !options.ForceReinstall {
			alreadyInstalled = append(alreadyInstalled, fmt.Sprintf("%s (%s)", status.Name, status.Version))
		} else {
			toInstall = append(toInstall, status.Name)
		}
	}

	if len(alreadyInstalled) > 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: fmt.Sprintf("Already installed: %s", strings.Join(alreadyInstalled, ", ")),
		}
	}

	if len(toInstall) == 0 {
		statusCh <- models.InstallationStatus{
			Module: module.Name,
			Status: "All packages already installed",
		}
		return nil
	}

	statusCh <- models.InstallationStatus{
		Module: module.Name,
		Status: fmt.Sprintf("Installing packages via %s: %s", pm, strings.Join(toInstall, ", ")),
	}

	return i.installWithPackageManager(pm, toInstall, statusCh, module.Name)
}

func (i *Installer) hasNoPackages(pm models.PackageManager) bool {
	return len(pm.Common) == 0 && len(pm.Specific) == 0
}

func (i *Installer) detectPackageManager() string {
	managers := []string{"brew", "apt", "pacman", "yum", "snap"}

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

func (i *Installer) isPackageInstalled(pm string, pkg string) (bool, string) {
	var cmd *exec.Cmd

	switch pm {
	case "brew":
		cmd = exec.Command("brew", "list", "--versions", pkg)
	case "apt":
		cmd = exec.Command("dpkg-query", "-W", "-f=${Status} ${Version}", pkg)
	case "pacman":
		cmd = exec.Command("pacman", "-Q", pkg)
	case "yum":
		cmd = exec.Command("rpm", "-q", "--queryformat", "%{VERSION}", pkg)
	case "snap":
		cmd = exec.Command("snap", "list", pkg)
	default:
		return false, ""
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, ""
	}

	outputStr := strings.TrimSpace(string(output))

	switch pm {
	case "brew":
		// brew list --versions returns: "package 1.2.3"
		if outputStr != "" && strings.Contains(outputStr, pkg) {
			parts := strings.Fields(outputStr)
			if len(parts) >= 2 {
				return true, parts[1]
			}
			return true, "installed"
		}
		return false, ""
	case "apt":
		// dpkg-query returns: "install ok installed 1.2.3"
		if strings.Contains(outputStr, "install ok installed") {
			parts := strings.Fields(outputStr)
			if len(parts) >= 4 {
				return true, parts[3]
			}
			return true, "installed"
		}
		return false, ""
	case "pacman":
		// pacman -Q returns: "package 1.2.3-1"
		if outputStr != "" {
			parts := strings.Fields(outputStr)
			if len(parts) >= 2 {
				return true, parts[1]
			}
			return true, "installed"
		}
		return false, ""
	case "yum":
		// rpm -q returns version directly
		if outputStr != "" && !strings.Contains(outputStr, "not installed") {
			return true, outputStr
		}
		return false, ""
	case "snap":
		// snap list returns table format, check if package exists
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, pkg+" ") || strings.HasPrefix(line, pkg+"\t") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return true, parts[1]
				}
				return true, "installed"
			}
		}
		return false, ""
	}

	return false, ""
}

func (i *Installer) checkPackagesStatus(pm string, packages []string) []models.PackageStatus {
	var status []models.PackageStatus

	for _, pkg := range packages {
		installed, version := i.isPackageInstalled(pm, pkg)
		status = append(status, models.PackageStatus{
			Name:      pkg,
			Installed: installed,
			Version:   version,
		})
	}

	return status
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
		updateCmd := exec.Command("sudo", "apt", "update")
		if err := updateCmd.Run(); err != nil {
			statusCh <- models.InstallationStatus{
				Module: moduleName,
				Status: "Warning: apt update failed, continuing...",
			}
		}
		cmd = exec.Command("sudo", append([]string{"apt", "install", "-y"}, packages...)...)
	case "pacman":
		cmd = exec.Command("sudo", append([]string{"pacman", "-S", "--noconfirm"}, packages...)...)
	case "yum":
		cmd = exec.Command("sudo", append([]string{"yum", "install", "-y"}, packages...)...)
	case "snap":
		for _, pkg := range packages {
			statusCh <- models.InstallationStatus{
				Module: moduleName,
				Status: fmt.Sprintf("Installing %s via snap", pkg),
			}
			snapCmd := exec.Command("sudo", "snap", "install", pkg)
			output, err := snapCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to install %s via snap: %w\nOutput: %s", pkg, err, string(output))
			}
		}
		return nil
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
		destPath := filepath.Join(i.homeDir, dotfile.Destination)

		if err := i.createSymlink(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", dotfile.Source, err)
		}
	}

	return nil
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
