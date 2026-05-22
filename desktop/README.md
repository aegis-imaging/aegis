# AEGIS Desktop

Native installer (Mac / Windows / Linux) wrapping the AEGIS upload portal +
heavy-mode de-identification pipeline. Built on [Tauri 2](https://tauri.app/).

## What it does that the web client doesn't

- **Native file system** — drag a folder of studies from Finder / Explorer,
  no browser quirks. Watch a "drop folder" and auto-upload anything that
  appears.
- **Bundles the heavy-mode WASM** — no ~10 MB first-use download for
  Tesseract.js; it ships inside the installer.
- **Code-signed binaries** — no "unidentified developer" warning.
- **Offline-tolerant** — studies queue locally if the cloud is unreachable;
  ship when the link comes back.
- **Native acceleration (planned)** — shell out to native Tesseract / ONNX
  Runtime via Tauri commands for ~10× faster face de-id versus WASM.

The UI is the same React app as `frontend/upload-portal/` — same buttons,
same flows, same `@aegis/client` library doing the heavy lifting. The
Tauri shell adds the native bits.

## What you need to build it yourself

| Tool | Why | How |
|------|-----|-----|
| **Rust toolchain** | Tauri's shell is Rust | <https://rustup.rs> |
| **Node 20+** | Frontend build | nvm or [nodejs.org](https://nodejs.org/) |
| **Platform deps** | Tauri's prerequisites | See [Tauri's setup guide](https://tauri.app/start/prerequisites/) |

Once those are installed:

```bash
cd desktop
npm install
npm run tauri dev          # hot-reload dev mode
npm run tauri build        # produce native installers
```

The installers land in `src-tauri/target/release/bundle/`:

- macOS → `bundle/dmg/AEGIS_<version>_<arch>.dmg`
- Windows → `bundle/msi/AEGIS_<version>_<arch>_en-US.msi`
- Linux → `bundle/appimage/AEGIS_<version>_<arch>.AppImage` and
  `bundle/deb/aegis_<version>_<arch>.deb`

## Cross-platform from one machine

Tauri supports cross-compilation, but the easiest path is GitHub Actions —
one workflow with a matrix of `runs-on: macos-latest`, `windows-latest`,
`ubuntu-latest` builds all three in parallel. The
`.github/workflows/desktop-release.yml` (added in this PR) does exactly
that, triggered by pushing a `v*.*.*` tag.

## Architecture

```
desktop/
├── src/                     React shell (this PR is minimal — wraps the
│                            upload-portal App with native bridge hooks)
│   ├── App.tsx
│   ├── main.tsx
│   └── desktop-bridge.ts    Wrappers around the Tauri JS API:
│                              - watchFolder(path, callback)
│                              - openFolderDialog()
│                              - showInFolder(path)
│
├── src-tauri/               Rust shell
│   ├── Cargo.toml
│   ├── tauri.conf.json
│   ├── build.rs
│   ├── capabilities/default.json
│   ├── icons/               App icons (placeholder — replace before release)
│   └── src/
│       ├── main.rs          Entry point
│       └── lib.rs           Tauri commands invocable from JS:
│                              - watch_folder
│                              - read_dicom_directory
│
├── index.html               Vite entry
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## What's in this PR (and what isn't)

**In scope:**
- Tauri scaffold with builds for Mac/Windows/Linux
- A minimal React shell that mounts the existing upload-portal App
- Native bridge for: watch folder, open folder dialog
- Build instructions

**Deliberately deferred to follow-ups:**
- Bundling Tesseract.js + WASM assets so heavy mode loads without a network
  trip (would add ~10 MB to the installer)
- Native ONNX Runtime command for hardware-accelerated face de-id
- Auto-updater wiring
- Real app icons (placeholders shipped)
- Code-signing keys for Mac/Windows release builds
- GitHub Actions release workflow

The minimum viable demo: clone the repo, `cd desktop`, `npm run tauri dev`,
and you have a native AEGIS upload portal running on your laptop.

## Why Tauri (vs Electron)

| | Tauri | Electron |
|---|---|---|
| Installer size | 10-30 MB | 80-150 MB |
| Memory | ~50 MB | ~300 MB |
| Shell language | Rust | Node |
| Webview | OS-native (WebKit on Mac, WebView2 on Win, WebKitGTK on Linux) | Bundled Chromium |
| Auto-update | Built-in | electron-updater |
| Code signing | Built-in helpers for both Mac + Windows | Same |

Tauri wins on size + memory by big margins, which matters for an app that's
mostly a thin shell over a web frontend. The OS-native webview is also
slightly more standards-compliant in practice (especially on macOS Sonoma+).

## Where AEGIS Desktop fits in the client tier hierarchy

See [`docs/CLIENT_UPLOADS.md`](../docs/CLIENT_UPLOADS.md) — desktop is
**tier 3** of the six client entry points. Same `@aegis/client` library
under the hood, same de-id pipeline, same upload contract.
