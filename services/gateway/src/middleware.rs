//! Gateway middleware for authentication, rate limiting, logging, etc.
//!
//! Provides middleware layers for request processing including:
//! - Request/response logging
//! - Request ID generation
//! - Metrics collection

use axum::{
    body::Body,
    extract::State,
    http::{Request, Response},
    middleware::Next,
};
use std::sync::Arc;
use std::time::Instant;
use uuid::Uuid;

use crate::state::AppState;

/// Request ID header name
pub const REQUEST_ID_HEADER: &str = "x-request-id";

/// User ID header name (set after authentication)
pub const USER_ID_HEADER: &str = "x-user-id";

/// Request logging middleware
///
/// Logs incoming requests and outgoing responses with timing information.
pub async fn request_logging(
    State(state): State<Arc<AppState>>,
    request: Request<Body>,
    next: Next,
) -> Response<Body> {
    let start = Instant::now();
    
    // Generate or extract request ID
    let request_id = request
        .headers()
        .get(REQUEST_ID_HEADER)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .unwrap_or_else(|| Uuid::new_v4().to_string());

    let method = request.method().clone();
    let uri = request.uri().clone();
    let version = request.version();

    // Log incoming request
    tracing::info!(
        request_id = %request_id,
        method = %method,
        uri = %uri,
        version = ?version,
        "Incoming request"
    );

    // Increment metrics
    state.increment_requests();

    // Process request
    let response = next.run(request).await;

    // Calculate duration
    let duration = start.elapsed();
    let duration_ms = duration.as_millis() as u64;

    // Update metrics
    state.complete_request(duration_ms);

    let status = response.status();
    if status.is_server_error() || status.is_client_error() {
        state.increment_errors();
    }

    // Log response
    tracing::info!(
        request_id = %request_id,
        method = %method,
        uri = %uri,
        status = %status.as_u16(),
        duration_ms = duration_ms,
        "Request completed"
    );

    response
}

/// Request ID middleware
///
/// Ensures every request has a unique request ID for tracing.
pub async fn request_id(
    mut request: Request<Body>,
    next: Next,
) -> Response<Body> {
    // Generate request ID if not present
    let request_id = request
        .headers()
        .get(REQUEST_ID_HEADER)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
        .unwrap_or_else(|| Uuid::new_v4().to_string());

    // Insert request ID into headers
    request.headers_mut().insert(
        REQUEST_ID_HEADER,
        request_id.parse().unwrap(),
    );

    let mut response = next.run(request).await;

    // Add request ID to response headers
    response.headers_mut().insert(
        REQUEST_ID_HEADER,
        request_id.parse().unwrap(),
    );

    response
}

/// Timeout middleware configuration
pub mod timeout {
    use axum::{
        body::Body,
        http::{Request, Response, StatusCode},
    };
    use std::time::Duration;
    use tower::timeout::TimeoutLayer;

    /// Create a timeout layer with the specified duration
    pub fn layer(timeout_secs: u64) -> TimeoutLayer {
        TimeoutLayer::new(Duration::from_secs(timeout_secs))
    }
}
