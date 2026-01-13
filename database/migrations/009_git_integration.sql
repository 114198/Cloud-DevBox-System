-- Cloud DevBox Git Integration Schema
-- Migration: 009_git_integration
-- Created: 2026-01-12

-- Git connections table - stores OAuth tokens for Git platforms
CREATE TABLE git_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_username VARCHAR(255),
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expires_at TIMESTAMP WITH TIME ZONE,
    scopes TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, provider),
    CONSTRAINT valid_git_provider CHECK (provider IN ('github', 'gitlab', 'gitee', 'bitbucket'))
);

-- Git repositories table - stores linked repositories
CREATE TABLE git_repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES git_connections(id) ON DELETE CASCADE,
    environment_id VARCHAR(255),
    provider VARCHAR(50) NOT NULL,
    repo_id VARCHAR(255) NOT NULL,
    repo_name VARCHAR(255) NOT NULL,
    repo_full_name VARCHAR(500) NOT NULL,
    clone_url TEXT NOT NULL,
    ssh_url TEXT,
    default_branch VARCHAR(255) DEFAULT 'main',
    is_private BOOLEAN DEFAULT FALSE,
    webhook_id VARCHAR(255),
    webhook_secret VARCHAR(255),
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(environment_id, provider, repo_id),
    CONSTRAINT valid_sync_status CHECK (sync_status IN ('pending', 'syncing', 'synced', 'failed'))
);

-- Git webhooks table - stores webhook events
CREATE TABLE git_webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository_id UUID NOT NULL REFERENCES git_repositories(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_id VARCHAR(255),
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT valid_event_type CHECK (event_type IN ('push', 'pull_request', 'create', 'delete', 'release'))
);

-- Git sync history table - tracks sync operations
CREATE TABLE git_sync_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository_id UUID NOT NULL REFERENCES git_repositories(id) ON DELETE CASCADE,
    environment_id VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    commit_message TEXT,
    commit_author VARCHAR(255),
    branch VARCHAR(255) NOT NULL,
    sync_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    error_message TEXT,
    
    CONSTRAINT valid_sync_type CHECK (sync_type IN ('clone', 'pull', 'webhook', 'manual')),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'in_progress', 'completed', 'failed'))
);

-- Indexes for performance
CREATE INDEX idx_git_connections_user_id ON git_connections(user_id);
CREATE INDEX idx_git_connections_provider ON git_connections(provider);
CREATE INDEX idx_git_repositories_user_id ON git_repositories(user_id);
CREATE INDEX idx_git_repositories_environment_id ON git_repositories(environment_id);
CREATE INDEX idx_git_repositories_connection_id ON git_repositories(connection_id);
CREATE INDEX idx_git_webhooks_repository_id ON git_webhooks(repository_id);
CREATE INDEX idx_git_webhooks_processed ON git_webhooks(processed) WHERE processed = FALSE;
CREATE INDEX idx_git_sync_history_repository_id ON git_sync_history(repository_id);
CREATE INDEX idx_git_sync_history_environment_id ON git_sync_history(environment_id);
CREATE INDEX idx_git_sync_history_status ON git_sync_history(status);
