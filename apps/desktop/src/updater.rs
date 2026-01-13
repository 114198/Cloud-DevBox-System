//! Auto-update functionality

use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use tauri::Manager;

use crate::state::AppState;

/// Update information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateInfo {
    pub version: String,
    pub current_version: String,
    pub release_notes: String,
    pub download_url: String,
    pub published_at: String,
    pub is_mandatory: bool,
}

/// Update progress information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateProgress {
    pub downloaded: u64,
    pub total: Option<u64>,
    pub percentage: f32,
    pub status: UpdateStatus,
}

/// Update status
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum UpdateStatus {
    Checking,
    Downloading,
    Installing,
    Ready,
    Error(String),
}

/// Check for available updates
#[tauri::command]
pub async fn check_for_updates(
    app: tauri::AppHandle,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<Option<UpdateInfo>, String> {
    use tauri_plugin_updater::UpdaterExt;
    
    let state_guard = state.read().await;
    if !state_guard.config.auto_check_updates {
        return Ok(None);
    }
    drop(state_guard);
    
    let current_version = env!("CARGO_PKG_VERSION");
    
    tracing::info!("Checking for updates, current version: {}", current_version);
    
    // Emit checking status
    emit_update_progress(&app, UpdateProgress {
        downloaded: 0,
        total: None,
        percentage: 0.0,
        status: UpdateStatus::Checking,
    });
    
    match app.updater() {
        Ok(updater) => {
            match updater.check().await {
                Ok(Some(update)) => {
                    let update_info = UpdateInfo {
                        version: update.version.clone(),
                        current_version: current_version.to_string(),
                        release_notes: update.body.clone().unwrap_or_default(),
                        download_url: String::new(), // Handled by Tauri updater
                        published_at: update.date.map(|d| d.to_string()).unwrap_or_default(),
                        is_mandatory: false,
                    };
                    
                    tracing::info!("Update available: {} -> {}", current_version, update.version);
                    
                    // Show notification
                    show_update_notification(&app, &update_info);
                    
                    Ok(Some(update_info))
                }
                Ok(None) => {
                    tracing::info!("No updates available");
                    Ok(None)
                }
                Err(e) => {
                    tracing::error!("Failed to check for updates: {}", e);
                    emit_update_progress(&app, UpdateProgress {
                        downloaded: 0,
                        total: None,
                        percentage: 0.0,
                        status: UpdateStatus::Error(e.to_string()),
                    });
                    Err(format!("Failed to check for updates: {}", e))
                }
            }
        }
        Err(e) => {
            tracing::error!("Updater not available: {}", e);
            Err(format!("Updater not available: {}", e))
        }
    }
}

/// Install available update
#[tauri::command]
pub async fn install_update(
    app: tauri::AppHandle,
) -> Result<(), String> {
    use tauri_plugin_updater::UpdaterExt;
    
    tracing::info!("Installing update...");
    
    match app.updater() {
        Ok(updater) => {
            match updater.check().await {
                Ok(Some(update)) => {
                    // Download and install the update
                    let mut downloaded: u64 = 0;
                    let app_clone = app.clone();
                    
                    emit_update_progress(&app, UpdateProgress {
                        downloaded: 0,
                        total: None,
                        percentage: 0.0,
                        status: UpdateStatus::Downloading,
                    });
                    
                    update.download_and_install(
                        move |chunk_length, content_length| {
                            downloaded += chunk_length as u64;
                            let percentage = content_length
                                .map(|total| (downloaded as f32 / total as f32) * 100.0)
                                .unwrap_or(0.0);
                            
                            emit_update_progress(&app_clone, UpdateProgress {
                                downloaded,
                                total: content_length.map(|l| l as u64),
                                percentage,
                                status: UpdateStatus::Downloading,
                            });
                            
                            tracing::debug!(
                                "Downloaded {} of {:?} bytes ({:.1}%)",
                                downloaded,
                                content_length,
                                percentage
                            );
                        },
                        || {
                            tracing::info!("Download complete, preparing to install...");
                        },
                    ).await.map_err(|e| format!("Failed to install update: {}", e))?;
                    
                    tracing::info!("Update installed, restart required");
                    
                    emit_update_progress(&app, UpdateProgress {
                        downloaded: 0,
                        total: None,
                        percentage: 100.0,
                        status: UpdateStatus::Ready,
                    });
                    
                    // Notify user to restart
                    show_restart_notification(&app);
                    
                    Ok(())
                }
                Ok(None) => {
                    Err("No update available".to_string())
                }
                Err(e) => {
                    emit_update_progress(&app, UpdateProgress {
                        downloaded: 0,
                        total: None,
                        percentage: 0.0,
                        status: UpdateStatus::Error(e.to_string()),
                    });
                    Err(format!("Failed to check for updates: {}", e))
                }
            }
        }
        Err(e) => {
            Err(format!("Updater not available: {}", e))
        }
    }
}

/// Get current application version
#[tauri::command]
pub async fn get_app_version() -> Result<String, String> {
    Ok(env!("CARGO_PKG_VERSION").to_string())
}

/// Emit update progress event to frontend
fn emit_update_progress(app: &tauri::AppHandle, progress: UpdateProgress) {
    if let Some(window) = app.get_webview_window("main") {
        window.emit("update-progress", progress).ok();
    }
}

/// Show notification about available update
fn show_update_notification(app: &tauri::AppHandle, update_info: &UpdateInfo) {
    use tauri_plugin_notification::NotificationExt;
    
    if let Err(e) = app.notification()
        .builder()
        .title("有新版本可用")
        .body(&format!(
            "Cloud DevBox {} 已发布，点击查看更新内容",
            update_info.version
        ))
        .show()
    {
        tracing::warn!("Failed to show update notification: {}", e);
    }
}

/// Show notification to restart after update
fn show_restart_notification(app: &tauri::AppHandle) {
    use tauri_plugin_notification::NotificationExt;
    
    if let Err(e) = app.notification()
        .builder()
        .title("更新已安装")
        .body("请重启应用以完成更新")
        .show()
    {
        tracing::warn!("Failed to show restart notification: {}", e);
    }
}

/// Schedule periodic update checks
pub fn schedule_update_checks(app: tauri::AppHandle, state: Arc<RwLock<AppState>>) {
    tauri::async_runtime::spawn(async move {
        // Initial delay before first check (5 minutes after startup)
        tokio::time::sleep(std::time::Duration::from_secs(5 * 60)).await;
        
        // Check for updates every 6 hours
        let interval = std::time::Duration::from_secs(6 * 60 * 60);
        
        loop {
            let state_guard = state.read().await;
            if !state_guard.config.auto_check_updates {
                drop(state_guard);
                tokio::time::sleep(interval).await;
                continue;
            }
            drop(state_guard);
            
            tracing::info!("Scheduled update check...");
            
            use tauri_plugin_updater::UpdaterExt;
            
            if let Ok(updater) = app.updater() {
                match updater.check().await {
                    Ok(Some(update)) => {
                        let update_info = UpdateInfo {
                            version: update.version.clone(),
                            current_version: env!("CARGO_PKG_VERSION").to_string(),
                            release_notes: update.body.clone().unwrap_or_default(),
                            download_url: String::new(),
                            published_at: update.date.map(|d| d.to_string()).unwrap_or_default(),
                            is_mandatory: false,
                        };
                        show_update_notification(&app, &update_info);
                        
                        // Emit event to frontend
                        if let Some(window) = app.get_webview_window("main") {
                            window.emit("update-available", &update_info).ok();
                        }
                    }
                    Ok(None) => {
                        tracing::debug!("No updates available");
                    }
                    Err(e) => {
                        tracing::error!("Scheduled update check failed: {}", e);
                    }
                }
            }
            
            tokio::time::sleep(interval).await;
        }
    });
}
