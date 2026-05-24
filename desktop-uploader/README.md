# AEGIS Uploader (desktop)

Tauri 2.x desktop wrapper around the AEGIS DICOM anonymization and upload pipeline. Reads files from the operator's local filesystem (via the Rust shell), runs PS3.15 Annex E de-identification client-side, and uploads de-identified bytes to an AEGIS server.

Phase 2A scope. See "Phase scoping" below.

## Prerequisites

- **Rust 1.77+** (`rustup install stable`)
- **pnpm 9+** (`npm i -g pnpm`)
- **Tauri CLI**: installed automatically into `node_modules` by `pnpm install`; invoke via `pnpm tauri ...`
- **macOS 11+ / Windows 10+ / Linux with WebKit2GTK 4.1**

On Linux you'll need a few system packages — see the Tauri prerequisites docs for the canonical list (the short version: `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `librsvg2-dev`, `libssl-dev`).

## Quick start

```bash
cd desktop-uploader
pnpm install
pnpm tauri dev
```

`pnpm tauri dev` will:

1. Build the `@aegis/client` sibling package (it's referenced via `file:../client`),
2. Boot Vite on `http://localhost:1420`,
3. `cargo build` the Rust shell and open the native window pointing at Vite.

The first launch will land on the manual `ServerSettings` screen because the dev binary has no `-pair-*` token in its filename. Paste a server URL (e.g. `http://localhost:8080` if you have the Go API running locally) and an API key minted via the admin dashboard's API Keys tab.

To exercise the auto-pairing flow during development:

```bash
# After `pnpm tauri build`:
cd src-tauri/target/release/bundle/macos
cp -R "AEGIS Uploader.app" "AEGIS Uploader-pair-devtest123.app"
open "AEGIS Uploader-pair-devtest123.app"
```

The app will detect `devtest123`, POST it to `/api/install/pair`, and persist the returned credential to the OS keychain.

## Production build

```bash
pnpm tauri build
```

Artifacts land in `src-tauri/target/release/bundle/`:

- macOS: `bundle/dmg/*.dmg` and `bundle/macos/*.app`
- Windows: `bundle/msi/*.msi` and `bundle/nsis/*.exe`
- Linux: `bundle/deb/*.deb` and `bundle/appimage/*.AppImage`

(Phase 2A CI only builds the macOS targets — see `.github/workflows/build-uploader.yml`.)

## Pairing token flow

The personalised installer the recipient downloads has a filename of the form:

```
aegis-uploader-1.0.0-pair-{token}.dmg
```

`{token}` is up to 16 URL-safe-base64 characters (server holds the full token on its side; the in-filename portion is sufficient because the server still validates uniqueness against its DB).

1. The Rust shell reads `std::env::current_exe()` on startup.
2. On macOS it also inspects the enclosing `.app` bundle name in case the bundler renamed the inner binary.
3. The `regex` crate extracts `{token}`.
4. The React layer calls `exchange_pairing(server_url, pairing_token)` which POSTs `{"pairing_token": "..."}` to `${server_url}/api/install/pair`.
5. The server returns `{"api_key": "aegis_…", "server_url": "https://..."}` (single-use; a second call returns HTTP 410).
6. `store_credential(...)` writes a JSON blob `{server_url, api_key}` to the OS keychain under service `aegis-uploader`, account `default`.

After that initial pairing, the in-filename token is consumed and irrelevant — the keychain entry persists across launches. Reset via the "Re-pair" button (calls `clear_credential` then surfaces the settings screen again).

## Signing — generate the updater key BEFORE the first release

The Tauri updater needs an ed25519 keypair. The placeholder in `src-tauri/tauri.conf.json` must be replaced before any signed release will be honoured by an installed copy.

```bash
pnpm tauri signer generate -w ~/.aegis-uploader-signing.key
```

This emits a public key (paste it into `tauri.conf.json` `plugins.updater.pubkey`) and a private key (write to disk, password-protected). Store:

- Private key file: 1Password / secrets vault
- `TAURI_SIGNING_PRIVATE_KEY`: GitHub Secret (the file's contents)
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`: GitHub Secret (the password you set)

Once both are in CI, the `tauri-action` step will produce a signed bundle and a `latest.json` manifest the updater consumes.

Apple notarisation is wired up via the matching `APPLE_*` secrets — see the workflow file for the exact secret names. Until those are populated CI will produce an unsigned `.dmg` that can be sideloaded for testing.

## Phase scoping

### Phase 2A (this commit) — macOS only

- macOS-only CI matrix (`macos-14` arm64 + `macos-13` x64)
- DMG + .app bundle targets in `tauri.conf.json` (the cross-platform targets are listed but not exercised in CI yet)
- Manual pairing fallback for ops without an invited download
- No signed releases yet (placeholder updater pubkey)
- No icons committed (the bundler will use a default until `pnpm tauri icon ...` is run against a 1024×1024 source)

### Phase 2B (next iteration)

- Uncomment the Windows + Linux matrix block in `build-uploader.yml`
- Generate and commit the updater pubkey
- Commit branded icon assets (`pnpm tauri icon assets/source-1024.png`)
- First signed release published to `gs://aegis-prod-installer-staging/uploader/` with a `latest.json` manifest the in-app updater can fetch
- Wire the GCS bucket into the personalised download endpoint so recipients of `desktop_installer_invites` receive a filename like `aegis-uploader-1.0.0-pair-{token}.dmg`

## Directory layout

```
desktop-uploader/
  src-tauri/                  Rust shell
    src/main.rs               Tauri commands + bootstrap
    src/pair.rs               Pairing token parser + /api/install/pair client
    src/credentials.rs        OS-keychain wrapper (keyring crate)
    capabilities/default.json IPC + plugin permissions
    Cargo.toml
    tauri.conf.json
  src/                        React + TypeScript app
    App.tsx                   Stage machine (loading / pairing / ready / etc.)
    components/               First-run pairing, settings, picker, preview, progress
    lib/aegis-api.ts          Authenticated AEGIS HTTP client
    lib/anon.ts               @aegis/client wrappers (parse + deidentify + serialize)
  package.json
  vite.config.ts
  tsconfig.json
  index.html
```
