// Prevents a console window on Windows in release builds.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod credentials;
mod pair;

use serde::Serialize;
use std::fs;
use std::path::Path;
use walkdir::WalkDir;

use crate::credentials::StoredCredential;
use crate::pair::PairResponse;

/// One enumerated file in a picked folder.
#[derive(Debug, Serialize)]
struct FileEntry {
    path: String,
    size_bytes: u64,
}

#[tauri::command]
fn get_pairing_token() -> Option<String> {
    pair::read_pairing_token()
}

#[tauri::command]
async fn exchange_pairing(
    server_url: String,
    pairing_token: String,
) -> Result<PairResponse, String> {
    pair::exchange(&server_url, &pairing_token).await
}

#[tauri::command]
fn store_credential(server_url: String, api_key: String) -> Result<(), String> {
    credentials::store(&server_url, &api_key)
}

#[tauri::command]
fn get_credential() -> Option<StoredCredential> {
    credentials::read()
}

#[tauri::command]
fn clear_credential() -> Result<(), String> {
    credentials::clear()
}

/// Detect whether a file is DICOM. Cheap path: extension match. Authoritative
/// path: DICM magic bytes at offset 128. We only open the file for the magic
/// check when the extension didn't already match — keeps enumeration fast on
/// folders mixing JPEGs and DICOM.
fn looks_like_dicom(path: &Path) -> bool {
    let ext_match = path
        .extension()
        .and_then(|e| e.to_str())
        .map(|e| {
            let lower = e.to_ascii_lowercase();
            lower == "dcm" || lower == "dicom"
        })
        .unwrap_or(false);
    if ext_match {
        return true;
    }

    // Magic bytes check. Skip files smaller than 132 bytes.
    let Ok(mut file) = fs::File::open(path) else {
        return false;
    };
    use std::io::{Read, Seek, SeekFrom};
    if file.seek(SeekFrom::Start(128)).is_err() {
        return false;
    }
    let mut magic = [0u8; 4];
    if file.read_exact(&mut magic).is_err() {
        return false;
    }
    &magic == b"DICM"
}

#[tauri::command]
fn enumerate_folder(path: String) -> Result<Vec<FileEntry>, String> {
    let root = Path::new(&path);
    if !root.exists() {
        return Err(format!("path does not exist: {path}"));
    }
    if !root.is_dir() {
        return Err(format!("not a directory: {path}"));
    }

    let mut out = Vec::new();
    for entry in WalkDir::new(root).follow_links(false) {
        let entry = match entry {
            Ok(e) => e,
            Err(e) => {
                // Permission or transient I/O errors during enumeration:
                // skip the entry rather than failing the whole walk.
                eprintln!("walkdir error: {e}");
                continue;
            }
        };
        if !entry.file_type().is_file() {
            continue;
        }
        let p = entry.path();
        if !looks_like_dicom(p) {
            continue;
        }
        let size_bytes = entry.metadata().map(|m| m.len()).unwrap_or(0);
        out.push(FileEntry {
            path: p.to_string_lossy().to_string(),
            size_bytes,
        });
    }
    Ok(out)
}

#[tauri::command]
fn read_dicom_bytes(path: String) -> Result<Vec<u8>, String> {
    fs::read(&path).map_err(|e| format!("read {path}: {e}"))
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            get_pairing_token,
            exchange_pairing,
            store_credential,
            get_credential,
            clear_credential,
            enumerate_folder,
            read_dicom_bytes,
        ])
        .run(tauri::generate_context!())
        .expect("error while running AEGIS Uploader");
}
