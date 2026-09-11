# Running Webux Locally

> New here? Read **[`README.md`](README.md)** first (what Webux is, which artifacts exist, how to
> install it). This file is the short "get it running on my machine" walkthrough.
>
> For the deeper guide — project layout, ports, test gates, debugging recipes, deployment — see
> **[`readme_new.md`](readme_new.md)** chapters 6 (first-time setup) and 7 (daily development).
>
> **中文读者**：本文是"本机跑起来"的最短路径；完整开发指南见 **[`readme_new.md`](readme_new.md)**。

## Prerequisites

| Tool    | Version | Install |
|---------|---------|---------|
| mise    | ≥ 2026.8 | https://mise.jdx.dev — installs the pinned Go / air / nub / goreleaser toolchain from `mise.toml` |
| Go      | 1.26.5  | via mise (`mise install`; version pinned in `mise.toml`) |
| Node    | 24 LTS  | not installed by hand — `nub` downloads it per the repo-root `.node-version` |
| nub     | ≥ 0.9   | https://nubjs.com (or `mise install`, see `mise.toml`) — provisions Node itself |

> The frontend is installed with **nub**, not npm. `web/` carries `nub.lock`
> and no `package-lock.json`, so `npm ci` / `npm install` in `web/` is not the
> supported path.

---

## First-time setup

```bash
# 1. Install the pinned toolchain (go / air / nub / goreleaser)
mise install

# 2. Install project deps: go mod tidy + frontend (nub install)
mise run setup

# 3. Build frontend, copy dist, build binary — in one command:
mise run build

# 4. Run
mkdir -p /tmp/webux-data
WEBUX_DATA_DIR=/tmp/webux-data ./build/webux
```

Open **https://localhost:8989** — the server is HTTPS-only with a self-signed
certificate, so the browser shows a security warning on first visit (use
"Advanced → Proceed", or `curl -k`). There are no plain-HTTP or port-9090
endpoints.

Login uses real Linux accounts, which means the process must be able to read
`/etc/shadow` — i.e. run it with `sudo`, or add `--no-auth` for a
development-only session:

```bash
# Full functionality (port→process mapping, service management, real login)
sudo WEBUX_DATA_DIR=/tmp/webux-data ./build/webux

# No auth — the API and WebSocket skip auth entirely
./build/webux --no-auth
```

> `--no-auth` only bypasses the `/api/*` and `/ws` middleware. The frontend
> still calls `GET /auth/whoami`, which answers 401 without a token, so the
> login page still appears. See `readme_new.md` §10.1 for the two ways around
> that (log in once for a session cookie, or set `auth.bypass_token`).

---

## Manual build steps (if not using mise)

```bash
# 1. Resolve deps
go mod tidy

# 2. Build frontend
cd web && nub ci && nub run build && cd ..

# 3. Copy dist next to embed.go (MUST happen before go build)
rm -rf cmd/webux/dist
cp -r web/dist cmd/webux/dist

# 4. Build Go binary
go build -o ./build/webux ./cmd/webux && echo "SUCCESS"

# 5. Run
mkdir -p /tmp/webux-data
WEBUX_DATA_DIR=/tmp/webux-data ./build/webux
```

---

## Known gotchas

**`go.mod` must have `go-sqlite3 v0.21.3`** — not v0.15.1.
If tidy fails, check: `grep sqlite3 go.mod`

**`pattern all:web/dist: no matching files found`**
The `cp -r web/dist cmd/webux/dist` step was skipped or the frontend
wasn't built yet. Run `mise run web` first.

**`"embed" imported and not used`**
The embed declaration lives in `cmd/webux/embed.go`. The `"embed"`
import must NOT appear in `cmd/webux/main.go`.

**Nested `cmd/webux/dist/dist/`**
The `cp` was run twice. Fix with:
```bash
rm -rf cmd/webux/dist && cp -r web/dist cmd/webux/dist
```

**`migration failed ... file is not a database`**
The backend exits *before* it binds `:8989`, so the only symptom you see is
vite reporting `connect ECONNREFUSED 127.0.0.1:8989` on `/api/*` and `/ws`.
Cause: `$WEBUX_DATA_DIR/webux.db` already exists but is not a SQLite database
(in dev the data directory is `/tmp/webux-data`, and a stray file written
there survives across runs). Check it, then move it aside:

```bash
file /tmp/webux-data/webux.db                       # "ASCII text" => junk
rm -rf /tmp/webux-data                              # sessions + cert are regenerated
```

**Service management requires root**
The dbus system bus requires elevated privileges. Run with `sudo`.

**Processes show `—` for owner**
Reading `/proc/<pid>/fd` for other users' processes requires root.
