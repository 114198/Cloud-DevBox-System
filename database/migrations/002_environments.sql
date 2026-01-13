-- Cloud DevBox Environments Schema
-- Migration: 002_environments
-- Created: 2026-01-09

-- Environments table
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_id UUID NOT NULL REFERENCES templates(id),
    config JSONB NOT NULL,
    status JSONB NOT NULL DEFAULT '{
        "phase": "creating",
        "message": null,
        "podIP": null,
        "externalIP": null,
        "sshPort": null,
        "webPorts": []
    }',
    resources JSONB NOT NULL DEFAULT '{
        "cpu": "1",
        "memory": "2Gi",
        "storage": "10Gi"
    }',
    networking JSONB DEFAULT '{}',
    storage JSONB DEFAULT '{}',
    runtime JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT unique_user_env_name UNIQUE(user_id, name)
);

-- Environment collaborators table
CREATE TABLE environment_collaborators (
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    invited_by UUID REFERENCES users(id),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (environment_id, user_id),
    CONSTRAINT valid_collab_role CHECK (role IN ('owner', 'editor', 'viewer'))
);

-- Environment SSH keys table
CREATE TABLE environment_ssh_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    key_type VARCHAR(50) NOT NULL DEFAULT 'ed25519',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Environment activity log
CREATE TABLE environment_activity (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for environments
CREATE INDEX idx_environments_user_id ON environments(user_id);
CREATE INDEX idx_environments_project_id ON environments(project_id);
CREATE INDEX idx_environments_template_id ON environments(template_id);
CREATE INDEX idx_environments_status ON environments USING GIN((status->>'phase'));
CREATE INDEX idx_environments_created_at ON environments(created_at);
CREATE INDEX idx_environments_last_accessed ON environments(last_accessed_at);
CREATE INDEX idx_environment_activity_env_id ON environment_activity(environment_id);
CREATE INDEX idx_environment_activity_created_at ON environment_activity(created_at);
