package models

type DotfileMapping struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

type SpecificPackage struct {
	Name    string `yaml:"name"`
	Manager string `yaml:"manager"` // brew, apt, pacman, yum, snap
}

type PackageManager struct {
	Common   []string          `yaml:"common,omitempty"`   // Packages with same name across managers
	Specific []SpecificPackage `yaml:"specific,omitempty"` // Packages with different names or specific managers
}

type ModuleConfig struct {
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Dependencies []string         `yaml:"dependencies"`
	Packages     PackageManager   `yaml:"packages,omitempty"`
	Dotfiles     []DotfileMapping `yaml:"dotfiles"`
	Path         string           `yaml:"-"`
}

type InstallOptions struct {
	ForceReinstall bool
}

type ModuleState struct {
	Config   ModuleConfig
	Selected bool
}

type InstallationStatus struct {
	Module   string
	Status   string
	Error    error
	Progress float64
}

type PackageStatus struct {
	Name      string
	Installed bool
	Version   string
}
