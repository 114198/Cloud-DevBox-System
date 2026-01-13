//! Environment management commands
//!
//! Handles create, list, start, stop, delete, and SSH commands.

use anyhow::Result;
use colored::Colorize;
use dialoguer::Confirm;
use indicatif::{ProgressBar, ProgressStyle};
use std::process::Command;
use std::time::Duration;

use crate::api::{ApiClient, CreateEnvironmentRequest, ResourcesRequest};
use crate::config::{CliConfig, OutputFormat};
use crate::output::{print_error, print_success, print_info, print_warning, print_output, TableDisplay};

/// Handle environment create command
pub async fn create(
    name: &str,
    template: &str,
    cpu: Option<&str>,
    memory: Option<&str>,
    storage: Option<&str>,
) -> Result<()> {
    let config = CliConfig::load()?;
    let client = ApiClient::new()?;

    // Build resources from args or defaults
    let resources = if cpu.is_some() || memory.is_some() || storage.is_some() {
        let defaults = config.default_resources.as_ref();
        Some(ResourcesRequest {
            cpu: cpu.map(String::from)
                .or_else(|| defaults.map(|d| d.cpu.clone()))
                .unwrap_or_else(|| "2".to_string()),
            memory: memory.map(String::from)
                .or_else(|| defaults.map(|d| d.memory.clone()))
                .unwrap_or_else(|| "4Gi".to_string()),
            storage: storage.map(String::from)
                .or_else(|| defaults.map(|d| d.storage.clone()))
                .unwrap_or_else(|| "20Gi".to_string()),
        })
    } else {
        config.default_resources.as_ref().map(|d| ResourcesRequest {
            cpu: d.cpu.clone(),
            memory: d.memory.clone(),
            storage: d.storage.clone(),
        })
    };

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.green} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Creating environment '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    let request = CreateEnvironmentRequest {
        name: name.to_string(),
        template_id: template.to_string(),
        resources,
        environment_vars: None,
    };

    match client.create_environment(request).await {
        Ok(env) => {
            spinner.finish_and_clear();
            print_success(&format!("Environment '{}' created successfully!", name.cyan()));
            println!();
            env.print_table();
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to create environment: {}", e));
            Err(e)
        }
    }
}

/// Handle environment list command
pub async fn list(status: Option<&str>, format: Option<OutputFormat>) -> Result<()> {
    let config = CliConfig::load()?;
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message("Fetching environments...");
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.list_environments().await {
        Ok(mut environments) => {
            spinner.finish_and_clear();
            
            // Filter by status if specified
            if let Some(status_filter) = status {
                environments.retain(|e| e.status.to_lowercase() == status_filter.to_lowercase());
            }
            
            let output_format = format.unwrap_or(config.output_format);
            print_output(&environments, &output_format);
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to list environments: {}", e));
            Err(e)
        }
    }
}

/// Handle environment start command
pub async fn start(name: &str) -> Result<()> {
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.green} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Starting environment '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.start_environment(name).await {
        Ok(env) => {
            spinner.finish_and_clear();
            print_success(&format!("Environment '{}' started!", name.cyan()));
            
            if let Some(ref url) = env.preview_url {
                print_info(&format!("Preview URL: {}", url.cyan()));
            }
            
            if let (Some(host), Some(port)) = (&env.ssh_host, env.ssh_port) {
                print_info(&format!("SSH: ssh -p {} devbox@{}", port, host));
            }
            
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to start environment: {}", e));
            Err(e)
        }
    }
}

/// Handle environment stop command
pub async fn stop(name: &str) -> Result<()> {
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.yellow} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Stopping environment '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.stop_environment(name).await {
        Ok(_) => {
            spinner.finish_and_clear();
            print_success(&format!("Environment '{}' stopped.", name.cyan()));
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to stop environment: {}", e));
            Err(e)
        }
    }
}

/// Handle environment delete command
pub async fn delete(name: &str, force: bool) -> Result<()> {
    // Confirm deletion unless --force is used
    if !force {
        let confirm = Confirm::new()
            .with_prompt(format!("Are you sure you want to delete environment '{}'?", name.red()))
            .default(false)
            .interact()?;
        
        if !confirm {
            print_info("Deletion cancelled.");
            return Ok(());
        }
    }

    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.red} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Deleting environment '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.delete_environment(name).await {
        Ok(()) => {
            spinner.finish_and_clear();
            print_success(&format!("Environment '{}' deleted.", name));
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to delete environment: {}", e));
            Err(e)
        }
    }
}

/// Handle environment info command
pub async fn info(name: &str, format: Option<OutputFormat>) -> Result<()> {
    let config = CliConfig::load()?;
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Fetching environment '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.get_environment(name).await {
        Ok(env) => {
            spinner.finish_and_clear();
            let output_format = format.unwrap_or(config.output_format);
            print_output(&env, &output_format);
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to get environment: {}", e));
            Err(e)
        }
    }
}

/// Handle SSH command
pub async fn ssh(name: &str) -> Result<()> {
    let client = ApiClient::new()?;

    // First check if environment is running
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Connecting to '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    let env = match client.get_environment(name).await {
        Ok(env) => {
            if env.status.to_lowercase() != "running" {
                spinner.finish_and_clear();
                print_warning(&format!("Environment '{}' is not running (status: {})", 
                    name, env.status));
                print_info("Starting environment...");
                
                // Try to start the environment
                match client.start_environment(name).await {
                    Ok(started_env) => started_env,
                    Err(e) => {
                        print_error(&format!("Failed to start environment: {}", e));
                        return Err(e);
                    }
                }
            } else {
                env
            }
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to get environment: {}", e));
            return Err(e);
        }
    };

    // Get SSH config
    let ssh_config = match client.get_ssh_config(name).await {
        Ok(config) => config,
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to get SSH config: {}", e));
            return Err(e);
        }
    };

    spinner.finish_and_clear();

    // Check if SSH host and port are available
    let (host, port) = match (&env.ssh_host, env.ssh_port) {
        (Some(h), Some(p)) => (h.clone(), p),
        _ => (ssh_config.host.clone(), ssh_config.port),
    };

    print_info(&format!("Connecting to {} on port {}...", host, port));
    println!();

    // Execute SSH command
    let status = Command::new("ssh")
        .arg("-p")
        .arg(port.to_string())
        .arg("-o")
        .arg("StrictHostKeyChecking=no")
        .arg("-o")
        .arg("UserKnownHostsFile=/dev/null")
        .arg(format!("{}@{}", ssh_config.username, host))
        .status();

    match status {
        Ok(exit_status) => {
            if !exit_status.success() {
                print_warning("SSH session ended with non-zero exit code");
            }
            Ok(())
        }
        Err(e) => {
            print_error(&format!("Failed to execute SSH: {}", e));
            print_info("Make sure SSH client is installed and available in PATH");
            Err(anyhow::anyhow!("SSH execution failed: {}", e))
        }
    }
}

/// Handle SSH config generation command
pub async fn ssh_config(name: &str) -> Result<()> {
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message("Generating SSH config...");
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.get_ssh_config(name).await {
        Ok(config) => {
            spinner.finish_and_clear();
            println!("{}", "# Add this to your ~/.ssh/config".dimmed());
            println!("{}", config.config_snippet);
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to get SSH config: {}", e));
            Err(e)
        }
    }
}
