//! Service discovery and load balancing module
//!
//! Provides service discovery with health checking and load balancing
//! across multiple service instances.

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

use crate::config::ServiceEndpoint;
use crate::error::GatewayError;

/// Service instance information
#[derive(Debug, Clone)]
pub struct ServiceInstance {
    /// Instance ID
    pub id: String,
    /// Service URL
    pub url: String,
    /// Health status
    pub healthy: bool,
    /// Last health check time
    pub last_check: Option<Instant>,
    /// Consecutive failure count
    pub failure_count: u32,
    /// Current active connections (for load balancing)
    pub active_connections: u32,
    /// Response time moving average (ms)
    pub avg_response_time_ms: f64,
}

/// Service health status
#[derive(Debug, Clone, PartialEq)]
pub enum HealthStatus {
    Healthy,
    Unhealthy,
    Unknown,
}


/// Load balancing strategy
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum LoadBalanceStrategy {
    /// Round-robin selection
    RoundRobin,
    /// Least connections
    LeastConnections,
    /// Weighted response time
    WeightedResponseTime,
    /// Random selection
    Random,
}

/// Circuit breaker state
#[derive(Debug, Clone, PartialEq)]
pub enum CircuitState {
    /// Circuit is closed, requests flow normally
    Closed,
    /// Circuit is open, requests are rejected
    Open,
    /// Circuit is half-open, testing if service recovered
    HalfOpen,
}

/// Circuit breaker for a service
#[derive(Debug, Clone)]
pub struct CircuitBreaker {
    pub state: CircuitState,
    pub failure_count: u32,
    pub success_count: u32,
    pub last_failure: Option<Instant>,
    pub failure_threshold: u32,
    pub success_threshold: u32,
    pub timeout: Duration,
}

impl Default for CircuitBreaker {
    fn default() -> Self {
        Self {
            state: CircuitState::Closed,
            failure_count: 0,
            success_count: 0,
            last_failure: None,
            failure_threshold: 5,
            success_threshold: 3,
            timeout: Duration::from_secs(30),
        }
    }
}


impl CircuitBreaker {
    /// Check if the circuit allows requests
    pub fn allow_request(&mut self) -> bool {
        match self.state {
            CircuitState::Closed => true,
            CircuitState::Open => {
                // Check if timeout has passed
                if let Some(last_failure) = self.last_failure {
                    if last_failure.elapsed() >= self.timeout {
                        self.state = CircuitState::HalfOpen;
                        self.success_count = 0;
                        true
                    } else {
                        false
                    }
                } else {
                    true
                }
            }
            CircuitState::HalfOpen => true,
        }
    }

    /// Record a successful request
    pub fn record_success(&mut self) {
        match self.state {
            CircuitState::Closed => {
                self.failure_count = 0;
            }
            CircuitState::HalfOpen => {
                self.success_count += 1;
                if self.success_count >= self.success_threshold {
                    self.state = CircuitState::Closed;
                    self.failure_count = 0;
                    self.success_count = 0;
                }
            }
            CircuitState::Open => {}
        }
    }

    /// Record a failed request
    pub fn record_failure(&mut self) {
        self.failure_count += 1;
        self.last_failure = Some(Instant::now());

        match self.state {
            CircuitState::Closed => {
                if self.failure_count >= self.failure_threshold {
                    self.state = CircuitState::Open;
                    tracing::warn!("Circuit breaker opened after {} failures", self.failure_count);
                }
            }
            CircuitState::HalfOpen => {
                self.state = CircuitState::Open;
                tracing::warn!("Circuit breaker re-opened after failure in half-open state");
            }
            CircuitState::Open => {}
        }
    }
}


/// Service registry for managing service instances
pub struct ServiceRegistry {
    /// Map of service name to instances
    services: RwLock<HashMap<String, Vec<ServiceInstance>>>,
    /// Circuit breakers per service
    circuit_breakers: RwLock<HashMap<String, CircuitBreaker>>,
    /// Load balancing strategy
    strategy: LoadBalanceStrategy,
    /// Round-robin counter per service
    rr_counters: RwLock<HashMap<String, usize>>,
    /// HTTP client for health checks
    http_client: reqwest::Client,
    /// Health check interval
    health_check_interval: Duration,
}

impl ServiceRegistry {
    /// Create a new service registry
    pub fn new(strategy: LoadBalanceStrategy) -> Self {
        let http_client = reqwest::Client::builder()
            .timeout(Duration::from_secs(5))
            .build()
            .expect("Failed to create HTTP client");

        Self {
            services: RwLock::new(HashMap::new()),
            circuit_breakers: RwLock::new(HashMap::new()),
            strategy,
            rr_counters: RwLock::new(HashMap::new()),
            http_client,
            health_check_interval: Duration::from_secs(10),
        }
    }

    /// Register a service instance
    pub async fn register(&self, service_name: &str, instance: ServiceInstance) {
        let mut services = self.services.write().await;
        services
            .entry(service_name.to_string())
            .or_insert_with(Vec::new)
            .push(instance);

        // Initialize circuit breaker if not exists
        let mut breakers = self.circuit_breakers.write().await;
        breakers
            .entry(service_name.to_string())
            .or_insert_with(CircuitBreaker::default);
    }


    /// Register services from configuration
    pub async fn register_from_config(&self, services_config: &crate::config::ServicesConfig) {
        // Register core service
        self.register("core", ServiceInstance {
            id: "core-1".to_string(),
            url: services_config.core.url.clone(),
            healthy: true,
            last_check: None,
            failure_count: 0,
            active_connections: 0,
            avg_response_time_ms: 0.0,
        }).await;

        // Register realtime service
        self.register("realtime", ServiceInstance {
            id: "realtime-1".to_string(),
            url: services_config.realtime.url.clone(),
            healthy: true,
            last_check: None,
            failure_count: 0,
            active_connections: 0,
            avg_response_time_ms: 0.0,
        }).await;

        // Register container service
        self.register("container", ServiceInstance {
            id: "container-1".to_string(),
            url: services_config.container.url.clone(),
            healthy: true,
            last_check: None,
            failure_count: 0,
            active_connections: 0,
            avg_response_time_ms: 0.0,
        }).await;

        // Register media service
        self.register("media", ServiceInstance {
            id: "media-1".to_string(),
            url: services_config.media.url.clone(),
            healthy: true,
            last_check: None,
            failure_count: 0,
            active_connections: 0,
            avg_response_time_ms: 0.0,
        }).await;
    }


    /// Get a healthy instance for a service using load balancing
    pub async fn get_instance(&self, service_name: &str) -> Result<ServiceInstance, GatewayError> {
        // Check circuit breaker
        {
            let mut breakers = self.circuit_breakers.write().await;
            if let Some(breaker) = breakers.get_mut(service_name) {
                if !breaker.allow_request() {
                    return Err(GatewayError::ServiceUnavailable(
                        format!("Service {} circuit breaker is open", service_name)
                    ));
                }
            }
        }

        let services = self.services.read().await;
        let instances = services.get(service_name).ok_or_else(|| {
            GatewayError::NotFound(format!("Service {} not found", service_name))
        })?;

        // Filter healthy instances
        let healthy: Vec<_> = instances.iter().filter(|i| i.healthy).collect();
        
        if healthy.is_empty() {
            return Err(GatewayError::ServiceUnavailable(
                format!("No healthy instances for service {}", service_name)
            ));
        }

        // Select instance based on strategy
        let instance = match self.strategy {
            LoadBalanceStrategy::RoundRobin => {
                self.select_round_robin(service_name, &healthy).await
            }
            LoadBalanceStrategy::LeastConnections => {
                self.select_least_connections(&healthy)
            }
            LoadBalanceStrategy::WeightedResponseTime => {
                self.select_weighted_response_time(&healthy)
            }
            LoadBalanceStrategy::Random => {
                self.select_random(&healthy)
            }
        };

        Ok(instance.clone())
    }


    /// Round-robin selection
    async fn select_round_robin<'a>(
        &self,
        service_name: &str,
        instances: &[&'a ServiceInstance],
    ) -> &'a ServiceInstance {
        let mut counters = self.rr_counters.write().await;
        let counter = counters.entry(service_name.to_string()).or_insert(0);
        let index = *counter % instances.len();
        *counter = counter.wrapping_add(1);
        instances[index]
    }

    /// Least connections selection
    fn select_least_connections<'a>(&self, instances: &[&'a ServiceInstance]) -> &'a ServiceInstance {
        instances
            .iter()
            .min_by_key(|i| i.active_connections)
            .copied()
            .unwrap()
    }

    /// Weighted response time selection
    fn select_weighted_response_time<'a>(&self, instances: &[&'a ServiceInstance]) -> &'a ServiceInstance {
        instances
            .iter()
            .min_by(|a, b| {
                a.avg_response_time_ms
                    .partial_cmp(&b.avg_response_time_ms)
                    .unwrap_or(std::cmp::Ordering::Equal)
            })
            .copied()
            .unwrap()
    }

    /// Random selection
    fn select_random<'a>(&self, instances: &[&'a ServiceInstance]) -> &'a ServiceInstance {
        use std::time::SystemTime;
        let seed = SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos() as usize;
        instances[seed % instances.len()]
    }


    /// Record a successful request to a service
    pub async fn record_success(&self, service_name: &str, response_time_ms: u64) {
        // Update circuit breaker
        {
            let mut breakers = self.circuit_breakers.write().await;
            if let Some(breaker) = breakers.get_mut(service_name) {
                breaker.record_success();
            }
        }

        // Update instance metrics (simplified - in production, track per instance)
        let mut services = self.services.write().await;
        if let Some(instances) = services.get_mut(service_name) {
            for instance in instances.iter_mut() {
                // Exponential moving average for response time
                instance.avg_response_time_ms = 
                    instance.avg_response_time_ms * 0.9 + response_time_ms as f64 * 0.1;
            }
        }
    }

    /// Record a failed request to a service
    pub async fn record_failure(&self, service_name: &str) {
        let mut breakers = self.circuit_breakers.write().await;
        if let Some(breaker) = breakers.get_mut(service_name) {
            breaker.record_failure();
        }
    }

    /// Perform health check on all services
    pub async fn health_check_all(&self, health_paths: &HashMap<String, String>) {
        let services = self.services.read().await;
        
        for (service_name, instances) in services.iter() {
            let health_path = health_paths.get(service_name).map(|s| s.as_str()).unwrap_or("/health");
            
            for instance in instances {
                let url = format!("{}{}", instance.url, health_path);
                let healthy = self.check_instance_health(&url).await;
                
                // Update instance health (need write lock)
                drop(services);
                self.update_instance_health(service_name, &instance.id, healthy).await;
                return; // Re-acquire read lock in next iteration
            }
        }
    }


    /// Check health of a single instance
    async fn check_instance_health(&self, url: &str) -> bool {
        match self.http_client.get(url).send().await {
            Ok(response) => response.status().is_success(),
            Err(_) => false,
        }
    }

    /// Update instance health status
    async fn update_instance_health(&self, service_name: &str, instance_id: &str, healthy: bool) {
        let mut services = self.services.write().await;
        if let Some(instances) = services.get_mut(service_name) {
            for instance in instances.iter_mut() {
                if instance.id == instance_id {
                    instance.healthy = healthy;
                    instance.last_check = Some(Instant::now());
                    if !healthy {
                        instance.failure_count += 1;
                    } else {
                        instance.failure_count = 0;
                    }
                    break;
                }
            }
        }
    }

    /// Get circuit breaker state for a service
    pub async fn get_circuit_state(&self, service_name: &str) -> Option<CircuitState> {
        let breakers = self.circuit_breakers.read().await;
        breakers.get(service_name).map(|b| b.state.clone())
    }

    /// Get all service statuses
    pub async fn get_all_statuses(&self) -> HashMap<String, Vec<ServiceInstance>> {
        self.services.read().await.clone()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_circuit_breaker_closed() {
        let mut cb = CircuitBreaker::default();
        assert!(cb.allow_request());
        assert_eq!(cb.state, CircuitState::Closed);
    }

    #[test]
    fn test_circuit_breaker_opens_after_failures() {
        let mut cb = CircuitBreaker {
            failure_threshold: 3,
            ..Default::default()
        };

        cb.record_failure();
        cb.record_failure();
        assert_eq!(cb.state, CircuitState::Closed);
        
        cb.record_failure();
        assert_eq!(cb.state, CircuitState::Open);
    }

    #[test]
    fn test_circuit_breaker_success_resets_failures() {
        let mut cb = CircuitBreaker::default();
        cb.record_failure();
        cb.record_failure();
        cb.record_success();
        assert_eq!(cb.failure_count, 0);
    }
}
