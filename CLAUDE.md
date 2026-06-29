# DotCLI

A TUI dotfiles manager: discovers modules under `~/dotfiles/modules`, resolves their
dependencies, and installs them — packages (brew/apt), `install.sh`, custom commands,
and dotfile symlinks.

## For this session, consider reading:

- Architecture and modules → `docs/architecture.md`
- Conventions (style, errors, logging, tests) → `docs/conventions.md`
- Build, run, env vars → `docs/runbook.md`
- Documented features → `docs/features/_index.md`
- Architectural decisions (ADRs) → `docs/decisions/_index.md`

## Critical rules

- **Package-manager support is brew + apt only.** The README lists pacman/yum/snap too,
  but the code (`internal/installer`) only implements brew and apt. Don't assume the
  others work.
- **Importing a dotfile deletes the original** before symlinking it back
  (`internal/manager`). Always test import flows with a throwaway `DOTFILES_PATH`.
- **Create/edit forms cap specific packages & commands at 2 slots each** — editing a
  module with more silently drops them on save. Hand-edit `config.yaml` for those.

## Local commands

```bash
# Run from source
go run .

# Build
go build -o bin/dotcli .

# Tests (none exist yet)
go test ./...

# Format
gofmt -w .
# Lint (no linter configured; vet is the de-facto check)
go vet ./...
```
