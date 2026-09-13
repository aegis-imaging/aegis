# PyInstaller spec for the AEGIS DIMSE bridge.
#
# IMPORTANT: --onedir mode (NOT --onefile).
#
# Why onedir:
#   1. The bridge runs as a long-lived Windows Service / launchd daemon /
#      systemd unit. With --onefile, every restart extracts the bundle into
#      a temp dir, which is slow (~hundreds of MB unpacked) and routinely
#      false-positives AV engines that watch temp-dir executions.
#   2. Patching/updating individual deps via differential installer becomes
#      possible — only changed files in _internal/ ship in the next .msi/.pkg.
#   3. Code-signing is per-file but PyInstaller's onefile bootstrap stub
#      is signed once; nested binaries inside the squashfs are not, which
#      trips macOS Gatekeeper on hardened-runtime distribution.
#
# Hidden imports cover transitive deps that the analyzer misses for cloud
# SDKs (boto3, azure-*, google-cloud-storage) and pydicom's pixel codecs.
#
# Build:  pyinstaller src/pyinstaller/bridge.spec
# Output: dist/aegis-bridge/   (executable + _internal/ tree)

# -*- mode: python ; coding: utf-8 -*-

from pathlib import Path

from PyInstaller.utils.hooks import (
    collect_submodules,
    collect_data_files,
    collect_dynamic_libs,
)

block_cipher = None

SPEC_DIR = Path(SPECPATH).resolve()
REPO_ROOT = SPEC_DIR.parents[2]               # dimse-bridge-installer/..
INSTALLER_ROOT = SPEC_DIR.parents[1]          # dimse-bridge-installer/
RECEIVER_PATH = REPO_ROOT / "dimse-receiver"

HIDDEN_IMPORTS = [
    # AWS / Azure / GCP cloud SDKs — PyInstaller misses some submodules.
    "boto3",
    "botocore",
    *collect_submodules("boto3"),
    *collect_submodules("botocore"),
    "azure.storage.blob",
    "azure.identity",
    *collect_submodules("azure.storage.blob"),
    *collect_submodules("azure.identity"),
    "google.cloud.storage",
    *collect_submodules("google.cloud.storage"),
    # DICOM stack.
    "pydicom",
    "pynetdicom",
    "pylibjpeg",
    "pylibjpeg_openjpeg",
    "pylibjpeg_libjpeg",
    # Keyring backends — platform-dependent, all bundled so the same
    # frozen binary can fall back if the preferred backend is missing.
    "keyring.backends.macOS",
    "keyring.backends.Windows",
    "keyring.backends.SecretService",
    "keyring.backends.kwallet",
    # Receiver app package — explicit re-export so analyzer picks it up.
    "app",
    "app.main",
    "app.config",
    "app.storage_backend",
    "app.ingest",
    "app.scp",
    "app.sender",
]

DATAS = []
DATAS += collect_data_files("pydicom")
DATAS += collect_data_files("pynetdicom")
DATAS += collect_data_files("certifi")  # requests/azure rely on CA bundle

# pydicom C extensions: pixel-codec shared libs that ctypes-loads at runtime.
BINARIES = []
BINARIES += collect_dynamic_libs("pylibjpeg_openjpeg")
BINARIES += collect_dynamic_libs("pylibjpeg_libjpeg")

a = Analysis(
    [str(INSTALLER_ROOT / "src" / "bridge" / "service_main.py")],
    pathex=[
        str(INSTALLER_ROOT),
        str(RECEIVER_PATH),  # so `import app.main` resolves at freeze time
    ],
    binaries=BINARIES,
    datas=DATAS,
    hiddenimports=HIDDEN_IMPORTS,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=["tkinter", "matplotlib", "PyQt5", "PyQt6", "PySide2", "PySide6"],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    [],
    exclude_binaries=True,
    name="aegis-bridge",
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=False,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)

coll = COLLECT(
    exe,
    a.binaries,
    a.zipfiles,
    a.datas,
    strip=False,
    upx=False,
    upx_exclude=[],
    name="aegis-bridge",
)
