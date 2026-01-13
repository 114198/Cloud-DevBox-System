-- Cloud DevBox Template Versions Migration
-- Migration: 008_template_versions
-- Created: 2026-01-09

-- Template versions table for version history
CREATE TABLE IF NOT EXISTS template_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    runtime JSONB NOT NULL,
    default_config JSONB NOT NULL,
    dockerfile TEXT NOT NULL,
    init_script TEXT,
    extensions JSONB DEFAULT '[]',
    change_log TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(template_id, version)
);

-- Add parent_id column to templates for template inheritance
ALTER TABLE templates ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES templates(id) ON DELETE SET NULL;

-- Indexes for template versions
CREATE INDEX IF NOT EXISTS idx_template_versions_template_id ON template_versions(template_id);
CREATE INDEX IF NOT EXISTS idx_template_versions_version ON template_versions(version);
CREATE INDEX IF NOT EXISTS idx_template_versions_created_at ON template_versions(created_at);

-- Index for template parent
CREATE INDEX IF NOT EXISTS idx_templates_parent_id ON templates(parent_id);

-- Index for template search
CREATE INDEX IF NOT EXISTS idx_templates_display_name_trgm ON templates USING gin(display_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_templates_description_trgm ON templates USING gin(description gin_trgm_ops);

-- Enable trigram extension for fuzzy search (if not exists)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create initial versions for existing templates
INSERT INTO template_versions (template_id, version, runtime, default_config, dockerfile, init_script, extensions, change_log, created_by, created_at)
SELECT 
    id,
    version,
    runtime,
    default_config,
    dockerfile,
    init_script,
    extensions,
    'Initial version',
    created_by,
    created_at
FROM templates
WHERE NOT EXISTS (
    SELECT 1 FROM template_versions tv WHERE tv.template_id = templates.id
);
