//! Rate limiting module using token bucket algorithm
//!
//! Implements distributed rate limiting with Redis backend.
//! Supports per-user rate limits with configurable requests per hour.

use axum::{
    body::Body,
    extract::State,
    http::Request,
    middleware::Next,
    response::Response,
};
use redis::AsyncCommands;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use crate::auth::extract_bearer_token;
use crate::config::RateLimitConfig;
use crate::error::GatewayError;
use crate::state::AppState;

/// Rate limit info returned in response headers
#[derive(Debug, Clone)]
pub struct RateLimitInfo {
    /// Maximum requests allowed per hour
    pub limit: u32,
    /// Remaining requests in current window
    pub remaining: u32,
    /// Unix timestamp when the rate limit resets
    pub reset: u64,
    /// Seconds until retry is allowed (only set when rate limited)
    pub retry_after: Option<u64>,
}

/// Token bucket rate limiter
pub struct TokenBucketRateLimiter {
    config: RateLimitConfig,
}

impl TokenBucketRateLimiter {
    /// Create a new rate limiter
    pub fn new(config: RateLimitConfig) -> Self {
        Self { config }
    }

    /// Check rate limit for a user using Redis
    ///
    /// Uses a sliding window token bucket algorithm:
    /// - Each user has a bucket that refills at a constant rate
    /// - Requests consume tokens from the bucket
    /// - If the bucket is empty, the request is rate limited
    pub async fn check_rate_limit(
        &self,
        redis: &mut redis::aio::ConnectionManager,
        user_id: &str,
    ) -> Result<RateLimitInfo, GatewayError> {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let key = format!("rate_limit:{}", user_id);
        let window_secs: u64 = 3600; // 1 hour window
        let max_requests = self.config.requests_per_hour;
        let window_start = now - (now % window_secs);
        let window_end = window_start + window_secs;

        // Use Redis transaction for atomic operations
        let result: Result<(u32, u64), redis::RedisError> = redis::pipe()
            .atomic()
            // Increment the counter
            .incr(&key, 1i32)
            // Set expiry if this is a new key
            .expire(&key, window_secs as i64)
            .ignore()
            // Get TTL to calculate reset time
            .ttl(&key)
            .query_async(redis)
            .await;

        match result {
            Ok((count, ttl)) => {
                let reset = now + ttl;
                
                if count > max_requests {
                    // Rate limit exceeded
                    let retry_after = if ttl > 0 { ttl } else { window_secs };
                    
                    Ok(RateLimitInfo {
                        limit: max_requests,
                        remaining: 0,
                        reset,
                        retry_after: Some(retry_after),
                    })
                } else {
                    // Request allowed
                    Ok(RateLimitInfo {
                        limit: max_requests,
                        remaining: max_requests.saturating_sub(count),
                        reset,
                        retry_after: None,
                    })
                }
            }
            Err(e) => {
                tracing::error!(error = %e, "Redis rate limit check failed");
                // On Redis error, allow the request but log the error
                // This prevents Redis failures from blocking all requests
                Ok(RateLimitInfo {
                    limit: max_requests,
                    remaining: max_requests,
                    reset: window_end,
                    retry_after: None,
                })
            }
        }
    }

    /// Check rate limit using Lua script for better atomicity
    ///
    /// This is more efficient than multiple Redis commands
    pub async fn check_rate_limit_lua(
        &self,
        redis: &mut redis::aio::ConnectionManager,
        user_id: &str,
    ) -> Result<RateLimitInfo, GatewayError> {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let key = format!("rate_limit:{}", user_id);
        let window_secs: u64 = 3600;
        let max_requests = self.config.requests_per_hour as i64;

        // Lua script for atomic rate limiting
        let script = redis::Script::new(
            r#"
            local key = KEYS[1]
            local max_requests = tonumber(ARGV[1])
            local window_secs = tonumber(ARGV[2])
            local now = tonumber(ARGV[3])
            
            -- Get current count
            local current = redis.call('GET', key)
            if current == false then
                current = 0
            else
                current = tonumber(current)
            end
            
            -- Check if rate limited
            if current >= max_requests then
                local ttl = redis.call('TTL', key)
                if ttl < 0 then ttl = window_secs end
                return {current, ttl, 1}  -- 1 = rate limited
            end
            
            -- Increment counter
            local new_count = redis.call('INCR', key)
            if new_count == 1 then
                redis.call('EXPIRE', key, window_secs)
            end
            
            local ttl = redis.call('TTL', key)
            if ttl < 0 then ttl = window_secs end
            
            return {new_count, ttl, 0}  -- 0 = allowed
            "#,
        );

        let result: Result<(i64, i64, i64), redis::RedisError> = script
            .key(&key)
            .arg(max_requests)
            .arg(window_secs as i64)
            .arg(now as i64)
            .invoke_async(redis)
            .await;

        match result {
            Ok((count, ttl, is_limited)) => {
                let reset = now + ttl as u64;
                let remaining = (max_requests - count).max(0) as u32;

                if is_limited == 1 {
                    Ok(RateLimitInfo {
                        limit: self.config.requests_per_hour,
                        remaining: 0,
                        reset,
                        retry_after: Some(ttl as u64),
                    })
                } else {
                    Ok(RateLimitInfo {
                        limit: self.config.requests_per_hour,
                        remaining,
                        reset,
                        retry_after: None,
                    })
                }
            }
            Err(e) => {
                tracing::error!(error = %e, "Redis Lua script failed");
                // Fallback: allow request on Redis error
                Ok(RateLimitInfo {
                    limit: self.config.requests_per_hour,
                    remaining: self.config.requests_per_hour,
                    reset: now + window_secs,
                    retry_after: None,
                })
            }
        }
    }
}

/// Rate limiting middleware
///
/// Applies rate limiting based on user ID (from JWT) or IP address.
/// Returns 429 Too Many Requests when rate limit is exceeded.
pub async fn rate_limit_middleware(
    State(state): State<Arc<AppState>>,
    request: Request<Body>,
    next: Next,
) -> Result<Response, GatewayError> {
    // Skip rate limiting if disabled
    if !state.config.rate_limit.enabled {
        return Ok(next.run(request).await);
    }

    // Get user identifier (user ID from JWT or IP address)
    let user_id = get_rate_limit_key(&request);

    // Check if Redis is available
    let redis = state.get_redis().await;
    
    if let Some(mut redis) = redis {
        let rate_limiter = TokenBucketRateLimiter::new(state.config.rate_limit.clone());
        let rate_info = rate_limiter.check_rate_limit_lua(&mut redis, &user_id).await?;

        // Check if rate limited
        if let Some(retry_after) = rate_info.retry_after {
            tracing::warn!(
                user_id = %user_id,
                retry_after = retry_after,
                "Rate limit exceeded"
            );
            return Err(GatewayError::RateLimitExceeded { retry_after });
        }

        // Process request and add rate limit headers to response
        let mut response = next.run(request).await;
        
        // Add rate limit headers
        let headers = response.headers_mut();
        headers.insert(
            "X-RateLimit-Limit",
            rate_info.limit.to_string().parse().unwrap(),
        );
        headers.insert(
            "X-RateLimit-Remaining",
            rate_info.remaining.to_string().parse().unwrap(),
        );
        headers.insert(
            "X-RateLimit-Reset",
            rate_info.reset.to_string().parse().unwrap(),
        );

        Ok(response)
    } else {
        // Redis not available, allow request but log warning
        tracing::warn!("Rate limiting skipped: Redis not available");
        Ok(next.run(request).await)
    }
}

/// Get the rate limit key for a request
///
/// Uses user ID from JWT if available, otherwise falls back to IP address.
fn get_rate_limit_key(request: &Request<Body>) -> String {
    // Try to get user ID from JWT token
    if let Some(token) = extract_bearer_token(request.headers()) {
        // Extract user ID from token without full validation
        // (validation happens in auth middleware)
        if let Some(user_id) = extract_user_id_from_token(&token) {
            return format!("user:{}", user_id);
        }
    }

    // Fall back to IP address
    let ip = request
        .headers()
        .get("x-forwarded-for")
        .and_then(|v| v.to_str().ok())
        .and_then(|s| s.split(',').next())
        .map(|s| s.trim().to_string())
        .or_else(|| {
            request
                .headers()
                .get("x-real-ip")
                .and_then(|v| v.to_str().ok())
                .map(|s| s.to_string())
        })
        .unwrap_or_else(|| "unknown".to_string());

    format!("ip:{}", ip)
}

/// Extract user ID from JWT token without full validation
///
/// This is used for rate limiting key extraction only.
/// Full token validation happens in the auth middleware.
fn extract_user_id_from_token(token: &str) -> Option<String> {
    // JWT tokens have 3 parts separated by dots
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 {
        return None;
    }

    // Decode the payload (second part)
    use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
    
    let payload = URL_SAFE_NO_PAD.decode(parts[1]).ok()?;
    let payload_str = String::from_utf8(payload).ok()?;
    let claims: serde_json::Value = serde_json::from_str(&payload_str).ok()?;
    
    claims.get("sub").and_then(|v| v.as_str()).map(|s| s.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_user_id_from_token() {
        // Create a simple JWT-like token for testing
        // Header: {"alg":"HS256","typ":"JWT"}
        // Payload: {"sub":"user-123","iat":1234567890}
        let token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyIsImlhdCI6MTIzNDU2Nzg5MH0.signature";
        
        let user_id = extract_user_id_from_token(token);
        assert_eq!(user_id, Some("user-123".to_string()));
    }

    #[test]
    fn test_extract_user_id_invalid_token() {
        let user_id = extract_user_id_from_token("invalid-token");
        assert_eq!(user_id, None);
    }

    #[test]
    fn test_rate_limit_info() {
        let info = RateLimitInfo {
            limit: 1000,
            remaining: 999,
            reset: 1234567890,
            retry_after: None,
        };

        assert_eq!(info.limit, 1000);
        assert_eq!(info.remaining, 999);
        assert!(info.retry_after.is_none());
    }
}
