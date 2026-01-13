//! Cloud DevBox Desktop Application
//!
//! Tauri 2.0-based desktop client for managing cloud development environments.
//! Features:
//! - Window management with minimize to tray
//! - System tray with quick actions
//! - Secure credential storage
//! - Environment management
//! - SSH integration

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod config;
mod auth;
mod environment;
mod ssh;
mod tray;
mod state;
mod updater;
mod sync;
mod shortcuts;

use std::sync::Arc;
use tauri::{Manager, WindowEvent};
use tokio::sync::RwLock;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::state::AppState;
use crate::tray::setup_system_tray;
use crate::auth::try_auto_login;
use crate::updater::schedule_update_checks;
use crate::sync::start_environment_sync;
use crate::shortcuts::register_shortcuts;

fn main() {
    // Initialize logging
    tracing_subscriber::registry()
        .with(tracing_subscriber::fmt::layer())
        .with(tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("devbox_desktop=debug".parse().unwrap()))
        .init();

    tracing::info!("Starting Cloud DevBox Desktop Application");

    let app_state = Arc::new(RwLock::new(AppState::new()));

    tauri::Builder::default()
        // Initialize plugins
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec!["--minimized"]),
        ))
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_store::Builder::new().build())
        .plugin(tauri_plugin_os::init())
        // Manage application state
        .manage(app_state)
        // Setup system tray
        .setup(|app| {
            setup_system_tray(app)?;
            
            // Handle command line arguments
            let args: Vec<String> = std::env::args().collect();
            if args.contains(&"--minimized".to_string()) {
                if let Some(window) = app.get_webview_window("main") {
                    window.hide().ok();
                }
            }
            
            // Try auto-login with stored credentials
            let app_handle = app.handle().clone();
            let state = app.state::<Arc<RwLock<AppState>>>().inner().clone();
            tauri::async_runtime::spawn(async move {
                if try_auto_login(&state).await {
                    tracing::info!("Auto-login successful");
                } else {
                    tracing::info!("No stored credentials or auto-login failed");
                }
            });
            
            // Schedule periodic update checks
            let app_handle_updates = app.handle().clone();
            let state_updates = app.state::<Arc<RwLock<AppState>>>().inner().clone();
            schedule_update_checks(app_handle_updates, state_updates);
            
            // Start background environment sync
            let app_handle_sync = app.handle().clone();
            let state_sync = app.state::<Arc<RwLock<AppState>>>().inner().clone();
            start_environment_sync(app_handle_sync, state_sync);
            
            // Register global shortcuts
            if let Err(e) = register_shortcuts(app) {
                tracing::warn!("Failed to register shortcuts: {}", e);
            }
            
            tracing::info!("Application setup complete");
            Ok(())
        })
        // Register command handlers
        .invoke_handler(tauri::generate_handler![
            // Environment commands
            commands::get_environments,
            commands::create_environment,
            commands::start_environment,
            commands::stop_environment,
            commands::delete_environment,
            commands::get_environment_details,
            // Auth commands
            auth::login,
            auth::logout,
            auth::get_current_user,
            auth::is_authenticated,
            auth::refresh_token,
            // SSH commands
            ssh::get_ssh_config,
            ssh::connect_ssh,
            ssh::open_in_vscode,
            ssh::open_in_jetbrains,
            ssh::get_available_ides,
            ssh::copy_ssh_command,
            // Config commands
            config::get_config,
            config::set_config,
            config::get_api_endpoint,
            // Updater commands
            updater::check_for_updates,
            updater::install_update,
            updater::get_app_version,
            // Shortcut and autostart commands
            shortcuts::is_autostart_enabled,
            shortcuts::enable_autostart,
            shortcuts::disable_autostart,
            shortcuts::get_shortcuts,
            shortcuts::set_shortcuts,
        ])
        // Handle window events
        .on_window_event(|window, event| {
            match event {
                WindowEvent::CloseRequested { api, .. } => {
                    // Minimize to tray instead of closing
                    let app_handle = window.app_handle();
                    if let Ok(state) = app_handle.try_state::<Arc<RwLock<AppState>>>() {
                        let state = futures::executor::block_on(state.read());
                        if state.config.minimize_to_tray {
                            window.hide().ok();
                            api.prevent_close();
                        }
                    }
                }
                _ => {}
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
