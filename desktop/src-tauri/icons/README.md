# Icons

Placeholder. Replace with real branded icons before release.

Required formats (Tauri reads these from `tauri.conf.json`):

- `32x32.png` — small dock / tray
- `128x128.png` — standard app icon
- `128x128@2x.png` — retina
- `icon.icns` — macOS bundle
- `icon.ico` — Windows installer

Generate from a single source SVG with:

```bash
npm install -g @tauri-apps/cli
tauri icon path/to/source.png
```

That command will populate this directory with all the required sizes +
the `.icns` / `.ico` archives.
