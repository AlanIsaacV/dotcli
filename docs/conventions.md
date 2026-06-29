# Conventions

## Code style

- **Language**: Go 1.24 (module `github.com/AlanIsaacV/dotcli`)
- **Formatter**: gofmt (`gofmt -w .`)
- **Linter**: none configured. `go vet ./...` is the de-facto check. [INFERRED]
- **Imports**: standard-library group first, third-party group second, separated by a
  blank line (gofmt/goimports ordering). [INFERRED]

## Error handling

Errors are returned, never panicked, in library packages. Functions wrap context with
`fmt.Errorf("...: %w", err)` so callers can unwrap. The TUI layer stores the error on
`Model.error` and renders it; the post-TUI install loop in `main.go` uses `log.Fatal`
for unrecoverable startup failures and a non-zero `os.Exit(1)` when an install fails.
A module that fails to parse during scanning is logged as a warning and skipped, not
fatal.

## Logging

No logging framework. The program communicates through:
- The TUI itself (status/error styles in `internal/ui`).
- stdout during installation, via the `models.InstallationStatus` channel printed in
  `main.go` (emoji-prefixed lines: 📦 progress, ✅ done, ❌ error).
- `log.Fatal` for fatal startup errors.

## Testing

- **Test runner**: `go test ./...`
- **Location**: no test files exist yet (`*_test.go` alongside source is the Go
  convention to adopt). [INFERRED]
- **Conventions**: `DOTFILES_PATH` env var exists specifically to point the binary at a
  throwaway dotfiles root for manual/automated testing (see `main.go`).

> TODO: review manually. There is currently no test suite; the convention above is the
> recommended target, not an observed practice.

## API design

Not applicable — this is a local TUI binary, not a networked service. There are no HTTP
endpoints. The "public surface" is the keybinding set (see `docs/features/`) and the
`config.yaml` schema (see `docs/architecture.md` → Entities and data model).

## Naming

- **Files**: one package per directory under `internal/`, file named after the package
  (`scanner.go`, `manager.go`, …). lowercase, no underscores.
- **Functions**: exported `CamelCase` for the package API (`ScanModules`,
  `InstallModule`, `AddDotfile`), unexported `camelCase` for helpers (`expandPath`,
  `createSymlink`).
- **Endpoints**: n/a (no HTTP). Keybindings are declared as `key.Binding` values in
  `internal/ui` (`newDelegateKeyMap`, `newListKeyMap`).

---

> Sections marked `[INFERRED]` or `> TODO` should be reviewed manually — they are
> inferences from the code at init time.
