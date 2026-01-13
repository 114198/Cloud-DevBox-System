//! Cloud DevBox API Gateway Library
//!
//! This library provides the core components for the API Gateway:
//! - JWT authentication
//! - Rate limiting with Redis
//! - Service discovery and load balancing
//! - Circuit breaker pattern

pub mod auth;
pub mod config;
pub mod discovery;
pub mod error;
pub mod handlers;
pub mod middleware;
pub mod rate_limit;
pub mod router;
pub mod state;

pub use auth::{AuthUser, Claims, JwtService, TokenPair, TokenType};
pub use config::GatewayConfig;
pub use discovery::{
    CircuitBreaker, CircuitState, LoadBalanceStrategy, ServiceInstance, ServiceRegistry,
};
pub use error::{GatewayError, GatewayResult};
pub use rate_limit::{RateLimitInfo, TokenBucketRateLimiter};
pub use state::AppState;
