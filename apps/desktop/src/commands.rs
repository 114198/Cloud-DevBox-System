//! Tauri command handlers for environment management

use std::sync::Arc;
use tokio::sync::RwLock;
use tauri::Manager;

use crate::environment::{
    Environment, EnvironmentClient, CreateEnvironmentRequest, ResourceAllocation,
};
use crate::state::AppState;

/// Get all environments for the current user
#[tauri::command]
pub async fn get_environments(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<Vec<Environment>, String> {
    let state = state.read().await;
    
    if !state.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state.auth.access_token.as_ref()
        .ok_or("No access token")?;
    
    let client = EnvironmentClient::new(&state.config.api_endpoint)
        .with_token(token);
    
    client.list_environments()
        .await
        .map_err(|e| format!("Failed to fetch environments: {}", e))
}

/// Create a new environment
#[tauri::command]
pub async fn create_environment(
    name: String,
    template_id: String,
    description: Option<String>,
    resources: Option<ResourceAllocation>,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
    app: tauri::AppHandle,
) -> Result<Environment, String> {
    let state_guard = state.read().await;
    
    if !state_guard.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state_guard.auth.access_token.as_ref()
        .ok_or("No access token")?
        .clone();
    let api_endpoint = state_guard.config.api_endpoint.clone();
    let show_notifications = state_guard.config.show_notifications;
    drop(state_guard);
    
    let client = EnvironmentClient::new(&api_endpoint)
        .with_token(&token);
    
    let request = CreateEnvironmentRequest {
        name: name.clone(),
        description,
        template_id,
        resources,
    };
    
    let env = client.create_environment(request)
        .await
        .map_err(|e| format!("Failed to create environment: {}", e))?;
    
    // Show notification
    if show_notifications {
        send_notification(&app, "环境创建成功", &format!("环境 '{}' 已创建", name));
    }
    
    Ok(env)
}

/// Start an environment
#[tauri::command]
pub async fn start_environment(
    id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
    app: tauri::AppHandle,
) -> Result<Environment, String> {
    let state_guard = state.read().await;
    
    if !state_guard.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state_guard.auth.access_token.as_ref()
        .ok_or("No access token")?
        .clone();
    let api_endpoint = state_guard.config.api_endpoint.clone();
    let show_notifications = state_guard.config.show_notifications;
    drop(state_guard);
    
    let client = EnvironmentClient::new(&api_endpoint)
        .with_token(&token);
    
    let env = client.start_environment(&id)
        .await
        .map_err(|e| format!("Failed to start environment: {}", e))?;
    
    // Show notification
    if show_notifications {
        send_notification(&app, "环境已启动", &format!("环境 '{}' 已启动", env.name));
    }
    
    Ok(env)
}

/// Stop an environment
#[tauri::command]
pub async fn stop_environment(
    id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
    app: tauri::AppHandle,
) -> Result<Environment, String> {
    let state_guard = state.read().await;
    
    if !state_guard.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state_guard.auth.access_token.as_ref()
        .ok_or("No access token")?
        .clone();
    let api_endpoint = state_guard.config.api_endpoint.clone();
    let show_notifications = state_guard.config.show_notifications;
    drop(state_guard);
    
    let client = EnvironmentClient::new(&api_endpoint)
        .with_token(&token);
    
    let env = client.stop_environment(&id)
        .await
        .map_err(|e| format!("Failed to stop environment: {}", e))?;
    
    // Show notification
    if show_notifications {
        send_notification(&app, "环境已停止", &format!("环境 '{}' 已停止", env.name));
    }
    
    Ok(env)
}

/// Delete an environment
#[tauri::command]
pub async fn delete_environment(
    id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
    app: tauri::AppHandle,
) -> Result<(), String> {
    let state_guard = state.read().await;
    
    if !state_guard.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state_guard.auth.access_token.as_ref()
        .ok_or("No access token")?
        .clone();
    let api_endpoint = state_guard.config.api_endpoint.clone();
    let show_notifications = state_guard.config.show_notifications;
    drop(state_guard);
    
    let client = EnvironmentClient::new(&api_endpoint)
        .with_token(&token);
    
    client.delete_environment(&id)
        .await
        .map_err(|e| format!("Failed to delete environment: {}", e))?;
    
    // Show notification
    if show_notifications {
        send_notification(&app, "环境已删除", "环境已成功删除");
    }
    
    Ok(())
}

/// Get environment details
#[tauri::command]
pub async fn get_environment_details(
    id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<Environment, String> {
    let state = state.read().await;
    
    if !state.is_authenticated() {
        return Err("Not authenticated".to_string());
    }
    
    let token = state.auth.access_token.as_ref()
        .ok_or("No access token")?;
    
    let client = EnvironmentClient::new(&state.config.api_endpoint)
        .with_token(token);
    
    client.get_environment(&id)
        .await
        .map_err(|e| format!("Failed to fetch environment details: {}", e))
}

/// Send a system notification
fn send_notification(app: &tauri::AppHandle, title: &str, body: &str) {
    use tauri_plugin_notification::NotificationExt;
    
    if let Err(e) = app.notification()
        .builder()
        .title(title)
        .body(body)
        .show()
    {
        tracing::warn!("Failed to show notification: {}", e);
    }
}
