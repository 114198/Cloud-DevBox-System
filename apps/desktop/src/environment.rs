//! Environment management module

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Environment status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum EnvironmentStatus {
    Creating,
    Running,
    Stopped,
    Failed,
    Suspended,
    Archived,
    Deleting,
}

impl Default for EnvironmentStatus {
    fn default() -> Self {
        Self::Stopped
    }
}

/// Resource allocation for an environment
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ResourceAllocation {
    pub cpu: String,
    pub memory: String,
    pub storage: String,
}

/// SSH connection information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshInfo {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub private_key: Option<String>,
}

/// Environment details
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Environment {
    pub id: String,
    pub name: String,
    pub description: Option<String>,
    pub status: EnvironmentStatus,
    pub template_id: String,
    pub template_name: String,
    pub resources: ResourceAllocation,
    pub ssh_info: Option<SshInfo>,
    pub preview_url: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_accessed_at: Option<DateTime<Utc>>,
}

/// Request to create a new environment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateEnvironmentRequest {
    pub name: String,
    pub description: Option<String>,
    pub template_id: String,
    pub resources: Option<ResourceAllocation>,
}

/// Environment list response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvironmentListResponse {
    pub environments: Vec<Environment>,
    pub total: u32,
    pub page: u32,
    pub page_size: u32,
}

/// Environment API client
pub struct EnvironmentClient {
    client: reqwest::Client,
    base_url: String,
}

impl EnvironmentClient {
    pub fn new(base_url: &str) -> Self {
        Self {
            client: reqwest::Client::new(),
            base_url: base_url.to_string(),
        }
    }
    
    /// Set authorization token
    pub fn with_token(self, token: &str) -> AuthenticatedClient {
        AuthenticatedClient {
            client: self.client,
            base_url: self.base_url,
            token: token.to_string(),
        }
    }
}

/// Authenticated API client
pub struct AuthenticatedClient {
    client: reqwest::Client,
    base_url: String,
    token: String,
}

impl AuthenticatedClient {
    /// List all environments
    pub async fn list_environments(&self) -> Result<Vec<Environment>, reqwest::Error> {
        let response = self.client
            .get(format!("{}/api/v1/environments", self.base_url))
            .header("Authorization", format!("Bearer {}", self.token))
            .send()
            .await?
            .json::<EnvironmentListResponse>()
            .await?;
        
        Ok(response.environments)
    }
    
    /// Get environment details
    pub async fn get_environment(&self, id: &str) -> Result<Environment, reqwest::Error> {
        self.client
            .get(format!("{}/api/v1/environments/{}", self.base_url, id))
            .header("Authorization", format!("Bearer {}", self.token))
            .send()
            .await?
            .json()
            .await
    }
    
    /// Create a new environment
    pub async fn create_environment(&self, request: CreateEnvironmentRequest) -> Result<Environment, reqwest::Error> {
        self.client
            .post(format!("{}/api/v1/environments", self.base_url))
            .header("Authorization", format!("Bearer {}", self.token))
            .json(&request)
            .send()
            .await?
            .json()
            .await
    }
    
    /// Start an environment
    pub async fn start_environment(&self, id: &str) -> Result<Environment, reqwest::Error> {
        self.client
            .post(format!("{}/api/v1/environments/{}/start", self.base_url, id))
            .header("Authorization", format!("Bearer {}", self.token))
            .send()
            .await?
            .json()
            .await
    }
    
    /// Stop an environment
    pub async fn stop_environment(&self, id: &str) -> Result<Environment, reqwest::Error> {
        self.client
            .post(format!("{}/api/v1/environments/{}/stop", self.base_url, id))
            .header("Authorization", format!("Bearer {}", self.token))
            .send()
            .await?
            .json()
            .await
    }
    
    /// Delete an environment
    pub async fn delete_environment(&self, id: &str) -> Result<(), reqwest::Error> {
        self.client
            .delete(format!("{}/api/v1/environments/{}", self.base_url, id))
            .header("Authorization", format!("Bearer {}", self.token))
            .send()
            .await?
            .error_for_status()?;
        
        Ok(())
    }
}
