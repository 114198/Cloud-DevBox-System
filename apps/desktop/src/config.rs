//! Desktop app configuration

use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Desktop application configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DesktopConfig {
    /// API endpoint URL
    pub api_endpoint: String,
    /// Whether to start the app on system boot
    pub auto_start: bool,
    /// Whether to minimize to tray instead of closing
    pub minimize_to_tray: bool,
    /// Show notifications for environment status changes
    pub show_notifications: bool,
    /// Default IDE for SSH connections
    pub default_ide: IdeType,
    /// Theme preference
    pub theme: Theme,
    /// Language preference
    pub language: String,
    /// Check for updates automatically
    pub auto_check_updates: bool,
    /// SSH configuration
    pub ssh: SshConfig,
    /// Keyboard shortcuts
    pub shortcuts: ShortcutConfig,
}

/// Supported IDE types
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum IdeType {
    #[default]
    VSCode,
    VSCodeInsiders,
    JetBrainsGateway,
    Cursor,
    Custom(String),
}

/// Theme preference
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum Theme {
    Light,
    Dark,
    #[default]
    System,
}

/// SSH configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshConfig {
    /// Path to SSH private key
    pub private_key_path: Option<PathBuf>,
    /// SSH connection timeout in seconds
    pub connection_timeout: u32,
    /// Keep alive interval in seconds
    pub keep_alive_interval: u32,
}

impl Default for SshConfig {
    fn default() -> Self {
        Self {
            private_key_path: None,
            connection_timeout: 30,
            keep_alive_interval: 60,
        }
    }
}

/// Keyboard shortcut configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShortcutConfig {
    /// Toggle main window visibility
    pub toggle_window: String,
    /// Quick create environment
    pub quick_create: String,
    /// Open environment list
    pub open_environments: String,
}

impl Default for ShortcutConfig {
    fn default() -> Self {
        Self {
            toggle_window: "CmdOrCtrl+Shift+D".to_string(),
            quick_create: "CmdOrCtrl+Shift+N".to_string(),
            open_environments: "CmdOrCtrl+Shift+E".to_string(),
        }
    }
}

impl Default for DesktopConfig {
    fn default() -> Self {
        Self {
            api_endpoint: "https://api.devbox.io".to_string(),
            auto_start: false,
            minimize_to_tray: true,
            show_notifications: true,
            default_ide: IdeType::default(),
            theme: Theme::default(),
            language: "zh-CN".to_string(),
            auto_check_updates: true,
            ssh: SshConfig::default(),
            shortcuts: ShortcutConfig::default(),
        }
    }
}

impl DesktopConfig {
    /// Get the configuration file path
    pub fn config_path() -> PathBuf {
        dirs::config_dir()
            .unwrap_or_else(|| PathBuf::from("."))
            .join("devbox")
            .join("config.json")
    }
    
    /// Load configuration from file
    pub fn load() -> Result<Self, Box<dyn std::error::Error>> {
        let path = Self::config_path();
        if path.exists() {
            let content = std::fs::read_to_string(&path)?;
            let config: Self = serde_json::from_str(&content)?;
            Ok(config)
        } else {
            Ok(Self::default())
        }
    }
    
    /// Save configuration to file
    pub fn save(&self) -> Result<(), Box<dyn std::error::Error>> {
        let path = Self::config_path();
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        let content = serde_json::to_string_pretty(self)?;
        std::fs::write(&path, content)?;
        Ok(())
    }
}

// Tauri commands for configuration

/// Get current configuration
#[tauri::command]
pub async fn get_config(
    state: tauri::State<'_, std::sync::Arc<tokio::sync::RwLock<crate::state::AppState>>>,
) -> Result<DesktopConfig, String> {
    let state = state.read().await;
    Ok(state.config.clone())
}

/// Update configuration
#[tauri::command]
pub async fn set_config(
    config: DesktopConfig,
    state: tauri::State<'_, std::sync::Arc<tokio::sync::RwLock<crate::state::AppState>>>,
) -> Result<(), String> {
    // Save to file
    config.save().map_err(|e| e.to_string())?;
    
    // Update state
    let mut state = state.write().await;
    state.config = config;
    
    Ok(())
}

/// Get API endpoint
#[tauri::command]
pub async fn get_api_endpoint(
    state: tauri::State<'_, std::sync::Arc<tokio::sync::RwLock<crate::state::AppState>>>,
) -> Result<String, String> {
    let state = state.read().await;
    Ok(state.config.api_endpoint.clone())
}
