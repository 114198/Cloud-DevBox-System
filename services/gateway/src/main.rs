//! Cloud DevBox API Gateway
//!
//! High-performance API Gateway built with Axum for routing,
//! authentication, rate limiting, and load balancing.
//!
//! # Features
//!
//! - JWT authentication middleware
//! - Token bucket rate limiting with Redis backend
//! - Service discovery and health checking
//! - Request/response logging with tracing
//! - Prometheus metrics endpoint
//! - Graceful shutdown handling

use std::sync::Arc;
use tokio::signal;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use devbox_gateway::{config, router, state, GatewayConfig, AppState};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load environment variables from .env file if present
    dotenvy::dotenv().ok();

    // Initialize tracing subscriber with JSON formatting for production
    let log_format = std::env::var("LOG_FORMAT").unwrap_or_else(|_| "pretty".to_string());
    
    if log_format == "json" {
        tracing_subscriber::registry()
            .with(tracing_subscriber::EnvFilter::new(
                std::env::var("RUST_LOG").unwrap_or_else(|_| "info,tower_http=debug".into()),
            ))
            .with(tracing_subscriber::fmt::layer().json())
            .init();
    } else {
        tracing_subscriber::registry()
            .with(tracing_subscriber::EnvFilter::new(
                std::env::var("RUST_LOG").unwrap_or_else(|_| "info,tower_http=debug".into()),
            ))
            .with(tracing_subscriber::fmt::layer())
            .init();
    }

    tracing::info!("Starting Cloud DevBox API Gateway v{}", env!("CARGO_PKG_VERSION"));

    // Load configuration
    let config = GatewayConfig::from_env();
    tracing::info!(
        host = %config.server.host,
        port = config.server.port,
        "Configuration loaded"
    );

    // Initialize application state
    let state = Arc::new(AppState::new(config.clone()).await?);
    tracing::info!("Application state initialized");

    // Build router with all routes and middleware
    let app = router::build_router(state.clone());

    // Get socket address
    let addr = config.socket_addr();
    tracing::info!("Binding to {}", addr);

    // Create TCP listener
    let listener = tokio::net::TcpListener::bind(addr).await?;
    tracing::info!("Listening on {}", addr);

    // Start server with graceful shutdown
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    tracing::info!("Gateway shutdown complete");
    Ok(())
}

/// Graceful shutdown signal handler
async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("Failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("Failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            tracing::info!("Received Ctrl+C, starting graceful shutdown");
        }
        _ = terminate => {
            tracing::info!("Received SIGTERM, starting graceful shutdown");
        }
    }
}
