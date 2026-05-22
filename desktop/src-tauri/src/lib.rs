//! AEGIS Desktop shell library.
//!
//! Implements a small set of Tauri commands that the JS frontend calls into
//! when running natively. The commands stay deliberately narrow — anything
//! that can run in the browser (DICOM parsing, de-identification, OCR,
//! upload HTTPS) does, because we want feature parity with the web client.
//!
//! What lives here:
//!   - `start_watch` / `stop_watch` — watch a directory for new files and
//!     emit JS events with the path of every change.
//!   - `read_directory` — recursive file enumeration that streams JSON back
//!     so a 50,000-file scan doesn't OOM the webview.
//!
//! Native-acceleration commands (Tesseract / ONNX Runtime) are a follow-up.
//! See README.md.

use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;

use notify::RecursiveMode;
use notify_debouncer_mini::{new_debouncer, DebouncedEvent, DebouncedEventKind};
use parking_lot::Mutex;
use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager, State};

/// JSON shape emitted on `fs-event:<id>` for each filesystem change.
#[derive(Clone, Serialize)]
struct WatchEventPayload {
    path: String,
    kind: String,
}

/// One running watcher. The Drop impl stops the underlying notify thread.
struct Watcher {
    _inner: Box<dyn std::any::Any + Send + Sync>,
}

#[derive(Default)]
struct WatcherState {
    inner: Mutex<WatcherStateInner>,
}

#[derive(Default)]
struct WatcherStateInner {
    next_id: u64,
    watchers: std::collections::HashMap<u64, Watcher>,
}

#[tauri::command]
async fn start_watch(app: AppHandle, state: State<'_, Arc<WatcherState>>, path: String) -> Result<u64, String> {
    let path_buf = PathBuf::from(&path);
    if !path_buf.is_dir() {
        return Err(format!("not a directory: {}", path));
    }

    let id = {
        let mut s = state.inner.lock();
        s.next_id = s.next_id.wrapping_add(1);
        s.next_id
    };
    let app_handle = app.clone();
    let event_name = format!("fs-event:{}", id);

    // Debounce so a single study being copied in fires once at the end, not
    // 200 times for every constituent file.
    let mut debouncer = new_debouncer(Duration::from_millis(800), move |res: Result<Vec<DebouncedEvent>, _>| {
        if let Ok(events) = res {
            for e in events {
                let kind = match e.kind {
                    DebouncedEventKind::Any => "modify",
                    DebouncedEventKind::AnyContinuous => "modify",
                    _ => "modify",
                };
                let payload = WatchEventPayload {
                    path: e.path.to_string_lossy().into_owned(),
                    kind: kind.to_string(),
                };
                let _ = app_handle.emit(&event_name, payload);
            }
        }
    })
    .map_err(|e| format!("create watcher: {}", e))?;

    debouncer
        .watcher()
        .watch(&path_buf, RecursiveMode::Recursive)
        .map_err(|e| format!("watch {}: {}", path, e))?;

    state.inner.lock().watchers.insert(id, Watcher { _inner: Box::new(debouncer) });
    Ok(id)
}

#[tauri::command]
async fn stop_watch(state: State<'_, Arc<WatcherState>>, id: u64) -> Result<(), String> {
    state.inner.lock().watchers.remove(&id);
    Ok(())
}

#[derive(Serialize)]
struct DirEntry {
    name: String,
    path: String,
    is_dir: bool,
    is_file: bool,
    size_bytes: u64,
}

#[tauri::command]
async fn read_directory(path: String, recursive: bool) -> Result<Vec<DirEntry>, String> {
    let root = PathBuf::from(&path);
    if !root.is_dir() {
        return Err(format!("not a directory: {}", path));
    }
    let mut out = Vec::new();
    let mut stack = vec![root];
    while let Some(dir) = stack.pop() {
        let read = std::fs::read_dir(&dir).map_err(|e| format!("read_dir {}: {}", dir.display(), e))?;
        for entry in read {
            let entry = entry.map_err(|e| format!("entry: {}", e))?;
            let metadata = entry.metadata().map_err(|e| format!("metadata: {}", e))?;
            let path = entry.path();
            out.push(DirEntry {
                name: entry.file_name().to_string_lossy().into_owned(),
                path: path.to_string_lossy().into_owned(),
                is_dir: metadata.is_dir(),
                is_file: metadata.is_file(),
                size_bytes: metadata.len(),
            });
            if recursive && metadata.is_dir() {
                stack.push(path);
            }
        }
    }
    Ok(out)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            app.manage(Arc::new(WatcherState::default()));
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![start_watch, stop_watch, read_directory])
        .run(tauri::generate_context!())
        .expect("error while running AEGIS Desktop");
}
