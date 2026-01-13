//! Application state management
//!
//! Manages shared state including configuration, Redis connection,
//! HTTP client, metrics, and service registry.

use axum::{
    http::{HeaderMap, Method, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use redis::aio::ConnectionManager;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::RwLock;

use crate::config::GatewayConfig;
use crate::discovery::{LoadBalanceStrategy, ServiceRegistry};
use crate::error::GatewayError;
use crate::handlers::{ServiceHealth, ServicesStatus};

/// Application metrics
#[derive(Debug, Default)]
pub struct Metrics {
    pub total_requests: AtomicU64,
    pub active_requests: AtomicU64,
    pub total_duration_secs: AtomicU64,
    pub error_count: AtomicU64,
}

/// Collected metrics snapshot
#[derive(Debug, Clone)]
pub struct MetricsSnapshot {
    pub total_requests: u64,
    pub active_requests: u64,
    pub total_duration_secs: f64,
    pub error_count: u64,
}

/// Application state shared across handlers
pub struct AppState {
    pub config: GatewayConfig,
    pub http_client: reqwest::Client,
    pub redis: RwLock<Option<ConnectionManager>>,
    pub service_registry: Arc<ServiceRegistry>,
    pub start_time: Instant,
    pub metrics: Arc<Metrics>,
}

impl AppState {
    /// Create new application state
    pub async fn new(config: GatewayConfig) -> anyhow::Result<Self> {
        // Create HTTP client with timeout
        let http_client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(config.server.request_timeout_secs))
            .pool_max_idle_per_host(10)
            .build()?;

        // Try to connect to Redis
        let redis = match redis::Client::open(config.redis.url.as_str()) {
            Ok(client) => {
                match ConnectionManager::new(client).await {
                    Ok(conn) => {
                        tracing::info!("Connected to Redis");
                        Some(conn)
                    }
                    Err(e) => {
                        tracing::warn!("Failed to connect to Redis: {}. Rate limiting will be disabled.", e);
                        None
                    }
                }
            }
            Err(e) => {
                tracing::warn!("Invalid Redis URL: {}. Rate limiting will be disabled.", e);
                None
            }
        };

        // Initialize service registry with round-robin load balancing
        let service_registry = Arc::new(ServiceRegistry::new(LoadBalanceStrategy::RoundRobin));
        service_registry.register_from_config(&config.services).await;
        tracing::info!("Service registry initialized");

        Ok(Self {
            config,
            http_client,
            redis: RwLock::new(redis),
            service_registry,
            start_time: Instant::now(),
            metrics: Arc::new(Metrics::default()),
        })
    }

    /// Get uptime in seconds
    pub fn uptime_secs(&self) -> u64 {
        self.start_time.elapsed().as_secs()
    }

    /// Check Redis connection
    pub async fn check_redis_connection(&self) -> bool {
        let redis = self.redis.read().await;
        if let Some(ref conn) = *redis {
            let mut conn = conn.clone();
            redis::cmd("PING")
                .query_async::<_, String>(&mut conn)
                .await
                .is_ok()
        } else {
            false
        }
    }

    /// Get Redis connection manager
    pub async fn get_redis(&self) -> Option<ConnectionManager> {
        self.redis.read().await.clone()
    }

    /// Check health of all backend services
    pub async fn check_services_health(&self) -> ServicesStatus {
        let (core, realtime, container, media) = tokio::join!(
            self.check_service_health("core", &self.config.services.core.url, &self.config.services.core.health_check_path),
            self.check_service_health("realtime", &self.config.services.realtime.url, &self.config.services.realtime.health_check_path),
            self.check_service_health("container", &self.config.services.container.url, &self.config.services.container.health_check_path),
            self.check_service_health("media", &self.config.services.media.url, &self.config.services.media.health_check_path),
        );

        ServicesStatus {
            core,
            realtime,
            container,
            media,
        }
    }

    /// Check health of a single service
    async fn check_service_health(&self, name: &str, base_url: &str, health_path: &str) -> ServiceHealth {
        let url = format!("{}{}", base_url, health_path);
        let start = Instant::now();

        match self.http_client.get(&url).send().await {
            Ok(response) if response.status().is_success() => {
                ServiceHealth {
                    status: "healthy".to_string(),
                    latency_ms: Some(start.elapsed().as_millis() as u64),
                    last_check: Some(chrono::Utc::now().to_rfc3339()),
                }
            }
            Ok(response) => {
                tracing::warn!(
                    service = name,
                    status = %response.status(),
                    "Service health check returned non-success status"
                );
                ServiceHealth {
                    status: "unhealthy".to_string(),
                    latency_ms: Some(start.elapsed().as_millis() as u64),
                    last_check: Some(chrono::Utc::now().to_rfc3339()),
                }
            }
            Err(e) => {
                tracing::warn!(
                    service = name,
                    error = %e,
                    "Service health check failed"
                );
                ServiceHealth {
                    status: "unavailable".to_string(),
                    latency_ms: None,
                    last_check: Some(chrono::Utc::now().to_rfc3339()),
                }
            }
        }
    }

    /// Proxy request to backend service using service registry
    pub async fn proxy_request(
        &self,
        service: &str,
        path: &str,
        method: Method,
        headers: HeaderMap,
        body: Option<String>,
    ) -> Result<Response, GatewayError> {
        // Get service instance from registry (with load balancing)
        let instance = self.service_registry.get_instance(service).await?;
        let url = format!("{}/{}", instance.url, path);
        
        let mut request = self.http_client.request(method.clone(), &url);

        // Forward relevant headers
        for (key, value) in headers.iter() {
            if !is_hop_by_hop_header(key.as_str()) {
                request = request.header(key.clone(), value.clone());
            }
        }

        // Add body if present
        if let Some(body) = body {
            request = request.body(body);
        }

        let start = Instant::now();
        let result = request.send().await;
        let latency = start.elapsed();

        match result {
            Ok(response) => {
                // Record success with response time
                self.service_registry.record_success(service, latency.as_millis() as u64).await;

                tracing::debug!(
                    service = service,
                    path = path,
                    status = %response.status(),
                    latency_ms = latency.as_millis() as u64,
                    "Proxy response received"
                );

                // Convert reqwest response to axum response
                let status = StatusCode::from_u16(response.status().as_u16())
                    .unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
                
                let response_headers = response.headers().clone();
                let body = response.bytes().await.map_err(|e| {
                    GatewayError::UpstreamError {
                        service: service.to_string(),
                        message: format!("Failed to read response body: {}", e),
                    }
                })?;

                let mut builder = axum::response::Response::builder().status(status);
                
                for (key, value) in response_headers.iter() {
                    if !is_hop_by_hop_header(key.as_str()) {
                        builder = builder.header(key.clone(), value.clone());
                    }
                }

                Ok(builder
                    .body(axum::body::Body::from(body))
                    .unwrap_or_else(|_| {
                        (StatusCode::INTERNAL_SERVER_ERROR, "Failed to build response").into_response()
                    }))
            }
            Err(e) => {
                // Record failure for circuit breaker
                self.service_registry.record_failure(service).await;

                if e.is_timeout() {
                    Err(GatewayError::Timeout(format!("Request to {} timed out", service)))
                } else if e.is_connect() {
                    Err(GatewayError::ServiceUnavailable(format!("Cannot connect to {}", service)))
                } else {
                    Err(GatewayError::UpstreamError {
                        service: service.to_string(),
                        message: e.to_string(),
                    })
                }
            }
        }
    }

    /// Get metrics snapshot
    pub fn get_metrics(&self) -> MetricsSnapshot {
        MetricsSnapshot {
            total_requests: self.metrics.total_requests.load(Ordering::Relaxed),
            active_requests: self.metrics.active_requests.load(Ordering::Relaxed),
            total_duration_secs: self.metrics.total_duration_secs.load(Ordering::Relaxed) as f64 / 1000.0,
            error_count: self.metrics.error_count.load(Ordering::Relaxed),
        }
    }

    /// Increment request counter
    pub fn increment_requests(&self) {
        self.metrics.total_requests.fetch_add(1, Ordering::Relaxed);
        self.metrics.active_requests.fetch_add(1, Ordering::Relaxed);
    }

    /// Decrement active requests and record duration
    pub fn complete_request(&self, duration_ms: u64) {
        self.metrics.active_requests.fetch_sub(1, Ordering::Relaxed);
        self.metrics.total_duration_secs.fetch_add(duration_ms, Ordering::Relaxed);
    }

    /// Increment error counter
    pub fn increment_errors(&self) {
        self.metrics.error_count.fetch_add(1, Ordering::Relaxed);
    }
}

/// Check if a header is a hop-by-hop header that shouldn't be forwarded
fn is_hop_by_hop_header(name: &str) -> bool {
    matches!(
        name.to_lowercase().as_str(),
        "connection"
            | "keep-alive"
            | "proxy-authenticate"
            | "proxy-authorization"
            | "te"
            | "trailers"
            | "transfer-encoding"
            | "upgrade"
            | "host"
    )
}
