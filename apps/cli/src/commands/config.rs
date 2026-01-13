//! Configuration management commands
//!
//! Handles config show, get, and set commands.

use anyhow::Result;
use colored::Colorize;

use crate::config::CliConfig;
use crate::output::{print_error, print_success, print_info};

/// Handle config show command
pub fn show() -> Result<()> {
    let config = CliConfig::load()?;
    
    println!("{}", "Current Configuration".bold());
    println!("{}", "─".repeat(40));
    
    println!("{}: {}", "API Endpoint".bold(), config.api_endpoint.cyan());
    
    if let Some(ref template) = config.default_template {
        println!("{}: {}", "Default Template".bold(), template);
    } else {
        println!("{}: {}", "Default Template".bold(), "(not set)".dimmed());
    }
    
    println!("{}: {:?}", "Output Format".bold(), config.output_format);
    println!("{}: {}", "Color Output".bold(), 
        if config.color { "enabled".green() } else { "disabled".yellow() });
    
    if let Some(ref resources) = config.default_resources {
        println!();
        println!("{}", "Default Resources".bold());
        println!("  {}: {}", "CPU".bold(), resources.cpu);
        println!("  {}: {}", "Memory".bold(), resources.memory);
        println!("  {}: {}", "Storage".bold(), resources.storage);
    }
    
    if let Some(ref user) = config.current_user {
        println!();
        println!("{}", "Current User".bold());
        println!("  {}: {}", "Username".bold(), user.username.cyan());
        println!("  {}: {}", "Email".bold(), user.email);
        println!("  {}: {}", "Role".bold(), user.role);
    } else {
        println!();
        println!("{}: {}", "Current User".bold(), "(not logged in)".dimmed());
    }
    
    println!();
    println!("{}: {}", "Config File".dimmed(), 
        CliConfig::config_path().display());
    
    Ok(())
}

/// Handle config get command
pub fn get(key: &str) -> Result<()> {
    let config = CliConfig::load()?;
    
    match config.get(key) {
        Some(value) => {
            println!("{}", value);
            Ok(())
        }
        None => {
            print_error(&format!("Unknown configuration key: {}", key));
            print_info(&format!("Available keys: {}", CliConfig::list_keys().join(", ")));
            Err(anyhow::anyhow!("Unknown key: {}", key))
        }
    }
}

/// Handle config set command
pub fn set(key: &str, value: &str) -> Result<()> {
    let mut config = CliConfig::load()?;
    
    match config.set(key, value) {
        Ok(()) => {
            print_success(&format!("Set {} = {}", key.cyan(), value.green()));
            Ok(())
        }
        Err(e) => {
            print_error(&format!("Failed to set configuration: {}", e));
            Err(e)
        }
    }
}

/// Handle config list command (list all available keys)
pub fn list_keys() -> Result<()> {
    println!("{}", "Available Configuration Keys".bold());
    println!("{}", "─".repeat(40));
    
    let keys = CliConfig::list_keys();
    for key in keys {
        let config = CliConfig::load()?;
        let value = config.get(key).unwrap_or_else(|| "(not set)".to_string());
        println!("  {}: {}", key.cyan(), value.dimmed());
    }
    
    Ok(())
}

/// Handle config reset command
pub fn reset() -> Result<()> {
    let config = CliConfig::default();
    config.save()?;
    print_success("Configuration reset to defaults");
    Ok(())
}
