//! CLI configuration management
//!
//! Handles loading, saving, and managing CLI configuration including
//! API endpoints, credentials, and user preferences.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

/// Service name for keyring credential storage
const KEYRING_SERVICE: &str = "cloud-devbox-cli";
const KEYRING_USER: &str = "api-token";

/// CLI configuration structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CliConfig {
    /// API endpoint URL
    pub api_endpoint: String,
    /// Default template for environment creation
    pub default_template: Option<String>,
    /// Default resource configuration
    pub default_resources: Option<ResourceConfig>,
    /// Output format preference
    #[serde(default)]
    pub output_format: OutputFormat,
    /// Enable colored output
    #[serde(default = "default_true")]
    pub color: bool,
    /// Current user info (cached)
    pub current_user: Option<UserInfo>,
}

fn default_true() -> bool {
    true
}

/// Resource configuration defaults
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceConfig {
    pub cpu: String,
    pub memory: String,
    pub storage: String,
}

/// Output format options
#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum OutputFormat {
    #[default]
    Table,
    Json,
    Yaml,
}

/// Cached user information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub id: String,
    pub email: String,
    pub username: String,
    pub display_name: Option<String>,
    pub role: String,
}

impl Default for CliConfig {
    fn default() -> Self {
        Self {
            api_endpoint: "https://api.devbox.io".to_string(),
            default_template: None,
            default_resources: Some(ResourceConfig {
                cpu: "2".to_string(),
                memory: "4Gi".to_string(),
                storage: "20Gi".to_string(),
            }),
            output_format: OutputFormat::Table,
            color: true,
            current_user: None,
        }
    }
}

impl CliConfig {
    /// Get the configuration directory path
    pub fn config_dir() -> PathBuf {
        dirs::config_dir()
            .unwrap_or_else(|| PathBuf::from("."))
            .join("devbox")
    }

    /// Get the configuration file path
    pub fn config_path() -> PathBuf {
        Self::config_dir().join("config.json")
    }

    /// Load configuration from file, creating default if not exists
    pub fn load() -> Result<Self> {
        let path = Self::config_path();
        
        if path.exists() {
            let content = fs::read_to_string(&path)
                .with_context(|| format!("Failed to read config file: {}", path.display()))?;
            let config: CliConfig = serde_json::from_str(&content)
                .with_context(|| "Failed to parse config file")?;
            Ok(config)
        } else {
            let config = Self::default();
            config.save()?;
            Ok(config)
        }
    }

    /// Save configuration to file
    pub fn save(&self) -> Result<()> {
        let path = Self::config_path();
        let dir = Self::config_dir();
        
        // Create config directory if it doesn't exist
        if !dir.exists() {
            fs::create_dir_all(&dir)
                .with_context(|| format!("Failed to create config directory: {}", dir.display()))?;
        }
        
        let content = serde_json::to_string_pretty(self)
            .with_context(|| "Failed to serialize config")?;
        fs::write(&path, content)
            .with_context(|| format!("Failed to write config file: {}", path.display()))?;
        
        Ok(())
    }

    /// Get a configuration value by key
    pub fn get(&self, key: &str) -> Option<String> {
        match key {
            "api_endpoint" | "endpoint" => Some(self.api_endpoint.clone()),
            "default_template" | "template" => self.default_template.clone(),
            "output_format" | "format" => Some(format!("{:?}", self.output_format).to_lowercase()),
            "color" => Some(self.color.to_string()),
            "default_cpu" | "cpu" => self.default_resources.as_ref().map(|r| r.cpu.clone()),
            "default_memory" | "memory" => self.default_resources.as_ref().map(|r| r.memory.clone()),
            "default_storage" | "storage" => self.default_resources.as_ref().map(|r| r.storage.clone()),
            _ => None,
        }
    }

    /// Set a configuration value by key
    pub fn set(&mut self, key: &str, value: &str) -> Result<()> {
        match key {
            "api_endpoint" | "endpoint" => {
                self.api_endpoint = value.to_string();
            }
            "default_template" | "template" => {
                self.default_template = Some(value.to_string());
            }
            "output_format" | "format" => {
                self.output_format = match value.to_lowercase().as_str() {
                    "json" => OutputFormat::Json,
                    "yaml" => OutputFormat::Yaml,
                    "table" => OutputFormat::Table,
                    _ => anyhow::bail!("Invalid output format: {}. Use 'table', 'json', or 'yaml'", value),
                };
            }
            "color" => {
                self.color = value.parse()
                    .with_context(|| format!("Invalid boolean value: {}", value))?;
            }
            "default_cpu" | "cpu" => {
                if let Some(ref mut resources) = self.default_resources {
                    resources.cpu = value.to_string();
                } else {
                    self.default_resources = Some(ResourceConfig {
                        cpu: value.to_string(),
                        memory: "4Gi".to_string(),
                        storage: "20Gi".to_string(),
                    });
                }
            }
            "default_memory" | "memory" => {
                if let Some(ref mut resources) = self.default_resources {
                    resources.memory = value.to_string();
                } else {
                    self.default_resources = Some(ResourceConfig {
                        cpu: "2".to_string(),
                        memory: value.to_string(),
                        storage: "20Gi".to_string(),
                    });
                }
            }
            "default_storage" | "storage" => {
                if let Some(ref mut resources) = self.default_resources {
                    resources.storage = value.to_string();
                } else {
                    self.default_resources = Some(ResourceConfig {
                        cpu: "2".to_string(),
                        memory: "4Gi".to_string(),
                        storage: value.to_string(),
                    });
                }
            }
            _ => anyhow::bail!("Unknown configuration key: {}", key),
        }
        self.save()?;
        Ok(())
    }

    /// List all available configuration keys
    pub fn list_keys() -> Vec<&'static str> {
        vec![
            "api_endpoint",
            "default_template",
            "output_format",
            "color",
            "default_cpu",
            "default_memory",
            "default_storage",
        ]
    }
}

/// Credential manager for secure token storage
pub struct CredentialManager;

impl CredentialManager {
    /// Store API token securely
    pub fn store_token(token: &str) -> Result<()> {
        let entry = keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER)
            .with_context(|| "Failed to create keyring entry")?;
        entry.set_password(token)
            .with_context(|| "Failed to store token in keyring")?;
        Ok(())
    }

    /// Retrieve API token
    pub fn get_token() -> Result<Option<String>> {
        let entry = keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER)
            .with_context(|| "Failed to create keyring entry")?;
        
        match entry.get_password() {
            Ok(token) => Ok(Some(token)),
            Err(keyring::Error::NoEntry) => Ok(None),
            Err(e) => Err(anyhow::anyhow!("Failed to retrieve token: {}", e)),
        }
    }

    /// Delete stored token
    pub fn delete_token() -> Result<()> {
        let entry = keyring::Entry::new(KEYRING_SERVICE, KEYRING_USER)
            .with_context(|| "Failed to create keyring entry")?;
        
        match entry.delete_credential() {
            Ok(()) => Ok(()),
            Err(keyring::Error::NoEntry) => Ok(()), // Already deleted
            Err(e) => Err(anyhow::anyhow!("Failed to delete token: {}", e)),
        }
    }

    /// Check if user is authenticated
    pub fn is_authenticated() -> bool {
        Self::get_token().ok().flatten().is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_default_config() {
        let config = CliConfig::default();
        assert_eq!(config.api_endpoint, "https://api.devbox.io");
        assert!(config.color);
        assert_eq!(config.output_format, OutputFormat::Table);
    }

    #[test]
    fn test_config_get_set() {
        let mut config = CliConfig::default();
        
        config.set("api_endpoint", "https://custom.api.io").unwrap();
        assert_eq!(config.get("api_endpoint"), Some("https://custom.api.io".to_string()));
        
        config.set("output_format", "json").unwrap();
        assert_eq!(config.output_format, OutputFormat::Json);
    }

    #[test]
    fn test_config_list_keys() {
        let keys = CliConfig::list_keys();
        assert!(keys.contains(&"api_endpoint"));
        assert!(keys.contains(&"output_format"));
    }
}
