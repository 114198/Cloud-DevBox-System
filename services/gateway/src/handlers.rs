//! HTTP handlers for the gateway
//!
//! Provides health check, status, and proxy handlers for routing
//! requests to backend services.

use axum::{
    extract::{Path, State},
    http::{HeaderMap, Method, StatusCode},
    response::IntoResponse,
    Json,
};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::sync::Arc;

use crate::auth::AuthUser;
use crate::error::GatewayError;
use crate::state::AppState;

/// Health check response
#[derive(Debug, Serialize)]
pub struct HealthResponse {
    pub status: String,
    pub version: String,
    pub timestamp: String,
}

/// API status response
#[derive(Debug, Serialize)]
pub struct StatusResponse {
    pub gateway: GatewayStatus,
    pub services: ServicesStatus,
}

/// Gateway status details
#[derive(Debug, Serialize)]
pub struct GatewayStatus {
    pub status: String,
    pub version: String,
    pub uptime_secs: u64,
}

/// Backend services status
#[derive(Debug, Serialize)]
pub struct ServicesStatus {
    pub core: ServiceHealth,
    pub realtime: ServiceHealth,
    pub container: ServiceHealth,
    pub media: ServiceHealth,
}

/// Individual service health status
#[derive(Debug, Serialize)]
pub struct ServiceHealth {
    pub status: String,
    pub latency_ms: Option<u64>,
    pub last_check: Option<String>,
}

/// Health check endpoint - GET /health
pub async fn health_check() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "healthy".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
        timestamp: chrono::Utc::now().to_rfc3339(),
    })
}

/// Readiness check endpoint - GET /ready
pub async fn readiness_check(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    // Check if Redis is connected (for rate limiting)
    let redis_ok = state.check_redis_connection().await;
    
    if redis_ok {
        (StatusCode::OK, Json(serde_json::json!({
            "status": "ready",
            "checks": {
                "redis": "ok"
            }
        })))
    } else {
        (StatusCode::SERVICE_UNAVAILABLE, Json(serde_json::json!({
            "status": "not_ready",
            "checks": {
                "redis": "failed"
            }
        })))
    }
}

/// API status endpoint - GET /api/v1/status
pub async fn api_status(State(state): State<Arc<AppState>>) -> Json<StatusResponse> {
    let uptime = state.uptime_secs();
    
    // Check backend services health
    let services = state.check_services_health().await;
    
    Json(StatusResponse {
        gateway: GatewayStatus {
            status: "running".to_string(),
            version: env!("CARGO_PKG_VERSION").to_string(),
            uptime_secs: uptime,
        },
        services,
    })
}

/// Metrics endpoint - GET /metrics
pub async fn metrics(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let metrics = state.get_metrics();
    
    // Return Prometheus-compatible metrics format
    let mut output = String::new();
    
    output.push_str(&format!(
        "# HELP gateway_requests_total Total number of requests\n\
         # TYPE gateway_requests_total counter\n\
         gateway_requests_total {}\n",
        metrics.total_requests
    ));
    
    output.push_str(&format!(
        "# HELP gateway_requests_active Current active requests\n\
         # TYPE gateway_requests_active gauge\n\
         gateway_requests_active {}\n",
        metrics.active_requests
    ));
    
    output.push_str(&format!(
        "# HELP gateway_request_duration_seconds Request duration histogram\n\
         # TYPE gateway_request_duration_seconds histogram\n\
         gateway_request_duration_seconds_sum {}\n\
         gateway_request_duration_seconds_count {}\n",
        metrics.total_duration_secs,
        metrics.total_requests
    ));
    
    (
        StatusCode::OK,
        [("content-type", "text/plain; charset=utf-8")],
        output,
    )
}

/// Generic proxy handler for forwarding requests to backend services
#[derive(Debug, Deserialize)]
pub struct ProxyPath {
    pub service: String,
    pub path: String,
}

/// Proxy request to backend service
pub async fn proxy_request(
    State(state): State<Arc<AppState>>,
    method: Method,
    Path((service, path)): Path<(String, String)>,
    headers: HeaderMap,
    body: Option<String>,
) -> impl IntoResponse {
    tracing::debug!(
        service = %service,
        path = %path,
        method = %method,
        "Proxying request"
    );
    
    match state.proxy_request(&service, &path, method, headers, body).await {
        Ok(response) => response,
        Err(e) => {
            tracing::error!(error = %e, "Proxy request failed");
            e.into_response()
        }
    }
}

/// Version info endpoint - GET /version
pub async fn version_info() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "name": "Cloud DevBox API Gateway",
        "version": env!("CARGO_PKG_VERSION"),
        "rust_version": env!("CARGO_PKG_RUST_VERSION"),
        "build_timestamp": option_env!("BUILD_TIMESTAMP").unwrap_or("unknown"),
        "git_commit": option_env!("GIT_COMMIT").unwrap_or("unknown"),
    }))
}

// ============================================================================
// Authentication Handlers
// ============================================================================

/// Login request body
#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

/// Login response
#[derive(Debug, Serialize)]
pub struct LoginResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub token_type: String,
    pub expires_in: i64,
    pub user: UserInfo,
}

/// User info in login response
#[derive(Debug, Serialize)]
pub struct UserInfo {
    pub id: String,
    pub email: String,
    pub username: String,
    pub role: String,
}

#[derive(Debug, Deserialize)]
struct CoreTokenResponse {
    access_token: String,
    refresh_token: String,
    expires_at: String,
    token_type: String,
}

#[derive(Debug, Deserialize)]
struct CoreUserResponse {
    id: String,
    email: String,
    username: String,
    role: String,
}

#[derive(Debug, Deserialize)]
struct CoreAuthResponse {
    user: CoreUserResponse,
    tokens: CoreTokenResponse,
}

fn expires_in_from_core(expires_at: &str, fallback: i64) -> i64 {
    if let Ok(parsed) = DateTime::parse_from_rfc3339(expires_at) {
        let now = Utc::now();
        let remaining = parsed.with_timezone(&Utc).signed_duration_since(now).num_seconds();
        if remaining > 0 {
            return remaining;
        }
    }
    fallback
}

/// Register request body
#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub email: String,
    pub username: String,
    pub password: String,
}

/// Refresh token request body
#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

/// Login handler - POST /api/v1/auth/login
///
/// Note: In production, this would validate credentials against the database.
/// For now, it demonstrates the JWT token generation flow.
pub async fn login(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<LoginRequest>,
) -> Result<Json<LoginResponse>, GatewayError> {
    tracing::info!(email = %payload.email, "Login attempt");

    let core_url = format!("{}/api/v1/auth/login", state.config.services.core.url);
    let response = state
        .http_client
        .post(core_url)
        .json(&payload)
        .send()
        .await
        .map_err(|err| GatewayError::ServiceUnavailable(err.to_string()))?;

    if !response.status().is_success() {
        let details = response.text().await.unwrap_or_default();
        return Err(GatewayError::UpstreamError {
            service: "core".to_string(),
            message: details,
        });
    }

    let core_response: CoreAuthResponse = response
        .json()
        .await
        .map_err(|err| GatewayError::UpstreamError {
            service: "core".to_string(),
            message: err.to_string(),
        })?;

    Ok(Json(LoginResponse {
        access_token: core_response.tokens.access_token,
        refresh_token: core_response.tokens.refresh_token,
        token_type: core_response.tokens.token_type,
        expires_in: expires_in_from_core(
            &core_response.tokens.expires_at,
            state.config.jwt.access_token_expiry_secs as i64,
        ),
        user: UserInfo {
            id: core_response.user.id,
            email: core_response.user.email,
            username: core_response.user.username,
            role: core_response.user.role,
        },
    }))
}

/// Register handler - POST /api/v1/auth/register
///
/// Note: In production, this would create a user in the database.
pub async fn register(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RegisterRequest>,
) -> Result<Json<LoginResponse>, GatewayError> {
    tracing::info!(email = %payload.email, username = %payload.username, "Registration attempt");

    let core_url = format!("{}/api/v1/auth/register", state.config.services.core.url);
    let response = state
        .http_client
        .post(core_url)
        .json(&payload)
        .send()
        .await
        .map_err(|err| GatewayError::ServiceUnavailable(err.to_string()))?;

    if !response.status().is_success() {
        let details = response.text().await.unwrap_or_default();
        return Err(GatewayError::UpstreamError {
            service: "core".to_string(),
            message: details,
        });
    }

    let core_response: CoreAuthResponse = response
        .json()
        .await
        .map_err(|err| GatewayError::UpstreamError {
            service: "core".to_string(),
            message: err.to_string(),
        })?;

    Ok(Json(LoginResponse {
        access_token: core_response.tokens.access_token,
        refresh_token: core_response.tokens.refresh_token,
        token_type: core_response.tokens.token_type,
        expires_in: expires_in_from_core(
            &core_response.tokens.expires_at,
            state.config.jwt.access_token_expiry_secs as i64,
        ),
        user: UserInfo {
            id: core_response.user.id,
            email: core_response.user.email,
            username: core_response.user.username,
            role: core_response.user.role,
        },
    }))
}

/// Refresh token handler - POST /api/v1/auth/refresh
pub async fn refresh_token(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RefreshTokenRequest>,
) -> Result<Json<crate::auth::TokenPair>, GatewayError> {
    tracing::debug!("Token refresh attempt");

    let core_url = format!("{}/api/v1/auth/refresh", state.config.services.core.url);
    let response = state
        .http_client
        .post(core_url)
        .json(&payload)
        .send()
        .await
        .map_err(|err| GatewayError::ServiceUnavailable(err.to_string()))?;

    if !response.status().is_success() {
        let details = response.text().await.unwrap_or_default();
        return Err(GatewayError::UpstreamError {
            service: "core".to_string(),
            message: details,
        });
    }

    let core_tokens: CoreTokenResponse = response
        .json()
        .await
        .map_err(|err| GatewayError::UpstreamError {
            service: "core".to_string(),
            message: err.to_string(),
        })?;

    Ok(Json(crate::auth::TokenPair {
        access_token: core_tokens.access_token,
        refresh_token: core_tokens.refresh_token,
        token_type: core_tokens.token_type,
        expires_in: expires_in_from_core(
            &core_tokens.expires_at,
            state.config.jwt.access_token_expiry_secs as i64,
        ),
    }))
}

/// Get current user handler - GET /api/v1/users/me
///
/// Returns the authenticated user's information from the JWT token.
pub async fn get_current_user(
    auth_user: AuthUser,
) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "id": auth_user.user_id,
        "email": auth_user.email,
        "role": auth_user.role,
        "token_id": auth_user.token_id,
    }))
}
