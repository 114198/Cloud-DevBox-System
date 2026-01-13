//! Tauri build script
//! 
//! This script handles the build process for the Cloud DevBox desktop application.
//! It configures Tauri and sets up platform-specific build options.

use std::env;

fn main() {
    // Build Tauri application
    tauri_build::build();
    
    // Print build information
    println!("cargo:rerun-if-changed=tauri.conf.json");
    println!("cargo:rerun-if-changed=icons/");
    println!("cargo:rerun-if-changed=entitlements.plist");
    println!("cargo:rerun-if-changed=Info.plist");
    println!("cargo:rerun-if-changed=wix/");
    println!("cargo:rerun-if-changed=nsis/");
    println!("cargo:rerun-if-changed=linux/");
    
    // Set build metadata
    let version = env::var("CARGO_PKG_VERSION").unwrap_or_else(|_| "0.1.0".to_string());
    let target = env::var("TARGET").unwrap_or_else(|_| "unknown".to_string());
    let profile = env::var("PROFILE").unwrap_or_else(|_| "debug".to_string());
    
    println!("cargo:rustc-env=BUILD_VERSION={}", version);
    println!("cargo:rustc-env=BUILD_TARGET={}", target);
    println!("cargo:rustc-env=BUILD_PROFILE={}", profile);
    
    // Platform-specific configuration
    #[cfg(target_os = "windows")]
    {
        // Windows-specific build configuration
        println!("cargo:rerun-if-changed=wix/main.wxs");
        println!("cargo:rerun-if-changed=nsis/installer.nsi");
        
        // Set Windows application manifest
        if let Ok(manifest) = env::var("CARGO_MANIFEST_DIR") {
            let manifest_path = format!("{}/windows.manifest", manifest);
            if std::path::Path::new(&manifest_path).exists() {
                println!("cargo:rustc-link-arg-bins=/MANIFEST:EMBED");
                println!("cargo:rustc-link-arg-bins=/MANIFESTINPUT:{}", manifest_path);
            }
        }
    }
    
    #[cfg(target_os = "macos")]
    {
        // macOS-specific build configuration
        println!("cargo:rerun-if-changed=entitlements.plist");
        println!("cargo:rerun-if-changed=entitlements.child.plist");
        println!("cargo:rerun-if-changed=Info.plist");
    }
    
    #[cfg(target_os = "linux")]
    {
        // Linux-specific build configuration
        println!("cargo:rerun-if-changed=linux/cloud-devbox.desktop");
        println!("cargo:rerun-if-changed=linux/cloud-devbox.metainfo.xml");
    }
}
