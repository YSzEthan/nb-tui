# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`NetBird-TUI` is a Go TUI + CLI for managing NetBird VPN connections. It shells out to the `netbird` CLI and system service manager rather than linking against any NetBird library. Supported platforms: **Linux** and **macOS**.

## .gitignore

`*.json` is globally ignored — NetBird profile files contain private keys and must never be committed.

## Dependencies

Runtime: `netbird` CLI, `sudo`  
Go: Bubble Tea (`bubbletea` + `bubbles` + `lipgloss`), Cobra

## Build / Run

```bash
make build                            # produces ./NetBird-TUI
make install                          # installs to ~/.local/bin/NetBird-TUI
go build ./...                        # syntax + type check
go vet ./...                          # vet all packages
./NetBird-TUI                         # launch TUI
./NetBird-TUI status                  # CLI subcommand
```

No test suite exists yet.

## Architecture

### Package layout

```
cmd/netbird-tui/   — Cobra root + one file per subcommand (status, up, down, save, use, …)
internal/
  netbird/         — shells out to `netbird` CLI; Status/Peer types; sudo helpers
  config/          — detect active config path, profile CRUD (~/.config/netbird-profiles/)
  svc/             — service start/stop via build-tag files (svc_linux.go / svc_darwin.go)
  ui/              — Bubble Tea model, views, commands (messages/Cmds)
```

### Bubble Tea model (`internal/ui/`)

- `app.go` — `model` struct, `Init`/`Update`/`View`, five `viewState` values (`viewMain`, `viewSwitch`, `viewPeers`, `viewSave`, `viewConfirm`)
- `commands.go` — all `tea.Cmd` factories; each wraps one side-effecting operation and returns a typed message (`statusMsg`, `actionDoneMsg`, `errMsg`, …)
- `styles.go` — all `lipgloss` styles in one place

`fetchStatusCmd` is the main polling driver: called on `Init`, after every action, and by `tickCmd` every 2 seconds. It calls `netbird.GetStatus()` + `svc.IsActive()` + `config.CurrentProfileName()` and returns a `statusMsg`.

### NetBird shelling (`internal/netbird/`)

- `GetStatus()` runs `netbird status --json` and parses into `Status`.
- `ManagementURL` has a custom `UnmarshalJSON` that handles both the legacy string shape and the NetBird 0.27+ object shape `{Scheme, Host, Path}`.
- `IsLoggedIn()` uses `DaemonStatus`: `NeedsLogin`/`LoginFailed`/empty → false. Empty daemonStatus falls back to `PrivateKey` presence.
- `StartKeepalive` runs a goroutine that refreshes `sudo -n -v` every N seconds so interactive sudo prompts don't appear mid-session.

### Config path detection (`internal/config/detect.go`)

`DetectActive()` probes `/var/lib/netbird/default.json` (NetBird 0.27+) then `/etc/netbird/config.json` (legacy), then parses `systemctl cat netbird` for a `--config` flag. **Never hardcode the config path.**

### Platform abstraction (`internal/svc/`)

Build-tagged files: `svc_linux.go` uses `systemctl`; `svc_darwin.go` uses `launchctl`. All service start/stop/check calls must go through `svc.IsActive()` / `svc.Start()` / `svc.Stop()`.

### Profile management (`internal/config/profile.go`)

Profiles are user-owned JSON files at `~/.config/netbird-profiles/<name>.json` (mode 600). `SaveProfile` copies `$ACTIVE` there; `WriteActive` copies a profile back via `sudo install`. Identity matching (for `CurrentProfileName`) uses SHA-256 of `ManagementURL|PrivateKey|SetupKey`.
