//! Error types for the API Gateway
//!
//! Provides unified error handling with proper HTTP status codes
//! and JSON error responses.

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::Serialize;
use std::fmt;

/// API error response structure
#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub error: ErrorDetail,
}

/// Error detail structure
#[derive(Debug, Serialize)]
pub struct ErrorDetail {
    pub code: String,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<serde_json::Value>,
}

/// Gateway error types
#[derive(Debug)]
pub enum GatewayError {
    /// Authentication errors
    Unauthorized(String),
    /// Token expired
    TokenExpired,
    /// Invalid token
    InvalidToken(String),
    /// Forbidden access
    Forbidden(String),
    /// Rate limit exceeded
    RateLimitExceeded { retry_after: u64 },
    /// Service unavailable
    ServiceUnavailable(String),
    /// Bad request
    BadRequest(String),
    /// Not found
    NotFound(String),
    /// Internal server error
    Internal(String),
    /// Timeout error
    Timeout(String),
    /// Upstream service error
    UpstreamError { service: String, message: String },
}

impl fmt::Display for GatewayError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            GatewayError::Unauthorized(msg) => write!(f, "Unauthorized: {}", msg),
            GatewayError::TokenExpired => write!(f, "Token has expired"),
            GatewayError::InvalidToken(msg) => write!(f, "Invalid token: {}", msg),
            GatewayError::Forbidden(msg) => write!(f, "Forbidden: {}", msg),
            GatewayError::RateLimitExceeded { retry_after } => {
                write!(f, "Rate limit exceeded. Retry after {} seconds", retry_after)
            }
            GatewayError::ServiceUnavailable(msg) => write!(f, "Service unavailable: {}", msg),
            GatewayError::BadRequest(msg) => write!(f, "Bad request: {}", msg),
            GatewayError::NotFound(msg) => write!(f, "Not found: {}", msg),
            GatewayError::Internal(msg) => write!(f, "Internal error: {}", msg),
            GatewayError::Timeout(msg) => write!(f, "Timeout: {}", msg),
            GatewayError::UpstreamError { service, message } => {
                write!(f, "Upstream error from {}: {}", service, message)
            }
        }
    }
}

impl std::error::Error for GatewayError {}

impl IntoResponse for GatewayError {
    fn into_response(self) -> Response {
        let (status, code, message) = match &self {
            GatewayError::Unauthorized(msg) => {
                (StatusCode::UNAUTHORIZED, "UNAUTHORIZED", msg.clone())
            }
            GatewayError::TokenExpired => {
                (StatusCode::UNAUTHORIZED, "TOKEN_EXPIRED", "Token has expired".to_string())
            }
            GatewayError::InvalidToken(msg) => {
                (StatusCode::UNAUTHORIZED, "INVALID_TOKEN", msg.clone())
            }
            GatewayError::Forbidden(msg) => {
                (StatusCode::FORBIDDEN, "FORBIDDEN", msg.clone())
            }
            GatewayError::RateLimitExceeded { retry_after } => {
                (
                    StatusCode::TOO_MANY_REQUESTS,
                    "RATE_LIMIT_EXCEEDED",
                    format!("Rate limit exceeded. Retry after {} seconds", retry_after),
                )
            }
            GatewayError::ServiceUnavailable(msg) => {
                (StatusCode::SERVICE_UNAVAILABLE, "SERVICE_UNAVAILABLE", msg.clone())
            }
            GatewayError::BadRequest(msg) => {
                (StatusCode::BAD_REQUEST, "BAD_REQUEST", msg.clone())
            }
            GatewayError::NotFound(msg) => {
                (StatusCode::NOT_FOUND, "NOT_FOUND", msg.clone())
            }
            GatewayError::Internal(msg) => {
                (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL_ERROR", msg.clone())
            }
            GatewayError::Timeout(msg) => {
                (StatusCode::GATEWAY_TIMEOUT, "TIMEOUT", msg.clone())
            }
            GatewayError::UpstreamError { service, message } => {
                (
                    StatusCode::BAD_GATEWAY,
                    "UPSTREAM_ERROR",
                    format!("Error from {}: {}", service, message),
                )
            }
        };

        let body = Json(ErrorResponse {
            error: ErrorDetail {
                code: code.to_string(),
                message,
                details: None,
            },
        });

        (status, body).into_response()
    }
}

/// Result type alias for gateway operations
pub type GatewayResult<T> = Result<T, GatewayError>;
