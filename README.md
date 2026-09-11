<div align="center">

# Webux

**Self-hosted Linux server management panel — one static binary, no runtime dependencies, no telemetry.**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://go.dev)
[![Platforms](https://img.shields.io/badge/platforms-amd64%20%7C%20386%20%7C%20arm64%20%7C%20armv7-lightgrey.svg)](#supported-distributions-and-architectures)

*A fork of [brendan4linux/webux](https://github.com/brendan4linux/webux), maintained at [sincerely-hello-world/webux](https://github.com/sincerely-hello-world/webux).*
*Original author: **brendan4linux** — see [License](#license) for attribution.*

</div>

---

> ### 📖 Which document should I read?
>
> | Document | Contents |
> | --- | --- |
> | **`README.md`** — this file | What Webux is, how to install it, what the released artifacts actually contain |
> | **[`readme_new.md`](readme_new.md)** | The full developer guide (11 chapters, Chinese): toolchain, project layout, daily workflow, test gates, debugging, GoReleaser internals, deployment |
>
> **中文读者**：本文以英文为主，文末有一节 **[中文速览](#中文速览)**，涵盖安装、构建命令、架构与许可要点。

---

## Screenshots

<table>
  <tr>
    <td align="center">
      <img src="docs/screenshots/dashboard.png" alt="Dashboard" width="480"><br>
      <sub><b>Dashboard</b> — live CPU, memory, disk and uptime</sub>
    </td>
    <td align="center">
      <img src="docs/screenshots/services.png" alt="Services" width="480"><br>
      <sub><b>Services</b> — systemd unit management</sub>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="docs/screenshots/processes.png" alt="Processes" width="480"><br>
      <sub><b>Processes</b> — live /proc scanner</sub>
    </td>
    <td align="center">
      <img src="docs/screenshots/interfaces.png" alt="Network Interfaces" width="480"><br>
      <sub><b>Network Interfaces</b> — live bandwidth sparklines</sub>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="docs/screenshots/terminal.png" alt="Terminal" width="480"><br>
      <sub><b>Terminal</b> — full PTY browser terminal with quick commands</sub>
    </td>
    <td align="center">
      <img src="docs/screenshots/containers.png" alt="Containers" width="480"><br>
      <sub><b>Containers</b> — Docker and Podman management</sub>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="docs/screenshots/webservers.png" alt="Webservers" width="480"><br>
      <sub><b>Webservers</b> — Nginx, Apache, Caddy status and config</sub>
    </td>
    <td align="center">
      <img src="docs/screenshots/migrationtemplate.png" alt="Migration Template" width="480"><br>
      <sub><b>Migration Template</b> — full server snapshot export</sub>
    </td>
  </tr>
</table>

---

## What is Webux?

Webux is a self-hosted Linux server management panel that runs as a **single static binary** with an embedded web UI, embedded SQLite database, and no external runtime dependencies. Install it in seconds on any Linux server — from RHEL 6 to the latest Arch — and get a full management panel immediately.

It is opinionated about being lightweight: no Docker required to run it, no systemd mandatory, no cloud account, no telemetry, no license server. Just a binary.

**Webux also teaches you Linux as you use it.** Every action — starting a service, extending a disk, running an Ansible playbook — emits the equivalent shell command to a Learn Mode drawer at the bottom of the page. A ▶ play button runs any command directly in the built-in terminal. It's a practical way to get comfortable with Linux administration without leaving the UI.

---

## Features

### System

| Feature                  | Description                                                                                                                                                                                                                                                                                              |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Dashboard**      | Live CPU, memory, disk, load average, uptime — updates in real time over WebSocket. Includes a**health check panel** showing the server's current status at a glance                                                                                                                              |
| **Health Checks**  | Six configurable pass/fail checks run on every dashboard load: kernel age, failed services, swap pressure, disk usage, CPU load, and pending security updates. Each check expands to show detail output. Fully customisable — add, edit, or remove checks via the Settings page using any shell command |
| **Services**       | systemd/OpenRC unit management — start, stop, enable, disable, view logs. Shows all units including disabled ones                                                                                                                                                                                       |
| **Processes**      | Live`/proc` scanner — CPU%, memory, PID, user, full command line                                                                                                                                                                                                                                      |
| **Disks**          | Block device tree, partition layout, mount usage bars. LVM-aware: shows Volume Groups, free space, and offers**online filesystem extension** (ext3/4, XFS, Btrfs) when VG free space is available — no reboot required                                                                            |
| **Users & Groups** | Full CRUD for Linux users and groups via`useradd`/`usermod`/`groupadd`                                                                                                                                                                                                                             |

### Network

| Feature                   | Description                                                                                                                                                                                  |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Ports & Sockets** | Reads`/proc/net/tcp*`, `/proc/net/udp*` directly — no `ss` or `netstat` needed. Cross-references `/proc/<pid>/fd` to show owning process. Enriches with systemd socket unit names |
| **Interfaces**      | Network interface list with live bandwidth sparklines (SSE streaming)                                                                                                                        |
| **Firewall**        | ufw, nftables, and iptables rule viewer and management                                                                                                                                       |

### Applications

| Feature              | Description                                                                                                                                                        |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Containers** | Docker and Podman via their Unix sockets — list, start, stop, remove                                                                                              |
| **Databases**  | Auto-detect MySQL/MariaDB, PostgreSQL, Redis. Inline query panel                                                                                                   |
| **Webservers** | Nginx, Apache, Caddy — status, config editor, reload, virtual host list                                                                                           |
| **Packages**   | pacman, apt, dnf/yum — install, remove, upgrade, search. Flatpak support.**Repository management**: add/remove/enable/disable repos, manage Flatpak remotes |
| **Files**      | Full file browser with inline editor and save-to-disk                                                                                                              |
| **Cron**       | System and per-user crontab viewer and editor                                                                                                                      |

### Automation

| Feature           | Description                                                                                                                                                                                                                                                   |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Ansible** | Scans a configurable playbook directory. Parses declared variables (`vars:`, `vars_prompt:`) and renders input boxes. Runs playbooks with live SSE output streaming. If Ansible is not installed, offers one-click install via the native package manager |
| **Puppet**  | Reads puppet.conf, views facts, catalog status, last run report. Supports AIO (`/opt/puppetlabs/bin/puppet`) and distro package installs                                                                                                                    |

### Tools

| Feature                      | Description                                                                                                                                                                                                                                                                                                                              |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Migration Template** | Snapshots everything needed to replicate a server: ports, services, databases, webserver vhosts, cron, users, firewall rules, env vars, Puppet facts. Exports as Markdown checklist, annotated YAML, or Ansible playbook skeleton                                                                                                        |
| **Terminal**           | Full PTY terminal in the browser (xterm.js). Spawns the user's login shell. Quick-command chips configurable in settings.**Play button** in Learn Mode runs any CLI-equivalent command directly in the terminal                                                                                                                    |
| **AI Assistant**       | Ollama-first (self-hosted, no API key needed). Includes a setup wizard, model browser with RAM requirements, and one-click model pull with progress streaming. Also supports OpenAI, Anthropic, and any OpenAI-compatible endpoint. Every chat message automatically injects live system context (CPU, RAM, failed services, open ports) |
| **Learn Mode**         | Every action emits its CLI shell equivalent to a collapsible pane at the bottom of every page. Each command has a**▶ play button** that runs it in the terminal — navigate to the terminal automatically if needed                                                                                                               |

---

## Health Checks

The dashboard includes a real-time health panel that runs six checks on every load:

| Check            | Pass condition                   | Detail on expand              |
| ---------------- | -------------------------------- | ----------------------------- |
| Kernel / uptime  | Uptime under 90 days             | Shows uptime, kernel version  |
| System services  | No failed systemd units          | Lists any failed unit names   |
| Memory / swap    | Swap usage under 75%             | Shows RAM and swap used/total |
| Disk usage       | All real filesystems under 80%   | Lists over-threshold mounts   |
| CPU load         | 15-min load under CPU core count | Shows load avg and core count |
| Security updates | No pending security patches      | Runs distro-appropriate check |

Checks are fully configurable in **Settings → Health Checks**. Each check is a shell command where exit code 0 = pass and non-zero = fail. Add your own checks for anything that matters on your servers — a running daemon, a reachable endpoint, a file's existence, a certificate's expiry date.

---

## Authentication

Two mutually exclusive backends, chosen at build time. **Everything published in the GitHub Releases
is the default (shadow) build.**

| Backend                    | How to get it                                                          | Build tag               | Capability                                                                   |
| -------------------------- | ---------------------------------------------------------------------- | ----------------------- | ---------------------------------------------------------------------------- |
| **shadow** (default) | `mise run build`, `mise run snapshot`, every deb / rpm / pacman        | —                      | Reads`/etc/shadow` and verifies the hash — local Linux accounts only     |
| **PAM**              | `mise run build-pam`, or `scripts/build-pam-*.sh`                    | `pam` + `CGO_ENABLED=1` | The full PAM stack: LDAP, SSSD, Kerberos, TOTP/2FA — whatever `/etc/pam.d/webux` says |

`webux --version` tells you which one you have:

```
webux 0.1.1 (b2483c1) built 2026-…
auth backend: shadow (rebuild with -tags pam for full PAM support)
```

### shadow backend (default)

Pure Go — no CGO, no `libxcrypt`, which is why the fully static cross-compiled binaries work:

| Hash                              | Support                                              |
| --------------------------------- | ---------------------------------------------------- |
| yescrypt `$y$`                  | ✅ via`github.com/openwall/yescrypt-go`           |
| SHA-512 `$6$`                   | ✅ implemented in this repo                          |
| SHA-256 `$5$`                   | ✅ implemented in this repo                          |
| MD5 `$1$`                       | ✅ implemented in this repo                          |
| bcrypt `$2a$` / `$2b$`        | ✅ via`golang.org/x/crypto/bcrypt`                |

Locked accounts (`*`, `!…`) are rejected, and hashes are compared in constant time.

> The generated `.deb` / `.rpm` still declare `libxcrypt2 | libcrypt1 | libc6` (deb) and `libxcrypt`
> (rpm) purely as a safety net. Those libraries ship on every modern distro, so `apt install` and
> `dnf install` never actually have to pull anything.

### PAM backend (optional, built separately)

```bash
sudo apt install libpam0g-dev      # Debian / Ubuntu
sudo dnf install pam-devel         # RHEL / Fedora        (Arch: sudo pacman -S pam)

mise run build-pam                 # -> build/webux-pam  (amd64, host build, needs the headers)
```

**PAM binaries are never part of the released packages.** `mise run snapshot` and `mise run release`
always produce `CGO_ENABLED=0` shadow builds — `.goreleaser.yaml` contains no reference to `pam` at
all. The PAM variant links the *target machine's* `libpam.so`, so it cannot be statically
cross-compiled the way the other artifacts are. Concretely, PAM is:

- **never packaged** — there is no `webux-pam-*.deb`, `.rpm` or `.pkg.tar.zst`;
- **amd64 only** — both `scripts/build-pam-ubuntu.sh` and `scripts/build-pam-rhel.sh` hardcode
  `webux-pam-linux-amd64`;
- **distributed by hand** — not listed in `checksums.txt`, no CI job, and the two scripts only print a
  `gh release upload …` hint when they finish.

Enable it by copying the stack template (see [PAM configuration](#pam-configuration)):

```bash
sudo cp scripts/webux.pam /etc/pam.d/webux
```

> ⚠️ **Do not skip that copy.** The PAM build passes the service name `webux`, and if that stack does
> not return `PAM_SUCCESS` it unconditionally retries with the service name `login`. In practice a
> missing `/etc/pam.d/webux` therefore falls back to whatever `/etc/pam.d/login` allows — which is how
> a 2FA-protected webux stack gets bypassed by a login stack without 2FA.
>
> The lookup order is `/etc/pam.d/webux` → `/usr/lib/pam.d/webux` → `/etc/pam.d/other` →
> `/etc/pam.d/login`. If none of them exist, logins fail with **401** — the PAM build does *not*
> degrade gracefully to the shadow backend.

### Sessions

Sessions are **JWT** (HS256, 24-hour HttpOnly cookie). The JWT secret is auto-generated on first run
and stored in SQLite. Every API endpoint except `/auth/login` and the static assets requires a valid
JWT, and WebSocket connections are gated the same way.

### SSO bypass token

For integration with internal SSO systems, set a bypass token in `config.yaml` or the Settings page:

```yaml
auth:
  bypass_token: "your-long-random-token-here"
```

Your SSO system redirects users to:

```
http://yourserver:8989/auth/bypass?token=<token>
```

Webux issues a real JWT session and redirects to the dashboard. The token can also be passed as an
`X-Webux-Token` HTTP header for API access.

---

## Quick start

```bash
git clone https://github.com/sincerely-hello-world/webux
cd webux
mise install             # pinned toolchain: Go 1.26, air, nub, goreleaser (see mise.toml)
mise run setup           # go mod tidy + frontend deps (nub install, Node 24 LTS via .node-version)
mise run build           # static build (CGO_ENABLED=0) with mysql + postgres drivers -> build/webux

sudo WEBUX_DATA_DIR=/var/lib/webux ./build/webux
# Open https://localhost:8989 and log in with your Linux username and password
```

> The upstream repository is `brendan4linux/webux`. `scripts/install.sh` still defaults to it, so to
> install from this fork's releases use `sudo WEBUX_REPO=<owner>/<repo> sh` (the variable must come
> *after* `sudo`, which clears the environment).

### Install as a service

```bash
mise run install         # builds, then installs the binary, the unit file and a config template
sudo systemctl enable --now webux
```

### Development

```bash
mise run dev             # backend hot-reload (air) + Vite dev server on :5173
mise run test            # go test + svelte-check
mise run ci              # exactly what GitHub Actions runs: gofmt + go vet + go test + svelte-check + goreleaser check
sudo WEBUX_DATA_DIR=/tmp/webux-data ./build/webux --no-auth   # run with auth disabled
```

See **[`readme_new.md`](readme_new.md)** for the full walkthrough (dev ports, inotify limits, debugging
recipes, test coverage status, deployment options).

---

## Build & packaging tasks (mise)

Every command lives in `mise.toml` — there is no Makefile and no justfile.

```bash
mise run build          # Current arch, CGO_ENABLED=0, with mysql + postgres drivers
mise run build-mysql    # MySQL driver only (smaller binary)
mise run build-postgres # PostgreSQL driver only (smaller binary)
mise run build-pam      # Full PAM auth + all DB drivers (requires libpam-dev, CGO_ENABLED=1)

mise run snapshot       # All 21 release artifacts into build/dist/ — no git tag needed
mise run release        # Publish the current git tag (builds + uploads to GitHub Releases)

mise run ci             # Full gate: gofmt + go vet + go test + svelte-check + goreleaser check
mise tasks ls           # List every available task
```

Packaging is declarative — see `.goreleaser.yaml`. It cross-compiles **amd64, 386, arm64 and armv7**
and produces deb, rpm, `.pkg.tar.zst` plus two kinds of tar.gz with **no external tools**:
GoReleaser's bundled nfpm is pure Go, so `fpm`, `rpmbuild` and `bsdtar` are not needed.

### What `mise run snapshot` / `mise run release` produce

4 architectures × 5 files + 1 checksum = **21 artifacts**, all `CGO_ENABLED=0` shadow builds:

| Kind            | Name                                  | Per arch | Notes                                                              |
| --------------- | ------------------------------------- | -------- | ------------------------------------------------------------------ |
| installer tree  | `webux_<ver>_linux_<arch>.tar.gz`   | 4        | **primary artifact** — extracts straight into `/`            |
| binary only     | `webux_<ver>_linux_<arch>_bin.tar.gz` | 4        | one bare`webux` file, no directory prefix                        |
| deb             | `webux_<ver>_<arch>.deb`            | 4        | amd64 / i386 / arm64 / armhf                                       |
| rpm             | `webux-<ver>-1.<arch>.rpm`          | 4        | x86_64 / i386 / aarch64 / armv7hl                                   |
| pacman          | `webux-<ver>-1-<arch>.pkg.tar.zst`  | 4        | x86_64 / i686 / aarch64 / armv7h                                    |
| checksums       | `webux_<ver>_checksums.txt`         | 1        | sha256 of the 20 files above                                        |

> ⚠️ The 32-bit package arch names are **not** uniform: deb and rpm both say `i386`, pacman says
> `i686`, while the tar.gz suffix is GoReleaser's `386`. Grep accordingly.
>
> Both tar.gz flavours contain the *same* binary; only the layout differs. `scripts/install.sh`
> needs the **installer tree** (it reads `etc/webux/config.yaml` as a template), so download the one
> without the `_bin` suffix.

### Package runtime dependency

The default build is pure Go and needs **no** crypt library at runtime. The generated `.deb` and
`.rpm` nevertheless declare `libxcrypt2 | libcrypt1 | libc6` / `libxcrypt` as a dependency as a
safety net — it is pre-installed on every modern Linux distro, so `apt install` and `dnf install`
never actually have to pull it.

The PAM variant is the only build that needs an extra runtime library (`libpam`) on the target
system. It is built separately and is **not** part of the released packages — see
[Authentication](#authentication).

---

## Architecture

```
cmd/webux/
  main.go             # Startup, config, auth wiring, graceful shutdown
  embed.go            # //go:embed dist — entire web UI in the binary

internal/
  api/
    router.go         # chi router — all routes, middleware, auth
    handlers/         # One file per feature: services, disks, packages, ...
  auth/
    auth.go           # JWT (HS256), SSO bypass, login flow, allow-list
    shadow.go         # /etc/shadow parsing + hash verification
    crypt_pure.go     # Pure Go crypt(3): $y$ yescrypt, $6$, $5$, $1$
    crypt_stub.go     # !cgo build-tag shim (both paths resolve to pure Go)
    crypt_cgo.go      # cgo build-tag shim (CGO is no longer needed for shadow)
    pam.go            # PAM via cgo (-tags pam, needs libpam)
    pam_stub.go       # !pam fallback to the shadow backend
    sssd.go           # SSSD / LDAP helpers
  config/config.go    # YAML + environment variable config
  db/
    db.go             # SQLite open + migration runner
    migrations/       # 001_init … 010_fix_sessions SQL files
  learn/              # CLI echo ring buffer + WebSocket broadcast
  system/
    detect.go         # Distro/init system detection
    initsys/          # systemd (dbus) + OpenRC + SysV interfaces
    processes/        # /proc scanner
    users/            # /etc/passwd + shadow + group
    network/
      ports/          # /proc/net scanner
      interfaces/     # Network interface info + bandwidth
      firewall/       # ufw / nftables / iptables
    containers/       # Docker + Podman socket clients
    databases/        # MySQL, PostgreSQL detection + query
    webservers/       # Nginx, Apache, Caddy
    packages/         # pacman, apt, dnf/yum + Flatpak + repo management
    files/            # File browser + editor
    cron/             # Crontab parser + editor
    disks/            # lsblk, df, LVM (pvs/vgs/lvs), filesystem resize
    health/           # Configurable pass/fail system checks
    ansible/          # Playbook scanner, variable extractor, runner
    ai/               # Ollama + OpenAI-compatible chat client
  ws/hub.go           # WebSocket hub — real-time metrics, CLI echoes

web/src/
  App.svelte          # SPA shell, auth check, hash router
  routes/             # One .svelte file per page (20+ pages)
  components/
    Sidebar.svelte    # Navigation
    Topbar.svelte     # Hostname, live indicator, logout
    HealthChecks.svelte # Dashboard health panel with expandable checks
    CLIEchoPane.svelte  # Learn mode — fixed bottom drawer with ▶ play buttons
  lib/
    api.ts            # Typed fetch wrapper
    ws.ts             # WebSocket store
```

---

## Dependency philosophy

Webux aims for the minimum viable set of Go dependencies at runtime:

| Concern                        | Solution                                   | CGO?          |
| ------------------------------ | ------------------------------------------ | ------------- |
| HTTP routing                   | `go-chi/chi`                             | No            |
| WebSocket                      | `gorilla/websocket`                      | No            |
| SQLite                         | `ncruces/go-sqlite3` (WASM via wazero)   | **No**  |
| systemd                        | `godbus/dbus` (no forking `systemctl`) | No            |
| PTY (terminal)                 | `creack/pty`                             | No            |
| bcrypt                         | `golang.org/x/crypto`                    | No            |
| crypt(3) — yescrypt, SHA, MD5 | `openwall/yescrypt-go` + in-repo impls   | **No**  |
| PAM (optional build)           | `libpam` via cgo (`-tags pam`)         | **Yes** |
| YAML config                    | `gopkg.in/yaml.v3`                       | No            |
| Frontend                       | Vite + Svelte 5, embedded at build time    | Dev only      |

Runtime: **one binary + one SQLite file**. The binary is ~21 MB with `-s -w`.

---

## Configuration

`/etc/webux/config.yaml` (created by installer, or pass `--config /path/to/config.yaml`):

```yaml
listen_addr: ":8989"          # Default port — change here or in Settings UI
data_dir: "/var/lib/webux"    # SQLite DB and other runtime data

tls_cert_file: ""             # Leave blank for auto-generated self-signed cert
tls_key_file:  ""             # Or set paths to your own cert/key (Let's Encrypt etc.)

log:
  level: "info"               # debug | info | warn | error

auth:
  bypass_token: ""            # SSO bypass — leave blank to disable
  jwt_secret: ""              # Leave blank — auto-generated on first run
  disabled: false             # Set to true for dev (or use --no-auth flag)
```

All settings can also be set via environment variables:

```bash
WEBUX_LISTEN_ADDR=":9000"
WEBUX_DATA_DIR="/opt/webux/data"
WEBUX_BYPASS_TOKEN="your-token"
WEBUX_AUTH_DISABLED="true"
```

Settings editable in the UI (persisted to SQLite):

- Web UI port
- Ansible playbook directory and inventory file
- Puppet config directory
- SSO bypass token
- AI provider (Ollama URL, model, API keys)
- Terminal shell override and quick commands
- Health check definitions (shell commands, labels, enable/disable)

---

## LVM disk extension

When the Disks page detects LVM and a Volume Group has free space, mounted logical volumes show a **+ Extend** button. The wizard:

1. Shows the LV path, current usage, filesystem type, and available VG free space
2. Accepts a size in GB (capped at VG free space)
3. Previews the exact commands before running
4. Streams `lvextend` + filesystem resize output live

Supported filesystems for online (no unmount) extension:

| Filesystem | Resize tool                                  | Requires mount?                  |
| ---------- | -------------------------------------------- | -------------------------------- |
| ext3       | `resize2fs`                                | No — works on unmounted too     |
| ext4       | `resize2fs`                                | No — works on unmounted too     |
| XFS        | `xfs_growfs <mountpoint>`                  | **Yes** — must be mounted |
| Btrfs      | `btrfs filesystem resize max <mountpoint>` | **Yes** — must be mounted |

---

## Supported distributions and architectures

Webux is a static binary — it runs on any Linux with kernel 3.10+.

| Distro family            | Package manager | Init system | Notes                            |
| ------------------------ | --------------- | ----------- | -------------------------------- |
| Arch / CachyOS / Manjaro | pacman          | systemd     | Fully tested                     |
| Debian / Ubuntu 18+      | apt             | systemd     | .deb available                   |
| RHEL / CentOS / Fedora   | dnf / yum       | systemd     | .rpm available                   |
| Alpine                   | apk             | OpenRC      | Binary works; no .apk yet        |
| Any SysV distro          | any             | SysV        | Universal installer handles init |

Cross-compiled architectures: **amd64 (x86_64)**, **386 (i386/i686)**, **arm64 (aarch64)**,
**armv7 (armhf)**. The 386 build targets `GO386=sse2`; `armv6` is not built.

---

## Universal installer

```bash
# Download and run (installs binary + detects init system automatically)
curl -fsSL https://github.com/sincerely-hello-world/webux/releases/latest/download/install.sh | sudo sh

# Or with a specific version (use the git tag, with its leading v)
sudo sh install.sh --version v1.0.0

# Skip service setup (binary only)
sudo sh install.sh --no-service
```

> **Installing from your own fork?** The script defaults to the upstream repo —
> `REPO="${WEBUX_REPO:-brendan4linux/webux}"`. Point it at other releases with
> `sudo WEBUX_REPO=<owner>/<repo> sh` (the variable must come *after* `sudo`, since `sudo` clears
> the environment).

The installer auto-detects:

- CPU architecture (`uname -m`) → `amd64` / `386` / `arm64` / `armv7`, matching the release asset
  suffixes
- OS and package manager (`/etc/os-release`)
- Init system (systemd → OpenRC → SysV)

---

## Package installation

```bash
# Debian / Ubuntu
sudo dpkg -i webux_1.0.0_amd64.deb          # 32-bit: webux_1.0.0_i386.deb
# Service is enabled and started automatically via the post-install hook

# RHEL / Fedora / CentOS
sudo rpm -i webux-1.0.0-1.x86_64.rpm        # 32-bit: webux-1.0.0-1.i386.rpm

# Arch / Manjaro / CachyOS
sudo pacman -U webux-1.0.0-1-x86_64.pkg.tar.zst   # 32-bit: ...-1-i686.pkg.tar.zst

# Universal tarball (the "installer tree" — carries etc/webux/config.yaml too)
tar xzf webux_1.0.0_linux_amd64.tar.gz
sudo sh usr/local/share/webux/install.sh

# Binary-only archive also exists: webux_1.0.0_linux_amd64_bin.tar.gz
```

Upgrades keep your config (dpkg conffiles / `%config(noreplace)` / pacman `backup=`), and removal
keeps `/var/lib/webux` — only `apt purge` clears it.

---

## PAM configuration

Only relevant to the PAM build (`mise run build-pam`). The template lives in the repo as
`scripts/webux.pam`:

```
# /etc/pam.d/webux
auth       required     pam_unix.so
auth       optional     pam_sss.so          # SSSD / LDAP
# auth    required     pam_google_authenticator.so  # TOTP 2FA

account    required     pam_unix.so
account    optional     pam_sss.so
```

Install it — **this step is required, not optional**:

```bash
sudo cp scripts/webux.pam /etc/pam.d/webux
```

No installer, package or archive ships this file, so it is always a manual copy. Skipping it makes
Webux fall back to `/etc/pam.d/login`, which can silently bypass 2FA configured in the webux stack —
see [Authentication](#authentication) for the details.

### Without PAM

The default build reads `/etc/shadow` directly using a pure-Go `crypt(3)` implementation, supporting
yescrypt (`$y$`), SHA-512 (`$6$`), SHA-256 (`$5$`), MD5 (`$1$`) and bcrypt (`$2b$`). No
`libxcrypt` and no CGO are needed, which is what makes the static cross-compiled binaries work
everywhere.

---

## Security notes

- Webux requires root (or equivalent) to access `/etc/shadow`, manage services, run LVM commands, and read `/proc`. Run it as root or with appropriate capabilities.
- Webux always runs HTTPS. A self-signed ECDSA certificate is auto-generated on first run (stored in `data_dir`, valid 2 years, includes all interface IPs as SANs). Set `tls_cert_file` and `tls_key_file` in config to use your own certificate.
- The JWT secret is stored in SQLite. Back up `/var/lib/webux/webux.db` to preserve sessions across reinstalls.
- The SSO bypass token grants full admin access — treat it like a root password.
- All API endpoints (except `/auth/login` and static assets) require a valid JWT. WebSocket connections are also gated.
- No data is sent to external services unless you configure an AI provider API key.
- If you build with `-tags pam`, always install `/etc/pam.d/webux`. Without it the PAM build retries
  against `/etc/pam.d/login`, which can bypass 2FA configured for webux — see
  [Authentication](#authentication).

---

## License

**GNU Affero General Public License v3.0** (AGPL-3.0). The full text is in [`LICENSE`](LICENSE).

```
Copyright (C) 2026 brendan4linux <brendan4linux@gmail.com>
```

Webux was created by **[brendan4linux](https://github.com/brendan4linux)** and is released under the
AGPL-3.0. This repository is a fork maintained at
[sincerely-hello-world/webux](https://github.com/sincerely-hello-world/webux); the original copyright
notice and license are retained unchanged, and all modifications stay under the same license.

Because AGPL-3.0 is a strong copyleft network license: **if you run a modified version of Webux as a
network service, you must offer the complete corresponding source of your version to its users.**

---

## 中文速览

> 这一节只是英文正文的摘要。细节请以英文部分与 **[`readme_new.md`](readme_new.md)**（完整开发指南）为准。

**Webux 是什么** —— 自托管的 Linux 服务器管理面板：单个静态二进制，内嵌 Web UI 与 SQLite，
没有运行时依赖、不需要 Docker、没有遥测。同时带 **Learn Mode**：每个操作都会显示等价的 shell
命令，可直接在面板内的终端里执行。

**安装（三条路）**

```bash
# A. 源码构建
mise install && mise run setup && mise run build
sudo WEBUX_DATA_DIR=/var/lib/webux ./build/webux      # https://localhost:8989

# B. 装系统包（服务会自动 enable + restart）
sudo dpkg -i  webux_<ver>_amd64.deb
sudo rpm  -i  webux-<ver>-1.x86_64.rpm
sudo pacman -U webux-<ver>-1-x86_64.pkg.tar.zst

# C. 在线安装脚本（识别架构与 init 系统）
sudo sh install.sh --version v0.1.1
```

**常用 mise 命令**

| 命令                  | 作用                                                         |
| --------------------- | ------------------------------------------------------------ |
| `mise run build`    | 构建前端 + 后端 →`build/webux`（`CGO_ENABLED=0`）        |
| `mise run dev`      | 后端热重载 + 前端 dev server（:5173 / :8989）               |
| `mise run test`     | `go test` + `svelte-check`                               |
| `mise run ci`       | 与 GitHub Actions 逐字等价的完整门禁                         |
| `mise run snapshot` | 本地产出**全部 21 个**发布产物 →`build/dist/`（无需 tag） |
| `mise run release`  | 用当前 git tag 正式发布（需要 tag 在 HEAD 且工作区干净）      |
| `mise run install`  | 安装到 `/usr/local/bin` + systemd 单元                      |
| `mise run build-pam` | PAM 鉴权变体（需 libpam 开发头，**不进 GoReleaser**）    |

**架构与产物**

四个架构 **amd64 / 386 / arm64 / armv7**（不含 armv6）× 5 种文件 + 1 个校验和 = **21 个产物**。
两套 tar.gz（安装树 / 纯二进制）+ deb + rpm + `.pkg.tar.zst`，全部由 GoReleaser + nfpm
（纯 Go）生成，不需要 `fpm` / `rpmbuild` / `bsdtar`。

> ⚠️ 32 位包的架构名不统一：**deb 与 rpm 都叫 `i386`，pacman 叫 `i686`**，而 tar.gz 后缀是
> `386`。搜产物时别只 grep 一个词。

**鉴权（最容易误解的一点）**

- 默认（也就是**所有发布产物**）是 **shadow** 鉴权：纯 Go 的 `crypt(3)`，支持 yescrypt / SHA-512 /
  SHA-256 / MD5 / bcrypt，**不需要 CGO，也不需要 libxcrypt**。
- **PAM 变体完全不在发布产物里** —— 没有 PAM 包、没有校验和条目、只有 amd64 裸二进制，且要自己
  `gh release upload`。想用就 `mise run build-pam`。
- 用 PAM 时**必须** `sudo cp scripts/webux.pam /etc/pam.d/webux`：否则会回退到 `/etc/pam.d/login`，
  可能**静默绕过 2FA**（仓库里没有任何安装路径会帮你复制这个文件，它是个孤儿模板）。

**许可**

**AGPL-3.0**，全文见 [`LICENSE`](LICENSE)。原作者 **brendan4linux**，本仓库
[sincerely-hello-world/webux](https://github.com/sincerely-hello-world/webux) 是它的 fork，
版权声明与许可证原样保留。把修改版作为网络服务对外提供时，**必须向使用者提供对应源码**。
