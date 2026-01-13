//! Cloud DevBox Media Service
//!
//! WebRTC SFU media server for video conferencing and screen sharing.

use axum::{
    routing::{delete, get, post, put},
    Router,
};
use std::net::SocketAddr;
use std::sync::Arc;
use tower_http::{cors::CorsLayer, trace::TraceLayer};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

mod config;
mod handlers;
mod models;
mod sfu;

use handlers::AppState;
use sfu::SfuService;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize tracing
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    tracing::info!("Starting Cloud DevBox Media Service");

    // Create SFU service
    let sfu = SfuService::new();
    let state = Arc::new(AppState { sfu });

    // Build router
    let app = Router::new()
        .route("/health", get(handlers::health_check))
        // Room management
        .route("/api/v1/rooms/join", post(handlers::join_room))
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id",
            delete(handlers::leave_room),
        )
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id/media-state",
            put(handlers::update_media_state),
        )
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id/screen-share/start",
            post(handlers::start_screen_share),
        )
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id/screen-share/stop",
            post(handlers::stop_screen_share),
        )
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id/quality",
            put(handlers::update_quality),
        )
        .route(
            "/api/v1/rooms/:meeting_id/participants/:participant_id/bandwidth",
            post(handlers::process_bandwidth),
        )
        .route("/api/v1/rooms/:meeting_id/stats", get(handlers::get_room_stats))
        .route("/api/v1/rooms/:meeting_id", delete(handlers::close_room))
        .layer(TraceLayer::new_for_http())
        .layer(CorsLayer::permissive())
        .with_state(state);

    // Start server
    let addr = SocketAddr::from(([0, 0, 0, 0], 8084));
    tracing::info!("Listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
