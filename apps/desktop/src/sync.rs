//! Background synchronization for environment status

use std::sync::Arc;
use tokio::sync::RwLock;
use tauri::Manager;

use crate::state::AppState;
use crate::environment::{Environment, EnvironmentClient, EnvironmentStatus};

/// Start background environment sync
pub fn start_environment_sync(app: tauri::AppHandle, state: Arc<RwLock<AppState>>) {
    tauri::async_runtime::spawn(async move {
        // Sync every 30 seconds
        let interval = std::time::Duration::from_secs(30);
        
        loop {
            tokio::time::sleep(interval).await;
            
            let state_guard = state.read().await;
            if !state_guard.is_authenticated() {
                continue;
            }
            
            let token = match &state_guard.auth.access_token {
                Some(t) => t.clone(),
                None => continue,
            };
            let api_endpoint = state_guard.config.api_endpoint.clone();
            let show_notifications = state_guard.config.show_notifications;
            let old_environments = state_guard.environments_cache.clone();
            drop(state_guard);
            
            // Fetch latest environments
            let client = EnvironmentClient::new(&api_endpoint)
                .with_token(&token);
            
            match client.list_environments().await {
                Ok(new_environments) => {
                    // Check for status changes
                    if show_notifications {
                        check_status_changes(&app, &old_environments, &new_environments);
                    }
                    
                    // Update cache
                    let mut state_guard = state.write().await;
                    state_guard.environments_cache = new_environments;
                    state_guard.last_sync = Some(chrono::Utc::now());
                    
                    // Emit event to frontend
                    if let Some(window) = app.get_webview_window("main") {
                        window.emit("environments-updated", ()).ok();
                    }
                }
                Err(e) => {
                    tracing::error!("Failed to sync environments: {}", e);
                }
            }
        }
    });
}

/// Check for environment status changes and notify user
fn check_status_changes(
    app: &tauri::AppHandle,
    old_envs: &[Environment],
    new_envs: &[Environment],
) {
    use tauri_plugin_notification::NotificationExt;
    
    for new_env in new_envs {
        if let Some(old_env) = old_envs.iter().find(|e| e.id == new_env.id) {
            if old_env.status != new_env.status {
                let (title, body) = match new_env.status {
                    EnvironmentStatus::Running => (
                        "环境已启动",
                        format!("环境 '{}' 已成功启动", new_env.name),
                    ),
                    EnvironmentStatus::Stopped => (
                        "环境已停止",
                        format!("环境 '{}' 已停止", new_env.name),
                    ),
                    EnvironmentStatus::Failed => (
                        "环境启动失败",
                        format!("环境 '{}' 启动失败，请检查日志", new_env.name),
                    ),
                    EnvironmentStatus::Suspended => (
                        "环境已挂起",
                        format!("环境 '{}' 因长时间未使用已被挂起", new_env.name),
                    ),
                    _ => continue,
                };
                
                if let Err(e) = app.notification()
                    .builder()
                    .title(title)
                    .body(&body)
                    .show()
                {
                    tracing::warn!("Failed to show notification: {}", e);
                }
            }
        }
    }
}

/// Force sync environments immediately
pub async fn force_sync(
    state: &Arc<RwLock<AppState>>,
) -> Result<Vec<Environment>, String> {
    let state_guard = state.read().await;
    
    if !state_guard.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state_guard.auth.access_token.as_ref()
        .ok_or("No access token")?
        .clone();
    let api_endpoint = state_guard.config.api_endpoint.clone();
    drop(state_guard);
    
    let client = EnvironmentClient::new(&api_endpoint)
        .with_token(&token);
    
    let environments = client.list_environments()
        .await
        .map_err(|e| format!("Failed to fetch environments: {}", e))?;
    
    // Update cache
    let mut state_guard = state.write().await;
    state_guard.environments_cache = environments.clone();
    state_guard.last_sync = Some(chrono::Utc::now());
    
    Ok(environments)
}
