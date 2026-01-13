//! API client for Cloud DevBox backend
//!
//! Handles all HTTP communication with the DevBox API server.

use anyhow::{Context, Result};
use reqwest::{Client, StatusCode};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use std::time::Duration;

use crate::config::{CliConfig, CredentialManager};

/// API client for DevBox backend
pub struct ApiClient {
    client: Client,
    base_url: String,
    token: Option<String>,
}

/// API error response
#[derive(Debug, Deserialize)]
pub struct ApiError {
    pub error: String,
    pub message: String,
    #[serde(default)]
    pub details: Option<String>,
}

/// Login request
#[derive(Debug, Serialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

/// Login response
#[derive(Debug, Deserialize)]
pub struct LoginResponse {
    pub token: String,
    pub refresh_token: String,
    pub user: UserResponse,
}

/// User response from API
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct UserResponse {
    pub id: String,
    pub email: String,
    pub username: String,
    pub display_name: Option<String>,
    pub role: String,
}

/// Environment response from API
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct EnvironmentResponse {
    pub id: String,
    pub name: String,
    pub status: String,
    pub template_id: String,
    pub template_name: Option<String>,
    pub resources: ResourcesResponse,
    pub created_at: String,
    pub last_accessed_at: Option<String>,
    pub ssh_host: Option<String>,
    pub ssh_port: Option<u16>,
    pub preview_url: Option<String>,
}

/// Resources response
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ResourcesResponse {
    pub cpu: String,
    pub memory: String,
    pub storage: String,
}

/// Create environment request
#[derive(Debug, Serialize)]
pub struct CreateEnvironmentRequest {
    pub name: String,
    pub template_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resources: Option<ResourcesRequest>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub environment_vars: Option<std::collections::HashMap<String, String>>,
}

/// Resources request
#[derive(Debug, Serialize)]
pub struct ResourcesRequest {
    pub cpu: String,
    pub memory: String,
    pub storage: String,
}

/// Template response from API
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TemplateResponse {
    pub id: String,
    pub name: String,
    pub display_name: String,
    pub description: Option<String>,
    pub category: String,
    pub tags: Vec<String>,
    pub runtime: RuntimeResponse,
    pub is_public: bool,
}

/// Runtime response
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RuntimeResponse {
    pub language: String,
    pub version: String,
    pub framework: Option<String>,
}

/// SSH config response
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct SshConfigResponse {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub private_key: Option<String>,
    pub config_snippet: String,
}

/// Paginated response wrapper
#[derive(Debug, Deserialize)]
pub struct PaginatedResponse<T> {
    pub data: Vec<T>,
    pub total: u64,
    pub page: u32,
    pub per_page: u32,
}

impl ApiClient {
    /// Create a new API client
    pub fn new() -> Result<Self> {
        let config = CliConfig::load()?;
        let token = CredentialManager::get_token()?;
        
        let client = Client::builder()
            .timeout(Duration::from_secs(30))
            .user_agent(format!("devbox-cli/{}", env!("CARGO_PKG_VERSION")))
            .build()
            .with_context(|| "Failed to create HTTP client")?;
        
        Ok(Self {
            client,
            base_url: config.api_endpoint,
            token,
        })
    }

    /// Create a new API client with custom endpoint
    pub fn with_endpoint(endpoint: &str) -> Result<Self> {
        let token = CredentialManager::get_token()?;
        
        let client = Client::builder()
            .timeout(Duration::from_secs(30))
            .user_agent(format!("devbox-cli/{}", env!("CARGO_PKG_VERSION")))
            .build()
            .with_context(|| "Failed to create HTTP client")?;
        
        Ok(Self {
            client,
            base_url: endpoint.to_string(),
            token,
        })
    }

    /// Check if client is authenticated
    pub fn is_authenticated(&self) -> bool {
        self.token.is_some()
    }

    /// Require authentication, returning error if not authenticated
    pub fn require_auth(&self) -> Result<&str> {
        self.token.as_deref()
            .ok_or_else(|| anyhow::anyhow!("Not authenticated. Please run 'devbox login' first."))
    }

    /// Make a GET request
    async fn get<T: DeserializeOwned>(&self, path: &str) -> Result<T> {
        let token = self.require_auth()?;
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.client
            .get(&url)
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .with_context(|| format!("Failed to connect to {}", url))?;
        
        self.handle_response(response).await
    }

    /// Make a POST request
    async fn post<T: DeserializeOwned, B: Serialize>(&self, path: &str, body: &B) -> Result<T> {
        let token = self.require_auth()?;
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.client
            .post(&url)
            .header("Authorization", format!("Bearer {}", token))
            .json(body)
            .send()
            .await
            .with_context(|| format!("Failed to connect to {}", url))?;
        
        self.handle_response(response).await
    }

    /// Make a POST request without authentication
    async fn post_unauth<T: DeserializeOwned, B: Serialize>(&self, path: &str, body: &B) -> Result<T> {
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.client
            .post(&url)
            .json(body)
            .send()
            .await
            .with_context(|| format!("Failed to connect to {}", url))?;
        
        self.handle_response(response).await
    }

    /// Make a DELETE request
    async fn delete(&self, path: &str) -> Result<()> {
        let token = self.require_auth()?;
        let url = format!("{}{}", self.base_url, path);
        
        let response = self.client
            .delete(&url)
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .with_context(|| format!("Failed to connect to {}", url))?;
        
        if response.status().is_success() {
            Ok(())
        } else {
            let error: ApiError = response.json().await
                .unwrap_or(ApiError {
                    error: "unknown".to_string(),
                    message: "Unknown error occurred".to_string(),
                    details: None,
                });
            Err(anyhow::anyhow!("{}: {}", error.error, error.message))
        }
    }

    /// Handle API response
    async fn handle_response<T: DeserializeOwned>(&self, response: reqwest::Response) -> Result<T> {
        let status = response.status();
        
        if status.is_success() {
            response.json().await
                .with_context(|| "Failed to parse response")
        } else {
            let error: ApiError = response.json().await
                .unwrap_or(ApiError {
                    error: status.to_string(),
                    message: match status {
                        StatusCode::UNAUTHORIZED => "Authentication required".to_string(),
                        StatusCode::FORBIDDEN => "Access denied".to_string(),
                        StatusCode::NOT_FOUND => "Resource not found".to_string(),
                        StatusCode::TOO_MANY_REQUESTS => "Rate limit exceeded".to_string(),
                        _ => "Unknown error occurred".to_string(),
                    },
                    details: None,
                });
            Err(anyhow::anyhow!("{}: {}", error.error, error.message))
        }
    }

    // ==================== Auth API ====================

    /// Login with email and password
    pub async fn login(&self, email: &str, password: &str) -> Result<LoginResponse> {
        self.post_unauth("/api/v1/auth/login", &LoginRequest {
            email: email.to_string(),
            password: password.to_string(),
        }).await
    }

    /// Get current user info
    pub async fn whoami(&self) -> Result<UserResponse> {
        self.get("/api/v1/auth/me").await
    }

    // ==================== Environment API ====================

    /// List all environments
    pub async fn list_environments(&self) -> Result<Vec<EnvironmentResponse>> {
        let response: PaginatedResponse<EnvironmentResponse> = 
            self.get("/api/v1/environments").await?;
        Ok(response.data)
    }

    /// Get environment by ID or name
    pub async fn get_environment(&self, id_or_name: &str) -> Result<EnvironmentResponse> {
        self.get(&format!("/api/v1/environments/{}", id_or_name)).await
    }

    /// Create a new environment
    pub async fn create_environment(&self, request: CreateEnvironmentRequest) -> Result<EnvironmentResponse> {
        self.post("/api/v1/environments", &request).await
    }

    /// Start an environment
    pub async fn start_environment(&self, id_or_name: &str) -> Result<EnvironmentResponse> {
        self.post(&format!("/api/v1/environments/{}/start", id_or_name), &()).await
    }

    /// Stop an environment
    pub async fn stop_environment(&self, id_or_name: &str) -> Result<EnvironmentResponse> {
        self.post(&format!("/api/v1/environments/{}/stop", id_or_name), &()).await
    }

    /// Delete an environment
    pub async fn delete_environment(&self, id_or_name: &str) -> Result<()> {
        self.delete(&format!("/api/v1/environments/{}", id_or_name)).await
    }

    /// Get SSH configuration for an environment
    pub async fn get_ssh_config(&self, id_or_name: &str) -> Result<SshConfigResponse> {
        self.get(&format!("/api/v1/environments/{}/ssh-config", id_or_name)).await
    }

    // ==================== Template API ====================

    /// List all templates
    pub async fn list_templates(&self, category: Option<&str>) -> Result<Vec<TemplateResponse>> {
        let path = match category {
            Some(cat) => format!("/api/v1/templates?category={}", cat),
            None => "/api/v1/templates".to_string(),
        };
        let response: PaginatedResponse<TemplateResponse> = self.get(&path).await?;
        Ok(response.data)
    }

    /// Get template by ID or name
    pub async fn get_template(&self, id_or_name: &str) -> Result<TemplateResponse> {
        self.get(&format!("/api/v1/templates/{}", id_or_name)).await
    }
}
