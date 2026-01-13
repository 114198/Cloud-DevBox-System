//! Output formatting utilities
//!
//! Handles formatting CLI output in various formats (table, JSON, YAML).

use colored::Colorize;
use serde::Serialize;
use tabled::{Table, Tabled};

use crate::api::{EnvironmentResponse, TemplateResponse, UserResponse};
use crate::config::OutputFormat;

/// Format and print data based on output format preference
pub fn print_output<T: Serialize + TableDisplay>(data: &T, format: &OutputFormat) {
    match format {
        OutputFormat::Json => print_json(data),
        OutputFormat::Yaml => print_yaml(data),
        OutputFormat::Table => data.print_table(),
    }
}

/// Print data as JSON
pub fn print_json<T: Serialize>(data: &T) {
    match serde_json::to_string_pretty(data) {
        Ok(json) => println!("{}", json),
        Err(e) => eprintln!("{}: {}", "Error".red(), e),
    }
}

/// Print data as YAML
pub fn print_yaml<T: Serialize>(data: &T) {
    match serde_json::to_value(data) {
        Ok(value) => {
            // Simple YAML-like output without external dependency
            print_yaml_value(&value, 0);
        }
        Err(e) => eprintln!("{}: {}", "Error".red(), e),
    }
}

fn print_yaml_value(value: &serde_json::Value, indent: usize) {
    let prefix = "  ".repeat(indent);
    match value {
        serde_json::Value::Object(map) => {
            for (key, val) in map {
                match val {
                    serde_json::Value::Object(_) | serde_json::Value::Array(_) => {
                        println!("{}{}:", prefix, key);
                        print_yaml_value(val, indent + 1);
                    }
                    _ => {
                        println!("{}{}: {}", prefix, key, format_yaml_scalar(val));
                    }
                }
            }
        }
        serde_json::Value::Array(arr) => {
            for item in arr {
                match item {
                    serde_json::Value::Object(_) => {
                        println!("{}-", prefix);
                        print_yaml_value(item, indent + 1);
                    }
                    _ => {
                        println!("{}- {}", prefix, format_yaml_scalar(item));
                    }
                }
            }
        }
        _ => println!("{}{}", prefix, format_yaml_scalar(value)),
    }
}

fn format_yaml_scalar(value: &serde_json::Value) -> String {
    match value {
        serde_json::Value::String(s) => s.clone(),
        serde_json::Value::Number(n) => n.to_string(),
        serde_json::Value::Bool(b) => b.to_string(),
        serde_json::Value::Null => "null".to_string(),
        _ => value.to_string(),
    }
}

/// Trait for types that can be displayed as a table
pub trait TableDisplay {
    fn print_table(&self);
}

/// Environment row for table display
#[derive(Tabled)]
struct EnvironmentRow {
    #[tabled(rename = "NAME")]
    name: String,
    #[tabled(rename = "STATUS")]
    status: String,
    #[tabled(rename = "TEMPLATE")]
    template: String,
    #[tabled(rename = "CPU")]
    cpu: String,
    #[tabled(rename = "MEMORY")]
    memory: String,
    #[tabled(rename = "CREATED")]
    created: String,
}

impl From<&EnvironmentResponse> for EnvironmentRow {
    fn from(env: &EnvironmentResponse) -> Self {
        Self {
            name: env.name.clone(),
            status: colorize_status(&env.status),
            template: env.template_name.clone().unwrap_or_else(|| env.template_id.clone()),
            cpu: env.resources.cpu.clone(),
            memory: env.resources.memory.clone(),
            created: format_timestamp(&env.created_at),
        }
    }
}

impl TableDisplay for Vec<EnvironmentResponse> {
    fn print_table(&self) {
        if self.is_empty() {
            println!("{}", "No environments found.".yellow());
            return;
        }
        
        let rows: Vec<EnvironmentRow> = self.iter().map(|e| e.into()).collect();
        let table = Table::new(rows).to_string();
        println!("{}", table);
    }
}

impl TableDisplay for EnvironmentResponse {
    fn print_table(&self) {
        println!("{}: {}", "Name".bold(), self.name);
        println!("{}: {}", "ID".bold(), self.id);
        println!("{}: {}", "Status".bold(), colorize_status(&self.status));
        println!("{}: {}", "Template".bold(), 
            self.template_name.clone().unwrap_or_else(|| self.template_id.clone()));
        println!("{}: {} CPU, {} Memory, {} Storage", "Resources".bold(),
            self.resources.cpu, self.resources.memory, self.resources.storage);
        println!("{}: {}", "Created".bold(), format_timestamp(&self.created_at));
        
        if let Some(ref url) = self.preview_url {
            println!("{}: {}", "Preview URL".bold(), url.cyan());
        }
        
        if let (Some(host), Some(port)) = (&self.ssh_host, self.ssh_port) {
            println!("{}: ssh -p {} devbox@{}", "SSH".bold(), port, host);
        }
    }
}

/// Template row for table display
#[derive(Tabled)]
struct TemplateRow {
    #[tabled(rename = "NAME")]
    name: String,
    #[tabled(rename = "CATEGORY")]
    category: String,
    #[tabled(rename = "RUNTIME")]
    runtime: String,
    #[tabled(rename = "DESCRIPTION")]
    description: String,
}

impl From<&TemplateResponse> for TemplateRow {
    fn from(tmpl: &TemplateResponse) -> Self {
        Self {
            name: tmpl.name.clone(),
            category: tmpl.category.clone(),
            runtime: format!("{} {}", tmpl.runtime.language, tmpl.runtime.version),
            description: truncate_string(
                tmpl.description.as_deref().unwrap_or(""),
                40
            ),
        }
    }
}

impl TableDisplay for Vec<TemplateResponse> {
    fn print_table(&self) {
        if self.is_empty() {
            println!("{}", "No templates found.".yellow());
            return;
        }
        
        let rows: Vec<TemplateRow> = self.iter().map(|t| t.into()).collect();
        let table = Table::new(rows).to_string();
        println!("{}", table);
    }
}

impl TableDisplay for TemplateResponse {
    fn print_table(&self) {
        println!("{}: {}", "Name".bold(), self.name);
        println!("{}: {}", "Display Name".bold(), self.display_name);
        println!("{}: {}", "ID".bold(), self.id);
        println!("{}: {}", "Category".bold(), self.category);
        println!("{}: {} {}", "Runtime".bold(), 
            self.runtime.language, self.runtime.version);
        if let Some(ref framework) = self.runtime.framework {
            println!("{}: {}", "Framework".bold(), framework);
        }
        if let Some(ref desc) = self.description {
            println!("{}: {}", "Description".bold(), desc);
        }
        if !self.tags.is_empty() {
            println!("{}: {}", "Tags".bold(), self.tags.join(", "));
        }
        println!("{}: {}", "Public".bold(), 
            if self.is_public { "Yes".green() } else { "No".yellow() });
    }
}

impl TableDisplay for UserResponse {
    fn print_table(&self) {
        println!("{}: {}", "Username".bold(), self.username.cyan());
        println!("{}: {}", "Email".bold(), self.email);
        println!("{}: {}", "ID".bold(), self.id);
        if let Some(ref name) = self.display_name {
            println!("{}: {}", "Display Name".bold(), name);
        }
        println!("{}: {}", "Role".bold(), colorize_role(&self.role));
    }
}

/// Colorize environment status
fn colorize_status(status: &str) -> String {
    match status.to_lowercase().as_str() {
        "running" => status.green().to_string(),
        "stopped" => status.yellow().to_string(),
        "creating" => status.blue().to_string(),
        "failed" => status.red().to_string(),
        "suspended" => status.magenta().to_string(),
        _ => status.to_string(),
    }
}

/// Colorize user role
fn colorize_role(role: &str) -> String {
    match role.to_lowercase().as_str() {
        "admin" => role.red().bold().to_string(),
        "org_admin" => role.yellow().bold().to_string(),
        "developer" => role.green().to_string(),
        _ => role.to_string(),
    }
}

/// Format timestamp for display
fn format_timestamp(timestamp: &str) -> String {
    // Parse ISO 8601 timestamp and format as relative time or short date
    if let Ok(dt) = chrono::DateTime::parse_from_rfc3339(timestamp) {
        let now = chrono::Utc::now();
        let duration = now.signed_duration_since(dt);
        
        if duration.num_minutes() < 1 {
            "just now".to_string()
        } else if duration.num_hours() < 1 {
            format!("{}m ago", duration.num_minutes())
        } else if duration.num_days() < 1 {
            format!("{}h ago", duration.num_hours())
        } else if duration.num_days() < 7 {
            format!("{}d ago", duration.num_days())
        } else {
            dt.format("%Y-%m-%d").to_string()
        }
    } else {
        timestamp.to_string()
    }
}

/// Truncate string with ellipsis
fn truncate_string(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}

/// Print success message
pub fn print_success(message: &str) {
    println!("{} {}", "✓".green().bold(), message);
}

/// Print error message
pub fn print_error(message: &str) {
    eprintln!("{} {}", "✗".red().bold(), message);
}

/// Print warning message
pub fn print_warning(message: &str) {
    println!("{} {}", "⚠".yellow().bold(), message);
}

/// Print info message
pub fn print_info(message: &str) {
    println!("{} {}", "ℹ".blue().bold(), message);
}
