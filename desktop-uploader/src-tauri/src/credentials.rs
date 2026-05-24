//! OS-keychain credential storage.
//!
//! We store a single JSON blob `{server_url, api_key}` under one keyring
//! entry rather than two separate entries so re-pair / clear operations are
//! atomic (no half-rotated state).
//!
//! Service name: `aegis-uploader`
//! Account name: `default`
//!
//! Backends:
//! - macOS:   Keychain Services
//! - Windows: Credential Manager
//! - Linux:   Secret Service (libsecret) — requires `dbus` + a running keyring
//!            daemon. Falls back with a clear error otherwise.

use keyring::Entry;
use serde::{Deserialize, Serialize};

const SERVICE: &str = "aegis-uploader";
const ACCOUNT: &str = "default";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StoredCredential {
    pub server_url: String,
    pub api_key: String,
}

fn entry() -> Result<Entry, String> {
    Entry::new(SERVICE, ACCOUNT).map_err(|e| format!("keyring init failed: {e}"))
}

pub fn store(server_url: &str, api_key: &str) -> Result<(), String> {
    let cred = StoredCredential {
        server_url: server_url.to_string(),
        api_key: api_key.to_string(),
    };
    let blob = serde_json::to_string(&cred)
        .map_err(|e| format!("credential serialize failed: {e}"))?;
    entry()?
        .set_password(&blob)
        .map_err(|e| format!("keyring write failed: {e}"))
}

pub fn read() -> Option<StoredCredential> {
    let e = entry().ok()?;
    let raw = e.get_password().ok()?;
    serde_json::from_str::<StoredCredential>(&raw).ok()
}

pub fn clear() -> Result<(), String> {
    match entry()?.delete_credential() {
        Ok(()) => Ok(()),
        // Missing-entry on clear is a no-op success — the user wanted it gone.
        Err(keyring::Error::NoEntry) => Ok(()),
        Err(e) => Err(format!("keyring delete failed: {e}")),
    }
}
