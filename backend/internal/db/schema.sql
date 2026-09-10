-- Hermes Database Schema (Fase 1 MVP)
-- PostgreSQL 14+

-- Tenants (klien yang menggunakan Hermes)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(50),
    parent_agency_id UUID REFERENCES tenants(id),
    subscription_tier VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users (akses dashboard)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'viewer',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Servers yang dimonitor
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hostname VARCHAR(255) NOT NULL,
    ip_address INET,
    os_type VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active',
    last_seen_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, hostname)
);

-- API keys untuk agent authentication
CREATE TABLE IF NOT EXISTS agent_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    revoked_at TIMESTAMP
);

-- Storage metrics time-series
CREATE TABLE IF NOT EXISTS storage_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    mount_path VARCHAR(500) NOT NULL,
    total_bytes BIGINT NOT NULL,
    used_bytes BIGINT NOT NULL,
    available_bytes BIGINT NOT NULL,
    used_percent DECIMAL(5,2) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- File candidates untuk cleanup
CREATE TABLE IF NOT EXISTS file_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    last_access TIMESTAMP,
    last_modified TIMESTAMP,
    category VARCHAR(100),
    confidence DECIMAL(3,2),
    detected_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Cleanup actions (audit trail)
CREATE TABLE IF NOT EXISTS cleanup_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    triggered_by_user_id UUID REFERENCES users(id),
    reason TEXT,
    file_count INT NOT NULL,
    total_bytes_freed BIGINT,
    snapshot_path TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    confirmed BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    restore_window_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Files yang di-cleanup (detail per file)
CREATE TABLE IF NOT EXISTS cleanup_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cleanup_action_id UUID NOT NULL REFERENCES cleanup_actions(id) ON DELETE CASCADE,
    file_candidate_id UUID REFERENCES file_candidates(id),
    original_path TEXT NOT NULL,
    snapshot_path TEXT,
    size_bytes BIGINT NOT NULL,
    deleted_at TIMESTAMP,
    restored_at TIMESTAMP
);

-- Backups metadata
CREATE TABLE IF NOT EXISTS backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    backup_name VARCHAR(255) NOT NULL,
    backup_type VARCHAR(100),
    size_bytes BIGINT,
    status VARCHAR(50),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Backup verifications (Fase 1: status check only)
CREATE TABLE IF NOT EXISTS backup_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
    verification_type VARCHAR(50) DEFAULT 'status_check',
    status VARCHAR(50),
    notes TEXT,
    checked_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Notifications log
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_servers_tenant ON servers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_storage_server_recent ON storage_snapshots(server_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_server_time ON storage_snapshots(server_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_candidates_server_status ON file_candidates(server_id, status);
CREATE INDEX IF NOT EXISTS idx_cleanup_server_status ON cleanup_actions(server_id, status);
CREATE INDEX IF NOT EXISTS idx_backups_server_time ON backups(server_id, completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant ON notifications(tenant_id, created_at DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers (use DO block to make idempotent)
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_tenants_updated_at') THEN
    CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;
END $$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_users_updated_at') THEN
    CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;
END $$;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_servers_updated_at') THEN
    CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;
END $$;

-- Enable pgcrypto for gen_random_uuid() if not already
CREATE EXTENSION IF NOT EXISTS pgcrypto;
