# Netsoryn

> A terminal dashboard for system, network, and service diagnostics.

```
 ⬡ NETSORYN                                                         dev
 <1> DASH  <2> PROC  <3> NET  <4> PORTS  <5> SVC  <6> DCKR  <7> DNS  <8> HTTP  <9> TCP  <0> SSH
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
| **Dashboard** | `1` | CPU per-core, RAM, swap, disk usage + I/O, load avg, uptime |
| **Processes** | `2` | Top processes by CPU/MEM, sortable, filterable |
| **Network** | `3` | Interfaces + I/O, active connections |
| **Ports** | `4` | Open/listening ports with owning process |
| **Services** | `5` | systemd (Linux) / launchd (macOS) / SCM (Windows) service states |
| **Docker** | `6` | Containers — name, image, state, ports |
| **DNS** | `7` | Interactive DNS resolver (A, AAAA, MX, NS, CNAME) |
| **HTTP** | `8` | HTTP/TLS endpoint checker with latency and cert info |
| **TCP** | `9` | Interactive TCP port reachability probe (host:port) |
| **SSH** | `0` | Configured SSH hosts; connect or run a remote Netsoryn dump |

## Installation

### Pre-built binaries

Download a binary from the GitHub Releases page.

**Linux**

```bash
# tar.gz
tar xzf netsoryn_*_linux_amd64.tar.gz
chmod +x netsoryn
sudo mv netsoryn /usr/local/bin/

# deb
sudo dpkg -i netsoryn_*_linux_amd64.deb

# rpm
sudo rpm -i netsoryn_*_linux_amd64.rpm
```

**macOS**

```bash
tar xzf netsoryn_*_darwin_arm64.tar.gz   # Apple Silicon
# tar xzf netsoryn_*_darwin_amd64.tar.gz  # Intel
chmod +x netsoryn
sudo mv netsoryn /usr/local/bin/
```

> Release binaries are not notarized. If macOS blocks the binary, run `xattr -d com.apple.quarantine ./netsoryn` or right-click → **Open** in Finder.

**Windows**

Download `netsoryn_*_windows_amd64.zip`, extract, and run `netsoryn.exe`.

tar.gz and zip archives include `config.example.yaml`. System packages do not — see the Configuration section below.

### Go install

Requires Go 1.26+.

```bash
go install github.com/vitlixe/netsoryn/cmd/netsoryn@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

### From source

Requires Go 1.26+.

```bash
git clone https://github.com/vitlixe/netsoryn.git
cd netsoryn
make run     # build and launch immediately
make build   # build only → dist/netsoryn
make install # install to $GOPATH/bin
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1`–`9` / `0` | Jump to view |
| `Tab` / `Shift+Tab` | Cycle views |
| `j` / `k` | Navigate down / up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+d` / `Ctrl+u` | Page down / up |
| `/` | Filter (type, then Enter) |
| `Esc` | Clear filter / close input |
| `s` | Cycle sort column (processes view) |
| `h` / `←` / `l` / `→` | Switch sub-tabs (network view) |
| `n` | New query (DNS / HTTP / TCP views) |
| `D` / `d` | Remove first result (DNS / HTTP / TCP views) |
| `a` | Add host (SSH view) |
| `Enter` | Connect to selected host (SSH view) |
| `?` | Toggle help overlay |
| `q` | Quit |

## Configuration

Config file locations — first match wins:

| Platform | Path |
|----------|------|
| Linux | `~/.config/netsoryn/config.yaml` |
| macOS | `~/Library/Application Support/netsoryn/config.yaml` |
| Windows | `%APPDATA%\netsoryn\config.yaml` |
| All | `/etc/netsoryn/config.yaml` (system-wide) |

Create a config from the example:

```bash
# from source
mkdir -p ~/.config/netsoryn
cp configs/netsoryn.example.yaml ~/.config/netsoryn/config.yaml
```

From a release archive, use `config.example.yaml` instead of `configs/netsoryn.example.yaml`.

Environment variables can override scalar config values with the `NETSORYN_` prefix, for example `NETSORYN_REFRESH_INTERVAL=5`.

Duration values use seconds by default (`refresh_interval: 5` means 5 seconds). String values must be valid Go duration strings: `"500ms"`, `"2s"`, `"1m30s"`.

```yaml
refresh_interval: 2          # seconds between auto-refresh
default_view: dashboard      # view on startup
process_limit: 50            # max processes to display
ports_listen_only: true      # show only LISTEN sockets

http_checks:
  - url: "https://example.com"
    timeout: 10                # seconds

dns_checks:
  - domain: "example.com"
    servers: ["8.8.8.8:53", "1.1.1.1:53"]

ssh_hosts:
  - name: "prod"
    host: "prod.example.com"
    user: "deploy"
    key: "~/.ssh/prod_ed25519"
    options: ["-o", "StrictHostKeyChecking=accept-new"]
```

See `configs/netsoryn.example.yaml` for all options.

## SSH helpers

Netsoryn can use configured SSH host aliases for quick read-only diagnostics. It does not implement SSH itself or store passwords — it executes your system `ssh` binary, so keys, agents, passphrases, and `known_hosts` stay under OpenSSH control. In the TUI, open the SSH view with `0`, press `a` to add a host, select a host with `j/k`, press `Enter` to connect, or `d` to run `netsoryn dump -f text --sections system,ports,services` remotely.

```bash
netsoryn ssh list
netsoryn ssh prod
netsoryn ssh prod -- uname -a
netsoryn ssh dump prod -f text
netsoryn ssh dump prod --sections system,ports,services --pretty
```

`netsoryn dump` defaults to the `system` section. Use `--sections system,ports,processes,network,services,docker` or `--sections all` for broader snapshots.

## Project Structure

```
netsoryn/
├── cmd/netsoryn/        # entry point (cobra CLI)
├── internal/
│   ├── config/          # viper-based config loading
│   ├── collectors/      # data sources
│   │   ├── system.go    # CPU, memory, disk, load (gopsutil)
│   │   ├── process.go   # process list (gopsutil)
│   │   ├── network.go   # interfaces + connections (gopsutil)
│   │   ├── ports.go     # listening ports (gopsutil)
│   │   ├── services.go  # systemd / launchd / SCM (Windows)
│   │   ├── docker.go    # docker CLI
│   │   ├── dns.go       # net.Resolver
│   │   └── http_check.go # net/http
│   └── ui/
│       ├── model.go     # root bubbletea model (layout, routing)
│       ├── splash.go    # startup splash screen
│       ├── styles/      # lipgloss colour palette + helpers
│       ├── keys/        # keybinding definitions
│       └── views/       # one Model per view
├── configs/             # example config
└── .github/workflows/   # CI (test, lint, cross-build, release)
```

## Safety Notes

- **Read-only by default.** No writes, no restarts, no destructive actions.
- **No root required for the basics.** CPU, RAM, disk, DNS, HTTP — all work without elevated privileges. Some features (all process details, low-numbered port ownership) may show partial data without root.
- **Diagnostic only.** No exploit features, no brute-force, no stealth scanning, no firewall evasion.

## License

MIT. See LICENSE.
