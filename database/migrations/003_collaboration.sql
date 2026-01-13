-- Cloud DevBox Collaboration Schema
-- Migration: 003_collaboration
-- Created: 2026-01-09

-- Collaboration sessions table
CREATE TABLE collaboration_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_session_status CHECK (status IN ('active', 'ended', 'expired'))
);

-- Collaboration participants table
CREATE TABLE collaboration_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'participant',
    cursor_position JSONB,
    selection JSONB,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(session_id, user_id),
    CONSTRAINT valid_participant_role CHECK (role IN ('host', 'participant', 'viewer'))
);

-- Collaboration history table
CREATE TABLE collaboration_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    operation_type VARCHAR(50) NOT NULL,
    file_path TEXT,
    operation_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Meetings table
CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    host_user_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'video',
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    settings JSONB NOT NULL DEFAULT '{
        "maxParticipants": 10,
        "allowRecording": false,
        "allowScreenShare": true,
        "enableTranscription": false,
        "enableVirtualBackground": true,
        "waitingRoom": false,
        "muteOnJoin": false
    }',
    scheduled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_meeting_type CHECK (type IN ('audio', 'video', 'screen_share')),
    CONSTRAINT valid_meeting_status CHECK (status IN ('scheduled', 'active', 'ended', 'cancelled'))
);

-- Meeting participants table
CREATE TABLE meeting_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    display_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'participant',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    audio_enabled BOOLEAN DEFAULT TRUE,
    video_enabled BOOLEAN DEFAULT TRUE,
    screen_sharing BOOLEAN DEFAULT FALSE,
    
    CONSTRAINT valid_meeting_participant_role CHECK (role IN ('host', 'co-host', 'participant')),
    UNIQUE(meeting_id, user_id)
);

-- Meeting recordings table
CREATE TABLE meeting_recordings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    duration INTEGER NOT NULL,
    file_size BIGINT NOT NULL,
    format VARCHAR(20) NOT NULL DEFAULT 'mp4',
    storage_url TEXT NOT NULL,
    transcription TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for collaboration
CREATE INDEX idx_collaboration_sessions_env_id ON collaboration_sessions(environment_id);
CREATE INDEX idx_collaboration_sessions_status ON collaboration_sessions(status);
CREATE INDEX idx_collaboration_participants_session ON collaboration_participants(session_id);
CREATE INDEX idx_collaboration_history_session ON collaboration_history(session_id);
CREATE INDEX idx_collaboration_history_created ON collaboration_history(created_at);
CREATE INDEX idx_meetings_environment_id ON meetings(environment_id);
CREATE INDEX idx_meetings_host_user_id ON meetings(host_user_id);
CREATE INDEX idx_meetings_status ON meetings(status);
CREATE INDEX idx_meeting_participants_meeting_id ON meeting_participants(meeting_id);
CREATE INDEX idx_meeting_participants_user_id ON meeting_participants(user_id);
