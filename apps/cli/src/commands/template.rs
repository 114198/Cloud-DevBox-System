//! Template management commands
//!
//! Handles template listing and details.

use anyhow::Result;
use indicatif::{ProgressBar, ProgressStyle};
use std::time::Duration;

use crate::api::ApiClient;
use crate::config::{CliConfig, OutputFormat};
use crate::output::{print_error, print_output, TableDisplay};

/// Handle template list command
pub async fn list(category: Option<&str>, format: Option<OutputFormat>) -> Result<()> {
    let config = CliConfig::load()?;
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message("Fetching templates...");
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.list_templates(category).await {
        Ok(templates) => {
            spinner.finish_and_clear();
            let output_format = format.unwrap_or(config.output_format);
            print_output(&templates, &output_format);
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to list templates: {}", e));
            Err(e)
        }
    }
}

/// Handle template show command
pub async fn show(name: &str, format: Option<OutputFormat>) -> Result<()> {
    let config = CliConfig::load()?;
    let client = ApiClient::new()?;

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .template("{spinner:.blue} {msg}")
            .unwrap()
    );
    spinner.set_message(format!("Fetching template '{}'...", name));
    spinner.enable_steady_tick(Duration::from_millis(100));

    match client.get_template(name).await {
        Ok(template) => {
            spinner.finish_and_clear();
            let output_format = format.unwrap_or(config.output_format);
            print_output(&template, &output_format);
            Ok(())
        }
        Err(e) => {
            spinner.finish_and_clear();
            print_error(&format!("Failed to get template: {}", e));
            Err(e)
        }
    }
}
