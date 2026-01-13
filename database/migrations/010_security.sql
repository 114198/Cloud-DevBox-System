-- Cloud DevBox Security Schema
-- Migration: 010_security
-- Created: 2026-01-13

-- IP Blacklist table for storing blocked IPs
CREATE TABLE ip_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ip_address INET NOT NULL,
    reason VARCHAR(255) NOT NULL,
    blocked_by VARCHAR(50) NOT NULL DEFAULT 'system', -- 'system' or 'admin'
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_ip_address UNIQUE (ip_address)
);

-- Access attempts table for tracking login/access attempts
CREATE TABLE access_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ip_address INET NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    attempt_type VARCHAR(50) NOT NULL, -- 'login', 'api', 'ssh', 'resource'
    success BOOLEAN NOT NULL DEFAULT FALSE,
    failure_reason VARCHAR(255),
    user_agent TEXT,
    endpoint VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security events table for anomaly detection
CREATE TABLE security_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL, -- 'brute_force', 'suspicious_activity', 'rate_limit_exceeded', etc.
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    ip_address INET,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Encryption keys metadata table (actual keys stored in Vault)
CREATE TABLE encryption_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_id VARCHAR(255) NOT NULL UNIQUE, -- Reference to key in Vault
    key_type VARCHAR(50) NOT NULL, -- 'data', 'config', 'backup'
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'rotating', 'retired'
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    rotated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Encrypted data references table
CREATE TABLE encrypted_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL, -- 'environment_secret', 'user_credential', etc.
    resource_id UUID NOT NULL,
    key_id UUID NOT NULL REFERENCES encryption_keys(id),
    encrypted_data BYTEA NOT NULL,
    nonce BYTEA NOT NULL, -- IV for AES-GCM
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for security tables
CREATE INDEX idx_ip_blacklist_ip ON ip_blacklist(ip_address);
CREATE INDEX idx_ip_blacklist_expires ON ip_blacklist(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_access_attempts_ip ON access_attempts(ip_address);
CREATE INDEX idx_access_attempts_user ON access_attempts(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_access_attempts_created ON access_attempts(created_at);
CREATE INDEX idx_access_attempts_type ON access_attempts(attempt_type);
CREATE INDEX idx_security_events_type ON security_events(event_type);
CREATE INDEX idx_security_events_severity ON security_events(severity);
CREATE INDEX idx_security_events_ip ON security_events(ip_address) WHERE ip_address IS NOT NULL;
CREATE INDEX idx_security_events_user ON security_events(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_security_events_created ON security_events(created_at);
CREATE INDEX idx_security_events_unresolved ON security_events(resolved) WHERE resolved = FALSE;
CREATE INDEX idx_encryption_keys_status ON encryption_keys(status);
CREATE INDEX idx_encrypted_data_resource ON encrypted_data(resource_type, resource_id);

-- Function to clean up old access attempts (keep 90 days)
CREATE OR REPLACE FUNCTION cleanup_old_access_attempts()
RETURNS void AS $$
BEGIN
    DELETE FROM access_attempts WHERE created_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired IP blacklist entries
CREATE OR REPLACE FUNCTION cleanup_expired_blacklist()
RETURNS void AS $$
BEGIN
    DELETE FROM ip_blacklist WHERE expires_at IS NOT NULL AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;
