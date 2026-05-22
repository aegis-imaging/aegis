// AEGIS Desktop entry point. Most logic lives in `lib.rs` so the same Rust
// code compiles into both the desktop binary and a mobile static lib (if we
// ever wrap this for iOS / Android later).

// On Windows release builds, suppress the spawning of a console window.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    aegis_desktop_lib::run()
}
