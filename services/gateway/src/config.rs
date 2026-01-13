//! Gateway configuration
//!
//! Provides configuration management for the API Gateway including
//! server settings, JWT authentication, rate limiting, and service discovery.

use serde::Deserialize;
use std::env;

/// Main gateway configuration
#[derive(Debug, Clone, Deserialize)]
pub struct GatewayConfig {
    pub server: ServerConfig,
    pub jwt: JwtConfig,
    pub rate_limit: RateLimitConfig,
    pub services: ServicesConfig,
    pub redis: RedisConfig,
}

/// Server configuration
#[derive(Debug, Clone, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    pub request_timeout_secs: u64,
    pub shutdown_timeout_secs: u64,
}

/// JWT authentication configuration
#[derive(Debug, Clone, Deserialize)]
pub struct JwtConfig {
    pub secret: String,
    pub issuer: String,
    pub audience: String,
    pub access_token_expiry_secs: u64,
    pub refresh_token_expiry_secs: u64,
}

/// Rate limiting configuration
#[derive(Debug, Clone, Deserialize)]
pub struct RateLimitConfig {
    pub requests_per_hour: u32,
    pub burst_size: u32,
    pub enabled: bool,
}

/// Backend services configuration
#[derive(Debug, Clone, Deserialize)]
pub struct ServicesConfig {
    pub core: ServiceEndpoint,
    pub realtime: ServiceEndpoint,
    pub container: ServiceEndpoint,
    pub media: ServiceEndpoint,
}

/// Individual service endpoint configuration
#[derive(Debug, Clone, Deserialize)]
pub struct ServiceEndpoint {
    pub url: String,
    pub timeout_secs: u64,
    pub health_check_path: String,
    pub retry_count: u32,
}

/// Redis configuration for distributed rate limiting
#[derive(Debug, Clone, Deserialize)]
pub struct RedisConfig {
    pub url: String,
    pub pool_size: u32,
}

impl Default for GatewayConfig {
    fn default() -> Self {
        Self {
            server: ServerConfig::default(),
            jwt: JwtConfig::default(),
            rate_limit: RateLimitConfig::default(),
            services: ServicesConfig::default(),
            redis: RedisConfig::default(),
        }
    }
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            host: env::var("GATEWAY_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: env::var("GATEWAY_PORT")
                .ok()
                .and_then(|p| p.parse().ok())
                .unwrap_or(8080),
            request_timeout_secs: 30,
            shutdown_timeout_secs: 30,
        }
    }
}

impl Default for JwtConfig {
    fn default() -> Self {
        Self {
            secret: env::var("JWT_SECRET").unwrap_or_else(|_| "change-me-in-production".to_string()),
            issuer: env::var("JWT_ISSUER").unwrap_or_else(|_| "cloud-devbox".to_string()),
            audience: env::var("JWT_AUDIENCE").unwrap_or_else(|_| "cloud-devbox-api".to_string()),
            access_token_expiry_secs: 3600,      // 1 hour
            refresh_token_expiry_secs: 604800,   // 7 days
        }
    }
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            requests_per_hour: 1000,
            burst_size: 100,
            enabled: true,
        }
    }
}

impl Default for ServicesConfig {
    fn default() -> Self {
        Self {
            core: ServiceEndpoint {
                url: env::var("CORE_SERVICE_URL").unwrap_or_else(|_| "http://localhost:8081".to_string()),
                timeout_secs: 30,
                health_check_path: "/health".to_string(),
                retry_count: 3,
            },
            realtime: ServiceEndpoint {
                url: env::var("REALTIME_SERVICE_URL").unwrap_or_else(|_| "http://localhost:8082".to_string()),
                timeout_secs: 30,
                health_check_path: "/health".to_string(),
                retry_count: 3,
            },
            container: ServiceEndpoint {
                url: env::var("CONTAINER_SERVICE_URL").unwrap_or_else(|_| "http://localhost:8083".to_string()),
                timeout_secs: 60,
                health_check_path: "/health".to_string(),
                retry_count: 3,
            },
            media: ServiceEndpoint {
                url: env::var("MEDIA_SERVICE_URL").unwrap_or_else(|_| "http://localhost:8084".to_string()),
                timeout_secs: 30,
                health_check_path: "/health".to_string(),
                retry_count: 3,
            },
        }
    }
}

impl Default for RedisConfig {
    fn default() -> Self {
        Self {
            url: env::var("REDIS_URL").unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            pool_size: 10,
        }
    }
}

impl GatewayConfig {
    /// Load configuration from environment variables
    pub fn from_env() -> Self {
        Self::default()
    }

    /// Get the server socket address
    pub fn socket_addr(&self) -> std::net::SocketAddr {
        std::net::SocketAddr::from((
            self.server.host.parse::<std::net::IpAddr>().unwrap_or(std::net::IpAddr::V4(std::net::Ipv4Addr::new(0, 0, 0, 0))),
            self.server.port,
        ))
    }
}
