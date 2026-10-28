-- Missing tables from 001_init_schema.sql
-- Run this in Supabase SQL Editor

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

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Agent API Keys (opsional, untuk agent mode)
CREATE TABLE IF NOT EXISTS agent_api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  api_key VARCHAR(255) NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_api_keys_server ON agent_api_keys(server_id);
CREATE INDEX IF NOT EXISTS idx_agent_api_keys_key ON agent_api_keys(api_key);

-- Analysis Results (disk breakdown)
CREATE TABLE IF NOT EXISTS analysis_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  analysis_type VARCHAR(50) NOT NULL, -- 'breakdown', 'spike_detection'
  results JSONB NOT NULL, -- {top_folders: [...], spike_detected: bool, ...}
  analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analysis_results_server ON analysis_results(server_id);
CREATE INDEX IF NOT EXISTS idx_analysis_results_type ON analysis_results(analysis_type);

-- Cleanup Categories (kategori A: file kandidat)
CREATE TABLE IF NOT EXISTS cleanup_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL UNIQUE,
  description TEXT,
  default_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cleanup Recipes (kategori B: reclaimable safe operations)
CREATE TABLE IF NOT EXISTS cleanup_recipes (
  id VARCHAR(100) PRIMARY KEY, -- apt-cache, journal-logs, etc
  name VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  category VARCHAR(50) NOT NULL, -- cache, logs, temp, docker
  command TEXT NOT NULL,
  safety_level VARCHAR(20) NOT NULL DEFAULT 'safe', -- safe, moderate
  estimate_command TEXT,
  reversible BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trigger auto updated_at untuk users
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed default tenant (kalau belum ada)
INSERT INTO tenants (id, name, email) 
VALUES ('b39f9091-eb31-4c18-b77c-d08ced5c8fb1', 'Default Tenant', 'admin@archon.local')
ON CONFLICT (id) DO NOTHING;

-- Seed cleanup recipes (kategori B)
INSERT INTO cleanup_recipes (id, name, description, category, command, safety_level, estimate_command, reversible)
VALUES 
  ('apt-cache', 'APT package cache', 'Hapus cache paket APT yang sudah diunduh (aman, bisa diunduh ulang kapan saja)', 'cache', 'sudo apt-get clean', 'safe', 'du -sb /var/cache/apt/archives 2>/dev/null | awk ''{print $1}''', FALSE),
  ('journal-logs', 'Journal logs lama', 'Hapus systemd journal log lebih dari 7 hari (log lama biasanya tidak diperlukan)', 'logs', 'sudo journalctl --vacuum-time=7d', 'safe', 'journalctl --disk-usage 2>/dev/null | grep -oP ''\d+\.\d+[MGK]'' | head -1', FALSE),
  ('docker-unused-images', 'Docker image tidak terpakai', 'Hapus Docker image yang tidak terkait dengan container aktif (dangling images)', 'docker', 'docker image prune -af --filter ''dangling=true''', 'safe', 'docker system df -v 2>/dev/null | grep ''Images space usage'' | awk ''{print $4}''', FALSE),
  ('docker-build-cache', 'Docker build cache', 'Hapus cache build Docker (aman, akan dibuat ulang saat build berikutnya)', 'docker', 'docker builder prune -af', 'safe', 'docker system df -v 2>/dev/null | grep ''Build cache usage'' | awk ''{print $4}''', FALSE),
  ('npm-cache', 'NPM global cache', 'Hapus cache NPM global (aman, package bisa diunduh ulang)', 'cache', 'npm cache clean --force', 'safe', 'du -sb ~/.npm 2>/dev/null | awk ''{print $1}''', FALSE),
  ('tmp-old-files', 'File /tmp lebih dari 7 hari', 'Hapus file di /tmp yang tidak diakses lebih dari 7 hari', 'temp', 'find /tmp -type f -atime +7 -delete 2>/dev/null', 'moderate', 'find /tmp -type f -atime +7 -exec du -cb {} + 2>/dev/null | tail -1 | awk ''{print $1}''', FALSE),
  ('rotated-logs', 'Log yang sudah dirotasi (.gz)', 'Hapus log terkompresi hasil rotasi logrotate (log lama sudah dirotasi)', 'logs', 'find /var/log -type f -name ''*.gz'' -mtime +30 -delete 2>/dev/null', 'moderate', 'find /var/log -type f -name ''*.gz'' -mtime +30 -exec du -cb {} + 2>/dev/null | tail -1 | awk ''{print $1}''', FALSE)
ON CONFLICT (id) DO NOTHING;
