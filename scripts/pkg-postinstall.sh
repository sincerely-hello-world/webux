#!/bin/sh
# Post-install hook for webux packages.
#
# Runs on a fresh install and (for rpm/pacman) on upgrade as well, so it must
# stay idempotent. All output goes to the package manager's log.
#
# The panel is enabled for boot and started here, so a plain package install
# leaves a working service. Hand `--no-service`-style control to the
# administrator by not installing the unit in the first place; the packaged
# units are always managed.

# Create data directory
mkdir -p /var/lib/webux
chmod 750 /var/lib/webux

# /etc/webux can hold a bypass_token and the JWT signing key, so keep it
# root-only. This matches what scripts/install.sh does; without it dpkg would
# leave the directory at 755.
mkdir -p /etc/webux
chmod 750 /etc/webux

# /etc/webux/config.yaml needs no handling here. It is shipped as a conffile
# (deb), %config(noreplace) (rpm) and backup= (pacman), so the package manager
# has already placed it and will preserve the administrator's edits on upgrade.
# The previous revision copied /etc/webux/config.yaml.dpkg-new over the top,
# but dpkg resolves that staging file before the postinst runs, so the block
# never executed.

# Work out the port for the URL printed below. There is no bare `port:` key in
# the schema — listen_addr is "host:port" or ":port", so take the text after
# the last colon. (The previous revision deleted every colon, which turned
# "127.0.0.1:8989" into "127.0.0.18989".)
PORT=8989
if [ -f /etc/webux/config.yaml ]; then
    ADDR=$(sed -n 's/^[[:space:]]*listen_addr:[[:space:]]*//p' /etc/webux/config.yaml \
           | head -1 | tr -d '"'\'' ')
    [ -n "$ADDR" ] && PORT="${ADDR##*:}"
fi

# Get primary IP
IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "$IP" ] && IP="localhost"

# Enable and start the service.
#
# `systemctl is-system-running` is deliberately NOT the test here: it exits
# non-zero for a "degraded" but perfectly functional system, which skipped this
# entire block and left the panel silently unreachable after installation.
# /run/systemd/system exists only when systemd is the running init.
if [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload 2>/dev/null || true
    systemctl enable webux 2>/dev/null || true
    systemctl restart webux 2>/dev/null || true
    echo "Webux service enabled and started."
elif [ -f /etc/init.d/webux ]; then
    chmod +x /etc/init.d/webux
    if command -v rc-update >/dev/null 2>&1; then
        rc-update add webux default 2>/dev/null || true
    elif command -v update-rc.d >/dev/null 2>&1; then
        update-rc.d webux defaults 2>/dev/null || true
    elif command -v chkconfig >/dev/null 2>&1; then
        chkconfig --add webux 2>/dev/null || true
    fi
    /etc/init.d/webux start 2>/dev/null || true
    echo "Webux service enabled and started."
fi

echo ""
echo "  Webux installed — panel available at https://${IP}:${PORT}"
echo "  Config: /etc/webux/config.yaml"
echo "  Note: self-signed cert — accept the browser security warning on first visit"
echo ""
