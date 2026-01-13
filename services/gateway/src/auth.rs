//! JWT Authentication module
//!
//! Provides JWT token validation, generation, and user context extraction.
//! Supports both access tokens and refresh tokens with RS256/HS256 algorithms.

use axum::{
    async_trait,
    body::Body,
    extract::{FromRequestParts, State},
    http::{header::AUTHORIZATION, request::Parts, Request, StatusCode},
    middleware::Next,
    response::Response,
};
use chrono::{Duration, Utc};
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, TokenData, Validation};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::config::JwtConfig;
use crate::error::GatewayError;
use crate::middleware::USER_ID_HEADER;
use crate::state::AppState;

/// JWT Claims structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    /// Subject (user ID)
    pub sub: String,
    /// Issuer
    pub iss: String,
    /// Audience
    pub aud: String,
    /// Expiration time (Unix timestamp)
    pub exp: i64,
    /// Issued at (Unix timestamp)
    pub iat: i64,
    /// Not before (Unix timestamp)
    pub nbf: i64,
    /// JWT ID (unique identifier)
    pub jti: String,
    /// Token type (access or refresh)
    pub token_type: TokenType,
    /// User role
    #[serde(default)]
    pub role: String,
    /// User email
    #[serde(default)]
    pub email: String,
}

/// Token type enum
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum TokenType {
    Access,
    Refresh,
}

/// Authenticated user context extracted from JWT
#[derive(Debug, Clone)]
pub struct AuthUser {
    pub user_id: String,
    pub email: String,
    pub role: String,
    pub token_id: String,
}

/// JWT token pair (access + refresh)
#[derive(Debug, Serialize)]
pub struct TokenPair {
    pub access_token: String,
    pub refresh_token: String,
    pub token_type: String,
    pub expires_in: i64,
}

/// JWT service for token operations
pub struct JwtService {
    config: JwtConfig,
    encoding_key: EncodingKey,
    decoding_key: DecodingKey,
}

impl JwtService {
    /// Create a new JWT service
    pub fn new(config: JwtConfig) -> Self {
        let encoding_key = EncodingKey::from_secret(config.secret.as_bytes());
        let decoding_key = DecodingKey::from_secret(config.secret.as_bytes());

        Self {
            config,
            encoding_key,
            decoding_key,
        }
    }

    /// Generate a new access token
    pub fn generate_access_token(&self, user_id: &str, email: &str, role: &str) -> Result<String, GatewayError> {
        let now = Utc::now();
        let exp = now + Duration::seconds(self.config.access_token_expiry_secs as i64);

        let claims = Claims {
            sub: user_id.to_string(),
            iss: self.config.issuer.clone(),
            aud: self.config.audience.clone(),
            exp: exp.timestamp(),
            iat: now.timestamp(),
            nbf: now.timestamp(),
            jti: Uuid::new_v4().to_string(),
            token_type: TokenType::Access,
            role: role.to_string(),
            email: email.to_string(),
        };

        encode(&Header::default(), &claims, &self.encoding_key)
            .map_err(|e| GatewayError::Internal(format!("Failed to generate token: {}", e)))
    }

    /// Generate a new refresh token
    pub fn generate_refresh_token(&self, user_id: &str) -> Result<String, GatewayError> {
        let now = Utc::now();
        let exp = now + Duration::seconds(self.config.refresh_token_expiry_secs as i64);

        let claims = Claims {
            sub: user_id.to_string(),
            iss: self.config.issuer.clone(),
            aud: self.config.audience.clone(),
            exp: exp.timestamp(),
            iat: now.timestamp(),
            nbf: now.timestamp(),
            jti: Uuid::new_v4().to_string(),
            token_type: TokenType::Refresh,
            role: String::new(),
            email: String::new(),
        };

        encode(&Header::default(), &claims, &self.encoding_key)
            .map_err(|e| GatewayError::Internal(format!("Failed to generate refresh token: {}", e)))
    }

    /// Generate both access and refresh tokens
    pub fn generate_token_pair(&self, user_id: &str, email: &str, role: &str) -> Result<TokenPair, GatewayError> {
        let access_token = self.generate_access_token(user_id, email, role)?;
        let refresh_token = self.generate_refresh_token(user_id)?;

        Ok(TokenPair {
            access_token,
            refresh_token,
            token_type: "Bearer".to_string(),
            expires_in: self.config.access_token_expiry_secs as i64,
        })
    }

    /// Validate and decode a token
    pub fn validate_token(&self, token: &str) -> Result<Claims, GatewayError> {
        let mut validation = Validation::default();
        validation.set_issuer(&[&self.config.issuer]);
        validation.set_audience(&[&self.config.audience]);

        let token_data: TokenData<Claims> = decode(token, &self.decoding_key, &validation)
            .map_err(|e| match e.kind() {
                jsonwebtoken::errors::ErrorKind::ExpiredSignature => GatewayError::TokenExpired,
                jsonwebtoken::errors::ErrorKind::InvalidToken => {
                    GatewayError::InvalidToken("Malformed token".to_string())
                }
                jsonwebtoken::errors::ErrorKind::InvalidSignature => {
                    GatewayError::InvalidToken("Invalid signature".to_string())
                }
                jsonwebtoken::errors::ErrorKind::InvalidIssuer => {
                    GatewayError::InvalidToken("Invalid issuer".to_string())
                }
                jsonwebtoken::errors::ErrorKind::InvalidAudience => {
                    GatewayError::InvalidToken("Invalid audience".to_string())
                }
                _ => GatewayError::InvalidToken(format!("Token validation failed: {}", e)),
            })?;

        Ok(token_data.claims)
    }

    /// Validate an access token specifically
    pub fn validate_access_token(&self, token: &str) -> Result<Claims, GatewayError> {
        let claims = self.validate_token(token)?;

        if claims.token_type != TokenType::Access {
            return Err(GatewayError::InvalidToken(
                "Expected access token, got refresh token".to_string(),
            ));
        }

        Ok(claims)
    }

    /// Validate a refresh token and generate new token pair
    pub fn refresh_tokens(&self, refresh_token: &str) -> Result<TokenPair, GatewayError> {
        let claims = self.validate_token(refresh_token)?;

        if claims.token_type != TokenType::Refresh {
            return Err(GatewayError::InvalidToken(
                "Expected refresh token, got access token".to_string(),
            ));
        }

        // Generate new token pair
        // Note: In production, you'd want to fetch fresh user data from the database
        self.generate_token_pair(&claims.sub, &claims.email, &claims.role)
    }
}

/// Extract bearer token from Authorization header
pub fn extract_bearer_token(headers: &axum::http::HeaderMap) -> Option<String> {
    headers
        .get(AUTHORIZATION)
        .and_then(|value| value.to_str().ok())
        .and_then(|value| {
            if value.to_lowercase().starts_with("bearer ") {
                Some(value[7..].to_string())
            } else {
                None
            }
        })
}

/// JWT authentication middleware
///
/// Validates JWT tokens and injects user context into request extensions.
/// Returns 401 Unauthorized for invalid or missing tokens.
pub async fn jwt_auth_middleware(
    State(state): State<Arc<AppState>>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response, GatewayError> {
    let token = extract_bearer_token(request.headers())
        .ok_or_else(|| GatewayError::Unauthorized("Missing authorization header".to_string()))?;

    let jwt_service = JwtService::new(state.config.jwt.clone());
    let claims = jwt_service.validate_access_token(&token)?;

    // Create authenticated user context
    let auth_user = AuthUser {
        user_id: claims.sub.clone(),
        email: claims.email,
        role: claims.role,
        token_id: claims.jti,
    };

    // Insert user context into request extensions
    request.extensions_mut().insert(auth_user);

    // Add user ID to headers for downstream services
    request.headers_mut().insert(
        USER_ID_HEADER,
        claims.sub.parse().unwrap(),
    );

    Ok(next.run(request).await)
}

/// Optional JWT authentication middleware
///
/// Similar to jwt_auth_middleware but doesn't fail on missing tokens.
/// Useful for endpoints that work with or without authentication.
pub async fn optional_jwt_auth_middleware(
    State(state): State<Arc<AppState>>,
    mut request: Request<Body>,
    next: Next,
) -> Response {
    if let Some(token) = extract_bearer_token(request.headers()) {
        let jwt_service = JwtService::new(state.config.jwt.clone());
        
        if let Ok(claims) = jwt_service.validate_access_token(&token) {
            let auth_user = AuthUser {
                user_id: claims.sub.clone(),
                email: claims.email,
                role: claims.role,
                token_id: claims.jti,
            };

            request.extensions_mut().insert(auth_user);
            request.headers_mut().insert(
                USER_ID_HEADER,
                claims.sub.parse().unwrap(),
            );
        }
    }

    next.run(request).await
}

/// Extractor for authenticated user from request extensions
#[async_trait]
impl<S> FromRequestParts<S> for AuthUser
where
    S: Send + Sync,
{
    type Rejection = GatewayError;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<AuthUser>()
            .cloned()
            .ok_or_else(|| GatewayError::Unauthorized("Not authenticated".to_string()))
    }
}

/// Role-based access control middleware factory
pub fn require_role(required_role: &'static str) -> impl Fn(AuthUser) -> Result<AuthUser, GatewayError> + Clone {
    move |user: AuthUser| {
        if user.role == required_role || user.role == "admin" {
            Ok(user)
        } else {
            Err(GatewayError::Forbidden(format!(
                "Required role '{}', but user has role '{}'",
                required_role, user.role
            )))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_config() -> JwtConfig {
        JwtConfig {
            secret: "test-secret-key-for-testing-only".to_string(),
            issuer: "test-issuer".to_string(),
            audience: "test-audience".to_string(),
            access_token_expiry_secs: 3600,
            refresh_token_expiry_secs: 604800,
        }
    }

    #[test]
    fn test_generate_and_validate_access_token() {
        let service = JwtService::new(test_config());
        
        let token = service
            .generate_access_token("user-123", "test@example.com", "developer")
            .expect("Failed to generate token");

        let claims = service
            .validate_access_token(&token)
            .expect("Failed to validate token");

        assert_eq!(claims.sub, "user-123");
        assert_eq!(claims.email, "test@example.com");
        assert_eq!(claims.role, "developer");
        assert_eq!(claims.token_type, TokenType::Access);
    }

    #[test]
    fn test_generate_and_validate_refresh_token() {
        let service = JwtService::new(test_config());
        
        let token = service
            .generate_refresh_token("user-123")
            .expect("Failed to generate token");

        let claims = service
            .validate_token(&token)
            .expect("Failed to validate token");

        assert_eq!(claims.sub, "user-123");
        assert_eq!(claims.token_type, TokenType::Refresh);
    }

    #[test]
    fn test_token_pair_generation() {
        let service = JwtService::new(test_config());
        
        let pair = service
            .generate_token_pair("user-123", "test@example.com", "developer")
            .expect("Failed to generate token pair");

        assert_eq!(pair.token_type, "Bearer");
        assert!(!pair.access_token.is_empty());
        assert!(!pair.refresh_token.is_empty());
    }

    #[test]
    fn test_invalid_token() {
        let service = JwtService::new(test_config());
        
        let result = service.validate_token("invalid-token");
        assert!(result.is_err());
    }

    #[test]
    fn test_refresh_token_cannot_be_used_as_access() {
        let service = JwtService::new(test_config());
        
        let refresh_token = service
            .generate_refresh_token("user-123")
            .expect("Failed to generate token");

        let result = service.validate_access_token(&refresh_token);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_bearer_token() {
        let mut headers = axum::http::HeaderMap::new();
        headers.insert(AUTHORIZATION, "Bearer test-token".parse().unwrap());

        let token = extract_bearer_token(&headers);
        assert_eq!(token, Some("test-token".to_string()));
    }

    #[test]
    fn test_extract_bearer_token_missing() {
        let headers = axum::http::HeaderMap::new();
        let token = extract_bearer_token(&headers);
        assert_eq!(token, None);
    }

    #[test]
    fn test_extract_bearer_token_wrong_scheme() {
        let mut headers = axum::http::HeaderMap::new();
        headers.insert(AUTHORIZATION, "Basic test-token".parse().unwrap());

        let token = extract_bearer_token(&headers);
        assert_eq!(token, None);
    }
}
