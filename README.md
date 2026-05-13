# Netsoryn

> A terminal dashboard for system, network, and service diagnostics.

```
 ⬡ NETSORYN                                                         dev
 <1> DASH  <2> PROC  <3> NET  <4> PORTS  <5> SVC  <6> DOCKER  <7> DNS  <8> HTTP
╭── Dashboard ────────────────────────────────────────────────────────╮
│  CPU                                                                │
│  Total    ████████░░░░░░░░   45.2%                                  │
│  Core 0   ██████░░░░░░░░░░   38.1%                                  │
│  Core 1   ██████████░░░░░░   60.3%                                  │
│                                                                     │
│  Memory                                                             │
│  RAM      ████████████░░░░   7.2 GB / 16 GB                         │
│  Swap     ██░░░░░░░░░░░░░░   0.5 GB / 4 GB                          │
│                                                                     │
│  System                                                             │
│  Load avg  2.45  1.89  1.23                                         │
│  Uptime    5d 3h 22m                                                │
╰─────────────────────────────────────────────────────────────────────╯
 <tab> next view   ? help   q quit
```

## Features

| View | Key | What you see |
|------|-----|--------------|
| **Dashboard** | `1` | CPU per-core, RAM, swap, disk usage, load avg, uptime |
| **Processes** | `2` | Top processes by CPU/MEM, sortable, filterable |
| **Network** | `3` | Interfaces + I/O, active connections |
| **Ports** | `4` | Open/listening ports with owning process |
| **Services** | `5` | systemd (Linux) / launchd (macOS) / SCM (Windows) service states |
| **Docker** | `6` | Containers — name, image, state, ports |
| **DNS** | `7` | Interactive DNS resolver (A, AAAA, MX, NS, CNAME) |
| **HTTP** | `8` | HTTP/TLS endpoint checker with latency and cert info |

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `1`–`8` | Jump to view |
| `Tab` / `Shift+Tab` | Cycle views |
| `j` / `k` | Navigate down / up |
| `gg` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Page down / up |
| `/` | Filter (type, then Enter) |
| `Esc` | Clear filter / close input |
| `s` | Cycle sort column (processes view) |
| `h` / `l` | Switch sub-tabs (network view) |
| `n` | New query (DNS / HTTP views) |
| `D` | Remove first result (DNS / HTTP views) |
| `?` | Toggle help overlay |
| `q` | Quit |

## Installation

### Quick start

```bash
# requires Go 1.26+
make run            # builds to dist/netsoryn and starts the TUI
```

The startup splash appears first, then Netsoryn opens the main dashboard.

### From source

```bash
# requires Go 1.26+
git clone https://github.com/vitlixe/netsoryn
cd netsoryn
make build          # builds to dist/netsoryn
./dist/netsoryn     # runs the built binary
make install        # installs to $GOPATH/bin
```

### Pre-built binaries

Download from the [Releases](https://github.com/vitlixe/netsoryn/releases) page.

Linux/macOS (tar.gz):

```bash
tar xzf netsoryn_*_linux_amd64.tar.gz
chmod +x netsoryn
sudo mv netsoryn /usr/local/bin/
netsoryn
```

The archive includes `config.example.yaml`. To use it as a starting config:

**Linux:**
```bash
mkdir -p ~/.config/netsoryn
cp config.example.yaml ~/.config/netsoryn/config.yaml
```

**macOS:**
```bash
mkdir -p "$HOME/Library/Application Support/netsoryn"
cp config.example.yaml "$HOME/Library/Application Support/netsoryn/config.yaml"
```

Use the archive that matches your system, for example `netsoryn_*_darwin_arm64.tar.gz` on Apple Silicon Macs.

> **macOS Gatekeeper note:** release binaries are not currently notarized with an Apple Developer ID. macOS may show a warning: *"Apple could not verify that netsoryn is free of malware."* This is a Gatekeeper quarantine on unsigned binaries, not necessarily an indication of malware. To remove the quarantine attribute after extracting:
>
> ```bash
> xattr -d com.apple.quarantine ./netsoryn
> ```
>
> Alternatively, right-click the binary in Finder and choose **Open**.

Linux packages are also available for Debian/Ubuntu and Fedora/RHEL systems.

Debian/Ubuntu:

```bash
sudo dpkg -i netsoryn_*_linux_amd64.deb
netsoryn
```

Fedora/RHEL:

```bash
sudo rpm -i netsoryn_*_linux_amd64.rpm
netsoryn
```

Windows:

1. Download `netsoryn_*_windows_amd64.zip`.
2. Extract the archive.
3. Optionally copy the included `config.example.yaml` to your config directory:

```powershell
New-Item -ItemType Directory -Force "$env:APPDATA\netsoryn"
Copy-Item config.example.yaml "$env:APPDATA\netsoryn\config.yaml"
```

4. Run `netsoryn.exe` from PowerShell:

```powershell
.\netsoryn.exe
```

### Go install

```bash
go install github.com/vitlixe/netsoryn/cmd/netsoryn@latest
netsoryn
```

Make sure `$GOPATH/bin` is in your `PATH`.

## Configuration

The release archive includes `config.example.yaml`. If you installed from source, use `configs/netsoryn.example.yaml` instead.

```bash
# from a release archive (Linux)
mkdir -p ~/.config/netsoryn
cp config.example.yaml ~/.config/netsoryn/config.yaml

# from source
mkdir -p ~/.config/netsoryn
cp configs/netsoryn.example.yaml ~/.config/netsoryn/config.yaml
```

Config is loaded from the first file found, in order:

| Platform | Path |
|----------|------|
| Linux | `~/.config/netsoryn/config.yaml` |
| macOS | `~/Library/Application Support/netsoryn/config.yaml` |
| Windows | `%APPDATA%\netsoryn\config.yaml` |
| All | `~/.config/netsoryn/config.yaml` (fallback) |
| All | `/etc/netsoryn/config.yaml` (system-wide) |

Environment variables override config values: prefix with `NETSORYN_` (e.g. `NETSORYN_REFRESH_INTERVAL=5`).

Key options:

```yaml
refresh_interval: 2          # seconds between auto-refresh
default_view: dashboard      # view on startup
process_limit: 50            # max processes to display
ports_listen_only: true      # show only LISTEN sockets

http_checks:
  - url: "https://example.com"
    timeout: 10

dns_checks:
  - domain: "example.com"
    servers: ["8.8.8.8:53", "1.1.1.1:53"]
```

## Design principles

- **Read-only by default.** No writes, no restarts, no destructive actions.
- **No root required for the basics.** CPU, RAM, disk, DNS, HTTP — all work without elevated privileges. Some features (all process details, low-numbered port ownership) may show partial data without root.
- **Diagnostic only.** No exploit features, no brute-force, no stealth scanning, no firewall evasion.

## Architecture

```
netsoryn/
├── cmd/netsoryn/        # entry point (cobra CLI)
├── internal/
│   ├── config/          # viper-based config loading
│   ├── collectors/      # data sources (implement Collector interface)
│   │   ├── system.go    # CPU, memory, disk, load (gopsutil)
│   │   ├── process.go   # process list (gopsutil)
│   │   ├── network.go   # interfaces + connections (gopsutil)
│   │   ├── ports.go     # listening ports (gopsutil)
│   │   ├── services.go  # systemd / launchd / SCM (Windows)
│   │   ├── docker.go    # docker CLI
│   │   ├── dns.go       # net.Resolver
│   │   └── http_check.go# net/http
│   └── ui/
│       ├── model.go     # root bubbletea model (layout, routing)
│       ├── splash.go    # startup splash screen
│       ├── styles/      # lipgloss colour palette + helpers
│       ├── keys/        # keybinding definitions
│       └── views/       # one Model per view
├── configs/             # example config
└── .github/workflows/   # CI (test, lint, cross-build, release)
```

### Adding a collector

1. Implement the `collectors.Collector` interface in `internal/collectors/`.
2. Create a new view in `internal/ui/views/` with a matching `tea.Model`.
3. Register the view in `internal/ui/model.go`.

## Dependencies

| Package | Purpose |
|---------|---------|
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) | Table, textinput, viewport components |
| [shirou/gopsutil](https://github.com/shirou/gopsutil) | Cross-platform system metrics |
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [spf13/viper](https://github.com/spf13/viper) | Configuration |
| [rs/zerolog](https://github.com/rs/zerolog) | Structured logging |

## License

MIT — see [LICENSE](LICENSE).
