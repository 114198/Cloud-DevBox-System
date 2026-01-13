//! Authentication commands
//!
//! Handles login, logout, and user info commands.

use anyhow::Result;
use colored::Colorize;
use dialoguer::{Input, Password};
use indicatif::{ProgressBar, ProgressStyle};
use std::time::Duration;

use crate::api::ApiClient;
use crate::config::{CliConfig, CredentialManager, UserInfo};
use crate::output::{print_error, print_success, print_info, TableDisplay};

/// Handle login command
pub async fn login(endpoint: &str) -> Result<()> {
    // Check if already logged in
    if CredentialManager::is_authenticated() {
        let config = CliConfig::load()?;
        if let Some(user) = &config.current_user {
            print_info(&format!("Already logged in as {}. Use 'devbox logout' first to switch accounts.", 
                user.username.cyan()));
            return Ok(());
        }
    }

    println!("{}", "Cloud DevBox Login".bold());
    println!("Endpoint: {}\n", endpoint.cyan());

    // Get credentials from user
    let email: String = Input::new()
        .with_prompt("Email")
        .interact_text()?;

    let password: String = Password::new()
        .with_prompt("Password")
        .interact()?;

    // Show progress
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message("Authenticating...");
    spinner.enable_steady_tick(Duration::from_millis(100));

    // Attempt login
    let client = ApiClient::with_endpoint(endpoint)?;
    match client.login(&email, &password).await {
        Ok(response) => {
            spinner.finish_and_clear();
            
            // Store token securely
            CredentialManager::store_token(&response.token)?;
            
            // Update config with user info and endpoint
            let mut config = CliConfig::load()?;
            config.api_endpoint = endpoint.to_string();
            config.current_user = Some(UserInfo {
                id: response.user.id.clone(),
                email: response.user.email.clone(),
                username: response.user.username.clone(),
                display_name: response.user.display_name.clone(),
                role: response.user.role.clone(),
            });
            config.save()?;
            
            print_success(&format!("Logged in as {}", response.user.username.cyan()));
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Login failed: {}", e));
            Err(e)
        }
    }
}

/// Handle logout command
pub async fn logout() -> Result<()> {
    if !CredentialManager::is_authenticated() {
        print_info("Not currently logged in.");
        return Ok(());
    }

    // Delete stored token
    CredentialManager::delete_token()?;
    
    // Clear user info from config
    let mut config = CliConfig::load()?;
    let username = config.current_user.as_ref().map(|u| u.username.clone());
    config.current_user = None;
    config.save()?;
    
    if let Some(name) = username {
        print_success(&format!("Logged out from {}", name.cyan()));
    } else {
        print_success("Logged out successfully");
    }
    
    Ok(())
}

/// Handle whoami command
pub async fn whoami() -> Result<()> {
    // First check local cache
    let config = CliConfig::load()?;
    
    if !CredentialManager::is_authenticated() {
        print_error("Not logged in. Use 'devbox login' to authenticate.");
        return Ok(());
    }

    // Try to get fresh user info from API
    let client = ApiClient::new()?;
    
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message("Fetching user info...");
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.whoami().await {
        Ok(user) => {
            spinner.finish_and_clear();
            
            // Update cached user info
            let mut config = CliConfig::load()?;
            config.current_user = Some(UserInfo {
                id: user.id.clone(),
                email: user.email.clone(),
                username: user.username.clone(),
                display_name: user.display_name.clone(),
                role: user.role.clone(),
            });
            config.save()?;
            
            user.print_table();
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            
            // Fall back to cached info if available
            if let Some(user) = config.current_user {
                print_info("Using cached user info (API unavailable):");
                println!();
                println!("{}: {}", "Username".bold(), user.username.cyan());
                println!("{}: {}", "Email".bold(), user.email);
                println!("{}: {}", "ID".bold(), user.id);
                if let Some(ref name) = user.display_name {
                    println!("{}: {}", "Display Name".bold(), name);
                }
                println!("{}: {}", "Role".bold(), user.role);
                Ok(())
            } else {
                print_error(&format!("Failed to get user info: {}", e));
                Err(e)
            }
        }
    }
}
