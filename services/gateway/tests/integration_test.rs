//! Integration tests for the API Gateway
//!
//! Tests the gateway components working together.

use std::time::Duration;

#[cfg(test)]
mod gateway_integration_tests {
    use super::*;

    /// Test JWT token generation and validation round-trip
    #[test]
    fn test_jwt_round_trip() {
        use devbox_gateway::{JwtService, GatewayConfig};
        
        let config = GatewayConfig::default();
        let jwt_service = JwtService::new(config.jwt);
        
        // Generate token
        let user_id = "test-user-123";
        let email = "test@example.com";
        let role = "developer";
        
        let token = jwt_service
            .generate_access_token(user_id, email, role)
            .expect("Failed to generate token");
        
        // Validate token
        let claims = jwt_service
            .validate_access_token(&token)
            .expect("Failed to validate token");
        
        assert_eq!(claims.sub, user_id);
        assert_eq!(claims.email, email);
        assert_eq!(claims.role, role);
    }

    /// Test token pair generation
    #[test]
    fn test_token_pair_generation() {
        use devbox_gateway::{JwtService, GatewayConfig};
        
        let config = GatewayConfig::default();
        let jwt_service = JwtService::new(config.jwt);
        
        let pair = jwt_service
            .generate_token_pair("user-123", "test@example.com", "developer")
            .expect("Failed to generate token pair");
        
        assert!(!pair.access_token.is_empty());
        assert!(!pair.refresh_token.is_empty());
        assert_eq!(pair.token_type, "Bearer");
        assert!(pair.expires_in > 0);
    }

    /// Test token refresh flow
    #[test]
    fn test_token_refresh() {
        use devbox_gateway::{JwtService, GatewayConfig};
        
        let config = GatewayConfig::default();
        let jwt_service = JwtService::new(config.jwt);
        
        // Generate initial tokens
        let pair = jwt_service
            .generate_token_pair("user-123", "test@example.com", "developer")
            .expect("Failed to generate token pair");
        
        // Refresh tokens
        let new_pair = jwt_service
            .refresh_tokens(&pair.refresh_token)
            .expect("Failed to refresh tokens");
        
        // New tokens should be different
        assert_ne!(pair.access_token, new_pair.access_token);
        assert_ne!(pair.refresh_token, new_pair.refresh_token);
    }

    /// Test service registry initialization
    #[tokio::test]
    async fn test_service_registry() {
        use devbox_gateway::{ServiceRegistry, LoadBalanceStrategy, ServiceInstance};
        
        let registry = ServiceRegistry::new(LoadBalanceStrategy::RoundRobin);
        
        // Register a test service
        registry.register("test-service", ServiceInstance {
            id: "test-1".to_string(),
            url: "http://localhost:8081".to_string(),
            healthy: true,
            last_check: None,
            failure_count: 0,
            active_connections: 0,
            avg_response_time_ms: 0.0,
        }).await;
        
        // Get instance
        let instance = registry.get_instance("test-service").await;
        assert!(instance.is_ok());
        
        let instance = instance.unwrap();
        assert_eq!(instance.id, "test-1");
        assert!(instance.healthy);
    }

    /// Test load balancing round-robin
    #[tokio::test]
    async fn test_round_robin_load_balancing() {
        use devbox_gateway::{ServiceRegistry, LoadBalanceStrategy, ServiceInstance};
        
        let registry = ServiceRegistry::new(LoadBalanceStrategy::RoundRobin);
        
        // Register multiple instances
        for i in 1..=3 {
            registry.register("multi-service", ServiceInstance {
                id: format!("instance-{}", i),
                url: format!("http://localhost:808{}", i),
                healthy: true,
                last_check: None,
                failure_count: 0,
                active_connections: 0,
                avg_response_time_ms: 0.0,
            }).await;
        }
        
        // Get instances multiple times - should round-robin
        let mut ids = Vec::new();
        for _ in 0..6 {
            let instance = registry.get_instance("multi-service").await.unwrap();
            ids.push(instance.id.clone());
        }
        
        // Should cycle through all instances
        assert!(ids.contains(&"instance-1".to_string()));
        assert!(ids.contains(&"instance-2".to_string()));
        assert!(ids.contains(&"instance-3".to_string()));
    }

    /// Test circuit breaker opens after failures
    #[test]
    fn test_circuit_breaker_opens() {
        use devbox_gateway::CircuitBreaker;
        
        let mut cb = CircuitBreaker::default();
        
        // Record failures up to threshold
        for _ in 0..5 {
            cb.record_failure();
        }
        
        // Circuit should be open now
        assert!(!cb.allow_request());
    }
}
