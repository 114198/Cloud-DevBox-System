//! Authentication module with secure credential storage

use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use keyring::Entry;

use crate::state::{AppState, AuthState, User};

const SERVICE_NAME: &str = "io.devbox.desktop";
const CREDENTIAL_KEY: &str = "auth_credentials";

/// Login credentials
#[derive(Debug, Serialize, Deserialize)]
pub struct LoginCredentials {
    pub email: String,
    pub password: String,
    pub remember_me: bool,
}

/// Login response from API
#[derive(Debug, Serialize, Deserialize)]
pub struct LoginResponse {
    pub user: User,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_in: i64,
}

/// Stored credentials for auto-login
#[derive(Debug, Serialize, Deserialize)]
struct StoredCredentials {
    refresh_token: String,
    user_id: String,
}

/// Credential manager for secure storage
pub struct CredentialManager;

impl CredentialManager {
    /// Store credentials securely
    pub fn store_credentials(refresh_token: &str, user_id: &str) -> Result<(), String> {
        let entry = Entry::new(SERVICE_NAME, CREDENTIAL_KEY)
            .map_err(|e| format!("Failed to create keyring entry: {}", e))?;
        
        let creds = StoredCredentials {
            refresh_token: refresh_token.to_string(),
            user_id: user_id.to_string(),
        };
        
        let json = serde_json::to_string(&creds)
            .map_err(|e| format!("Failed to serialize credentials: {}", e))?;
        
        entry.set_password(&json)
            .map_err(|e| format!("Failed to store credentials: {}", e))?;
        
        tracing::info!("Credentials stored securely");
        Ok(())
    }
    
    /// Retrieve stored credentials
    pub fn get_credentials() -> Result<Option<StoredCredentials>, String> {
        let entry = Entry::new(SERVICE_NAME, CREDENTIAL_KEY)
            .map_err(|e| format!("Failed to create keyring entry: {}", e))?;
        
        match entry.get_password() {
            Ok(json) => {
                let creds: StoredCredentials = serde_json::from_str(&json)
                    .map_err(|e| format!("Failed to deserialize credentials: {}", e))?;
                Ok(Some(creds))
            }
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(e) => Err(format!("Failed to retrieve credentials: {}", e)),
        }
    }
    
    /// Delete stored credentials
    pub fn delete_credentials() -> Result<(), String> {
        let entry = Entry::new(SERVICE_NAME, CREDENTIAL_KEY)
            .map_err(|e| format!("Failed to create keyring entry: {}", e))?;
        
        match entry.delete_credential() {
            Ok(_) => {
                tracing::info!("Credentials deleted");
                Ok(())
            }
            Err(keyring::Error::NoEntry) => Ok(()),
            Err(e) => Err(format!("Failed to delete credentials: {}", e)),
        }
    }
}

/// Login with email and password
#[tauri::command]
pub async fn login(
    credentials: LoginCredentials,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<User, String> {
    let state_guard = state.read().await;
    let api_endpoint = state_guard.config.api_endpoint.clone();
    drop(state_guard);
    
    let client = reqwest::Client::new();
    
    let response = client
        .post(format!("{}/api/v1/auth/login", api_endpoint))
        .json(&serde_json::json!({
            "email": credentials.email,
            "password": credentials.password,
        }))
        .send()
        .await
        .map_err(|e| format!("Login request failed: {}", e))?;
    
    if !response.status().is_success() {
        let error_text = response.text().await.unwrap_or_default();
        return Err(format!("Login failed: {}", error_text));
    }
    
    let login_response: LoginResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse login response: {}", e))?;
    
    // Store credentials if remember_me is enabled
    if credentials.remember_me {
        CredentialManager::store_credentials(
            &login_response.refresh_token,
            &login_response.user.id,
        )?;
    }
    
    // Update application state
    let mut state_guard = state.write().await;
    state_guard.auth = AuthState {
        is_authenticated: true,
        user: Some(login_response.user.clone()),
        access_token: Some(login_response.access_token),
        refresh_token: Some(login_response.refresh_token),
        expires_at: Some(chrono::Utc::now().timestamp() + login_response.expires_in),
    };
    
    tracing::info!("User logged in: {}", login_response.user.email);
    Ok(login_response.user)
}

/// Logout and clear credentials
#[tauri::command]
pub async fn logout(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    // Clear stored credentials
    CredentialManager::delete_credentials()?;
    
    // Clear application state
    let mut state_guard = state.write().await;
    state_guard.clear_auth();
    
    tracing::info!("User logged out");
    Ok(())
}

/// Get current authenticated user
#[tauri::command]
pub async fn get_current_user(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<Option<User>, String> {
    let state = state.read().await;
    Ok(state.auth.user.clone())
}

/// Check if user is authenticated
#[tauri::command]
pub async fn is_authenticated(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<bool, String> {
    let state = state.read().await;
    Ok(state.is_authenticated())
}

/// Refresh access token using stored refresh token
#[tauri::command]
pub async fn refresh_token(
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<bool, String> {
    // Try to get stored credentials first
    let stored_creds = CredentialManager::get_credentials()?;
    
    let refresh_token = {
        let state_guard = state.read().await;
        state_guard.auth.refresh_token.clone()
            .or_else(|| stored_creds.map(|c| c.refresh_token))
    };
    
    let refresh_token = match refresh_token {
        Some(token) => token,
        None => return Ok(false),
    };
    
    let state_guard = state.read().await;
    let api_endpoint = state_guard.config.api_endpoint.clone();
    drop(state_guard);
    
    let client = reqwest::Client::new();
    
    let response = client
        .post(format!("{}/api/v1/auth/refresh", api_endpoint))
        .json(&serde_json::json!({
            "refresh_token": refresh_token,
        }))
        .send()
        .await
        .map_err(|e| format!("Token refresh failed: {}", e))?;
    
    if !response.status().is_success() {
        // Clear invalid credentials
        CredentialManager::delete_credentials()?;
        let mut state_guard = state.write().await;
        state_guard.clear_auth();
        return Ok(false);
    }
    
    let login_response: LoginResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse refresh response: {}", e))?;
    
    // Update stored credentials
    CredentialManager::store_credentials(
        &login_response.refresh_token,
        &login_response.user.id,
    )?;
    
    // Update application state
    let mut state_guard = state.write().await;
    state_guard.auth = AuthState {
        is_authenticated: true,
        user: Some(login_response.user),
        access_token: Some(login_response.access_token),
        refresh_token: Some(login_response.refresh_token),
        expires_at: Some(chrono::Utc::now().timestamp() + login_response.expires_in),
    };
    
    tracing::info!("Token refreshed successfully");
    Ok(true)
}

/// Try auto-login with stored credentials
pub async fn try_auto_login(state: &Arc<RwLock<AppState>>) -> bool {
    match CredentialManager::get_credentials() {
        Ok(Some(_)) => {
            // We have stored credentials, try to refresh
            let state_clone = state.clone();
            match refresh_token_internal(&state_clone).await {
                Ok(true) => {
                    tracing::info!("Auto-login successful");
                    true
                }
                _ => {
                    tracing::info!("Auto-login failed, credentials may be expired");
                    false
                }
            }
        }
        _ => false,
    }
}

/// Internal token refresh function
async fn refresh_token_internal(state: &Arc<RwLock<AppState>>) -> Result<bool, String> {
    let stored_creds = CredentialManager::get_credentials()?;
    
    let refresh_token = {
        let state_guard = state.read().await;
        state_guard.auth.refresh_token.clone()
            .or_else(|| stored_creds.map(|c| c.refresh_token))
    };
    
    let refresh_token = match refresh_token {
        Some(token) => token,
        None => return Ok(false),
    };
    
    let state_guard = state.read().await;
    let api_endpoint = state_guard.config.api_endpoint.clone();
    drop(state_guard);
    
    let client = reqwest::Client::new();
    
    let response = client
        .post(format!("{}/api/v1/auth/refresh", api_endpoint))
        .json(&serde_json::json!({
            "refresh_token": refresh_token,
        }))
        .send()
        .await
        .map_err(|e| format!("Token refresh failed: {}", e))?;
    
    if !response.status().is_success() {
        CredentialManager::delete_credentials()?;
        let mut state_guard = state.write().await;
        state_guard.clear_auth();
        return Ok(false);
    }
    
    let login_response: LoginResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse refresh response: {}", e))?;
    
    CredentialManager::store_credentials(
        &login_response.refresh_token,
        &login_response.user.id,
    )?;
    
    let mut state_guard = state.write().await;
    state_guard.auth = AuthState {
        is_authenticated: true,
        user: Some(login_response.user),
        access_token: Some(login_response.access_token),
        refresh_token: Some(login_response.refresh_token),
        expires_at: Some(chrono::Utc::now().timestamp() + login_response.expires_in),
    };
    
    Ok(true)
}
