//! SSH integration module

use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

use crate::state::AppState;
use crate::config::IdeType;
use crate::environment::EnvironmentClient;

/// SSH configuration for an environment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshConfig {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub private_key_path: Option<String>,
    pub config_content: String,
}

/// Get SSH configuration for an environment
#[tauri::command]
pub async fn get_ssh_config(
    environment_id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<SshConfig, String> {
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
    
    let env = client.get_environment(&environment_id)
        .await
        .map_err(|e| format!("Failed to fetch environment: {}", e))?;
    
    let ssh_info = env.ssh_info
        .ok_or("Environment does not have SSH info")?;
    
    // Generate SSH config content
    let config_content = format!(
        r#"Host devbox-{}
    HostName {}
    Port {}
    User {}
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ServerAliveInterval 60
    ServerAliveCountMax 3
"#,
        environment_id,
        ssh_info.host,
        ssh_info.port,
        ssh_info.username,
    );
    
    Ok(SshConfig {
        host: ssh_info.host,
        port: ssh_info.port,
        username: ssh_info.username,
        private_key_path: ssh_info.private_key,
        config_content,
    })
}

/// Connect to environment via SSH (opens terminal)
#[tauri::command]
pub async fn connect_ssh(
    environment_id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    let ssh_config = get_ssh_config_internal(&environment_id, &state).await?;
    
    // Build SSH command
    let ssh_command = if let Some(key_path) = &ssh_config.private_key_path {
        format!(
            "ssh -i {} -p {} {}@{}",
            key_path,
            ssh_config.port,
            ssh_config.username,
            ssh_config.host
        )
    } else {
        format!(
            "ssh -p {} {}@{}",
            ssh_config.port,
            ssh_config.username,
            ssh_config.host
        )
    };
    
    // Open terminal with SSH command based on OS
    #[cfg(target_os = "windows")]
    {
        std::process::Command::new("cmd")
            .args(["/c", "start", "cmd", "/k", &ssh_command])
            .spawn()
            .map_err(|e| format!("Failed to open terminal: {}", e))?;
    }
    
    #[cfg(target_os = "macos")]
    {
        std::process::Command::new("osascript")
            .args([
                "-e",
                &format!(
                    r#"tell application "Terminal" to do script "{}""#,
                    ssh_command
                ),
            ])
            .spawn()
            .map_err(|e| format!("Failed to open terminal: {}", e))?;
    }
    
    #[cfg(target_os = "linux")]
    {
        // Try common terminal emulators
        let terminals = ["gnome-terminal", "konsole", "xterm", "x-terminal-emulator"];
        let mut opened = false;
        
        for terminal in terminals {
            let result = match terminal {
                "gnome-terminal" => {
                    std::process::Command::new(terminal)
                        .args(["--", "bash", "-c", &format!("{}; exec bash", ssh_command)])
                        .spawn()
                }
                "konsole" => {
                    std::process::Command::new(terminal)
                        .args(["-e", "bash", "-c", &format!("{}; exec bash", ssh_command)])
                        .spawn()
                }
                _ => {
                    std::process::Command::new(terminal)
                        .args(["-e", &ssh_command])
                        .spawn()
                }
            };
            
            if result.is_ok() {
                opened = true;
                break;
            }
        }
        
        if !opened {
            return Err("No supported terminal emulator found".to_string());
        }
    }
    
    Ok(())
}

/// Open environment in VSCode via Remote SSH
#[tauri::command]
pub async fn open_in_vscode(
    environment_id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    let ssh_config = get_ssh_config_internal(&environment_id, &state).await?;
    
    // Ensure SSH config is written
    write_ssh_config(&environment_id, &ssh_config)?;
    
    let state_guard = state.read().await;
    let ide_type = state_guard.config.default_ide.clone();
    drop(state_guard);
    
    let vscode_cmd = match ide_type {
        IdeType::VSCodeInsiders => "code-insiders",
        IdeType::Cursor => "cursor",
        _ => "code",
    };
    
    // Open VSCode with Remote SSH
    let remote_uri = format!(
        "vscode-remote://ssh-remote+devbox-{}/home/{}",
        environment_id,
        ssh_config.username
    );
    
    std::process::Command::new(vscode_cmd)
        .args(["--folder-uri", &remote_uri])
        .spawn()
        .map_err(|e| format!("Failed to open VSCode: {}", e))?;
    
    tracing::info!("Opened VSCode for environment: {}", environment_id);
    Ok(())
}

/// Open environment in JetBrains Gateway
#[tauri::command]
pub async fn open_in_jetbrains(
    environment_id: String,
    ide_type: Option<String>,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<(), String> {
    let ssh_config = get_ssh_config_internal(&environment_id, &state).await?;
    
    // Ensure SSH config is written
    write_ssh_config(&environment_id, &ssh_config)?;
    
    // JetBrains Gateway uses a specific URI scheme
    let gateway_uri = format!(
        "jetbrains-gateway://connect#host=devbox-{}&type=ssh&deploy=false",
        environment_id
    );
    
    // Try to open JetBrains Gateway
    #[cfg(target_os = "windows")]
    {
        std::process::Command::new("cmd")
            .args(["/c", "start", "", &gateway_uri])
            .spawn()
            .map_err(|e| format!("Failed to open JetBrains Gateway: {}", e))?;
    }
    
    #[cfg(target_os = "macos")]
    {
        std::process::Command::new("open")
            .arg(&gateway_uri)
            .spawn()
            .map_err(|e| format!("Failed to open JetBrains Gateway: {}", e))?;
    }
    
    #[cfg(target_os = "linux")]
    {
        std::process::Command::new("xdg-open")
            .arg(&gateway_uri)
            .spawn()
            .map_err(|e| format!("Failed to open JetBrains Gateway: {}", e))?;
    }
    
    tracing::info!("Opened JetBrains Gateway for environment: {}", environment_id);
    Ok(())
}

/// Get list of available IDEs on the system
#[tauri::command]
pub async fn get_available_ides() -> Result<Vec<IdeInfo>, String> {
    let mut ides = Vec::new();
    
    // Check for VSCode
    if is_command_available("code") {
        ides.push(IdeInfo {
            id: "vscode".to_string(),
            name: "Visual Studio Code".to_string(),
            command: "code".to_string(),
            available: true,
        });
    }
    
    // Check for VSCode Insiders
    if is_command_available("code-insiders") {
        ides.push(IdeInfo {
            id: "vscode-insiders".to_string(),
            name: "Visual Studio Code Insiders".to_string(),
            command: "code-insiders".to_string(),
            available: true,
        });
    }
    
    // Check for Cursor
    if is_command_available("cursor") {
        ides.push(IdeInfo {
            id: "cursor".to_string(),
            name: "Cursor".to_string(),
            command: "cursor".to_string(),
            available: true,
        });
    }
    
    // JetBrains Gateway is always listed (user may have it)
    ides.push(IdeInfo {
        id: "jetbrains-gateway".to_string(),
        name: "JetBrains Gateway".to_string(),
        command: "jetbrains-gateway".to_string(),
        available: true, // We can't easily check this
    });
    
    Ok(ides)
}

/// IDE information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdeInfo {
    pub id: String,
    pub name: String,
    pub command: String,
    pub available: bool,
}

/// Check if a command is available in PATH
fn is_command_available(cmd: &str) -> bool {
    #[cfg(target_os = "windows")]
    {
        std::process::Command::new("where")
            .arg(cmd)
            .output()
            .map(|o| o.status.success())
            .unwrap_or(false)
    }
    
    #[cfg(not(target_os = "windows"))]
    {
        std::process::Command::new("which")
            .arg(cmd)
            .output()
            .map(|o| o.status.success())
            .unwrap_or(false)
    }
}

/// Copy SSH command to clipboard
#[tauri::command]
pub async fn copy_ssh_command(
    environment_id: String,
    state: tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<String, String> {
    let ssh_config = get_ssh_config_internal(&environment_id, &state).await?;
    
    let ssh_command = if let Some(key_path) = &ssh_config.private_key_path {
        format!(
            "ssh -i {} -p {} {}@{}",
            key_path,
            ssh_config.port,
            ssh_config.username,
            ssh_config.host
        )
    } else {
        format!(
            "ssh -p {} {}@{}",
            ssh_config.port,
            ssh_config.username,
            ssh_config.host
        )
    };
    
    Ok(ssh_command)
}

/// Internal function to get SSH config
async fn get_ssh_config_internal(
    environment_id: &str,
    state: &tauri::State<'_, Arc<RwLock<AppState>>>,
) -> Result<SshConfig, String> {
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
    
    let env = client.get_environment(environment_id)
        .await
        .map_err(|e| format!("Failed to fetch environment: {}", e))?;
    
    let ssh_info = env.ssh_info
        .ok_or("Environment does not have SSH info")?;
    
    let config_content = format!(
        r#"Host devbox-{}
    HostName {}
    Port {}
    User {}
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ServerAliveInterval 60
    ServerAliveCountMax 3
"#,
        environment_id,
        ssh_info.host,
        ssh_info.port,
        ssh_info.username,
    );
    
    Ok(SshConfig {
        host: ssh_info.host,
        port: ssh_info.port,
        username: ssh_info.username,
        private_key_path: ssh_info.private_key,
        config_content,
    })
}

/// Write SSH config to user's SSH config file
fn write_ssh_config(environment_id: &str, config: &SshConfig) -> Result<(), String> {
    let ssh_dir = dirs::home_dir()
        .ok_or("Could not find home directory")?
        .join(".ssh");
    
    // Ensure .ssh directory exists
    std::fs::create_dir_all(&ssh_dir)
        .map_err(|e| format!("Failed to create .ssh directory: {}", e))?;
    
    let config_path = ssh_dir.join("config");
    
    // Read existing config
    let existing_config = std::fs::read_to_string(&config_path)
        .unwrap_or_default();
    
    let host_marker = format!("Host devbox-{}", environment_id);
    
    // Check if config already exists
    if existing_config.contains(&host_marker) {
        // Config already exists, no need to add
        return Ok(());
    }
    
    // Append new config
    let new_config = if existing_config.is_empty() {
        config.config_content.clone()
    } else {
        format!("{}\n\n{}", existing_config.trim_end(), config.config_content)
    };
    
    std::fs::write(&config_path, new_config)
        .map_err(|e| format!("Failed to write SSH config: {}", e))?;
    
    // Set proper permissions on Unix
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&config_path, std::fs::Permissions::from_mode(0o600))
            .map_err(|e| format!("Failed to set SSH config permissions: {}", e))?;
    }
    
    tracing::info!("SSH config written for environment: {}", environment_id);
    Ok(())
}
