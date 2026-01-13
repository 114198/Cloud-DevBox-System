//! System tray functionality

use std::sync::Arc;
use tauri::{
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    menu::{Menu, MenuItem, PredefinedMenuItem, Submenu},
    App, AppHandle, Manager,
};
use tokio::sync::RwLock;

use crate::state::AppState;

/// Setup the system tray icon and menu
pub fn setup_system_tray(app: &App) -> Result<(), Box<dyn std::error::Error>> {
    let app_handle = app.handle().clone();
    
    // Create tray menu
    let menu = create_tray_menu(&app_handle)?;
    
    // Build tray icon
    let _tray = TrayIconBuilder::new()
        .icon(app.default_window_icon().unwrap().clone())
        .menu(&menu)
        .menu_on_left_click(false)
        .on_tray_icon_event(move |tray, event| {
            match event {
                TrayIconEvent::Click {
                    button: MouseButton::Left,
                    button_state: MouseButtonState::Up,
                    ..
                } => {
                    // Show/hide main window on left click
                    let app = tray.app_handle();
                    if let Some(window) = app.get_webview_window("main") {
                        if window.is_visible().unwrap_or(false) {
                            window.hide().ok();
                        } else {
                            window.show().ok();
                            window.set_focus().ok();
                        }
                    }
                }
                _ => {}
            }
        })
        .on_menu_event(move |app, event| {
            handle_menu_event(app, event.id.as_ref());
        })
        .build(app)?;
    
    tracing::info!("System tray initialized");
    Ok(())
}

/// Create the tray menu
fn create_tray_menu(app: &AppHandle) -> Result<Menu<tauri::Wry>, Box<dyn std::error::Error>> {
    let show = MenuItem::with_id(app, "show", "显示主窗口", true, None::<&str>)?;
    let hide = MenuItem::with_id(app, "hide", "隐藏主窗口", true, None::<&str>)?;
    let separator1 = PredefinedMenuItem::separator(app)?;
    
    // Environment submenu
    let env_new = MenuItem::with_id(app, "env_new", "新建环境", true, None::<&str>)?;
    let env_list = MenuItem::with_id(app, "env_list", "环境列表", true, None::<&str>)?;
    let env_submenu = Submenu::with_items(app, "环境管理", true, &[&env_new, &env_list])?;
    
    let separator2 = PredefinedMenuItem::separator(app)?;
    
    // Settings and about
    let settings = MenuItem::with_id(app, "settings", "设置", true, None::<&str>)?;
    let check_update = MenuItem::with_id(app, "check_update", "检查更新", true, None::<&str>)?;
    let about = MenuItem::with_id(app, "about", "关于", true, None::<&str>)?;
    
    let separator3 = PredefinedMenuItem::separator(app)?;
    let quit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
    
    let menu = Menu::with_items(
        app,
        &[
            &show,
            &hide,
            &separator1,
            &env_submenu,
            &separator2,
            &settings,
            &check_update,
            &about,
            &separator3,
            &quit,
        ],
    )?;
    
    Ok(menu)
}

/// Handle tray menu events
fn handle_menu_event(app: &AppHandle, event_id: &str) {
    match event_id {
        "show" => {
            if let Some(window) = app.get_webview_window("main") {
                window.show().ok();
                window.set_focus().ok();
            }
        }
        "hide" => {
            if let Some(window) = app.get_webview_window("main") {
                window.hide().ok();
            }
        }
        "env_new" => {
            if let Some(window) = app.get_webview_window("main") {
                window.show().ok();
                window.set_focus().ok();
                // Navigate to new environment page
                window.eval("window.location.href = '/environments/new'").ok();
            }
        }
        "env_list" => {
            if let Some(window) = app.get_webview_window("main") {
                window.show().ok();
                window.set_focus().ok();
                // Navigate to environments list
                window.eval("window.location.href = '/environments'").ok();
            }
        }
        "settings" => {
            if let Some(window) = app.get_webview_window("main") {
                window.show().ok();
                window.set_focus().ok();
                // Navigate to settings
                window.eval("window.location.href = '/settings'").ok();
            }
        }
        "check_update" => {
            // Trigger update check
            let app_clone = app.clone();
            tauri::async_runtime::spawn(async move {
                if let Err(e) = check_for_updates_internal(&app_clone).await {
                    tracing::error!("Failed to check for updates: {}", e);
                }
            });
        }
        "about" => {
            // Show about dialog
            if let Some(window) = app.get_webview_window("main") {
                let version = env!("CARGO_PKG_VERSION");
                window.eval(&format!(
                    "alert('Cloud DevBox Desktop\\n版本: {}\\n\\n云端开发环境管理客户端')",
                    version
                )).ok();
            }
        }
        "quit" => {
            app.exit(0);
        }
        _ => {}
    }
}

/// Internal function to check for updates
async fn check_for_updates_internal(app: &AppHandle) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    tracing::info!("Checking for updates...");
    // Update check logic will be implemented in updater module
    Ok(())
}

/// Update tray menu based on authentication state
pub async fn update_tray_menu(app: &AppHandle, state: &Arc<RwLock<AppState>>) {
    let state = state.read().await;
    let is_authenticated = state.is_authenticated();
    
    // Update menu items based on auth state
    // This would require recreating the menu or using dynamic menu items
    tracing::debug!("Tray menu updated, authenticated: {}", is_authenticated);
}
