#!/bin/sh
# Post-remove hook — pacman (.pkg.tar.zst) ONLY.
#
# ═══════════════════════════════════════════════════════════════════════════
# WHY THIS IS A SEPARATE FILE — do not merge it back into pkg-postremove.sh.
#
# nfpm's archlinux backend does NOT install these scripts as executables. It
# writes each one into the package's .INSTALL file as a same-named shell
# function, with the file itself at mode 0644:
#
#     function post_install() { <scripts/pkg-postinstall.sh verbatim> }
#     function post_remove()  { <this file verbatim> }
#
# pacman sources .INSTALL and then calls the function by name, so this body
# runs with pacman's own positional parameters — NOT the deb/rpm convention
# (remove|purge|upgrade|0|1).
#
# pkg-postremove.sh decides removal-vs-upgrade by inspecting $1 and treats
# every unrecognised value as "upgrade, leave the service alone". Under pacman
# that guard silently matched nothing, so every uninstall skipped stop/disable
# and left an enabled unit behind pointing at a deleted binary.
#
# This file therefore states the intent outright instead of inferring it.
# It is wired up via nfpms.overrides.archlinux.scripts.postremove.
# ═══════════════════════════════════════════════════════════════════════════
#
# /var/lib/webux is deliberately preserved: it holds webux.db and the JWT
# signing key, so wiping it would destroy every existing session.
#
# NOTE: not exercised on a real Arch box — no Arch environment was available
# when this was written. See readme_new.md §10.4 for the static checks that
# were run and what still needs a live `pacman -U` / `pacman -R` test.

# Best-effort "is this really the end of an installation?" check.
#
# pacman runs post_remove() *after* the package's files have been removed, so
# an intact binary means we are on the upgrade path (where the new package's
# post_install/post_upgrade will re-enable and restart the service — tearing it
# down here would only add a pointless outage).
if [ -x /usr/local/bin/webux ]; then
  exit 0
fi

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
