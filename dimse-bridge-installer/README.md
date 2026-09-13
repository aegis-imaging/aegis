# AEGIS DIMSE Bridge — Native Installer

Native service packaging for the AEGIS DIMSE bridge (the Python
`dimse-receiver` from the main repo). Lets a hospital workstation run the
DICOM C-STORE receiver as a first-class service without Docker.

## What the bridge does

- Listens on TCP 11112 (DICOM IANA-registered port) for C-STORE
  associations from local PACS, modalities, and CT/MR consoles.
- Buffers, encrypts, and forwards each study to the AEGIS control plane
  for de-identification, QC, defacing, and downstream routing.
- Pairs with the AEGIS tenant via a one-time pairing token embedded in the
  installer filename.

The bridge wraps the existing `dimse-receiver/app/` package (unchanged)
into a PyInstaller `--onedir` bundle, then ships that bundle inside a
platform-native installer (`.msi` / `.pkg` / `.deb`).

## Install

### Linux (Debian / Ubuntu)

```sh
sudo dpkg -i aegis-dimse-bridge_1.0.0_amd64.deb
sudo systemctl status aegis-dimse-bridge
```

The postinst creates the `_aegis` system user, writes `/etc/aegis/bridge.yml`,
and enables the service. Logs at `/var/log/aegis/bridge.out.log`.

### macOS

```sh
sudo installer -pkg AEGIS-DIMSE-Bridge-1.0.0.pkg -target /
```

Daemon binary lands at `/usr/local/aegis-bridge/aegis-bridge`. Config at
`/Library/Application Support/AEGIS/bridge.yml`. Logs at
`/Library/Logs/AEGIS/`. Managed by launchd
(`com.aegis.dimse-bridge`).

### Windows

```powershell
msiexec /i aegis-dimse-bridge-1.0.0.msi /qn
```

Installs to `C:\Program Files\AEGIS\dimse-bridge`. Registers the
`AEGISDimseBridge` Windows Service. Config at
`%ProgramData%\AEGIS\bridge.yml`. Opens inbound firewall rule for TCP 11112.

> Windows packaging is Phase 3B; not built by the current CI workflow.

## Pairing flow

The installer filename embeds a one-time token:

```
aegis-dimse-bridge-1.0.0-pair-abc123xyz.msi
```

Between `-pair-` and the extension. On first launch the bridge:

1. Inspects `sys.executable` for the `-pair-<TOKEN>` pattern.
2. POSTs `{ "pairing_token": "abc123xyz" }` to
   `https://api.aegisimaging.ai/api/install/pair`.
3. Stores the returned `api_key` in the OS keyring (Credential Manager /
   Keychain / Secret Service) under service `aegis-dimse-bridge`,
   account `api-key`.
4. Writes the returned `server_url` to `bridge.yml`.

If the filename was renamed before install (or the user is re-pairing),
run pairing manually:

```sh
sudo aegis-bridge pair \
    --server https://api.aegisimaging.ai \
    --token abc123xyz
sudo systemctl restart aegis-dimse-bridge   # or launchctl kickstart, sc.exe
```

The pairing endpoint is single-use server-side; tokens cannot be replayed.

## Per-platform paths

| Path           | Linux                       | macOS                                      | Windows                                  |
| -------------- | --------------------------- | ------------------------------------------ | ---------------------------------------- |
| Binary         | `/opt/aegis-bridge/`        | `/usr/local/aegis-bridge/`                 | `C:\Program Files\AEGIS\dimse-bridge\`   |
| Config         | `/etc/aegis/bridge.yml`     | `/Library/Application Support/AEGIS/bridge.yml` | `%ProgramData%\AEGIS\bridge.yml`    |
| Logs           | `/var/log/aegis/`           | `/Library/Logs/AEGIS/`                     | `%ProgramData%\AEGIS\logs\`              |
| Service mgr    | `systemctl`                 | `launchctl`                                | `sc.exe` / Services.msc                  |
| Service name   | `aegis-dimse-bridge`        | `com.aegis.dimse-bridge`                   | `AEGISDimseBridge`                       |

## Develop

```sh
git clone https://github.com/aegis-imaging/aegis.git
cd aegis/dimse-bridge-installer
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# Dev-mode run (uses ../dimse-receiver/ via sys.path):
python -m src.bridge.service_main run

# Test pairing flow without an installer:
python -m src.bridge.service_main pair \
    --server http://localhost:8080 \
    --token testtoken123
```

## Build a distributable

```sh
# 1. Build the PyInstaller bundle (writes ./dist/aegis-bridge/)
pyinstaller src/pyinstaller/bridge.spec

# 2. Wrap it for the target platform
bash packaging/linux/build-deb.sh        # Linux
bash packaging/macos/build-pkg.sh        # macOS
# Windows: run candle/light against packaging/windows/installer.wxs
```

The CI workflow at `.github/workflows/build-dimse-bridge.yml` automates
all of the above for tagged releases (`dimse-bridge-v*`).

## Scope of this directory

This installer wraps `dimse-receiver/` — it does not fork or modify it.
The receiver's source remains the single source of truth; the installer
only adds packaging, a service entrypoint, the pairing client, and the
platform-native service manifests.
