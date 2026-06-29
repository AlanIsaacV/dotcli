# Operational runbook

> DotCLI is a local single-user CLI/TUI, not a deployed service. "Operations" here
> means build, install, and run on a developer machine — there is no server, no
> container, and nothing to deploy or monitor in production.

## Build & install

```bash
# Build a binary into bin/ (bin/ is gitignored)
go build -o bin/dotcli .

# Run from source
go run .

# Run the built binary
./bin/dotcli
```

The binary expects a dotfiles root at `~/dotfiles` (created automatically with a
`modules/` subdir on first run if missing). Override the root with the `DOTFILES_PATH`
env var.

### Detected targets

No deployment targets — no Dockerfile, no Cloud Build/Run config, no CI workflow
detected at init time.

## Env vars / configuration

| Variable | Description | Required |
| -------- | ----------- | -------- |
| `DOTFILES_PATH` | Path to the dotfiles root. Defaults to `~/dotfiles`. Primarily an override for testing against a throwaway directory. | No |
| `HOME` | Used (via `os.UserHomeDir`) to resolve the default dotfiles root, expand `~`/`$HOME` in dotfile destinations, and compute home-relative config paths. | Yes (provided by the OS) |

> No secrets and no config files are read by this tool beyond each module's
> `config.yaml`.

## Runtime requirements

- **Go 1.21+** to build (go.mod declares 1.24).
- A supported package manager on PATH for the install pipeline: **brew** or **apt**
  (`detectPackageManager` only probes these two, in that order, despite the README
  mentioning pacman/yum/snap). [INFERRED]
- `bash` to run module `install.sh` scripts and custom commands (or `cmd` on Windows).
- `sudo` is invoked non-interactively for `apt-get update` / `apt-get install -y`; on a
  machine without passwordless sudo this will block or fail.

## Common operational issues

- **Symptom**: "no supported package manager found" → **Diagnosis**: neither `brew`
  nor `apt` is on PATH → **Resolution**: install one, or use export mode (`x`) to apply
  dotfiles only.
- **Symptom**: "circular dependency detected: X" → **Diagnosis**: two modules depend on
  each other (directly or transitively) → **Resolution**: break the cycle in the
  modules' `config.yaml` `dependencies`.
- **Symptom**: import of `~/.foo` fails midway → **Diagnosis**: import copies → deletes
  original → symlinks; if the symlink step fails it attempts to restore the copy →
  **Resolution**: check the reported error; verify the original was restored before
  retrying.

## External dependencies

- **3rd-party APIs**: none.
- **Internal services**: none.
- **Infra**: none. The only external touch points are the host filesystem (`~/dotfiles`
  and symlink targets in `$HOME`) and the host package manager (brew/apt).
