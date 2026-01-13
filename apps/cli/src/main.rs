//! Cloud DevBox CLI Tool
//!
//! Command-line interface for managing cloud development environments.
//! 
//! # Usage
//! 
//! ```bash
//! # Login to Cloud DevBox
//! devbox login
//! 
//! # List environments
//! devbox env list
//! 
//! # Create a new environment
//! devbox env create my-project --template nodejs
//! 
//! # SSH into an environment
//! devbox env ssh my-project
//! ```

use clap::{CommandFactory, Parser, Subcommand, ValueEnum};
use clap_complete::{generate, Shell};
use std::io;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

mod api;
mod commands;
mod config;
mod output;

use config::OutputFormat;

#[derive(Parser)]
#[command(name = "devbox")]
#[command(author = "Cloud DevBox Team")]
#[command(version = env!("CARGO_PKG_VERSION"))]
#[command(about = "Cloud DevBox CLI - Manage your cloud development environments")]
#[command(long_about = "Cloud DevBox CLI provides a command-line interface for managing \
    cloud development environments. Create, start, stop, and connect to your \
    development environments from anywhere.")]
#[command(propagate_version = true)]
struct Cli {
    /// Enable verbose output
    #[arg(short, long, global = true)]
    verbose: bool,

    /// Output format
    #[arg(short, long, global = true, value_enum)]
    format: Option<OutputFormatArg>,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Clone, ValueEnum)]
enum OutputFormatArg {
    Table,
    Json,
    Yaml,
}

impl From<OutputFormatArg> for OutputFormat {
    fn from(arg: OutputFormatArg) -> Self {
        match arg {
            OutputFormatArg::Table => OutputFormat::Table,
            OutputFormatArg::Json => OutputFormat::Json,
            OutputFormatArg::Yaml => OutputFormat::Yaml,
        }
    }
}

#[derive(Subcommand)]
enum Commands {
    /// Login to Cloud DevBox
    Login {
        /// API endpoint URL
        #[arg(short, long, default_value = "https://api.devbox.io")]
        endpoint: String,
    },

    /// Logout from Cloud DevBox
    Logout,

    /// Show current user information
    Whoami,

    /// Environment management commands
    #[command(subcommand, alias = "environment")]
    Env(EnvCommands),

    /// Create a new environment (shortcut for 'env create')
    Create {
        /// Environment name
        name: String,

        /// Template to use
        #[arg(short, long)]
        template: String,

        /// CPU allocation (e.g., "2", "4")
        #[arg(long)]
        cpu: Option<String>,

        /// Memory allocation (e.g., "4Gi", "8Gi")
        #[arg(long)]
        memory: Option<String>,

        /// Storage allocation (e.g., "20Gi", "50Gi")
        #[arg(long)]
        storage: Option<String>,
    },

    /// List all environments (shortcut for 'env list')
    #[command(alias = "ls")]
    List {
        /// Filter by status
        #[arg(short, long)]
        status: Option<String>,
    },

    /// Start an environment (shortcut for 'env start')
    Start {
        /// Environment name or ID
        name: String,
    },

    /// Stop an environment (shortcut for 'env stop')
    Stop {
        /// Environment name or ID
        name: String,
    },

    /// Delete an environment (shortcut for 'env delete')
    #[command(alias = "rm")]
    Delete {
        /// Environment name or ID
        name: String,

        /// Skip confirmation
        #[arg(short, long)]
        force: bool,
    },

    /// SSH into an environment (shortcut for 'env ssh')
    Ssh {
        /// Environment name or ID
        name: String,
    },

    /// Template management commands
    #[command(subcommand)]
    Template(TemplateCommands),

    /// Configuration management commands
    #[command(subcommand)]
    Config(ConfigCommands),

    /// Generate shell completions
    Completions {
        /// Shell to generate completions for
        #[arg(value_enum)]
        shell: Shell,
    },

    /// Show version information
    Version,
}

#[derive(Subcommand)]
enum EnvCommands {
    /// Create a new environment
    Create {
        /// Environment name
        name: String,

        /// Template to use
        #[arg(short, long)]
        template: String,

        /// CPU allocation (e.g., "2", "4")
        #[arg(long)]
        cpu: Option<String>,

        /// Memory allocation (e.g., "4Gi", "8Gi")
        #[arg(long)]
        memory: Option<String>,

        /// Storage allocation (e.g., "20Gi", "50Gi")
        #[arg(long)]
        storage: Option<String>,
    },

    /// List all environments
    #[command(alias = "ls")]
    List {
        /// Filter by status (running, stopped, creating, failed)
        #[arg(short, long)]
        status: Option<String>,
    },

    /// Show environment details
    Info {
        /// Environment name or ID
        name: String,
    },

    /// Start an environment
    Start {
        /// Environment name or ID
        name: String,
    },

    /// Stop an environment
    Stop {
        /// Environment name or ID
        name: String,
    },

    /// Delete an environment
    #[command(alias = "rm")]
    Delete {
        /// Environment name or ID
        name: String,

        /// Skip confirmation
        #[arg(short, long)]
        force: bool,
    },

    /// SSH into an environment
    Ssh {
        /// Environment name or ID
        name: String,
    },

    /// Generate SSH config for an environment
    SshConfig {
        /// Environment name or ID
        name: String,
    },
}

#[derive(Subcommand)]
enum TemplateCommands {
    /// List available templates
    #[command(alias = "ls")]
    List {
        /// Filter by category
        #[arg(short, long)]
        category: Option<String>,
    },

    /// Show template details
    Show {
        /// Template name or ID
        name: String,
    },
}

#[derive(Subcommand)]
enum ConfigCommands {
    /// Show current configuration
    Show,

    /// Set a configuration value
    Set {
        /// Configuration key
        key: String,

        /// Configuration value
        value: String,
    },

    /// Get a configuration value
    Get {
        /// Configuration key
        key: String,
    },

    /// List all available configuration keys
    #[command(alias = "ls")]
    List,

    /// Reset configuration to defaults
    Reset,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    // Initialize logging if verbose
    if cli.verbose {
        tracing_subscriber::registry()
            .with(tracing_subscriber::fmt::layer())
            .with(tracing_subscriber::EnvFilter::new("debug"))
            .init();
    }

    // Convert format argument
    let format = cli.format.map(OutputFormat::from);

    match cli.command {
        // Auth commands
        Commands::Login { endpoint } => {
            commands::auth::login(&endpoint).await
        }
        Commands::Logout => {
            commands::auth::logout().await
        }
        Commands::Whoami => {
            commands::auth::whoami().await
        }

        // Environment shortcuts
        Commands::Create { name, template, cpu, memory, storage } => {
            commands::environment::create(
                &name,
                &template,
                cpu.as_deref(),
                memory.as_deref(),
                storage.as_deref(),
            ).await
        }
        Commands::List { status } => {
            commands::environment::list(status.as_deref(), format).await
        }
        Commands::Start { name } => {
            commands::environment::start(&name).await
        }
        Commands::Stop { name } => {
            commands::environment::stop(&name).await
        }
        Commands::Delete { name, force } => {
            commands::environment::delete(&name, force).await
        }
        Commands::Ssh { name } => {
            commands::environment::ssh(&name).await
        }

        // Environment subcommands
        Commands::Env(cmd) => match cmd {
            EnvCommands::Create { name, template, cpu, memory, storage } => {
                commands::environment::create(
                    &name,
                    &template,
                    cpu.as_deref(),
                    memory.as_deref(),
                    storage.as_deref(),
                ).await
            }
            EnvCommands::List { status } => {
                commands::environment::list(status.as_deref(), format).await
            }
            EnvCommands::Info { name } => {
                commands::environment::info(&name, format).await
            }
            EnvCommands::Start { name } => {
                commands::environment::start(&name).await
            }
            EnvCommands::Stop { name } => {
                commands::environment::stop(&name).await
            }
            EnvCommands::Delete { name, force } => {
                commands::environment::delete(&name, force).await
            }
            EnvCommands::Ssh { name } => {
                commands::environment::ssh(&name).await
            }
            EnvCommands::SshConfig { name } => {
                commands::environment::ssh_config(&name).await
            }
        },

        // Template commands
        Commands::Template(cmd) => match cmd {
            TemplateCommands::List { category } => {
                commands::template::list(category.as_deref(), format).await
            }
            TemplateCommands::Show { name } => {
                commands::template::show(&name, format).await
            }
        },

        // Config commands
        Commands::Config(cmd) => match cmd {
            ConfigCommands::Show => {
                commands::config::show()
            }
            ConfigCommands::Set { key, value } => {
                commands::config::set(&key, &value)
            }
            ConfigCommands::Get { key } => {
                commands::config::get(&key)
            }
            ConfigCommands::List => {
                commands::config::list_keys()
            }
            ConfigCommands::Reset => {
                commands::config::reset()
            }
        },

        // Version command
        Commands::Version => {
            println!("devbox {}", env!("CARGO_PKG_VERSION"));
            println!("Cloud DevBox CLI - Manage your cloud development environments");
            Ok(())
        }

        // Completions command
        Commands::Completions { shell } => {
            let mut cmd = Cli::command();
            generate(shell, &mut cmd, "devbox", &mut io::stdout());
            Ok(())
        }
    }
}
