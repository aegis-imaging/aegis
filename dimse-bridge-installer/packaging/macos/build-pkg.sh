#!/bin/bash
# AEGIS DIMSE Bridge — macOS .pkg builder.
#
# This script assumes PyInstaller has already produced ./dist/aegis-bridge/.
# It codesigns the daemon binary, packages it via pkgbuild + productbuild,
# signs the .pkg, then submits to notarytool and staples the ticket.
#
# CRITICAL: the daemon binary ITSELF must be codesigned with the hardened
# runtime + entitlements.plist BEFORE pkgbuild wraps it. Signing only the
# .pkg is insufficient — Gatekeeper checks the inner binary at launch and
# rejects an unsigned launchd target even if the .pkg is notarized.
#
# Required signing identity:
#   - "Developer ID Application: AEGIS Imaging (TEAMID)"   for the binary
#   - "Developer ID Installer: AEGIS Imaging (TEAMID)"     for the .pkg
#
# Required environment (set by CI from secrets — script no-ops cleanly when
# any of these are absent so local devs can produce unsigned packages):
#   APPLE_DEVELOPER_ID_APPLICATION   — application signing identity name
#   APPLE_DEVELOPER_ID_INSTALLER     — installer signing identity name
#   APPLE_NOTARY_KEYCHAIN_PROFILE    — `xcrun notarytool store-credentials` profile
#   PKG_VERSION                      — e.g. 1.0.0

set -euo pipefail

VERSION="${PKG_VERSION:-0.0.0}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALLER_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DIST_DIR="${INSTALLER_ROOT}/dist/aegis-bridge"
BUILD_DIR="${INSTALLER_ROOT}/build/macos"
STAGE_DIR="${BUILD_DIR}/stage"
PAYLOAD_DIR="${STAGE_DIR}/usr/local/aegis-bridge"
OUTPUT_PKG="${INSTALLER_ROOT}/dist/aegis-dimse-bridge-${VERSION}.pkg"
UNSIGNED_PKG="${BUILD_DIR}/aegis-dimse-bridge-${VERSION}-unsigned.pkg"

echo "==> Staging payload"
rm -rf "${STAGE_DIR}"
mkdir -p "${PAYLOAD_DIR}"
cp -R "${DIST_DIR}/"* "${PAYLOAD_DIR}/"
cp "${SCRIPT_DIR}/com.aegis.dimse-bridge.plist" "${PAYLOAD_DIR}/"

SCRIPTS_DIR="${BUILD_DIR}/scripts"
rm -rf "${SCRIPTS_DIR}"
mkdir -p "${SCRIPTS_DIR}"
cp "${SCRIPT_DIR}/postinstall" "${SCRIPTS_DIR}/postinstall"
chmod 0755 "${SCRIPTS_DIR}/postinstall"

if [ -n "${APPLE_DEVELOPER_ID_APPLICATION:-}" ]; then
    echo "==> Codesigning binaries with hardened runtime"
    find "${PAYLOAD_DIR}" -type f \( -name "*.dylib" -o -name "*.so" \) -print0 |
        while IFS= read -r -d '' f; do
            codesign --force --sign "${APPLE_DEVELOPER_ID_APPLICATION}" \
                --options runtime --timestamp "${f}"
        done
    codesign --force --sign "${APPLE_DEVELOPER_ID_APPLICATION}" \
        --options runtime \
        --entitlements "${SCRIPT_DIR}/entitlements.plist" \
        --timestamp \
        "${PAYLOAD_DIR}/aegis-bridge"
else
    echo "==> Skipping codesign (APPLE_DEVELOPER_ID_APPLICATION not set)"
fi

echo "==> Building component pkg"
pkgbuild \
    --root "${STAGE_DIR}" \
    --identifier "com.aegis.dimse-bridge" \
    --version "${VERSION}" \
    --scripts "${SCRIPTS_DIR}" \
    --install-location "/" \
    "${BUILD_DIR}/component.pkg"

echo "==> Building product archive"
DISTRIBUTION_XML="${BUILD_DIR}/distribution.xml"
cat > "${DISTRIBUTION_XML}" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>AEGIS DIMSE Bridge</title>
    <organization>ai.aegisimaging</organization>
    <pkg-ref id="com.aegis.dimse-bridge"/>
    <options customize="never" require-scripts="true"/>
    <choices-outline>
        <line choice="default">
            <line choice="com.aegis.dimse-bridge"/>
        </line>
    </choices-outline>
    <choice id="default"/>
    <choice id="com.aegis.dimse-bridge" visible="false">
        <pkg-ref id="com.aegis.dimse-bridge"/>
    </choice>
    <pkg-ref id="com.aegis.dimse-bridge" version="${VERSION}" onConclusion="none">component.pkg</pkg-ref>
</installer-gui-script>
EOF

productbuild \
    --distribution "${DISTRIBUTION_XML}" \
    --package-path "${BUILD_DIR}" \
    "${UNSIGNED_PKG}"

if [ -n "${APPLE_DEVELOPER_ID_INSTALLER:-}" ]; then
    echo "==> Signing installer"
    productsign --sign "${APPLE_DEVELOPER_ID_INSTALLER}" \
        "${UNSIGNED_PKG}" "${OUTPUT_PKG}"
else
    echo "==> Skipping productsign (APPLE_DEVELOPER_ID_INSTALLER not set)"
    cp "${UNSIGNED_PKG}" "${OUTPUT_PKG}"
fi

if [ -n "${APPLE_NOTARY_KEYCHAIN_PROFILE:-}" ] && \
   [ -n "${APPLE_DEVELOPER_ID_INSTALLER:-}" ]; then
    echo "==> Submitting to notarytool"
    xcrun notarytool submit "${OUTPUT_PKG}" \
        --keychain-profile "${APPLE_NOTARY_KEYCHAIN_PROFILE}" \
        --wait
    echo "==> Stapling notarization ticket"
    xcrun stapler staple "${OUTPUT_PKG}"
else
    echo "==> Skipping notarization (APPLE_NOTARY_KEYCHAIN_PROFILE not set)"
fi

echo "==> Done: ${OUTPUT_PKG}"
