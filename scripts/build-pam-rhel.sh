#!/bin/sh
# build-pam-rhel.sh
# Builds a PAM-enabled webux binary inside a RHEL UBI9 container,
# producing a binary that runs on RHEL 9, CentOS Stream 9, Fedora, Rocky, Alma.
#
# Usage:
#   ./scripts/build-pam-rhel.sh           # uses git tag for version
#   ./scripts/build-pam-rhel.sh 0.9.6     # explicit version
#
# Requires: podman or docker
set -e
cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --exact-match 2>/dev/null || git describe --tags 2>/dev/null | sed 's/^v//')}"
[ -z "$VERSION" ] && { echo "Error: no version — pass as argument or set a git tag"; exit 1; }
VERSION="$(echo "$VERSION" | sed 's/^v//')"

if command -v podman >/dev/null 2>&1; then
    RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
    RUNTIME=docker
else
    echo "Error: neither podman nor docker found"; exit 1
fi

echo "→ Building webux-pam v${VERSION} for RHEL/UBI9 using $RUNTIME"
echo "→ This will take a few minutes on first run (downloading UBI9 image)"

mkdir -p build/release

# Use Red Hat UBI9 — freely redistributable, matches RHEL 9 / CentOS Stream 9 / Rocky / Alma
$RUNTIME run --rm \
    -v "$(pwd):/build" \
    -w /build \
    registry.access.redhat.com/ubi9/ubi:latest \
    bash -c "
set -e

echo '→ Installing build dependencies...'
dnf install -y -q \
    golang git curl \
    pam-devel libxcrypt-devel \
    gcc make ca-certificates \
    2>/dev/null

# Install Node.js 24 LTS via NodeSource (Node 22 reaches EOL on 2026-07-28)
curl -fsSL https://rpm.nodesource.com/setup_24.x | bash - >/dev/null 2>&1
dnf install -y -q nodejs 2>/dev/null

# Install nub — the frontend is installed with nub, not npm
# (web/ carries nub.lock and no package-lock.json, so 'npm ci' cannot work)
npm install -g @nubjs/nub >/dev/null 2>&1

# Build the frontend and sync it into cmd/webux/dist (//go:embed dist needs it)
echo '→ Building frontend...'
cd web && nub ci --silent && nub run build && cd ..
rm -rf cmd/webux/dist && cp -r web/dist cmd/webux/dist

echo '→ Building webux-pam...'
go mod tidy
CGO_ENABLED=1 go build -tags 'mysql postgres pam' \
    -ldflags '-s -w -X main.version=${VERSION} -X main.commit=\$(git rev-parse --short HEAD 2>/dev/null || echo unknown) -X main.date=\$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
    -trimpath \
    -o build/release/webux-pam-linux-amd64-rhel \
    ./cmd/webux

echo '✓ Built: build/release/webux-pam-linux-amd64-rhel'
ldd build/release/webux-pam-linux-amd64-rhel
"

echo ""
echo "✓ Done — build/release/webux-pam-linux-amd64-rhel"
echo ""
echo "Next steps:"
echo "  GoReleaser (mise run snapshot / mise run release) generates the standard packages"
echo "  with CGO_ENABLED=0 and does NOT build this PAM binary, so ship it yourself:"
echo "    gh release upload v${VERSION} build/release/webux-pam-linux-amd64-rhel --clobber"
echo "  The -rhel suffix marks the glibc/RHEL baseline (the ubuntu script builds the"
echo "  plain name); it needs libpam on the target system."
