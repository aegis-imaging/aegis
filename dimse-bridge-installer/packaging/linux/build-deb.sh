#!/bin/bash
# AEGIS DIMSE Bridge — Debian .deb builder.
#
# Assumes PyInstaller has already produced ./dist/aegis-bridge/.
# Stages the payload under /opt/aegis-bridge, installs the systemd unit
# and maintainer scripts, then runs dpkg-deb.
#
# Env:
#   PKG_VERSION   — version to embed in control file (default 1.0.0)
#   PKG_ARCH      — Debian architecture (default amd64)

set -euo pipefail

VERSION="${PKG_VERSION:-1.0.0}"
ARCH="${PKG_ARCH:-amd64}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALLER_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${INSTALLER_ROOT}/dist/aegis-bridge"
DEBIAN_SRC="${SCRIPT_DIR}/debian"
STAGE_DIR="${INSTALLER_ROOT}/build/linux/aegis-dimse-bridge_${VERSION}_${ARCH}"
OUTPUT_DEB="${INSTALLER_ROOT}/dist/aegis-dimse-bridge_${VERSION}_${ARCH}.deb"

if [ ! -d "${DIST_DIR}" ]; then
    echo "error: ${DIST_DIR} not found — run pyinstaller first" >&2
    exit 1
fi

echo "==> Staging payload at ${STAGE_DIR}"
rm -rf "${STAGE_DIR}"
mkdir -p \
    "${STAGE_DIR}/DEBIAN" \
    "${STAGE_DIR}/opt/aegis-bridge" \
    "${STAGE_DIR}/lib/systemd/system" \
    "${STAGE_DIR}/etc/aegis"

cp -R "${DIST_DIR}/"* "${STAGE_DIR}/opt/aegis-bridge/"
chmod 0755 "${STAGE_DIR}/opt/aegis-bridge/aegis-bridge" || true

# Symlink for $PATH access — useful for `sudo aegis-bridge pair ...`.
mkdir -p "${STAGE_DIR}/usr/local/bin"
ln -sf /opt/aegis-bridge/aegis-bridge "${STAGE_DIR}/usr/local/bin/aegis-bridge"

cp "${DEBIAN_SRC}/aegis-dimse-bridge.service" \
   "${STAGE_DIR}/lib/systemd/system/aegis-dimse-bridge.service"

# Control file — interpolate VERSION + ARCH.
sed -e "s/^Version:.*/Version: ${VERSION}/" \
    -e "s/^Architecture:.*/Architecture: ${ARCH}/" \
    "${DEBIAN_SRC}/control" > "${STAGE_DIR}/DEBIAN/control"

cp "${DEBIAN_SRC}/conffiles" "${STAGE_DIR}/DEBIAN/conffiles"
install -m 0755 "${DEBIAN_SRC}/postinst" "${STAGE_DIR}/DEBIAN/postinst"
install -m 0755 "${DEBIAN_SRC}/prerm"    "${STAGE_DIR}/DEBIAN/prerm"

echo "==> Building .deb"
mkdir -p "${INSTALLER_ROOT}/dist"
dpkg-deb --build --root-owner-group "${STAGE_DIR}" "${OUTPUT_DEB}"

echo "==> Done: ${OUTPUT_DEB}"
