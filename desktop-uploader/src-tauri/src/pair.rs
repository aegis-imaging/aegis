//! Pairing flow.
//!
//! The desktop installer's filename embeds a single-use pairing token, e.g.
//! `aegis-uploader-1.0.0-pair-abc123xyz.dmg`. On first launch we:
//!
//! 1. Read `std::env::current_exe()` to discover the installer/binary path,
//! 2. Regex-match `-pair-{token}` out of the file stem,
//! 3. POST the token to `{server_url}/api/install/pair`,
//! 4. Receive `{api_key, server_url}` and persist it via the `credentials` module.
//!
//! The token segment is at most 16 URL-safe-base64 characters in the filename.
//! The server validates uniqueness against the full token table on its side.
//!
//! Note: on macOS the binary inside an installed `.app` bundle is renamed by
//! the bundler, so we walk a few candidates — the current exe path itself,
//! and (on macOS) the parent `.app` bundle name — to find a `-pair-` segment.

use regex::Regex;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

/// Response from `POST /api/install/pair`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PairResponse {
    pub api_key: String,
    pub server_url: String,
}

/// Body for `POST /api/install/pair`.
#[derive(Debug, Serialize)]
struct PairRequest<'a> {
    pairing_token: &'a str,
}

/// Match `-pair-<token>` in the stem (before the extension). The captured
/// token is URL-safe base64: `[A-Za-z0-9_-]+`. We bound the length at 16 so
/// stray hyphens in a custom filename don't accidentally consume more than
/// the installer suffix.
fn token_regex() -> Regex {
    // SAFETY: literal pattern, compiles at runtime once.
    Regex::new(r"-pair-([A-Za-z0-9_-]{1,16})(?:\.[A-Za-z0-9]+)?$")
        .expect("pairing token regex must compile")
}

/// Extract a pairing token from a single path's filename, if present.
fn token_from_path(path: &Path) -> Option<String> {
    let name = path.file_name()?.to_string_lossy().to_string();
    let re = token_regex();
    re.captures(&name)
        .and_then(|c| c.get(1))
        .map(|m| m.as_str().to_string())
}

/// Walk a small set of candidate paths derived from the running executable
/// and return the first pairing token found. Candidates include:
/// - the executable path itself
/// - on macOS, the enclosing `.app` bundle name (so a renamed binary inside
///   `AEGIS Uploader-pair-xyz.app/Contents/MacOS/aegis-uploader` still works)
pub fn read_pairing_token() -> Option<String> {
    let exe = std::env::current_exe().ok()?;
    let mut candidates: Vec<PathBuf> = vec![exe.clone()];

    // Walk up to four parents looking for a `.app` bundle on macOS.
    let mut cursor = exe.as_path();
    for _ in 0..4 {
        if let Some(parent) = cursor.parent() {
            if let Some(name) = parent.file_name().and_then(|n| n.to_str()) {
                if name.ends_with(".app") {
                    candidates.push(parent.to_path_buf());
                    break;
                }
            }
            cursor = parent;
        } else {
            break;
        }
    }

    for candidate in candidates {
        if let Some(tok) = token_from_path(&candidate) {
            return Some(tok);
        }
    }
    None
}

/// Exchange the pairing token for an `{api_key, server_url}` pair.
pub async fn exchange(server_url: &str, pairing_token: &str) -> Result<PairResponse, String> {
    let trimmed = server_url.trim_end_matches('/');
    let url = format!("{}/api/install/pair", trimmed);

    let client = reqwest::Client::builder()
        .build()
        .map_err(|e| format!("http client init failed: {e}"))?;

    let res = client
        .post(&url)
        .json(&PairRequest { pairing_token })
        .send()
        .await
        .map_err(|e| format!("network error contacting {url}: {e}"))?;

    let status = res.status();
    if !status.is_success() {
        let body = res.text().await.unwrap_or_default();
        return Err(format!(
            "pairing failed ({}): {}",
            status.as_u16(),
            body.trim()
        ));
    }

    let pair: PairResponse = res
        .json()
        .await
        .map_err(|e| format!("invalid pairing response: {e}"))?;

    if pair.api_key.is_empty() {
        return Err("pairing response missing api_key".to_string());
    }

    // The server returns its own canonical server_url; fall back to the one
    // the user provided if the server omitted it.
    let resolved = if pair.server_url.is_empty() {
        PairResponse {
            api_key: pair.api_key,
            server_url: trimmed.to_string(),
        }
    } else {
        pair
    };

    Ok(resolved)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    #[test]
    fn extracts_token_from_dmg() {
        let p = PathBuf::from("/Users/x/Downloads/aegis-uploader-1.0.0-pair-abc123xyz.dmg");
        assert_eq!(token_from_path(&p).as_deref(), Some("abc123xyz"));
    }

    #[test]
    fn extracts_token_from_app_bundle() {
        let p = PathBuf::from("/Applications/AEGIS Uploader-pair-tok-12345.app");
        assert_eq!(token_from_path(&p).as_deref(), Some("tok-12345"));
    }

    #[test]
    fn no_token_when_absent() {
        let p = PathBuf::from("/Applications/AEGIS Uploader.app");
        assert_eq!(token_from_path(&p), None);
    }

    #[test]
    fn caps_token_length_at_16() {
        // 17 valid chars after -pair- — regex should not match because the
        // upper bound on the capture group is 16. This guards against a long
        // ambiguous filename being misread as a token.
        let p = PathBuf::from("/tmp/installer-pair-abcdefghijklmnopq.dmg");
        assert_eq!(token_from_path(&p), None);
    }
}
