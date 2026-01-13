//! Global keyboard shortcuts

use std::sync::Arc;
use tokio::sync::RwLock;
use tauri::Manager;
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutState};

use crate::state::AppState;

/// Register global keyboard shortcuts
pub fn register_shortcuts(app: &tauri::App) -> Result<(), Box<dyn std::error::Error>> {
    let app_handle = app.handle().clone();
    let state = app.state::<Arc<RwLock<AppState>>>().inner().clone();
    
    // Get shortcut configuration
    let shortcuts_config = futures::executor::block_on(async {
        let state = state.read().await;
        state.config.shortcuts.clone()
    });
    
    // Register toggle window shortcut
    if let Ok(shortcut) = shortcuts_config.toggle_window.parse::<Shortcut>() {
        let app_clone = app_handle.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                toggle_main_window(&app_clone);
            }
        })?;
        tracing::info!("Registered toggle window shortcut: {}", shortcuts_config.toggle_window);
    }
    
    // Register quick create shortcut
    if let Ok(shortcut) = shortcuts_config.quick_create.parse::<Shortcut>() {
        let app_clone = app_handle.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                open_quick_create(&app_clone);
            }
        })?;
        tracing::info!("Registered quick create shortcut: {}", shortcuts_config.quick_create);
    }
    
    // Register open environments shortcut
    if let Ok(shortcut) = shortcuts_config.open_environments.parse::<Shortcut>() {
        let app_clone = app_handle.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                open_environments(&app_clone);
            }
        })?;
        tracing::info!("Registered open environments shortcut: {}", shortcuts_config.open_environments);
    }
    
    Ok(())
}

/// Toggle main window visibility
fn toggle_main_window(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        if window.is_visible().unwrap_or(false) {
            window.hide().ok();
        } else {
            window.show().ok();
            window.set_focus().ok();
        }
    }
}

/// Open quick create dialog
fn open_quick_create(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        window.show().ok();
        window.set_focus().ok();
        window.eval("window.location.href = '/environments/new'").ok();
    }
}

/// Open environments list
fn open_environments(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        window.show().ok();
        window.set_focus().ok();
        window.eval("window.location.href = '/environments'").ok();
    }
}

/// Update shortcuts with new configuration
pub async fn update_shortcuts(
    app: &tauri::AppHandle,
    state: &Arc<RwLock<AppState>>,
) -> Result<(), String> {
    // Unregister all existing shortcuts
    app.global_shortcut()
        .unregister_all()
        .map_err(|e| format!("Failed to unregister shortcuts: {}", e))?;
    
    let state_guard = state.read().await;
    let shortcuts_config = state_guard.config.shortcuts.clone();
    drop(state_guard);
    
    // Re-register shortcuts with new configuration
    let app_clone = app.clone();
    
    if let Ok(shortcut) = shortcuts_config.toggle_window.parse::<Shortcut>() {
        let app_inner = app_clone.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                toggle_main_window(&app_inner);
            }
        }).map_err(|e| format!("Failed to register toggle shortcut: {}", e))?;
    }
    
    if let Ok(shortcut) = shortcuts_config.quick_create.parse::<Shortcut>() {
        let app_inner = app_clone.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                open_quick_create(&app_inner);
            }
        }).map_err(|e| format!("Failed to register quick create shortcut: {}", e))?;
    }
    
    if let Ok(shortcut) = shortcuts_config.open_environments.parse::<Shortcut>() {
        let app_inner = app_clone.clone();
        app.global_shortcut().on_shortcut(shortcut, move |_app, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                open_environments(&app_inner);
            }
        }).map_err(|e| format!("Failed to register environments shortcut: {}", e))?;
    }
    
    tracing::info!("Shortcuts updated");
    Ok(())
}

// Tauri commands for autostart management

/// Check if autostart is enabled
#[tauri::command]
pub async fn is_autostart_enabled(
    app: tauri::AppHandle,
) -> Result<bool, String> {
    use tauri_plugin_autostart::ManagerExt;
    
    app.autolaunch()
        .is_enabled()
        .map_err(|e| format!("Failed to check autostart status: {}", e))
}

/// Enable autostart
#[tauri::command]
pub async fn enable_autostart(
    app: tauri::AppHandle,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    use tauri_plugin_autostart::ManagerExt;
    
    app.autolaunch()
        .enable()
        .map_err(|e| format!("Failed to enable autostart: {}", e))?;
    
    // Update config
    let mut state_guard = state.write().await;
    state_guard.config.auto_start = true;
    state_guard.config.save().map_err(|e| e.to_string())?;
    
    tracing::info!("Autostart enabled");
    Ok(())
}

/// Disable autostart
#[tauri::command]
pub async fn disable_autostart(
    app: tauri::AppHandle,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    use tauri_plugin_autostart::ManagerExt;
    
    app.autolaunch()
        .disable()
        .map_err(|e| format!("Failed to disable autostart: {}", e))?;
    
    // Update config
    let mut state_guard = state.write().await;
    state_guard.config.auto_start = false;
    state_guard.config.save().map_err(|e| e.to_string())?;
    
    tracing::info!("Autostart disabled");
    Ok(())
}

/// Get current shortcut configuration
#[tauri::command]
pub async fn get_shortcuts(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<crate::config::ShortcutConfig, String> {
    let state = state.read().await;
    Ok(state.config.shortcuts.clone())
}

/// Update shortcut configuration
#[tauri::command]
pub async fn set_shortcuts(
    shortcuts: crate::config::ShortcutConfig,
    app: tauri::AppHandle,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    // Update config
    {
        let mut state_guard = state.write().await;
        state_guard.config.shortcuts = shortcuts;
        state_guard.config.save().map_err(|e| e.to_string())?;
    }
    
    // Re-register shortcuts
    let state_arc = state.inner().clone();
    update_shortcuts(&app, &state_arc).await?;
    
    Ok(())
}
