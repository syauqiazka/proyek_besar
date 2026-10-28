-- Archon Fase 1 MVP Schema
-- Migration 001: Core tables untuk monitoring storage via SSH

-- Tenants (klien yang pakai Archon)
CREATE TABLE IF NOT EXISTS tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  phone VARCHAR(50),
  parent_agency_id UUID REFERENCES tenants(id), -- untuk white-label masa depan
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_parent ON tenants(parent_agency_id);

-- Users (akses dashboard)
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'viewer', -- admin, editor, viewer
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);

-- Servers yang dimonitor
CREATE TABLE IF NOT EXISTS servers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  hostname VARCHAR(255) NOT NULL,
  os VARCHAR(100), -- detected: ubuntu, debian, centos, etc
  connection_mode VARCHAR(20) NOT NULL DEFAULT 'ssh', -- ssh, agent
  cleanup_enabled BOOLEAN NOT NULL DEFAULT FALSE, -- default read-only
  stack_detected JSONB, -- {web_server: nginx, database: postgresql, docker: true}
  status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, connected, error
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_servers_tenant ON servers(tenant_id);
CREATE INDEX idx_servers_status ON servers(status);

-- SSH connection details
CREATE TABLE IF NOT EXISTS server_connections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  host VARCHAR(255) NOT NULL,
  port INTEGER NOT NULL DEFAULT 22,
  username VARCHAR(100) NOT NULL DEFAULT 'archon',
  private_key_encrypted TEXT NOT NULL, -- encrypted at rest
  host_fingerprint VARCHAR(255), -- SSH host key fingerprint
  last_test_at TIMESTAMPTZ,
  last_test_status VARCHAR(50), -- success, failed, timeout
  last_test_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_server_connections_server ON server_connections(server_id);

-- SSH command audit log
CREATE TABLE IF NOT EXISTS ssh_command_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  command_template VARCHAR(255) NOT NULL, -- disk_scan, file_list, backup_check
  parameters JSONB, -- {path: /var/log, depth: 3}
  executed_command TEXT NOT NULL, -- rendered command (tanpa sensitif data)
  exit_code INTEGER,
  duration_ms INTEGER,
  triggered_by VARCHAR(100), -- cron, user:{id}, agent
  executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssh_logs_server ON ssh_command_logs(server_id);
CREATE INDEX idx_ssh_logs_executed_at ON ssh_command_logs(executed_at DESC);

-- Agent API keys (opsional, untuk mode agent)
CREATE TABLE IF NOT EXISTS agent_api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  api_key VARCHAR(255) NOT NULL UNIQUE,
  last_used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_agent_keys_server ON agent_api_keys(server_id);
CREATE INDEX idx_agent_keys_api_key ON agent_api_keys(api_key);

-- Storage snapshots (time-series disk usage)
CREATE TABLE IF NOT EXISTS storage_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  mount_path VARCHAR(500) NOT NULL,
  total_bytes BIGINT NOT NULL,
  used_bytes BIGINT NOT NULL,
  available_bytes BIGINT NOT NULL,
  usage_percent NUMERIC(5,2) NOT NULL,
  inodes_total BIGINT,
  inodes_used BIGINT,
  captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snapshots_server ON storage_snapshots(server_id);
CREATE INDEX idx_snapshots_captured ON storage_snapshots(captured_at DESC);
CREATE INDEX idx_snapshots_server_mount ON storage_snapshots(server_id, mount_path, captured_at DESC);

-- Disk usage breakdown (folder/file terbesar)
CREATE TABLE IF NOT EXISTS disk_usage_breakdowns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  mount_path VARCHAR(500) NOT NULL,
  breakdown_type VARCHAR(50) NOT NULL, -- top_folders, top_files, spike_detection
  items JSONB NOT NULL, -- [{path: /var/log, size_bytes: 1234567890, percent: 12.3}]
  analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_breakdowns_server ON disk_usage_breakdowns(server_id);
CREATE INDEX idx_breakdowns_analyzed ON disk_usage_breakdowns(analyzed_at DESC);

-- File cleanup candidates (kategori A)
CREATE TABLE IF NOT EXISTS file_candidates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  file_path VARCHAR(1000) NOT NULL,
  size_bytes BIGINT NOT NULL,
  category VARCHAR(100) NOT NULL, -- temp_files, old_logs, cache, orphaned
  confidence NUMERIC(3,2) NOT NULL, -- 0.0 - 1.0
  reason TEXT,
  detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status VARCHAR(50) NOT NULL DEFAULT 'pending' -- pending, approved, rejected, deleted
);

CREATE INDEX idx_candidates_server ON file_candidates(server_id);
CREATE INDEX idx_candidates_status ON file_candidates(status);
CREATE INDEX idx_candidates_detected ON file_candidates(detected_at DESC);

-- Cleanup actions audit trail
CREATE TABLE IF NOT EXISTS cleanup_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  category VARCHAR(50) NOT NULL, -- file_candidate, recipe
  recipe_id VARCHAR(100), -- apt_cache, docker_prune, journal_logs (kategori B)
  files_deleted INTEGER NOT NULL DEFAULT 0,
  bytes_freed BIGINT NOT NULL DEFAULT 0,
  triggered_by UUID REFERENCES users(id), -- NULL kalau dari WhatsApp command
  confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  dry_run BOOLEAN NOT NULL DEFAULT FALSE,
  executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  details JSONB -- {file_paths: [...], recipe_params: {...}}
);

CREATE INDEX idx_cleanup_server ON cleanup_actions(server_id);
CREATE INDEX idx_cleanup_executed ON cleanup_actions(executed_at DESC);
CREATE INDEX idx_cleanup_category ON cleanup_actions(category);

-- Backup metadata
CREATE TABLE IF NOT EXISTS backups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  backup_path VARCHAR(1000) NOT NULL,
  backup_type VARCHAR(50), -- full, incremental, snapshot
  size_bytes BIGINT,
  last_modified_at TIMESTAMPTZ,
  detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backups_server ON backups(server_id);
CREATE INDEX idx_backups_detected ON backups(detected_at DESC);

-- Backup verifications (MVP: status check + integrity test)
CREATE TABLE IF NOT EXISTS backup_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  backup_id UUID NOT NULL REFERENCES backups(id) ON DELETE CASCADE,
  verification_type VARCHAR(50) NOT NULL, -- age_check, size_check, integrity_test
  status VARCHAR(50) NOT NULL, -- success, warning, failed
  message TEXT,
  verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_verifications_backup ON backup_verifications(backup_id);
CREATE INDEX idx_verifications_verified ON backup_verifications(verified_at DESC);

-- Notifications log
CREATE TABLE IF NOT EXISTS notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
  channel VARCHAR(50) NOT NULL, -- whatsapp
  recipient VARCHAR(255) NOT NULL,
  message TEXT NOT NULL,
  status VARCHAR(50) NOT NULL, -- sent, failed, pending
  sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  metadata JSONB -- {alert_type: storage_warning, threshold: 85}
);

CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
CREATE INDEX idx_notifications_server ON notifications(server_id);
CREATE INDEX idx_notifications_sent ON notifications(sent_at DESC);

-- Updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_server_connections_updated_at BEFORE UPDATE ON server_connections FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
