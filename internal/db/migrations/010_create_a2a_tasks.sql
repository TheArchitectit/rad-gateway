CREATE TABLE IF NOT EXISTS a2a_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL,
    session_id TEXT NOT NULL,
    message JSONB NOT NULL,
    artifacts JSONB DEFAULT '[]',
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    parent_id UUID REFERENCES a2a_tasks(id),
    workspace_id TEXT REFERENCES workspaces(id),
    assigned_agent_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_a2a_tasks_status ON a2a_tasks(status);
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_session_id ON a2a_tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_workspace_id ON a2a_tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_created_at ON a2a_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_expires_at ON a2a_tasks(expires_at) WHERE expires_at IS NOT NULL;
