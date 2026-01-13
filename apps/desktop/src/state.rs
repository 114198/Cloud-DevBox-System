//! Application state management

use serde::{Deserialize, Serialize};
use crate::config::DesktopConfig;

/// User information
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct User {
    pub id: String,
    pub email: String,
    pub username: String,
    pub display_name: String,
    pub avatar: Option<String>,
}

/// Authentication state
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct AuthState {
    pub is_authenticated: bool,
    pub user: Option<User>,
    pub access_token: Option<String>,
    pub refresh_token: Option<String>,
    pub expires_at: Option<i64>,
}

/// Application state
#[derive(Debug, Default)]
pub struct AppState {
    pub config: DesktopConfig,
    pub auth: AuthState,
    pub environments_cache: Vec<crate::environment::Environment>,
    pub last_sync: Option<chrono::DateTime<chrono::Utc>>,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            config: DesktopConfig::load().unwrap_or_default(),
            auth: AuthState::default(),
            environments_cache: Vec::new(),
            last_sync: None,
        }
    }
    
    pub fn is_authenticated(&self) -> bool {
        self.auth.is_authenticated && self.auth.access_token.is_some()
    }
    
    pub fn clear_auth(&mut self) {
        self.auth = AuthState::default();
    }
}
