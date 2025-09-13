package models

type DotfileMapping struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

type PackageManager struct {
	Brew   []string `yaml:"brew,omitempty"`
	Apt    []string `yaml:"apt,omitempty"`
	Pacman []string `yaml:"pacman,omitempty"`
	Yum    []string `yaml:"yum,omitempty"`
	Snap   []string `yaml:"snap,omitempty"`
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
	SkipInstalled  bool
}

type ModuleState struct {
	Config    ModuleConfig
	Selected  bool
	Installed bool
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
