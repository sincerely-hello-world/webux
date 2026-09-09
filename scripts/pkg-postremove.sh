#!/bin/sh
# Post-remove hook — runs after package removal.
# /var/lib/webux is intentionally preserved: it holds webux.db and the JWT
# signing key, so wiping it would destroy every existing session.
#
# Used by the deb and rpm packages. The archlinux (.pkg.tar.zst) package uses
# scripts/pkg-postremove-pacman.sh instead — see nfpms.overrides.archlinux.scripts
# in .goreleaser.yaml and the comment on the case block below.
#
# This script is %postun on RPM, which rpm runs on upgrades as well as on
# removal — with $1 set to the number of versions still installed (1 on an
# upgrade, 0 on the final removal).
#
# On Debian the same code sits inside a function that dpkg only invokes for a
# genuine "remove", so $1 is empty there; hence the ":-remove" default.
case "${1:-remove}" in
  # The function names are belt-and-braces for the pacman .INSTALL path: nfpm
  # writes this body into .INSTALL as "function post_remove() { ... }", so the
  # function name can surface as $1 (pacman passes none of remove|purge|0).
  # The archlinux package does not rely on this — it uses the dedicated script.
  remove|purge|disappear|0|post_remove) IS_REMOVAL=1 ;;
  upgrade|1|install|post_install|post_upgrade) IS_REMOVAL=0 ;;
  *) IS_REMOVAL=0 ;;
esac

if [ "$IS_REMOVAL" -eq 0 ]; then
  # Upgrade. The new package's %post has already restarted the service with the
  # new binary; stopping or disabling it here would undo exactly that. Worse,
  # `systemctl disable` would silently discard an administrator's
  # `systemctl enable webux` on every upgrade. Leave the service alone.
  exit 0
fi

# Debian teardown is additionally handled by the snippet the deb packager
# generates around this script; this path matters most for RPM.
if command -v systemctl >/dev/null 2>&1; then
  systemctl stop webux 2>/dev/null || true
  systemctl disable webux 2>/dev/null || true
  systemctl daemon-reload 2>/dev/null || true
elif [ -f /etc/init.d/webux ]; then
  /etc/init.d/webux stop 2>/dev/null || true
  if command -v rc-update >/dev/null 2>&1; then
    rc-update del webux default 2>/dev/null || true
  elif command -v update-rc.d >/dev/null 2>&1; then
    update-rc.d webux remove 2>/dev/null || true
  elif command -v chkconfig >/dev/null 2>&1; then
    chkconfig --del webux 2>/dev/null || true
  fi
fi

echo "Webux removed. Data preserved at /var/lib/webux"
