//! Error types for Cloud DevBox

use thiserror::Error;

/// Common error type for Cloud DevBox
#[derive(Error, Debug)]
pub enum Error {
    #[error("Authentication error: {0}")]
    Auth(String),

    #[error("Authorization error: {0}")]
    Forbidden(String),

    #[error("Resource not found: {0}")]
    NotFound(String),

    #[error("Validation error: {0}")]
    Validation(String),

    #[error("Internal error: {0}")]
    Internal(String),

    #[error("Database error: {0}")]
    Database(String),

    #[error("External service error: {0}")]
    ExternalService(String),

    #[error("Rate limit exceeded")]
    RateLimitExceeded,

    #[error("Resource quota exceeded: {0}")]
    QuotaExceeded(String),
}

/// Result type alias using our Error type
pub type Result<T> = std::result::Result<T, Error>;
