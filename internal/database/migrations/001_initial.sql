-- Create repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_provider TEXT NOT NULL,
    source_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create replication_targets table
CREATE TABLE IF NOT EXISTS replication_targets (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    remote_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for faster lookups on repository_id
CREATE INDEX IF NOT EXISTS idx_replication_targets_repository_id 
ON replication_targets(repository_id);
