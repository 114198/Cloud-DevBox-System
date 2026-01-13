//! Router configuration for the API Gateway
//!
//! Defines all routes and applies middleware layers.

use axum::{
    middleware,
    routing::{any, get, post},
    Router,
};
use std::sync::Arc;
use tower::ServiceBuilder;
use tower_http::{
    compression::CompressionLayer,
    cors::{Any, CorsLayer},
    trace::TraceLayer,
};

use crate::auth;
use crate::config::GatewayConfig;
use crate::handlers;
use crate::middleware as mw;
use crate::rate_limit;
use crate::state::AppState;

/// Build the main application router
pub fn build_router(state: Arc<AppState>) -> Router {
    // Health and system routes (no auth required)
    let health_routes = Router::new()
        .route("/health", get(handlers::health_check))
        .route("/ready", get(handlers::readiness_check))
        .route("/metrics", get(handlers::metrics))
        .route("/version", get(handlers::version_info));

    // Public auth routes (no auth required)
    let public_auth_routes = Router::new()
        .route("/auth/login", post(handlers::login))
        .route("/auth/register", post(handlers::register))
        .route("/auth/refresh", post(handlers::refresh_token));

    // Protected API v1 routes (auth required)
    let protected_routes = Router::new()
        .route("/status", get(handlers::api_status))
        // Proxy routes to backend services
        .route("/core/*path", any(proxy_to_core))
        .route("/realtime/*path", any(proxy_to_realtime))
        .route("/container/*path", any(proxy_to_container))
        .route("/media/*path", any(proxy_to_media))
        // Direct service routes
        .route("/environments", any(proxy_environments))
        .route("/environments/*path", any(proxy_environments))
        .route("/templates", any(proxy_templates))
        .route("/templates/*path", any(proxy_templates))
        .route("/users/me", get(handlers::get_current_user))
        .route("/users", any(proxy_users))
        .route("/users/*path", any(proxy_users))
        .route("/projects", any(proxy_projects))
        .route("/projects/*path", any(proxy_projects))
        // Apply rate limiting middleware first, then JWT auth
        .layer(middleware::from_fn_with_state(state.clone(), auth::jwt_auth_middleware))
        .layer(middleware::from_fn_with_state(state.clone(), rate_limit::rate_limit_middleware));

    // API v1 routes combining public and protected
    let api_v1_routes = Router::new()
        .merge(public_auth_routes)
        .merge(protected_routes);

    // Combine all routes
    let app = Router::new()
        .merge(health_routes)
        .nest("/api/v1", api_v1_routes)
        .with_state(state.clone())
        // Apply middleware layers
        .layer(
            ServiceBuilder::new()
                // Request ID middleware (outermost - runs first)
                .layer(middleware::from_fn(mw::request_id))
                // Request logging middleware
                .layer(middleware::from_fn_with_state(state.clone(), mw::request_logging))
                // Compression
                .layer(CompressionLayer::new())
                // CORS
                .layer(build_cors_layer())
                // Tracing
                .layer(TraceLayer::new_for_http()),
        );

    app
}

/// Build CORS layer
fn build_cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any)
        .expose_headers(Any)
        .max_age(std::time::Duration::from_secs(3600))
}

// Proxy handler functions for different services

async fn proxy_to_core(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    axum::extract::Path(path): axum::extract::Path<String>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), path)), headers, body).await
}

async fn proxy_to_realtime(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    axum::extract::Path(path): axum::extract::Path<String>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    handlers::proxy_request(state, method, axum::extract::Path(("realtime".to_string(), path)), headers, body).await
}

async fn proxy_to_container(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    axum::extract::Path(path): axum::extract::Path<String>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    handlers::proxy_request(state, method, axum::extract::Path(("container".to_string(), path)), headers, body).await
}

async fn proxy_to_media(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    axum::extract::Path(path): axum::extract::Path<String>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    handlers::proxy_request(state, method, axum::extract::Path(("media".to_string(), path)), headers, body).await
}

// Convenience proxy handlers that route to the core service

async fn proxy_environments(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    path: Option<axum::extract::Path<String>>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    let path = path.map(|p| p.0).unwrap_or_default();
    let full_path = if path.is_empty() {
        "environments".to_string()
    } else {
        format!("environments/{}", path)
    };
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), full_path)), headers, body).await
}

async fn proxy_templates(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    path: Option<axum::extract::Path<String>>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    let path = path.map(|p| p.0).unwrap_or_default();
    let full_path = if path.is_empty() {
        "templates".to_string()
    } else {
        format!("templates/{}", path)
    };
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), full_path)), headers, body).await
}

async fn proxy_users(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    path: Option<axum::extract::Path<String>>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    let path = path.map(|p| p.0).unwrap_or_default();
    let full_path = if path.is_empty() {
        "users".to_string()
    } else {
        format!("users/{}", path)
    };
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), full_path)), headers, body).await
}

async fn proxy_projects(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    path: Option<axum::extract::Path<String>>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    let path = path.map(|p| p.0).unwrap_or_default();
    let full_path = if path.is_empty() {
        "projects".to_string()
    } else {
        format!("projects/{}", path)
    };
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), full_path)), headers, body).await
}

async fn proxy_auth(
    state: axum::extract::State<Arc<AppState>>,
    method: axum::http::Method,
    axum::extract::Path(path): axum::extract::Path<String>,
    headers: axum::http::HeaderMap,
    body: Option<String>,
) -> impl axum::response::IntoResponse {
    let full_path = format!("auth/{}", path);
    handlers::proxy_request(state, method, axum::extract::Path(("core".to_string(), full_path)), headers, body).await
}
