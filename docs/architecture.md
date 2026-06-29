# Architecture

## Purpose

DotCLI is a terminal UI (TUI) dotfiles manager. It scans a `~/dotfiles/modules/`
directory for self-contained "modules" (each describing packages, custom
commands, and dotfile→home symlinks via a `config.yaml`), lets the user browse
and select them in an interactive list, resolves inter-module dependencies, and
then installs the selected modules in topological order: installing packages
through the host package manager (brew/apt), running optional setup scripts and
commands, and symlinking the dotfiles into place. It is a single-binary,
single-user developer tool — not a service. [INFERRED]

## Code layout

```
.
├── main.go                     # Entrypoint: bootstrap + post-TUI install loop
├── internal/
│   ├── models/module.go        # Shared data types (ModuleConfig, PackageManager, ...)
│   ├── scanner/scanner.go      # Discover & parse modules from disk (config.yaml)
│   ├── manager/manager.go      # Create/edit modules, add/import dotfiles, templates
│   ├── installer/installer.go  # Dependency resolution + install pipeline + symlinks
│   └── ui/ui.go                # Bubble Tea model: list browser + huh forms
├── go.mod / go.sum
└── README.md
```

Module descriptions:

- **`main.go`** — Resolves the dotfiles root (`DOTFILES_PATH` env override, else
  `~/dotfiles`), scans modules, runs the Bubble Tea program, and — if the user
  chose to install — drives the installation goroutine, streaming status updates
  to stdout. This is the only place the install pipeline is invoked.
- **`internal/models`** — Pure data types shared across packages. No behavior.
  `ModuleConfig` mirrors the on-disk `config.yaml` (yaml tags); `InstallationStatus`
  is the message streamed over the status channel during installs.
- **`internal/scanner`** — Read-only discovery: lists `modules/*/`, unmarshals each
  `config.yaml` into a `ModuleConfig`, and stamps `Path`. Bad modules are warned
  about and skipped, not fatal.
- **`internal/manager`** — All on-disk mutations to modules: create (from template
  or from a config), update (preserving existing dotfiles), add a dotfile mapping,
  and import an existing file/dir (copy into module → delete original → symlink back
  → register in config). Also owns the built-in templates.
- **`internal/installer`** — Stateless install engine: topological dependency
  resolution with cycle detection (`ResolveDependencies` / `GetInstallationOrder`),
  package-manager detection (brew/apt), running `install.sh` and custom commands,
  and creating dotfile symlinks. `InstallDotfilesOnly` is the export-mode path.
- **`internal/ui`** — The Bubble Tea `Model`. Wraps `bubbles/list` with a custom
  delegate for per-item keybindings (space/e/a/i/x) and `huh` forms for
  create/edit/add/import. Holds selection state and signals install intent back to
  `main.go` via `ShouldInstall()` / `GetSelected()` / `GetExportMode()`.

## Entities and data model

The on-disk contract is `~/dotfiles/modules/<name>/config.yaml`, deserialized into
`models.ModuleConfig`:

- `name`, `description` — identity.
- `dependencies: []string` — names of modules that must install first.
- `packages.common: []string` — package names identical across managers.
- `packages.specific: [{name, manager}]` — per-manager package names (`brew`/`apt`/…).
- `commands: [{command, os}]` — custom shell commands; `os` empty = run everywhere,
  else gated to a specific package manager.
- `dotfiles: [{source, destination}]` — `source` is relative to the module dir,
  `destination` is a home-relative or `~`/`$HOME`-prefixed path.

A module on disk is the directory + `config.yaml` + optional `install.sh` +
a `dotfiles/` subtree. There is no database; the filesystem is the source of truth.

## Internal data flow

1. **Startup** (`main.go`): resolve dotfiles path → ensure `modules/` exists →
   `scanner.ScanModules()` → build `ui.Model` → run Bubble Tea program (alt screen).
2. **Browse/edit** (`ui`): the user navigates the list, toggles selection (which
   auto-selects dependencies), and may create/edit modules or add/import dotfiles
   through `huh` forms; each mutating action calls into `manager` then reloads the
   module list via the `scanner`.
3. **Install** (post-TUI, `main.go`): on quit-with-install, `installer.GetInstallationOrder`
   produces a dependency-respecting order; a goroutine installs each module
   (`InstallModule` or, in export mode, `InstallDotfilesOnly`) and streams
   `InstallationStatus` over a channel that the main goroutine prints. First failure
   aborts the run with a non-zero exit.

## Key libraries

- **`github.com/charmbracelet/bubbletea`** — The TUI runtime (Elm-style
  model/update/view) that drives the whole interactive interface.
- **`github.com/charmbracelet/bubbles`** — Provides the `list` component used as the
  module browser, with filtering, pagination, and a custom item delegate.
- **`github.com/charmbracelet/huh`** — Form library for the create/edit/add/import
  flows (inputs, multi-selects, notes).
- **`github.com/charmbracelet/lipgloss`** — Styling/layout for titles, status, and
  error rendering.
- **`gopkg.in/yaml.v3`** — Marshals/unmarshals each module's `config.yaml`.

## Critical notes for collaborators

1. **Package-manager support is brew + apt only, despite the README.** The README
   advertises brew/apt/pacman/yum/snap, but `installer.detectPackageManager` only probes
   `brew` and `apt` (in that order), and `installWithPackageManager` only implements
   those two. pacman/yum/snap are **not** wired up — treat the README list as aspirational.
2. **Importing a dotfile is destructive.** `manager.ImportDotfileWithDestination` copies
   the source into the module, **deletes the original** (`os.RemoveAll`), then creates a
   symlink back. There is a best-effort restore if the symlink step fails, but a crash
   between the delete and the symlink leaves the file living only inside the module.
   Test import flows against throwaway paths (set `DOTFILES_PATH`).
3. **Create/edit forms cap "specific packages" and "specific commands" at 2 each.** The
   `huh` forms expose exactly two slots per category. Editing a module whose
   `config.yaml` declares more than two will not show the extras, and saving rewrites the
   config from the form — silently dropping them. Hand-edit `config.yaml` for modules
   that need more.
